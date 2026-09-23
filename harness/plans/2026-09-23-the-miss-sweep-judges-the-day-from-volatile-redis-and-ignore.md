---
idea: harness/ideas/_inbox/the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md
status: draft
priority: medium
merged: false
---
# pet: own the day's verdict — durable once-per-day success, a civil-date sweep, and a revival that resolves its day — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea (head):** `harness/ideas/_inbox/the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md`
**Also planned here (each idea's frontmatter points at this plan):**
- `harness/ideas/_inbox/service-ontargetmet-ignores-localdate-so-pet-has-no-idempote.md` → Tasks 2, 5
- `harness/ideas/_inbox/ontargetmet-is-lost-forever-if-a-write-after-the-incrby-fail.md` → Task 7
- `harness/ideas/_inbox/a-passed-revival-is-knocked-from-50-back-to-20-by-the-same-l.md` → Tasks 2, 5, 6
- `harness/ideas/_inbox/sweep-reads-updated-at-then-writes-unconditionally-so-two-sw.md` → Tasks 3, 6
- `harness/ideas/_inbox/zones-that-skip-local-midnight-on-spring-forward-are-never-s.md` → Task 6
- `harness/ideas/_inbox/pet-sweep-tests-cannot-fail-on-the-branches-they-are-named-f.md` → Tasks 4, 6, 8

**Goal:** The pet decides for itself, from state it owns and writes atomically, whether a local day was met, missed or resolved — so a lost Redis counter, a failed write on the crossing call, a second sweeper, a zone with no local midnight, or an evening revival can no longer give a user +40 and two streak days, cost them the +20 forever, double the −30, skip a zone for a day, or knock a revived plant from 50 to 20 before breakfast.

**Why now (`priority: medium`, post-MVP triage 2026-09-23):** every one of the seven findings is a user-visible plant effect with no log line and nothing in the system that can reconstruct whether it was justified. They share one defect, so they are one branch and one merge.

## Root cause — confirmed, with one correction

The lead's reading is right: **the pet's once-per-day judgement is borrowed, not owned.** `Service.OnTargetMet` receives `localDate` and discards it (`backend/internal/pet/service.go:49-58`); once-ness exists only because quests infers a rising edge on the 48-hour Redis counter (`quests/service.go:110-111`); the hourly sweep judges the previous day from that same counter (`pet/service.go:157`) with `redis.Nil` read as zero (`quests/counter.go:56`); its trigger is `Hour() == 0` (`pet/service.go:134`); its write is an unconditional `UPDATE … WHERE user_id = $1` (`pet/repo.go:51-54`); and a passed revival stamps only `updated_at`, so 900 s < 1800 s at midnight (`pet/service.go:107-110`, `engine.go:83-89`).

The correction: **the durable record that already exists is not durable either.** `daily_progress` is rewritten from the Redis total on every progress call — `minutes_spent = EXCLUDED.minutes_spent, is_target_met = EXCLUDED.is_target_met` (`quests/repo.go:93-98`). Lose the counter after the user met the target and the next call downgrades the row to `minutes_spent = 10, is_target_met = FALSE`. So "read `daily_progress` instead of Redis" would not by itself have fixed the sweep; the row has to become monotonic first. That is why this plan changes quests' upsert, not only pet.

## Design decisions (read before the tasks)

**1. Where the day's verdict durably lives.** Two records, each owned by the package that writes it, both keyed by the same `YYYY-MM-DD` string `quests.LocalDate` already produces:

- **quests → `daily_progress.is_target_met`** becomes *monotonic* (`minutes_spent` never lowers, `is_target_met` never returns to FALSE) and gains a precise meaning: *the pet has been told about this day*. The upsert returns the row's current flag (`alreadyMet`); the hook fires when `total ≥ 1800 && !alreadyMet`; the flag is set **after** the hook returns nil (`MarkTargetMet`). A failed upsert, a failed `MarkComplete`, or a failed hook leaves the flag FALSE, so the next progress call fires the hook again. That makes the hook *at-least-once* from quests' side.
- **pet → two new `pet_states` columns** (migration `0003`): `last_target_met_date DATE` — the local day whose §8 success was last applied, written only by `OnTargetMet` through a conditional `UPDATE`; and `judged_through DATE` — the latest local day that can no longer be penalised (its miss was applied, it was spared, or a revival resolved it), written by the sweep and by a passed revival, also conditionally. Pet's conditional write makes quests' at-least-once *exactly-once*.

*Why not have the sweep read `daily_progress`?* Two reasons, either sufficient. (a) CODEMAP's boundary: pet owns `pet_states`, `daily_progress` is quests' table, and the only way through is an interface implemented in quests over the pool — a new object that `cmd/api/main.go` would have to construct and pass, and this plan must not touch that file. (b) Once the hook is reliable, `pet_states.last_target_met_date` *is* the durable "met" verdict, and it is the one the pet can write atomically together with the health it changes. The Redis counter stays in the sweep only as a **leniency fallback** (no marker but counter ≥ 1800 → spared), never as the thing that decides a penalty.

*Why two columns rather than one, or a reuse of `last_practiced_at`?* A single "judged" date cannot serve both writers: a passed revival must resolve the day (no miss at midnight) *without* forbidding the +20 if the user goes on to 30 minutes — today's behaviour, and worth keeping. Reusing `last_practiced_at` for the guard would need `(last_practiced_at AT TIME ZONE $tz)::date` in the predicate and a timezone lookup in the hook; a DATE column keyed by the string quests already passes is exact, tz-free, and makes the guard one SQL predicate. `last_practiced_at` keeps its §6.3 meaning (last *success*) and is not stamped by revive.

**2. Pet-side idempotence.** `OnTargetMet(ctx, user, D)` → `UPDATE pet_states SET … , last_target_met_date = D WHERE user_id = $1 AND (last_target_met_date IS NULL OR last_target_met_date < D)`. `RowsAffected() == 0` is a no-op, not an error (quests logs hook errors; a no-op must not look like a failure). `<` rather than `<>` keeps the marker monotonic, so a user who edits their timezone backwards cannot re-earn a day. **Migration `0003_pet_verdict_dates`** (`ALTER TABLE pet_states ADD COLUMN last_target_met_date DATE, ADD COLUMN judged_through DATE`) plus the same statement appended to the backend spec §3.2 DDL block, as `0002` was (AGENTS.md: spec DDL = migrations).

**3. Sweep: idempotent and DST-correct.** "Local hour is 0" is replaced by **civil-date arithmetic per pet**: at every `:00` UTC the sweep looks at *every* pet, computes `judged = PreviousDate(LocalDate(now, tz))` — the local day that most recently ended — and does nothing when `judged_through >= judged`. Otherwise: spared if `last_target_met_date == judged` or (fallback) the counter for `judged` is ≥ 1800 → `MarkJudged`; else `PenaliseMiss` = `UPDATE … SET health_points = GREATEST(0, health_points - 30), current_streak = 0, stage = CASE … END, judged_through = judged WHERE user_id = $1 AND (judged_through IS NULL OR judged_through < judged)`. Pre-image and write are one statement, so N concurrent sweepers penalise once, and the count returned is the number of `RowsAffected() == 1`. Because selection is by date, not hour, a zone whose clocks jump 23:59:59 → 01:00 is judged at 01:00; a zone whose hour 0 happens twice is judged once; and a tick the process slept through (a deploy at :00) is caught up at the next tick instead of being lost. No instant of "midnight" is ever constructed, so Go's undefined resolution of a non-existent `time.Date(…, 0, 0, 0, 0, loc)` never enters the arithmetic. **First contact rule:** a pet with `judged_through IS NULL` is judged for `judged` only if `LocalDate(updated_at, tz) <= judged` (it existed on that day — `updated_at` is the creation stamp until something writes the row); otherwise the sweep just records `judged_through = judged`, so from its first sweep on no pet depends on `updated_at` again. Cost: one full `pet_states ⋈ users` scan per hour with no I/O for already-judged rows — fine at MVP scale; the SQL prefilter to add later is noted in *Notes*.

**4. Revival — the product decision, made visible.** *A passed revival resolves the local day it was passed on: the sweep will not apply that day's miss.* Rationale: §6.3 promises "resets health to 50%" and §7 frames revival as *the* way out of a wilted plant; a reading where the same day's §8 penalty takes 30 of those 50 back within hours makes the promise false and, worse, makes the outcome depend on whether the user revived before or after that hour's tick. The 15-minute challenge is the price of the day, by design shorter than the 30-minute target. The +20 for reaching 1800 s the same day remains available (see decision 1). Mechanism: `ApplyRevive` sets `judged_through = today` (monotonic `GREATEST`), and the sweep skips any day ≤ `judged_through`. The mirror case from the finding (revive at 00:10, after the tick) now behaves identically to revive at 20:00: both resolve their own day. One consequence to state plainly: because the marker is monotonic, a revival on D+1 also closes an *unapplied* miss for D (the process slept through D's tick). That is right, not a leak — revival requires health 0, so the plant sat at 0 through D and the miss had nothing left to take; applying it late would take it from the 50 that came after. `OnTargetMet` does **not** move `judged_through`, so a met day never shields an unjudged earlier miss (that is the other reason for two columns).

**5. Tests are half the work.** Every named behaviour has a test that turns red under a specific mutation; the table is in *Verification*. Fakes gain per-user error hooks, a clock for `Ensure`, and a mutex so a concurrent sweep is expressible; the real SQL predicates and the SQL mirror of `ApplyMiss` are pinned by an integration test against Postgres.

**Boundary and wiring check (done on `main`, 2026-09-23):** `cmd/api/main.go` constructs `pet.NewService(pet.NewPgRepo(pool), pet.NewRedisChallengeStore(rdb), studyCounter, time.Now)` and `quests.NewService(studyCounter, questRepo, questRepo, pet.NewQuestHook(petSvc), time.Now)`. Nothing outside `pet`/`quests` calls `Repo.Timezones`, `Repo.SweepCandidates`, `ProgressRepo.Upsert`, `ApplyMiss`, `ApplyTargetMet` or `ApplyRevive` (`grep -rn` over `backend/`), so every signature this plan changes is package-internal. **This plan does not edit `backend/cmd/api/main.go`.** If you find you need to, stop and report — do not work around it.

**Spec precedence:** backend spec §8 (arithmetic, hourly cron), §6.2/§6.3 (wire shapes — unchanged by this plan), §3.2 (DDL, edited in Task 1); 1st-thinking §5.2 for the flow. Both spec files are backslash-escaped — grep by content, not `^## `.

**Tech stack:** Go 1.25, Gin, `pgx/v5`, `go-redis/v9`. No new dependencies.

**Run every command from `backend/` inside the worktree** unless a step says otherwise. `rg`/`timeout` are not installed — use `grep -n`, `go test -timeout`. Integration tests: `COMPOSE_PROJECT_NAME=<slug>` in the scratch `backend/.env` with free `POSTGRES_PORT`/`REDIS_PORT`, `make up`, export `TEST_DATABASE_URL`/`TEST_REDIS_URL`, `make test-integration`, `make down` the same project (AGENTS.md).

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/store/migrations/0003_pet_verdict_dates.up.sql` / `.down.sql` | **new** — the two DATE columns |
| `backend/internal/store/migrations_test.go` | expected version list gains `0003_pet_verdict_dates`; `TestMigration0003AddsPetVerdictDates` |
| `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` | §3.2 DDL block: append the `0003` statement after the `0002` block |
| `backend/internal/pet/engine.go` / `engine_test.go` | `State` gains `LastTargetMetDate`, `JudgedThrough`; `ApplyTargetMet`/`ApplyMiss`/`ApplyRevive` take the local date; `PreviousDate`, `laterDate` |
| `backend/internal/pet/repo.go` | `Repo`: `SaveTargetMet`, `PenaliseMiss`, `MarkJudged`; `SweepCandidates(ctx)` (no tz filter); `Timezones` removed; two columns scanned/written |
| `backend/internal/pet/fakes_test.go` | mutex, clock for `Ensure`, per-user `saveErrFor`/`errFor`, `timezoneErr`, the three conditional writers |
| `backend/internal/pet/service.go` / `service_test.go` | `OnTargetMet` guard, `Revive` resolves the day, `Sweep` rewritten on civil dates; the test suite that can fail |
| `backend/internal/pet/handler_test.go` | `newPetRouter` variant with no user id; two 401 tests |
| `backend/internal/pet/integration_test.go` | `SweepCandidates(ctx)`; `TestIntegrationVerdictWritesAreConditional` (SQL predicates + `ApplyMiss` mirror) |
| `backend/internal/quests/repo.go` | `ProgressRepo`: `Upsert(…) (alreadyMet bool, err)` monotonic + `RETURNING`; `MarkTargetMet` |
| `backend/internal/quests/service.go` | hook fires on the durable flag, before `MarkTargetMet` and `MarkComplete`; `IsTargetMet` = counter *or* durable flag |
| `backend/internal/quests/fakes_test.go` / `service_test.go` | fake mirrors the monotonic upsert; `markCompleteErr`, `markTargetMetErr`; call-log order updated; lost-counter / failed-write / failed-hook tests |
| `backend/internal/quests/integration_test.go` | monotonicity against real Postgres after `DEL` of the counter |
| `harness/CODEMAP.md` | `store`, `quests`, `pet` bullets tell the new truth |

---

## Tasks

### Task 1: Migration `0003` and the spec DDL

**Files:**
- Create: `backend/internal/store/migrations/0003_pet_verdict_dates.up.sql`
- Create: `backend/internal/store/migrations/0003_pet_verdict_dates.down.sql`
- Modify: `backend/internal/store/migrations_test.go:117-135, 225-245`
- Modify: `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` (the ```sql block ending with `google_sync`)

