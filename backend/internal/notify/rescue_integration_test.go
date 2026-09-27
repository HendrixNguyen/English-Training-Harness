package notify

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// TestIntegrationRescueCandidatesAndFlagClaim proves the candidates join
// (only users with a push subscription, with their timezone) and the SET NX
// EX once-per-day flag against real Postgres and Redis. Gated on
// TEST_DATABASE_URL and TEST_REDIS_URL — never on the production URLs.
func TestIntegrationRescueCandidatesAndFlagClaim(t *testing.T) {
	dbURL, redisURL := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_REDIS_URL")
	if dbURL == "" || redisURL == "" {
		t.Skip("TEST_DATABASE_URL/TEST_REDIS_URL unset; run `make up` and export them to run integration tests")
	}
	ctx := context.Background()

	pg, err := store.NewPostgres(ctx, dbURL)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	t.Cleanup(pg.Close)
	if _, err := store.Migrate(ctx, pg.Migrator(), store.MigrationsFS); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	rdb, err := store.NewRedis(ctx, redisURL)
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	ids := map[string]string{}
	for gid, tz := range map[string]string{"rescue-integration-a": "Asia/Ho_Chi_Minh", "rescue-integration-b": "America/New_York"} {
		_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)
		var id string
		if err := pg.Pool.QueryRow(ctx,
			`INSERT INTO users (email, google_id, target_goal, timezone) VALUES ($1,$2,$3,$4) RETURNING id`,
			gid+"@example.com", gid, "", tz).Scan(&id); err != nil {
			t.Fatalf("inserting %s: %v", gid, err)
		}
		ids[gid] = id
		g := gid
		t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, g) })
	}
	a, b := ids["rescue-integration-a"], ids["rescue-integration-b"]

	repo := NewPgRepo(pg.Pool)
	sub := Subscription{Endpoint: "https://push.example.test/rescue-integration/ep1", P256dh: "BNc5T", Auth: "aX8v"}
	if err := repo.SaveSubscription(ctx, a, sub); err != nil {
		t.Fatalf("SaveSubscription: %v", err)
	}

	// --- candidates: a (subscribed) with its timezone; never b ---
	cands, err := repo.RescueCandidates(ctx)
	if err != nil {
		t.Fatalf("RescueCandidates: %v", err)
	}
	var foundA bool
	for _, c := range cands {
		if c.UserID == b {
			t.Errorf("unsubscribed user b is a candidate: %+v", cands)
		}
		if c.UserID == a {
			foundA = true
			if c.Timezone != "Asia/Ho_Chi_Minh" {
				t.Errorf("a's timezone = %q, want Asia/Ho_Chi_Minh", c.Timezone)
			}
		}
	}
	if !foundA {
		t.Errorf("subscribed user a missing from candidates: %+v", cands)
	}

	// --- the SET NX EX flag ---
	flags := NewRedisRescueFlags(rdb)
	const day, next = "2026-09-22", "2026-09-23"
	keys := []string{store.RescueKey(a, day), store.RescueKey(a, next)}
	_ = rdb.Client.Del(ctx, keys...).Err()
	t.Cleanup(func() { _ = rdb.Client.Del(ctx, keys...).Err() })

	if ok, err := flags.Claim(ctx, a, day); err != nil || !ok {
		t.Fatalf("first Claim = %v, %v; want true", ok, err)
	}
	if ok, err := flags.Claim(ctx, a, day); err != nil || ok {
		t.Errorf("second Claim same day = %v, %v; want false", ok, err)
	}
	if ok, err := flags.Claim(ctx, a, next); err != nil || !ok {
		t.Errorf("Claim next day = %v, %v; want true", ok, err)
	}
	ttl, err := rdb.Client.TTL(ctx, store.RescueKey(a, day)).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if ttl <= 47*time.Hour || ttl > 48*time.Hour {
		t.Errorf("TTL = %v, want within (47h, 48h]", ttl)
	}
}
