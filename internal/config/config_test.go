package config_test

import (
	"testing"

	"github.com/ntpoppe/fuse/internal/config"
)

func TestValidate_ValidConfig(t *testing.T) {
	c := &config.Config{Port: 8080, Env: "dev"}
	if err := c.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidate_PortTooLow(t *testing.T) {
	c := &config.Config{Port: 0, Env: "dev"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for port 0, got nil")
	}
}

func TestValidate_PortTooHigh(t *testing.T) {
	c := &config.Config{Port: 99999, Env: "dev"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for port 99999, got nil")
	}
}

func TestValidate_InvalidEnv(t *testing.T) {
	c := &config.Config{Port: 8080, Env: "staging"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for invalid env, got nil")
	}
}

func TestValidate_ProdEnv(t *testing.T) {
	c := &config.Config{Port: 8080, Env: "prod"}
	if err := c.Validate(); err != nil {
		t.Errorf("expected no error for prod env, got %v", err)
	}
}

func TestValidate_PrivilegedPort(t *testing.T) {
	// Port 80 is privileged but should not return an error
	c := &config.Config{Port: 80, Env: "dev"}
	if err := c.Validate(); err != nil {
		t.Errorf("expected no error for privileged port, got %v", err)
	}
}
