package executor

import (
	"context"
	"fmt"

	"github.com/ntpoppe/fuse/internal/registry"
)

type Executor struct {
	registry *registry.Registry
}

func NewExecutor(reg *registry.Registry) *Executor {
	return &Executor{registry: reg}
}

func (e *Executor) ExecuteQuery(ctx context.Context, id string, sql string) ([]map[string]any, error) {
	dbConn, exists := e.registry.Fetch(id)
	if !exists {
		return nil, fmt.Errorf("id %q does not exist in registry", id)
	}

	rows, err := dbConn.QueryContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("error querying: %w", err)
	}

	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("rows are closed: %w", err)
	}

	var result []map[string]any

	colCount := len(cols)
	valuesBuffer := make([]any, colCount)
	pointersBuffer := make([]any, colCount)

	for i := 0; i < colCount; i++ {
		pointersBuffer[i] = &valuesBuffer[i]
	}

	for rows.Next() {
		if err := rows.Scan(pointersBuffer...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowMap := make(map[string]any)
		for i, colName := range cols {
			val := valuesBuffer[i]

			// many relational database drivers default into converting flexible types (VARHCAR, raw blobs, etc.)
			// into raw byte slices. if we attempt to serialize this into JSON, Go encodes this as a Base64 string.
			// prevent this by converting it into a Go string before setting in rowMap.
			if bytes, ok := val.([]byte); ok {
				rowMap[colName] = string(bytes)
			} else {
				rowMap[colName] = val
			}
		}

		result = append(result, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred during row streaming: %w", err)
	}

	return result, nil
}
