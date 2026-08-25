package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgres://future:future@localhost:5432/future_environs?sslmode=disable")
	t.Setenv("JWT_SECRET", "test-secret")

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

	t.Setenv("DATABASE_URL", "postgres://app:pass@db:5432/future_environs?sslmode=disable")
	t.Setenv("JWT_SECRET", "prod-secret")
	t.Setenv("JWT_ISSUER", "future-environs")
	t.Setenv("JWT_EXPIRY_SECONDS", "7200")
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
	if cfg.DatabaseURL != "postgres://app:pass@db:5432/future_environs?sslmode=disable" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != "prod-secret" {
		t.Errorf("JWTSecret = %q, want prod-secret", cfg.JWTSecret)
	}
	if cfg.JWTIssuer != "future-environs" {
		t.Errorf("JWTIssuer = %q, want future-environs", cfg.JWTIssuer)
	}
	if cfg.JWTExpiry != 7200*time.Second {
		t.Errorf("JWTExpiry = %v, want 2h", cfg.JWTExpiry)
	}
}

func TestLoadRequiresDatabaseURLAndJWTSecret(t *testing.T) {
	clearConfigEnv(t)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error when DATABASE_URL and JWT_SECRET are missing")
	}

	t.Setenv("DATABASE_URL", "postgres://future:future@localhost:5432/future_environs?sslmode=disable")
	_, err = Load()
	if err == nil {
		t.Fatal("Load() expected error when JWT_SECRET is missing")
	}
}

func TestLoadInvalidTimeout(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgres://future:future@localhost:5432/future_environs?sslmode=disable")
	t.Setenv("JWT_SECRET", "test-secret")
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
		"DATABASE_URL",
		"JWT_SECRET",
		"JWT_ISSUER",
		"JWT_EXPIRY_SECONDS",
	}
	for _, key := range keys {
		_ = os.Unsetenv(key)
	}
}
