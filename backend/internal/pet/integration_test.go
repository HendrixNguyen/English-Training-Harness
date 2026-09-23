package pet

import (
	"context"
	"fmt"
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
	if st.LastTargetMetDate == nil || *st.LastTargetMetDate != "2026-09-22" {
		t.Errorf("LastTargetMetDate = %v, want 2026-09-22", st.LastTargetMetDate)
	}
	for i := 0; i < 4; i++ {
		if _, err := repo.PenaliseMiss(ctx, userID, fmt.Sprintf("2026-09-2%d", 3+i), time.Now()); err != nil {
			t.Fatalf("PenaliseMiss %d: %v", i+1, err)
		}
	}
	st, _ = repo.Get(ctx, userID)
	if st.HealthPoints != 0 || st.Stage != StageWilted {
		t.Errorf("after four misses = %+v, want 0/wilted — the pet_stage cast must accept 'wilted'", st)
	}

	cands, err := repo.SweepCandidates(ctx)
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

// The service's once-per-day guarantees are SQL predicates, not Go checks;
// this test is what proves them. It also pins PgRepo.PenaliseMiss's SQL
// arithmetic to ApplyMiss.
func TestIntegrationVerdictWritesAreConditional(t *testing.T) {
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
	const gid = "google-pet-verdict-integration"
	_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)
	var userID string
	if err := pg.Pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, target_goal, timezone) VALUES ($1, $2, '', 'UTC') RETURNING id`,
		"verdict@example.com", gid).Scan(&userID); err != nil {
		t.Fatalf("inserting user: %v", err)
	}
	t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid) })

	repo := NewPgRepo(pg.Pool)
	if err := repo.Ensure(ctx, userID); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	now := time.Now().UTC()

	// 1. SaveTargetMet is once per local date: the second write for the same
	//    date is refused by the predicate, a later date is accepted.
	st, _ := repo.Get(ctx, userID)
	applied, err := repo.SaveTargetMet(ctx, userID, ApplyTargetMet(st, now, "2026-09-22"))
	if err != nil || !applied {
		t.Fatalf("first SaveTargetMet = (%t, %v), want (true, nil)", applied, err)
	}
	st, _ = repo.Get(ctx, userID)
	applied, err = repo.SaveTargetMet(ctx, userID, ApplyTargetMet(st, now, "2026-09-22"))
	if err != nil || applied {
		t.Fatalf("repeat SaveTargetMet = (%t, %v), want (false, nil)", applied, err)
	}
	st, _ = repo.Get(ctx, userID)
	if st.CurrentStreak != 1 || st.LastTargetMetDate == nil || *st.LastTargetMetDate != "2026-09-22" {
		t.Errorf("after two same-day writes = %+v, want streak 1 and last_target_met_date 2026-09-22", st)
	}
	if applied, _ = repo.SaveTargetMet(ctx, userID, ApplyTargetMet(st, now, "2026-09-23")); !applied {
		t.Error("SaveTargetMet for the next day was refused")
	}

	// 2. PenaliseMiss under 8 concurrent writers for one judged day: exactly
	//    one applies, health drops by exactly 30.
	var wg sync.WaitGroup
	results := make(chan bool, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := repo.PenaliseMiss(ctx, userID, "2026-09-23", now)
			if err != nil {
				t.Errorf("concurrent PenaliseMiss: %v", err)
			}
			results <- ok
		}()
	}
	wg.Wait()
	close(results)
	appliedCount := 0
	for ok := range results {
		if ok {
			appliedCount++
		}
	}
	st, _ = repo.Get(ctx, userID)
	if appliedCount != 1 || st.HealthPoints != 70 || st.CurrentStreak != 0 || *st.JudgedThrough != "2026-09-23" {
		t.Errorf("8 concurrent PenaliseMiss: applied %d, state %+v; want 1 and 70/0 judged through 2026-09-23", appliedCount, st)
	}

	// 3. MarkJudged then PenaliseMiss for the same day: the spare wins.
	if ok, err := repo.MarkJudged(ctx, userID, "2026-09-24"); err != nil || !ok {
		t.Fatalf("MarkJudged = (%t, %v)", ok, err)
	}
	if ok, _ := repo.PenaliseMiss(ctx, userID, "2026-09-24", now); ok {
		t.Error("PenaliseMiss applied to a day already marked judged")
	}
	if ok, _ := repo.MarkJudged(ctx, userID, "2026-09-20"); ok {
		t.Error("MarkJudged moved judged_through backwards")
	}

	// 4. The SQL arithmetic mirrors ApplyMiss across the floor and the wilt.
	for i, pre := range []State{{HealthPoints: 100, CurrentStreak: 9, Stage: StageFlowering}, {HealthPoints: 30, CurrentStreak: 1, Stage: StageSprout}, {HealthPoints: 20, Stage: StageSprout}} {
		judged := fmt.Sprintf("2026-10-%02d", i+1)
		if err := repo.Save(ctx, userID, pre); err != nil {
			t.Fatalf("Save pre-image %d: %v", i, err)
		}
		if _, err := repo.PenaliseMiss(ctx, userID, judged, now); err != nil {
			t.Fatalf("PenaliseMiss %d: %v", i, err)
		}
		got, _ := repo.Get(ctx, userID)
		want := ApplyMiss(pre, now, judged)
		if got.HealthPoints != want.HealthPoints || got.Stage != want.Stage || got.CurrentStreak != 0 {
			t.Errorf("SQL miss on %+v = %d/%s, ApplyMiss says %d/%s", pre, got.HealthPoints, got.Stage, want.HealthPoints, want.Stage)
		}
	}

	// 5. Save (the revive path) keeps judged_through monotonic.
	earlier := "2026-01-01"
	if err := repo.Save(ctx, userID, State{HealthPoints: 50, Stage: StageSprout, JudgedThrough: &earlier}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if st, _ = repo.Get(ctx, userID); *st.JudgedThrough != "2026-10-03" {
		t.Errorf("Save moved judged_through back to %s", *st.JudgedThrough)
	}

	cands, err := repo.SweepCandidates(ctx)
	if err != nil {
		t.Fatalf("SweepCandidates: %v", err)
	}
	seen := false
	for _, c := range cands {
		seen = seen || (c.UserID == userID && c.Timezone == "UTC" && c.State.JudgedThrough != nil)
	}
	if !seen {
		t.Errorf("SweepCandidates did not return the test user with its verdict dates: %+v", cands)
	}
}
