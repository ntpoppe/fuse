package driver

import "context"

type Target interface {
	ID() string
	Kind() Kind
	Dialect() Dialect
	Ping(ctx context.Context) error
	Close() error
	Query(ctx context.Context, sql string, maxRows int) ([]map[string]any, error)
}
