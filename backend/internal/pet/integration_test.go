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
		if _, _, err := repo.PenaliseMiss(ctx, userID, fmt.Sprintf("2026-09-2%d", 3+i), time.Now()); err != nil {
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
	applied, err := repo.SaveTargetMet(ctx, userID, now, "2026-09-22")
	if err != nil || !applied {
		t.Fatalf("first SaveTargetMet = (%t, %v), want (true, nil)", applied, err)
	}
	st, _ = repo.Get(ctx, userID)
	applied, err = repo.SaveTargetMet(ctx, userID, now, "2026-09-22")
	if err != nil || applied {
		t.Fatalf("repeat SaveTargetMet = (%t, %v), want (false, nil)", applied, err)
	}
	st, _ = repo.Get(ctx, userID)
	if st.CurrentStreak != 1 || st.LastTargetMetDate == nil || *st.LastTargetMetDate != "2026-09-22" {
		t.Errorf("after two same-day writes = %+v, want streak 1 and last_target_met_date 2026-09-22", st)
	}
	if applied, _ = repo.SaveTargetMet(ctx, userID, now, "2026-09-23"); !applied {
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
			ok, _, err := repo.PenaliseMiss(ctx, userID, "2026-09-23", now)
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
	if ok, _, _ := repo.PenaliseMiss(ctx, userID, "2026-09-24", now); ok {
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
		if _, _, err := repo.PenaliseMiss(ctx, userID, judged, now); err != nil {
			t.Fatalf("PenaliseMiss %d: %v", i, err)
		}
		got, _ := repo.Get(ctx, userID)
		want := ApplyMiss(pre, now, judged)
		if got.HealthPoints != want.HealthPoints || got.Stage != want.Stage || got.CurrentStreak != 0 {
			t.Errorf("SQL miss on %+v = %d/%s, ApplyMiss says %d/%s", pre, got.HealthPoints, got.Stage, want.HealthPoints, want.Stage)
		}
	}

	// 4b. The success write adds to the LIVE row: a miss that landed after any
	//     earlier read is kept. -30 then +20 is 90, never 100 (the review's
	//     live reproduction of the erased penalty). And the SQL mirrors
	//     ApplyTargetMet the way penaliseMissSQL mirrors ApplyMiss.
	pre := State{HealthPoints: 100, CurrentStreak: 5, Stage: StageSapling}
	if err := repo.Save(ctx, userID, pre); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if ok, _, _ := repo.PenaliseMiss(ctx, userID, "2026-10-04", now); !ok {
		t.Fatal("PenaliseMiss for 2026-10-04 did not apply")
	}
	if ok, err := repo.SaveTargetMet(ctx, userID, now, "2026-10-05"); err != nil || !ok {
		t.Fatalf("SaveTargetMet after a miss = (%t, %v), want (true, nil)", ok, err)
	}
	st, _ = repo.Get(ctx, userID)
	if st.HealthPoints != 90 || st.CurrentStreak != 1 || st.Stage != StageSprout || st.LastTargetMetDate == nil || *st.LastTargetMetDate != "2026-10-05" {
		t.Errorf("miss then met = %+v, want health 90 (70 + 20), streak 1, sprout, marker 2026-10-05 — the -30 must survive", st)
	}
	for i, pre := range []State{
		{HealthPoints: 95, CurrentStreak: 2, Stage: StageSprout},      // cap at 100, sapling at 3
		{HealthPoints: 40, CurrentStreak: 6, Stage: StageSapling},     // flowering at 7
		{HealthPoints: 0, CurrentStreak: 0, Stage: StageWilted},       // a met day revives arithmetic-wise: 20, sprout
		{HealthPoints: 100, CurrentStreak: 13, Stage: StageFlowering}, // fruitful at 14
	} {
		d := fmt.Sprintf("2026-11-%02d", i+1)
		if err := repo.Save(ctx, userID, pre); err != nil {
			t.Fatalf("Save pre-image %d: %v", i, err)
		}
		if ok, err := repo.SaveTargetMet(ctx, userID, now, d); err != nil || !ok {
			t.Fatalf("SaveTargetMet %d = (%t, %v)", i, ok, err)
		}
		got, _ := repo.Get(ctx, userID)
		want := ApplyTargetMet(pre, now, d)
		if got.HealthPoints != want.HealthPoints || got.CurrentStreak != want.CurrentStreak || got.Stage != want.Stage || got.LastPracticedAt == nil {
			t.Errorf("SQL success on %+v = %d/%d/%s, ApplyTargetMet says %d/%d/%s", pre, got.HealthPoints, got.CurrentStreak, got.Stage, want.HealthPoints, want.CurrentStreak, want.Stage)
		}
	}

	// 5. Save (the revive path) keeps judged_through monotonic.
	earlier := "2026-01-01"
	if err := repo.Save(ctx, userID, State{HealthPoints: 50, Stage: StageSprout, JudgedThrough: &earlier}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if st, _ = repo.Get(ctx, userID); *st.JudgedThrough != "2026-10-04" {
		t.Errorf("Save moved judged_through back to %s", *st.JudgedThrough)
	}
	if st.LastTargetMetDate == nil || *st.LastTargetMetDate != "2026-11-04" {
		t.Errorf("Save with a nil marker moved last_target_met_date to %v, want 2026-11-04 kept (GREATEST ignores NULL)", st.LastTargetMetDate)
	}

	// 6. Shields: award every 7th met day capped at 2, spend before the penalty,
	//    fall through when none is held — SQL held to ApplyTargetMet/ApplyMiss.
	//    Save never writes shields (decision 4), so pre-images are set directly.
	setShields := func(streak, shields int) {
		if _, err := pg.Pool.Exec(ctx, `UPDATE pet_states SET health_points = 100, current_streak = $2, stage = 'flowering', shields = $3, last_shield_used_on = NULL WHERE user_id = $1`, userID, streak, shields); err != nil {
			t.Fatalf("setting shields: %v", err)
		}
	}
	for i, tt := range []struct{ streak, shields, want int }{{6, 0, 1}, {13, 1, 2}, {20, 2, 2}, {7, 1, 1}} {
		setShields(tt.streak, tt.shields)
		d := fmt.Sprintf("2026-12-%02d", i+1)
		pre, _ := repo.Get(ctx, userID)
		if ok, err := repo.SaveTargetMet(ctx, userID, now, d); err != nil || !ok {
			t.Fatalf("SaveTargetMet %d = (%t, %v)", i, ok, err)
		}
		got, _ := repo.Get(ctx, userID)
		if want := ApplyTargetMet(pre, now, d); got.Shields != want.Shields || got.Shields != tt.want || got.LastShieldUsedOn != nil {
			t.Errorf("award from streak %d / shields %d: SQL %d, ApplyTargetMet %d, want %d (last used must stay NULL)", tt.streak, tt.shields, got.Shields, want.Shields, tt.want)
		}
	}
	setShields(21, 2)
	pre, _ = repo.Get(ctx, userID)
	applied, shielded, err := repo.PenaliseMiss(ctx, userID, "2026-12-10", now)
	if err != nil || !applied || !shielded {
		t.Fatalf("shielded PenaliseMiss = (%t, %t, %v), want (true, true, nil)", applied, shielded, err)
	}
	st, _ = repo.Get(ctx, userID)
	if want := ApplyMiss(pre, now, "2026-12-10"); st.HealthPoints != 100 || st.CurrentStreak != 21 || st.Stage != StageFlowering || st.Shields != 1 || st.LastShieldUsedOn == nil || *st.LastShieldUsedOn != "2026-12-10" || *st.JudgedThrough != "2026-12-10" || st.Shields != want.Shields {
		t.Errorf("shielded miss = %+v, want the plant untouched, shields 1, last used and judged through 2026-12-10 (ApplyMiss: shields %d)", st, want.Shields)
	}
	if applied, shielded, _ := repo.PenaliseMiss(ctx, userID, "2026-12-10", now); applied || shielded {
		t.Error("a repeat PenaliseMiss for a shielded day must be a no-op")
	}
	if applied, shielded, _ := repo.PenaliseMiss(ctx, userID, "2026-12-11", now); !applied || !shielded {
		t.Error("the second shield must be spent on the next missed day")
	}
	pre, _ = repo.Get(ctx, userID)
	applied, shielded, _ = repo.PenaliseMiss(ctx, userID, "2026-12-12", now)
	st, _ = repo.Get(ctx, userID)
	if want := ApplyMiss(pre, now, "2026-12-12"); !applied || shielded || st.HealthPoints != want.HealthPoints || st.HealthPoints != 70 || st.CurrentStreak != 0 || st.Shields != 0 || *st.LastShieldUsedOn != "2026-12-11" {
		t.Errorf("unshielded miss = (%t, %t) %+v, want (true, false) 70/0, shields 0, last used kept at 2026-12-11", applied, shielded, st)
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
