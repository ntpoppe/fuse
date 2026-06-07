package registry_test

import (
	"database/sql"
	"sync"
	"testing"

	"github.com/ntpoppe/fuse/internal/registry"
)

func TestRegistry_SaveAndFetch(t *testing.T) {
	reg := registry.NewRegistry()
	mockDB := &sql.DB{} // Placeholder connection handle for testing
	key := "mysql_production"

	// Test case 1: Key does not exist initially
	_, found := reg.Fetch(key)
	if found {
		t.Errorf("expected key %q to be missing, but it was found", key)
	}

	// Test case 2: Save key and fetch it back
	reg.Save(key, mockDB)

	fetchedDB, found := reg.Fetch(key)
	if !found {
		t.Fatalf("expected key %q to be found after saving, but it was missing", key)
	}

	if fetchedDB != mockDB {
		t.Error("fetched connection pool pointer did not match the saved pointer instance")
	}
}

func TestRegistry_FetchMissingKey(t *testing.T) {
	reg := registry.NewRegistry()

	val, found := reg.Fetch("non_existent_connection")
	if found {
		t.Errorf("expected found to be false for a missing key, got true")
	}
	if val != nil {
		t.Errorf("expected returned pool to be nil for a missing key, got %v", val)
	}
}

func TestRegistry_Delete(t *testing.T) {
	reg := registry.NewRegistry()
	mockDB := &sql.DB{}
	key := "postgres_staging"

	reg.Save(key, mockDB)

	reg.Delete(key)

	_, found := reg.Fetch(key)
	if found {
		t.Errorf("expected key %q to be missing after delete, but it was found", key)
	}
}

func TestRegistry_DeleteMissingKey(t *testing.T) {
	reg := registry.NewRegistry()

	reg.Delete("non_existent_connection")

	_, found := reg.Fetch("non_existent_connection")
	if found {
		t.Error("expected delete of missing key to be a no-op, but key was found")
	}
}

func TestRegistry_ConcurrencyRaceCondition(t *testing.T) {
	reg := registry.NewRegistry()
	mockDB := &sql.DB{}

	var wg sync.WaitGroup
	workers := 50
	iterations := 100

	// Kick off concurrent writers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Each worker writes cleanly to its own specific database slot
				reg.Save(string(rune(workerID)), mockDB)
			}
		}(i)
	}

	// Kick off concurrent readers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Simultaneous reading paths hammering the RWMutex
				_, _ = reg.Fetch(string(rune(workerID)))
			}
		}(i)
	}

	// Block until all concurrent goroutines finish execution
	wg.Wait()
}
