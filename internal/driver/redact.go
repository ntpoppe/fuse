package driver

import "strings"

// RedactHost returns a display-safe connection string with credentials masked.
// MySQL user:password@… DSNs redact the password; other drivers pass through unchanged.
func RedactHost(driverName, host string) string {
	if driverName != DriverMySQL {
		return host
	}

	at := strings.LastIndex(host, "@")
	if at <= 0 {
		return host
	}

	userPart := host[:at]
	rest := host[at:]
	colon := strings.Index(userPart, ":")
	if colon <= 0 {
		return host
	}

	return userPart[:colon+1] + "***" + rest
}
