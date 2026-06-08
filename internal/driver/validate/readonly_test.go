package validate_test

import (
	"strings"
	"testing"

	"github.com/ntpoppe/fuse/internal/driver"
)

func TestSQLiteValidateReadOnly(t *testing.T) {
	t.Parallel()

	dialect := driver.NewSQLiteDialect()

	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{name: "select allowed", sql: "SELECT 1", wantErr: false},
		{name: "with select allowed", sql: "WITH x AS (SELECT 1 AS n) SELECT n FROM x", wantErr: false},
		{name: "explain allowed", sql: "EXPLAIN SELECT 1", wantErr: false},
		{name: "delete rejected", sql: "DELETE FROM users", wantErr: true},
		{name: "multi statement rejected", sql: "SELECT 1; DELETE FROM users", wantErr: true},
		{name: "modifying cte rejected", sql: "WITH d AS (DELETE FROM users RETURNING *) SELECT * FROM d", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := dialect.ValidateReadOnly(tt.sql)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestMySQLValidateReadOnly(t *testing.T) {
	t.Parallel()

	dialect := driver.NewMySQLDialect()

	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{name: "select allowed", sql: "SELECT 1", wantErr: false},
		{name: "show allowed", sql: "SHOW TABLES", wantErr: false},
		{name: "explain allowed", sql: "EXPLAIN SELECT 1", wantErr: false},
		{name: "update rejected", sql: "UPDATE users SET active = 1", wantErr: true},
		{name: "explain delete rejected", sql: "EXPLAIN DELETE FROM users", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := dialect.ValidateReadOnly(tt.sql)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestReadOnlyRejectsEmptySQL(t *testing.T) {
	t.Parallel()

	err := driver.NewSQLiteDialect().ValidateReadOnly("   ")
	if err == nil || !strings.Contains(err.Error(), "empty SQL") {
		t.Fatalf("error = %v, want empty SQL error", err)
	}
}