- [ ] **Step 1: Extend the failing tests**

In `migrations_test.go`, change the expected list in `TestMigrateAppliesPendingVersions`:
```go
	if want := []string{"0001_init", "0002_google_sync", "0003_pet_verdict_dates"}; !reflect.DeepEqual(got, want) {
```
Append after `TestMigration0002CreatesGoogleSync`:
```go
func TestMigration0003AddsPetVerdictDates(t *testing.T) {
	up := readMigration(t, "0003_pet_verdict_dates.up.sql")
	for _, w := range []string{
		"ALTER TABLE pet_states",
		"ADD COLUMN last_target_met_date DATE",
		"ADD COLUMN judged_through DATE",
	} {
		if !strings.Contains(up, w) {
			t.Errorf("0003_pet_verdict_dates.up.sql is missing %q", w)
		}
	}
	down := readMigration(t, "0003_pet_verdict_dates.down.sql")
	for _, w := range []string{"DROP COLUMN IF EXISTS last_target_met_date", "DROP COLUMN IF EXISTS judged_through"} {
		if !strings.Contains(down, w) {
			t.Errorf("0003_pet_verdict_dates.down.sql is missing %q", w)
		}
	}
}
```

- [ ] **Step 2: Run and confirm they fail**

```sh
go test ./internal/store/... -run 'AppliesPendingVersions|Migration0003' -v
```
Expected: `TestMigrateAppliesPendingVersions` fails on the list; `TestMigration0003AddsPetVerdictDates` fails in `readMigration` (file missing).

- [ ] **Step 3: Write the migration**

`0003_pet_verdict_dates.up.sql`:
```sql
-- Migration 0003 — the pet's own once-per-day verdict markers (pet day-judgement plan).
-- last_target_met_date: the local YYYY-MM-DD whose §8 success (+20, streak+1) was
--   applied last; Service.OnTargetMet writes it under a conditional UPDATE so a
--   second call for the same day is a no-op whatever quests' Redis counter says.
-- judged_through: the latest local YYYY-MM-DD that can no longer be penalised —
--   its miss was applied, it was spared, or a passed revival resolved it. The
--   hourly sweep and Revive write it, also conditionally.
-- Both are NULL for existing rows; the sweep initialises judged_through on first contact.

ALTER TABLE pet_states
    ADD COLUMN last_target_met_date DATE,
    ADD COLUMN judged_through DATE;
```
`0003_pet_verdict_dates.down.sql`:
```sql
-- Reverse of 0003_pet_verdict_dates.up.sql.

ALTER TABLE pet_states
    DROP COLUMN IF EXISTS last_target_met_date,
    DROP COLUMN IF EXISTS judged_through;
```

- [ ] **Step 4: Append the same DDL to the backend spec**

In the backend spec, find the end of the `google_sync` block (`grep -n 'synced_at TIMESTAMP' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md"`) and insert, after its closing `);` and before the closing ```` ``` ````:
```sql

-- Added by migration 0003 (pet day-judgement): the pet's own once-per-day verdict markers.
ALTER TABLE pet_states
    ADD COLUMN last_target_met_date DATE,
    ADD COLUMN judged_through DATE;
```
The statement must be byte-identical to the migration's `ALTER TABLE` (AGENTS.md). Check: `grep -c 'ADD COLUMN judged_through DATE;' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md" internal/store/migrations/0003_pet_verdict_dates.up.sql` (run from the repo root) → `1` for each.

- [ ] **Step 5: Run and confirm they pass**

```sh
go test ./internal/store/... -count=1
```
Expected: `ok`.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend/internal/store/migrations/0003_pet_verdict_dates.up.sql backend/internal/store/migrations/0003_pet_verdict_dates.down.sql backend/internal/store/migrations_test.go "project-base/Adaptive English Learning Platform - Backend Technical Specification.md" && git commit -m "store: migration 0003 adds pet_states.last_target_met_date and judged_through" && cd backend
```

---

### Task 2: Engine — the state carries its own verdict dates

**Files:**
- Modify: `backend/internal/pet/engine.go`
- Modify: `backend/internal/pet/engine_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `engine_test.go`:
```go
func TestApplyTargetMetStampsTheLocalDateItWasAppliedFor(t *testing.T) {
	got := ApplyTargetMet(State{HealthPoints: 80, CurrentStreak: 4}, sept22, "2026-09-22")
	if got.LastTargetMetDate == nil || *got.LastTargetMetDate != "2026-09-22" {
		t.Errorf("LastTargetMetDate = %v, want 2026-09-22", got.LastTargetMetDate)
	}
	if got.JudgedThrough != nil {
		t.Errorf("JudgedThrough = %v, want untouched (success does not judge the day)", *got.JudgedThrough)
	}
}

func TestApplyMissRecordsTheJudgedDayAndNeverMovesItBackwards(t *testing.T) {
	later := "2026-09-25"
	got := ApplyMiss(State{HealthPoints: 100, JudgedThrough: &later}, sept22, "2026-09-22")
	if got.HealthPoints != 70 || *got.JudgedThrough != "2026-09-25" {
		t.Errorf("got %d / %s, want 70 and judged_through kept at 2026-09-25", got.HealthPoints, *got.JudgedThrough)
	}
	got = ApplyMiss(State{HealthPoints: 100}, sept22, "2026-09-22")
	if got.JudgedThrough == nil || *got.JudgedThrough != "2026-09-22" {
		t.Errorf("JudgedThrough = %v, want 2026-09-22", got.JudgedThrough)
	}
}

func TestApplyReviveResolvesTheDayWithoutCountingItAsASuccess(t *testing.T) {
	got := ApplyRevive(State{HealthPoints: 0, Stage: StageWilted}, sept22, "2026-09-22")
	if got.HealthPoints != 50 || got.Stage != StageSprout || got.CurrentStreak != 0 {
		t.Errorf("got %+v, want 50/sprout/0", got)
	}
	if got.JudgedThrough == nil || *got.JudgedThrough != "2026-09-22" {
		t.Errorf("JudgedThrough = %v, want 2026-09-22 — a passed revival resolves its day", got.JudgedThrough)
	}
	if got.LastTargetMetDate != nil || got.LastPracticedAt != nil {
		t.Error("a revival must not look like a met target: LastTargetMetDate and LastPracticedAt stay nil")
	}
}

func TestPreviousDateIsCivilArithmetic(t *testing.T) {
	for in, want := range map[string]string{
		"2026-09-23": "2026-09-22",
		"2026-09-01": "2026-08-31",
		"2026-03-01": "2026-02-28",
		"2028-03-01": "2028-02-29",
		"2027-01-01": "2026-12-31",
	} {
		if got := PreviousDate(in); got != want {
			t.Errorf("PreviousDate(%s) = %s, want %s", in, got, want)
		}
	}
}
```
`sept22` is declared in `service_test.go` (same package). If `engine_test.go` does not already import `time`, it does not need to — `sept22` is a package-level var.

- [ ] **Step 2: Run and confirm they fail**

```sh
go test ./internal/pet/... -run 'ApplyTargetMetStamps|ApplyMissRecords|ApplyReviveResolves|PreviousDate' 2>&1 | head -20
```
Expected: build failure — `too many arguments`, `undefined: PreviousDate`, unknown fields.

- [ ] **Step 3: Implement**

In `engine.go`, extend `State`:
```go
// State is the pet_states row (§3.2 + migration 0003) minus its ids.
type State struct {
	PlantName       string
	HealthPoints    int
	Stage           string
	CurrentStreak   int
	LastPracticedAt *time.Time
	UpdatedAt       time.Time
	// LastTargetMetDate is the local YYYY-MM-DD whose §8 success was applied
	// last — the pet's own once-per-day guard. nil until the first target is met.
	LastTargetMetDate *string
	// JudgedThrough is the latest local YYYY-MM-DD that can no longer be
	// penalised: its miss was applied, it was spared, or a revival resolved it.
	// nil for a pet the sweep has never seen.
	JudgedThrough *string
}
```
Replace the three `Apply*` functions and add the helpers:
```go
// ApplyTargetMet is §8's success logic for localDate, applied once per local
// day (Repo.SaveTargetMet enforces the once): +20 capped at 100, streak+1,
// last_practiced_at = now, last_target_met_date = localDate.
func ApplyTargetMet(s State, now time.Time, localDate string) State {
	s.HealthPoints = min(MaxHealth, s.HealthPoints+TargetMetHealthBonus)
	s.CurrentStreak++
	s.Stage = StageFor(s.HealthPoints, s.CurrentStreak)
	t := now
	s.LastPracticedAt = &t
	s.LastTargetMetDate = &localDate
	s.UpdatedAt = now
	return s
}

// ApplyMiss is §8's inactivity logic for the local day judged, run by the
// hourly sweep when that day stayed under 1800s: -30 floored at 0, wilted at
// 0, streak reset (§8 is silent on the streak; a streak with a missed day in
// it is not a streak), and judged_through advanced to judged. The real repo
// performs this arithmetic in SQL (PgRepo.PenaliseMiss); this Go form is the
// reference the fake repo and the integration test hold it to.
func ApplyMiss(s State, now time.Time, judged string) State {
	s.HealthPoints = max(0, s.HealthPoints-MissPenalty)
	s.CurrentStreak = 0
	s.Stage = StageFor(s.HealthPoints, s.CurrentStreak)
	s.JudgedThrough = laterDate(s.JudgedThrough, judged)
	s.UpdatedAt = now
	return s
}

// ApplyRevive is the §6.3 pass: health 50, sprout, streak 0 — and the local
// day it was passed on is resolved (judged_through = localDate), so that
// night's sweep does not take the §8 penalty out of the 50 (plan decision 4).
// It is not a success: last_practiced_at and last_target_met_date are untouched,
// so reaching 1800s later the same day still earns the +20.
func ApplyRevive(s State, now time.Time, localDate string) State {
	s.HealthPoints = ReviveHealth
	s.CurrentStreak = 0
	s.Stage = StageFor(s.HealthPoints, s.CurrentStreak)
	s.JudgedThrough = laterDate(s.JudgedThrough, localDate)
	s.UpdatedAt = now
	return s
}

// laterDate keeps a verdict date monotonic. YYYY-MM-DD strings order lexically.
func laterDate(cur *string, d string) *string {
	if cur != nil && *cur > d {
		return cur
	}
	return &d
}

// PreviousDate is the civil day before a YYYY-MM-DD. It is pure calendar
// arithmetic — no timezone, no instant — so it is unaffected by a zone whose
// local midnight does not exist (spring-forward at 00:00) or exists twice.
func PreviousDate(date string) string {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		// Callers pass quests.LocalDate output; a bad value would judge the
		// wrong day silently, so fail loudly.
		panic(fmt.Sprintf("pet: PreviousDate got a malformed date %q", date))
	}
	return d.AddDate(0, 0, -1).Format("2006-01-02")
}
```
Add `"fmt"` to the import block. The existing `engine_test.go` calls to `ApplyTargetMet(st, now)`/`ApplyMiss(st, now)`/`ApplyRevive(st, now)` gain a third argument (`"2026-09-22"`); `service.go`, `fakes_test.go`, `service_test.go` and `integration_test.go` will not compile until Tasks 3–5 — that is expected; run only `-run` filters until then, or build the package with `go vet ./internal/pet/ 2>&1 | head` to see what remains.

- [ ] **Step 4: Run the engine tests**

