package federation

import (
	"fmt"
)

// HashJoin performs an in-memory INNER JOIN on two leg result sets.
func HashJoin(leftRows, rightRows []map[string]any, join JoinSpec, selectCols []ColumnRef, limit *int) ([]map[string]any, error) {
	if join.Kind != InnerJoin {
		return nil, fmt.Errorf("unsupported join kind %v", join.Kind)
	}

	buildSide, probeSide := leftSide, rightSide
	if len(rightRows) < len(leftRows) {
		buildSide, probeSide = rightSide, leftSide
	}

	buildRows := rowsForSide(buildSide, leftRows, rightRows)
	probeRows := rowsForSide(probeSide, leftRows, rightRows)
	buildKey := keyColumnForSide(buildSide, join)
	probeKey := keyColumnForSide(probeSide, join)

	hash := make(map[any][]map[string]any)
	for _, row := range buildRows {
		key := row[buildKey]
		hash[key] = append(hash[key], row)
	}

	maxRows := 0
	if limit != nil {
		maxRows = *limit
	}

	out := make([]map[string]any, 0)
	for _, probeRow := range probeRows {
		matches, ok := hash[probeRow[probeKey]]
		if !ok {
			continue
		}

		for _, buildRow := range matches {
			leftRow, rightRow := rowsForJoinSides(buildSide, buildRow, probeRow)
			out = append(out, projectRow(leftRow, rightRow, join, selectCols))

			if maxRows > 0 && len(out) >= maxRows {
				return out, nil
			}
		}
	}

	return out, nil
}

type joinSide int

const (
	leftSide joinSide = iota
	rightSide
)

func rowsForSide(side joinSide, leftRows, rightRows []map[string]any) []map[string]any {
	if side == leftSide {
		return leftRows
	}
	return rightRows
}

func keyColumnForSide(side joinSide, join JoinSpec) string {
	if side == leftSide {
		return join.LeftKey.Column
	}
	return join.RightKey.Column
}

func rowsForJoinSides(buildSide joinSide, buildRow, probeRow map[string]any) (leftRow, rightRow map[string]any) {
	if buildSide == leftSide {
		return buildRow, probeRow
	}
	return probeRow, buildRow
}

func projectRow(leftRow, rightRow map[string]any, join JoinSpec, selectCols []ColumnRef) map[string]any {
	out := make(map[string]any, len(selectCols))
	for _, col := range selectCols {
		name := outputColumnName(col, selectCols)
		switch col.Table {
		case join.LeftAlias:
			out[name] = leftRow[col.Column]
		case join.RightAlias:
			out[name] = rightRow[col.Column]
		default:
			out[name] = nil
		}
	}
	return out
}

func outputColumnName(col ColumnRef, selectCols []ColumnRef) string {
	count := 0
	for _, candidate := range selectCols {
		if candidate.Column == col.Column {
			count++
		}
	}
	if count == 1 {
		return col.Column
	}
	return col.Table + "." + col.Column
}
