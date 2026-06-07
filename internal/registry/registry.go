package registry

import (
	"database/sql"
	"sync"
)

type Registry struct {
	mu    sync.RWMutex
	cache map[string]*sql.DB
}

func NewRegistry() *Registry {
	cache := make(map[string]*sql.DB)
	registry := Registry{cache: cache}
	return &registry
}

func (r *Registry) Fetch(key string) (*sql.DB, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	val, exists := r.cache[key]
	return val, exists
}

func (r *Registry) Save(key string, val *sql.DB) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cache[key] = val
}

func (r *Registry) Delete(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.cache, key)
}
