// Package config loads the process environment described in spec §8.
package config

import (
	"fmt"
	"os"
)

// Config holds the environment this slice needs. Later slices add fields
// (GOOGLE_CLIENT_ID, JWT_SECRET, VAPID_*) as they are introduced.
type Config struct {
	DatabaseURL        string
	RedisURL           string
	Port               string
	GoogleClientID     string
	GoogleClientSecret string
	// JWTSecret signs session tokens. NOTE: JWT_SECRET is NOT in the spec §8
	// environment list — see the CODEMAP auth paragraph and the plan's open questions.
	JWTSecret string
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

	cfg.GoogleClientID = os.Getenv("GOOGLE_CLIENT_ID")
	cfg.GoogleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	cfg.JWTSecret = os.Getenv("JWT_SECRET")

	for name, v := range map[string]string{
		"GOOGLE_CLIENT_ID":     cfg.GoogleClientID,
		"GOOGLE_CLIENT_SECRET": cfg.GoogleClientSecret,
		"JWT_SECRET":           cfg.JWTSecret,
	} {
		if v == "" {
			return Config{}, fmt.Errorf("config: %s is required", name)
		}
	}
	return cfg, nil
}
