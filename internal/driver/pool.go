package driver

import (
	"database/sql"
	"time"
)

const (
	defaultMaxOpenConns = 4
	defaultMaxIdleConns = 2
	connMaxLifetime     = 30 * time.Minute
)

func configurePool(db *sql.DB) {
	db.SetMaxOpenConns(defaultMaxOpenConns)
	db.SetMaxIdleConns(defaultMaxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
}
