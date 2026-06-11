package driver_test

import (
	"reflect"
	"testing"

	"github.com/ntpoppe/fuse/internal/driver"
)

func billingUsersLeg() driver.SelectLeg {
	return driver.SelectLeg{
		Table:   "users",
		Columns: []string{"id", "name", "active"},
		Where: []driver.SelectPredicate{
			{Column: "active", Op: "=", Value: 1},
		},
	}
}

func analyticsOrdersLeg() driver.SelectLeg {
	return driver.SelectLeg{
		Table:   "orders",
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
	leg.Where = append(leg.Where, driver.SelectPredicate{
		Column: "name",
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
	_, _, err := d.RenderSelect(driver.SelectLeg{Table: "users"})
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
