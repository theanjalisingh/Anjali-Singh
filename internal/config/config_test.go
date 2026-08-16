package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.AppName != "futureEnvironsBE" {
		t.Errorf("AppName = %q, want futureEnvironsBE", cfg.AppName)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.Addr() != ":8080" {
		t.Errorf("Addr() = %q, want :8080", cfg.Addr())
	}
	if cfg.ReadTimeout != 15*time.Second {
		t.Errorf("ReadTimeout = %v, want 15s", cfg.ReadTimeout)
	}
	if !cfg.IsDevelopment() {
		t.Errorf("IsDevelopment() = false, want true for default APP_ENV")
	}
}

func TestLoadCustomValues(t *testing.T) {
	clearConfigEnv(t)

	t.Setenv("APP_NAME", "test-api")
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("HTTP_READ_TIMEOUT_SECONDS", "30")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.AppName != "test-api" {
		t.Errorf("AppName = %q, want test-api", cfg.AppName)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout = %v, want 30s", cfg.ReadTimeout)
	}
	if cfg.IsDevelopment() {
		t.Errorf("IsDevelopment() = true, want false for production")
	}
}

func TestLoadInvalidTimeout(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("HTTP_READ_TIMEOUT_SECONDS", "abc")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid timeout, got nil")
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"APP_NAME",
		"APP_ENV",
		"APP_PORT",
		"HTTP_READ_TIMEOUT_SECONDS",
		"HTTP_WRITE_TIMEOUT_SECONDS",
		"HTTP_IDLE_TIMEOUT_SECONDS",
		"HTTP_SHUTDOWN_TIMEOUT_SECONDS",
		"LOG_LEVEL",
	}
	for _, key := range keys {
		_ = os.Unsetenv(key)
	}
}
