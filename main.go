package main

import (
    "bufio"
    "fmt"
    "io"
    "os"
    "reflect"
    "strings"

    "github.com/dbnski/query-stats/config"
    "github.com/dbnski/query-stats/dsn"
    "github.com/dbnski/query-stats/runner"
    "github.com/alecthomas/kong"
    "golang.org/x/term"
)

var (
    Version    = "0.0.0"
    CommitHash = "0000000"
    Build      = "dev"
    BuildTime  = "(recently)"
)

func readQueryFromTerminal() (string, error) {
    fd := int(os.Stdin.Fd())
    old, err := term.MakeRaw(fd)
    if err != nil {
        data, err := io.ReadAll(os.Stdin)
        return string(data), err
    }
    defer term.Restore(fd, old)

    fmt.Fprint(os.Stderr, "Enter query (Ctrl+D to run, Ctrl+C to abort):\r\n")

    var buf []byte
    in := bufio.NewReader(os.Stdin)

    for {
        b, err := in.ReadByte()
        if err != nil {
            if err == io.EOF {
                break
            }
            return "", err
        }

        switch b {
        case 0x03: // Ctrl+C
            fmt.Fprint(os.Stderr, "^C\r\n")
            os.Exit(130)

        case 0x04: // Ctrl+D — end of input
            fmt.Fprint(os.Stderr, "\r\n")
            return string(buf), nil

        case 0x7f, 0x08: // Backspace / DEL
            if len(buf) > 0 {
                buf = buf[:len(buf)-1]
                fmt.Fprint(os.Stderr, "\b \b")
            }

        case '\r': // Enter — newline in the query
            buf = append(buf, '\n')
            fmt.Fprint(os.Stderr, "\r\n")

        case 0x1b: // Escape — swallow CSI sequences (arrow keys, etc.)
            if peeked, _ := in.Peek(1); len(peeked) > 0 && peeked[0] == '[' {
                in.ReadByte() // consume '['
                for {
                    c, err := in.ReadByte()
                    if err != nil {
                        break
                    }
                    // CSI sequences end at a letter or '~'
                    if c >= 0x40 && c <= 0x7e {
                        break
                    }
                }
            }

        default:
            buf = append(buf, b)
            fmt.Fprintf(os.Stderr, "%c", b)
        }
    }

    return string(buf), nil
}

func main() {
    cli := new(config.CLI)
    kong.Parse(
        cli,
        kong.Description(
            fmt.Sprintf("Version: %s-%s.%s %s", Version, Build, CommitHash, BuildTime)),
        kong.TypeMapper(
            reflect.TypeOf((*dsn.MySQL)(nil)), dsn.MySQLMapper),
        kong.UsageOnError(),
    )

    if err := cli.ValidateConfig(); err != nil {
        fmt.Println("Configuration error:", err.Error())
        os.Exit(1)
    }

    var query string
    if term.IsTerminal(int(os.Stdin.Fd())) {
        q, err := readQueryFromTerminal()
        if err != nil {
            fmt.Fprintln(os.Stderr, "error reading query:", err)
            os.Exit(1)
        }
        query = strings.TrimSpace(q)
    } else {
        raw, err := io.ReadAll(os.Stdin)
        if err != nil {
            fmt.Fprintln(os.Stderr, "error reading stdin:", err)
            os.Exit(1)
        }
        query = strings.TrimSpace(string(raw))
    }

    if query == "" {
        fmt.Fprintln(os.Stderr, "error: empty query")
        os.Exit(1)
    }

    if err := runner.Run(cli.DSN, query, cli.SetVar); err != nil {
        fmt.Fprintln(os.Stderr, "error:", err)
        os.Exit(1)
    }
}
