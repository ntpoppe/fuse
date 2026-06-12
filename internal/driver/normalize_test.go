package driver_test

import (
	"testing"

	"github.com/ntpoppe/fuse/internal/driver"
)

func TestNormalizeHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		driver string
		host   string
		want   string
	}{
		{name: "sqlite plain path", driver: driver.DriverSQLite, host: "dev_target.db", want: "file:dev_target.db?mode=ro"},
		{name: "sqlite file prefix", driver: driver.DriverSQLite, host: "file:dev_target.db", want: "file:dev_target.db?mode=ro"},
		{name: "sqlite already read only", driver: driver.DriverSQLite, host: "file:dev_target.db?mode=ro", want: "file:dev_target.db?mode=ro"},
		{name: "sqlite plain path with existing mode suffix", driver: driver.DriverSQLite, host: "dev_target.db?mode=ro", want: "file:dev_target.db?mode=ro"},
		{name: "sqlite with extra query params", driver: driver.DriverSQLite, host: "file:path.db?cache=shared", want: "file:path.db?cache=shared&mode=ro"},
		{name: "sqlite plain path with cache param", driver: driver.DriverSQLite, host: "dev.db?cache=shared", want: "file:dev.db?cache=shared&mode=ro"},
		{name: "mysql passthrough", driver: driver.DriverMySQL, host: "user:pass@tcp(localhost:3306)/mydb", want: "user:pass@tcp(localhost:3306)/mydb"},
		{name: "sql server passthrough", driver: "sqlserver", host: "sqlserver://user:pass@localhost:1433?database=mydb", want: "sqlserver://user:pass@localhost:1433?database=mydb"},
		{name: "unknown driver passthrough", driver: "postgres", host: "postgres://localhost/mydb", want: "postgres://localhost/mydb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := driver.NormalizeHost(tt.driver, tt.host)
			if got != tt.want {
				t.Fatalf("NormalizeHost(%q, %q) = %q, want %q", tt.driver, tt.host, got, tt.want)
			}
		})
	}
}
