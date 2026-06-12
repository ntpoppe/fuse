package driver

import (
	"net/url"
	"strings"
)

const (
	sqliteFilePrefix = "file:"
)

func NormalizeHost(driverName, host string) string {
	if driverName != DriverSQLite {
		return host
	}

	rest := strings.TrimPrefix(host, sqliteFilePrefix)
	filePart := rest
	queryPart := ""
	if idx := strings.Index(rest, "?"); idx >= 0 {
		filePart = rest[:idx]
		queryPart = rest[idx+1:]
	}

	vals, _ := url.ParseQuery(queryPart)
	vals.Set("mode", "ro")

	return sqliteFilePrefix + filePart + "?" + vals.Encode()
}
