package driver

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

const (
	multiStatementsDSNFragment = "multistatements=true"
	targetPingTimeout          = 3 * time.Second
)

func OpenTarget(id, driverName, host string) (Target, error) {
	if strings.Contains(strings.ToLower(host), multiStatementsDSNFragment) {
		return nil, fmt.Errorf("multiStatements is not allowed in connection host")
	}

	dsn := NormalizeHost(driverName, host)
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open connection for driver %q: %w", driverName, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), targetPingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping connection for driver %q: %w", driverName, err)
	}

	return newSQLTarget(id, driverName, db), nil
}
