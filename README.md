# query-stats

A command-line tool that runs a MySQL query and reports execution statistics such as row counts, data sizes, column-level metrics, and session status changes.

## Usage

```
query-stats <dsn> [--set-var name=value ...] [--mode text|binary] [--ask-pass]
```

```sh
# Interactive prompt (Ctrl+D to run, Ctrl+C to abort)
query-stats mysql://address/dbname

# Pipe a query
echo "SELECT * FROM mydb.orders WHERE status = 'open'" | query-stats mysql://user:pass@address/

# Set a session variable
query-stats --set-var optimizer_switch=mrr=off mysql://user:pass@adress/

# Load credential file
query-stats mysql://address/?defaultsFile=~/.my.cnf

# Enable the connection encryption
query-stats mysql://user:pass@address/?ssl

# Set the connection encoding & collation
query-stats mysql://user:pass@address/?collation=utf8mb4_unicode_ci
```

## DSN Format

```
mysql://[user[:password]@]host[:port]/[database][?options]
```

If no user is specified, the current OS user is used. If no port is specified, 3306 is used.

### DSN Options

| Option | Description |
|--------|-------------|
| `defaultsFile=<path>` | Read connection details from an ini file |
| `defaultsGroup=<group>` | Section to read from the defaults file (default: client) |
| `ssl` | Enable TLS for the connection |
| `collation=<name>` | Set the connection collation and implied character set |

## Size Measurement Modes

`--mode text` (default) - measures actual wire bytes as sent by MySQL over COM_QUERY. Integer and float column sizes vary with the value magnitude; temporal columns are their canonical string lengths (e.g. DATE is always 10 bytes).

`--mode binary` - reports type storage sizes as used in the binary protocol: fixed widths for integers and floats, disk-storage sizes for temporal types. Useful for estimating how much data a schema stores rather than how much a specific query transfers.
