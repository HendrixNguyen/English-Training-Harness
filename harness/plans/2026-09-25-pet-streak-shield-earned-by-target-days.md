---
idea: harness/ideas/2026-09-22-run-01/pet-streak-shield-earned-by-target-days.md
status: approved
priority: medium
merged: false
design: harness/designs/pet-streak-shield.md
---
# pet: a streak shield, earned every 7th met day, is spent in place of the miss penalty — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/2026-09-22-run-01/pet-streak-shield-earned-by-target-days.md`
**Design:** `harness/designs/pet-streak-shield.md`

**Goal:** Every 7th consecutive target-met day awards the pet one shield (max 2). When the hourly sweep judges a missed day and a shield is held, the shield is spent instead: health, streak and stage are untouched, `last_shield_used_on` records the day, `judged_through` still advances. `GET /api/v1/pet/status` reports `shields` and `last_shield_used_on`; the hub draws a two-slot shield rack under the health bar, marks a spend for seven days, and the plant announces the earn.

**Why now (`priority: medium`, feature slot 1 of 5 — two-cap rule, owner 2026-09-25):** oldest `selected` feature without a plan (since 2026-09-22). Confirmed on `origin/main` today: `backend/internal/pet/engine.go` `ApplyMiss` is −30 / streak 0 / wilt with no safety net; `repo.go` `penaliseMissSQL` and `saveTargetMetSQL` are the two single conditional `UPDATE`s that carry the once-per-day guarantee; `frontend/pages/index.vue` shows plant, `HealthBar`, `SpeechBubble` and nothing that survives a miss. The streak-break moment is the churn cluster; a shield earned only by seven met days keeps the daily pressure and makes one bad day survivable.

**Design decisions (taken by the evaluator; do not re-litigate):**
1. **Migration `0004_pet_shields`** adds `shields INT NOT NULL DEFAULT 0 CHECK (shields BETWEEN 0 AND 2)` and `last_shield_used_on DATE` to `pet_states`. The backend spec's DDL gets the identical block after its 0003 block (AGENTS.md: spec DDL == migrations). `store/integration_test.go`'s `reset` down-loop is **unchanged**: 0004, like 0003, only adds columns to `pet_states`, which `0001_init.down.sql`'s `DROP TABLE` removes.
2. **The award lives inside `saveTargetMetSQL`**, the consume inside `penaliseMissSQL` — one conditional statement each, so the once-per-day predicate that protects health also protects the shield. Every right-hand side of an `UPDATE ... SET` reads the pre-image, so `shields` in each `CASE` is the value before the write. `ApplyTargetMet` / `ApplyMiss` gain the same logic as the Go reference; the fake mirrors it; `TestIntegrationVerdictWritesAreConditional` holds the SQL to the Go.
3. **`PenaliseMiss` returns `(applied, shielded bool, err)`** via `RETURNING COALESCE(last_shield_used_on = $2::date, FALSE)` (true iff this write consumed a shield — a second consume for the same `judged` is impossible because the predicate requires `judged_through < judged`). `Sweep` counts `applied && !shielded`, so the operator's log line "penalised N" excludes shielded misses. Chosen over the pre-image `c.State.Shields` (smaller, but stale under the concurrent sweepers this package is built for) and over a re-read (a second query per pet).
4. **`Save` (the revive path) does not write `shields`/`last_shield_used_on`.** `ApplyRevive` leaves them alone and a shielded miss never wilts, so `Revive` is unaffected. Consequence for tests: integration pre-images that need a shield count are set with a direct `UPDATE`, not `Save`.
5. **Wire:** `GET /pet/status` gains additive `shields` (int) and `last_shield_used_on` (`YYYY-MM-DD` or `null`). The backend spec §6.3 example is updated; the exact-body handler test matches it byte for byte. `POST /quests/progress` is untouched — the hub re-reads `/pet/status` on mount after the learning room navigates back, which is where the earn is seen.
6. **Frontend per the design:** `ShieldRow` (two always-drawn slots: `streak` when held, `mute` + tick when spent within 7 days, `mute/30` outline when empty) under `HealthBar`; caption "Khiên đã đỡ cho ngày dd/mm." within 7 days; `speechLine` earn line when `streak > 0 && streak % 7 === 0 && shields > 0`, worded so it stays true at the cap. Reduced-motion safe, no confetti. Today's growth-moment feature plan (`harness/designs/growth-moment.md`, written in parallel) may later announce "Shield earned" in its celebration; **this plan must not depend on it.**

**Tech stack:** Go 1.25 / Gin / pgx (backend), Nuxt 3 + Pinia + Vitest + `@vue/test-utils` (frontend). No new dependencies.

