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

// TestRequirePostgresProceedsWithTestDatabaseURL is the other half of the gate —
// it must let a nominated database through, or every TestIntegration* here
// silently skips. NewPostgres builds a lazy pool, so the unreachable address is
// never dialled; the pool is closed by requirePostgres's own t.Cleanup.
func TestRequirePostgresProceedsWithTestDatabaseURL(t *testing.T) {
	t.Setenv("TEST_DATABASE_URL", "postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2")

	var skipped bool
	var pg *Postgres
	t.Run("gated", func(t *testing.T) {
		defer func() { skipped = t.Skipped() }()
		pg = requirePostgres(t)
	})
	if skipped {
		t.Fatal("requirePostgres must not skip when TEST_DATABASE_URL is set")
	}
	if pg == nil || pg.Pool == nil {
		t.Fatal("requirePostgres must return a Postgres with a pool when TEST_DATABASE_URL is set")
	}
}

// TestRequireRedisURLProceedsWithTestRedisURL is the same property for Redis.
func TestRequireRedisURLProceedsWithTestRedisURL(t *testing.T) {
	const want = "redis://127.0.0.1:59999/0"
	t.Setenv("TEST_REDIS_URL", want)

	var skipped bool
	var got string
	t.Run("gated", func(t *testing.T) {
		defer func() { skipped = t.Skipped() }()
		got = requireRedisURL(t)
	})
	if skipped {
		t.Fatal("requireRedisURL must not skip when TEST_REDIS_URL is set")
	}
	if got != want {
		t.Fatalf("requireRedisURL = %q, want %q", got, want)
	}
}
