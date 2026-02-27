package runner

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/mysql"

	"github.com/dbnski/query-stats/dsn"
)

type statusGroup struct {
	title string
	vars  []string
}

var statusGroups = []statusGroup{
	{
		title: "Rows Examined (Handler)",
		vars: []string{
			"Handler_read_first",
			"Handler_read_key",
			"Handler_read_last",
			"Handler_read_next",
			"Handler_read_prev",
			"Handler_read_rnd",
			"Handler_read_rnd_next",
		},
	},
	{
		title: "Temp Tables",
		vars: []string{
			"Created_tmp_disk_tables",
			"Created_tmp_files",
			"Created_tmp_tables",
		},
	},
	{
		title: "Sort",
		vars: []string{
			"Sort_merge_passes",
			"Sort_range",
			"Sort_rows",
			"Sort_scan",
		},
	},
	{
		title: "Select",
		vars: []string{
			"Select_full_join",
			"Select_full_range_join",
			"Select_range",
			"Select_range_check",
			"Select_scan",
		},
	},
}

type colStats struct {
	name       string
	typeName   string
	typeCode   byte
	nullable   bool
	minLen     int
	maxLen     int
	sumLen     int64
	count      int64
	nullCount  int64
	emptyCount int64
}

func getSessionStatus(conn *client.Conn) (map[string]int64, error) {
	r, err := conn.Execute("SHOW SESSION STATUS")
	if err != nil {
		return nil, fmt.Errorf("show session status: %w", err)
	}
	status := make(map[string]int64)
	for _, row := range r.Values {
		name := string(row[0].AsString())
		valStr := string(row[1].AsString())
		var v int64
		fmt.Sscan(valStr, &v)
		status[name] = v
	}
	return status, nil
}

