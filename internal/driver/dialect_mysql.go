package driver

import "github.com/ntpoppe/fuse/internal/driver/validate"

type mysqlDialect struct{}

func NewMySQLDialect() Dialect {
	return mysqlDialect{}
}

func (mysqlDialect) Kind() Kind {
	return KindMySQL
}

func (mysqlDialect) ValidateReadOnly(sql string) error {
	return validate.ReadOnlySQL(sql, validate.OptionsMySQL)
}

func (mysqlDialect) QuoteIdent(name string) string {
	return backtickQuoteIdent(name)
}

func (mysqlDialect) Placeholder(index int) string {
	return questionPlaceholder(index)
}

func (d mysqlDialect) RenderSelect(leg SelectLeg) (string, []any, error) {
	return renderSelect(d, leg)
}
