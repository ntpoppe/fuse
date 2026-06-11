package federation_test

import (
	"reflect"
	"testing"

	"github.com/ntpoppe/fuse/internal/driver"
	"github.com/ntpoppe/fuse/internal/federation"
)

const plannerRenderJoinSQL = `
SELECT u.id, u.name, o.total
FROM billing.users AS u
JOIN analytics.orders AS o ON u.id = o.user_id
WHERE u.active = 1
LIMIT 100
`

func TestPlannerRenderGoldenTwoTableJoin(t *testing.T) {
	q, err := federation.Parse(plannerRenderJoinSQL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	plan, err := federation.Plan(q)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	tests := []struct {
		name     string
		dialect  driver.Dialect
		legIndex int
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "billing sqlite",
			dialect:  driver.NewSQLiteDialect(),
			legIndex: 0,
			wantSQL:  `SELECT "id", "name", "active" FROM "users" WHERE "active" = ?`,
			wantArgs: []any{1},
		},
		{
			name:     "billing mysql",
			dialect:  driver.NewMySQLDialect(),
			legIndex: 0,
			wantSQL:  "SELECT `id`, `name`, `active` FROM `users` WHERE `active` = ?",
			wantArgs: []any{1},
		},
		{
			name:     "analytics sqlite",
			dialect:  driver.NewSQLiteDialect(),
			legIndex: 1,
			wantSQL:  `SELECT "total", "user_id" FROM "orders"`,
			wantArgs: []any{},
		},
		{
			name:     "analytics mysql",
			dialect:  driver.NewMySQLDialect(),
			legIndex: 1,
			wantSQL:  "SELECT `total`, `user_id` FROM `orders`",
			wantArgs: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.dialect.RenderSelect(federation.SelectLegForDriver(plan.Legs[tt.legIndex]))
			if err != nil {
				t.Fatalf("RenderSelect() error = %v", err)
			}
			if sql != tt.wantSQL {
				t.Fatalf("sql = %q, want %q", sql, tt.wantSQL)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Fatalf("args = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}

func TestPlannerRenderGoldenSingleTable(t *testing.T) {
	q, err := federation.Parse(`SELECT u.id, u.name FROM billing.users u WHERE u.active = 1 LIMIT 10`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	plan, err := federation.Plan(q)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(plan.Legs) != 1 {
		t.Fatalf("Legs len = %d, want 1", len(plan.Legs))
	}

	sql, args, err := driver.NewSQLiteDialect().RenderSelect(federation.SelectLegForDriver(plan.Legs[0]))
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

func TestPlannerRenderAllLegsPassReadOnlyValidation(t *testing.T) {
	q, err := federation.Parse(plannerRenderJoinSQL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	plan, err := federation.Plan(q)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	dialects := []driver.Dialect{
		driver.NewSQLiteDialect(),
		driver.NewMySQLDialect(),
	}

	for _, d := range dialects {
		for i, leg := range plan.Legs {
			if _, _, err := d.RenderSelect(federation.SelectLegForDriver(leg)); err != nil {
				t.Fatalf("%T leg %d RenderSelect() error = %v", d, i, err)
			}
		}
	}
}