func parseValue(s string) interface{} {
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

func valueBytes(v interface{}) []byte {
	switch val := v.(type) {
	case []byte:
		return val
	case string:
		return []byte(val)
	default:
		return []byte(fmt.Sprintf("%v", val))
	}
}

func setSessionVars(conn *client.Conn, vars []string) error {
	for _, v := range vars {
		idx := strings.IndexByte(v, '=')
		if idx < 1 || idx == len(v)-1 {
			return fmt.Errorf("--set-var: expected name=value, got '%q'", v)
		}
		name, value := v[:idx], v[idx+1:]
		if _, err := conn.Execute(fmt.Sprintf("SET SESSION `%s` = ?", name), parseValue(value)); err != nil {
			return fmt.Errorf("set-var %s: %w", name, err)
		}
	}
	return nil
}

func fieldTypeName(t byte) string {
	switch t {
	case mysql.MYSQL_TYPE_DECIMAL:
		return "DECIMAL"
	case mysql.MYSQL_TYPE_TINY:
		return "TINYINT"
	case mysql.MYSQL_TYPE_SHORT:
		return "SMALLINT"
	case mysql.MYSQL_TYPE_LONG:
		return "INT"
	case mysql.MYSQL_TYPE_FLOAT:
		return "FLOAT"
	case mysql.MYSQL_TYPE_DOUBLE:
		return "DOUBLE"
	case mysql.MYSQL_TYPE_NULL:
		return "NULL"
	case mysql.MYSQL_TYPE_TIMESTAMP, mysql.MYSQL_TYPE_TIMESTAMP2:
		return "TIMESTAMP"
	case mysql.MYSQL_TYPE_LONGLONG:
		return "BIGINT"
	case mysql.MYSQL_TYPE_INT24:
		return "MEDIUMINT"
	case mysql.MYSQL_TYPE_DATE, mysql.MYSQL_TYPE_NEWDATE:
		return "DATE"
	case mysql.MYSQL_TYPE_TIME, mysql.MYSQL_TYPE_TIME2:
		return "TIME"
	case mysql.MYSQL_TYPE_DATETIME, mysql.MYSQL_TYPE_DATETIME2:
		return "DATETIME"
	case mysql.MYSQL_TYPE_YEAR:
		return "YEAR"
	case mysql.MYSQL_TYPE_VARCHAR, mysql.MYSQL_TYPE_VAR_STRING:
		return "VARCHAR"
	case mysql.MYSQL_TYPE_BIT:
		return "BIT"
	case mysql.MYSQL_TYPE_JSON:
		return "JSON"
	case mysql.MYSQL_TYPE_NEWDECIMAL:
		return "DECIMAL"
	case mysql.MYSQL_TYPE_ENUM:
		return "ENUM"
	case mysql.MYSQL_TYPE_SET:
		return "SET"
	case mysql.MYSQL_TYPE_TINY_BLOB:
		return "TINYBLOB"
	case mysql.MYSQL_TYPE_MEDIUM_BLOB:
		return "MEDIUMBLOB"
	case mysql.MYSQL_TYPE_LONG_BLOB:
		return "LONGBLOB"
	case mysql.MYSQL_TYPE_BLOB:
		return "BLOB"
	case mysql.MYSQL_TYPE_STRING:
		return "CHAR"
	case mysql.MYSQL_TYPE_GEOMETRY:
		return "GEOMETRY"
	default:
		return fmt.Sprintf("TYPE(%d)", t)
	}
}

func isStringType(t byte) bool {
	switch t {
	case mysql.MYSQL_TYPE_VARCHAR,
		mysql.MYSQL_TYPE_VAR_STRING,
		mysql.MYSQL_TYPE_STRING,
		mysql.MYSQL_TYPE_BLOB,
		mysql.MYSQL_TYPE_TINY_BLOB,
		mysql.MYSQL_TYPE_MEDIUM_BLOB,
		mysql.MYSQL_TYPE_LONG_BLOB,
		mysql.MYSQL_TYPE_ENUM,
		mysql.MYSQL_TYPE_SET:
		return true
	}
	return false
}

func initColStats(fields []*mysql.Field) []colStats {
	s := make([]colStats, len(fields))
	for i, f := range fields {
		s[i].name = string(f.Name)
		s[i].typeName = fieldTypeName(f.Type)
		s[i].typeCode = f.Type
		s[i].nullable = f.Flag&mysql.NOT_NULL_FLAG == 0
		s[i].minLen = math.MaxInt32
	}
	return s
}

func Run(d *dsn.MySQL, query string, setVars []string) error {
	addr := fmt.Sprintf("%s:%d", d.Host(), d.Port())
	conn, err := client.Connect(addr, d.User(), d.Password(), d.Db())
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	if err := setSessionVars(conn, setVars); err != nil {
		return err
	}

	before, err := getSessionStatus(conn)
	if err != nil {
		return err
	}

	var (
		stats      []colStats
		rowCount   int64
		totalSize  int64
		minRowSize int64 = math.MaxInt64
		maxRowSize int64
	)

	result := &mysql.Result{}
	start := time.Now()
	err = conn.ExecuteSelectStreaming(query, result,
		func(row []mysql.FieldValue) error {
			if stats == nil {
				stats = initColStats(result.Fields)
			}
			rowCount++
			var rowSize int64
			for i := range row {
				v := row[i].Value()
				if v == nil {
					stats[i].nullCount++
					continue
				}
				b := valueBytes(v)
				l := len(b)
				if l == 0 {
					stats[i].emptyCount++
				}
				if l < stats[i].minLen {
					stats[i].minLen = l
				}
				if l > stats[i].maxLen {
					stats[i].maxLen = l
				}
				stats[i].sumLen += int64(l)
				stats[i].count++
				rowSize += int64(l)
			}
			totalSize += rowSize
			if rowSize < minRowSize {
				minRowSize = rowSize
			}
			if rowSize > maxRowSize {
				maxRowSize = rowSize
			}
			return nil
		},
		nil,
	)
	elapsed := time.Since(start)

	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	// Empty result set: no rows were scanned, init stats from fields.
	if stats == nil {
		stats = initColStats(result.Fields)
	}

	after, err := getSessionStatus(conn)
	if err != nil {
		return err
	}

	printResults(elapsed, rowCount, totalSize, minRowSize, maxRowSize, before, after, stats)
	return nil
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%.2f µs", float64(d.Nanoseconds())/1000)
	case d < time.Second:
		return fmt.Sprintf("%.2f ms", float64(d.Nanoseconds())/1e6)
	default:
		return fmt.Sprintf("%.3f s", d.Seconds())
	}
}

func formatBytes(n int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case n < kb:
		return fmt.Sprintf("%d B", n)
	case n < mb:
		return fmt.Sprintf("%.1f KB", float64(n)/kb)
	case n < gb:
		return fmt.Sprintf("%.1f MB", float64(n)/mb)
	default:
		return fmt.Sprintf("%.2f GB", float64(n)/gb)
	}
}

func formatInt(n int64) string {
	return fmt.Sprintf("%d", n)
}

