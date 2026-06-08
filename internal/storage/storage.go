package storage

import (
	"context"
	"database/sql"
	"errors"
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
	_, err := s.db.Exec(schemaConnectionsCreate)
	if err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}
	return nil
}

func (s *Store) SaveConnection(ctx context.Context, conn SavedConnection) error {
	query := fmt.Sprintf("INSERT OR REPLACE INTO %s (id, driver, host) VALUES (?, ?, ?);", connectionsTable)

	_, err := s.db.ExecContext(ctx, query, conn.ID, conn.Driver, conn.Host)
	if err != nil {
		return fmt.Errorf("save connection %q: %w", conn.ID, err)
	}
	return nil
}

func (s *Store) GetConnection(ctx context.Context, id string) (SavedConnection, bool, error) {
	query := fmt.Sprintf("SELECT id, driver, host FROM %s WHERE id = ?;", connectionsTable)

	var conn SavedConnection
	err := s.db.QueryRowContext(ctx, query, id).Scan(&conn.ID, &conn.Driver, &conn.Host)
	if errors.Is(err, sql.ErrNoRows) {
		return SavedConnection{}, false, nil
	}
	if err != nil {
		return SavedConnection{}, false, fmt.Errorf("get connection %q: %w", id, err)
	}
	return conn, true, nil
}

func (s *Store) RemoveConnection(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?;", connectionsTable)

	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("remove connection %q: %w", id, err)
	}
	return nil
}

func (s *Store) GetAllConnections(ctx context.Context) ([]SavedConnection, error) {
	query := fmt.Sprintf("SELECT id, driver, host FROM %s;", connectionsTable)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}
	defer rows.Close()

	var connections []SavedConnection
	for rows.Next() {
		var conn SavedConnection
		if err := rows.Scan(&conn.ID, &conn.Driver, &conn.Host); err != nil {
			return nil, fmt.Errorf("scan connection row: %w", err)
		}
		connections = append(connections, conn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate connection rows: %w", err)
	}

	return connections, nil
}
