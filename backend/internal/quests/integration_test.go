package quests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// TestIntegration* names are counted by CI's backend-integration job, which
// exports TEST_DATABASE_URL/TEST_REDIS_URL and fails on any --- SKIP, so this
// test must pass there. Locally it skips without them — `go test ./...` stays
// green with no services. It deliberately does NOT read DATABASE_URL/REDIS_URL:
// those are the production variables (spec §9), and internal/store's tests
// drop every table in the database they are pointed at (see
// internal/store/integration_test.go and internal/auth/integration_test.go for
// the same convention). Run with -p 1 (make test-integration): all packages
// share the one database.
func TestIntegrationDailyAndProgressAgainstRealServices(t *testing.T) {
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

	const gid = "google-quests-integration"
	var userID string
	_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)
	if err := pg.Pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, target_goal, timezone) VALUES ($1,$2,$3,$4) RETURNING id`,
		"quests@example.com", gid, "", "UTC").Scan(&userID); err != nil {
		t.Fatalf("inserting user: %v", err)
	}
	t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid) })

	if _, err := store.SeedDemoRoadmap(ctx, pg.Pool, userID); err != nil {
		t.Fatalf("SeedDemoRoadmap: %v", err)
	}

	now := time.Now().UTC()
	repo := NewPgRepo(pg.Pool)
	counter := NewRedisCounter(rdb)
	t.Cleanup(func() {
		rdb.Client.Del(ctx, store.DailyAccumulatedKey(userID, now))
	})

	svc := NewService(counter, repo, repo, NopPet{}, func() time.Time { return now })

	suite, err := svc.Daily(ctx, userID)
	if err != nil {
		t.Fatalf("Daily: %v", err)
	}
	if suite.Date != LocalDate(now, time.UTC) {
		t.Errorf("Date = %q, want %q", suite.Date, LocalDate(now, time.UTC))
	}
	if suite.DayNumber != 1 {
		t.Errorf("DayNumber = %d, want 1 on a freshly seeded roadmap", suite.DayNumber)
	}
	if suite.TotalMinutesRequired != 30 || suite.AccumulatedSeconds != 0 || suite.IsTargetMet {
		t.Errorf("suite = %+v, want 30 required, 0 accumulated, target unmet", suite)
	}
	if len(suite.Tasks) != 3 {
		t.Fatalf("len(Tasks) = %d, want 3", len(suite.Tasks))
	}
	if suite.Tasks[0].Title == "" || suite.Tasks[0].DurationMinutes != 10 {
		t.Errorf("task = %+v, want the seed's title and 10-minute duration", suite.Tasks[0])
	}

	out, err := svc.RecordProgress(ctx, userID, suite.Tasks[0].ID, 1800)
	if err != nil {
		t.Fatalf("RecordProgress: %v", err)
	}
	if out.DailySecondsSpent != 1800 || out.DailyMinutesSpent != 30 || !out.IsTargetMet || !out.NewlyMet {
		t.Errorf("out = %+v, want 1800s/30m and the target newly met", out)
	}
	if out.PetHealth != 100 || out.StreakCount != 0 {
		t.Errorf("pet = (%d, %d), want NopPet's §3.2 defaults (100, 0)", out.PetHealth, out.StreakCount)
	}

	var minutes int
	var met bool
	if err := pg.Pool.QueryRow(ctx,
		`SELECT minutes_spent, is_target_met FROM daily_progress WHERE user_id = $1 AND date = $2::date`,
		userID, LocalDate(now, time.UTC)).Scan(&minutes, &met); err != nil {
		t.Fatalf("reading daily_progress: %v", err)
	}
	if minutes != 30 || !met {
		t.Errorf("daily_progress = (%d, %t), want (30, true)", minutes, met)
	}

	var completed bool
	if err := pg.Pool.QueryRow(ctx, `SELECT is_completed FROM exercises WHERE id = $1`, suite.Tasks[0].ID).Scan(&completed); err != nil {
		t.Fatalf("reading exercise: %v", err)
	}
	if !completed {
		t.Error("exercises.is_completed = false after progress was recorded against it")
	}

	again, err := svc.Daily(ctx, userID)
	if err != nil {
		t.Fatalf("second Daily: %v", err)
	}
	if again.AccumulatedSeconds != 1800 || !again.IsTargetMet || !again.Tasks[0].IsCompleted {
		t.Errorf("second Daily = %+v, want 1800s accumulated, target met, task[0] completed", again)
	}
}
