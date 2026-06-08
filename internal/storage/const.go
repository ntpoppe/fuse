package storage

const (
	DefaultStateDBPath      = "fuse.db"
	connectionsTable        = "saved_connections"
	schemaConnectionsCreate = `
CREATE TABLE IF NOT EXISTS saved_connections (
	id TEXT PRIMARY KEY,
	driver TEXT NOT NULL,
	host TEXT NOT NULL
);`
)
