package executor_test

import (
	"errors"
	"testing"

	"github.com/ntpoppe/fuse/internal/config"
	"github.com/ntpoppe/fuse/internal/driver"
	"github.com/ntpoppe/fuse/internal/executor"
	"github.com/ntpoppe/fuse/internal/fuseerr"
	"github.com/ntpoppe/fuse/internal/registry"
	"github.com/ntpoppe/fuse/internal/testutil"
)

const validFederatedSQL = `
SELECT u.id, u.name, o.total
FROM billing.users u
JOIN analytics.orders o ON u.id = o.user_id
WHERE u.active = 1
LIMIT 100
`

func registerFederatedSQLiteConnections(t *testing.T, reg *registry.Registry) {
	t.Helper()

	billingPath := testutil.SeedSQLiteFile(t, `
CREATE TABLE users (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	active INTEGER NOT NULL
);
INSERT INTO users (id, name, active) VALUES (1, 'alice', 1), (2, 'bob', 1);
`)
	analyticsPath := testutil.SeedSQLiteFile(t, `
CREATE TABLE orders (
	user_id INTEGER NOT NULL,
	total REAL NOT NULL
);
INSERT INTO orders (user_id, total) VALUES (1, 10.5), (2, 20.0);
`)

	for _, tc := range []struct {
		id   string
		path string
	}{
		{"billing", billingPath},
		{"analytics", analyticsPath},
	} {
		target, err := driver.OpenTarget(tc.id, driver.DriverSQLite, tc.path)
		if err != nil {
			t.Fatalf("open target %q: %v", tc.id, err)
		}
		t.Cleanup(func() { _ = target.Close() })
		reg.Save(tc.id, target)
	}
}

func TestFederatedExecutorTwoDatabaseJoin(t *testing.T) {
	reg := registry.NewRegistry()
	registerFederatedSQLiteConnections(t, reg)

	fed := executor.NewFederatedExecutor(reg, config.DefaultMaxQueryRows)
	rows, err := fed.ExecuteFederatedQuery(testutil.Context(t), validFederatedSQL)
	if err != nil {
		t.Fatalf("ExecuteFederatedQuery() error = %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("rows len = %d, want 2", len(rows))
	}
	if rows[0]["name"] != "alice" || rows[0]["total"] != 10.5 {
		t.Fatalf("row[0] = %+v", rows[0])
	}
	if rows[1]["name"] != "bob" || rows[1]["total"] != 20.0 {
		t.Fatalf("row[1] = %+v", rows[1])
	}
}

func TestFederatedExecutorInvalidSQL(t *testing.T) {
	fed := executor.NewFederatedExecutor(registry.NewRegistry(), config.DefaultMaxQueryRows)
	_, err := fed.ExecuteFederatedQuery(testutil.Context(t), `SELECT u.id FROM users u`)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFederatedExecutorUnknownConnection(t *testing.T) {
	reg := registry.NewRegistry()
	reg.Save("billing", testutil.NewStubTarget("billing"))

	fed := executor.NewFederatedExecutor(reg, config.DefaultMaxQueryRows)
	_, err := fed.ExecuteFederatedQuery(testutil.Context(t), validFederatedSQL)
	var notFound fuseerr.NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("error = %v, want NotFoundError", err)
	}
	if notFound.ID != "analytics" {
		t.Fatalf("NotFoundError.ID = %q, want analytics", notFound.ID)
	}
}
