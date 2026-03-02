package config

import (
    "errors"

    "github.com/dbnski/query-stats/dsn"
)

type CLI struct {
    AskPass bool `
                help:"Prompt for MySQL password"`
    SetVar  []string `
                help:"Set a MySQL session variable (name=value)"`
    Mode    string  `
                help:"Column size measurement mode: text = COM_QUERY, binary = COM_STMT_EXECUTE" 
                default:"text" 
                enum:"binary,text"`
    DSN     *dsn.MySQL `
                help:"Syntax is mysql://[user[:password]@]host[:port]/[?options]" 
                required:"" 
                arg:""`
}

func (cli *CLI) ValidateConfig() error {
    if cli.DSN == nil {
        return errors.New("database endpoint is required")
    }

    return nil
}
