package config_test

import (
	"os"
	"testing"

	"github.com/sharonowusu/servicepulse/internal/config"
)

func TestLoad_MissingDatabaseURL(t *testing.T) {
	// Ensure DATABASE_URL is not set so we can test the required-field error.
	os.Unsetenv("DATABASE_URL")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is missing, got nil")
	}
}

func TestLoad_Success(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db?sslmode=disable")
	defer os.Unsetenv("DATABASE_URL")

	os.Setenv("SERVER_PORT", "9090")
	defer os.Unsetenv("SERVER_PORT")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ServerPort != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.ServerPort)
	}
	if cfg.DatabaseURL == "" {
		t.Error("expected DatabaseURL to be set")
	}
}

func TestGetIntEnv_Default(t *testing.T) {
	os.Unsetenv("MY_INT")
	if got := config.GetIntEnv("MY_INT", 42); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestGetIntEnv_Set(t *testing.T) {
	os.Setenv("MY_INT", "99")
	defer os.Unsetenv("MY_INT")
	if got := config.GetIntEnv("MY_INT", 0); got != 99 {
		t.Errorf("expected 99, got %d", got)
	}
}

func TestGetIntEnv_Invalid(t *testing.T) {
	os.Setenv("MY_INT", "not-a-number")
	defer os.Unsetenv("MY_INT")
	if got := config.GetIntEnv("MY_INT", 7); got != 7 {
		t.Errorf("expected fallback 7, got %d", got)
	}
}
