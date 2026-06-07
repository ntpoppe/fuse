package connectionmanager_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	connectionmanager "github.com/ntpoppe/fuse/internal/connection_manager"
	"github.com/ntpoppe/fuse/internal/registry"
)

type mockDriver struct{ failPing bool }
type mockConn struct{ failPing bool }

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{failPing: d.failPing}, nil
}

// Implement Pinger interface to control db.PingContext behavior
func (c *mockConn) Ping(ctx context.Context) error {
	if c.failPing {
		return errors.New("network destination unreachable")
	}
	return nil
}

// Minimal interface fulfillments required by database/sql
func (c *mockConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (c *mockConn) Close() error                              { return nil }
func (c *mockConn) Begin() (driver.Tx, error)                 { return nil, nil }

func TestRegisterNewConnection_Success(t *testing.T) {
	// Register a unique "happy" driver name for this test execution
	sql.Register("mock_healthy", &mockDriver{failPing: false})

	reg := registry.NewRegistry()
	cm := connectionmanager.NewConnectionManager(reg)

	// Run registration pipeline
	err := cm.RegisterConnection("db_1", "mock_healthy", "localhost:3306")
	if err != nil {
		t.Fatalf("expected successful registration, got error: %v", err)
	}

	// Confirm pool was successfully stored in the registry
	_, found := reg.Fetch("db_1")
	if !found {
		t.Error("expected connection pool to be saved in the registry, but it was missing")
	}
}

func TestRegisterNewConnection_PingFailure(t *testing.T) {
	// Register a unique "broken" driver that explicitly fails pings
	sql.Register("mock_broken", &mockDriver{failPing: true})

	reg := registry.NewRegistry()
	cm := connectionmanager.NewConnectionManager(reg)

	// Verify that a failing ping bubbles up an error cleanly
	err := cm.RegisterConnection("db_2", "mock_broken", "localhost:1433")
	if err == nil {
		t.Error("expected error due to network ping failure, got nil")
	}

	// Confirm nothing was cached in the registry
	_, found := reg.Fetch("db_2")
	if found {
		t.Error("expected registry to be empty after a failed connection attempt, but found data")
	}
}

func TestRegisterNewConnection_InvalidDriver(t *testing.T) {
	reg := registry.NewRegistry()
	cm := connectionmanager.NewConnectionManager(reg)

	// Verify sql.Open behavior when an unregistered string is provided
	err := cm.RegisterConnection("db_3", "non_existent_driver", "localhost:9999")
	if err == nil {
		t.Error("expected error for unregistered driver string, got nil")
	}
}

func TestRemoveConnection_Success(t *testing.T) {
	sql.Register("mock_removable", &mockDriver{failPing: false})

	reg := registry.NewRegistry()
	cm := connectionmanager.NewConnectionManager(reg)

	if err := cm.RegisterConnection("db_4", "mock_removable", "localhost:3306"); err != nil {
		t.Fatalf("expected successful registration, got error: %v", err)
	}

	if err := cm.RemoveConnection("db_4"); err != nil {
		t.Fatalf("expected successful removal, got error: %v", err)
	}

	_, found := reg.Fetch("db_4")
	if found {
		t.Error("expected connection to be removed from the registry, but it was still present")
	}
}

func TestRemoveConnection_NotFound(t *testing.T) {
	reg := registry.NewRegistry()
	cm := connectionmanager.NewConnectionManager(reg)

	err := cm.RemoveConnection("missing_connection")
	if err == nil {
		t.Fatal("expected error when removing a missing connection, got nil")
	}
}
