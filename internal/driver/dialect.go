package driver

type Dialect interface {
	Kind() Kind
	ValidateReadOnly(sql string) error
	QuoteIdent(name string) string
	Placeholder(index int) string
}
