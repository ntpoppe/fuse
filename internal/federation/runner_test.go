package federation_test

import (
	"errors"
	"testing"

	"github.com/ntpoppe/fuse/internal/config"
	"github.com/ntpoppe/fuse/internal/driver"
	"github.com/ntpoppe/fuse/internal/federation"
	"github.com/ntpoppe/fuse/internal/fuseerr"
	"github.com/ntpoppe/fuse/internal/registry"
	"github.com/ntpoppe/fuse/internal/testutil"
)

func TestRunnerTwoDatabaseJoin(t *testing.T) {
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

	reg := registry.NewRegistry()
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

	q, err := federation.Parse(plannerRenderJoinSQL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if err := federation.ResolveConnections(q, reg); err != nil {
		t.Fatalf("ResolveConnections() error = %v", err)
	}

	plan, err := federation.Plan(q)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	runner := federation.NewRunner(reg, config.DefaultMaxQueryRows)
	rows, err := runner.Run(testutil.Context(t), plan)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
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

func TestRunnerMissingConnection(t *testing.T) {
	q, err := federation.Parse(plannerRenderJoinSQL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	plan, err := federation.Plan(q)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	runner := federation.NewRunner(registry.NewRegistry(), config.DefaultMaxQueryRows)
	_, err = runner.Run(testutil.Context(t), plan)
	var notFound fuseerr.NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("error = %v, want NotFoundError", err)
	}
}

func TestRunnerRowLimitError(t *testing.T) {
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

	reg := registry.NewRegistry()
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

	q, err := federation.Parse(plannerRenderJoinSQL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	plan, err := federation.Plan(q)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	runner := federation.NewRunner(reg, 1)
	_, err = runner.Run(testutil.Context(t), plan)
	if !errors.Is(err, fuseerr.ErrQueryRowLimit) {
		t.Fatalf("error = %v, want ErrQueryRowLimit", err)
	}
}

func TestRegistryImplementsTargetLookup(t *testing.T) {
	var _ federation.TargetLookup = (*registry.Registry)(nil)
}
