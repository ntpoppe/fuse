package driver

import "strings"

func doubleQuoteIdent(name string) string {
	escaped := strings.ReplaceAll(name, `"`, `""`)
	return `"` + escaped + `"`
}

func backtickQuoteIdent(name string) string {
	escaped := strings.ReplaceAll(name, "`", "``")
	return "`" + escaped + "`"
}

func questionPlaceholder(int) string {
	return "?"
}
