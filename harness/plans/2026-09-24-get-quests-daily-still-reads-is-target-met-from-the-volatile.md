---
idea: harness/ideas/_inbox/get-quests-daily-still-reads-is-target-met-from-the-volatile.md
status: done
priority: medium
merged: true
branch: harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile
worktree: .worktrees/get-quests-daily-still-reads-is-target-met-from-the-volatile
pr: "https://github.com/HendrixNguyen/English-Training-Harness/pull/17"
---
# quests + pet: the daily screen reads the durable flag, and every verdict write is one conditional SQL statement — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea (head):** `harness/ideas/_inbox/get-quests-daily-still-reads-is-target-met-from-the-volatile.md`
**Also planned here (each idea's frontmatter points at this plan):**
- `harness/ideas/_inbox/marktargetmet-ignores-rowsaffected-so-a-missing-row-silently.md` → Task 2
- `harness/ideas/_inbox/fakeprogressrepo-upsert-s-monotonic-max-is-not-covered-by-an.md` → Task 3
- `harness/ideas/_inbox/ontargetmet-s-read-then-write-erases-a-concurrent-sweep-s-mi.md` → Task 4
- `harness/ideas/_inbox/pgrepo-save-resets-last-target-met-date-to-null-on-the-reviv.md` → Task 5
- `harness/ideas/_inbox/testtwoconcurrentsweepspenaliseonce-misses-its-defect-in-1-r.md` → Task 6

**Goal:** Finish what `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` started: `GET /quests/daily` answers the same `is_target_met` as `POST /quests/progress` for the same local day (the durable `daily_progress` row wins over a lost Redis counter), the pet's success write does its arithmetic on the live row so a concurrent sweep's −30 can no longer be silently erased, no write path can move `last_target_met_date` backwards, the one verdict writer that could not tell "done" from "nothing there" now can, and the unit suite holds all of it in place deterministically.

**Why now (`priority: medium`):** the head defect is on the screen the learner reads most — after a Redis eviction, restart or `FLUSHDB` the daily screen says the day is unmet while the pet has already been watered for it. The pet race erases a whole penalty (100 → 70 → **100**, reproduced live in review). Both are the exact contradictions the parent plan exists to remove, left in place on two paths.

**Root causes (from each idea's `## Evaluation`, re-read on this branch):**
- `backend/internal/quests/service.go` `Daily` builds `IsTargetMet: total >= TargetSeconds` from the counter alone; `RecordProgress` uses `total >= TargetSeconds || alreadyMet`. `ProgressRepo` has no read method.
- `backend/internal/quests/repo.go` `MarkTargetMet` discards the `CommandTag`; the fake creates the row when absent.
- `backend/internal/quests/fakes_test.go` `Upsert`'s `max(row.minutes, minutes)` is exercised by no unit test that actually lowers the value.
- `backend/internal/pet/repo.go` `saveTargetMetSQL` writes `health_points = $2, stage = $3, current_streak = $4` from a Go pre-image read by `Service.OnTargetMet` (`Ensure` → `SaveTargetMet(ApplyTargetMet(st, …))`), while `penaliseMissSQL` computes on the live row.
- `backend/internal/pet/repo.go` `saveSQL` writes `last_target_met_date = $7::date` unconditionally (only `judged_through` has `GREATEST`); `Revive` is the only caller and builds from a pre-image.
- `backend/internal/pet/service_test.go` `TestTwoConcurrentSweepsPenaliseOnce` has no barrier, so the second sweep usually sees fresh state and skips every pet.

**Design decisions (read before the tasks):**
1. **Same rule on both endpoints.** `is_target_met = counter ≥ 1800 || daily_progress.is_target_met`, computed the same way in `Daily` and `RecordProgress`. `AccumulatedSeconds` keeps reading the counter — it is a live progress bar and the backend spec §6.2 shape does not change.
2. **`SaveTargetMet` takes no state.** New signature `SaveTargetMet(ctx, userID string, now time.Time, localDate string) (applied bool, err error)`: the UPDATE adds the bonus, bumps the streak, derives the stage from the *new* streak and stamps the dates, all against the row's current values, under the unchanged `last_target_met_date IS NULL OR < $d` predicate. `ApplyTargetMet` stays as the Go reference the fake mirrors and the integration test holds the SQL to — exactly how `ApplyMiss` ↔ `penaliseMissSQL` already works. `OnTargetMet` becomes `Ensure` (row creation only) then the write; there is nothing left to go stale.
3. **`Save` keeps both dates monotonic.** `last_target_met_date = GREATEST(last_target_met_date, $7::date)`; Postgres' `GREATEST` ignores NULL on either side, so a revive carrying no marker leaves the stored one alone.
4. **A test barrier, not a sleep.** `fakeRepo.afterCandidates func()` runs once per sweep after `SweepCandidates` has built its list and released the mutex; the test installs a two-party `sync.WaitGroup` there, so both sweeps carry the same stale list into their 20 writes. Deterministic; no timing.
5. **Package boundaries hold.** quests never reads `pet_states`; pet never reads `daily_progress`. Each package's fix stays in its own files; `cmd/api/main.go` is untouched (constructor signatures do not change).

**Tech stack:** Go 1.25, pgx/v5, stdlib `sync`. No new dependencies, no migration.

**Run every command from `backend/` inside the worktree** unless a step says otherwise. `rg` and `timeout` are not installed — use `grep -n` and `go test -timeout`.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/quests/repo.go` | `ProgressRepo.TargetMet`; `targetMetSQL`; `PgRepo.TargetMet`; `ErrNoProgressRow`; `MarkTargetMet` checks `RowsAffected` |
| `backend/internal/quests/service.go` | `Daily` ORs the flag in |
| `backend/internal/quests/fakes_test.go` | `fakeProgressRepo.TargetMet`; `MarkTargetMet` refuses a missing row |
| `backend/internal/quests/service_test.go` | `TestDailyReportsTheDurableFlagWhenTheCounterIsLost`, `TestALostCounterNeverLowersTheDurableMinutes`, `TestFakeMarkTargetMetRefusesAMissingRowLikeTheSQL` |
| `backend/internal/quests/integration_test.go` | `Daily` after the `DEL`; `MarkTargetMet` on a date with no row |
| `backend/internal/pet/repo.go` | `SaveTargetMet(ctx, userID, now, localDate)`; `saveTargetMetSQL` arithmetic in SQL; `saveSQL` `GREATEST` on the marker |
| `backend/internal/pet/service.go` | `OnTargetMet` = `Ensure` + write; doc comment |
| `backend/internal/pet/fakes_test.go` | `SaveTargetMet` mirrors under one lock; `Save` keeps the later marker; `afterCandidates` hook |
| `backend/internal/pet/service_test.go` | Barrier in `TestTwoConcurrentSweepsPenaliseOnce`; `TestFakeSaveKeepsAMarkerItWasNotGiven` |
| `backend/internal/pet/integration_test.go` | Section 1 new signature; new section 4b (miss-then-met = 90; SQL ↔ `ApplyTargetMet` mirror); section 5 marker assertion |
| `harness/CODEMAP.md` | quests and pet paragraphs |
| `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` | One-line resolution under *Notes → Residual race, documented* |

---

## Tasks

### Task 1: quests — `Daily` reads the durable flag

**Files:**
- Modify: `backend/internal/quests/service_test.go` (append)
- Modify: `backend/internal/quests/repo.go`, `backend/internal/quests/fakes_test.go`, `backend/internal/quests/service.go`
- Modify: `backend/internal/quests/integration_test.go`

- [ ] **Step 1: Write the failing unit test**

```go
func TestDailyReportsTheDurableFlagWhenTheCounterIsLost(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	ctx := context.Background()
	if _, err := h.svc.RecordProgress(ctx, "u1", "ex-2-reading", 1800); err != nil {
		t.Fatal(err)
	}
	h.counter.totals = map[string]int64{} // eviction / restart / FLUSHDB

	got, err := h.svc.Daily(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if got.AccumulatedSeconds != 0 || !got.IsTargetMet {
		t.Errorf("Daily after the counter was lost = accumulated %d, is_target_met %t; want 0 and true — the durable row wins, exactly as POST /quests/progress already answers", got.AccumulatedSeconds, got.IsTargetMet)
	}
}
```

- [ ] **Step 2: Run to see it fail**

Run: `go test ./internal/quests/ -run TestDailyReportsTheDurableFlagWhenTheCounterIsLost -count=1`
Expected: FAIL — `is_target_met false`.

- [ ] **Step 3: Add the read to the interface, the SQL, the repo and the fake**

`repo.go` — in `ProgressRepo`, after `MarkTargetMet`:

```go
	// TargetMet reports the row's is_target_met for (userID, localDate); no
	// row → false. GET /quests/daily reads it so both endpoints answer the
	// same is_target_met for the same local day, counter or no counter.
	TargetMet(ctx context.Context, userID, localDate string) (bool, error)
```

in the `const (` block:

```go
	targetMetSQL = `
SELECT COALESCE(is_target_met, FALSE)
FROM daily_progress
WHERE user_id = $1 AND date = $2::date`
```

and the method:

```go
func (r *PgRepo) TargetMet(ctx context.Context, userID, localDate string) (bool, error) {
	var met bool
	err := r.Pool.QueryRow(ctx, targetMetSQL, userID, localDate).Scan(&met)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("quests: reading daily_progress flag: %w", err)
	}
	return met, nil
}
```

`fakes_test.go`:

```go
func (f *fakeProgressRepo) TargetMet(_ context.Context, userID, localDate string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.rows[userID+"|"+localDate].targetMet, nil
}
```

`service.go` `Daily` — after the `total` read:

```go
	flagged, err := s.progress.TargetMet(ctx, userID, date)
	if err != nil {
		return DailySuite{}, err
	}
```

and in the returned struct:

```go
		// The durable row wins over a lost counter — the same rule
		// RecordProgress answers with, so the two endpoints never disagree
		// about one local day. AccumulatedSeconds stays live (the progress bar).
		IsTargetMet:          total >= TargetSeconds || flagged,
```

Update `Daily`'s doc comment: "…returns that day's tasks, today's running total and whether the day is met (counter or durable flag)."

- [ ] **Step 4: Extend the integration test** — after the `out3` block (the counter-loss section at the end), append:

```go
	after, err := svc.Daily(ctx, userID)
	if err != nil {
		t.Fatalf("Daily after the counter was lost: %v", err)
	}
	if after.AccumulatedSeconds != 60 || !after.IsTargetMet {
		t.Errorf("Daily after the counter was lost = accumulated %d, is_target_met %t; want 60 (the one post-loss report) and true (durable row)", after.AccumulatedSeconds, after.IsTargetMet)
	}
```

- [ ] **Step 5: Run the package tests**

Run: `go test ./internal/quests/ -count=1 -v -run 'Daily'`
Expected: PASS incl. the new test; `go test ./internal/quests/ -count=1` → ok (integration skips locally).

- [ ] **Step 6: Commit**

```bash
git add internal/quests
git commit -m "quests: GET /quests/daily reports is_target_met from the durable row too"
```

### Task 2: quests — `MarkTargetMet` reports a missing row

**Files:**
- Modify: `backend/internal/quests/service_test.go` (append), `backend/internal/quests/integration_test.go`
- Modify: `backend/internal/quests/repo.go`, `backend/internal/quests/fakes_test.go`

- [ ] **Step 1: Write the failing tests**

Unit (the fake must mirror the SQL, or the unit suite cannot see the case):

```go
func TestFakeMarkTargetMetRefusesAMissingRowLikeTheSQL(t *testing.T) {
	f := newFakeProgressRepo(&callLog{})
	if err := f.MarkTargetMet(context.Background(), "u1", "2026-09-22"); !errors.Is(err, ErrNoProgressRow) {
		t.Fatalf("MarkTargetMet with no row: err = %v, want ErrNoProgressRow (PgRepo returns it on RowsAffected() == 0)", err)
	}
	if _, ok := f.rows["u1|2026-09-22"]; ok {
		t.Error("the fake created a row; the SQL UPDATE cannot")
	}
}
```

Integration — after the `MarkTargetMet`-independent seeding at the top of `TestIntegrationDailyAndProgressAgainstRealServices` (right before `suite, err := svc.Daily(ctx, userID)`):

```go
	if err := repo.MarkTargetMet(ctx, userID, "1999-01-01"); !errors.Is(err, ErrNoProgressRow) {
		t.Errorf("MarkTargetMet for a date with no row: err = %v, want ErrNoProgressRow", err)
	}
```

- [ ] **Step 2: Run to see it fail**

Run: `go test ./internal/quests/ -run TestFakeMarkTargetMetRefusesAMissingRow -count=1`
Expected: compile error — `undefined: ErrNoProgressRow`.

- [ ] **Step 3: Implement**

`repo.go` — next to `ErrExerciseNotFound`:

```go
// ErrNoProgressRow means MarkTargetMet found no daily_progress row for that
// local date. RecordProgress cannot hit it (Upsert creates the row on the
// same call); it exists so no caller can mistake "nothing there" for
// "flagged" — every verdict writer in pet already reports what it did.
var ErrNoProgressRow = errors.New("quests: no daily_progress row for that date")
```

`MarkTargetMet`:

```go
func (r *PgRepo) MarkTargetMet(ctx context.Context, userID, localDate string) error {
	tag, err := r.Pool.Exec(ctx, markTargetMetSQL, userID, localDate)
	if err != nil {
		return fmt.Errorf("quests: marking target met: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNoProgressRow
	}
	return nil
}
```

Interface comment: `// MarkTargetMet flips is_target_met to TRUE. Idempotent; ErrNoProgressRow when the row does not exist.`

`fakes_test.go` `MarkTargetMet` — replace the `row := f.rows[key]` line with:

```go
	row, ok := f.rows[key]
	if !ok {
		return ErrNoProgressRow
	}
```

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/quests/ -count=1`
Expected: ok — every existing test still upserts before it flags.

- [ ] **Step 5: Commit**

```bash
git add internal/quests
git commit -m "quests: MarkTargetMet reports a missing daily_progress row"
```

### Task 3: quests — the monotonic upsert is pinned by a test that actually lowers the value

**Files:**
- Modify: `backend/internal/quests/service_test.go` (append)

- [ ] **Step 1: Write the test**

```go
func TestALostCounterNeverLowersTheDurableMinutes(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	ctx := context.Background()
	if _, err := h.svc.RecordProgress(ctx, "u1", "ex-2-reading", 1800); err != nil { // row: 30 minutes
		t.Fatal(err)
	}
	h.counter.totals = map[string]int64{} // the counter restarts at 0…
	if _, err := h.svc.RecordProgress(ctx, "u1", "ex-2-practice", 60); err != nil { // …so this report upserts minutes = 1
		t.Fatal(err)
	}
	if row := h.progress.rows["u1|2026-09-22"]; row.minutes != 30 || !row.targetMet {
		t.Errorf("row = %+v after a short report on a lost counter, want minutes 30 (never lowered) and target met", row)
	}
}
```

- [ ] **Step 2: Run it, then prove it has teeth**

Run: `go test ./internal/quests/ -run TestALostCounterNeverLowersTheDurableMinutes -count=1` → PASS.
Mutation: in `fakes_test.go` change `row.minutes = max(row.minutes, minutes)` to `row.minutes = minutes`, rerun → FAIL (`minutes 1`), restore, rerun → PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/quests/service_test.go
git commit -m "quests: pin the monotonic minutes_spent in the unit suite with a value that drops"
```

### Task 4: pet — `SaveTargetMet` computes on the live row

**Files:**
- Modify: `backend/internal/pet/repo.go`, `backend/internal/pet/service.go`, `backend/internal/pet/fakes_test.go`
- Modify: `backend/internal/pet/integration_test.go`

- [ ] **Step 1: Change the contract and the callers (tests first: the integration test's section 1 and the new 4b)**

In `integration_test.go` section 1, the three `repo.SaveTargetMet(ctx, userID, ApplyTargetMet(st, now, D))` calls become `repo.SaveTargetMet(ctx, userID, now, D)`; the intermediate `st, _ = repo.Get(...)` before the second call can stay (it is now only used for the streak assertion after it). Then insert **section 4b** between section 4 and section 5:

```go
	// 4b. The success write adds to the LIVE row: a miss that landed after any
	//     earlier read is kept. -30 then +20 is 90, never 100 (the review's
	//     live reproduction of the erased penalty). And the SQL mirrors
	//     ApplyTargetMet the way penaliseMissSQL mirrors ApplyMiss.
	pre := State{HealthPoints: 100, CurrentStreak: 5, Stage: StageSapling}
	if err := repo.Save(ctx, userID, pre); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if ok, _ := repo.PenaliseMiss(ctx, userID, "2026-10-04", now); !ok {
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
		{HealthPoints: 95, CurrentStreak: 2, Stage: StageSprout},     // cap at 100, sapling at 3
		{HealthPoints: 40, CurrentStreak: 6, Stage: StageSapling},    // flowering at 7
		{HealthPoints: 0, CurrentStreak: 0, Stage: StageWilted},      // a met day revives arithmetic-wise: 20, sprout
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
```

Section 5's `judged_through` expectation moves from `"2026-10-03"` to `"2026-10-04"` (4b's `PenaliseMiss` advanced it). Section 5 also gains Task 5's marker assertion — see there.

- [ ] **Step 2: Run to see it fail**

Run: `go vet ./internal/pet/`
Expected: type errors on the new call shape.

- [ ] **Step 3: Implement**

`repo.go` — interface:

```go
	// SaveTargetMet applies §8's success arithmetic in SQL on the live row —
	// +TargetMetHealthBonus capped at MaxHealth, streak+1, stage from the new
	// streak, last_practiced_at = updated_at = now, last_target_met_date =
	// localDate — only while last_target_met_date is NULL or before
	// localDate. Pre-image and write are one statement, like PenaliseMiss: a
	// concurrent miss can no longer be overwritten by a stale Go-side read.
	// ApplyTargetMet is the Go reference this SQL is held to.
	SaveTargetMet(ctx context.Context, userID string, now time.Time, localDate string) (applied bool, err error)
```

SQL (replace `saveTargetMetSQL`):

```go
	// Pre-image and write in one statement: every right-hand side reads the
	// row's current values. health' is always > 0 (health ≥ 0 plus the
	// bonus), so the stage CASE needs only StageFor's streak thresholds;
	// integration section 4b pins it to ApplyTargetMet.
	saveTargetMetSQL = `
UPDATE pet_states
SET health_points = LEAST($4, COALESCE(health_points, $4) + $3),
    current_streak = COALESCE(current_streak, 0) + 1,
    stage = (CASE
               WHEN COALESCE(current_streak, 0) + 1 >= 14 THEN 'fruitful'
               WHEN COALESCE(current_streak, 0) + 1 >= 7  THEN 'flowering'
               WHEN COALESCE(current_streak, 0) + 1 >= 3  THEN 'sapling'
               ELSE 'sprout'
             END)::pet_stage,
    last_practiced_at = $5,
    updated_at = $5,
    last_target_met_date = $2::date
WHERE user_id = $1 AND (last_target_met_date IS NULL OR last_target_met_date < $2::date)`
```

Method:

```go
func (r *PgRepo) SaveTargetMet(ctx context.Context, userID string, now time.Time, localDate string) (bool, error) {
	tag, err := r.Pool.Exec(ctx, saveTargetMetSQL, userID, localDate, TargetMetHealthBonus, MaxHealth, now)
	if err != nil {
		return false, fmt.Errorf("pet: saving target met: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
```

Also update the `Repo` type comment ("The three verdict writers are conditional UPDATEs…") to add: "and compute on the live row — no writer takes a Go-side pre-image except Save (revive), whose two date columns are GREATEST-protected."

`service.go` `OnTargetMet`:

```go
// OnTargetMet is §8's success logic for the user's local day localDate:
// quests fires it when the day's total first reaches 1800s (backend spec
// §6.2), and may fire it again after a failure on the same call or after a
// lost Redis counter. The pet owns the once: Repo.SaveTargetMet's predicate
// on last_target_met_date refuses a second write for the same (or an
// earlier) local date, and that refusal is a silent no-op. Ensure only
// creates the row; the arithmetic runs in SQL on the live row, so nothing
// read here can go stale before the write — a sweep's -30 landing between
// the two calls is kept, not overwritten.
func (s *Service) OnTargetMet(ctx context.Context, userID, localDate string) error {
	if err := s.repo.Ensure(ctx, userID); err != nil {
		return err
	}
	_, err := s.repo.SaveTargetMet(ctx, userID, s.now(), localDate)
	return err
}
```

`fakes_test.go`:

```go
// SaveTargetMet mirrors saveTargetMetSQL: read and write under one lock is
// the fake's "one statement"; ApplyTargetMet is the shared reference.
func (f *fakeRepo) SaveTargetMet(_ context.Context, userID string, now time.Time, localDate string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.writeErr(userID); err != nil {
		return false, err
	}
	cur, ok := f.states[userID]
	if !ok || !dateBefore(cur.LastTargetMetDate, localDate) {
		return false, nil
	}
	f.saved++
	f.states[userID] = ApplyTargetMet(cur, now, localDate)
	return true, nil
}
```

`grep -n 'SaveTargetMet(' internal/pet/*.go` — every remaining call must have the new shape (service.go, fakes_test.go, integration_test.go).

- [ ] **Step 4: Run the package tests**

Run: `go vet ./internal/pet/ && go test ./internal/pet/ -count=1 -v -run 'OnTargetMet|TargetMet'`
Expected: PASS — the existing `TestOnTargetMetAppliesSpec8SuccessOnceAndPersists`, `…TwiceForTheSameLocalDateBumpsOnce`, `…ForTheNextLocalDateBumpsAgain`, `…ForAnEarlierLocalDateIsIgnored` are unchanged in meaning and still green through the fake. `go test ./internal/pet/ -count=1` → ok.

- [ ] **Step 5: Commit**

```bash
git add internal/pet
git commit -m "pet: SaveTargetMet does the success arithmetic in SQL on the live row"
```

### Task 5: pet — `Save` never moves `last_target_met_date` backwards

**Files:**
- Modify: `backend/internal/pet/service_test.go` (append), `backend/internal/pet/integration_test.go` (section 5)
- Modify: `backend/internal/pet/repo.go`, `backend/internal/pet/fakes_test.go`

- [ ] **Step 1: Write the failing tests**

Unit (keeps the fake honest to the SQL):

```go
func TestFakeSaveKeepsAMarkerItWasNotGiven(t *testing.T) {
	h := newHarness(sept22)
	marker := "2026-09-22"
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted, LastTargetMetDate: &marker}

	// Revive's Save carries the pre-image's marker — nil when the pre-image
	// predates a concurrent OnTargetMet. GREATEST(last_target_met_date, NULL)
	// keeps the stored one.
	if err := h.repo.Save(ctx, "u1", State{HealthPoints: 50, Stage: StageSprout}); err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["u1"].LastTargetMetDate; got == nil || *got != marker {
		t.Errorf("Save with a nil marker left last_target_met_date = %v, want %s kept", got, marker)
	}
	earlier := "2026-01-01"
	_ = h.repo.Save(ctx, "u1", State{HealthPoints: 50, Stage: StageSprout, LastTargetMetDate: &earlier})
	if got := h.repo.states["u1"].LastTargetMetDate; got == nil || *got != marker {
		t.Errorf("Save with an earlier marker moved last_target_met_date to %v, want %s kept", got, marker)
	}
}
```

Integration — section 5, after the existing `judged_through` assertion (now `"2026-10-04"`):

```go
	if st.LastTargetMetDate == nil || *st.LastTargetMetDate != "2026-11-04" {
		t.Errorf("Save with a nil marker moved last_target_met_date to %v, want 2026-11-04 kept (GREATEST ignores NULL)", st.LastTargetMetDate)
	}
```

- [ ] **Step 2: Run to see it fail**

Run: `go test ./internal/pet/ -run TestFakeSaveKeepsAMarkerItWasNotGiven -count=1`
Expected: FAIL — `left last_target_met_date = <nil>`.

- [ ] **Step 3: Implement**

`repo.go` `saveSQL`:

```go
	// GREATEST ignores NULL on either side, so both verdict dates only ever
	// move forward: a revive built from a pre-image that predates a concurrent
	// OnTargetMet cannot erase that day's marker.
	saveSQL = `
UPDATE pet_states
SET health_points = $2, stage = $3::pet_stage, current_streak = $4, last_practiced_at = $5, updated_at = $6,
    last_target_met_date = GREATEST(last_target_met_date, $7::date), judged_through = GREATEST(judged_through, $8::date)
WHERE user_id = $1`
```

Interface comment for `Save`: "Save writes every mutable column unconditionally except the two verdict dates, which only ever move forward. Revive uses it; the verdict writers do not."

`fakes_test.go` `Save` — after the `JudgedThrough` mirror:

```go
	if s.LastTargetMetDate != nil {
		s.LastTargetMetDate = laterDate(cur.LastTargetMetDate, *s.LastTargetMetDate) // GREATEST(last_target_met_date, $7)
	} else {
		s.LastTargetMetDate = cur.LastTargetMetDate
	}
```

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/pet/ -count=1`
Expected: ok (the revive tests are unaffected: `ApplyRevive` never sets the marker).

- [ ] **Step 5: Commit**

```bash
git add internal/pet
git commit -m "pet: Save keeps last_target_met_date monotonic like judged_through"
```

### Task 6: pet — the concurrent-sweep test forces the overlap

**Files:**
- Modify: `backend/internal/pet/fakes_test.go`, `backend/internal/pet/service_test.go`

- [ ] **Step 1: Add the hook to the fake**

`fakeRepo` gains a field:

```go
	// afterCandidates, when set, runs once per SweepCandidates call after the
	// list is built and the mutex released — a test barrier so two sweeps can
	// be made to hold the same stale list before either writes.
	afterCandidates func()
```

`SweepCandidates` releases the lock before the hook:

```go
func (f *fakeRepo) SweepCandidates(context.Context) ([]Candidate, error) {
	f.mu.Lock()
	var out []Candidate
	for userID, s := range f.states {
		tz, _ := f.Timezone(context.Background(), userID)
		out = append(out, Candidate{UserID: userID, Timezone: tz, State: s})
	}
	f.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].UserID < out[j].UserID })
	if f.afterCandidates != nil {
		f.afterCandidates()
	}
	return out, nil
}
```

- [ ] **Step 2: Install the barrier in the test** — in `TestTwoConcurrentSweepsPenaliseOnce`, after the 20-pet seeding loop and before the goroutines:

```go
	// Neither sweep may write until both have read: with identical stale
	// candidate lists, only the conditional write can keep the count at 20.
	var ready sync.WaitGroup
	ready.Add(2)
	h.repo.afterCandidates = func() { ready.Done(); ready.Wait() }
```

Assertions stay exactly as they are (sum 20; every pet at 70).

- [ ] **Step 3: Prove determinism both ways**

```bash
go test ./internal/pet/ -run TestTwoConcurrentSweepsPenaliseOnce -count=300 -timeout 120s
# expect: ok (300/300)
go test ./internal/pet/ -run TestTwoConcurrentSweepsPenaliseOnce -count=100 -race -timeout 120s
# expect: ok
```

Mutation A — in `service.go` `Sweep`, replace `if applied { penalised++ }` with an unconditional `penalised++`:
`go test ./internal/pet/ -run TestTwoConcurrentSweepsPenaliseOnce -count=30` → FAIL 30/30 (`reported 40 penalties`). Restore.
Mutation B — in `fakes_test.go` `PenaliseMiss`, delete the `!dateBefore(cur.JudgedThrough, judged)` half of the guard:
`go test ./internal/pet/ -run TestTwoConcurrentSweepsPenaliseOnce -count=30` → FAIL 30/30 (`health = 40, want 70`). Restore.

- [ ] **Step 4: Commit**

```bash
git add internal/pet/fakes_test.go internal/pet/service_test.go
git commit -m "pet: the concurrent-sweep test forces both sweeps to read before either writes"
```

### Task 7: CODEMAP and the parent plan's note

**Files:**
- Modify: `harness/CODEMAP.md`
- Modify: `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` (body only — never frontmatter)

- [ ] **Step 1: CODEMAP `quests`** — in the `GET /api/v1/quests/daily` sentence, after `is_target_met`: "(`accumulated_seconds >= 1800 || daily_progress.is_target_met` via `ProgressRepo.TargetMet` — the same rule `POST /quests/progress` answers with, so a lost counter never makes the two endpoints disagree)". In the `MarkTargetMet` sentence: "`MarkTargetMet` flips the flag (and returns `ErrNoProgressRow` on zero rows — unreachable from `RecordProgress`, which upserts first)".

- [ ] **Step 2: CODEMAP `pet`** — replace "`Service.OnTargetMet(user, D)` writes through `Repo.SaveTargetMet`, an `UPDATE … WHERE last_target_met_date IS NULL OR last_target_met_date < D`, so a repeat call … is a silent no-op" with "`Service.OnTargetMet(user, D)` is `Ensure` then `Repo.SaveTargetMet(user, now, D)` — one `UPDATE` that does the +20/streak/stage arithmetic **on the live row** under `last_target_met_date IS NULL OR last_target_met_date < D` (no Go-side pre-image; `ApplyTargetMet` is the reference the fake and `TestIntegrationVerdictWritesAreConditional` hold the SQL to), so a repeat call … is a silent no-op and a sweep's −30 landing mid-request is kept". After "`ApplyRevive` sets `judged_through = today`": "— `Save` protects both `judged_through` and `last_target_met_date` with `GREATEST`, so a revive can never move a verdict date backwards".

- [ ] **Step 3: Parent plan** — under *Notes and open questions* → the "Residual race, documented" bullet, append one line: `_Resolved 2026-09-24 by harness/plans/2026-09-24-get-quests-daily-still-reads-is-target-met-from-the-volatile.md: SaveTargetMet computes in SQL on the live row; the third outcome (100/1, the erased penalty) can no longer occur._`

- [ ] **Step 4: Commit**

```bash
git add ../harness/CODEMAP.md "../harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md"
git commit -m "harness: CODEMAP tells the truth about the durable flag and the single-statement success write"
```

---

## Verification

```bash
cd backend
gofmt -l ./internal/quests ./internal/pet
# expect: no output for files this plan touched (internal/quests/handler_test.go is a known pre-existing hit until the CI gofmt plan lands — do not reformat it here)
go build ./... && go vet ./... && go test ./... -count=1
# expect: ok for every package, no live service needed
go test ./internal/quests/ -count=1 -v -run 'DailyReportsTheDurableFlag|ALostCounterNeverLowers|FakeMarkTargetMetRefuses'
# expect: PASS ×3
go test ./internal/pet/ -count=1 -v -run 'OnTargetMet|FakeSaveKeepsAMarker|TwoConcurrentSweeps'
# expect: PASS — the four pre-existing OnTargetMet tests, FakeSaveKeepsAMarkerItWasNotGiven, TwoConcurrentSweepsPenaliseOnce
go test ./internal/pet/ -run TestTwoConcurrentSweepsPenaliseOnce -count=300 -timeout 120s
# expect: ok
grep -n 'IsTargetMet:' internal/quests/service.go
# expect: 2 lines, both containing `||`
grep -n 'health_points = \$2' internal/pet/repo.go
# expect: 1 hit — saveSQL only (the revive path); saveTargetMetSQL computes with LEAST
grep -c 'GREATEST(last_target_met_date' internal/pet/repo.go
# expect: 1
grep -n 'ApplyTargetMet(' internal/pet/service.go
# expect: no output — the service no longer builds a pre-image
grep -rn 'daily_progress' internal/pet/ ; grep -rn 'pet_states' internal/quests/
# expect: no hits — the boundary holds
git log --oneline origin/main..HEAD | wc -l
# expect: 7 commits, one per task, each with the Co-Authored-By trailer
python3 ../tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0

# With TEST_DATABASE_URL / TEST_REDIS_URL exported (unique compose project, -p 1):
go test ./internal/quests/ ./internal/pet/ -count=1 -v -run Integration -p 1
# expect: PASS TestIntegrationDailyAndProgressAgainstRealServices (Daily after the DEL → 60/true; MarkTargetMet on 1999-01-01 → ErrNoProgressRow),
#         PASS TestIntegrationVerdictWritesAreConditional (section 4b: 70 + 20 = 90; four SQL ↔ ApplyTargetMet mirrors; section 5 keeps the 2026-11-04 marker),
#         PASS TestIntegrationEnsureCreatesExactlyOnePetRow
```

Mutation checks (each must turn the named test red, then restore):

| Mutation | Test that fails |
| --- | --- |
| `Daily`: `IsTargetMet: total >= TargetSeconds` (drop `|| flagged`) | `DailyReportsTheDurableFlagWhenTheCounterIsLost`; integration `Daily after the DEL` |
| `fakeProgressRepo.Upsert`: `row.minutes = minutes` | `ALostCounterNeverLowersTheDurableMinutes` |
| `PgRepo.MarkTargetMet`: drop the `RowsAffected` check | integration `MarkTargetMet on 1999-01-01` |
| `saveTargetMetSQL`: `health_points = $4` (absolute) | integration 4b (`want health 90`) and the mirror loop |
| `saveTargetMetSQL`: drop the `WHERE … last_target_met_date` predicate | integration section 1 (`repeat SaveTargetMet = (true, …)`) |
| `saveSQL`: `last_target_met_date = $7::date` | `FakeSaveKeepsAMarkerItWasNotGiven` (fake mirror) and integration section 5 |
| `Sweep`: unconditional `penalised++` | `TwoConcurrentSweepsPenaliseOnce` 30/30 |

## Notes and open questions

- **Why not a Go-side `SELECT … FOR UPDATE` transaction for the success write?** It would reintroduce a pre-image (now locked) and a second round trip; the single conditional `UPDATE` is what the parent plan's design decision 2 asked for and what `PenaliseMiss` already does. The stage thresholds (3/7/14) now appear in two places — `StageFor` and the SQL `CASE` — exactly as `penaliseMissSQL` already duplicates the wilt rule; integration section 4b is the drift alarm.
- **`Daily` makes one more Postgres read per call.** `daily_progress` is keyed `(user_id, date)` (the §3.2 unique constraint), so it is an index lookup. Acceptable; the alternative (widening `ExercisesForDay`) crosses no boundary but muddles a read that belongs to `ProgressRepo`.
- **`TestFakeMarkTargetMetRefusesAMissingRowLikeTheSQL` and `TestFakeSaveKeepsAMarkerItWasNotGiven` test fakes.** Deliberate: both fakes mirror SQL predicates the unit suite otherwise cannot observe; a fake that drifts from its SQL is how the reviewer's finding went unseen. The SQL itself is pinned by the integration test.
- **Out of scope:** the multi-day sweep catch-up (`a-multi-day-sweep-outage-collapses-…`, selected medium, decision recorded in its Evaluation) — a separate plan.

## Execution summary

Status: **done**. Branch `harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile`, worktree `.worktrees/get-quests-daily-still-reads-is-target-met-from-the-volatile` (created off freshly fetched `origin/main`). All 7 tasks implemented exactly as specified, TDD throughout (failing test → implementation → pass → commit).

**Commits** (8, one extra beyond the plan's 7 — see deviation below):
1. `46c3ae6` quests: GET /quests/daily reports is_target_met from the durable row too
2. `474d32f` quests: MarkTargetMet reports a missing daily_progress row
3. `f9fce85` quests: pin the monotonic minutes_spent in the unit suite with a value that drops
4. `c94d288` pet: SaveTargetMet does the success arithmetic in SQL on the live row
5. `4431d16` pet: Save keeps last_target_met_date monotonic like judged_through
6. `6cd65e1` pet: the concurrent-sweep test forces both sweeps to read before either writes
7. `eb693c0` harness: CODEMAP tells the truth about the durable flag and the single-statement success write
8. `db1f5e8` quests: gofmt the monotonic-minutes test (deviation, see below)

**Deviations:**
- Added an 8th commit. `gofmt -l` flagged `internal/quests/service_test.go` after Task 3's `TestALostCounterNeverLowersTheDurableMinutes` was inserted verbatim from the plan — the plan's own snippet left an EOL-comment alignment gofmt wants across two adjacent commented lines. Ran `gofmt -w` on that one file and committed the whitespace-only fix separately rather than folding it into Task 3's commit (repo convention favors new commits over amends). `internal/quests/repo.go` and `internal/quests/handler_test.go` remain gofmt-dirty exactly as before this plan (`repo.go`'s hit is a pre-existing `Exercise.TaskType` comment-alignment issue unrelated to any line this plan touched, confirmed identical on `origin/main`; `handler_test.go` is the plan's documented pre-existing exception, left alone).
- Two of the plan's own grep-based verification lines are cosmetically off from their literal "expect" text, though the underlying behavior is exactly as designed:
  - `grep -n 'IsTargetMet:' internal/quests/service.go` → 2 lines, but only 1 contains `||` textually. `Daily`'s line does (`total >= TargetSeconds || flagged`); `RecordProgress`'s line reads a precomputed `targetMet` variable (`targetMet := total >= TargetSeconds || alreadyMet`, unchanged pre-existing code, outside Task 1's scope) rather than inlining the `||`. Design decision 1's actual requirement — "the same rule, computed the same way" — holds; only the textual grep assumption doesn't match RecordProgress's existing style.
  - `grep -rn 'pet_states' internal/quests/` → not empty; all hits are pre-existing doc comments (`internal/quests/pet.go`) and one test error string (`service_test.go:310`, `"pet_states unreachable"`) explaining *why* quests never reads that table — confirmed present verbatim on `origin/main` before this plan. No SQL/query against `pet_states` exists in `internal/quests/`; the actual boundary holds.

**Verification (from `backend/`):**
- `gofmt -l ./internal/quests ./internal/pet` → only `internal/quests/handler_test.go` (documented pre-existing) and `internal/quests/repo.go` (pre-existing, unrelated line, confirmed on `origin/main`).
- `go build ./... && go vet ./... && go test ./... -count=1` → all 10 packages `ok`.
- `go test ./internal/quests/ -count=1 -v -run 'DailyReportsTheDurableFlag|ALostCounterNeverLowers|FakeMarkTargetMetRefuses'` → PASS ×3.
- `go test ./internal/pet/ -count=1 -v -run 'OnTargetMet|FakeSaveKeepsAMarker|TwoConcurrentSweeps'` → PASS ×7 (4 pre-existing `OnTargetMet` tests unchanged in meaning, `TwoConcurrentSweepsPenaliseOnce`, `FakeSaveKeepsAMarkerItWasNotGiven`).
- `go test ./internal/pet/ -run TestTwoConcurrentSweepsPenaliseOnce -count=300 -timeout 120s` → ok (300/300); also `-count=100 -race` → ok.
- Mutation checks, all as specified: unconditional `penalised++` → 30/30 FAIL (`reported 40 penalties`), restored; fake `PenaliseMiss` guard minus the `JudgedThrough` half → 30/30 FAIL (`health = 40, want 70`), restored; fake `Upsert`'s `row.minutes = minutes` (drop the `max`) → FAIL (`minutes 1`), restored.
- Grep checks: `health_points = $2` → 1 hit (`saveSQL` only); `GREATEST(last_target_met_date` → 1 hit; `ApplyTargetMet(` in `pet/service.go` → no output (service no longer builds a pre-image); `daily_progress` in `internal/pet/` → no hits. (`IsTargetMet:` and `pet_states` in `internal/quests/` results explained under Deviations above.)
- `git log --oneline origin/main..HEAD | wc -l` → 8 (7 planned + 1 gofmt fix, see Deviations).
- `python3 ../tools/harness/cli.py validate; echo exit=$?` → exit=0.

**Integration tests** (isolated compose project `exec-quests`, Postgres on 55435, Redis on 56382):
- `go test ./internal/quests/ ./internal/pet/ -count=1 -v -run Integration -p 1` → PASS `TestIntegrationDailyAndProgressAgainstRealServices` (Daily after the Redis `DEL` → 60/true; `MarkTargetMet` on `1999-01-01` → `ErrNoProgressRow`), PASS `TestIntegrationVerdictWritesAreConditional` (section 4b: 70+20=90, four SQL↔`ApplyTargetMet` mirrors; section 5 keeps the `2026-11-04` marker and `2026-10-04` `judged_through`), PASS `TestIntegrationEnsureCreatesExactlyOnePetRow`.
- `make test-integration` (the documented command, run for real with `TEST_DATABASE_URL`/`TEST_REDIS_URL` exported) → PASS across all 10 packages, no `--- SKIP`.

**Runtime proof:** Built and booted `cmd/api` on port 18084 against the isolated compose stack (`DATABASE_URL`/`REDIS_URL` pointed at 55435/56382, dummy `JWT_SECRET`/`GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`). Seeded one real user + active roadmap via `onboarding.PgRepo.SaveAssessment` (same helper shape as the integration test), minted a real session JWT + Redis `sess:{user_id}:token` key, and drove the actual HTTP surface end to end:
- `GET /quests/daily` (fresh) → `is_target_met: false`, 3 tasks.
- `POST /quests/progress` (1800s) → `is_target_met: true`, `pet_health: 100`, `streak_count: 1`.
- `redis-cli DEL daily:accumulated:...` (simulated eviction/restart/FLUSHDB) then `GET /quests/daily` again → **`accumulated_seconds: 0`, `is_target_met: true`** — the exact head defect, reproduced and confirmed fixed live over real HTTP against a real Postgres row.
- `GET /pet/status` → `health_points: 100, current_streak: 1, stage: sprout` — matches `ApplyTargetMet`'s live-row arithmetic.
The seed helper (`backend/cmd/e2eseed/`) and its build artifacts were scratch-only, never committed, and removed afterward; `git status` in the worktree is clean.

**Cleanup verified:** API process killed (`pgrep e2e-api` → no matches), `docker compose -p exec-quests down` (containers + network removed, confirmed via `docker ps -a --filter name=exec-quests` → empty), scratch `backend/.env` removed.

**CI:** green on the pushed branch — https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35960183886 (`backend-unit`, `backend-integration`, `harness-tooling`, `frontend` all ✓).

Branch pushed (`git push -u origin harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile`); no PR opened.
