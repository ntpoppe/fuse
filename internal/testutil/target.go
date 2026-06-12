package testutil

import (
	"context"

	"github.com/ntpoppe/fuse/internal/driver"
)

type StubTarget struct {
	IDVal   string
	QueryFn func(context.Context, string, []any, int) ([]map[string]any, error)
}

func NewStubTarget(id string) *StubTarget {
	return &StubTarget{IDVal: id}
}

func (s *StubTarget) ID() string {
	return s.IDVal
}

func (s *StubTarget) Kind() driver.Kind {
	return driver.KindUnknown
}

func (s *StubTarget) Dialect() driver.Dialect {
	return driver.GenericDialect(driver.KindUnknown)
}

func (s *StubTarget) Ping(context.Context) error {
	return nil
}

func (s *StubTarget) Close() error {
	return nil
}

func (s *StubTarget) Query(ctx context.Context, query string, args []any, maxRows int) ([]map[string]any, error) {
	if s.QueryFn != nil {
		return s.QueryFn(ctx, query, args, maxRows)
	}
	return nil, nil
}