```sh
go test ./internal/pet/... -run 'StageFor|ApplyTargetMet|ApplyMiss|ApplyRevive|PreviousDate|Spec8' -v 2>&1 | tail -25
```
Expected: the package still fails to *build* because `service.go` calls the old signatures. Fix `service.go`'s three call sites minimally now so the package builds — `ApplyTargetMet(st, s.now(), localDate)`, `ApplyRevive(st, now, today)`, and in `Sweep` `ApplyMiss(c.State, now, yesterday)` — then re-run. Expected: `--- PASS` for every engine test. (Fakes also need `defaultState` untouched; they compile.)

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/pet/engine.go backend/internal/pet/engine_test.go backend/internal/pet/service.go && git commit -m "pet: State carries last_target_met_date and judged_through; Apply* take the local date; PreviousDate" && cd backend
```

---

### Task 3: Repo — conditional verdict writes, all pets as candidates

**Files:**
- Modify: `backend/internal/pet/repo.go`
- Modify: `backend/internal/pet/integration_test.go`

- [ ] **Step 1: Write the failing integration test**

Append to `integration_test.go` (same gating as the existing test — it skips without `TEST_DATABASE_URL`, CI runs it):
```go
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
```
Add `"fmt"` to the file's imports. In the **existing** `TestIntegrationEnsureCreatesExactlyOnePetRow`: change `svc.OnTargetMet(ctx, userID, "2026-09-22")` expectations to also check `st.LastTargetMetDate != nil`; replace the `for i := 0; i < 4; i++ { repo.Save(ctx, userID, ApplyMiss(st, time.Now())) … }` loop with
```go
	for i := 0; i < 4; i++ {
		if _, err := repo.PenaliseMiss(ctx, userID, fmt.Sprintf("2026-09-2%d", 3+i), time.Now()); err != nil {
			t.Fatalf("PenaliseMiss %d: %v", i+1, err)
		}
	}
	st, _ = repo.Get(ctx, userID)
```
delete the `repo.Timezones` block, and change `repo.SweepCandidates(ctx, []string{"Asia/Ho_Chi_Minh"})` to `repo.SweepCandidates(ctx)`.

- [ ] **Step 2: Confirm it fails to build**

```sh
go vet ./internal/pet/ 2>&1 | head
```
Expected: `repo.SaveTargetMet undefined`, `repo.PenaliseMiss undefined`, `repo.MarkJudged undefined`, `too many arguments` on `SweepCandidates`.

- [ ] **Step 3: Implement the repo**

Replace the `Repo` interface and the SQL constants in `repo.go`:
```go
// Repo is the Postgres side. Everything but Timezone/SweepCandidates' join
// touches pet_states (this package's table); users.timezone is the shared
// root table, read the same way quests reads it for day_number.
//
// The three verdict writers are conditional UPDATEs: the predicate on the
// verdict date is what makes OnTargetMet and Sweep safe to run twice, from
// two processes, for the same local day. applied == false is "already done",
// never an error.
type Repo interface {
	// Ensure creates the 1:1 row idempotently: INSERT ... ON CONFLICT DO NOTHING.
	Ensure(ctx context.Context, userID string) error
	Get(ctx context.Context, userID string) (State, error)
	// Save writes every mutable column unconditionally (judged_through only
	// ever forwards). Revive uses it; the two verdict writers below do not.
	Save(ctx context.Context, userID string, s State) error
	// SaveTargetMet writes s only while the row's last_target_met_date is
	// NULL or before s.LastTargetMetDate.
	SaveTargetMet(ctx context.Context, userID string, s State) (applied bool, err error)
	// PenaliseMiss applies §8's inactivity arithmetic in SQL and advances
	// judged_through to judged, only while judged_through is NULL or earlier.
	PenaliseMiss(ctx context.Context, userID, judged string, now time.Time) (applied bool, err error)
	// MarkJudged advances judged_through to judged without touching health —
	// the spared day. Same predicate as PenaliseMiss.
	MarkJudged(ctx context.Context, userID, judged string) (applied bool, err error)
	Timezone(ctx context.Context, userID string) (string, error)
	// SweepCandidates returns every pet with its user's timezone; the sweep
	// decides per pet, by civil date, whether there is anything to judge.
	SweepCandidates(ctx context.Context) ([]Candidate, error)
}

