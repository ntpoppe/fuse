package executor

import (
	"context"

	"github.com/ntpoppe/fuse/internal/federation"
	"github.com/ntpoppe/fuse/internal/registry"
)

type FederatedExecutor struct {
	registry     *registry.Registry
	maxQueryRows int
}

func NewFederatedExecutor(reg *registry.Registry, maxQueryRows int) *FederatedExecutor {
	return &FederatedExecutor{
		registry:     reg,
		maxQueryRows: maxQueryRows,
	}
}

func (e *FederatedExecutor) ExecuteFederatedQuery(ctx context.Context, sql string) ([]map[string]any, error) {
	q, err := federation.Parse(sql)
	if err != nil {
		return nil, err
	}

	if err := federation.ResolveConnections(q, e.registry); err != nil {
		return nil, err
	}

	plan, err := federation.Plan(q)
	if err != nil {
		return nil, err
	}

	runner := federation.NewRunner(e.registry, e.maxQueryRows)
	return runner.Run(ctx, plan)
}
