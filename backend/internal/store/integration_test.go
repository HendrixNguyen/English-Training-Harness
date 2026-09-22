package store

import (
	"context"
	"os"
	"testing"
)

// requirePostgres skips the test when no database is configured. This is
// deliberate: the default `go test ./...` run must not need live services.
func requirePostgres(t *testing.T) *Postgres {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is unset; run `make up` and export it to run integration tests")
	}
	pg, err := NewPostgres(context.Background(), url)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	t.Cleanup(pg.Close)
	return pg
}

// reset drops everything 0001 creates plus the bookkeeping table, so each test
// starts from an empty database.
func reset(t *testing.T, pg *Postgres) {
	t.Helper()
	down, err := MigrationsFS.ReadFile("migrations/0001_init.down.sql")
	if err != nil {
		t.Fatalf("reading down migration: %v", err)
	}
	ctx := context.Background()
	if _, err := pg.Pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("down migration: %v", err)
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
	if len(first) != 1 || first[0] != "0001_init" {
		t.Fatalf("first run applied %v, want [0001_init]", first)
	}

	second, err := Migrate(ctx, pg.Migrator(), MigrationsFS)
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if len(second) != 0 {
		t.Errorf("second run applied %v, want nothing", second)
	}

	for _, table := range []string{"users", "push_subscriptions", "pet_states", "daily_progress", "roadmaps", "exercises"} {
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
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL is unset; run `make up` and export it to run integration tests")
	}
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
