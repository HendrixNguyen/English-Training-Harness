package auth

import (
	"context"
	"os"
	"testing"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// These are the only tests in this package that need a database, and they skip
// without TEST_DATABASE_URL — `go test ./...` stays green with no services.
// This deliberately does NOT read DATABASE_URL: that is the production
// variable from spec §8, and CI's backend-integration job only exports
// TEST_DATABASE_URL/TEST_REDIS_URL (see internal/store/integration_test.go
// for the same convention).
func TestIntegrationUpsertCreatesThenPreservesTheLearnerState(t *testing.T) {
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

	repo := NewPgUserRepo(pg.Pool)
	const gid = "google-integration-1"
	t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid) })
	_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)

	first, err := repo.UpsertByGoogleID(ctx, gid, "a@example.com", "A Person", "rt-1")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if first.ID == "" {
		t.Fatal("first upsert returned no id")
	}
	if first.CEFRCurrent != "A1" {
		t.Errorf("CEFRCurrent = %q, want the §3.2 default A1", first.CEFRCurrent)
	}

	// The learner progresses and sets a goal.
	if _, err := pg.Pool.Exec(ctx,
		`UPDATE users SET cefr_current = 'B2', target_goal = 'IELTS 7.0' WHERE id = $1`, first.ID); err != nil {
		t.Fatalf("simulating onboarding: %v", err)
	}

	// Re-login with no refresh token (Google omits it on silent consent).
	second, err := repo.UpsertByGoogleID(ctx, gid, "a@example.com", "Renamed Person", "")
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("re-login created a new row: %q vs %q", second.ID, first.ID)
	}
	if second.CEFRCurrent != "B2" {
		t.Errorf("CEFRCurrent = %q, want B2 preserved", second.CEFRCurrent)
	}
	if second.FullName != "Renamed Person" {
		t.Errorf("FullName = %q, want the Google value refreshed", second.FullName)
	}

	var goal, refresh string
	if err := pg.Pool.QueryRow(ctx,
		`SELECT target_goal, google_refresh_token FROM users WHERE id = $1`, first.ID).Scan(&goal, &refresh); err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if goal != "IELTS 7.0" {
		t.Errorf("target_goal = %q, want it preserved across re-login", goal)
	}
	if refresh != "rt-1" {
		t.Errorf("google_refresh_token = %q, want the stored token kept when Google sends none", refresh)
	}
}
