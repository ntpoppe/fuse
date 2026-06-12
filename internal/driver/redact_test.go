package driver_test

import (
	"testing"

	"github.com/ntpoppe/fuse/internal/driver"
)

func TestRedactHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		driver string
		host   string
		want   string
	}{
		{driver: driver.DriverSQLite, host: "/data/app.db", want: "/data/app.db"},
		{driver: driver.DriverMySQL, host: "user:secret@tcp(127.0.0.1:3306)/app", want: "user:***@tcp(127.0.0.1:3306)/app"},
		{driver: driver.DriverMySQL, host: "127.0.0.1:3306", want: "127.0.0.1:3306"},
	}

	for _, tt := range tests {
		t.Run(tt.driver+"_"+tt.host, func(t *testing.T) {
			t.Parallel()
			got := driver.RedactHost(tt.driver, tt.host)
			if got != tt.want {
				t.Fatalf("RedactHost() = %q, want %q", got, tt.want)
			}
		})
	}
}