**Run every command from the worktree root** unless a step says otherwise. `rg` and `timeout` are not installed — use `grep -n`. Backend unit tests run with `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL` in front (abbreviated `env -u …` below).

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/store/migrations/0004_pet_shields.up.sql` / `.down.sql` | **new** — the two columns |
| `backend/internal/store/integration_test.go` | `want` gains `0004_pet_shields`; `versions = 4`; `reset` comment names 0004 |
| `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` | DDL: "Added by migration 0004" block after the 0003 block; §6.3 `GET /pet/status` example + one bullet |
| `backend/internal/pet/engine.go` | `MaxShields`, `ShieldEveryDays`; `State.Shields`, `State.LastShieldUsedOn`; award in `ApplyTargetMet`, consume in `ApplyMiss` |
| `backend/internal/pet/engine_test.go` | award (7, 14, cap at 21), consume before penalty, no-shield fallthrough, revive keeps shields |
| `backend/internal/pet/repo.go` | `stateColumns`/`scanState` read the two columns; `saveTargetMetSQL` award; `penaliseMissSQL` consume + `RETURNING`; `Repo.PenaliseMiss` tri-return |
| `backend/internal/pet/fakes_test.go` | `Save` keeps shields; `PenaliseMiss` returns `shielded` |
| `backend/internal/pet/service.go` | `Sweep` counts only unshielded misses |
| `backend/internal/pet/service_test.go` | `TestSweepSpendsAShieldInsteadOfPenalising`; one shielded row in the per-zone table |
| `backend/internal/pet/handler.go` / `handler_test.go` | `statusResponse` gains the two fields; exact-body test updated; new keys test |
| `backend/internal/pet/integration_test.go` | section 6: award / cap / consume / fallthrough against `ApplyTargetMet` / `ApplyMiss`; `PenaliseMiss` call sites take three values |
| `frontend/stores/pet.ts` | `PetStatus.shields`, `PetStatus.last_shield_used_on` |
| `frontend/utils/plant.ts` | `MAX_SHIELDS`, `SHIELD_EVERY_DAYS`, `localDateYmd`, `daysBetweenDates`, `shieldSpentLine`; `speechLine` earn line |
| `frontend/components/plant/ShieldRow.vue` | **new** — the rack and caption |
| `frontend/pages/index.vue` | `ShieldRow` under `HealthBar`; bubble receives streak + shields |
| `frontend/tests/unit/plant.test.ts`, `tests/unit/ShieldRow.test.ts` (new), `tests/unit/petStore.test.ts` | the cases in Tasks 4 and 5 |
| `harness/CODEMAP.md` | `pet`, `store`, `shell` paragraphs |

---

## Tasks

### Task 1: Migration 0004 and the spec DDL

**Files:**
- Create: `backend/internal/store/migrations/0004_pet_shields.up.sql`, `backend/internal/store/migrations/0004_pet_shields.down.sql`
- Modify: `backend/internal/store/integration_test.go`, `project-base/Adaptive English Learning Platform - Backend Technical Specification.md`

- [ ] **Step 1: Write the failing test change.** In `store/integration_test.go`: the `want` list in `TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent` becomes `[]string{"0001_init", "0002_google_sync", "0003_pet_verdict_dates", "0004_pet_shields"}`; `const versions = 4 // 0001_init, 0002_google_sync, 0003_pet_verdict_dates, 0004_pet_shields`; the `reset` comment's first sentence becomes "0003_pet_verdict_dates.up.sql and 0004_pet_shields.up.sql only add columns to pet_states, so 0001_init's DROP TABLE (which the loop below still runs) removes them too — unlike …" (the down-loop itself is unchanged).
- [ ] **Step 2: Write the migration.** `0004_pet_shields.up.sql`:

```sql
-- Migration 0004 — the streak shield (pet streak shield plan).
-- shields: how many shields the pet holds, 0..2. saveTargetMetSQL awards one on
--   every 7th consecutive met day (LEAST(2, …)); penaliseMissSQL spends one in
--   place of the -30 / streak reset when a judged day was missed.
-- last_shield_used_on: the local YYYY-MM-DD a shield was last spent for; NULL
--   until the first spend. The client shows the spend for seven days.

ALTER TABLE pet_states
    ADD COLUMN shields INT NOT NULL DEFAULT 0 CHECK (shields BETWEEN 0 AND 2),
    ADD COLUMN last_shield_used_on DATE;
```

  `0004_pet_shields.down.sql`:

```sql
-- Reverse of 0004_pet_shields.up.sql.

ALTER TABLE pet_states
    DROP COLUMN IF EXISTS shields,
    DROP COLUMN IF EXISTS last_shield_used_on;
```

