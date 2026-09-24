// Package config loads the process environment described in spec §8.
package config

import (
	"fmt"
	"os"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/secrets"
)

// DefaultVAPIDSubject is used when VAPID_SUBJECT is unset. Replace with a real
// contact before the first production push (see the notify plan's notes).
const DefaultVAPIDSubject = "mailto:admin@example.com"

// MinJWTSecretBytes is the HS256 floor: RFC 7518 §3.2 requires a key at least
// as long as the hash output (256 bits). A shorter secret is brute-forceable
// offline from one captured token.
const MinJWTSecretBytes = 32

// Config holds the environment this slice needs (GOOGLE_CLIENT_ID, JWT_SECRET,
// VAPID_* have been added as their slices landed).
type Config struct {
	DatabaseURL        string
	RedisURL           string
	Port               string
	GoogleClientID     string
	GoogleClientSecret string
	// JWTSecret signs session tokens (HS256) and must be ≥ MinJWTSecretBytes.
	// NOTE: JWT_SECRET is NOT in the 1st-thinking §8 list — see CODEMAP auth.
	JWTSecret string
	// EncryptionKey is the decoded ENCRYPTION_SECRET_KEY (backend spec §7/§9):
	// 32 raw bytes for AES-256-GCM over users.google_refresh_token. Required.
	EncryptionKey []byte
	// VAPIDPublicKey / VAPIDPrivateKey sign Web Push requests (spec §9). They
	// are OPTIONAL at boot: without both, the notify worker does not start and
	// reminder settings are stored but nothing is sent (see cmd/api/main.go).
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	// VAPIDSubject is the VAPID JWT `sub` claim (a mailto: or https: URL push
	// services may contact). NOT in spec §9; defaults to DefaultVAPIDSubject.
	VAPIDSubject string
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

	if len(cfg.JWTSecret) < MinJWTSecretBytes {
		return Config{}, fmt.Errorf("config: JWT_SECRET must be at least %d bytes (generate one with: openssl rand -base64 32)", MinJWTSecretBytes)
	}
	rawKey := os.Getenv("ENCRYPTION_SECRET_KEY")
	if rawKey == "" {
		return Config{}, fmt.Errorf("config: ENCRYPTION_SECRET_KEY is required — 32 bytes as 64 hex characters (generate one with: openssl rand -hex 32)")
	}
	key, err := secrets.ParseHexKey(rawKey)
	if err != nil {
		return Config{}, fmt.Errorf("config: ENCRYPTION_SECRET_KEY must be 32 bytes as 64 hex characters (generate one with: openssl rand -hex 32)")
	}
	cfg.EncryptionKey = key

	cfg.VAPIDPublicKey = os.Getenv("VAPID_PUBLIC_KEY")
	cfg.VAPIDPrivateKey = os.Getenv("VAPID_PRIVATE_KEY")
	if cfg.VAPIDSubject = os.Getenv("VAPID_SUBJECT"); cfg.VAPIDSubject == "" {
		cfg.VAPIDSubject = DefaultVAPIDSubject
	}
	return cfg, nil
}
