package config

import (
	"fmt"
	"strings"
	"testing"
)

// testJWTSecret is 32 bytes — the HS256 floor config enforces.
const testJWTSecret = "0123456789abcdef0123456789abcdef"

// testHexKey is 64 hex characters — a well-formed ENCRYPTION_SECRET_KEY.
const testHexKey = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("ENCRYPTION_SECRET_KEY", testHexKey)

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when DATABASE_URL is unset, got nil")
	}
}

func TestLoadRequiresRedisURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("ENCRYPTION_SECRET_KEY", testHexKey)

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when REDIS_URL is unset, got nil")
	}
}

func TestLoadDefaultsPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("PORT", "")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("ENCRYPTION_SECRET_KEY", testHexKey)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil error", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.DatabaseURL != "postgres://u:p@localhost:5432/db" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("RedisURL = %q", cfg.RedisURL)
	}
}

func TestLoadHonoursPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("PORT", "9999")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("ENCRYPTION_SECRET_KEY", testHexKey)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil error", err)
	}
	if cfg.Port != "9999" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9999")
	}
}

func TestLoadRequiresGoogleAndJWTSecrets(t *testing.T) {
	base := func(t *testing.T) {
		t.Helper()
		t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
		t.Setenv("REDIS_URL", "redis://localhost:6379/0")
		t.Setenv("GOOGLE_CLIENT_ID", "cid")
		t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
		t.Setenv("JWT_SECRET", testJWTSecret)
		t.Setenv("ENCRYPTION_SECRET_KEY", testHexKey)
	}

	for _, missing := range []string{"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "JWT_SECRET"} {
		t.Run("missing "+missing, func(t *testing.T) {
			base(t)
			t.Setenv(missing, "")
			if _, err := Load(); err == nil {
				t.Fatalf("expected an error when %s is unset, got nil", missing)
			}
		})
	}

	t.Run("all present", func(t *testing.T) {
		base(t)
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() = %v, want nil error", err)
		}
		if cfg.GoogleClientID != "cid" || cfg.GoogleClientSecret != "csecret" || cfg.JWTSecret != testJWTSecret {
			t.Errorf("got %+v", cfg)
		}
	})
}

func TestLoadReadsOptionalVAPIDKeys(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("ENCRYPTION_SECRET_KEY", testHexKey)
	t.Setenv("VAPID_PUBLIC_KEY", "")
	t.Setenv("VAPID_PRIVATE_KEY", "")
	t.Setenv("VAPID_SUBJECT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() without VAPID keys = %v, want nil (they are optional)", err)
	}
	if cfg.VAPIDPublicKey != "" || cfg.VAPIDPrivateKey != "" {
		t.Errorf("VAPID keys = %q/%q, want empty", cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey)
	}
	if cfg.VAPIDSubject != DefaultVAPIDSubject {
		t.Errorf("VAPIDSubject = %q, want the default %q", cfg.VAPIDSubject, DefaultVAPIDSubject)
	}

	t.Setenv("VAPID_PUBLIC_KEY", "BPub")
	t.Setenv("VAPID_PRIVATE_KEY", "priv")
	t.Setenv("VAPID_SUBJECT", "mailto:ops@example.com")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.VAPIDPublicKey != "BPub" || cfg.VAPIDPrivateKey != "priv" || cfg.VAPIDSubject != "mailto:ops@example.com" {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestLoadDefaultsGinModeToRelease(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("ENCRYPTION_SECRET_KEY", testHexKey)
	t.Setenv("GIN_MODE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	// gin.Default() alone would run in debug mode; production must not.
	if cfg.GinMode != "release" {
		t.Fatalf("GinMode = %q, want release", cfg.GinMode)
	}
}

func TestLoadRejectsAnUnknownGinMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("ENCRYPTION_SECRET_KEY", testHexKey)

	for _, mode := range []string{"debug", "release", "test"} {
		t.Setenv("GIN_MODE", mode)
		if cfg, err := Load(); err != nil || cfg.GinMode != mode {
			t.Fatalf("GIN_MODE=%s: cfg=%+v err=%v", mode, cfg, err)
		}
	}
	t.Setenv("GIN_MODE", "verbose")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for GIN_MODE=verbose (gin.SetMode would panic), got nil")
	}
}

func TestLoadDefaultsFrontendOriginToTheNuxtDevServer(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("ENCRYPTION_SECRET_KEY", testHexKey)
	t.Setenv("FRONTEND_ORIGIN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if cfg.FrontendOrigin != DefaultFrontendOrigin || DefaultFrontendOrigin != "http://localhost:3000" {
		t.Errorf("FrontendOrigin = %q, want the Nuxt dev server default %q", cfg.FrontendOrigin, "http://localhost:3000")
	}

	t.Setenv("FRONTEND_ORIGIN", "https://app.example.com, https://staging.example.com")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FrontendOrigin != "https://app.example.com, https://staging.example.com" {
		t.Errorf("FrontendOrigin = %q, want the raw value (middleware.ParseOrigins validates it)", cfg.FrontendOrigin)
	}
}

func TestLoadRejectsAShortJWTSecret(t *testing.T) {
	for _, tc := range []struct {
		n  int
		ok bool
	}{{1, false}, {31, false}, {32, true}, {64, true}} {
		t.Run(fmt.Sprintf("%d bytes", tc.n), func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
			t.Setenv("REDIS_URL", "redis://localhost:6379/0")
			t.Setenv("GOOGLE_CLIENT_ID", "cid")
			t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
			t.Setenv("ENCRYPTION_SECRET_KEY", testHexKey)
			t.Setenv("JWT_SECRET", strings.Repeat("x", tc.n))
			_, err := Load()
			if tc.ok && err != nil {
				t.Fatalf("Load() with a %d-byte JWT_SECRET = %v, want nil", tc.n, err)
			}
			if !tc.ok {
				if err == nil {
					t.Fatalf("Load() accepted a %d-byte JWT_SECRET; RFC 7518 §3.2 wants ≥ %d", tc.n, MinJWTSecretBytes)
				}
				if !strings.Contains(err.Error(), "JWT_SECRET must be at least 32 bytes") || !strings.Contains(err.Error(), "openssl rand -base64 32") {
					t.Errorf("error = %q, want the requirement and the generate hint", err)
				}
			}
		})
	}
}

func TestLoadRequiresAWellFormedEncryptionKey(t *testing.T) {
	for _, tc := range []struct {
		name, key, wantErr string
	}{
		{"unset", "", "ENCRYPTION_SECRET_KEY is required"},
		{"62 hex", strings.Repeat("ab", 31), "64 hex characters"},
		{"66 hex", strings.Repeat("ab", 33), "64 hex characters"},
		{"not hex", strings.Repeat("zz", 32), "64 hex characters"},
		{"64 hex", testHexKey, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
			t.Setenv("REDIS_URL", "redis://localhost:6379/0")
			t.Setenv("GOOGLE_CLIENT_ID", "cid")
			t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
			t.Setenv("JWT_SECRET", testJWTSecret)
			t.Setenv("ENCRYPTION_SECRET_KEY", tc.key)
			cfg, err := Load()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Load() = %v, want nil", err)
				}
				if len(cfg.EncryptionKey) != 32 || cfg.EncryptionKey[0] != 0x00 || cfg.EncryptionKey[1] != 0x11 {
					t.Errorf("EncryptionKey = %x, want the 32 decoded bytes", cfg.EncryptionKey)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) || !strings.Contains(err.Error(), "openssl rand -hex 32") {
				t.Errorf("Load() err = %v, want it to mention %q and the generate hint", err, tc.wantErr)
			}
		})
	}
}
