package demo_test

import (
	"testing"

	"github.com/ntpoppe/fuse/internal/config"
	"github.com/ntpoppe/fuse/internal/demo"
	"github.com/ntpoppe/fuse/internal/driver"
	"github.com/ntpoppe/fuse/internal/storage"
	"github.com/ntpoppe/fuse/internal/testutil"
)

type fakeRegistrar struct {
	conns map[string]storage.SavedConnection
}

func newFakeRegistrar() *fakeRegistrar {
	return &fakeRegistrar{conns: make(map[string]storage.SavedConnection)}
}

func (f *fakeRegistrar) RegisterConnection(id, driverName, host string) error {
	f.conns[id] = storage.SavedConnection{ID: id, Driver: driverName, Host: host}
	return nil
}

func (f *fakeRegistrar) RemoveConnection(id string) error {
	delete(f.conns, id)
	return nil
}

func TestSeedConnections(t *testing.T) {
	t.Parallel()

	store := storage.NewStore(testutil.OpenSQLiteMemory(t))
	if err := store.InitializeSchema(); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}

	ctx := testutil.Context(t)
	if err := store.SaveConnection(ctx, storage.SavedConnection{
		ID: "old", Driver: driver.DriverSQLite, Host: "/tmp/old.db",
	}); err != nil {
		t.Fatalf("save old connection: %v", err)
	}

	reg := newFakeRegistrar()
	cfg := config.NewConfig()
	cfg.DemoSQLitePath = "/data/shop.db"
	cfg.DemoMySQLDSN = "demo:demo@tcp(mysql:3306)/fuse_test"

	if err := demo.SeedConnections(ctx, reg, store, cfg); err != nil {
		t.Fatalf("seed connections: %v", err)
	}

	saved, err := store.GetAllConnections(ctx)
	if err != nil {
		t.Fatalf("list connections: %v", err)
	}
	if len(saved) != 2 {
		t.Fatalf("saved count = %d, want 2", len(saved))
	}

	want := map[string]storage.SavedConnection{
		"shop":      {ID: "shop", Driver: driver.DriverSQLite, Host: cfg.DemoSQLitePath},
		"warehouse": {ID: "warehouse", Driver: driver.DriverMySQL, Host: cfg.DemoMySQLDSN},
	}
	for _, conn := range saved {
		expected, ok := want[conn.ID]
		if !ok {
			t.Fatalf("unexpected connection id %q", conn.ID)
		}
		if conn != expected {
			t.Fatalf("connection %q = %+v, want %+v", conn.ID, conn, expected)
		}
	}

	if len(reg.conns) != 2 {
		t.Fatalf("registered count = %d, want 2", len(reg.conns))
	}
}
