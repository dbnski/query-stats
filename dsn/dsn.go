package dsn

import (
    "errors"
    "fmt"
    "net/url"
    "os/user"
    "reflect"
    "strconv"
    "unsafe"

    "github.com/alecthomas/kong"
    "gopkg.in/ini.v1"
    "golang.org/x/term"
    "github.com/go-sql-driver/mysql"
)

const (
    defaultPrompt string = "Enter password"
)

var (
    ErrInvalidScheme      = errors.New("invalid or missing scheme")
    ErrInvalidHostname    = errors.New("invalid or missing hostname")
    ErrUsernameIsRequired = errors.New("username is required")
    ErrUnsupportedField   = errors.New("unsupported field type")
)

var MySQLMapper      = defaultMapper((*MySQL)(nil).Scheme())

type MySQL struct {
    dsn *url.URL
}

func (m *MySQL) Scheme() string {
    return "mysql"
}

func (m *MySQL) URL() string {
    return m.dsn.String()
}

func (m *MySQL) DSN() string {
    cfg := mysql.NewConfig()
    cfg.Addr = m.dsn.Host
    cfg.Net = "tcp"
    cfg.User = m.dsn.User.Username()
    cfg.Passwd, _ = m.dsn.User.Password()
    cfg.DBName = m.dsn.Path[1:]
    return cfg.FormatDSN()
}

func (m *MySQL) String() string {
    return m.dsn.Redacted()
}

func (m *MySQL) Addr() string {
    return m.dsn.Host
}

func (m *MySQL) Db() string {
    if len(m.dsn.Path) == 0 {
        return ""
    } else {
        return m.dsn.Path[1:]
    }
}

func (m *MySQL) Options() url.Values {
    return m.dsn.Query()
}

func (m *MySQL) User() string {
    return m.dsn.User.Username()
}

func (m *MySQL) Password() string {
    p, _ := m.dsn.User.Password()
    return p
}

func (m *MySQL) Host() string {
    return m.dsn.Hostname()
}

func (m *MySQL) Port() int {
    p, err := strconv.Atoi(m.dsn.Port())
    if err != nil {
        p = 3306
    }
    return p
}

func (m *MySQL) AskPass() error {
    return m.AskPassWithPrompt(defaultPrompt)
}

func (m *MySQL) AskPassWithPrompt(prompt string) error {
    pass, err := askPass(prompt)
    if err != nil {
        return err
    }
    name := m.dsn.User.Username()
    m.dsn.User = url.UserPassword(name, pass)
    return nil
}

type iniConfig struct {
    Host     string `ini:"host"`
    Port     string `ini:"port"`
    User     string `ini:"user"`
    Password string `ini:"password"`
    Database string `ini:"database"`
}

func defaultMapper(scheme string) kong.MapperFunc {
    return func(ctx *kong.DecodeContext, target reflect.Value) error {
        var s string
        err := ctx.Scan.PopValueInto("string", &s)
        if err != nil {
            return err
        }
        dsn, err := stringToUrl(s)
        if err != nil {
            return err
        }
        if dsn.Scheme != scheme {
            return ErrInvalidScheme
        }
        if dsn.Hostname() == "" {
            return ErrInvalidHostname
        }
        field := target.Elem().FieldByName("dsn") // 'dsn' MUST exist
        wr := reflect.NewAt(
            field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
        wr.Set(reflect.ValueOf(dsn))
        return nil
    }
}

func Parse(s string) (MySQL, error) {
    dsn, err := stringToUrl(s)
    return MySQL{dsn}, err
}

func stringToUrl(s string) (*url.URL, error) {
    dsn, err := url.Parse(s)
    if err != nil {
        return nil, err
    }
    // sanitize path (i.e. database name)
    if len(dsn.Path) == 0 || dsn.Path[0] != '/' {
        dsn.Path = "/"
        dsn.RawPath = ""
    }

    q := dsn.Query()
    if _, exists := q["defaultsFile"]; exists {
        cfg := urlToConfig(dsn)

        group := "client"
        if _, exists := q["defaultsGroup"]; exists {
            group = q["defaultsGroup"][0]
        }
        cfg2, err := loadConfigFromIniFile(q["defaultsFile"][0], group)
        if err != nil {
            return nil, err
        }
        if err := mergeConfigs(&cfg, cfg2); err != nil {
            return nil, err
        }

        dsn.User = url.UserPassword(cfg.User, cfg.Password)
        dsn.Host = cfg.Host
        if cfg.Port != "" {
            dsn.Host += ":" + cfg.Port
        }
        dsn.Path = "/" + cfg.Database

        q.Del("defaultsFile")
        q.Del("defaultsGroup")
        dsn.RawQuery = q.Encode()
    }

    if dsn.User.Username() == "" {
        pass, _ := dsn.User.Password()
        u, err := user.Current()
        if err != nil {
            return nil, err
        }
        dsn.User = url.UserPassword(u.Username, pass)
    }
    return dsn, nil
}

func urlToConfig(dsn *url.URL) iniConfig {
    return iniConfig{
        Host:     dsn.Hostname(),
        Port:     dsn.Port(),
        User:     dsn.User.Username(),
        Password: func () string {
            p, _ := dsn.User.Password()
            return p
        }(),
        Database: func () string {
            // just in case
            if len(dsn.Path) > 0 {
                return dsn.Path[1:]
            }
            return ""
        }(),
    }
}

func loadConfigFromIniFile(filename, group string) (iniConfig, error) {
    var cfg iniConfig
    f, err := ini.Load(filename)
    if err != nil {
        return iniConfig{}, fmt.Errorf("defaults file: %w", err)
    }
    // silently ignores non-existent group name
    if s := f.Section(group); s != nil {
        if err := s.MapTo(&cfg); err != nil {
            return iniConfig{}, err
        }
    }
    return cfg, nil
}

func mergeConfigs(target *iniConfig, source iniConfig) error {
    sourceItem := reflect.ValueOf(source)
    targetItem := reflect.ValueOf(target).Elem()
    for i := 0; i < sourceItem.NumField(); i++ {
        sourceValue := sourceItem.Field(i)
        targetValue := targetItem.Field(i)
        // no need to check if CanSet()
        if targetValue.IsZero() {
            targetValue.Set(sourceValue)
        }
    }
    return nil
}

func askPass(prompt string) (string, error) {
    fmt.Printf("%s: ", prompt)
    pass, err := term.ReadPassword(0)
    if err != nil {
        return "", err
    }
    fmt.Println()
    return string(pass), nil
}