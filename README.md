# query-stats

A command-line tool that runs a MySQL query and reports execution statistics such as row counts, data sizes, column-level metrics, and session status changes.

## Usage

```
query-stats <dsn> [--set-var name=value ...]
```

```sh
# Pipe a query
echo "SELECT * FROM orders WHERE status = 'pending'" | query-stats mysql://user:pass@localhost/mydb

# Interactive prompt
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
mysql://host/db?defaultsFile=/etc/mysql/my.cnf&defaultsGroup=client
```
