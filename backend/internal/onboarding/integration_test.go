package onboarding

import (
	"context"
	"os"
	"testing"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// Gated on TEST_DATABASE_URL, never the production DATABASE_URL (spec §9);
// internal/store's tests drop every table in the database they point at. CI
// exports TEST_* and fails on --- SKIP; run with -p 1 (make test-integration).
func TestIntegrationSaveAssessmentPersists84ExercisesAndDeactivatesPrevious(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is unset; run `make up` and export it to run integration tests")
	}
	ctx := context.Background()
	pg, err := store.NewPostgres(ctx, url)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	t.Cleanup(pg.Close)
	if _, err := store.Migrate(ctx, pg.Migrator(), store.MigrationsFS); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	const gid = "google-onboarding-integration"
	_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)
	var userID string
	if err := pg.Pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, target_goal) VALUES ($1, $2, '') RETURNING id`,
		"onboarding@example.com", gid).Scan(&userID); err != nil {
		t.Fatalf("inserting user: %v", err)
	}
	t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid) })

	repo := NewPgRepo(pg.Pool)
	if _, ok, err := repo.ActiveRoadmapID(ctx, userID); err != nil || ok {
		t.Fatalf("ActiveRoadmapID before = ok:%t err:%v, want none", ok, err)
	}

	first, err := repo.SaveAssessment(ctx, userID, Assessment{CEFRLevel: "B1", TargetGoal: "IELTS 7.0", Timezone: "Asia/Ho_Chi_Minh", NotificationTime: "20:00:00", Roadmap: fixtureRoadmap()})
	if err != nil {
		t.Fatalf("first SaveAssessment: %v", err)
	}
	second, err := repo.SaveAssessment(ctx, userID, Assessment{CEFRLevel: "B2", TargetGoal: "TOEFL 100", Timezone: "UTC", NotificationTime: "07:30:00", Roadmap: fixtureRoadmap()})
	if err != nil {
		t.Fatalf("second SaveAssessment: %v", err)
	}
	if first == second {
		t.Fatal("second save returned the first roadmap id")
	}

	id, ok, err := repo.ActiveRoadmapID(ctx, userID)
	if err != nil || !ok || id != second {
		t.Errorf("ActiveRoadmapID = %q ok:%t err:%v, want the second roadmap", id, ok, err)
	}
	var active, total int
	if err := pg.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE is_active), count(*) FROM roadmaps WHERE user_id = $1`, userID).Scan(&active, &total); err != nil {
		t.Fatalf("counting roadmaps: %v", err)
	}
	if active != 1 || total != 2 {
		t.Errorf("roadmaps: active=%d total=%d, want 1 of 2", active, total)
	}

	var exercises, days, vocab int
	if err := pg.Pool.QueryRow(ctx,
		`SELECT count(*), count(DISTINCT day_number), count(*) FILTER (WHERE task_type = 'vocabulary') FROM exercises WHERE roadmap_id = $1`, second).Scan(&exercises, &days, &vocab); err != nil {
		t.Fatalf("counting exercises: %v", err)
	}
	if exercises != 84 || days != 28 || vocab != 28 {
		t.Errorf("exercises=%d days=%d vocabulary=%d, want 84/28/28", exercises, days, vocab)
	}
	var title string
	var minutes int
	if err := pg.Pool.QueryRow(ctx,
		`SELECT content_json->>'title', (content_json->>'duration_minutes')::int FROM exercises WHERE roadmap_id = $1 AND day_number = 1 AND task_type = 'reading'`, second).Scan(&title, &minutes); err != nil {
		t.Fatalf("reading content_json: %v", err)
	}
	if title != "reading task" || minutes != 10 {
		t.Errorf("content_json title/duration = %q/%d — quests' toTask reads exactly these keys", title, minutes)
	}

	var cefr, goal, tz, notif string
	if err := pg.Pool.QueryRow(ctx,
		`SELECT cefr_current::text, target_goal, timezone, notification_time::text FROM users WHERE id = $1`, userID).Scan(&cefr, &goal, &tz, &notif); err != nil {
		t.Fatalf("reading user: %v", err)
	}
	if cefr != "B2" || goal != "TOEFL 100" || tz != "UTC" || notif != "07:30:00" {
		t.Errorf("user = %s/%s/%s/%s, want the second assessment's values", cefr, goal, tz, notif)
	}
	p, err := repo.Profile(ctx, userID)
	if err != nil || p.CEFRCurrent != "B2" {
		t.Errorf("Profile = %+v, %v", p, err)
	}
}
