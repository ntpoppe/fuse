package driver_test

import (
	"reflect"
	"testing"

	"github.com/ntpoppe/fuse/internal/driver"
	"github.com/ntpoppe/fuse/internal/federation"
)

func billingUsersLeg() federation.QueryLeg {
	return federation.QueryLeg{
		ConnectionID: "billing",
		Table: federation.QualifiedTable{
			ConnectionID: "billing",
			Table:        "users",
			Alias:        "u",
		},
		Columns: []string{"id", "name", "active"},
		Where: []federation.Predicate{
			{
				Column: federation.ColumnRef{Table: "u", Column: "active"},
				Op:     "=",
				Value:  1,
			},
		},
	}
}

func analyticsOrdersLeg() federation.QueryLeg {
	return federation.QueryLeg{
		ConnectionID: "analytics",
		Table: federation.QualifiedTable{
			ConnectionID: "analytics",
			Table:        "orders",
			Alias:        "o",
		},
		Columns: []string{"user_id", "total"},
	}
}

func TestRenderSelectSQLite(t *testing.T) {
	d := driver.NewSQLiteDialect()
	sql, args, err := d.RenderSelect(billingUsersLeg())
	if err != nil {
		t.Fatalf("RenderSelect() error = %v", err)
	}

	wantSQL := `SELECT "id", "name", "active" FROM "users" WHERE "active" = ?`
	if sql != wantSQL {
		t.Fatalf("sql = %q, want %q", sql, wantSQL)
	}
	if !reflect.DeepEqual(args, []any{1}) {
		t.Fatalf("args = %v, want [1]", args)
	}
}

func TestRenderSelectMySQL(t *testing.T) {
	d := driver.NewMySQLDialect()
	sql, args, err := d.RenderSelect(billingUsersLeg())
	if err != nil {
		t.Fatalf("RenderSelect() error = %v", err)
	}

	wantSQL := "SELECT `id`, `name`, `active` FROM `users` WHERE `active` = ?"
	if sql != wantSQL {
		t.Fatalf("sql = %q, want %q", sql, wantSQL)
	}
	if !reflect.DeepEqual(args, []any{1}) {
		t.Fatalf("args = %v, want [1]", args)
	}
}

func TestRenderSelectWithoutWhere(t *testing.T) {
	d := driver.NewSQLiteDialect()
	sql, args, err := d.RenderSelect(analyticsOrdersLeg())
	if err != nil {
		t.Fatalf("RenderSelect() error = %v", err)
	}

	wantSQL := `SELECT "user_id", "total" FROM "orders"`
	if sql != wantSQL {
		t.Fatalf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 0 {
		t.Fatalf("args = %v, want empty", args)
	}
}

func TestRenderSelectMultiplePredicates(t *testing.T) {
	leg := billingUsersLeg()
	leg.Where = append(leg.Where, federation.Predicate{
		Column: federation.ColumnRef{Table: "u", Column: "name"},
		Op:     "<>",
		Value:  "deleted",
	})

	d := driver.NewMySQLDialect()
	sql, args, err := d.RenderSelect(leg)
	if err != nil {
		t.Fatalf("RenderSelect() error = %v", err)
	}

	wantSQL := "SELECT `id`, `name`, `active` FROM `users` WHERE `active` = ? AND `name` <> ?"
	if sql != wantSQL {
		t.Fatalf("sql = %q, want %q", sql, wantSQL)
	}
	if !reflect.DeepEqual(args, []any{1, "deleted"}) {
		t.Fatalf("args = %v", args)
	}
}

func TestRenderSelectEmptyColumns(t *testing.T) {
	d := driver.NewSQLiteDialect()
	_, _, err := d.RenderSelect(federation.QueryLeg{ConnectionID: "billing"})
	if err == nil {
		t.Fatal("expected error for empty columns")
	}
}

func TestRenderSelectPassesReadOnlyValidation(t *testing.T) {
	dialects := []driver.Dialect{
		driver.NewSQLiteDialect(),
		driver.NewMySQLDialect(),
	}

	for _, d := range dialects {
		if _, _, err := d.RenderSelect(billingUsersLeg()); err != nil {
			t.Fatalf("%T RenderSelect() error = %v", d, err)
		}
		if _, _, err := d.RenderSelect(analyticsOrdersLeg()); err != nil {
			t.Fatalf("%T RenderSelect() error = %v", d, err)
		}
	}
}
