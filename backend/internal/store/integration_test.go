package store

import (
	"context"
	"os"
	"reflect"
	"testing"
)

// requirePostgres skips the test unless the developer has explicitly nominated a
// disposable database in TEST_DATABASE_URL. It deliberately does NOT read
// DATABASE_URL: that is the production variable from spec §8, and these tests
// drop every table (see reset). A plain `go test ./...` must never be
// destructive, whatever the shell has exported.
func requirePostgres(t *testing.T) *Postgres {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is unset; these tests DROP every table, so they only run against a database you nominate (see backend/.env.example)")
	}
	pg, err := NewPostgres(context.Background(), url)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	t.Cleanup(pg.Close)
	return pg
}

// requireRedisURL skips the test unless TEST_REDIS_URL nominates a disposable
// Redis. Same reasoning as requirePostgres.
func requireRedisURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL is unset; run `make up` and export it to run integration tests")
	}
	return url
}

// reset drops everything every migration creates (newest first), plus the
// bookkeeping table, so each test starts from an empty database.
func reset(t *testing.T, pg *Postgres) {
	t.Helper()
	// Belt and braces: reset is the destructive step. Even if a future test
	// reaches it by another path, it must not run against a database the
	// developer did not nominate as disposable.
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Fatal("reset called without TEST_DATABASE_URL; refusing to drop tables")
	}
	ctx := context.Background()
	// 0003_pet_verdict_dates.up.sql and 0004_pet_shields.up.sql only add
	// columns to pet_states, so 0001_init's DROP TABLE (which the loop below
	// still runs) removes them too — unlike 0001/0002's statements, 0003's
	// and 0004's ALTER TABLE have no "IF EXISTS" on the table itself, so
	// running them here would fail whenever reset() is called against an
	// already-empty database (its first use in a test binary, or right after
	// a sibling test's own cleanup).
	for _, name := range []string{"migrations/0002_google_sync.down.sql", "migrations/0001_init.down.sql"} {
		down, err := MigrationsFS.ReadFile(name)
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if _, err := pg.Pool.Exec(ctx, string(down)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, err := pg.Pool.Exec(ctx, `DROP TABLE IF EXISTS schema_migrations`); err != nil {
		t.Fatalf("dropping schema_migrations: %v", err)
	}
}

func TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent(t *testing.T) {
	pg := requirePostgres(t)
	reset(t, pg)
	t.Cleanup(func() { reset(t, pg) })

	ctx := context.Background()

	first, err := Migrate(ctx, pg.Migrator(), MigrationsFS)
	if err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if want := []string{"0001_init", "0002_google_sync", "0003_pet_verdict_dates", "0004_pet_shields"}; !reflect.DeepEqual(first, want) {
		t.Fatalf("first run applied %v, want %v", first, want)
	}

	second, err := Migrate(ctx, pg.Migrator(), MigrationsFS)
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if len(second) != 0 {
		t.Errorf("second run applied %v, want nothing", second)
	}

	for _, table := range []string{"users", "push_subscriptions", "pet_states", "daily_progress", "roadmaps", "exercises", "google_sync"} {
		var exists bool
		err := pg.Pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)`, table).Scan(&exists)
		if err != nil {
			t.Fatalf("checking %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s was not created", table)
		}
	}
}

// TestIntegrationConcurrentMigrateDoesNotRace reproduces the scenario that
// bites in CI: two packages' integration tests (e.g. internal/store and
// internal/auth) each call Migrate independently against one shared
// TEST_DATABASE_URL when `go test ./...` runs their test binaries in
// parallel. Without the PgMigrator.Lock serialization, this used to fail
// intermittently with "duplicate key value violates unique constraint
// pg_type_typname_nsp_index", because CREATE TABLE/TYPE IF NOT EXISTS is a
// check-then-act, not atomic, under concurrency.
func TestIntegrationConcurrentMigrateDoesNotRace(t *testing.T) {
	pg := requirePostgres(t)
	reset(t, pg)
	t.Cleanup(func() { reset(t, pg) })

	ctx := context.Background()
	// Each goroutine gets its own Postgres (own pool/connection), the way two
	// independent packages' test binaries would.
	dbURL := os.Getenv("TEST_DATABASE_URL")
	const n = 8
	errs := make(chan error, n)
	applied := make(chan []string, n)

	for i := 0; i < n; i++ {
		go func() {
			p, err := NewPostgres(ctx, dbURL)
			if err != nil {
				errs <- err
				applied <- nil
				return
			}
			defer p.Close()
			got, err := Migrate(ctx, p.Migrator(), MigrationsFS)
			errs <- err
			applied <- got
		}()
	}

	total := 0
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent Migrate() call %d: %v", i, err)
		}
		total += len(<-applied)
	}
	const versions = 4 // 0001_init, 0002_google_sync, 0003_pet_verdict_dates, 0004_pet_shields
	if total != versions {
		t.Errorf("migrations were applied %d times across %d concurrent callers, want exactly %d", total, n, versions)
	}
}

func TestIntegrationPetStatesRejectsASecondRowForTheSameUser(t *testing.T) {
	pg := requirePostgres(t)
	reset(t, pg)
	t.Cleanup(func() { reset(t, pg) })

	ctx := context.Background()
	if _, err := Migrate(ctx, pg.Migrator(), MigrationsFS); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var userID string
	err := pg.Pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, target_goal) VALUES ($1, $2, $3) RETURNING id`,
		"a@example.com", "google-1", "").Scan(&userID)
	if err != nil {
		t.Fatalf("inserting user: %v", err)
	}

	if _, err := pg.Pool.Exec(ctx, `INSERT INTO pet_states (user_id) VALUES ($1)`, userID); err != nil {
		t.Fatalf("first pet_states insert: %v", err)
	}
	if _, err := pg.Pool.Exec(ctx, `INSERT INTO pet_states (user_id) VALUES ($1)`, userID); err == nil {
		t.Fatal("second pet_states insert succeeded; user_id is not UNIQUE")
	}
}

func TestIntegrationRedisRoundTrip(t *testing.T) {
	url := requireRedisURL(t)
	rdb, err := NewRedis(context.Background(), url)
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	if err := rdb.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	key := SessionKey("integration-test-user")
	t.Cleanup(func() { rdb.Client.Del(ctx, key) })

	if err := rdb.Client.Set(ctx, key, "token", SessionTTL).Err(); err != nil {
		t.Fatalf("Set: %v", err)
	}
	ttl, err := rdb.Client.TTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if ttl <= 0 || ttl > SessionTTL {
		t.Errorf("TTL = %v, want (0, %v]", ttl, SessionTTL)
	}
}
