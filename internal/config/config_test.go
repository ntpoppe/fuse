package config_test

import (
	"testing"

	"github.com/ntpoppe/fuse/internal/config"
	"github.com/ntpoppe/fuse/internal/storage"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: config.Config{
				Host:         config.DefaultHost,
				Port:         8080,
				StateDBPath:  storage.DefaultStateDBPath,
				MaxQueryRows: config.DefaultMaxQueryRows,
			},
		},
		{
			name: "privileged port allowed",
			cfg: config.Config{
				Host:         config.DefaultHost,
				Port:         80,
				StateDBPath:  storage.DefaultStateDBPath,
				MaxQueryRows: config.DefaultMaxQueryRows,
			},
		},
		{
			name: "port too low",
			cfg: config.Config{
				Host:         config.DefaultHost,
				Port:         0,
				StateDBPath:  storage.DefaultStateDBPath,
				MaxQueryRows: config.DefaultMaxQueryRows,
			},
			wantErr: true,
		},
		{
			name: "port too high",
			cfg: config.Config{
				Host:         config.DefaultHost,
				Port:         99999,
				StateDBPath:  storage.DefaultStateDBPath,
				MaxQueryRows: config.DefaultMaxQueryRows,
			},
			wantErr: true,
		},
		{
			name: "empty state db path",
			cfg: config.Config{
				Host:         config.DefaultHost,
				Port:         8080,
				StateDBPath:  "",
				MaxQueryRows: config.DefaultMaxQueryRows,
			},
			wantErr: true,
		},
		{
			name: "max query rows too low",
			cfg: config.Config{
				Host:         config.DefaultHost,
				Port:         8080,
				StateDBPath:  storage.DefaultStateDBPath,
				MaxQueryRows: 0,
			},
			wantErr: true,
		},
		{
			name: "demo mode allows 0.0.0.0 host",
			cfg: config.Config{
				Host:           "0.0.0.0",
				Port:           8080,
				StateDBPath:    storage.DefaultStateDBPath,
				MaxQueryRows:   config.DefaultMaxQueryRows,
				DemoMode:       true,
				DemoSQLitePath: config.DefaultDemoSQLitePath,
				DemoMySQLDSN:   config.DefaultDemoMySQLDSN,
			},
		},
		{
			name: "demo mode requires sqlite path",
			cfg: config.Config{
				Host:         config.DefaultHost,
				Port:         8080,
				StateDBPath:  storage.DefaultStateDBPath,
				MaxQueryRows: config.DefaultMaxQueryRows,
				DemoMode:     true,
				DemoMySQLDSN: config.DefaultDemoMySQLDSN,
			},
			wantErr: true,
		},
		{
			name: "invalid host",
			cfg: config.Config{
				Host:         "not-a-host",
				Port:         8080,
				StateDBPath:  storage.DefaultStateDBPath,
				MaxQueryRows: config.DefaultMaxQueryRows,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestApplyEnv_CORSOrigins(t *testing.T) {
	t.Setenv("FUSE_CORS_ORIGINS", "http://localhost:8080, http://127.0.0.1:8080")

	cfg := config.NewConfig()
	config.ApplyEnv(cfg)

	if len(cfg.CORSOrigins) != 2 {
		t.Fatalf("CORS origins = %v, want 2", cfg.CORSOrigins)
	}
}

func TestApplyDemoDefaults_MaxQueryRows(t *testing.T) {
	t.Parallel()

	cfg := config.NewConfig()
	cfg.DemoMode = true
	config.ApplyDemoDefaults(cfg)

	if cfg.MaxQueryRows != config.DefaultDemoMaxQueryRows {
		t.Fatalf("MaxQueryRows = %d, want %d", cfg.MaxQueryRows, config.DefaultDemoMaxQueryRows)
	}

	cfgCustom := config.NewConfig()
	cfgCustom.DemoMode = true
	cfgCustom.MaxQueryRows = 500
	config.ApplyDemoDefaults(cfgCustom)
	if cfgCustom.MaxQueryRows != 500 {
		t.Fatalf("MaxQueryRows = %d, want explicit 500 preserved", cfgCustom.MaxQueryRows)
	}
}

func TestApplyDemoDefaults(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DemoMode:       true,
		DemoSQLitePath: config.DefaultDemoSQLitePath,
		DemoMySQLDSN:   config.DefaultDemoMySQLDSN,
	}
	config.ApplyDemoDefaults(&cfg)

	if len(cfg.CORSOrigins) != 2 {
		t.Fatalf("CORS origins = %v, want 2 demo defaults", cfg.CORSOrigins)
	}

	cfgWithCORS := config.Config{
		DemoMode:    true,
		CORSOrigins: []string{"http://example.com"},
	}
	config.ApplyDemoDefaults(&cfgWithCORS)
	if len(cfgWithCORS.CORSOrigins) != 1 || cfgWithCORS.CORSOrigins[0] != "http://example.com" {
		t.Fatalf("CORS origins = %v, want preserved custom origin", cfgWithCORS.CORSOrigins)
	}
}
