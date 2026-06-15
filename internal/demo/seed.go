package demo

import (
	"context"
	"errors"
	"fmt"

	"github.com/ntpoppe/fuse/internal/config"
	"github.com/ntpoppe/fuse/internal/driver"
	"github.com/ntpoppe/fuse/internal/fuseerr"
	"github.com/ntpoppe/fuse/internal/storage"
)

type ConnectionRegistrar interface {
	RegisterConnection(id, driverName, host string) error
	RemoveConnection(id string) error
}

type ConnectionStore interface {
	GetAllConnections(ctx context.Context) ([]storage.SavedConnection, error)
	RemoveAllConnections(ctx context.Context) error
	SaveConnection(ctx context.Context, conn storage.SavedConnection) error
}

func SeedConnections(ctx context.Context, cr ConnectionRegistrar, store ConnectionStore, cfg *config.Config) error {
	existing, err := store.GetAllConnections(ctx)
	if err != nil {
		return fmt.Errorf("list existing connections: %w", err)
	}

	for _, conn := range existing {
		if err := cr.RemoveConnection(conn.ID); err != nil && !errors.Is(err, fuseerr.ErrNotFound) {
			return fmt.Errorf("remove connection %q: %w", conn.ID, err)
		}
	}

	if err := store.RemoveAllConnections(ctx); err != nil {
		return fmt.Errorf("clear saved connections: %w", err)
	}

	connections := []storage.SavedConnection{
		{ID: "shop", Driver: driver.DriverSQLite, Host: cfg.DemoSQLitePath},
		{ID: "warehouse", Driver: driver.DriverMySQL, Host: cfg.DemoMySQLDSN},
	}

	for _, conn := range connections {
		if err := cr.RegisterConnection(conn.ID, conn.Driver, conn.Host); err != nil {
			return fmt.Errorf("register demo connection %q: %w", conn.ID, err)
		}

		if err := store.SaveConnection(ctx, conn); err != nil {
			_ = cr.RemoveConnection(conn.ID)
			return fmt.Errorf("save demo connection %q: %w", conn.ID, err)
		}
	}

	return nil
}