- [ ] **Step 3: Spec DDL.** In the backend spec, directly after the `judged_through DATE;` line of the "Added by migration 0003" block (inside the same ```sql fence, before the closing fence), append:

```sql

-- Added by migration 0004 (streak shield): one shield per 7th consecutive met day, max 2; spent in place of a miss penalty.
ALTER TABLE pet_states
    ADD COLUMN shields INT NOT NULL DEFAULT 0 CHECK (shields BETWEEN 0 AND 2),
    ADD COLUMN last_shield_used_on DATE;
```

- [ ] **Step 4: Run.** From `backend/`: `env -u … go test ./internal/store/ -count=1` → `ok` (integration tests skip locally; CI's `backend-integration` runs them — the `want` list is what proves 0004 is applied and named correctly). `diff <(grep -A3 'Added by migration 0004' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md" | tail -3) <(grep -A2 '^ALTER TABLE' internal/store/migrations/0004_pet_shields.up.sql)` → no output.
- [ ] **Step 5: Commit:** `git commit -am "store: migration 0004 adds pet_states.shields and last_shield_used_on; spec DDL mirrors it"` (use `git add` for the two new files first).

### Task 2: Engine — the Go reference

**Files:**
- Modify: `backend/internal/pet/engine.go`, `backend/internal/pet/engine_test.go`

- [ ] **Step 1: Write the failing tests** in `engine_test.go`:

```go
func TestApplyTargetMetAwardsAShieldEverySeventhDayCappedAtTwo(t *testing.T) {
	for _, tt := range []struct{ streak, shields, wantShields int }{
		{6, 0, 1},  // day 7
		{7, 1, 1},  // day 8: nothing
		{13, 1, 2}, // day 14
		{20, 2, 2}, // day 21 with a full rack: capped
		{13, 0, 1}, // day 14 after a spend
	} {
		got := ApplyTargetMet(State{HealthPoints: 100, CurrentStreak: tt.streak, Shields: tt.shields}, sept22, "2026-09-22")
		if got.Shields != tt.wantShields || got.CurrentStreak != tt.streak+1 {
			t.Errorf("streak %d, shields %d → shields %d (streak %d), want %d", tt.streak, tt.shields, got.Shields, got.CurrentStreak, tt.wantShields)
		}
		if got.LastShieldUsedOn != nil {
			t.Error("a success must not touch LastShieldUsedOn")
		}
	}
}

func TestApplyMissSpendsAShieldBeforeThePenalty(t *testing.T) {
	pre := State{HealthPoints: 100, CurrentStreak: 9, Stage: StageFlowering, Shields: 1}
	got := ApplyMiss(pre, sept22, "2026-09-22")
	if got.HealthPoints != 100 || got.CurrentStreak != 9 || got.Stage != StageFlowering {
		t.Errorf("shielded miss changed the plant: %+v", got)
	}
	if got.Shields != 0 || got.LastShieldUsedOn == nil || *got.LastShieldUsedOn != "2026-09-22" {
		t.Errorf("shields/last used = %d/%v, want 0/2026-09-22", got.Shields, got.LastShieldUsedOn)
	}
	if got.JudgedThrough == nil || *got.JudgedThrough != "2026-09-22" || !got.UpdatedAt.Equal(sept22) {
		t.Errorf("a shielded miss must still resolve the day and stamp the clock: %+v", got)
	}
	// No shield left: the ordinary §8 penalty, and the spend date is kept.
	again := ApplyMiss(got, sept22, "2026-09-23")
	if again.HealthPoints != 70 || again.CurrentStreak != 0 || again.Stage != StageSprout || again.Shields != 0 || *again.LastShieldUsedOn != "2026-09-22" {
		t.Errorf("unshielded miss = %+v, want 70/0/sprout, shields 0, last used kept", again)
	}
}
```

  and in `TestApplyReviveResetsToFiftySproutZeroStreak` add: `if r := ApplyRevive(State{HealthPoints: 0, Stage: StageWilted, Shields: 1}, now, "2026-09-22"); r.Shields != 1 { t.Error("revive must not touch shields") }`.
- [ ] **Step 2: Run red:** from `backend/`: `env -u … go test ./internal/pet/ -run 'TestApplyTargetMetAwards|TestApplyMissSpends|TestApplyRevive' -count=1` → compile errors (`Shields` undefined).
- [ ] **Step 3: Make them pass.** In `engine.go`:
  - constants: `MaxShields = 2 // migration 0004 CHECK (shields BETWEEN 0 AND 2)` and `ShieldEveryDays = 7 // a shield per 7th consecutive met day`.
  - `State` gains, after `JudgedThrough`:

```go
	// Shields is how many streak shields the pet holds (0..MaxShields,
	// migration 0004). ApplyTargetMet awards one on every ShieldEveryDays-th
	// consecutive met day; ApplyMiss spends one instead of the §8 penalty.
	Shields int
	// LastShieldUsedOn is the local YYYY-MM-DD a shield was last spent for;
	// nil until the first spend. The client shows the spend for seven days.
	LastShieldUsedOn *string
```

  - `ApplyTargetMet`, right after `s.CurrentStreak++`:

```go
	if s.CurrentStreak%ShieldEveryDays == 0 {
		s.Shields = min(MaxShields, s.Shields+1)
	}
```

  - `ApplyMiss` becomes:

```go
func ApplyMiss(s State, now time.Time, judged string) State {
	if s.Shields > 0 {
		// The shield takes the hit: health, streak and stage are untouched.
		s.Shields--
		s.LastShieldUsedOn = &judged
	} else {
		s.HealthPoints = max(0, s.HealthPoints-MissPenalty)
		s.CurrentStreak = 0
		s.Stage = StageFor(s.HealthPoints, s.CurrentStreak)
	}
	s.JudgedThrough = laterDate(s.JudgedThrough, judged)
	s.UpdatedAt = now
	return s
}
```

  Extend the two doc comments with one sentence each (award / spend), and `ApplyRevive`'s with "Shields are untouched."
- [ ] **Step 4: Run green:** `env -u … go test ./internal/pet/ -run 'TestApply|TestStageFor|TestSpec8' -count=1` → `ok`.
- [ ] **Step 5: Commit:** `git commit -am "pet: ApplyTargetMet awards a shield every 7th met day (max 2); ApplyMiss spends one before the penalty"`.

### Task 3: Repo, fake, sweep, handler

**Files:**
- Modify: `backend/internal/pet/repo.go`, `fakes_test.go`, `service.go`, `service_test.go`, `handler.go`, `handler_test.go`, `integration_test.go`, the backend spec §6.3

- [ ] **Step 1: Write the failing tests.**
  - `service_test.go`, new:

```go
func TestSweepSpendsAShieldInsteadOfPenalising(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, CurrentStreak: 9, Stage: StageFlowering, Shields: 1, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}

	n, err := h.svc.Sweep(ctx, midnite) // judges the 22nd, which was missed
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	got := h.repo.states["u1"]
	if n != 0 || got.HealthPoints != 100 || got.CurrentStreak != 9 || got.Stage != StageFlowering {
		t.Errorf("n=%d state=%+v; want 0 penalised and the plant untouched — the shield took the hit", n, got)
	}
	if got.Shields != 0 || got.LastShieldUsedOn == nil || *got.LastShieldUsedOn != "2026-09-22" || got.JudgedThrough == nil || *got.JudgedThrough != "2026-09-22" {
		t.Errorf("shields=%d last used=%v judged=%v; want 0 / 2026-09-22 / 2026-09-22", got.Shields, got.LastShieldUsedOn, got.JudgedThrough)
	}
	if n, _ := h.svc.Sweep(ctx, midnite.Add(time.Hour)); n != 0 || h.repo.states["u1"].Shields != 0 {
		t.Error("a second sweep the same day must be a no-op")
	}
	if n, _ := h.svc.Sweep(ctx, midnite.AddDate(0, 0, 1)); n != 1 || h.repo.states["u1"].HealthPoints != 70 || h.repo.states["u1"].CurrentStreak != 0 {
		t.Errorf("with no shield left the 23rd is penalised: n=%d state=%+v", n, h.repo.states["u1"])
	}
}
```

    and in `TestSweepJudgesEachPetsOwnLocalYesterday` add a seeded row `seed("utc-shielded", "UTC", 100, 9, "2026-09-21")` with `Shields: 1` set on it (the `seed` helper takes no shields; set `st.Shields = 1` the way the test sets `LastTargetMetDate`), keep `n != 3` (a shielded miss is not counted), and add `"utc-shielded": {100, 9, "2026-09-22"}` to `want`.
  - `handler_test.go`: in `TestStatusFormatsLastPracticedAtAsUTCRFC3339` set `Shields: 1` on the seeded state and change `want` to `{"plant_name":"My Green Buddy","stage":"sprout","health_points":80,"current_streak":5,"last_practiced_at":"2026-09-21T20:15:00Z","shields":1,"last_shield_used_on":null}`; in `TestStatusReturnsTheSpec63BodyAndCreatesTheRow` add `Shields int` / `LastShieldUsedOn *string` to the decoded struct and assert `body.Shields == 0 && body.LastShieldUsedOn == nil` and `strings.Contains(w.Body.String(), `"last_shield_used_on":null`)`; new `TestStatusReportsASpentShieldDate`: seed `Shields: 0, LastShieldUsedOn: ptr("2026-09-21")`, expect the body to contain `"shields":0,"last_shield_used_on":"2026-09-21"`.
  - `integration_test.go`: every `repo.PenaliseMiss(...)` call takes three values (`ok, _, err :=` / `_, _, err :=` / `ok, _, _ :=`); the concurrent section 2 collects `ok` as before. Append **section 6** to `TestIntegrationVerdictWritesAreConditional`, after section 5:

```go
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
```

    (`ptr` for the handler test: a two-line helper in `handler_test.go` if none exists; `strings` is already imported there.) Also add `Shields: 1` to one of section 4b's rows is **not** done — 4b compares health/streak/stage only and its `Save`d pre-images carry `Shields: 0` while the row may hold one (decision 4); leave 4b as is.
- [ ] **Step 2: Run red:** `env -u … go test ./internal/pet/ -count=1` → compile errors (`Shields`, three-value `PenaliseMiss`).
- [ ] **Step 3: Make them pass.**
  - `repo.go`:
    - `stateColumns` gains `, p.shields, to_char(p.last_shield_used_on, 'YYYY-MM-DD')` at the end; `scanState` appends `&s.Shields, &s.LastShieldUsedOn` to `targets` (after `&s.JudgedThrough`).
    - `Repo` interface: `PenaliseMiss(ctx context.Context, userID, judged string, now time.Time) (applied, shielded bool, err error)` — doc: "…advances judged_through to judged, only while judged_through is NULL or earlier. While `shields > 0` it spends one instead (health/streak/stage untouched, `last_shield_used_on = judged`) and reports `shielded`; ApplyMiss is the Go reference." `Save`'s doc gains "It never writes `shields` / `last_shield_used_on` (only the verdict writers move them)." The interface preamble's "three verdict writers" sentence gains "and every right-hand side reads the pre-image, so the shield CASEs see the count before the write".
    - `saveTargetMetSQL`: add the line `shields = LEAST($6, shields + CASE WHEN (COALESCE(current_streak, 0) + 1) % $7 = 0 THEN 1 ELSE 0 END),` after the `stage` assignment, and pass `MaxShields, ShieldEveryDays` as `$6, $7` in `SaveTargetMet`. Comment: "// The award (decision 2): the 7th, 14th … consecutive met day adds a shield, capped."
    - `penaliseMissSQL` becomes:

```go
	// Pre-image and write in one statement: every right-hand side reads the
	// row's current values, so `shields` in each CASE is the count before the
	// write. With a shield held the plant is untouched and the shield is spent
	// (last_shield_used_on = judged); otherwise the §8 arithmetic — the CASE is
	// StageFor(health, 0) for the two stages a streak of 0 can produce. Either
	// way judged_through advances. RETURNING tells the caller which branch ran:
	// last_shield_used_on can equal $2 only if this write set it, because the
	// predicate refuses a day already judged. The integration test pins it to
	// ApplyMiss.
	penaliseMissSQL = `
UPDATE pet_states
SET health_points = CASE WHEN shields > 0 THEN health_points ELSE GREATEST(0, COALESCE(health_points, 100) - $3) END,
    current_streak = CASE WHEN shields > 0 THEN current_streak ELSE 0 END,
    stage = CASE WHEN shields > 0 THEN stage
                 ELSE (CASE WHEN COALESCE(health_points, 100) - $3 <= 0 THEN 'wilted' ELSE 'sprout' END)::pet_stage END,
    last_shield_used_on = CASE WHEN shields > 0 THEN $2::date ELSE last_shield_used_on END,
    shields = CASE WHEN shields > 0 THEN shields - 1 ELSE shields END,
    judged_through = $2::date,
    updated_at = $4
WHERE user_id = $1 AND (judged_through IS NULL OR judged_through < $2::date)
RETURNING COALESCE(last_shield_used_on = $2::date, FALSE)`
```

    - `PgRepo.PenaliseMiss`:

```go
func (r *PgRepo) PenaliseMiss(ctx context.Context, userID, judged string, now time.Time) (bool, bool, error) {
	var shielded bool
	err := r.Pool.QueryRow(ctx, penaliseMissSQL, userID, judged, MissPenalty, now).Scan(&shielded)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil // already judged: the predicate refused the write
	}
	if err != nil {
		return false, false, fmt.Errorf("pet: penalising miss: %w", err)
	}
	return true, shielded, nil
}
```

  - `fakes_test.go`: `Save` keeps `s.Shields, s.LastShieldUsedOn = cur.Shields, cur.LastShieldUsedOn` (comment: "saveSQL never writes them"); `PenaliseMiss` returns `(false, false, err)` / `(false, false, nil)` / `(true, cur.Shields > 0, nil)` with `f.states[userID] = ApplyMiss(cur, now, judged)` unchanged.
  - `service.go` `Sweep`: `applied, shielded, err := s.repo.PenaliseMiss(...)`; `if applied && !shielded { penalised++ }`; doc comment: "…the count returned is the number of penalties that applied — a shielded miss (the shield spent, the plant untouched) writes the row but is not counted."
  - `handler.go` `statusResponse` gains `Shields int \`json:"shields"\`` and `LastShieldUsedOn *string \`json:"last_shield_used_on"\``, with the comment "additive (streak shield plan); last_shield_used_on is null until the first spend"; `toStatus` copies both.
  - Backend spec §6.3: the `GET /api/v1/pet/status` example becomes `{"plant_name": "My Green Buddy", "health_points": 80, "stage": "sprout", "current_streak": 5, "last_practiced_at": "2026-09-21T20:15:00Z", "shields": 1, "last_shield_used_on": null}` and, under its `Description:` bullet, add `  * shields (0\-2) and last\_shield\_used\_on (YYYY\-MM\-DD or null) are additive (streak shield, 2026\-09\-25): one shield is earned on every 7th consecutive met day and one is spent, in place of the miss penalty, on a missed day.` in the file's escaped style.
- [ ] **Step 4: Run green:** `env -u … go test ./internal/pet/ -count=1 -race` → `ok`; `cd backend && make check` → all `ok`. `grep -rn 'PenaliseMiss(' backend/internal --include='*.go' | grep -v 'func '` → only `service.go` and `integration_test.go` (nothing outside pet calls it).
- [ ] **Step 5: Commit:** `git commit -am "pet: SQL award/spend of the streak shield; PenaliseMiss reports shielded; status carries shields and last_shield_used_on"`.

### Task 4: Frontend helpers and store

**Files:**
- Modify: `frontend/stores/pet.ts`, `frontend/utils/plant.ts`, `frontend/tests/unit/plant.test.ts`, `frontend/tests/unit/petStore.test.ts`

- [ ] **Step 1: Write the failing tests** in `plant.test.ts`:

```ts
  it('announces the shield on every 7th day while one is held (design §4.2)', () => {
    const earn = 'Tròn 7 ngày liên tiếp! Bạn có khiên bảo vệ streak rồi 🛡️'
    expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 7, shields: 1 })).toBe(earn)
    expect(speechLine({ stage: 'fruitful', health: 100, targetMet: true, streak: 21, shields: 2 })).toBe(earn) // full rack: still true
    expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 8, shields: 1 })).toBe('Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿')
    expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 7, shields: 0 })).toBe('Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿')
    expect(speechLine({ stage: 'wilted', health: 0, targetMet: false, streak: 7, shields: 1 })).toBe('…')
    expect(speechLine({ stage: 'sprout', health: 80, targetMet: false })).toBe('Tưới cho tớ 10 phút học đi!') // callers without the new fields
  })

  it('shows the spent-shield caption for seven days, dd/mm', () => {
    expect(shieldSpentLine('2026-09-24', '2026-09-24')).toBe('Khiên đã đỡ cho ngày 24/09.')
    expect(shieldSpentLine('2026-09-24', '2026-10-01')).toBe('Khiên đã đỡ cho ngày 24/09.') // day 7
    expect(shieldSpentLine('2026-09-24', '2026-10-02')).toBeNull() // day 8
    expect(shieldSpentLine('2026-09-24', '2026-09-23')).toBeNull() // clock skew: a future spend is not shown
    expect(shieldSpentLine(null, '2026-09-24')).toBeNull()
    expect(shieldSpentLine('not a date', '2026-09-24')).toBeNull()
    expect(daysBetweenDates('2026-02-28', '2026-03-01')).toBe(1)
    expect(localDateYmd(new Date(2026, 8, 5, 23, 30))).toBe('2026-09-05') // local getters, never toISOString
  })
```

  In `petStore.test.ts` add `shields: 0, last_shield_used_on: null` to the `status` fixture (the type check needs it) and one case: `load exposes shields and last_shield_used_on` — `api.get.mockResolvedValue({ ...status, shields: 1, last_shield_used_on: '2026-09-24' })`, expect `pet.status?.shields` `1` and `pet.status?.last_shield_used_on` `'2026-09-24'`.
- [ ] **Step 2: Run red:** from `frontend/` (`npm ci` first if `node_modules` is missing): `npx vitest run plant petStore` → `shieldSpentLine` / `daysBetweenDates` / `localDateYmd` not exported; the earn cases return the target-met line.
- [ ] **Step 3: Make them pass.** `stores/pet.ts` `PetStatus` gains `shields: number` and `last_shield_used_on: string | null` with the comment `/** streak shield (backend spec §6.3, additive): 0–2 held; the local YYYY-MM-DD a shield was last spent for, or null. */`. `utils/plant.ts`:

```ts
/** Migration 0004 CHECK and the award period (backend pet engine). */
export const MAX_SHIELDS = 2
export const SHIELD_EVERY_DAYS = 7

export function speechLine(o: { stage: string, health: number, targetMet: boolean, streak?: number, shields?: number }): string {
  if (o.stage === 'wilted' || o.health <= 0) return '…'
  const streak = o.streak ?? 0
  // design §4.2: the earn line states the milestone and that a shield is held — true on day 14 and on day 21 at the cap alike.
  if (streak > 0 && streak % SHIELD_EVERY_DAYS === 0 && (o.shields ?? 0) > 0) return 'Tròn 7 ngày liên tiếp! Bạn có khiên bảo vệ streak rồi 🛡️'
  if (o.targetMet) return 'Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿'
  …unchanged…
}

/** The device's local calendar date as YYYY-MM-DD (local getters — toISOString would give the UTC day). */
export function localDateYmd(now: Date = new Date()): string {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${now.getFullYear()}-${p(now.getMonth() + 1)}-${p(now.getDate())}`
}

function parseYmd(s: string | null | undefined): number | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s ?? '')
  return m ? Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])) : null
}

