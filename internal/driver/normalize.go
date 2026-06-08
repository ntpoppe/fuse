package driver

import (
	"fmt"
	"strings"
)

const (
	sqliteFilePrefix     = "file:"
	sqliteReadOnlySuffix = "?mode=ro"
)

func NormalizeHost(driverName, host string) string {
	if driverName != DriverSQLite {
		return host
	}

	cleaned := strings.TrimPrefix(host, sqliteFilePrefix)
	cleaned = strings.TrimSuffix(cleaned, sqliteReadOnlySuffix)
	return fmt.Sprintf("%s%s%s", sqliteFilePrefix, cleaned, sqliteReadOnlySuffix)
}
