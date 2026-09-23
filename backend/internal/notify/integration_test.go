package notify

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// TestIntegrationScheduleAndSubscriptionRoundTrip proves the ZSET round trip
// (ZADD → ZRANGEBYSCORE → overwrite → ZREM) and the push_subscriptions
// dedupe against real Redis and Postgres. Gated on TEST_DATABASE_URL and
// TEST_REDIS_URL like internal/quests — never on the production URLs.
func TestIntegrationScheduleAndSubscriptionRoundTrip(t *testing.T) {
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

	// Two users; the second steals the first one's browser endpoint later.
	ids := map[string]string{}
	for _, gid := range []string{"notify-integration-a", "notify-integration-b"} {
		_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)
		var id string
		if err := pg.Pool.QueryRow(ctx,
			`INSERT INTO users (email, google_id, target_goal, timezone) VALUES ($1,$2,$3,$4) RETURNING id`,
			gid+"@example.com", gid, "", "UTC").Scan(&id); err != nil {
			t.Fatalf("inserting %s: %v", gid, err)
		}
		ids[gid] = id
		g := gid
		t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, g) })
	}
	a, b := ids["notify-integration-a"], ids["notify-integration-b"]

	repo := NewPgRepo(pg.Pool)

	// --- preferences ---
	if err := repo.UpdatePreferences(ctx, a, "07:30:00", "Asia/Ho_Chi_Minh"); err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}
	prefs, err := repo.Preferences(ctx, a)
	if err != nil || prefs.NotificationTime != "07:30:00" || prefs.Timezone != "Asia/Ho_Chi_Minh" {
		t.Fatalf("Preferences = %+v, %v", prefs, err)
	}
	if err := repo.UpdatePreferences(ctx, a, "08:00:00", ""); err != nil {
		t.Fatal(err)
	}
	if prefs, _ = repo.Preferences(ctx, a); prefs.Timezone != "Asia/Ho_Chi_Minh" || prefs.NotificationTime != "08:00:00" {
		t.Errorf("empty timezone must leave it unchanged: %+v", prefs)
	}
	if err := repo.UpdatePreferences(ctx, "00000000-0000-0000-0000-000000000000", "08:00:00", ""); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("unknown user: err = %v, want ErrUserNotFound", err)
	}

	// --- subscriptions: same endpoint twice → one row; other user → moves ---
	sub := Subscription{Endpoint: "https://push.example.test/notify-integration/ep1", P256dh: "BNc5T", Auth: "aX8v"}
	for i := 0; i < 2; i++ {
		if err := repo.SaveSubscription(ctx, a, sub); err != nil {
			t.Fatalf("SaveSubscription #%d: %v", i+1, err)
		}
	}
	subsA, err := repo.Subscriptions(ctx, a)
	if err != nil || len(subsA) != 1 || subsA[0].Endpoint != sub.Endpoint || subsA[0].ID == "" {
		t.Fatalf("Subscriptions(a) = %+v, %v; want exactly one", subsA, err)
	}
	if err := repo.SaveSubscription(ctx, b, sub); err != nil {
		t.Fatal(err)
	}
	subsA, _ = repo.Subscriptions(ctx, a)
	subsB, _ := repo.Subscriptions(ctx, b)
	if len(subsA) != 0 || len(subsB) != 1 {
		t.Errorf("after b subscribes with a's endpoint: a=%d b=%d, want 0/1", len(subsA), len(subsB))
	}
	if err := repo.DeleteSubscription(ctx, subsB[0].ID); err != nil {
		t.Fatal(err)
	}
	if subsB, _ = repo.Subscriptions(ctx, b); len(subsB) != 0 {
		t.Errorf("after delete: %d subscriptions, want 0", len(subsB))
	}

	// --- the ZSET ---
	q := NewRedisQueue(rdb)
	t.Cleanup(func() { _ = q.Remove(ctx, a); _ = q.Remove(ctx, b) })
	_ = q.Remove(ctx, a)
	_ = q.Remove(ctx, b)

	now := time.Now().Truncate(time.Second)
	if err := q.Schedule(ctx, a, now.Add(time.Hour)); err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	if err := q.Schedule(ctx, b, now.Add(-time.Minute)); err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	due, err := q.Due(ctx, now, 100)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if !contains(due, b) || contains(due, a) {
		t.Errorf("Due(now) = %v; want b (past) and not a (in an hour)", due)
	}
	// ZADD on an existing member overwrites its score — the re-slot.
	if err := q.Schedule(ctx, b, now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if due, _ = q.Due(ctx, now, 100); contains(due, b) {
		t.Errorf("after re-slot b is still due: %v", due)
	}
	if due, _ = q.Due(ctx, now.Add(3*time.Hour), 100); !contains(due, a) || !contains(due, b) {
		t.Errorf("Due(now+3h) = %v; want both", due)
	}
	n := 0
	for _, m := range due {
		if m == b {
			n++
		}
	}
	if n != 1 {
		t.Errorf("b appears %d times in the ZSET, want 1 (ZADD must not duplicate)", n)
	}
	if err := q.Remove(ctx, a); err != nil {
		t.Fatal(err)
	}
	if due, _ = q.Due(ctx, now.Add(3*time.Hour), 100); contains(due, a) {
		t.Errorf("after Remove a is still there: %v", due)
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