/** Calendar days from one YYYY-MM-DD to another (negative when `to` is earlier); null when either is malformed. Pure arithmetic — no timezone. */
export function daysBetweenDates(from: string | null | undefined, to: string): number | null {
  const a = parseYmd(from), b = parseYmd(to)
  return a === null || b === null ? null : Math.round((b - a) / 86_400_000)
}

/** design §4.1: the caption under the shield rack while the spend is 0–7 days old. */
export function shieldSpentLine(lastUsedOn: string | null | undefined, today: string = localDateYmd()): string | null {
  const d = daysBetweenDates(lastUsedOn, today)
  if (d === null || d < 0 || d > SHIELD_EVERY_DAYS) return null
  const [, mm, dd] = (lastUsedOn as string).split('-')
  return `Khiên đã đỡ cho ngày ${dd}/${mm}.`
}
```

- [ ] **Step 4: Run green:** `npm run lint && npm run typecheck && npx vitest run plant petStore` → clean.
- [ ] **Step 5: Commit:** `git commit -am "frontend: PetStatus carries shields; speechLine earn line; shieldSpentLine caption"`.

### Task 5: `ShieldRow` and the hub

**Files:**
- Create: `frontend/components/plant/ShieldRow.vue`, `frontend/tests/unit/ShieldRow.test.ts`
- Modify: `frontend/pages/index.vue`

- [ ] **Step 1: Write the failing tests** in `ShieldRow.test.ts` (pattern: `PlantSvg.test.ts`):

```ts
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ShieldRow from '~/components/plant/ShieldRow.vue'

