package google

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/secrets"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// TestIntegrationSyncStateIsOneRowPerUser proves the google_sync upsert
// (migration 0002) and the read-only lookups against a real Postgres. Gated on
// TEST_DATABASE_URL like internal/store — never on the production DATABASE_URL.
func TestIntegrationSyncStateIsOneRowPerUser(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL unset; run `make up` and export it to run integration tests")
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

	box, err := secrets.New(bytes.Repeat([]byte{3}, secrets.KeyBytes))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal("1//refresh")
	if err != nil {
		t.Fatal(err)
	}

	const gid = "google-sync-integration"
	var userID string
	_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)
	if err := pg.Pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, target_goal, timezone, notification_time, google_refresh_token)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		"google@example.com", gid, "", "Asia/Ho_Chi_Minh", "07:30:00", sealed).Scan(&userID); err != nil {
		t.Fatalf("inserting user: %v", err)
	}
	t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid) })

	repo := NewPgRepo(pg.Pool)

	// Profile renders TIME as HH:MM:SS.
	prof, err := repo.Profile(ctx, userID)
	if err != nil {
		t.Fatalf("Profile: %v", err)
	}
	if prof.NotificationTime != "07:30:00" || prof.Timezone != "Asia/Ho_Chi_Minh" {
		t.Errorf("profile = %+v", prof)
	}

	// The refresh-token seam opens what auth sealed.
	tok, err := NewPgRefreshTokenSource(pg.Pool, box).RefreshToken(ctx, userID)
	if err != nil || tok != "1//refresh" {
		t.Errorf("RefreshToken = %q, %v; want the opened plaintext", tok, err)
	}
	// A pre-encryption row (plaintext, no v1: prefix) is "no token on file":
	// Service maps it to reauth_required and the next sign-in stores a sealed one.
	if _, err := pg.Pool.Exec(ctx, `UPDATE users SET google_refresh_token = '1//legacy-plaintext' WHERE id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPgRefreshTokenSource(pg.Pool, box).RefreshToken(ctx, userID); !errors.Is(err, ErrNoRefreshToken) {
		t.Errorf("RefreshToken on a legacy plaintext row: err = %v, want ErrNoRefreshToken", err)
	}

	// No roadmap yet.
	if _, err := repo.ActiveRoadmap(ctx, userID); !errors.Is(err, ErrNoActiveRoadmap) {
		t.Errorf("ActiveRoadmap without a roadmap: err = %v, want ErrNoActiveRoadmap", err)
	}
	if _, err := repo.SyncState(ctx, userID); !errors.Is(err, ErrNoSyncState) {
		t.Errorf("SyncState before any sync: err = %v, want ErrNoSyncState", err)
	}

	// A roadmap with two days; titles come from content_json->>'title' — the
	// key onboarding writes and quests' toTask reads.
	var roadmapID string
	if err := pg.Pool.QueryRow(ctx,
		`INSERT INTO roadmaps (user_id, roadmap_json, is_active) VALUES ($1, '{}'::jsonb, TRUE) RETURNING id`, userID).Scan(&roadmapID); err != nil {
		t.Fatalf("inserting roadmap: %v", err)
	}
	for _, row := range []struct {
		day   int
		typ   string
		title string
	}{{1, "vocabulary", "Greetings"}, {1, "reading", "Short story"}, {2, "practice", "Order a coffee"}} {
		if _, err := pg.Pool.Exec(ctx,
			`INSERT INTO exercises (roadmap_id, day_number, task_type, content_json) VALUES ($1,$2,$3::task_category,$4::jsonb)`,
			roadmapID, row.day, row.typ, `{"title":"`+row.title+`","duration_minutes":10}`); err != nil {
			t.Fatalf("inserting exercise: %v", err)
		}
	}
	rm, err := repo.ActiveRoadmap(ctx, userID)
	if err != nil || rm.ID != roadmapID {
		t.Fatalf("ActiveRoadmap = %+v, %v", rm, err)
	}
	days, err := repo.DayTitles(ctx, roadmapID)
	if err != nil {
		t.Fatalf("DayTitles: %v", err)
	}
	if len(days) != 2 || days[0].Day != 1 || len(days[0].Titles) != 2 || days[1].Day != 2 || days[1].Titles[0] != "Order a coffee" {
		t.Errorf("DayTitles = %+v", days)
	}

	// Upsert twice → one row, latest values.
	first := SyncState{UserID: userID, CalendarEventID: "evt_1"}
	if err := repo.SaveSyncState(ctx, first); err != nil {
		t.Fatalf("SaveSyncState #1: %v", err)
	}
	second := SyncState{UserID: userID, CalendarEventID: "evt_1", TasklistID: "list_1", RoadmapID: roadmapID, TasksCreatedCount: 2}
	if err := repo.SaveSyncState(ctx, second); err != nil {
		t.Fatalf("SaveSyncState #2: %v", err)
	}
	got, err := repo.SyncState(ctx, userID)
	if err != nil {
		t.Fatalf("SyncState: %v", err)
	}
	if got != second {
		t.Errorf("SyncState = %+v, want %+v", got, second)
	}
	var n int
	if err := pg.Pool.QueryRow(ctx, `SELECT count(*) FROM google_sync WHERE user_id = $1`, userID).Scan(&n); err != nil || n != 1 {
		t.Errorf("google_sync rows = %d (%v), want 1", n, err)
	}
}
