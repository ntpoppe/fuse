package connectionmanager

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ntpoppe/fuse/internal/registry"
)

type ConnectionManager struct {
	reg *registry.Registry
}

func NewConnectionManager(reg *registry.Registry) *ConnectionManager {
	return &ConnectionManager{reg}
}

func (cm *ConnectionManager) RegisterNewConnection(id string, driver string, host string) error {
	db, openErr := sql.Open(driver, host)
	if openErr != nil {
		return fmt.Errorf("failed to open db conn for driver %q and host %q", driver, host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pingErr := db.PingContext(ctx)
	if pingErr != nil {
		return fmt.Errorf("failed to ping db conn for driver %q and host %q", driver, host)
	}

	cm.reg.Save(id, db)
	return nil
}
