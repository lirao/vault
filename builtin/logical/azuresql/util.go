package azuresql

import (
	"fmt"
	"strings"
)

// SplitSQL is used to split a series of SQL statements
func SplitSQL(sql string) []string {
	parts := strings.Split(sql, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		clean := strings.TrimSpace(p)
		if len(clean) > 0 {
			out = append(out, clean)
		}
	}
	return out
}

// Query templates a query for us.
func Query(tpl string, data map[string]string) string {
	for k, v := range data {
		tpl = strings.Replace(tpl, fmt.Sprintf("{{%s}}", k), v, -1)
	}

	return tpl
}

// Parse a MSSQL connection string
func ParseConnString(dsn string) (res map[string]string) {
	res = map[string]string{}
	parts := strings.Split(dsn, ";")
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		lst := strings.SplitN(part, "=", 2)
		name := strings.TrimSpace(strings.ToLower(lst[0]))
		if len(name) == 0 {
			continue
		}
		var value string
		if len(lst) > 1 {
			value = strings.TrimSpace(lst[1])
		}
		res[name] = value
	}
	return res
}
