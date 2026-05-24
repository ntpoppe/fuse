package storage

import (
	"context"
	"database/sql"
	"fmt"
)

type SavedConnection struct {
	ID     string
	Driver string
	Host   string
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) InitializeSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS saved_connections (
		id TEXT PRIMARY KEY,
		driver TEXT NOT NULL,
		host TEXT NOT NULL
	);`

	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to process local configuration migration schema: %w", err)
	}
	return nil
}

func (s *Store) SaveConnection(ctx context.Context, conn SavedConnection) error {
	query := `INSERT OR REPLACE INTO saved_connections (id, driver, host) VALUES (?, ?, ?);`

	_, err := s.db.ExecContext(ctx, query, conn.ID, conn.Driver, conn.Host)
	if err != nil {
		return fmt.Errorf("failed to save configuration parameters to local store file: %w", err)
	}
	return nil
}

func (s *Store) GetAllConnections(ctx context.Context) ([]SavedConnection, error) {
	query := `SELECT id, driver, host FROM saved_connections;`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to look up saved configuration rows: %w", err)
	}
	defer rows.Close()

	var connections []SavedConnection
	for rows.Next() {
		var conn SavedConnection
		if err := rows.Scan(&conn.ID, &conn.Driver, &conn.Host); err != nil {
			return nil, fmt.Errorf("failed to deserialize connection tracking record: %w", err)
		}
		connections = append(connections, conn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row stream encountered mid-flight failure: %w", err)
	}

	return connections, nil
}