const slots = (w: ReturnType<typeof mount>) => w.findAll('[data-shield]').map(el => el.attributes('data-shield'))

describe('ShieldRow (design pet-streak-shield §4.1)', () => {
  it.each([[0, ['empty', 'empty']], [1, ['held', 'empty']], [2, ['held', 'held']]])('draws two slots for %d held', (shields, want) => {
    const w = mount(ShieldRow, { props: { shields, lastUsedOn: null, today: '2026-09-25' } })
    expect(slots(w)).toEqual(want)
    expect(w.find('[role="img"]').attributes('aria-label')).toBe(`Khiên: ${shields} trên 2`)
    expect(w.text()).not.toContain('đã đỡ')
  })

  it('marks the first empty slot as spent and shows the caption within seven days', () => {
    const w = mount(ShieldRow, { props: { shields: 1, lastUsedOn: '2026-09-24', today: '2026-09-25' } })
    expect(slots(w)).toEqual(['held', 'spent'])
    expect(w.text()).toContain('Khiên đã đỡ cho ngày 24/09.')
    expect(w.find('[role="img"]').attributes('aria-label')).toContain('24/09')
    expect(slots(mount(ShieldRow, { props: { shields: 0, lastUsedOn: '2026-09-24', today: '2026-09-25' } }))).toEqual(['spent', 'empty'])
  })

  it('shows no spent marker after seven days, and none on a full rack', () => {
    const old = mount(ShieldRow, { props: { shields: 1, lastUsedOn: '2026-09-17', today: '2026-09-25' } })
    expect(slots(old)).toEqual(['held', 'empty'])
    expect(old.text()).not.toContain('đã đỡ')
    const full = mount(ShieldRow, { props: { shields: 2, lastUsedOn: '2026-09-24', today: '2026-09-25' } })
    expect(slots(full)).toEqual(['held', 'held'])
    expect(full.text()).toContain('Khiên đã đỡ cho ngày 24/09.') // the caption still teaches that the hit was taken
  })

  it('clamps out-of-range counts', () => {
    expect(slots(mount(ShieldRow, { props: { shields: 5, lastUsedOn: null, today: '2026-09-25' } }))).toEqual(['held', 'held'])
    expect(slots(mount(ShieldRow, { props: { shields: -1, lastUsedOn: null, today: '2026-09-25' } }))).toEqual(['empty', 'empty'])
  })
})
```

- [ ] **Step 2: Run red:** `npx vitest run ShieldRow` → module not found.
- [ ] **Step 3: Build the component** — `components/plant/ShieldRow.vue`, from the design §2–§5:

```vue
<script setup lang="ts">
import { computed } from 'vue'
import { MAX_SHIELDS, localDateYmd, shieldSpentLine } from '~/utils/plant'

