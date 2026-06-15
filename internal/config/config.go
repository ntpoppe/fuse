package config

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/ntpoppe/fuse/internal/storage"
)

const (
	DefaultHost              = "127.0.0.1"
	DefaultPort              = 5000
	DefaultMaxQueryRows      = 10_000
	DefaultDemoMaxQueryRows  = 1_000
	DefaultDemoSQLitePath    = "/data/shop.db"
	DefaultDemoMySQLDSN      = "demo:demo@tcp(mysql:3306)/fuse_test"
)

var DefaultDemoCORSOrigins = []string{
	"http://localhost:8080",
	"http://127.0.0.1:8080",
}

type Config struct {
	Host           string
	Port           int
	StateDBPath    string
	MaxQueryRows   int
	DemoMode       bool
	DemoSQLitePath string
	DemoMySQLDSN   string
	CORSOrigins    []string
}

func NewConfig() *Config {
	return &Config{
		Host:           DefaultHost,
		StateDBPath:    storage.DefaultStateDBPath,
		MaxQueryRows:   DefaultMaxQueryRows,
		DemoSQLitePath: DefaultDemoSQLitePath,
		DemoMySQLDSN:   DefaultDemoMySQLDSN,
	}
}

func ApplyEnv(cfg *Config) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("FUSE_DEMO_MODE"))) {
	case "1", "true":
		cfg.DemoMode = true
	}
	if v := os.Getenv("FUSE_DEMO_SQLITE_PATH"); v != "" {
		cfg.DemoSQLitePath = v
	}
	if v := os.Getenv("FUSE_DEMO_MYSQL_DSN"); v != "" {
		cfg.DemoMySQLDSN = v
	}
	if v := os.Getenv("FUSE_CORS_ORIGINS"); v != "" {
		cfg.CORSOrigins = SplitCSV(v)
	}
}

func ApplyDemoDefaults(cfg *Config) {
	if !cfg.DemoMode {
		return
	}
	if cfg.MaxQueryRows == DefaultMaxQueryRows {
		cfg.MaxQueryRows = DefaultDemoMaxQueryRows
	}
	if len(cfg.CORSOrigins) == 0 {
		cfg.CORSOrigins = append([]string(nil), DefaultDemoCORSOrigins...)
	}
}

func SplitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host must not be empty")
	}
	if ip := net.ParseIP(c.Host); ip == nil && c.Host != "localhost" {
		return fmt.Errorf("host %q is not a valid IP address or localhost", c.Host)
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port %d is out of range (1-65535)", c.Port)
	}

	if c.Port < 1024 {
		fmt.Printf("warning: port %d is privileged; root access may be required\n", c.Port)
	}

	if c.StateDBPath == "" {
		return fmt.Errorf("state database path must not be empty")
	}

	if c.MaxQueryRows < 1 {
		return fmt.Errorf("max query rows must be at least 1")
	}

	if c.DemoMode {
		if c.DemoSQLitePath == "" {
			return fmt.Errorf("demo sqlite path must not be empty")
		}
		if c.DemoMySQLDSN == "" {
			return fmt.Errorf("demo mysql DSN must not be empty")
		}
	}

	return nil
}