const (
	ensureSQL = `INSERT INTO pet_states (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`

	// Every pet_states column except user_id is nullable in §3.2 (DEFAULT
	// without NOT NULL), so COALESCE to the DDL defaults. The two DATE
	// columns are read as YYYY-MM-DD text: the same string quests.LocalDate
	// produces and the service compares.
	stateColumns = `COALESCE(p.plant_name, 'My Green Buddy'), COALESCE(p.health_points, 100), COALESCE(p.stage::text, 'sprout'),
	COALESCE(p.current_streak, 0), p.last_practiced_at, COALESCE(p.updated_at, CURRENT_TIMESTAMP),
	to_char(p.last_target_met_date, 'YYYY-MM-DD'), to_char(p.judged_through, 'YYYY-MM-DD')`

	getSQL = `SELECT ` + stateColumns + ` FROM pet_states p WHERE p.user_id = $1`

	// GREATEST ignores NULL, so a NULL judged_through takes $8 and a later one is kept.
	saveSQL = `
UPDATE pet_states
SET health_points = $2, stage = $3::pet_stage, current_streak = $4, last_practiced_at = $5, updated_at = $6,
    last_target_met_date = $7::date, judged_through = GREATEST(judged_through, $8::date)
WHERE user_id = $1`

	saveTargetMetSQL = `
UPDATE pet_states
SET health_points = $2, stage = $3::pet_stage, current_streak = $4, last_practiced_at = $5, updated_at = $6,
    last_target_met_date = $7::date
WHERE user_id = $1 AND (last_target_met_date IS NULL OR last_target_met_date < $7::date)`

	// Pre-image and write in one statement: health_points on the right-hand
	// side is the row's current value. The CASE is StageFor(health, 0) for
	// the two stages a streak of 0 can produce; the integration test pins it
	// to ApplyMiss.
	penaliseMissSQL = `
UPDATE pet_states
SET health_points = GREATEST(0, COALESCE(health_points, 100) - $3),
    current_streak = 0,
    stage = (CASE WHEN COALESCE(health_points, 100) - $3 <= 0 THEN 'wilted' ELSE 'sprout' END)::pet_stage,
    judged_through = $2::date,
    updated_at = $4
WHERE user_id = $1 AND (judged_through IS NULL OR judged_through < $2::date)`

	markJudgedSQL = `
UPDATE pet_states
SET judged_through = $2::date
WHERE user_id = $1 AND (judged_through IS NULL OR judged_through < $2::date)`

	timezoneSQL = `SELECT COALESCE(timezone, 'UTC') FROM users WHERE id = $1`

	candidatesSQL = `
SELECT p.user_id, COALESCE(u.timezone, 'UTC'), ` + stateColumns + `
FROM pet_states p JOIN users u ON u.id = p.user_id`
)
```
Update `scanState` to scan the two new columns:
```go
func scanState(row pgx.Row, dst ...any) (State, error) {
	var s State
	var last *time.Time
	targets := append(dst, &s.PlantName, &s.HealthPoints, &s.Stage, &s.CurrentStreak, &last, &s.UpdatedAt, &s.LastTargetMetDate, &s.JudgedThrough)
	if err := row.Scan(targets...); err != nil {
		return State{}, err
	}
	if last != nil {
		u := last.UTC()
		s.LastPracticedAt = &u
	}
	s.UpdatedAt = s.UpdatedAt.UTC()
	return s, nil
}
```
Replace `Save`, delete `Timezones`, replace `SweepCandidates`, and add the three writers:
```go
func (r *PgRepo) Save(ctx context.Context, userID string, s State) error {
	tag, err := r.Pool.Exec(ctx, saveSQL, userID, s.HealthPoints, s.Stage, s.CurrentStreak, s.LastPracticedAt, s.UpdatedAt, s.LastTargetMetDate, s.JudgedThrough)
	if err != nil {
		return fmt.Errorf("pet: saving pet_states: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNoPet
	}
	return nil
}

func (r *PgRepo) SaveTargetMet(ctx context.Context, userID string, s State) (bool, error) {
	if s.LastTargetMetDate == nil {
		return false, errors.New("pet: SaveTargetMet needs LastTargetMetDate — build the state with ApplyTargetMet")
	}
	tag, err := r.Pool.Exec(ctx, saveTargetMetSQL, userID, s.HealthPoints, s.Stage, s.CurrentStreak, s.LastPracticedAt, s.UpdatedAt, *s.LastTargetMetDate)
	if err != nil {
		return false, fmt.Errorf("pet: saving target met: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PgRepo) PenaliseMiss(ctx context.Context, userID, judged string, now time.Time) (bool, error) {
	tag, err := r.Pool.Exec(ctx, penaliseMissSQL, userID, judged, MissPenalty, now)
	if err != nil {
		return false, fmt.Errorf("pet: penalising miss: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PgRepo) MarkJudged(ctx context.Context, userID, judged string) (bool, error) {
	tag, err := r.Pool.Exec(ctx, markJudgedSQL, userID, judged)
	if err != nil {
		return false, fmt.Errorf("pet: marking day judged: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PgRepo) SweepCandidates(ctx context.Context) ([]Candidate, error) {
	rows, err := r.Pool.Query(ctx, candidatesSQL)
	if err != nil {
		return nil, fmt.Errorf("pet: listing sweep candidates: %w", err)
	}
	defer rows.Close()
	var out []Candidate
	for rows.Next() {
		var c Candidate
		s, err := scanState(rows, &c.UserID, &c.Timezone)
		if err != nil {
			return nil, fmt.Errorf("pet: scanning candidate: %w", err)
		}
		c.State = s
		out = append(out, c)
	}
	return out, rows.Err()
}
```
Note: a `nil *string` bound to `$7::date` is a SQL NULL; a non-nil one is text that Postgres casts to `date` — the same pattern quests' `upsertProgressSQL` already relies on with `$2::date`. `SaveTargetMet` rejects a nil date rather than silently writing NULL and passing the predicate forever.

Then make `service.go` compile against the new interface with the smallest edits — Tasks 5–6 replace these lines properly:
- in `Sweep`, delete the `Timezones` loop and the `atMidnight` variable, call `s.repo.SweepCandidates(ctx)`, and replace `s.repo.Save(ctx, c.UserID, ApplyMiss(c.State, now))` with `if _, err := s.repo.PenaliseMiss(ctx, c.UserID, yesterday, now); err != nil {` (keep the surrounding error handling and `penalised++`).

- [ ] **Step 4: Build; run the integration test if a stack is available (CI runs it regardless)**

```sh
go build ./...                              # expect: green — the non-test packages compile after this task
go vet ./internal/pet/ 2>&1 | head          # expect: only the test fakes fail (old interface) — Task 4 fixes them
# with TEST_DATABASE_URL exported, after Task 5:
go test ./internal/pet/... -run 'Integration' -count=1 -timeout 120s -v 2>&1 | tail -20
```
Expected after Task 5: `--- PASS: TestIntegrationVerdictWritesAreConditional` and the updated `TestIntegrationEnsureCreatesExactlyOnePetRow`. Without a stack: `--- SKIP`. Do not commit a red integration test: if you cannot run it locally, read the SQL twice and rely on CI's `backend-integration` job — it fails the branch on any SKIP.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/pet/repo.go backend/internal/pet/service.go backend/internal/pet/integration_test.go && git commit -m "pet: conditional verdict writes (SaveTargetMet, PenaliseMiss, MarkJudged); SweepCandidates returns every pet" && cd backend
```

---

### Task 4: Fakes that can fail per user, tick a clock, and race

**Files:**
- Modify: `backend/internal/pet/fakes_test.go`

- [ ] **Step 1: Rewrite `fakeRepo` and `fakeStudy`**

Replace the `fakeRepo` type, its constructor and methods, and `fakeStudy`, with:
```go
type fakeRepo struct {
	mu         sync.Mutex
	states     map[string]State  // by userID; present == row exists
	timezones  map[string]string // by userID; missing == "UTC"
	now        func() time.Time  // stamps a fresh row's updated_at like the DDL's CURRENT_TIMESTAMP
	ensured    int
	saved      int // applied writes across Save, SaveTargetMet, PenaliseMiss, MarkJudged
	ensureErr  error
	saveErr    error
	saveErrFor map[string]error // per-user failure of any writer; nil == fine
	timezoneErr error
}

func newFakeRepo(now func() time.Time) *fakeRepo {
	return &fakeRepo{states: map[string]State{}, timezones: map[string]string{}, now: now, saveErrFor: map[string]error{}}
}

// defaultState mirrors the §3.2 column defaults a fresh INSERT produces —
// including updated_at = CURRENT_TIMESTAMP, which the sweep's first-contact
// rule reads.
func defaultState(now time.Time) State {
	return State{PlantName: "My Green Buddy", HealthPoints: 100, Stage: StageSprout, UpdatedAt: now}
}

func (f *fakeRepo) writeErr(userID string) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	return f.saveErrFor[userID]
}

func (f *fakeRepo) Ensure(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ensureErr != nil {
		return f.ensureErr
	}
	f.ensured++
	if _, ok := f.states[userID]; !ok {
		f.states[userID] = defaultState(f.now())
	}
	return nil
}

func (f *fakeRepo) Get(_ context.Context, userID string) (State, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.states[userID]
	if !ok {
		return State{}, ErrNoPet
	}
	return s, nil
}

func (f *fakeRepo) Save(_ context.Context, userID string, s State) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.writeErr(userID); err != nil {
		return err
	}
	cur, ok := f.states[userID]
	if !ok {
		return ErrNoPet
	}
	f.saved++
	s.PlantName = cur.PlantName
	if s.JudgedThrough != nil {
		s.JudgedThrough = laterDate(cur.JudgedThrough, *s.JudgedThrough) // GREATEST(judged_through, $8)
	} else {
		s.JudgedThrough = cur.JudgedThrough
	}
	f.states[userID] = s
	return nil
}

// dateBefore is the SQL predicate `col IS NULL OR col < $d`.
func dateBefore(col *string, d string) bool { return col == nil || *col < d }

func (f *fakeRepo) SaveTargetMet(_ context.Context, userID string, s State) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.writeErr(userID); err != nil {
		return false, err
	}
	cur, ok := f.states[userID]
	if !ok || s.LastTargetMetDate == nil || !dateBefore(cur.LastTargetMetDate, *s.LastTargetMetDate) {
		return false, nil
	}
	f.saved++
	s.PlantName = cur.PlantName
	s.JudgedThrough = cur.JudgedThrough
	f.states[userID] = s
	return true, nil
}

func (f *fakeRepo) PenaliseMiss(_ context.Context, userID, judged string, now time.Time) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.writeErr(userID); err != nil {
		return false, err
	}
	cur, ok := f.states[userID]
	if !ok || !dateBefore(cur.JudgedThrough, judged) {
		return false, nil
	}
	f.saved++
	f.states[userID] = ApplyMiss(cur, now, judged)
	return true, nil
}

func (f *fakeRepo) MarkJudged(_ context.Context, userID, judged string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.writeErr(userID); err != nil {
		return false, err
	}
	cur, ok := f.states[userID]
	if !ok || !dateBefore(cur.JudgedThrough, judged) {
		return false, nil
	}
	f.saved++
	cur.JudgedThrough = &judged
	f.states[userID] = cur
	return true, nil
}

func (f *fakeRepo) Timezone(_ context.Context, userID string) (string, error) {
	if f.timezoneErr != nil {
		return "", f.timezoneErr
	}
	if tz, ok := f.timezones[userID]; ok {
		return tz, nil
	}
	return "UTC", nil
}

func (f *fakeRepo) SweepCandidates(context.Context) ([]Candidate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Candidate
	for userID, s := range f.states {
		tz, _ := f.Timezone(context.Background(), userID)
		out = append(out, Candidate{UserID: userID, Timezone: tz, State: s})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UserID < out[j].UserID })
	return out, nil
}
```
and
```go
type fakeStudy struct {
	totals map[string]int64 // keyed by userID|localDate
	err    error            // every read fails
	errFor map[string]error // one user's reads fail; the rest are served
}

func newFakeStudy() *fakeStudy { return &fakeStudy{totals: map[string]int64{}, errFor: map[string]error{}} }

func (f *fakeStudy) set(userID, localDate string, seconds int64) {
	f.totals[userID+"|"+localDate] = seconds
}

func (f *fakeStudy) Total(_ context.Context, userID, localDate string) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	if err := f.errFor[userID]; err != nil {
		return 0, err
	}
	return f.totals[userID+"|"+localDate], nil
}
```
Add `"sync"` to the imports. `fakeChallenges`, `errBoom`, `fixedClock` are unchanged. In `service_test.go`, `newHarness` becomes:
```go
func newHarness(now time.Time) *harness {
	h := &harness{repo: newFakeRepo(fixedClock(now)), challenges: newFakeChallenges(), study: newFakeStudy()}
	h.svc = NewService(h.repo, h.challenges, h.study, fixedClock(now))
	return h
}
```
`sort` is already imported. Note `SweepCandidates` calls `Timezone` while holding `mu` — `Timezone` does not lock, so this is fine; keep it that way.

- [ ] **Step 2: Build the test binary**

```sh
go vet ./internal/pet/ 2>&1 | head
```
Expected: clean, or only errors in `service_test.go`'s old sweep tests (they reference behaviour Task 6 replaces) — if so, fix `fakes_test.go` errors now and leave the sweep tests for Task 6.

- [ ] **Step 3: Commit**

```sh
cd .. && git add backend/internal/pet/fakes_test.go backend/internal/pet/service_test.go && git commit -m "pet: test fakes with a clock, per-user error hooks, a mutex and the conditional verdict writers" && cd backend
```

---

### Task 5: Service — `OnTargetMet` owns its once, `Revive` resolves its day

**Files:**
- Modify: `backend/internal/pet/service.go:49-116`
- Modify: `backend/internal/pet/service_test.go`

- [ ] **Step 1: Write the failing tests**

In `service_test.go`, update `TestOnTargetMetAppliesSpec8SuccessOnceAndPersists` to also assert the marker, and add four tests:
```go
	// (inside TestOnTargetMetAppliesSpec8SuccessOnceAndPersists, after the LastPracticedAt check)
	if got.LastTargetMetDate == nil || *got.LastTargetMetDate != "2026-09-22" {
		t.Errorf("LastTargetMetDate = %v, want 2026-09-22", got.LastTargetMetDate)
	}
```
```go
func TestOnTargetMetTwiceForTheSameLocalDateBumpsOnce(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{PlantName: "Fern", HealthPoints: 80, CurrentStreak: 4, Stage: StageSapling}

	for i := 0; i < 2; i++ {
		if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-22"); err != nil {
			t.Fatalf("call %d: %v — a repeat must be a silent no-op, not an error", i+1, err)
		}
	}
	got := h.repo.states["u1"]
	if got.HealthPoints != 100 || got.CurrentStreak != 5 {
		t.Errorf("state = %+v, want 100/5 — the second call for the same day double-bumped", got)
	}
	if h.repo.saved != 1 {
		t.Errorf("saved %d times, want 1", h.repo.saved)
	}
}

func TestOnTargetMetForTheNextLocalDateBumpsAgain(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 40, CurrentStreak: 4, Stage: StageSapling}

	if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-22"); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-23"); err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["u1"]; got.HealthPoints != 80 || got.CurrentStreak != 6 || *got.LastTargetMetDate != "2026-09-23" {
		t.Errorf("state = %+v, want 80/6 marked 2026-09-23 — the guard must not simply never bump", got)
	}
}

func TestOnTargetMetForAnEarlierLocalDateIsIgnored(t *testing.T) {
	// A user who moves their timezone west can make "today" an earlier date
	// than the one already counted; the marker is monotonic, so no re-earn.
	h := newHarness(sept22)
	d := "2026-09-23"
	h.repo.states["u1"] = State{HealthPoints: 40, CurrentStreak: 1, Stage: StageSprout, LastTargetMetDate: &d}

	if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-22"); err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["u1"]; got.HealthPoints != 40 || h.repo.saved != 0 {
		t.Errorf("state = %+v saved=%d, want untouched", got, h.repo.saved)
	}
}

func TestRevivePassResolvesTheLocalDayItWasPassedOn(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}
	h.study.set("u1", "2026-09-22", 0)
	if _, err := h.svc.Revive(ctx, "u1"); err != nil {
		t.Fatalf("start: %v", err)
	}
	h.study.set("u1", "2026-09-22", 900)
	out, err := h.svc.Revive(ctx, "u1")
	if err != nil || !out.Passed {
		t.Fatalf("pass: out=%+v err=%v", out, err)
	}
	got := h.repo.states["u1"]
	if got.JudgedThrough == nil || *got.JudgedThrough != "2026-09-22" {
		t.Errorf("JudgedThrough = %v, want 2026-09-22 (plan decision 4: a passed revival resolves its day)", got.JudgedThrough)
	}
	if got.LastTargetMetDate != nil || got.LastPracticedAt != nil {
		t.Error("a revival is not a met target")
	}
	// The +20 for a full 30 minutes the same day is still available.
	if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-22"); err != nil {
		t.Fatal(err)
	}
	if got = h.repo.states["u1"]; got.HealthPoints != 70 || got.CurrentStreak != 1 {
		t.Errorf("after revive then target met = %+v, want 70/1", got)
	}
}

func TestReviveSurfacesATimezoneReadFailure(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}
	h.repo.timezoneErr = errBoom
	if _, err := h.svc.Revive(ctx, "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want errBoom", err)
	}
	if h.challenges.started != 0 {
		t.Error("a challenge was started without knowing the user's local day")
	}
}
```

- [ ] **Step 2: Run and confirm they fail**

```sh
go test ./internal/pet/... -run 'OnTargetMet|RevivePassResolves|ReviveSurfaces' -v 2>&1 | tail -30
```
Expected: `TestOnTargetMetTwice…` fails with `100/6`, `TestRevivePassResolves…` fails with `JudgedThrough = <nil>`, `TestOnTargetMetForAnEarlierLocalDateIsIgnored` fails with health 60. `TestReviveSurfacesATimezoneReadFailure` passes already (`Revive` returns that error today) — it is the coverage the tests finding asked for, kept.

- [ ] **Step 3: Implement `OnTargetMet` and `Revive`**

Replace `OnTargetMet`:
```go
// OnTargetMet is §8's success logic for the user's local day localDate:
// quests fires it when the day's total first reaches 1800s (backend spec
// §6.2), and may fire it again after a failure on the same call or after a
// lost Redis counter. The pet owns the once: Repo.SaveTargetMet's predicate
// on last_target_met_date refuses a second write for the same (or an
// earlier) local date, and that refusal is a silent no-op — quests logs hook
// errors, and "already counted" is not one. There is deliberately no Go-side
// pre-check: one mechanism, in the database, is what the tests pin.
func (s *Service) OnTargetMet(ctx context.Context, userID, localDate string) error {
	st, err := s.Ensure(ctx, userID)
	if err != nil {
		return err
	}
	_, err = s.repo.SaveTargetMet(ctx, userID, ApplyTargetMet(st, s.now(), localDate))
	return err
}
```
In `Revive`, change the pass:
```go
	st = ApplyRevive(st, now, today)
```
(`today` is already computed above it.) Update the `Revive` doc comment's last sentence: "On pass the state becomes 50 / sprout / 0 (§6.3), the local day is resolved so that night's sweep applies no miss for it (judged_through = today), and the challenge is cleared." The Task 3 stub in `Sweep` stays until Task 6.

- [ ] **Step 4: Run the target-met and revive tests**

```sh
go test ./internal/pet/... -run 'OnTargetMet|Revive|QuestHook|Ensure' -v 2>&1 | tail -30
```
Expected: `--- PASS` for all of them (the existing revive tests still pass: `ApplyRevive` only adds a field).

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/pet/service.go backend/internal/pet/service_test.go && git commit -m "pet: OnTargetMet is once per local date by its own marker; a passed revival resolves its day" && cd backend
```

---

### Task 6: Sweep — civil dates, conditional writes, and a suite that can fail

**Files:**
- Modify: `backend/internal/pet/service.go` (`Sweep` and its comment)
- Modify: `backend/internal/pet/service_test.go` (replace every `TestSweep*`)

- [ ] **Step 1: Replace the sweep tests**

Delete `TestSweepPenalisesOnlyUsersAtLocalMidnightWhoMissedYesterday`, `TestSweepAtSeventeenUTCCatchesHoChiMinhMidnight`, `TestSweepIsIdempotentWithinTheSameLocalDay`, `TestSweepWiltsAfterFourMissedDays` and `TestSweepSkipsAUnreadableCounterAndContinues`, and add:
```go
// judgedThrough seeds a pet the sweep has already seen, so tests are not
// exercising the first-contact rule unless they mean to.
func judgedThrough(d string) *string { return &d }

func TestSweepJudgesEachPetsOwnLocalYesterday(t *testing.T) {
	h := newHarness(midnite) // 00:00Z on the 23rd: UTC just ended the 22nd; Ho Chi Minh (UTC+7) ended it seven hours ago
	seed := func(user, tz string, health, streak int, through string) {
		h.repo.states[user] = State{HealthPoints: health, CurrentStreak: streak, Stage: StageFor(health, streak), UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough(through)}
		h.repo.timezones[user] = tz
	}
	seed("utc-missed", "UTC", 100, 9, "2026-09-21")
	seed("utc-met-marker", "UTC", 100, 9, "2026-09-21")
	seed("utc-met-counter", "UTC", 100, 9, "2026-09-21")
	seed("utc-short", "UTC", 40, 1, "2026-09-21")
	seed("hcm-done", "Asia/Ho_Chi_Minh", 100, 9, "2026-09-22") // the 17:00Z tick already judged its 22nd
	seed("hcm-late", "Asia/Ho_Chi_Minh", 100, 9, "2026-09-21") // that tick was missed (a restart): catch up now
	d := "2026-09-22"
	st := h.repo.states["utc-met-marker"]
	st.LastTargetMetDate = &d
	h.repo.states["utc-met-marker"] = st
	h.study.set("utc-met-counter", "2026-09-22", 1800)
	h.study.set("utc-short", "2026-09-22", 1799)

	n, err := h.svc.Sweep(ctx, midnite)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if n != 3 {
		t.Errorf("penalised %d, want 3 (utc-missed, utc-short, hcm-late)", n)
	}
	want := map[string]struct {
		health, streak int
		through        string
	}{
		"utc-missed":      {70, 0, "2026-09-22"},
		"utc-met-marker":  {100, 9, "2026-09-22"}, // spared by its own marker, no counter needed
		"utc-met-counter": {100, 9, "2026-09-22"}, // spared by the counter fallback, day recorded
		"utc-short":       {10, 0, "2026-09-22"},  // 1799s is a miss
		"hcm-done":        {100, 9, "2026-09-22"}, // nothing to do
		"hcm-late":        {70, 0, "2026-09-22"},  // judged at 07:00 local, one tick is never lost
	}
	for user, w := range want {
		got := h.repo.states[user]
		if got.HealthPoints != w.health || got.CurrentStreak != w.streak || got.JudgedThrough == nil || *got.JudgedThrough != w.through {
			t.Errorf("%s = %+v, want %d/%d judged through %s", user, got, w.health, w.streak, w.through)
		}
	}
	if !h.repo.states["utc-missed"].UpdatedAt.Equal(midnite) {
		t.Error("a penalised pet must be stamped with the sweep's clock")
	}
}

func TestSweepIsIdempotentAcrossTicksAndHours(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}

	total := 0
	for _, at := range []time.Time{midnite, midnite.Add(20 * time.Minute), midnite.Add(time.Hour), midnite.Add(13 * time.Hour)} {
		n, err := h.svc.Sweep(ctx, at)
		if err != nil {
			t.Fatalf("sweep at %v: %v", at, err)
		}
		total += n
	}
	if total != 1 || h.repo.states["u1"].HealthPoints != 70 {
		t.Errorf("four sweeps in one local day penalised %d times, health %d; want 1 and 70", total, h.repo.states["u1"].HealthPoints)
	}
}

func TestTwoConcurrentSweepsPenaliseOnce(t *testing.T) {
	h := newHarness(midnite)
	for i := 0; i < 20; i++ {
		h.repo.states[fmt.Sprintf("u%02d", i)] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	}
	var wg sync.WaitGroup
	counts := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := h.svc.Sweep(ctx, midnite)
			if err != nil {
				t.Errorf("Sweep: %v", err)
			}
			counts <- n
		}()
	}
	wg.Wait()
	close(counts)
	sum := 0
	for n := range counts {
		sum += n
	}
	if sum != 20 {
		t.Errorf("two concurrent sweeps reported %d penalties over 20 pets, want 20 — the count must follow the conditional write", sum)
	}
	for user, st := range h.repo.states {
		if st.HealthPoints != 70 {
			t.Errorf("%s health = %d, want 70 (penalised exactly once)", user, st.HealthPoints)
		}
	}
}

func TestSweepWiltsAfterFourMissedDays(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, CurrentStreak: 20, Stage: StageFruitful, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}

	for day := 0; day < 4; day++ {
		if _, err := h.svc.Sweep(ctx, midnite.AddDate(0, 0, day)); err != nil {
			t.Fatalf("day %d: %v", day, err)
		}
	}
	if got := h.repo.states["u1"]; got.HealthPoints != 0 || got.Stage != StageWilted || got.CurrentStreak != 0 || *got.JudgedThrough != "2026-09-25" {
		t.Errorf("after four misses = %+v, want 0/wilted/0 judged through 2026-09-25", got)
	}
}

func TestSweepDoesNotPenaliseADayResolvedByARevive(t *testing.T) {
	eight := time.Date(2026, time.September, 22, 20, 0, 0, 0, time.UTC) // 20:00 local (UTC) on the 22nd
	h := newHarness(eight)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted, UpdatedAt: eight.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	if _, err := h.svc.Revive(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	h.study.set("u1", "2026-09-22", 900) // the challenge, and nothing more: 900 < 1800
	if out, err := h.svc.Revive(ctx, "u1"); err != nil || !out.Passed {
		t.Fatalf("pass: %+v %v", out, err)
	}

	n, err := h.svc.Sweep(ctx, midnite) // 00:00 on the 23rd
	if err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["u1"]; n != 0 || got.HealthPoints != 50 || got.Stage != StageSprout {
		t.Errorf("after revive at 20:00 and the midnight sweep: n=%d state=%+v; want 0 and 50/sprout — the 15-minute challenge bought the day", n, got)
	}
	// The mirror case: the tick for the 22nd was missed (restart), the user
	// revives at 00:10 on the 23rd, the sweep runs late at 01:00. The plant
	// sat at 0 all through the 22nd, so that day's miss has nothing left to
	// take — it must not be taken from the 50 that came after (plan decision 4).
	h2 := newHarness(midnite.Add(10 * time.Minute))
	h2.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	if _, err := h2.svc.Revive(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	h2.study.set("u1", "2026-09-23", 900)
	if out, _ := h2.svc.Revive(ctx, "u1"); !out.Passed {
		t.Fatal("expected the pass")
	}
	if n, _ := h2.svc.Sweep(ctx, midnite.Add(time.Hour)); n != 0 || h2.repo.states["u1"].HealthPoints != 50 {
		t.Errorf("late sweep after a next-morning revival: n=%d health=%d, want 0 and 50", n, h2.repo.states["u1"].HealthPoints)
	}
}

func TestSweepStillJudgesYesterdayWhenTodaysTargetWasMetFirst(t *testing.T) {
	// The tick for the 22nd was missed; the user meets the 23rd's target at
	// 00:40 (a 1800s call at 00:00:30 counts toward the 23rd); the sweep runs
	// at 01:00. OnTargetMet moves last_target_met_date, not judged_through,
	// so the 22nd is still judged — and penalised.
	h := newHarness(midnite.Add(40 * time.Minute))
	h.repo.states["u1"] = State{HealthPoints: 100, CurrentStreak: 3, Stage: StageSapling, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-23"); err != nil {
		t.Fatal(err)
	}
	n, err := h.svc.Sweep(ctx, midnite.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["u1"]; n != 1 || got.HealthPoints != 70 || got.CurrentStreak != 0 || *got.LastTargetMetDate != "2026-09-23" || *got.JudgedThrough != "2026-09-22" {
		t.Errorf("n=%d %+v; want 1 and 70/0, last_target_met_date 2026-09-23, judged through 2026-09-22", n, got)
	}
}

func TestSweepFirstContactJudgesOnlyDaysThePetExisted(t *testing.T) {
	// Created at 00:10 on the 23rd (after the midnight tick), swept at 01:00:
	// there is no 22nd to judge; the marker is initialised instead.
	h := newHarness(midnite.Add(10 * time.Minute))
	if _, err := h.svc.Ensure(ctx, "late"); err != nil {
		t.Fatal(err)
	}
	n, err := h.svc.Sweep(ctx, midnite.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["late"]; n != 0 || got.HealthPoints != 100 || got.JudgedThrough == nil || *got.JudgedThrough != "2026-09-22" {
		t.Errorf("fresh pet after its first sweep: n=%d %+v; want untouched and judged through 2026-09-22", n, got)
	}
	// Created at 23:50 on the 22nd, swept at 00:00: it existed on the 22nd
	// (for ten minutes) and studied nothing — that is a miss, as today.
	h = newHarness(midnite.Add(-10 * time.Minute))
	if _, err := h.svc.Ensure(ctx, "early"); err != nil {
		t.Fatal(err)
	}
	if n, _ := h.svc.Sweep(ctx, midnite); n != 1 || h.repo.states["early"].HealthPoints != 70 {
		t.Errorf("pet created at 23:50: n=%d health=%d, want 1 and 70", n, h.repo.states["early"].HealthPoints)
	}
}

func TestSweepJudgesEveryLocalDayExactlyOnceInEveryZone(t *testing.T) {
	zones := []struct {
		tz   string
		from time.Time
	}{
		{"UTC", time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)},
		{"Asia/Ho_Chi_Minh", time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)},
		{"Asia/Kolkata", time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)},   // +05:30: midnight at :30
		{"Asia/Kathmandu", time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)}, // +05:45
		{"Pacific/Chatham", time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)}, // +12:45 / +13:45, DST 2026-09-27
		{"America/Havana", time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)},  // falls back 2026-11-01 01:00→00:00: local hour 0 twice
		{"America/Santiago", time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)},  // springs forward 2026-09-06 00:00→01:00: no local hour 0
	}
	for _, z := range zones {
		t.Run(z.tz, func(t *testing.T) {
			loc := quests.Location(z.tz)
			h := newHarness(z.from)
			h.repo.timezones["u"] = z.tz
			// Already judged through the day before the first tick's local
			// date, so every penalty below is one local-date change.
			first := quests.LocalDate(z.from, loc)
			h.repo.states["u"] = State{HealthPoints: 100, CurrentStreak: 5, Stage: StageSapling, UpdatedAt: z.from.Add(-96 * time.Hour), JudgedThrough: judgedThrough(PreviousDate(first))}

			seen := map[string]bool{first: true}
			penalised := 0
			for tick := z.from; tick.Before(z.from.Add(72 * time.Hour)); tick = tick.Add(time.Hour) {
				n, err := h.svc.Sweep(ctx, tick)
				if err != nil {
					t.Fatalf("sweep at %v: %v", tick, err)
				}
				if n > 1 {
					t.Errorf("tick %v penalised %d pets, there is one", tick, n)
				}
				penalised += n
				seen[quests.LocalDate(tick, loc)] = true
			}
			// Independent oracle: the number of local dates the ticks entered
			// after the first one — 2 for UTC (its first tick is already a
			// midnight), 3 for every other zone here. The oracle, not a
			// constant, decides, so a DST rule change cannot rot the test.
			want := len(seen) - 1
			if penalised != want {
				t.Errorf("penalised %d times over 72 hourly ticks, want %d (one per local day that ended)", penalised, want)
			}
			if got := h.repo.states["u"].HealthPoints; got != max(0, 100-30*want) {
				t.Errorf("health = %d, want %d", got, max(0, 100-30*want))
			}
		})
	}
}

func TestSweepStillJudgesAZoneWhoseMidnightDoesNotExist(t *testing.T) {
	// America/Santiago, 2026-09-06: clocks go 23:59:59 -04 → 01:00:00 -03.
	// The 04:00Z tick is 01:00 local; local hour 0 never happens that day.
	h := newHarness(time.Date(2026, 9, 6, 4, 0, 0, 0, time.UTC))
	h.repo.timezones["u"] = "America/Santiago"
	h.repo.states["u"] = State{HealthPoints: 100, UpdatedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), JudgedThrough: judgedThrough("2026-09-04")}

	n, err := h.svc.Sweep(ctx, time.Date(2026, 9, 6, 4, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["u"]; n != 1 || got.HealthPoints != 70 || *got.JudgedThrough != "2026-09-05" {
		t.Errorf("n=%d %+v; want 2026-09-05 judged at 01:00 local", n, got)
	}
}

func TestSweepContinuesPastAUserWhoseCounterIsUnreadable(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	h.repo.states["u2"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	h.study.errFor["u1"] = errBoom

	n, err := h.svc.Sweep(ctx, midnite)
	if err == nil || !strings.Contains(err.Error(), "u1") {
		t.Fatalf("err = %v, want the failing user named", err)
	}
	if n != 1 || h.repo.states["u1"].HealthPoints != 100 || h.repo.states["u2"].HealthPoints != 70 {
		t.Errorf("n=%d u1=%d u2=%d; want 1, u1 untouched, u2 penalised — the loop must continue past u1", n, h.repo.states["u1"].HealthPoints, h.repo.states["u2"].HealthPoints)
	}
}

func TestSweepContinuesPastAUserWhoseWriteFails(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	h.repo.states["u2"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	h.repo.saveErrFor["u1"] = errBoom

	n, err := h.svc.Sweep(ctx, midnite)
	if err == nil || !strings.Contains(err.Error(), "u1") {
		t.Fatalf("err = %v, want the failing user named", err)
	}
	if n != 1 || h.repo.states["u1"].HealthPoints != 100 || h.repo.states["u2"].HealthPoints != 70 {
		t.Errorf("n=%d u1=%d u2=%d; want 1, u1 untouched, u2 penalised", n, h.repo.states["u1"].HealthPoints, h.repo.states["u2"].HealthPoints)
	}
}
```
Add `"fmt"`, `"strings"` and `"sync"` to `service_test.go`'s imports.

- [ ] **Step 2: Run and confirm they fail**

```sh
go test ./internal/pet/... -run 'Sweep' 2>&1 | tail -30
```
Expected: with the Task 5 stub, `TestSweepJudgesEachPetsOwnLocalYesterday` fails (hcm-late untouched, utc-met-marker not spared without a counter), `…EveryZone/America/Santiago` fails (0 penalties for the 6th), `…FirstContact…` fails (the late pet is penalised), `…ResolvedByARevive` fails (50 → 20).

- [ ] **Step 3: Rewrite `Sweep`**

```go
// Sweep is the body of the §8 hourly cron. It runs at every :00 UTC and looks
// at every pet: for each, the local day that most recently ended is
// judged = PreviousDate(LocalDate(now, tz)), and there is work only while
// judged_through is before it. That replaces "local hour is 0": a zone whose
// clocks jump 23:59:59 → 01:00 is judged at 01:00, a zone whose hour 0
// happens twice is judged once, and a tick the process slept through is
// caught up at the next one. No midnight instant is ever constructed.
//
// A day is spared when the pet's own marker says its target was met
// (last_target_met_date == judged) or — leniency fallback only — the Redis
// counter for it reads >= 1800s; a spared day is recorded (MarkJudged) so it
// is never re-read. Otherwise PenaliseMiss applies §8's inactivity logic in
// one conditional UPDATE, so N concurrent sweepers penalise once and the
// count returned is the number of writes that applied.
//
// First contact: a pet with no judged_through yet is judged only for days it
// existed (LocalDate(updated_at) <= judged — updated_at is the creation stamp
// until something writes the row); otherwise its marker is initialised.
//
// Errors on one pet are collected and the rest are still processed.
func (s *Service) Sweep(ctx context.Context, now time.Time) (int, error) {
	cands, err := s.repo.SweepCandidates(ctx)
	if err != nil {
		return 0, err
	}

	penalised := 0
	var errs []error
	fail := func(userID string, err error) { errs = append(errs, fmt.Errorf("user %s: %w", userID, err)) }

	for _, c := range cands {
		loc := quests.Location(c.Timezone)
		judged := PreviousDate(quests.LocalDate(now, loc))

		switch {
		case c.State.JudgedThrough != nil && *c.State.JudgedThrough >= judged:
			continue // nothing has ended since the last judgement
		case c.State.JudgedThrough == nil && quests.LocalDate(c.State.UpdatedAt, loc) > judged:
			// Never judged and not yet alive on the judged day: nothing to
			// judge, but record the day so the row stops depending on updated_at.
			if _, err := s.repo.MarkJudged(ctx, c.UserID, judged); err != nil {
				fail(c.UserID, err)
			}
			continue
		}

		met := c.State.LastTargetMetDate != nil && *c.State.LastTargetMetDate == judged
		if !met {
			total, err := s.study.Total(ctx, c.UserID, judged)
			if err != nil {
				fail(c.UserID, err)
				continue
			}
			met = total >= quests.TargetSeconds
		}
		if met {
			if _, err := s.repo.MarkJudged(ctx, c.UserID, judged); err != nil {
				fail(c.UserID, err)
			}
			continue
		}

		applied, err := s.repo.PenaliseMiss(ctx, c.UserID, judged, now)
		if err != nil {
			fail(c.UserID, err)
			continue
		}
		if applied {
			penalised++
		}
	}
	return penalised, errors.Join(errs...)
}
```
`RunHourly` and `cron.go` are unchanged.

- [ ] **Step 4: Run the whole pet package**

```sh
go test ./internal/pet/... -count=1 -race -timeout 120s -v 2>&1 | grep -E '^(--- |ok|FAIL|panic)' | sort | uniq -c | sort -rn | head -60
```
Expected: every test `--- PASS`, no `FAIL`, no race report (the fakes lock; `TestTwoConcurrentSweepsPenaliseOnce` is what `-race` is for).

- [ ] **Step 5: Mutation check (do not commit the mutations)**

Run each, confirm the named test goes red, revert:
1. In `Sweep`, replace `judged := PreviousDate(quests.LocalDate(now, loc))` with the old `if now.In(loc).Hour() != 0 { continue }` guard before it → `TestSweepStillJudgesAZoneWhoseMidnightDoesNotExist` and `…EveryZone/America/Santiago` fail; `…EachPetsOwnLocalYesterday` fails on `hcm-late`.
2. In `fakeRepo.PenaliseMiss`, drop the `!dateBefore(...)` check (simulating an unconditional `UPDATE`) → `TestSweepIsIdempotentAcrossTicksAndHours` (health 10) and `TestTwoConcurrentSweepsPenaliseOnce` fail. (The real SQL's predicate is pinned by `TestIntegrationVerdictWritesAreConditional`.)
3. In `Sweep`, replace `continue` after `fail(c.UserID, err)` in the counter branch with `return penalised, errors.Join(errs...)` → `TestSweepContinuesPastAUserWhoseCounterIsUnreadable` fails (u2 = 100).
4. In `ApplyRevive`, drop the `JudgedThrough` line → `TestSweepDoesNotPenaliseADayResolvedByARevive` fails (50 → 20), `TestRevivePassResolvesTheLocalDayItWasPassedOn` fails.
5. In `Sweep`, drop `met := c.State.LastTargetMetDate != nil && …` (always consult the counter) → `…EachPetsOwnLocalYesterday` fails on `utc-met-marker` (100 → 70).
6. In `fakeRepo.Ensure`, stamp `time.Time{}` again → `TestSweepFirstContactJudgesOnlyDaysThePetExisted` fails (the late pet is penalised).

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend/internal/pet/service.go backend/internal/pet/service_test.go && git commit -m "pet: the sweep judges each pet's local yesterday by civil date with conditional writes; tests that fail on their named branches" && cd backend
```

---

### Task 7: quests — the hook fires from the durable flag, and the flag is monotonic

**Files:**
- Modify: `backend/internal/quests/repo.go:59-63, 91-98, 168-173`
- Modify: `backend/internal/quests/service.go:52-147`
- Modify: `backend/internal/quests/fakes_test.go:48-121`
- Modify: `backend/internal/quests/service_test.go`
- Modify: `backend/internal/quests/integration_test.go:95-115`

- [ ] **Step 1: Write the failing tests**

In `fakes_test.go`, extend the fakes (the service will not compile against them until Step 3 — that is the point):
```go
type fakeQuestRepo struct {
	log             *callLog
	timezone        string
	roadmap         *Roadmap
	exercises       map[int][]Exercise // by day_number
	completed       map[string]bool
	markCompleteErr error
}

func (f *fakeQuestRepo) MarkComplete(_ context.Context, _, exerciseID string) error {
	if f.markCompleteErr != nil {
		return f.markCompleteErr
	}
	f.log.add("MARK COMPLETE %s", exerciseID)
	f.completed[exerciseID] = true
	return nil
}

type fakeProgressRepo struct {
	log  *callLog
	rows map[string]struct {
		minutes   int
		targetMet bool
	}
	err     error // Upsert fails
	markErr error // MarkTargetMet fails
}

// Upsert mirrors upsertProgressSQL: minutes never lower, and the row's
// is_target_met is reported, never written here.
func (f *fakeProgressRepo) Upsert(_ context.Context, userID, localDate string, minutes int) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	key := userID + "|" + localDate
	row := f.rows[key]
	row.minutes = max(row.minutes, minutes)
	f.rows[key] = row
	f.log.add("UPSERT daily_progress %s minutes=%d", key, row.minutes)
	return row.targetMet, nil
}

func (f *fakeProgressRepo) MarkTargetMet(_ context.Context, userID, localDate string) error {
	if f.markErr != nil {
		return f.markErr
	}
	key := userID + "|" + localDate
	row := f.rows[key]
	row.targetMet = true
	f.rows[key] = row
	f.log.add("MARK TARGET MET %s", key)
	return nil
}
```
(Keep `newFakeQuestRepo`/`newFakeProgressRepo` as they are; the new fields zero-initialise.)

In `service_test.go`, update the expected call log in `TestRecordProgressIncrementsRedisBeforeWritingPostgres` — find its `want := []string{...}` and make the sequence:
```go
	want := []string{
		"INCRBY u1|2026-09-22 600",
		"EXPIRE u1|2026-09-22",
		"UPSERT daily_progress u1|2026-09-22 minutes=10",
		"MARK COMPLETE ex-2-reading",
	}
```
(600 s does not cross the target, so no hook and no flip appear.) Then add:
```go
func TestTheCrossingCallLogsHookThenFlagThenExercise(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	if _, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 1800); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"INCRBY u1|2026-09-22 1800",
		"EXPIRE u1|2026-09-22",
		"UPSERT daily_progress u1|2026-09-22 minutes=30",
		"ON TARGET MET u1|2026-09-22",
		"MARK TARGET MET u1|2026-09-22",
		"MARK COMPLETE ex-2-reading",
	}
	if !reflect.DeepEqual(h.log.calls, want) {
		t.Errorf("calls = %q\nwant  %q", h.log.calls, want)
	}
}

func TestTheHookFiresFromTheDurableFlagNotTheCounterEdge(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	ctx := context.Background()
	if _, err := h.svc.RecordProgress(ctx, "u1", "ex-2-reading", 1800); err != nil {
		t.Fatal(err)
	}
	// The Redis key is lost mid-day (eviction, restart, FLUSHDB). The counter
	// restarts at 0 and the next 1800s "cross" the edge a second time.
	h.counter.totals = map[string]int64{}
	out, err := h.svc.RecordProgress(ctx, "u1", "ex-2-practice", 1800)
	if err != nil {
		t.Fatal(err)
	}
	if h.pet.fired != 1 {
		t.Errorf("hook fired %d times, want 1 — daily_progress.is_target_met already says the pet was told", h.pet.fired)
	}
	if out.NewlyMet || !out.IsTargetMet {
		t.Errorf("out = %+v, want NewlyMet false and IsTargetMet true (the durable row wins)", out)
	}
	if row := h.progress.rows["u1|2026-09-22"]; row.minutes != 30 || !row.targetMet {
		t.Errorf("row = %+v, want minutes 30 (never lowered) and target met", row)
	}
}

func TestAFailedUpsertOnTheCrossingCallIsRetriedByTheNextCall(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	ctx := context.Background()
	h.progress.err = errors.New("pool exhausted")
	if _, err := h.svc.RecordProgress(ctx, "u1", "ex-2-reading", 1800); err == nil {
		t.Fatal("want the upsert error surfaced")
	}
	if h.pet.fired != 0 {
		t.Fatalf("hook fired %d times on a failed write, want 0", h.pet.fired)
	}
	h.progress.err = nil
	out, err := h.svc.RecordProgress(ctx, "u1", "ex-2-practice", 60) // counter is at 1860: the old edge test says "not newly met"
	if err != nil {
		t.Fatal(err)
	}
	if h.pet.fired != 1 || !out.NewlyMet {
		t.Errorf("fired=%d NewlyMet=%t; want 1 and true — the day was never handed to the pet", h.pet.fired, out.NewlyMet)
	}
}

func TestAFailedMarkCompleteOnTheCrossingCallDoesNotLoseTheHook(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	ctx := context.Background()
	h.quests.markCompleteErr = ErrExerciseNotFound
	if _, err := h.svc.RecordProgress(ctx, "u1", "ex-2-reading", 1800); !errors.Is(err, ErrExerciseNotFound) {
		t.Fatalf("err = %v, want ErrExerciseNotFound", err)
	}
	if h.pet.fired != 1 || !h.progress.rows["u1|2026-09-22"].targetMet {
		t.Errorf("fired=%d targetMet=%t; want the pet told and the day flagged before the exercise write", h.pet.fired, h.progress.rows["u1|2026-09-22"].targetMet)
	}
	h.quests.markCompleteErr = nil
	if _, err := h.svc.RecordProgress(ctx, "u1", "ex-2-practice", 60); err != nil {
		t.Fatal(err)
	}
	if h.pet.fired != 1 {
		t.Errorf("hook fired %d times, want still 1", h.pet.fired)
	}
}

func TestAPetHookFailureLeavesTheDayUnflaggedSoTheNextCallRetries(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	ctx := context.Background()
	h.pet.hookErr = errors.New("pet is on fire")
	if _, err := h.svc.RecordProgress(ctx, "u1", "ex-2-reading", 1800); err != nil {
		t.Fatalf("a hook failure must not fail the request: %v", err)
	}
	if h.pet.fired != 1 || h.progress.rows["u1|2026-09-22"].targetMet {
		t.Fatalf("fired=%d targetMet=%t; want 1 and false — the flag means 'the pet was told'", h.pet.fired, h.progress.rows["u1|2026-09-22"].targetMet)
	}
	h.pet.hookErr = nil
	if _, err := h.svc.RecordProgress(ctx, "u1", "ex-2-practice", 60); err != nil {
		t.Fatal(err)
	}
	if h.pet.fired != 2 || !h.progress.rows["u1|2026-09-22"].targetMet {
		t.Errorf("fired=%d targetMet=%t; want 2 (pet's own marker makes the re-fire a no-op) and true", h.pet.fired, h.progress.rows["u1|2026-09-22"].targetMet)
	}
}

func TestAFailedTargetMetFlagIsLoggedNotReturned(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.progress.markErr = errors.New("pool exhausted")
	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 1800)
	if err != nil {
		t.Fatalf("a failed flag must not 500 the recorded session (a retry would INCRBY again): %v", err)
	}
	if h.pet.fired != 1 || !out.NewlyMet || !h.quests.completed["ex-2-reading"] {
		t.Errorf("fired=%d out=%+v completed=%t; want the hook, the response and the exercise unaffected", h.pet.fired, out, h.quests.completed["ex-2-reading"])
	}
}
```
`TestFurtherProgressTheSameDayDoesNotRefire`, `TestCrossingExactly1800SecondsMeetsTheTargetAndFiresOnce` and `TestAPetHookFailureDoesNotFailTheRequest` stay as they are — they must still pass. In the existing `TestCrossingExactly1800…`, the row assertion `row.minutes != 30 || !row.targetMet` stays valid.

In `integration_test.go`, after the block that reads `daily_progress` and asserts `(30, true)`, add:
```go
	// Monotonic: a lost counter cannot lower minutes_spent or unset is_target_met.
	if _, err := svc.RecordProgress(ctx, userID, suite.Tasks[1].ID, 600); err != nil {
		t.Fatalf("second RecordProgress: %v", err)
	}
	rdb.Client.Del(ctx, store.DailyAccumulatedKey(userID, now)) // the eviction / restart / FLUSHDB case
	out, err = svc.RecordProgress(ctx, userID, suite.Tasks[2].ID, 60)
	if err != nil {
		t.Fatalf("RecordProgress after the counter was lost: %v", err)
	}
	if !out.IsTargetMet || out.NewlyMet {
		t.Errorf("out = %+v after the counter was lost, want IsTargetMet true (durable row) and NewlyMet false", out)
	}
	if err := pg.Pool.QueryRow(ctx,
		`SELECT minutes_spent, is_target_met FROM daily_progress WHERE user_id = $1 AND date = $2::date`,
		userID, LocalDate(now, time.UTC)).Scan(&minutes, &met); err != nil {
		t.Fatalf("reading daily_progress: %v", err)
	}
	if minutes != 40 || !met {
		t.Errorf("daily_progress = (%d, %t) after the counter was lost, want (40, true) — never lowered", minutes, met)
	}
```
(`suite.Tasks` has three entries; check the test's later sections do not already consume `Tasks[1]`/`Tasks[2]` — if they do, use the ids they leave free.)

- [ ] **Step 2: Confirm the build fails**

```sh
go vet ./internal/quests/ 2>&1 | head
```
Expected: `*fakeProgressRepo does not implement ProgressRepo` / wrong `Upsert` signature.

- [ ] **Step 3: Implement**

`repo.go` — the interface, SQL and methods:
```go
// ProgressRepo owns daily_progress. The row is monotonic: minutes_spent never
// lowers and is_target_met never returns to FALSE, so a Redis counter lost
// mid-day cannot rewrite the durable record from a restarted total.
// is_target_met means "the pet has been told about this day": Upsert reports
// it, and MarkTargetMet sets it after Pet.OnTargetMet has returned nil.
type ProgressRepo interface {
	// Upsert writes minutes_spent for (userID, localDate) — never lowering it —
	// and reports whether is_target_met is already TRUE on that row.
	Upsert(ctx context.Context, userID, localDate string, minutes int) (alreadyMet bool, err error)
	// MarkTargetMet flips is_target_met to TRUE. Idempotent.
	MarkTargetMet(ctx context.Context, userID, localDate string) error
}
```
```go
	// daily_progress.date defaults to CURRENT_DATE, which is the *server's*
	// date — always pass the user's local date explicitly. A new row starts
	// is_target_met = FALSE whatever the total: only MarkTargetMet sets it,
	// after the pet hook, so the flag never gets ahead of the pet.
	upsertProgressSQL = `
INSERT INTO daily_progress (user_id, date, minutes_spent, is_target_met)
VALUES ($1, $2::date, $3, FALSE)
ON CONFLICT (user_id, date) DO UPDATE SET
    minutes_spent = GREATEST(COALESCE(daily_progress.minutes_spent, 0), EXCLUDED.minutes_spent)
RETURNING COALESCE(is_target_met, FALSE)`

	markTargetMetSQL = `
UPDATE daily_progress
SET is_target_met = TRUE
WHERE user_id = $1 AND date = $2::date`
```
```go
func (r *PgRepo) Upsert(ctx context.Context, userID, localDate string, minutes int) (bool, error) {
	var alreadyMet bool
	if err := r.Pool.QueryRow(ctx, upsertProgressSQL, userID, localDate, minutes).Scan(&alreadyMet); err != nil {
		return false, fmt.Errorf("quests: upserting daily_progress: %w", err)
	}
	return alreadyMet, nil
}

func (r *PgRepo) MarkTargetMet(ctx context.Context, userID, localDate string) error {
	if _, err := r.Pool.Exec(ctx, markTargetMetSQL, userID, localDate); err != nil {
		return fmt.Errorf("quests: marking target met: %w", err)
	}
	return nil
}
```
`service.go` — the `RecordProgress` doc comment's steps 2–4 and the body from the `Add` call down:
```go
//  1. INCRBY the Redis counter (+ EXPIRE) and read the running total back.
//  2. Upsert daily_progress.minutes_spent from that total (never lowering it)
//     and learn whether the day's is_target_met is already set.
//  3. If the total is at or past 1800s and the day is not yet flagged: fire
//     Pet.OnTargetMet, and only when it returns nil flag the day with
//     MarkTargetMet. A hook failure leaves the day unflagged so the next
//     progress call fires again; pet's own last_target_met_date makes a
//     re-fire a no-op once it has landed (at-least-once here, exactly-once
//     there). A failed flag is logged, not returned: the session is recorded
//     and a 500 would make the client retry and INCRBY again.
//  4. Mark the exercise complete, then read Pet.State so the §6.2 response
//     carries pet_health / streak_count.
```
```go
	total, err = s.counter.Add(ctx, userID, date, seconds)
	if err != nil {
		return ProgressResult{}, err
	}

	alreadyMet, err := s.progress.Upsert(ctx, userID, date, int(total/60))
	if err != nil {
		return ProgressResult{}, err
	}
	// The durable row wins over the counter for the response: a counter lost
	// mid-day does not un-meet a met day.
	targetMet := total >= TargetSeconds || alreadyMet
	newlyMet := total >= TargetSeconds && !alreadyMet

	if newlyMet {
		if err := s.pet.OnTargetMet(ctx, userID, date); err != nil {
			// Best-effort for this call: the study session is already recorded
			// and must not be rolled back by a pet failure. The day stays
			// unflagged, so the next call retries the hook.
			log.Printf("quests: pet target-met hook failed for user %s on %s (will retry on the next progress call): %v", userID, date, err)
		} else if err := s.progress.MarkTargetMet(ctx, userID, date); err != nil {
			log.Printf("quests: flagging daily_progress.is_target_met for user %s on %s: %v", userID, date, err)
		}
	}

	// MarkComplete keeps its own ErrExerciseNotFound for the race where the
	// row vanished between CheckExercise and here; the handler still maps it
	// to 404. It runs after the hook so a vanished exercise cannot cost the
	// user the day's +20.
	if err := s.quests.MarkComplete(ctx, roadmap.ID, exerciseID); err != nil {
		return ProgressResult{}, err
	}
```
The `ProgressResult.NewlyMet` comment becomes: "NewlyMet is true only on a call that fired Pet.OnTargetMet — the total is at or past TargetSeconds and daily_progress did not yet say the pet was told. …"

- [ ] **Step 4: Run the quests package**

```sh
go test ./internal/quests/... -count=1 -timeout 120s -v 2>&1 | grep -E '^(--- |ok|FAIL|panic)' | sort | uniq -c | sort -rn | head -60
```
Expected: all `--- PASS` (integration `--- SKIP` without a stack). Mutation check, then revert: restore `newlyMet := targetMet && total-seconds < TargetSeconds` → `TestTheHookFiresFromTheDurableFlagNotTheCounterEdge` (fired 2) and `TestAFailedUpsertOnTheCrossingCallIsRetriedByTheNextCall` (fired 0) fail. Move `MarkTargetMet` before the hook → `TestAPetHookFailureLeavesTheDayUnflagged…` fails. Move the hook back below `MarkComplete` → `TestAFailedMarkCompleteOnTheCrossingCallDoesNotLoseTheHook` fails.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/quests/repo.go backend/internal/quests/service.go backend/internal/quests/fakes_test.go backend/internal/quests/service_test.go backend/internal/quests/integration_test.go && git commit -m "quests: daily_progress is monotonic and the pet hook fires from its is_target_met flag, not the counter edge" && cd backend
```

---

### Task 8: Handler 401 branches, and CODEMAP that tells the truth

**Files:**
- Modify: `backend/internal/pet/handler_test.go:15-25`
- Modify: `harness/CODEMAP.md` (`store`, `quests`, `pet` bullets)

- [ ] **Step 1: The two dead 401 branches**

In `handler_test.go`, give `newPetRouter` an unauthenticated variant and two tests:
```go
// newPetRouter stands in for auth.Require() by injecting the user id under
// auth.ContextUserID (Require itself is covered in the auth slice). An empty
// userID injects nothing, which is what the handlers' own 401 guard sees.
func newPetRouter(svc *Service, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1", func(c *gin.Context) {
		if userID != "" {
			c.Set(auth.ContextUserID, userID)
		}
		c.Next()
	})
	g.GET("/pet/status", StatusHandler(svc))
	g.POST("/pet/revive", ReviveHandler(svc))
	return r
}

func TestHandlersAnswer401WithoutAnAuthenticatedUser(t *testing.T) {
	h := newHarness(sept22)
	r := newPetRouter(h.svc, "")
	for _, tc := range []struct{ method, path string }{{"GET", "/api/v1/pet/status"}, {"POST", "/api/v1/pet/revive"}} {
		w := do(r, tc.method, tc.path, "")
		if w.Code != http.StatusUnauthorized || strings.TrimSpace(w.Body.String()) != `{"error":"unauthorized"}` {
			t.Errorf("%s %s = %d %s, want 401 {\"error\":\"unauthorized\"}", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
	if h.repo.ensured != 0 {
		t.Error("an unauthenticated request touched the repo")
	}
}
```
Run: `go test ./internal/pet/... -run 'Handler|Status|Revive' -v 2>&1 | tail -12` → all `--- PASS`.

- [ ] **Step 2: CODEMAP**

In `harness/CODEMAP.md`:

*`store` bullet* — after "`0001_init` is the spec §3.2 DDL verbatim; …" add: "`0003_pet_verdict_dates` adds `pet_states.last_target_met_date` / `judged_through` (DATE; the pet's own once-per-day markers — see `pet`); the spec's §3.2 block carries the same `ALTER TABLE` after its `0002` block."

*`quests` bullet* — replace from "then upserts `daily_progress` (`minutes_spent = total/60`, `is_target_met = total >= 1800`) from that Redis total, then sets `exercises.is_completed`, …" through "…if the upsert or `MarkComplete` fails on the call that crossed 1800 the counter is already past it and `OnTargetMet` never fires for that user that day (reviewer 2026-09-22)." with:

"then upserts `daily_progress.minutes_spent = total/60` **monotonically** (`GREATEST`, never lowered; a fresh row starts `is_target_met = FALSE`) and reads back the row's `is_target_met`, which means *the pet has been told about this day*; if `total >= 1800` and the row was not yet flagged it fires `quests.Pet.OnTargetMet` and, only when that returns nil, `MarkTargetMet` flips the flag; then sets `exercises.is_completed`, and answers `{daily_seconds_spent, daily_minutes_spent, is_target_met, pet_health, streak_count}` with `is_target_met = total >= 1800 || row flagged` (the durable row wins over a lost counter). The hook is therefore **at-least-once from quests** (a failed upsert, hook or `MarkComplete` on the crossing call leaves the flag FALSE and the next progress call fires again) and **exactly-once in pet** (`pet_states.last_target_met_date`); a failed flag is logged, never returned. `service_test.go` pins the order INCRBY → EXPIRE → UPSERT → ON TARGET MET → MARK TARGET MET → MARK COMPLETE and the lost-counter / failed-write / failed-hook replays. Known accepted gap: a crash between the INCRBY and the upsert loses the Postgres row but keeps the Redis count, and the next progress call re-derives and re-upserts `minutes_spent` — the flag and the hook no longer depend on that re-derivation."

*`pet` bullet* — replace from "**Success (+20 capped at 100, streak+1, `last_practiced_at = now`) is applied exactly once per local day from `quests.Pet.OnTargetMet`** …" through "…which makes a re-run in the same hour idempotent." with:

"**Success (+20 capped at 100, streak+1, `last_practiced_at = now`, `last_target_met_date = D`) is applied exactly once per local day D** — `Service.OnTargetMet(user, D)` writes through `Repo.SaveTargetMet`, an `UPDATE … WHERE last_target_met_date IS NULL OR last_target_met_date < D`, so a repeat call (quests re-fires after a failure; a lost Redis counter; any future caller) is a silent no-op; `pet.QuestHook` registers it in `main.go`. The §8 hourly cron (`pet.RunHourly`, in-process, every `:00` UTC) applies **only** the inactivity logic and does it **by civil date, not local hour**: for every pet, `judged = PreviousDate(LocalDate(now, tz))`; nothing to do while `pet_states.judged_through >= judged`; the day is spared when `last_target_met_date == judged` or (leniency fallback only) the previous day's `daily:accumulated` total via `StudyCounter` is ≥ 1800, and `MarkJudged` records it; otherwise `PenaliseMiss` runs `health = GREATEST(0, health-30), streak = 0, wilted at 0, judged_through = judged` in one conditional `UPDATE`, so any number of sweepers penalise once and a tick the process slept through is caught up at the next one. Zones with no local hour 0 (spring-forward at midnight — Santiago, Havana in March, Asunción, Beirut) are judged at 01:00; a repeated hour 0 (fall-back) is judged once; `service_test.go` walks 72 hourly ticks across UTC / Ho Chi Minh / Kolkata / Kathmandu / Chatham / Havana / Santiago. First contact: a pet with `judged_through NULL` is judged only for days it existed (`LocalDate(updated_at)`), then carries the marker. **A passed revival resolves its local day** (`ApplyRevive` sets `judged_through = today`; owner decision 2026-09-23): the sweep applies no miss for it — nor, because the marker is monotonic, for an earlier day whose tick was missed while the plant already sat at 0 — and the +20 for reaching 1800 s later that day remains available. `OnTargetMet` never moves `judged_through`, so a met day cannot shield an unjudged earlier miss. `daily_progress` is quests' table and pet never reads it — pet's own markers are the durable verdict."

Also in the `pet` bullet, in the revive sentence, after "setting `{health_points: 50, stage: sprout, current_streak: 0}` (§6.3)" add "and resolving the day (above)". Leave the rest.

- [ ] **Step 3: Full backend suite, vet, formatting**

```sh
go build ./... && go vet ./... && gofmt -l ./internal/pet ./internal/quests ./internal/store
# expect: no output from gofmt for the files you touched
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -race -timeout 300s
# expect: ok for every package
```

- [ ] **Step 4: Commit**

```sh
cd .. && git add backend/internal/pet/handler_test.go harness/CODEMAP.md && git commit -m "pet: 401 handler tests; CODEMAP: pet owns the day's verdict, quests' hook fires from the durable flag" && cd backend
```

---

## Verification

Run from the worktree root unless noted. Every command's expectation is what the reviewer re-runs.

```sh
cd backend && go build ./... && go vet ./...
# expect: no output

env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -race -timeout 300s
# expect: ok for every package, no live service needed

go test ./internal/pet/... -run 'OnTargetMet' -v | grep -E '^--- '
# expect: PASS ×5 incl. TwiceForTheSameLocalDateBumpsOnce, ForTheNextLocalDateBumpsAgain, ForAnEarlierLocalDateIsIgnored

go test ./internal/pet/... -run 'Sweep|TwoConcurrent' -v | grep -E '^(--- |    --- )'
# expect: PASS for EachPetsOwnLocalYesterday, IsIdempotentAcrossTicksAndHours, TwoConcurrentSweepsPenaliseOnce,
#         WiltsAfterFourMissedDays, DoesNotPenaliseADayResolvedByARevive, StillJudgesYesterdayWhenTodaysTargetWasMetFirst,
#         FirstContactJudgesOnlyDaysThePetExisted,
#         JudgesEveryLocalDayExactlyOnceInEveryZone (+7 subtests incl. America/Santiago), StillJudgesAZoneWhoseMidnightDoesNotExist,
#         ContinuesPastAUserWhoseCounterIsUnreadable, ContinuesPastAUserWhoseWriteFails

go test ./internal/quests/... -run 'Hook|Flag|Crossing|Refire|Retried|MarkComplete' -v | grep -E '^--- '
# expect: PASS for TheHookFiresFromTheDurableFlagNotTheCounterEdge, AFailedUpsertOnTheCrossingCallIsRetriedByTheNextCall,
#         AFailedMarkCompleteOnTheCrossingCallDoesNotLoseTheHook, APetHookFailureLeavesTheDayUnflaggedSoTheNextCallRetries,
#         AFailedTargetMetFlagIsLoggedNotReturned, TheCrossingCallLogsHookThenFlagThenExercise, and the pre-existing once-only tests

grep -n 'Hour() == 0\|time.Date(local.Year()' internal/pet/service.go
# expect: no hits — no hour test, no midnight instant

grep -n 'total-seconds < TargetSeconds' internal/quests/service.go
# expect: no hits — the counter edge is gone

grep -c 'IS NULL OR last_target_met_date <\|IS NULL OR judged_through <' internal/pet/repo.go
# expect: 3 — every verdict write is conditional

grep -n 'GREATEST' internal/quests/repo.go internal/pet/repo.go
# expect: quests: minutes_spent; pet: saveSQL judged_through, penaliseMissSQL health_points

grep -rn --include='*.go' 'daily_progress' internal/pet/
# expect: no hits — pet never reads quests' table

grep -n 'Timezones' internal/pet/*.go
# expect: no hits

grep -c 'ADD COLUMN judged_through DATE;' internal/store/migrations/0003_pet_verdict_dates.up.sql "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md"
# expect: 1 and 1 — spec DDL = migrations

git diff main..HEAD --stat -- cmd/api/main.go
# expect: empty — this plan does not touch the wiring file

cd .. && python3 tools/harness/cli.py validate; echo exit=$?
# expect: exit=0

git log --oneline main..HEAD
# expect: 8 commits, one per task, each with the Co-Authored-By trailer
```

**With a stack** (`COMPOSE_PROJECT_NAME=<slug>`, free ports, `make up`, export `TEST_DATABASE_URL`/`TEST_REDIS_URL`; CI's `backend-integration` job runs these regardless and fails on any SKIP):
```sh
cd backend && make test-integration 2>&1 | grep -E '^(--- |ok|FAIL)'
# expect: PASS TestIntegrationVerdictWritesAreConditional (SQL predicates, 8 concurrent PenaliseMiss → 1, ApplyMiss mirror),
#         PASS TestIntegrationEnsureCreatesExactlyOnePetRow, PASS TestIntegrationDailyAndProgressAgainstRealServices (40/true after DEL)
make down
```

**Mutation table — which assertion turns red** (the reviewer may spot-check any row):

| Behaviour | Mutation | Failing test |
| --- | --- | --- |
| Pet-side once per day | `OnTargetMet` writes through `Save` instead of `SaveTargetMet` (or fake: drop `dateBefore`) | `TestOnTargetMetTwiceForTheSameLocalDateBumpsOnce` (`100/6`, saved 2); the real predicate: `TestIntegrationVerdictWritesAreConditional` §1 |
| Guard is not "never bump" | `OnTargetMet` returns early always | `TestOnTargetMetForTheNextLocalDateBumpsAgain` |
| Marker monotonic | predicate `<>` instead of `<` (fake: `*col != d`) | `TestOnTargetMetForAnEarlierLocalDateIsIgnored` |
| Hook from durable flag | restore `total-seconds < TargetSeconds` | `TestTheHookFiresFromTheDurableFlagNotTheCounterEdge` (fired 2), `TestAFailedUpsert…` (fired 0) |
| Flag after hook | `MarkTargetMet` before `OnTargetMet` | `TestAPetHookFailureLeavesTheDayUnflagged…` |
| Hook before exercise write | hook below `MarkComplete` | `TestAFailedMarkCompleteOnTheCrossingCall…` |
| `daily_progress` monotonic | `minutes_spent = EXCLUDED.minutes_spent` | `TestIntegrationDailyAndProgress…` (10 after DEL); fake: `TestTheHookFires…` row.minutes |
| Sweep by civil date | reinstate `Hour() != 0 → continue` | `TestSweepStillJudgesAZoneWhoseMidnightDoesNotExist`, `…EveryZone/America/Santiago`, `…EachPetsOwnLocalYesterday` (`hcm-late`) |
| Sweep idempotent / concurrent | `PenaliseMiss` unconditional | `TestSweepIsIdempotentAcrossTicksAndHours` (10), `TestTwoConcurrentSweepsPenaliseOnce`; SQL: integration §2 |
| Durable spare | drop `LastTargetMetDate == judged` | `…EachPetsOwnLocalYesterday` (`utc-met-marker` → 70) |
| Counter fallback | drop the `study.Total` branch | `…EachPetsOwnLocalYesterday` (`utc-met-counter` → 70) |
| Revival resolves its day | `ApplyRevive` without `JudgedThrough` | `TestSweepDoesNotPenaliseADayResolvedByARevive` (50 → 20), `TestRevivePassResolvesTheLocalDayItWasPassedOn` |
| A met day never shields an unjudged miss | `OnTargetMet`/`ApplyTargetMet` also advances `JudgedThrough` | `TestSweepStillJudgesYesterdayWhenTodaysTargetWasMetFirst` (n 0, health 100) |
| Late sweep after a next-morning revival takes nothing | `ApplyRevive` uses `PreviousDate(today)` or the sweep ignores `judged_through` for penalties | `TestSweepDoesNotPenaliseADayResolvedByARevive` second half (`h2`, 50 → 20) |
| First contact | fake `Ensure` stamps zero time / drop the `UpdatedAt` rule | `TestSweepFirstContactJudgesOnlyDaysThePetExisted` |
| Loop continues | `return` instead of `continue` in either error branch | `TestSweepContinuesPastAUserWhoseCounterIsUnreadable`, `…WhoseWriteFails` |
| Count is honest | `penalised++` regardless of `applied` | `TestTwoConcurrentSweepsPenaliseOnce` (sum 40) |
| SQL miss = `ApplyMiss` | `CASE WHEN … < 0` (off by one at 30) | integration §4 (pre-image 30 → `sprout` vs `wilted`) |
| 401 guard | handlers skip the `userID == ""` check | `TestHandlersAnswer401WithoutAnAuthenticatedUser` |

After pushing: `gh run list --branch <branch>` must show `backend-unit`, `backend-integration`, `harness-tooling` green.

## Notes and open questions

- **One plan, not two.** The quests change (Task 7) and the pet change (Tasks 2–6) are two halves of one guarantee: quests makes the hook at-least-once by flagging the day only after the pet confirms; pet makes it exactly-once by its own marker. Shipping either half alone would be correct-but-incomplete (quests alone: a lost counter still cannot double-bump, but a pet failure still loses the day; pet alone: a failed write on the crossing call still loses the day). They also share the fakes and the CODEMAP paragraph. Hence one branch.
- **Residual race, documented:** a sweep judging D−1 and an `OnTargetMet` for D within the same second (a task straddling midnight reported at 00:00:0x while the tick runs) can land in either order; miss-then-met gives 90/1, met-then-miss gives 70/0 — the difference is one streak day and 20 health, in a sub-second window, and the streak reset is spec-conformant either way. Per-day rows would remove it; not worth a table.
- **Revival closes unapplied earlier misses too** (monotonic `judged_through`). Only reachable when a tick was missed *and* the user revives before the catch-up sweep, and only for a plant that was already at 0 — the miss would have applied `max(0, 0-30) = 0`. Stated in decision 4 and pinned by the second half of `TestSweepDoesNotPenaliseADayResolvedByARevive`.
- **Residual leniency on first contact:** a pre-`0003` pet (or a brand-new one) whose first sweep runs *after* an `OnTargetMet` for today (service down over midnight, user studies before it comes back) has `LocalDate(updated_at) > judged` and yesterday's miss is skipped once. After that first sweep the marker exists and the path is gone. Accepted; the alternative was a timezone read inside `Ensure` on every `GET /pet/status`.
- **Pet-hook failure with a healthy `daily_progress`** still costs the +20 until the next progress call that day. Both tables live in one Postgres, so the realistic failure (pool down) fails the upsert first and returns 500 before the hook; the case left is a pet-package bug, which logs on every call. Not engineered around.
- **Hourly full scan.** `SweepCandidates` returns every pet with its timezone; already-judged rows cost no I/O. When `pet_states` is large, push `WHERE p.judged_through IS NULL OR p.judged_through < CURRENT_DATE` into `candidatesSQL` — a local yesterday is never later than the server's today, so the prefilter is safe in every zone. Not done now.
- **`is_target_met` semantics changed** from "total ≥ 1800 at last write" to "the pet has been told". Nothing else reads the column today (`grep -rn is_target_met backend/ --include='*.go'` = quests only). If a reporting feature later wants "met the target" independent of the pet, add a column rather than overloading this one.
- **§3.2 in the 1st-thinking doc** still shows the original `pet_states`; the backend spec wins for its layer (AGENTS.md) and is the one edited. The spec's ERD sketch (its §3.1 box diagram) is not updated — `0002` set that precedent.
- **`ReviveSeconds` (900) vs `TargetSeconds` (1800)** stays as is: the challenge is deliberately cheaper than the day, and now buys the day. If the owner instead wants revival to be a partial top-up the night can reduce, delete the `JudgedThrough` line in `ApplyRevive` and flip `TestSweepDoesNotPenaliseADayResolvedByARevive` to expect 20 — one line and one assertion, which is why the decision is recorded here rather than buried.
- **`updated_at` is no longer an idempotence guard** anywhere; it is a plain audit stamp (and, until first contact, the creation proxy). `MarkJudged` deliberately does not touch it.
