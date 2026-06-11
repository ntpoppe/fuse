package federation

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ntpoppe/fuse/internal/driver"
	"github.com/ntpoppe/fuse/internal/fuseerr"
)

// TargetLookup resolves live database targets for leg execution.
type TargetLookup interface {
	Fetch(id string) (driver.Target, bool)
}

// Runner executes a federated plan: render legs, query in parallel, join in memory.
type Runner struct {
	lookup       TargetLookup
	maxQueryRows int
}

func NewRunner(lookup TargetLookup, maxQueryRows int) *Runner {
	return &Runner{
		lookup:       lookup,
		maxQueryRows: maxQueryRows,
	}
}

// Run executes plan and returns joined (or single-leg projected) rows.
func (r *Runner) Run(ctx context.Context, plan *FederatedPlan) ([]map[string]any, error) {
	if plan == nil {
		return nil, errors.New("nil federated plan")
	}
	if len(plan.Legs) == 0 {
		return nil, errors.New("plan has no legs")
	}
	if len(plan.Legs) > 2 {
		return nil, fmt.Errorf("v1 supports up to 2 legs, got %d", len(plan.Legs))
	}

	legRows, err := r.runLegsParallel(ctx, plan.Legs)
	if err != nil {
		return nil, err
	}

	if len(plan.Legs) == 1 {
		return projectLegRows(legRows[0], plan.SelectCols, plan.Limit), nil
	}

	return HashJoin(legRows[0], legRows[1], plan.Join, plan.SelectCols, plan.Limit)
}

func (r *Runner) runLegsParallel(ctx context.Context, legs []QueryLeg) ([][]map[string]any, error) {
	results := make([][]map[string]any, len(legs))
	errs := make([]error, len(legs))

	var wg sync.WaitGroup
	for i, leg := range legs {
		wg.Add(1)
		go func(i int, leg QueryLeg) {
			defer wg.Done()
			rows, err := r.runLeg(ctx, leg)
			results[i] = rows
			errs[i] = err
		}(i, leg)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (r *Runner) runLeg(ctx context.Context, leg QueryLeg) ([]map[string]any, error) {
	target, ok := r.lookup.Fetch(leg.ConnectionID)
	if !ok {
		return nil, fuseerr.NotFoundError{ID: leg.ConnectionID}
	}

	sql, args, err := target.Dialect().RenderSelect(SelectLegForDriver(leg))
	if err != nil {
		return nil, fmt.Errorf("render leg %q: %w", leg.ConnectionID, err)
	}

	rows, err := target.Query(ctx, sql, args, r.maxQueryRows)
	if err != nil {
		return nil, fmt.Errorf("query leg %q: %w", leg.ConnectionID, err)
	}

	return rows, nil
}

func projectLegRows(rows []map[string]any, selectCols []ColumnRef, limit *int) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, projectSingleLegRow(row, selectCols))
		if limit != nil && len(out) >= *limit {
			break
		}
	}
	return out
}

func projectSingleLegRow(row map[string]any, selectCols []ColumnRef) map[string]any {
	out := make(map[string]any, len(selectCols))
	for _, col := range selectCols {
		out[outputColumnName(col, selectCols)] = row[col.Column]
	}
	return out
}
