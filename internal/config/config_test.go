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
				Port:         8080,
				StateDBPath:  storage.DefaultStateDBPath,
				MaxQueryRows: config.DefaultMaxQueryRows,
			},
		},
		{
			name: "privileged port allowed",
			cfg: config.Config{
				Port:         80,
				StateDBPath:  storage.DefaultStateDBPath,
				MaxQueryRows: config.DefaultMaxQueryRows,
			},
		},
		{
			name: "port too low",
			cfg: config.Config{
				Port:         0,
				StateDBPath:  storage.DefaultStateDBPath,
				MaxQueryRows: config.DefaultMaxQueryRows,
			},
			wantErr: true,
		},
		{
			name: "port too high",
			cfg: config.Config{
				Port:         99999,
				StateDBPath:  storage.DefaultStateDBPath,
				MaxQueryRows: config.DefaultMaxQueryRows,
			},
			wantErr: true,
		},
		{
			name: "empty state db path",
			cfg: config.Config{
				Port:         8080,
				StateDBPath:  "",
				MaxQueryRows: config.DefaultMaxQueryRows,
			},
			wantErr: true,
		},
		{
			name: "max query rows too low",
			cfg: config.Config{
				Port:         8080,
				StateDBPath:  storage.DefaultStateDBPath,
				MaxQueryRows: 0,
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
