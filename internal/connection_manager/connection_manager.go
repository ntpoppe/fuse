package connectionmanager

import (
	"fmt"
	"sync"

	"github.com/ntpoppe/fuse/internal/driver"
	"github.com/ntpoppe/fuse/internal/fuseerr"
	"github.com/ntpoppe/fuse/internal/registry"
)

type ConnectionManager struct {
	mu  sync.Mutex
	reg *registry.Registry
}

func NewConnectionManager(reg *registry.Registry) *ConnectionManager {
	return &ConnectionManager{reg: reg}
}

func (cm *ConnectionManager) RegisterConnection(id, driverName, host string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.reg.Fetch(id); exists {
		return fuseerr.AlreadyExistsError{ID: id}
	}

	target, err := driver.OpenTarget(id, driverName, host)
	if err != nil {
		return err
	}

	cm.reg.Save(id, target)
	return nil
}

func (cm *ConnectionManager) RemoveConnection(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	target, exists := cm.reg.Fetch(id)
	if !exists {
		return fuseerr.NotFoundError{ID: id}
	}

	if err := target.Close(); err != nil {
		return fmt.Errorf("close connection %q: %w", id, err)
	}

	cm.reg.Delete(id)
	return nil
}
