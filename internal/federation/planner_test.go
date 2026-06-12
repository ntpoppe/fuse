package federation

import (
	"reflect"
	"testing"
)

func TestPlanTwoTableJoin(t *testing.T) {
	q, err := Parse(validJoinSQL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	plan, err := Plan(q)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if len(plan.Legs) != 2 {
		t.Fatalf("Legs len = %d, want 2", len(plan.Legs))
	}

	left := plan.Legs[0]
	if left.ConnectionID != "billing" || left.Table.Alias != "u" {
		t.Fatalf("left leg = %+v", left)
	}
	if !reflect.DeepEqual(left.Columns, []string{"id", "name", "active"}) {
		t.Fatalf("left columns = %v, want [id name active]", left.Columns)
	}
	if len(left.Where) != 1 || left.Where[0].Column.Column != "active" {
		t.Fatalf("left where = %+v", left.Where)
	}

	right := plan.Legs[1]
	if right.ConnectionID != "analytics" || right.Table.Alias != "o" {
		t.Fatalf("right leg = %+v", right)
	}
	if !reflect.DeepEqual(right.Columns, []string{"total", "user_id"}) {
		t.Fatalf("right columns = %v, want [total user_id]", right.Columns)
	}
	if len(right.Where) != 0 {
		t.Fatalf("right where = %+v, want none", right.Where)
	}

	if plan.Join == nil || plan.Join.LeftAlias != "u" || plan.Join.RightAlias != "o" {
		t.Fatalf("join = %+v", plan.Join)
	}
	if plan.Limit == nil || *plan.Limit != 100 {
		t.Fatalf("Limit = %v, want 100", plan.Limit)
	}
	if len(plan.SelectCols) != 3 {
		t.Fatalf("SelectCols len = %d, want 3", len(plan.SelectCols))
	}
}

func TestPlanSameConnectionTwoTables(t *testing.T) {
	q, err := Parse(`
SELECT u.id, o.total
FROM billing.users u
JOIN billing.orders o ON u.id = o.user_id
WHERE u.active = 1`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	plan, err := Plan(q)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if len(plan.Legs) != 2 {
		t.Fatalf("Legs len = %d, want 2", len(plan.Legs))
	}
	if plan.Legs[0].ConnectionID != "billing" || plan.Legs[1].ConnectionID != "billing" {
		t.Fatalf("legs = %+v, want two billing legs", plan.Legs)
	}
	if plan.Legs[0].Table.Table != "users" || plan.Legs[1].Table.Table != "orders" {
		t.Fatalf("leg tables = %q and %q", plan.Legs[0].Table.Table, plan.Legs[1].Table.Table)
	}
}

func TestPlanSingleTable(t *testing.T) {
	q, err := Parse(`SELECT u.id, u.name FROM billing.users u WHERE u.active = 1 LIMIT 10`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	plan, err := Plan(q)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if len(plan.Legs) != 1 {
		t.Fatalf("Legs len = %d, want 1", len(plan.Legs))
	}
	if !reflect.DeepEqual(plan.Legs[0].Columns, []string{"id", "name", "active"}) {
		t.Fatalf("columns = %v", plan.Legs[0].Columns)
	}
	if len(plan.Legs[0].Where) != 1 {
		t.Fatalf("where = %+v", plan.Legs[0].Where)
	}
	if plan.Join != nil {
		t.Fatalf("join = %+v, want nil", plan.Join)
	}
}

func TestPlanNilQuery(t *testing.T) {
	_, err := Plan(nil)
	if err == nil {
		t.Fatal("expected error for nil query")
	}
}

func TestPlanWherePushdownPerLeg(t *testing.T) {
	q, err := Parse(`
SELECT u.id, o.total
FROM billing.users u
JOIN analytics.orders o ON u.id = o.user_id
WHERE u.active = 1 AND o.amount > 50`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	plan, err := Plan(q)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if len(plan.Legs[0].Where) != 1 || plan.Legs[0].Where[0].Column.Column != "active" {
		t.Fatalf("left where = %+v", plan.Legs[0].Where)
	}
	if len(plan.Legs[1].Where) != 1 || plan.Legs[1].Where[0].Column.Column != "amount" {
		t.Fatalf("right where = %+v", plan.Legs[1].Where)
	}
	if !reflect.DeepEqual(plan.Legs[1].Columns, []string{"total", "user_id", "amount"}) {
		t.Fatalf("right columns = %v", plan.Legs[1].Columns)
	}
}