const props = defineProps<{ shields: number, lastUsedOn: string | null, today?: string }>()

const held = computed(() => Math.min(MAX_SHIELDS, Math.max(0, Math.floor(props.shields))))
const caption = computed(() => shieldSpentLine(props.lastUsedOn, props.today ?? localDateYmd()))
/** design §4.1: held slots first; the first empty slot is "spent" while the caption shows. */
const slots = computed<Array<'held' | 'spent' | 'empty'>>(() =>
  Array.from({ length: MAX_SHIELDS }, (_, i) => i < held.value ? 'held' : i === held.value && caption.value ? 'spent' : 'empty'))
const label = computed(() => `Khiên: ${held.value} trên ${MAX_SHIELDS}` + (caption.value ? `, một chiếc vừa đỡ cho ngày ${caption.value.slice(-6, -1)}` : ''))
</script>

<template>
  <div>
    <div class="flex items-center gap-3" role="img" :aria-label="label">
      <span class="text-sm text-mute" aria-hidden="true">Khiên:</span>
      <span class="flex items-center gap-2" aria-hidden="true">
        <svg
          v-for="(state, i) in slots"
          :key="i"
          :data-shield="state"
          viewBox="0 0 20 20"
          class="size-5 transition-colors duration-200 motion-reduce:transition-none"
          :class="{ 'text-streak': state === 'held', 'text-mute': state !== 'held' }"
        >
          <path
            d="M10 2 L17 5 V10 C17 14.5 13.5 17.5 10 18.5 C6.5 17.5 3 14.5 3 10 V5 Z"
            :fill="state === 'held' ? 'currentColor' : state === 'spent' ? 'currentColor' : 'none'"
            :fill-opacity="state === 'spent' ? 0.25 : 1"
            stroke="currentColor"
            :stroke-opacity="state === 'empty' ? 0.3 : 1"
            stroke-width="1.5"
            stroke-linejoin="round"
          />
          <path v-if="state === 'spent'" d="M6.5 10.5 L9 13 L13.5 7.5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </span>
    </div>
    <p v-if="caption" class="mt-1 text-sm text-mute">{{ caption }}</p>
  </div>
