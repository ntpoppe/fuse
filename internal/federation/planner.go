package federation

import "errors"

// Plan turns a parsed federated query into per-table execution legs.
func Plan(q *ParsedQuery) (*FederatedPlan, error) {
	if q == nil {
		return nil, errors.New("nil parsed query")
	}
	if len(q.Tables) == 0 {
		return nil, errors.New("query has no tables")
	}

	legs := make([]QueryLeg, 0, len(q.Tables))
	for _, table := range q.Tables {
		legs = append(legs, QueryLeg{
			ConnectionID: table.ConnectionID,
			Table:        table,
			Columns:      columnsForLeg(q, table.Alias),
			Where:        whereForLeg(q.Where, table.Alias),
		})
	}

	plan := &FederatedPlan{
		Legs:       legs,
		SelectCols: append([]ColumnRef(nil), q.SelectCols...),
		Limit:      q.Limit,
	}
	if q.Join != nil {
		plan.Join = *q.Join
	}

	return plan, nil
}

func columnsForLeg(q *ParsedQuery, alias string) []string {
	seen := make(map[string]struct{})
	cols := make([]string, 0, len(q.SelectCols)+2)

	add := func(name string) {
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		cols = append(cols, name)
	}

	for _, col := range q.SelectCols {
		if col.Table == alias {
			add(col.Column)
		}
	}

	if q.Join != nil {
		if q.Join.LeftAlias == alias {
			add(q.Join.LeftKey.Column)
		}
		if q.Join.RightAlias == alias {
			add(q.Join.RightKey.Column)
		}
	}

	for _, pred := range q.Where {
		if pred.Column.Table == alias {
			add(pred.Column.Column)
		}
	}

	return cols
}

func whereForLeg(where []Predicate, alias string) []Predicate {
	if len(where) == 0 {
		return nil
	}

	out := make([]Predicate, 0, len(where))
	for _, pred := range where {
		if pred.Column.Table == alias {
			out = append(out, pred)
		}
	}
	return out
}
