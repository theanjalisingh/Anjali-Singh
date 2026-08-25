// Package config loads application settings from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime configuration for the API process.
type Config struct {
	AppName string
	AppEnv  string
	Port    string

	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	LogLevel string

	DatabaseURL string

	JWTSecret string
	JWTIssuer string
	JWTExpiry time.Duration
}

// Load reads configuration from the process environment.
// Missing optional values fall back to safe defaults.
func Load() (*Config, error) {
	cfg := &Config{
		AppName: getEnv("APP_NAME", "futureEnvironsBE"),
		AppEnv:  getEnv("APP_ENV", "development"),
		Port:    getEnv("APP_PORT", "8080"),

		LogLevel: getEnv("LOG_LEVEL", "info"),

		DatabaseURL: getEnv("DATABASE_URL", ""),

		JWTSecret: getEnv("JWT_SECRET", ""),
		JWTIssuer: getEnv("JWT_ISSUER", "futureEnvironsBE"),
	}

	var err error

	cfg.ReadTimeout, err = durationSeconds("HTTP_READ_TIMEOUT_SECONDS", 15)
	if err != nil {
		return nil, err
	}

	cfg.WriteTimeout, err = durationSeconds("HTTP_WRITE_TIMEOUT_SECONDS", 15)
	if err != nil {
		return nil, err
	}

	cfg.IdleTimeout, err = durationSeconds("HTTP_IDLE_TIMEOUT_SECONDS", 60)
	if err != nil {
		return nil, err
	}

	cfg.ShutdownTimeout, err = durationSeconds("HTTP_SHUTDOWN_TIMEOUT_SECONDS", 10)
	if err != nil {
		return nil, err
	}

	cfg.JWTExpiry, err = durationSeconds("JWT_EXPIRY_SECONDS", 3600)
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if strings.TrimSpace(c.Port) == "" {
		return fmt.Errorf("config: APP_PORT must not be empty")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("config: DATABASE_URL must not be empty")
	}
	if strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("config: JWT_SECRET must not be empty")
	}
	if c.JWTExpiry <= 0 {
		return fmt.Errorf("config: JWT_EXPIRY_SECONDS must be > 0")
	}
	return nil
}

// Addr returns the HTTP listen address (for example ":8080").
func (c *Config) Addr() string {
	return ":" + c.Port
}

// IsDevelopment reports whether the app is running in development mode.
func (c *Config) IsDevelopment() bool {
	return strings.EqualFold(c.AppEnv, "development") || strings.EqualFold(c.AppEnv, "dev")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func durationSeconds(key string, fallback int) (time.Duration, error) {
	raw := getEnv(key, strconv.Itoa(fallback))
	seconds, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be an integer number of seconds: %w", key, err)
	}
	if seconds < 0 {
		return 0, fmt.Errorf("config: %s must be >= 0", key)
	}
	return time.Duration(seconds) * time.Second, nil
}
