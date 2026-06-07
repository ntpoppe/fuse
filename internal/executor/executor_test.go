package executor_test

import (
	"strings"
	"testing"

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

	db := testutil.OpenSQLiteMemory(t)
	if _, err := db.Exec(usersDDL); err != nil {
		t.Fatalf("seed database: %v", err)
	}

	reg := registry.NewRegistry()
	id := "test_sqlite_pool"
	reg.Save(id, db)

	return executor.NewExecutor(reg), id
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
				return executor.NewExecutor(registry.NewRegistry()), "invalid_lookup_id", "SELECT * FROM mock_users"
			},
			wantErr:   true,
			errSubstr: "does not exist in registry",
		},
		{
			name: "invalid sql",
			setup: func(t *testing.T) (*executor.Executor, string, string) {
				exec, id := newExecutor(t)
				return exec, id, "SELECT * FROM missing_table"
			},
			wantErr:   true,
			errSubstr: "error querying",
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
