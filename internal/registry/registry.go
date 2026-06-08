package registry

import (
	"sync"

	"github.com/ntpoppe/fuse/internal/driver"
)

type Registry struct {
	mu    sync.RWMutex
	cache map[string]driver.Target
}

func NewRegistry() *Registry {
	return &Registry{
		cache: make(map[string]driver.Target),
	}
}

func (r *Registry) Fetch(key string) (driver.Target, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	val, exists := r.cache[key]
	return val, exists
}

func (r *Registry) HasConnection(id string) bool {
	_, ok := r.Fetch(id)
	return ok
}

func (r *Registry) Save(key string, val driver.Target) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cache[key] = val
}

func (r *Registry) Delete(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.cache, key)
}
