package config

import "testing"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", "s3cret")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when DATABASE_URL is unset, got nil")
	}
}

func TestLoadRequiresRedisURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", "s3cret")

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
	t.Setenv("JWT_SECRET", "s3cret")

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
	t.Setenv("JWT_SECRET", "s3cret")

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
		t.Setenv("JWT_SECRET", "s3cret")
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
		if cfg.GoogleClientID != "cid" || cfg.GoogleClientSecret != "csecret" || cfg.JWTSecret != "s3cret" {
			t.Errorf("got %+v", cfg)
		}
	})
}

func TestLoadReadsOptionalVAPIDKeys(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", "s3cret")
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
