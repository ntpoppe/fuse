package executor

import (
	"context"
	"errors"

	"github.com/ntpoppe/fuse/internal/federation"
	"github.com/ntpoppe/fuse/internal/registry"
)

var ErrNotImplemented = errors.New("federated query execution is not implemented yet")

type FederatedExecutor struct {
	registry *registry.Registry
}

func NewFederatedExecutor(reg *registry.Registry) *FederatedExecutor {
	return &FederatedExecutor{registry: reg}
}

func (e *FederatedExecutor) ExecuteFederatedQuery(_ context.Context, sql string) ([]map[string]any, error) {
	q, err := federation.Parse(sql)
	if err != nil {
		return nil, err
	}

	if err := federation.ResolveConnections(q, e.registry); err != nil {
		return nil, err
	}

	return nil, ErrNotImplemented
}
