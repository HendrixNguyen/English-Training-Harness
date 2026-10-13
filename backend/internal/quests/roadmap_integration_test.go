package quests

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/onboarding"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
	"os"
	"testing"
	"time"
)

func TestIntegrationRoadmapOwnedReadOnlyProgress(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	ctx := context.Background()
	pg, err := store.NewPostgres(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pg.Close)
	if _, err = store.Migrate(ctx, pg.Migrator(), store.MigrationsFS); err != nil {
		t.Fatal(err)
	}
	var owner, other string
	for i, p := range []*string{&owner, &other} {
		gid := []string{"roadmap-read-owner", "roadmap-read-other"}[i]
		_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id=$1`, gid)
		if err = pg.Pool.QueryRow(ctx, `INSERT INTO users(email,google_id,timezone) VALUES($1,$2,'America/New_York') RETURNING id`, gid+"@example.com", gid).Scan(p); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id=$1`, gid) })
	}
	repo := NewPgRepo(pg.Pool)
	now := time.Date(2026, 3, 10, 5, 0, 0, 0, time.UTC)
	svc := NewRoadmapService(repo, func() time.Time { return now })
	if _, err = svc.Read(ctx, other); !errors.Is(err, ErrNoActiveRoadmap) {
		t.Fatal(err)
	}
	if _, err = svc.Read(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrNoActiveRoadmap) {
		t.Fatal(err)
	}
	if _, err = onboarding.NewPgRepo(pg.Pool).SaveAssessment(ctx, owner, onboarding.Assessment{CEFRLevel: "B1", Timezone: "America/New_York", NotificationTime: "07:30:00", Roadmap: integrationRoadmap()}); err != nil {
		t.Fatal(err)
	}
	if _, err = pg.Pool.Exec(ctx, `UPDATE roadmaps SET created_at='2026-03-07T15:00:00Z' WHERE user_id=$1`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err = pg.Pool.Exec(ctx, `INSERT INTO daily_progress(user_id,date,minutes_spent,is_target_met) VALUES($1,'2026-03-07',30,true),($1,'2026-03-08',12,false),($2,'2026-03-09',99,true)`, owner, other); err != nil {
		t.Fatal(err)
	}
	if _, err = pg.Pool.Exec(ctx, `UPDATE exercises SET is_completed=true WHERE roadmap_id IN(SELECT id FROM roadmaps WHERE user_id=$1) AND day_number=1 AND task_type='reading'`, owner); err != nil {
		t.Fatal(err)
	}
	var before []byte
	if err = pg.Pool.QueryRow(ctx, `SELECT roadmap_json FROM roadmaps WHERE user_id=$1`, owner).Scan(&before); err != nil {
		t.Fatal(err)
	}
	out, err := svc.Read(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if out.DayNumber != 4 || out.Today != "2026-03-10" || len(out.Modules) != 4 || len(out.Modules[3].Days) != 7 {
		t.Fatalf("%+v", out)
	}
	d := out.Modules[0].Days
	if !d[0].IsTargetMet || d[0].MinutesSpent != 30 || d[1].MinutesSpent != 12 || d[1].IsTargetMet || d[2].MinutesSpent != 0 || d[2].IsTargetMet {
		t.Fatalf("%+v", d)
	}
	completed := 0
	for _, task := range d[0].Tasks {
		if task.ID == "" || task.Title == "" || task.DurationMinutes != 10 || task.IsCompleted == nil {
			t.Fatalf("%+v", task)
		}
		if *task.IsCompleted {
			completed++
		}
	}
	if completed != 1 {
		t.Fatalf("completed=%d", completed)
	}
	// Another learner's active roadmap and progress must never enter this read.
	otherPlan := integrationRoadmap()
	otherPlan.Title = "Other learner only"
	if _, err = onboarding.NewPgRepo(pg.Pool).SaveAssessment(ctx, other, onboarding.Assessment{CEFRLevel: "B1", Timezone: "America/New_York", NotificationTime: "08:00:00", Roadmap: otherPlan}); err != nil {
		t.Fatal(err)
	}
	otherOutline, err := svc.Read(ctx, other)
	if err != nil || otherOutline.RoadmapID == out.RoadmapID || otherOutline.Title != "Other learner only" {
		t.Fatalf("other=%+v %v", otherOutline, err)
	}
	reread, err := svc.Read(ctx, owner)
	if err != nil || reread.RoadmapID != out.RoadmapID || reread.Title == otherOutline.Title {
		t.Fatalf("owner=%+v %v", reread, err)
	}
	wire, _ := json.Marshal(out)
	var decoded map[string]any
	if err = json.Unmarshal(wire, &decoded); err != nil {
		t.Fatal(err)
	}
	var after []byte
	if err = pg.Pool.QueryRow(ctx, `SELECT roadmap_json FROM roadmaps WHERE user_id=$1`, owner).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("GET mutated roadmap")
	}
	// Replacement uses only the latest active roadmap without changing it on read.
	replacement := integrationRoadmap()
	replacement.Title = "Replacement plan"
	if _, err = onboarding.NewPgRepo(pg.Pool).SaveAssessment(ctx, owner, onboarding.Assessment{CEFRLevel: "B1", Timezone: "America/New_York", NotificationTime: "07:30:00", Roadmap: replacement}); err != nil {
		t.Fatal(err)
	}
	replaced, err := svc.Read(ctx, owner)
	if err != nil || replaced.RoadmapID == out.RoadmapID || replaced.Title != "Replacement plan" {
		t.Fatalf("replacement=%+v %v", replaced, err)
	}
	for _, task := range replaced.Modules[0].Days[0].Tasks {
		if task.IsCompleted == nil || *task.IsCompleted {
			t.Fatalf("old completion leaked: %+v", task)
		}
	}
}
