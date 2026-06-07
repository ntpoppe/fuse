package testutil

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func OpenSQLiteMemory(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func SeedSQLiteFile(t *testing.T, ddl string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "target.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open seed sqlite file: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("seed sqlite file: %v", err)
	}
	return path
}
