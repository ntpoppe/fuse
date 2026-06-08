package driver

import "github.com/ntpoppe/fuse/internal/federation"

type Dialect interface {
	Kind() Kind
	ValidateReadOnly(sql string) error
	QuoteIdent(name string) string
	Placeholder(index int) string
	RenderSelect(leg federation.QueryLeg) (string, []any, error)
}
