package driver

import (
	"fmt"
	"strings"

	"github.com/ntpoppe/fuse/internal/federation"
)

func renderSelect(d Dialect, leg federation.QueryLeg) (string, []any, error) {
	if len(leg.Columns) == 0 {
		return "", nil, fmt.Errorf("leg %q has no columns to select", leg.ConnectionID)
	}

	quotedCols := make([]string, len(leg.Columns))
	for i, col := range leg.Columns {
		quotedCols[i] = d.QuoteIdent(col)
	}

	var b strings.Builder
	b.WriteString("SELECT ")
	b.WriteString(strings.Join(quotedCols, ", "))
	b.WriteString(" FROM ")
	b.WriteString(d.QuoteIdent(leg.Table.Table))

	args := make([]any, 0, len(leg.Where))
	if len(leg.Where) > 0 {
		b.WriteString(" WHERE ")
		for i, pred := range leg.Where {
			if i > 0 {
				b.WriteString(" AND ")
			}
			b.WriteString(d.QuoteIdent(pred.Column.Column))
			b.WriteString(" ")
			b.WriteString(pred.Op)
			b.WriteString(" ")
			b.WriteString(d.Placeholder(len(args) + 1))
			args = append(args, pred.Value)
		}
	}

	sql := b.String()
	if err := validateRenderedSelect(d, sql, args); err != nil {
		return "", nil, err
	}

	return sql, args, nil
}

func validateRenderedSelect(d Dialect, sql string, args []any) error {
	checkSQL := bindPlaceholders(sql, args)
	if d.Kind() == KindSQLite {
		// Vitess parses in MySQL mode; strip SQLite double quotes for shape validation only.
		checkSQL = stripDoubleQuotedIdents(checkSQL)
	}
	if err := d.ValidateReadOnly(checkSQL); err != nil {
		return fmt.Errorf("rendered SQL failed read-only validation: %w", err)
	}
	return nil
}

func bindPlaceholders(sql string, args []any) string {
	var b strings.Builder
	argIdx := 0
	for i := 0; i < len(sql); i++ {
		if sql[i] == '?' {
			if argIdx < len(args) {
				b.WriteString(validationLiteral(args[argIdx]))
				argIdx++
			} else {
				b.WriteString("NULL")
			}
			continue
		}
		b.WriteByte(sql[i])
	}
	return b.String()
}

func validationLiteral(v any) string {
	switch val := v.(type) {
	case string:
		return "'x'"
	case bool:
		if val {
			return "1"
		}
		return "0"
	case nil:
		return "NULL"
	default:
		return fmt.Sprint(val)
	}
}

func stripDoubleQuotedIdents(sql string) string {
	var b strings.Builder
	for i := 0; i < len(sql); i++ {
		if sql[i] != '"' {
			b.WriteByte(sql[i])
			continue
		}

		j := i + 1
		for j < len(sql) {
			if sql[j] == '"' {
				if j+1 < len(sql) && sql[j+1] == '"' {
					j += 2
					continue
				}
				break
			}
			j++
		}
		if j >= len(sql) {
			b.WriteByte(sql[i])
			continue
		}

		ident := strings.ReplaceAll(sql[i+1:j], `""`, `"`)
		b.WriteString(ident)
		i = j
	}
	return b.String()
}
