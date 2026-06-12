package api

import (
	"context"

	"github.com/ntpoppe/fuse/internal/storage"
)

// ConnectionRegistrar opens and closes live database targets.
type ConnectionRegistrar interface {
	RegisterConnection(id, driverName, host string) error
	RemoveConnection(id string) error
}

// ConnectionStore persists connection metadata.
type ConnectionStore interface {
	GetAllConnections(ctx context.Context) ([]storage.SavedConnection, error)
	SaveConnection(ctx context.Context, conn storage.SavedConnection) error
	GetConnection(ctx context.Context, id string) (storage.SavedConnection, bool, error)
	RemoveConnection(ctx context.Context, id string) error
}
