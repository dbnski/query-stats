# query-stats

A command-line tool that runs a MySQL query and reports execution statistics such as row counts, data sizes, column-level metrics, and session status changes.

## Usage

```
query-stats <dsn> [--set-var name=value ...] [--mode text|binary]
```

```sh
# Pipe a query
echo "SELECT * FROM orders WHERE status = 'pending'" | query-stats mysql://user:pass@localhost/mydb

# Interactive prompt (Ctrl+D to run, Ctrl+C to abort)
query-stats mysql://user@localhost/mydb

# Set a session variable
echo "SELECT ..." | query-stats mysql://user@localhost/mydb --set-var optimizer_switch=mrr=off
```

## DSN Format

```
mysql://[user[:password]@]host[:port]/[database][?options]
```

Credentials can also be loaded from a MySQL defaults file:

```
mysql://host/db?defaultsFile=~/.my.cnf&defaultsGroup=readonly
```

If no user is specified, the current OS user is used. If no port is specified, 3306 is used.

## Size Measurement Modes

`--mode text` (default) — measures actual wire bytes as sent by MySQL over COM_QUERY. Integer and float column sizes vary with the value magnitude; temporal columns are their canonical string lengths (e.g. DATE is always 10 bytes).

`--mode binary` — reports type storage sizes as used in the binary protocol: fixed widths for integers and floats, disk-storage sizes for temporal types. Useful for estimating how much data a schema stores rather than how much a specific query transfers.
