package connectionmanager

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"

	"github.com/ntpoppe/fuse/internal/registry"
)

type ConnectionManager struct {
	reg *registry.Registry
}

func NewConnectionManager(reg *registry.Registry) *ConnectionManager {
	return &ConnectionManager{reg}
}

func (cm *ConnectionManager) RegisterConnection(id string, driver string, host string) error {
	if _, exists := cm.reg.Fetch(id); exists {
		return fmt.Errorf("id %q already exists in registry, remove before re-assigning", id)
	}

	cleanedHost := NormalizeHost(driver, host)
	db, openErr := sql.Open(driver, cleanedHost)
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

func (cm *ConnectionManager) RemoveConnection(id string) error {
	db, exists := cm.reg.Fetch(id)
	if !exists {
		return fmt.Errorf("connection pool for %q does not exist", id)
	}

	db.Close()
	cm.reg.Delete(id)
	return nil
}

func NormalizeHost(driver string, host string) string {
	switch driver {
	case "sqlite":
		cleanedPrefix := strings.TrimPrefix(host, "file:")
		cleanedHost := strings.TrimSuffix(cleanedPrefix, "?mode=ro")
		return fmt.Sprintf("file:%s?mode=ro", cleanedHost)
	}

	return host
}
