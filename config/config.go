package config

import (
    "errors"

    "github.com/dbnski/query-stats/dsn"
)

type CLI struct {
    DSN    *dsn.MySQL `
                help:"Syntax is mysql://[user[:password]@]host[:port]/[?options]" 
                required:"" 
                arg:""`
    SetVar []string `
                help:"Set a MySQL session variable (name=value)"`
}

func (cli *CLI) ValidateConfig() error {
    if cli.DSN == nil {
        return errors.New("database endpoint is required")
    }

    return nil
}
