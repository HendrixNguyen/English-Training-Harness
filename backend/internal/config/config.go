// Package config loads the process environment described in spec §8.
package config

import (
	"fmt"
	"os"
)

// DefaultVAPIDSubject is used when VAPID_SUBJECT is unset. Replace with a real
// contact before the first production push (see the notify plan's notes).
const DefaultVAPIDSubject = "mailto:admin@example.com"

// DefaultFrontendOrigin is the Nuxt dev server. Production sets FRONTEND_ORIGIN
// to the PWA's own Railway origin (backend spec §9): the PWA and the API are
// two services on two origins, so every browser call is cross-origin.
const DefaultFrontendOrigin = "http://localhost:3000"

// Config holds the environment this slice needs (GOOGLE_CLIENT_ID, JWT_SECRET,
// VAPID_* have been added as their slices landed).
type Config struct {
	DatabaseURL        string
	RedisURL           string
	Port               string
	GoogleClientID     string
	GoogleClientSecret string
	// JWTSecret signs session tokens. NOTE: JWT_SECRET is NOT in the spec §8
	// environment list — see the CODEMAP auth paragraph and the plan's open questions.
	JWTSecret string
	// VAPIDPublicKey / VAPIDPrivateKey sign Web Push requests (spec §9). They
	// are OPTIONAL at boot: without both, the notify worker does not start and
	// reminder settings are stored but nothing is sent (see cmd/api/main.go).
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	// VAPIDSubject is the VAPID JWT `sub` claim (a mailto: or https: URL push
	// services may contact). NOT in spec §9; defaults to DefaultVAPIDSubject.
	VAPIDSubject string
	// GinMode is Gin's run mode: "release" (default), "debug" or "test". Read
	// from GIN_MODE and validated here so the deployed binary never runs Gin's
	// debug logging by accident — gin.Default() alone defaults to debug — and
	// so gin.SetMode (which panics on an unknown value) is only ever given a
	// valid one. Not in spec §8; documented in backend/.env.example.
	GinMode string
	// FrontendOrigin is FRONTEND_ORIGIN as given: a comma-separated allow-list
	// of exact scheme://host[:port] origins. middleware.ParseOrigins validates
	// it at wiring time; config only supplies the default.
	FrontendOrigin string
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

	cfg.VAPIDPublicKey = os.Getenv("VAPID_PUBLIC_KEY")
	cfg.VAPIDPrivateKey = os.Getenv("VAPID_PRIVATE_KEY")
	if cfg.VAPIDSubject = os.Getenv("VAPID_SUBJECT"); cfg.VAPIDSubject == "" {
		cfg.VAPIDSubject = DefaultVAPIDSubject
	}

	cfg.GinMode = os.Getenv("GIN_MODE")
	if cfg.GinMode == "" {
		cfg.GinMode = "release"
	}
	switch cfg.GinMode {
	case "debug", "release", "test":
	default:
		return Config{}, fmt.Errorf("config: GIN_MODE must be debug, release or test, got %q", cfg.GinMode)
	}

	if cfg.FrontendOrigin = os.Getenv("FRONTEND_ORIGIN"); cfg.FrontendOrigin == "" {
		cfg.FrontendOrigin = DefaultFrontendOrigin
	}
	return cfg, nil
}
