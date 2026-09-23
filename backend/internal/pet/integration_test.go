package pet

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// Gated on TEST_DATABASE_URL, never the production DATABASE_URL (spec §9):
// internal/store's tests drop every table in the database they are pointed
// at. CI exports TEST_* and fails on --- SKIP; run with -p 1 (make
// test-integration) because all packages share the one database.
func TestIntegrationEnsureCreatesExactlyOnePetRow(t *testing.T) {
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

	const gid = "google-pet-integration"
	_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)
	var userID string
	if err := pg.Pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, target_goal, timezone) VALUES ($1, $2, '', 'Asia/Ho_Chi_Minh') RETURNING id`,
		"pet@example.com", gid).Scan(&userID); err != nil {
		t.Fatalf("inserting user: %v", err)
	}
	t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid) })

	repo := NewPgRepo(pg.Pool)
	svc := NewService(repo, newFakeChallenges(), newFakeStudy(), time.Now)

	// Eight concurrent first calls: UNIQUE(user_id) + ON CONFLICT DO NOTHING
	// must yield one row and zero errors.
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.Ensure(ctx, userID); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent Ensure: %v", err)
	}

	var n int
	if err := pg.Pool.QueryRow(ctx, `SELECT count(*) FROM pet_states WHERE user_id = $1`, userID).Scan(&n); err != nil {
		t.Fatalf("counting rows: %v", err)
	}
	if n != 1 {
		t.Fatalf("pet_states rows = %d, want 1", n)
	}

	st, err := repo.Get(ctx, userID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if st.PlantName != "My Green Buddy" || st.HealthPoints != 100 || st.Stage != StageSprout || st.CurrentStreak != 0 || st.LastPracticedAt != nil {
		t.Errorf("fresh row = %+v, want the §3.2 defaults", st)
	}

	// Round-trip the write path and the enum cast.
	if err := svc.OnTargetMet(ctx, userID, "2026-09-22"); err != nil {
		t.Fatalf("OnTargetMet: %v", err)
	}
	st, _ = repo.Get(ctx, userID)
	if st.HealthPoints != 100 || st.CurrentStreak != 1 || st.LastPracticedAt == nil {
		t.Errorf("after target met = %+v, want 100/1 with last_practiced_at set", st)
	}
	for i := 0; i < 4; i++ {
		if err := repo.Save(ctx, userID, ApplyMiss(st, time.Now(), "2026-09-22")); err != nil {
			t.Fatalf("Save miss %d: %v", i+1, err)
		}
		st, _ = repo.Get(ctx, userID)
	}
	if st.HealthPoints != 0 || st.Stage != StageWilted {
		t.Errorf("after four misses = %+v, want 0/wilted — the pet_stage cast must accept 'wilted'", st)
	}

	zones, err := repo.Timezones(ctx)
	if err != nil {
		t.Fatalf("Timezones: %v", err)
	}
	found := false
	for _, z := range zones {
		found = found || z == "Asia/Ho_Chi_Minh"
	}
	if !found {
		t.Errorf("Timezones = %v, want Asia/Ho_Chi_Minh included", zones)
	}
	cands, err := repo.SweepCandidates(ctx, []string{"Asia/Ho_Chi_Minh"})
	if err != nil {
		t.Fatalf("SweepCandidates: %v", err)
	}
	seen := false
	for _, c := range cands {
		seen = seen || (c.UserID == userID && c.State.Stage == StageWilted)
	}
	if !seen {
		t.Errorf("SweepCandidates did not return the wilted test user: %+v", cands)
	}
}