</template>
```

  (`caption.value.slice(-6, -1)` is the `dd/mm` inside "…ngày 24/09." — if the executor prefers, expose a `shieldSpentDate` helper from `utils/plant.ts` instead; keep the aria sentence.)
- [ ] **Step 4: Wire the hub.** In `pages/index.vue`: after `<HealthBar class="mt-3" … />` add `<ShieldRow class="mt-2" :shields="pet.status.shields" :last-used-on="pet.status.last_shield_used_on" :today="quest.daily?.date" />`; the `bubble` computed passes `streak: pet.status.current_streak, shields: pet.status.shields` to `speechLine`. Nothing else on the page moves (design §4.3).
- [ ] **Step 5: Run green:** `npm run lint && npm run typecheck && npm run test:unit` → clean; `npm run build` → succeeds. Reduced-motion check: the slot's only transition is `transition-colors` with `motion-reduce:transition-none` — `grep -c 'motion-reduce' components/plant/ShieldRow.vue` → 1.
- [ ] **Step 6: Commit:** `git add components/plant/ShieldRow.vue tests/unit/ShieldRow.test.ts && git commit -am "frontend: ShieldRow under the health bar — two slots, spent marker for seven days, earn line in the bubble"`.

### Task 6: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md`

- [ ] **Step 1:** `pet` paragraph — after the sentence ending "…`pet.QuestHook` registers it in `main.go`." add: "**Streak shield (migration `0004`, 2026-09-25):** `pet_states.shields` (0–2) and `last_shield_used_on`. `saveTargetMetSQL` awards one on every 7th consecutive met day (`LEAST(2, …)`, `ShieldEveryDays`/`MaxShields`); `penaliseMissSQL` spends one instead of the −30/streak reset when the judged day was missed (health/streak/stage untouched, `last_shield_used_on = judged`, `judged_through` still advances) and `RETURNING`s whether it did, so `PenaliseMiss` is `(applied, shielded, err)` and `Sweep`'s logged count excludes shielded misses. `ApplyTargetMet`/`ApplyMiss` are the Go reference; `Save` never writes the two columns; `Revive` is unaffected (a shielded miss never wilts). `GET /pet/status` carries `shields` and `last_shield_used_on` (additive; spec §6.3 updated)." Also change the `{plant_name, stage, health_points, current_streak, last_practiced_at}` list at the start of the paragraph to include `shields, last_shield_used_on`.
- [ ] **Step 2:** `store` paragraph — after the 0003 sentence add: "`0004_pet_shields` adds `pet_states.shields` / `last_shield_used_on` (the streak shield — see `pet`); the spec's block carries the same `ALTER TABLE` after its `0003` block, and `store/integration_test.go`'s `reset` still only runs the `0002`/`0001` downs because both later migrations only add columns to `pet_states`."
- [ ] **Step 3:** `shell` paragraph — in the `stores/pet.ts` clause add "`PetStatus` carries `shields` / `last_shield_used_on` (2026-09-25)"; in the `/` (7.2) clause add "plus `components/plant/ShieldRow.vue` under the health bar (design `harness/designs/pet-streak-shield.md`: two always-drawn slots, spent marker + caption for seven days via `utils/plant.ts` `shieldSpentLine`, earn line from `speechLine` when `streak % 7 === 0 && shields > 0`)".
- [ ] **Step 4:** `python3 tools/harness/cli.py validate` → 0. Commit: `git commit -am "harness: CODEMAP — streak shield in pet, store and shell"`.

