package executor

import (
	"context"

	"github.com/ntpoppe/fuse/internal/fuseerr"
	"github.com/ntpoppe/fuse/internal/registry"
)

type Executor struct {
	registry     *registry.Registry
	maxQueryRows int
}

func NewExecutor(reg *registry.Registry, maxQueryRows int) *Executor {
	return &Executor{
		registry:     reg,
		maxQueryRows: maxQueryRows,
	}
}

func (e *Executor) ExecuteQuery(ctx context.Context, id, sql string) ([]map[string]any, error) {
	target, exists := e.registry.Fetch(id)
	if !exists {
		return nil, fuseerr.NotFoundError{ID: id}
	}

	if err := target.Dialect().ValidateReadOnly(sql); err != nil {
		return nil, fuseerr.ReadOnlyError{Cause: err}
	}

	return target.Query(ctx, sql, nil, e.maxQueryRows)
}
