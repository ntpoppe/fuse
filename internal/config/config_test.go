package config_test

import (
	"testing"

	"github.com/ntpoppe/fuse/internal/config"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     config.Config
		wantErr bool
	}{
		{
			name: "valid dev config",
			cfg:  config.Config{Port: 8080, Env: "dev"},
		},
		{
			name: "valid prod config",
			cfg:  config.Config{Port: 8080, Env: "prod"},
		},
		{
			name: "privileged port allowed",
			cfg:  config.Config{Port: 80, Env: "dev"},
		},
		{
			name:    "port too low",
			cfg:     config.Config{Port: 0, Env: "dev"},
			wantErr: true,
		},
		{
			name:    "port too high",
			cfg:     config.Config{Port: 99999, Env: "dev"},
			wantErr: true,
		},
		{
			name:    "invalid environment",
			cfg:     config.Config{Port: 8080, Env: "staging"},
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
