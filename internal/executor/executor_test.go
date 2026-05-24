package executor_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/ntpoppe/fuse/internal/executor"
	"github.com/ntpoppe/fuse/internal/registry"
	_ "modernc.org/sqlite"
)

func TestExecutor_ExecuteQuery_Success(t *testing.T) {
	// Establish a real in-memory SQLite target database for the evaluation sandbox
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite db: %v", err)
	}
	defer db.Close()

	// Hydrate seed schema and structural records into the temporary sandbox database
	_, err = db.Exec(`
		CREATE TABLE mock_users (
			id INTEGER PRIMARY KEY,
			username TEXT,
			is_admin BOOLEAN
		);
		INSERT INTO mock_users (username, is_admin) VALUES ('nate', 1), ('guest', 0);
	`)
	if err != nil {
		t.Fatalf("failed to seed mock database: %v", err)
	}

	// Store the open connection pool in your thread-safe Registry map matrix
	reg := registry.NewRegistry()
	targetID := "test_sqlite_pool"
	reg.Save(targetID, db)

	// Instantiate the execution unit injection engine
	exec := executor.NewExecutor(reg)

	// Build a short defensive timeout context lifecycle wrapper
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Execute an arbitrary query statement containing mixed types
	results, err := exec.ExecuteQuery(ctx, targetID, "SELECT id, username, is_admin FROM mock_users ORDER BY id ASC")
	if err != nil {
		t.Fatalf("ExecuteQuery returned unexpected error: %v", err)
	}

	// Validate matrix depth width matching expectations
	if len(results) != 2 {
		t.Errorf("expected 2 result records, got %v", len(results) != 2)
	}

	// Assert Type Unpacking and String Conversion worked flawlessly on the first record
	firstRecord := results[0]

	if firstRecord["username"] != "nate" {
		t.Errorf("expected 'username' key to be 'nate', got %v (Type: %T)", firstRecord["username"], firstRecord["username"])
	}

	// SQLite returns boolean representations as integers (1 or 0) or values depending on driver serialization
	if firstRecord["is_admin"] == nil {
		t.Error("expected 'is_admin' field key to be present and populated, got nil")
	}
}

func TestExecutor_ExecuteQuery_MissingID(t *testing.T) {
	// Setup empty structural registers
	reg := registry.NewRegistry()
	exec := executor.NewExecutor(reg)

	ctx := context.Background()

	// Fire command targeting a connection ID that has never been registered
	_, err := exec.ExecuteQuery(ctx, "invalid_lookup_id", "SELECT * FROM items")
	if err == nil {
		t.Error("expected an error when executing against an unregistered target connection ID, got nil")
	}
}