---

## Verification

```bash
cd backend
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./internal/pet/ -count=1 -race -v 2>&1 | grep -c '^--- PASS'
# expect: 48 (44 today + TestApplyTargetMetAwardsAShieldEverySeventhDayCappedAtTwo + TestApplyMissSpendsAShieldBeforeThePenalty + TestSweepSpendsAShieldInsteadOfPenalising + TestStatusReportsASpentShieldDate); the 2 integration tests still SKIP locally
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./internal/store/ ./internal/pet/ -count=1 -race 2>&1 | grep -E '^(ok|FAIL)'
# expect: two ok lines
make check
# expect: fmt-check silent, vet silent, ok for every package under -race
grep -c 'RETURNING COALESCE(last_shield_used_on' internal/pet/repo.go
# expect: 1
grep -c 'shields' internal/pet/repo.go
# expect: ≥ 8 (today: 0)
find internal/store/migrations -name '0004_pet_shields.*.sql' | wc -l
# expect: 2
grep -c '0004_pet_shields' internal/store/integration_test.go
# expect: 3 (the want list, the versions comment, the reset comment)
cd ..
grep -c 'Added by migration' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md"
# expect: 3 (today: 2)
diff <(grep -A3 'Added by migration 0004' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md" | tail -3) <(grep -A2 '^ALTER TABLE' backend/internal/store/migrations/0004_pet_shields.up.sql)
# expect: no output (spec DDL == migration)
grep -c '"shields": 1, "last_shield_used_on": null' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md"
# expect: 1
cd frontend
npm run lint && npm run typecheck && npm run test:unit
# expect: all clean; ShieldRow.test.ts (6 cases: 3 it.each rows + 3) and the 2 new plant.test.ts cases and 1 new petStore case pass — no absolute total is pinned here because node_modules is not installed in the planning checkout
npm run build
# expect: succeeds
grep -c 'ShieldRow' pages/index.vue
# expect: 1
grep -c 'motion-reduce' components/plant/ShieldRow.vue
# expect: 1
cd ..
grep -c 'shield' harness/CODEMAP.md
# expect: ≥ 6 (today: 1 — the pet paragraph's "cannot shield an unjudged earlier miss")
git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP
# expect: no output
python3 tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0
gh run list --branch "$(git branch --show-current)" --limit 1
# expect: backend-unit, backend-integration, harness-tooling, frontend green (backend-integration is what runs section 6 and the migration list)
```

Mutation checks (record the results): (a) in `penaliseMissSQL` change `CASE WHEN shields > 0 THEN health_points` to `CASE WHEN shields > 1 THEN health_points` → section 6's shielded miss goes red in CI (locally: the same edit to `ApplyMiss` makes `TestApplyMissSpendsAShieldBeforeThePenalty` and `TestSweepSpendsAShieldInsteadOfPenalising` red); revert. (b) in `speechLine` drop the `(o.shields ?? 0) > 0` clause → the `streak: 7, shields: 0` case goes red; revert.

## Notes and open questions

- **Merge-day overlap.** Today's pet bug plan (`2026-09-25-a-pet-state-failure-reports-pet-health-0-…`) edits `stores/pet.ts` `applyProgress` and `stores/quest.ts`, and adds `petStore.test.ts` cases; the growth-moment feature plan (parallel) may edit `stores/pet.ts`, `stores/quest.ts` and `pages/index.vue`. This plan touches `PetStatus` (two interface fields), the `status` fixture in `petStore.test.ts`, and two lines of `pages/index.vue`; expect small textual conflicts in those three files on the daily integration branch, resolvable by keeping both sides. Nothing here depends on either plan.
- **Why `Save` stays away from shields.** `Revive` builds `Save`'s pre-image from a `Get` that predates the write; if `Save` wrote `shields` it could erase an award landing between the two calls. The verdict writers own the column, like the two verdict dates.
- **Day 21 with a full rack** shows the earn line although no shield was added (cap). The copy was chosen to be true there ("you hold a shield"); a status field saying "awarded today" would fix the ambiguity properly — an ideator candidate, not this plan.
- **Sweep leniency and shields.** A day spared by the marker or the Redis counter is still `MarkJudged`, never a spend; a shield is spent only on a genuine miss. Unchanged behaviour, stated for the reviewer.
- **The seed of a full 7-day cycle** is `current_streak`, which `ApplyMiss` resets on an unshielded miss — so after a real miss the next shield is seven met days away, by design (the idea's "earned only by hitting the target on 7 consecutive days").