func printResults(elapsed time.Duration, rowCount, totalSize, minRowSize, maxRowSize int64, before, after map[string]int64, cols []colStats) {
	// Execution time
	fmt.Println("=== Query Execution ===")
	fmt.Printf("  Execution time:   %s\n", formatDuration(elapsed))
	fmt.Println()

	// Session status
	printSessionStatus(before, after)

	// Result summary
	fmt.Println("=== Result Summary ===")
	fmt.Printf("  Rows returned:    %s\n", formatInt(rowCount))
	if rowCount > 0 {
		fmt.Printf("  Total data size:  %s\n", formatBytes(totalSize))
		fmt.Printf("  Min row size:     %s\n", formatBytes(minRowSize))
		fmt.Printf("  Avg row size:     %s\n", formatBytes(totalSize/rowCount))
		fmt.Printf("  Max row size:     %s\n", formatBytes(maxRowSize))
	} else {
		fmt.Printf("  Total data size:  0 B\n")
		fmt.Printf("  Min row size:     0 B\n")
		fmt.Printf("  Avg row size:     0 B\n")
		fmt.Printf("  Max row size:     0 B\n")
	}
	fmt.Println()

	// Column statistics
	printColumnStats(cols)
}

func printSessionStatus(before, after map[string]int64) {
	hasAny := false
	for _, grp := range statusGroups {
		for _, v := range grp.vars {
			if after[v]-before[v] != 0 {
				hasAny = true
				break
			}
		}
	}
	if !hasAny {
		return
	}

	fmt.Println("=== Session Status Changes ===")
	for _, grp := range statusGroups {
		// Collect nonzero diffs for this group
		type entry struct {
			name string
			diff int64
		}
		var entries []entry
		for _, v := range grp.vars {
			d := after[v] - before[v]
			if d != 0 {
				entries = append(entries, entry{v, d})
			}
		}
		if len(entries) == 0 {
			continue
		}

		fmt.Printf("  %s:\n", grp.title)
		// Compute name column width
		maxNameLen := 0
		for _, e := range entries {
			if len(e.name) > maxNameLen {
				maxNameLen = len(e.name)
			}
		}
		for _, e := range entries {
			fmt.Printf("    %-*s  %s\n", maxNameLen, e.name, formatInt(e.diff))
		}
	}
	fmt.Println()
}

func printColumnStats(cols []colStats) {
	if len(cols) == 0 {
		return
	}

	type row struct {
		name, typeName, minLen, maxLen, avgLen, totalBytes, empty, nulls string
	}

	rows := make([]row, len(cols))
	for i, c := range cols {
		minStr, maxStr, avgStr, totalBytesStr := "-", "-", "-", "-"
		if c.count > 0 {
			minStr = formatBytes(int64(c.minLen))
			maxStr = formatBytes(int64(c.maxLen))
			avgStr = formatBytes(c.sumLen / c.count)
			totalBytesStr = formatBytes(c.sumLen)
		}
		emptyStr := "-"
		if isStringType(c.typeCode) {
			emptyStr = formatInt(c.emptyCount)
		}
		nullsStr := "-"
		if c.nullable {
			nullsStr = formatInt(c.nullCount)
		}
		rows[i] = row{
			name:       c.name,
			typeName:   c.typeName,
			minLen:     minStr,
			maxLen:     maxStr,
			avgLen:     avgStr,
			totalBytes: totalBytesStr,
			empty:      emptyStr,
			nulls:      nullsStr,
		}
	}

	// Compute column widths
	headers := []string{"Column", "Type", "MinLen", "MaxLen", "AvgLen", "Total", "Empty", "Null"}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, r := range rows {
		cells := []string{r.name, r.typeName, r.minLen, r.maxLen, r.avgLen, r.totalBytes, r.empty, r.nulls}
		for i, c := range cells {
			if len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}

	sep := func() string {
		parts := make([]string, len(widths))
		for i, w := range widths {
			parts[i] = strings.Repeat("-", w)
		}
		return "  " + strings.Join(parts, "  ")
	}

	printRow := func(cells []string, rightAlign []bool) {
		parts := make([]string, len(cells))
		for i, c := range cells {
			if rightAlign != nil && rightAlign[i] {
				parts[i] = fmt.Sprintf("%*s", widths[i], c)
			} else {
				parts[i] = fmt.Sprintf("%-*s", widths[i], c)
			}
		}
		fmt.Println("  " + strings.Join(parts, "  "))
	}

	fmt.Println("=== Column Statistics ===")
	printRow(headers, nil)
	fmt.Println(sep())
	rightAlign := []bool{false, false, true, true, true, true, true, true}
	for _, r := range rows {
		printRow([]string{r.name, r.typeName, r.minLen, r.maxLen, r.avgLen, r.totalBytes, r.empty, r.nulls}, rightAlign)
	}
	fmt.Println()
}
