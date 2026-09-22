// Package config loads the process environment described in spec §8.
package config

import (
	"fmt"
	"os"
)

// Config holds the environment this slice needs. Later slices add fields
// (GOOGLE_CLIENT_ID, JWT_SECRET, VAPID_*) as they are introduced.
type Config struct {
	DatabaseURL string
	RedisURL    string
	Port        string
}

// Load reads the environment and validates the required variables.
func Load() (Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		Port:        os.Getenv("PORT"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: DATABASE_URL is required")
	}
	if cfg.RedisURL == "" {
		return Config{}, fmt.Errorf("config: REDIS_URL is required")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	return cfg, nil
}
