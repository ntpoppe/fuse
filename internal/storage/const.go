package storage

const (
	DefaultStateDBPath = "fuse.db"
	connectionsTable   = "saved_connections"

	querySaveConnection = `
INSERT OR REPLACE INTO saved_connections (id, driver, host) VALUES (?, ?, ?);`

	queryGetConnection = `
SELECT id, driver, host FROM saved_connections WHERE id = ?;`

	queryRemoveConnection = `
DELETE FROM saved_connections WHERE id = ?;`

	queryListConnections = `
SELECT id, driver, host FROM saved_connections;`

	queryRemoveAllConnections = `
DELETE FROM saved_connections;`

	schemaConnectionsCreate = `
CREATE TABLE IF NOT EXISTS saved_connections (
	id TEXT PRIMARY KEY,
	driver TEXT NOT NULL,
	host TEXT NOT NULL
);`
)
