package connectionmanager_test

import (
	"strings"
	"testing"

	connectionmanager "github.com/ntpoppe/fuse/internal/connection_manager"
	"github.com/ntpoppe/fuse/internal/registry"
	"github.com/ntpoppe/fuse/internal/testutil"
)

type env struct {
	reg *registry.Registry
	cm  *connectionmanager.ConnectionManager
}

func newEnv(t *testing.T) env {
	t.Helper()
	reg := registry.NewRegistry()
	return env{
		reg: reg,
		cm:  connectionmanager.NewConnectionManager(reg),
	}
}

func TestConnectionManager_RegisterConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		driver    func(t *testing.T) string
		id        string
		host      string
		wantErr   bool
		errSubstr string
		wantSaved bool
	}{
		{
			name: "success",
			driver: func(t *testing.T) string {
				return testutil.RegisterNamedMockDriver(t, "healthy", false)
			},
			id:        "db_1",
			host:      "localhost:3306",
			wantSaved: true,
		},
		{
			name: "ping failure",
			driver: func(t *testing.T) string {
				return testutil.RegisterNamedMockDriver(t, "broken", true)
			},
			id:        "db_2",
			host:      "localhost:1433",
			wantErr:   true,
			errSubstr: "failed to ping",
		},
		{
			name:      "invalid driver",
			driver:    func(*testing.T) string { return "non_existent_driver" },
			id:        "db_3",
			host:      "localhost:9999",
			wantErr:   true,
			errSubstr: "failed to open db conn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			driver := tt.driver(t)

			err := e.cm.RegisterConnection(tt.id, driver, tt.host)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.errSubstr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			_, found := e.reg.Fetch(tt.id)
			if found != tt.wantSaved {
				t.Fatalf("saved = %v, want %v", found, tt.wantSaved)
			}
		})
	}
}

func TestConnectionManager_RegisterConnection_DuplicateID(t *testing.T) {
	e := newEnv(t)
	driver := testutil.RegisterNamedMockDriver(t, "duplicate", false)

	if err := e.cm.RegisterConnection("db_1", driver, "localhost:3306"); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	err := e.cm.RegisterConnection("db_1", driver, "localhost:3306")
	if err == nil {
		t.Fatal("expected duplicate registration error, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %q, want duplicate id message", err.Error())
	}
}

func TestConnectionManager_RemoveConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T, e env) string
		wantErr bool
	}{
		{
			name: "success",
			setup: func(t *testing.T, e env) string {
				t.Helper()
				driver := testutil.RegisterNamedMockDriver(t, "removable", false)
				id := "db_4"
				if err := e.cm.RegisterConnection(id, driver, "localhost:3306"); err != nil {
					t.Fatalf("register connection: %v", err)
				}
				return id
			},
		},
		{
			name: "not found",
			setup: func(*testing.T, env) string {
				return "missing_connection"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			id := tt.setup(t, e)

			err := e.cm.RemoveConnection(id)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.wantErr {
				if _, found := e.reg.Fetch(id); found {
					t.Fatal("expected connection to be removed from registry")
				}
			}
		})
	}
}

func TestNormalizeHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		driver string
		host   string
		want   string
	}{
		{name: "sqlite plain path", driver: "sqlite", host: "dev_target.db", want: "file:dev_target.db?mode=ro"},
		{name: "sqlite file prefix", driver: "sqlite", host: "file:dev_target.db", want: "file:dev_target.db?mode=ro"},
		{name: "sqlite already read only", driver: "sqlite", host: "file:dev_target.db?mode=ro", want: "file:dev_target.db?mode=ro"},
		{name: "sqlite plain path with existing mode suffix", driver: "sqlite", host: "dev_target.db?mode=ro", want: "file:dev_target.db?mode=ro"},
		{name: "mysql passthrough", driver: "mysql", host: "user:pass@tcp(localhost:3306)/mydb", want: "user:pass@tcp(localhost:3306)/mydb"},
		{name: "sql server passthrough", driver: "sqlserver", host: "sqlserver://user:pass@localhost:1433?database=mydb", want: "sqlserver://user:pass@localhost:1433?database=mydb"},
		{name: "unknown driver passthrough", driver: "postgres", host: "postgres://localhost/mydb", want: "postgres://localhost/mydb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := connectionmanager.NormalizeHost(tt.driver, tt.host)
			if got != tt.want {
				t.Fatalf("NormalizeHost(%q, %q) = %q, want %q", tt.driver, tt.host, got, tt.want)
			}
		})
	}
}
