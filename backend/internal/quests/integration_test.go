package quests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
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

	if err := repo.MarkTargetMet(ctx, userID, "1999-01-01"); !errors.Is(err, ErrNoProgressRow) {
		t.Errorf("MarkTargetMet for a date with no row: err = %v, want ErrNoProgressRow", err)
	}

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

	after, err := svc.Daily(ctx, userID)
	if err != nil {
		t.Fatalf("Daily after the counter was lost: %v", err)
	}
	if after.AccumulatedSeconds != 60 || !after.IsTargetMet {
		t.Errorf("Daily after the counter was lost = accumulated %d, is_target_met %t; want 60 (the one post-loss report) and true (durable row)", after.AccumulatedSeconds, after.IsTargetMet)
	}
}

// TestIntegrationRoadmapOutlineJoinsDailyProgress proves the storage reads
// (ActiveRoadmapDoc, ProgressBetween) and Service.Roadmap against a real
// Postgres, seeded the way production seeds it (onboarding.SaveAssessment).
func TestIntegrationRoadmapOutlineJoinsDailyProgress(t *testing.T) {
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

	const gid = "google-roadmap-integration"
	var userID string
	_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)
	if err := pg.Pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, target_goal, timezone) VALUES ($1,$2,$3,$4) RETURNING id`,
		"roadmap@example.com", gid, "", "UTC").Scan(&userID); err != nil {
		t.Fatalf("inserting user: %v", err)
	}
	t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid) })

	if _, err := onboarding.NewPgRepo(pg.Pool).SaveAssessment(ctx, userID, onboarding.Assessment{
		CEFRLevel: "B1", TargetGoal: "integration", Timezone: "UTC", NotificationTime: "20:00:00",
		Roadmap: integrationRoadmap(),
	}); err != nil {
		t.Fatalf("SaveAssessment: %v", err)
	}

	repo := NewPgRepo(pg.Pool)
	doc, err := repo.ActiveRoadmapDoc(ctx, userID)
	if err != nil {
		t.Fatalf("ActiveRoadmapDoc: %v", err)
	}
	if doc.ID == "" {
		t.Fatalf("doc.ID is empty")
	}
	if time.Since(doc.CreatedAt) > time.Minute {
		t.Errorf("doc.CreatedAt = %v, want within the last minute", doc.CreatedAt)
	}
	if !json.Valid(doc.JSON) {
		t.Fatalf("doc.JSON is not valid JSON: %s", doc.JSON)
	}
	if !strings.Contains(string(doc.JSON), `"modules"`) {
		t.Errorf("doc.JSON does not contain modules: %s", doc.JSON)
	}

	loc := Location("UTC")
	d1 := DayDate(doc.CreatedAt, 1, loc)
	d2 := DayDate(doc.CreatedAt, 2, loc)

	if _, err := repo.Upsert(ctx, userID, d1, 30); err != nil {
		t.Fatalf("Upsert(d1): %v", err)
	}
	if err := repo.MarkTargetMet(ctx, userID, d1); err != nil {
		t.Fatalf("MarkTargetMet(d1): %v", err)
	}
	if _, err := repo.Upsert(ctx, userID, d2, 12); err != nil {
		t.Fatalf("Upsert(d2): %v", err)
	}
	if _, err := repo.Upsert(ctx, userID, "1999-01-01", 5); err != nil {
		t.Fatalf("Upsert(out of range): %v", err)
	}

	rows, err := repo.ProgressBetween(ctx, userID, d1, DayDate(doc.CreatedAt, 28, loc))
	if err != nil {
		t.Fatalf("ProgressBetween: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2 (rows = %+v)", len(rows), rows)
	}
	if rows[d1] != (DayProgress{MinutesSpent: 30, IsTargetMet: true}) {
		t.Errorf("rows[d1] = %+v", rows[d1])
	}
	if rows[d2] != (DayProgress{MinutesSpent: 12, IsTargetMet: false}) {
		t.Errorf("rows[d2] = %+v", rows[d2])
	}
	if _, ok := rows["1999-01-01"]; ok {
		t.Errorf("rows contains the out-of-range date 1999-01-01")
	}

	svc := NewService(NewRedisCounter(rdb), repo, repo, NopPet{}, func() time.Time { return time.Now().UTC() })
	out, err := svc.Roadmap(ctx, userID)
	if err != nil {
		t.Fatalf("Roadmap: %v", err)
	}
	if out.DayNumber != 1 {
		t.Errorf("DayNumber = %d, want 1", out.DayNumber)
	}
	if len(out.Modules) != 4 {
		t.Fatalf("len(Modules) = %d, want 4", len(out.Modules))
	}
	for i, m := range out.Modules {
		if len(m.Days) != 7 {
			t.Errorf("Modules[%d] has %d days, want 7", i, len(m.Days))
		}
	}
	day1 := out.Modules[0].Days[0]
	if day1.Date != d1 || day1.MinutesSpent != 30 || !day1.IsTargetMet || day1.Title != "Day 1" {
		t.Errorf("Modules[0].Days[0] = %+v, want date=%s minutes=30 met=true title=Day 1", day1, d1)
	}
	day2 := out.Modules[0].Days[1]
	if day2.MinutesSpent != 12 || day2.IsTargetMet {
		t.Errorf("Modules[0].Days[1] = %+v, want minutes=12 met=false", day2)
	}
	day3 := out.Modules[0].Days[2]
	if day3.MinutesSpent != 0 || day3.IsTargetMet {
		t.Errorf("Modules[0].Days[2] = %+v, want minutes=0 met=false", day3)
	}
	for _, m := range out.Modules {
		for _, d := range m.Days {
			if len(d.Tasks) != 3 {
				t.Fatalf("day %d has %d tasks, want 3", d.DayNumber, len(d.Tasks))
			}
			for _, task := range d.Tasks {
				if task.DurationMinutes != 10 {
					t.Errorf("day %d task %+v duration = %d, want 10", d.DayNumber, task, task.DurationMinutes)
				}
				if !strings.HasPrefix(task.Title, "Day task: ") {
					t.Errorf("day %d task %+v title does not start with 'Day task: '", d.DayNumber, task)
				}
			}
		}
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
