package executor_test

import (
	"strings"
	"testing"

	"github.com/ntpoppe/fuse/internal/config"
	"github.com/ntpoppe/fuse/internal/driver"
	"github.com/ntpoppe/fuse/internal/executor"
	"github.com/ntpoppe/fuse/internal/registry"
	"github.com/ntpoppe/fuse/internal/testutil"
)

const usersDDL = `
CREATE TABLE mock_users (
	id INTEGER PRIMARY KEY,
	username TEXT,
	is_admin BOOLEAN
);
INSERT INTO mock_users (username, is_admin) VALUES ('nate', 1), ('guest', 0);
`

func newExecutor(t *testing.T) (*executor.Executor, string) {
	t.Helper()

	path := testutil.SeedSQLiteFile(t, usersDDL)
	reg := registry.NewRegistry()
	id := "test_sqlite_pool"
	target, err := driver.OpenTarget(id, driver.DriverSQLite, path)
	if err != nil {
		t.Fatalf("open target: %v", err)
	}
	t.Cleanup(func() { _ = target.Close() })

	reg.Save(id, target)
	return executor.NewExecutor(reg, config.DefaultMaxQueryRows), id
}

func TestExecutor_ExecuteQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T) (exec *executor.Executor, id string, sql string)
		wantRows  int
		wantErr   bool
		errSubstr string
		assert    func(t *testing.T, rows []map[string]any)
	}{
		{
			name: "success",
			setup: func(t *testing.T) (*executor.Executor, string, string) {
				exec, id := newExecutor(t)
				return exec, id, "SELECT id, username, is_admin FROM mock_users ORDER BY id ASC"
			},
			wantRows: 2,
			assert: func(t *testing.T, rows []map[string]any) {
				t.Helper()
				if rows[0]["username"] != "nate" {
					t.Fatalf("username = %v, want nate", rows[0]["username"])
				}
				if rows[0]["is_admin"] == nil {
					t.Fatal("expected is_admin to be populated")
				}
			},
		},
		{
			name: "missing connection id",
			setup: func(*testing.T) (*executor.Executor, string, string) {
				return executor.NewExecutor(registry.NewRegistry(), config.DefaultMaxQueryRows), "invalid_lookup_id", "SELECT * FROM mock_users"
			},
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name: "invalid sql",
			setup: func(t *testing.T) (*executor.Executor, string, string) {
				exec, id := newExecutor(t)
				return exec, id, "SELECT * FROM missing_table"
			},
			wantErr:   true,
			errSubstr: "query:",
		},
		{
			name: "delete rejected before execution",
			setup: func(t *testing.T) (*executor.Executor, string, string) {
				exec, id := newExecutor(t)
				return exec, id, "DELETE FROM mock_users"
			},
			wantErr:   true,
			errSubstr: "read-only violation",
		},
		{
			name: "multi statement rejected before execution",
			setup: func(t *testing.T) (*executor.Executor, string, string) {
				exec, id := newExecutor(t)
				return exec, id, "SELECT 1; DELETE FROM mock_users"
			},
			wantErr:   true,
			errSubstr: "read-only violation",
		},
		{
			name: "modifying cte rejected before execution",
			setup: func(t *testing.T) (*executor.Executor, string, string) {
				exec, id := newExecutor(t)
				return exec, id, "WITH deleted AS (DELETE FROM mock_users RETURNING *) SELECT * FROM deleted"
			},
			wantErr:   true,
			errSubstr: "read-only violation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			exec, id, query := tt.setup(t)
			rows, err := exec.ExecuteQuery(testutil.Context(t), id, query)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(rows) != tt.wantRows {
				t.Fatalf("row count = %d, want %d", len(rows), tt.wantRows)
			}
			if tt.assert != nil {
				tt.assert(t, rows)
			}
		})
	}
}
