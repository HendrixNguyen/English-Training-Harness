package quests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/onboarding"
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

	// Seeded the way production does it: through onboarding's repository, so
	// this test breaks if onboarding stops writing title/duration_minutes.
	if _, err := onboarding.NewPgRepo(pg.Pool).SaveAssessment(ctx, userID, onboarding.Assessment{
		CEFRLevel: "B1", TargetGoal: "integration", Timezone: "UTC", NotificationTime: "20:00:00",
		Roadmap: integrationRoadmap(),
	}); err != nil {
		t.Fatalf("SaveAssessment: %v", err)
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

	// --- Review reproduction 1: a 404 must leave both stores untouched. ---
	// Another user's real exercise (ids are UUIDs; a random string would be a
	// Postgres type error, not a 404 — see the plan).
	const otherGid = "google-quests-integration-other"
	var otherID string
	_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, otherGid)
	if err := pg.Pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, target_goal, timezone) VALUES ($1,$2,$3,$4) RETURNING id`,
		"quests-other@example.com", otherGid, "", "UTC").Scan(&otherID); err != nil {
		t.Fatalf("inserting the other user: %v", err)
	}
	t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, otherGid) })
	// Same onboarding-repo seeding as above, for the "other user" fixture; the
	// plan's Task 8 only shows the first call being replaced, but the old
	// store-package demo seed helper must be fully retired, so this one goes too.
	if _, err := onboarding.NewPgRepo(pg.Pool).SaveAssessment(ctx, otherID, onboarding.Assessment{
		CEFRLevel: "B1", TargetGoal: "integration", Timezone: "UTC", NotificationTime: "20:00:00",
		Roadmap: integrationRoadmap(),
	}); err != nil {
		t.Fatalf("SaveAssessment(other): %v", err)
	}
	otherSuite, err := svc.Daily(ctx, otherID)
	if err != nil {
		t.Fatalf("Daily(other): %v", err)
	}

	_, err = svc.RecordProgress(ctx, userID, otherSuite.Tasks[0].ID, 999)
	if !errors.Is(err, ErrExerciseNotFound) {
		t.Fatalf("progress against another user's exercise: err = %v, want ErrExerciseNotFound", err)
	}
	assertUntouched := func(t *testing.T, what string) {
		t.Helper()
		total, err := counter.Total(ctx, userID, LocalDate(now, time.UTC))
		if err != nil {
			t.Fatalf("%s: reading counter: %v", what, err)
		}
		if total != 1800 {
			t.Errorf("%s: daily:accumulated = %d, want 1800 untouched", what, total)
		}
		var m int
		var met bool
		if err := pg.Pool.QueryRow(ctx,
			`SELECT minutes_spent, is_target_met FROM daily_progress WHERE user_id = $1 AND date = $2::date`,
			userID, LocalDate(now, time.UTC)).Scan(&m, &met); err != nil {
			t.Fatalf("%s: reading daily_progress: %v", what, err)
		}
		if m != 30 || !met {
			t.Errorf("%s: daily_progress = (%d, %t), want (30, true) untouched", what, m, met)
		}
	}
	assertUntouched(t, "after the 404")
	var otherCompleted bool
	if err := pg.Pool.QueryRow(ctx, `SELECT is_completed FROM exercises WHERE id = $1`, otherSuite.Tasks[0].ID).Scan(&otherCompleted); err != nil {
		t.Fatalf("reading the other user's exercise: %v", err)
	}
	if otherCompleted {
		t.Error("the other user's exercise was marked complete by a rejected request")
	}

	// --- Review reproduction 2: an oversized report is a 400, not a 48h brick. ---
	for _, seconds := range []int64{MaxDurationSeconds + 1, 100_000_000_000_000} {
		if _, err := svc.RecordProgress(ctx, userID, suite.Tasks[1].ID, seconds); !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("duration %d: err = %v, want ErrInvalidDuration", seconds, err)
		}
	}
	assertUntouched(t, "after the oversized reports")

	// The day is still writable afterwards — the reviewer's ordinary 60s call
	// was a 500 before this fix.
	out2, err := svc.RecordProgress(ctx, userID, suite.Tasks[1].ID, 60)
	if err != nil {
		t.Fatalf("an ordinary 60s report after the rejections failed: %v", err)
	}
	if out2.DailySecondsSpent != 1860 || out2.DailyMinutesSpent != 31 || out2.NewlyMet {
		t.Errorf("out = %+v, want 1860s/31m and not newly met", out2)
	}

	again, err := svc.Daily(ctx, userID)
	if err != nil {
		t.Fatalf("second Daily: %v", err)
	}
	if again.AccumulatedSeconds != 1860 || !again.IsTargetMet || !again.Tasks[0].IsCompleted || !again.Tasks[1].IsCompleted {
		t.Errorf("second Daily = %+v, want 1860s accumulated, target met, tasks[0] and [1] completed", again)
	}

	// Monotonic: a lost counter cannot lower minutes_spent or unset
	// is_target_met. Runs last and uses the still-unconsumed Tasks[2] so it
	// does not disturb the (1800s, 30, true) state the "Review reproduction"
	// assertions above depend on. Accumulated is 1860s/31m going in (from
	// out2 above); +600s here makes it 2460s/41m before the counter is lost.
	if _, err := svc.RecordProgress(ctx, userID, suite.Tasks[2].ID, 600); err != nil {
		t.Fatalf("third RecordProgress: %v", err)
	}
	rdb.Client.Del(ctx, store.DailyAccumulatedKey(userID, now)) // the eviction / restart / FLUSHDB case
	out3, err := svc.RecordProgress(ctx, userID, suite.Tasks[2].ID, 60)
	if err != nil {
		t.Fatalf("RecordProgress after the counter was lost: %v", err)
	}
	if !out3.IsTargetMet || out3.NewlyMet {
		t.Errorf("out = %+v after the counter was lost, want IsTargetMet true (durable row) and NewlyMet false", out3)
	}
	if err := pg.Pool.QueryRow(ctx,
		`SELECT minutes_spent, is_target_met FROM daily_progress WHERE user_id = $1 AND date = $2::date`,
		userID, LocalDate(now, time.UTC)).Scan(&minutes, &met); err != nil {
		t.Fatalf("reading daily_progress: %v", err)
	}
	if minutes != 41 || !met {
		t.Errorf("daily_progress = (%d, %t) after the counter was lost, want (41, true) — never lowered", minutes, met)
	}
}

// integrationRoadmap is a valid 4x7x3 roadmap whose tasks carry the title and
// duration_minutes the §6.2 daily response exposes.
func integrationRoadmap() airouter.Roadmap {
	r := airouter.Roadmap{Title: "Integration", CEFRLevel: "B1"}
	for m := 1; m <= airouter.Modules; m++ {
		mod := airouter.Module{Week: m, Title: fmt.Sprintf("Week %d", m), Focus: "integration"}
		for d := 1; d <= airouter.DaysPerModule; d++ {
			day := airouter.Day{Title: fmt.Sprintf("Day %d", (m-1)*airouter.DaysPerModule+d)}
			for _, tt := range airouter.TaskTypes {
				day.Tasks = append(day.Tasks, airouter.Task{Type: tt, Title: "Day task: " + tt, DurationMinutes: 10, Content: json.RawMessage(`{}`)})
			}
			mod.Days = append(mod.Days, day)
		}
		r.Modules = append(r.Modules, mod)
	}
	return r
}
