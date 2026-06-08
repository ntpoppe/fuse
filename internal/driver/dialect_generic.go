package driver

import (
	"github.com/ntpoppe/fuse/internal/driver/validate"
	"github.com/ntpoppe/fuse/internal/federation"
)

type genericDialect struct {
	kind Kind
}

func dialectFor(driverName string) Dialect {
	switch driverName {
	case DriverSQLite:
		return NewSQLiteDialect()
	case DriverMySQL:
		return NewMySQLDialect()
	default:
		return genericDialect{kind: KindFromDriver(driverName)}
	}
}

// GenericDialect returns the fallback dialect for tests and unknown drivers.
func GenericDialect(kind Kind) Dialect {
	return genericDialect{kind: kind}
}

func (d genericDialect) Kind() Kind {
	return d.kind
}

func (d genericDialect) ValidateReadOnly(sql string) error {
	return validate.ReadOnlySQL(sql, validate.OptionsStandard)
}

func (d genericDialect) QuoteIdent(name string) string {
	return doubleQuoteIdent(name)
}

func (d genericDialect) Placeholder(index int) string {
	return questionPlaceholder(index)
}

func (d genericDialect) RenderSelect(leg federation.QueryLeg) (string, []any, error) {
	return renderSelect(d, leg)
}
