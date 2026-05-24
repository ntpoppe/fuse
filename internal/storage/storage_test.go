package storage_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/ntpoppe/fuse/internal/storage"
	_ "modernc.org/sqlite"
)

func TestStore_Lifecycle(t *testing.T) {
	// Setup a completely fresh, isolated in-memory SQLite database for the test
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test sqlite db: %v", err)
	}
	defer db.Close()

	// Instantiate the Repository Store using our test database pointer
	store := storage.NewStore(db)

	// Test Schema Initialization (Migration Pass)
	if err := store.InitializeSchema(); err != nil {
		t.Fatalf("failed to initialize schema: %v", err)
	}

	// Create defensive context rules for our data modifications
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test Insertion Pass (SaveConnection)
	mockConn1 := storage.SavedConnection{
		ID:     "prod_mysql",
		Driver: "mysql",
		Host:   "root:secret@tcp(127.0.0.1:3306)/production",
	}
	mockConn2 := storage.SavedConnection{
		ID:     "analytics_postgres",
		Driver: "postgres",
		Host:   "postgres://user:pass@localhost:5432/analytics",
	}

	if err := store.SaveConnection(ctx, mockConn1); err != nil {
		t.Errorf("failed to save mockConn1: %v", err)
	}
	if err := store.SaveConnection(ctx, mockConn2); err != nil {
		t.Errorf("failed to save mockConn2: %v", err)
	}

	// Test Query Pass (GetAllConnections)
	connections, err := store.GetAllConnections(ctx)
	if err != nil {
		t.Fatalf("failed to fetch connections from disk: %v", err)
	}

	// Assertions: Validate that we got exactly 2 records back
	if len(connections) != 2 {
		t.Fatalf("expected 2 saved connections, got %d", len(connections))
	}

	// Assertions: Verify the records retain all original properties
	foundMySQL := false
	for _, c := range connections {
		if c.ID == "prod_mysql" {
			foundMySQL = true
			if c.Driver != "mysql" || c.Host != mockConn1.Host {
				t.Errorf("data mismatch for prod_mysql, got %+v", c)
			}
		}
	}

	if !foundMySQL {
		t.Error("expected to find 'prod_mysql' in the extracted records, but it was missing")
	}

	// Test Item Overwrite Protection (INSERT OR REPLACE behavior)
	updatedMySQL := storage.SavedConnection{
		ID:     "prod_mysql",
		Driver: "mysql",
		Host:   "new_connection_string",
	}
	if err := store.SaveConnection(ctx, updatedMySQL); err != nil {
		t.Fatalf("failed to update record: %v", err)
	}

	// Fetch again to verify update
	connectionsAfterUpdate, _ := store.GetAllConnections(ctx)
	if len(connectionsAfterUpdate) != 2 {
		t.Errorf("row count shouldn't increase on duplicate keys, expected 2 rows but got %d", len(connectionsAfterUpdate))
	}
}
