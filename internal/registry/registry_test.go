package registry_test

import (
	"strconv"
	"sync"
	"testing"

	"github.com/ntpoppe/fuse/internal/registry"
	"github.com/ntpoppe/fuse/internal/testutil"
)

func newRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	return registry.NewRegistry()
}

func TestRegistry_Fetch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       string
		setup     func(*registry.Registry, string)
		wantFound bool
	}{
		{
			name:      "missing key",
			key:       "missing",
			wantFound: false,
		},
		{
			name: "existing key",
			key:  "mysql_production",
			setup: func(reg *registry.Registry, key string) {
				reg.Save(key, testutil.NewStubTarget(key))
			},
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reg := newRegistry(t)
			if tt.setup != nil {
				tt.setup(reg, tt.key)
			}

			_, found := reg.Fetch(tt.key)
			if found != tt.wantFound {
				t.Fatalf("Fetch(%q) found = %v, want %v", tt.key, found, tt.wantFound)
			}
		})
	}
}

func TestRegistry_SaveAndFetchPointer(t *testing.T) {
	reg := newRegistry(t)
	target := testutil.NewStubTarget("mysql_production")
	key := "mysql_production"

	reg.Save(key, target)

	got, found := reg.Fetch(key)
	if !found {
		t.Fatal("expected saved key to be found")
	}
	if got != target {
		t.Fatal("fetched target does not match saved target")
	}
}

func TestRegistry_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(*registry.Registry) string
		wantFound bool
	}{
		{
			name: "delete existing key",
			setup: func(reg *registry.Registry) string {
				key := "postgres_staging"
				reg.Save(key, testutil.NewStubTarget(key))
				reg.Delete(key)
				return key
			},
			wantFound: false,
		},
		{
			name: "delete missing key is no-op",
			setup: func(reg *registry.Registry) string {
				key := "missing"
				reg.Delete(key)
				return key
			},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reg := newRegistry(t)
			key := tt.setup(reg)

			_, found := reg.Fetch(key)
			if found != tt.wantFound {
				t.Fatalf("Fetch(%q) found = %v, want %v", key, found, tt.wantFound)
			}
		})
	}
}

func TestRegistry_Concurrency(t *testing.T) {
	reg := newRegistry(t)
	target := testutil.NewStubTarget("worker")

	const workers = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(workers * 2)

	for i := range workers {
		workerID := strconv.Itoa(i)

		go func() {
			defer wg.Done()
			for range iterations {
				reg.Save(workerID, target)
			}
		}()

		go func() {
			defer wg.Done()
			for range iterations {
				_, _ = reg.Fetch(workerID)
			}
		}()
	}

	wg.Wait()

	for i := range workers {
		key := strconv.Itoa(i)
		if _, found := reg.Fetch(key); !found {
			t.Fatalf("expected key %q to exist after concurrent writes", key)
		}
	}
}
