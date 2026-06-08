package driver

import (
	"github.com/ntpoppe/fuse/internal/driver/validate"
	"github.com/ntpoppe/fuse/internal/federation"
)

type sqliteDialect struct{}

func NewSQLiteDialect() Dialect {
	return sqliteDialect{}
}

func (sqliteDialect) Kind() Kind {
	return KindSQLite
}

func (sqliteDialect) ValidateReadOnly(sql string) error {
	return validate.ReadOnlySQL(sql, validate.OptionsStandard)
}

func (sqliteDialect) QuoteIdent(name string) string {
	return doubleQuoteIdent(name)
}

func (sqliteDialect) Placeholder(index int) string {
	return questionPlaceholder(index)
}

func (d sqliteDialect) RenderSelect(leg federation.QueryLeg) (string, []any, error) {
	return renderSelect(d, leg)
}
