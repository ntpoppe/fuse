package driver

import (
	"context"
	"database/sql"
	"fmt"
)

type sqlTarget struct {
	id      string
	kind    Kind
	dialect Dialect
	db      *sql.DB
}

func newSQLTarget(id, driverName string, db *sql.DB) *sqlTarget {
	return &sqlTarget{
		id:      id,
		kind:    KindFromDriver(driverName),
		dialect: dialectFor(driverName),
		db:      db,
	}
}

func (t *sqlTarget) ID() string {
	return t.id
}

func (t *sqlTarget) Kind() Kind {
	return t.kind
}

func (t *sqlTarget) Dialect() Dialect {
	return t.dialect
}

func (t *sqlTarget) Ping(ctx context.Context) error {
	return t.db.PingContext(ctx)
}

func (t *sqlTarget) Close() error {
	return t.db.Close()
}

func (t *sqlTarget) Query(ctx context.Context, sql string) ([]map[string]any, error) {
	rows, err := t.db.QueryContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("error querying: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("rows are closed: %w", err)
	}

	colCount := len(cols)
	valuesBuffer := make([]any, colCount)
	pointersBuffer := make([]any, colCount)
	for i := range colCount {
		pointersBuffer[i] = &valuesBuffer[i]
	}

	var result []map[string]any
	for rows.Next() {
		if err := rows.Scan(pointersBuffer...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowMap := make(map[string]any, colCount)
		for i, colName := range cols {
			val := valuesBuffer[i]
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
