package executor

import (
	"context"
	"fmt"

	"github.com/ntpoppe/fuse/internal/fuseerr"
	"github.com/ntpoppe/fuse/internal/registry"
)

type Executor struct {
	registry *registry.Registry
}

func NewExecutor(reg *registry.Registry) *Executor {
	return &Executor{registry: reg}
}

func (e *Executor) ExecuteQuery(ctx context.Context, id, sql string) ([]map[string]any, error) {
	target, exists := e.registry.Fetch(id)
	if !exists {
		return nil, fuseerr.NotFoundError{ID: id}
	}

	if err := target.Dialect().ValidateReadOnly(sql); err != nil {
		return nil, fmt.Errorf("read-only violation: %w", err)
	}

	return target.Query(ctx, sql)
}
