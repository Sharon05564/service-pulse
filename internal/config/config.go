package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	ServerPort  string
	DatabaseURL string
	Env         string
}

// Load reads the .env file (if present) and then populates Config from
// environment variables. Missing required values cause an error.
func Load() (*Config, error) {
	// godotenv.Load is a no-op if the file does not exist, which is fine
	// in production where env vars are injected directly.
	_ = godotenv.Load()

	cfg := &Config{
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		Env:         getEnv("APP_ENV", "development"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	return cfg, nil
}

// getEnv returns the value of the named env var, or fallback if it is unset.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// GetIntEnv returns an integer env var, or fallback on missing / parse error.
func GetIntEnv(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
