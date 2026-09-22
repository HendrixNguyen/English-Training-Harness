package store

import "testing"

// TestRequirePostgresSkipsWithoutTestDatabaseURL pins the safety property that
// `go test ./...` must never run the destructive integration tests just because
// the production DATABASE_URL (spec §8) happens to be exported.
func TestRequirePostgresSkipsWithoutTestDatabaseURL(t *testing.T) {
	// A syntactically valid URL pointing nowhere: if the gate ever lets the test
	// through, NewPostgres is reached and this subtest fails instead of skipping.
	t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2")
	t.Setenv("TEST_DATABASE_URL", "")

	var skipped bool
	t.Run("gated", func(t *testing.T) {
		defer func() { skipped = t.Skipped() }()
		requirePostgres(t)
		t.Error("requirePostgres did not skip; it would have connected and dropped tables")
	})
	if !skipped {
		t.Fatal("requirePostgres must skip when TEST_DATABASE_URL is unset")
	}
}

// TestRequireRedisURLSkipsWithoutTestRedisURL is the same property for Redis.
func TestRequireRedisURLSkipsWithoutTestRedisURL(t *testing.T) {
	t.Setenv("REDIS_URL", "redis://127.0.0.1:59999/0")
	t.Setenv("TEST_REDIS_URL", "")

	var skipped bool
	t.Run("gated", func(t *testing.T) {
		defer func() { skipped = t.Skipped() }()
		_ = requireRedisURL(t)
		t.Error("requireRedisURL did not skip")
	})
	if !skipped {
		t.Fatal("requireRedisURL must skip when TEST_REDIS_URL is unset")
	}
}
