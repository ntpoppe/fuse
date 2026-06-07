package testutil

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"testing"
)

type mockDriver struct {
	failPing bool
}

type mockConn struct {
	failPing bool
}

func (d *mockDriver) Open(string) (driver.Conn, error) {
	return &mockConn{failPing: d.failPing}, nil
}

func (c *mockConn) Ping(context.Context) error {
	if c.failPing {
		return errors.New("network destination unreachable")
	}
	return nil
}

func (c *mockConn) Prepare(string) (driver.Stmt, error) { return nil, nil }
func (c *mockConn) Close() error                        { return nil }
func (c *mockConn) Begin() (driver.Tx, error)           { return nil, nil }

func RegisterNamedMockDriver(t *testing.T, name string, failPing bool) string {
	t.Helper()
	driverName := fmt.Sprintf("%s_%s", name, t.Name())
	sql.Register(driverName, &mockDriver{failPing: failPing})
	return driverName
}
