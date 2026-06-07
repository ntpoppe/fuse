package storage_test

import (
	"testing"

	"github.com/ntpoppe/fuse/internal/storage"
	"github.com/ntpoppe/fuse/internal/testutil"
)

func newStore(t *testing.T) (*storage.Store, *storage.SavedConnection) {
	t.Helper()

	store := storage.NewStore(testutil.OpenSQLiteMemory(t))
	if err := store.InitializeSchema(); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}

	sample := &storage.SavedConnection{
		ID:     "prod_mysql",
		Driver: "mysql",
		Host:   "root:secret@tcp(127.0.0.1:3306)/production",
	}
	return store, sample
}

func saveConnection(t *testing.T, store *storage.Store, conn storage.SavedConnection) {
	t.Helper()
	if err := store.SaveConnection(testutil.Context(t), conn); err != nil {
		t.Fatalf("save connection %q: %v", conn.ID, err)
	}
}

func getConnections(t *testing.T, store *storage.Store) []storage.SavedConnection {
	t.Helper()

	connections, err := store.GetAllConnections(testutil.Context(t))
	if err != nil {
		t.Fatalf("get connections: %v", err)
	}
	return connections
}

func TestStore_InitializeSchema(t *testing.T) {
	store, _ := newStore(t)

	if err := store.InitializeSchema(); err != nil {
		t.Fatalf("re-run schema init: %v", err)
	}
}

func TestStore_SaveConnection(t *testing.T) {
	store, sample := newStore(t)
	saveConnection(t, store, *sample)

	connections := getConnections(t, store)
	if len(connections) != 1 {
		t.Fatalf("connection count = %d, want 1", len(connections))
	}
	if connections[0] != *sample {
		t.Fatalf("saved connection = %+v, want %+v", connections[0], *sample)
	}
}

func TestStore_GetAllConnections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, store *storage.Store)
		want  int
	}{
		{
			name:  "empty store",
			setup: func(*testing.T, *storage.Store) {},
			want:  0,
		},
		{
			name: "multiple connections",
			setup: func(t *testing.T, store *storage.Store) {
				saveConnection(t, store, storage.SavedConnection{
					ID: "one", Driver: "mysql", Host: "host1",
				})
				saveConnection(t, store, storage.SavedConnection{
					ID: "two", Driver: "sqlite", Host: "host2",
				})
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, _ := newStore(t)
			tt.setup(t, store)

			if got := len(getConnections(t, store)); got != tt.want {
				t.Fatalf("connection count = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestStore_SaveConnection_Upsert(t *testing.T) {
	store, sample := newStore(t)
	saveConnection(t, store, *sample)

	updated := *sample
	updated.Host = "new_connection_string"
	saveConnection(t, store, updated)

	connections := getConnections(t, store)
	if len(connections) != 1 {
		t.Fatalf("connection count = %d, want 1", len(connections))
	}
	if connections[0].Host != updated.Host {
		t.Fatalf("host = %q, want %q", connections[0].Host, updated.Host)
	}
}
