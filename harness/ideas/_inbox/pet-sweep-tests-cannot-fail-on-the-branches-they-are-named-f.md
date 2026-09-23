---
type: bug
status: planned
source: reviewer
run: _inbox
priority: low
plan: harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md
---
# Pet sweep tests cannot fail on the branches they are named for

## Why
`internal/pet`'s suite is unusually good — the handler tests compare whole §6.3 bodies byte for
byte, and `TestIntegrationEnsureCreatesExactlyOnePetRow` genuinely exercises the real SQL, the
`pet_stage` cast and eight concurrent `Ensure` calls. Four gaps stand out against that bar, all in
the sweep, which is the one piece of this slice nobody can observe in production until a plant
wilts.

**1. The "continues" test has nothing to continue to.**

```go
// backend/internal/pet/service_test.go:287-299
func TestSweepSkipsAUnreadableCounterAndContinues(t *testing.T) {
        h.repo.states["u1"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-30 * time.Hour)}
        h.study.err = errBoom
```

One candidate. The test proves the error is surfaced and that pet is not penalised, but the
`continue` at `service.go:160` — the whole point of collecting into `errs` instead of returning —
is never exercised. Deleting the `continue` and returning the error immediately keeps this test
green. `fakeStudy.err` is also all-or-nothing, so "one user fails, the rest are processed" is not
expressible with the current fake.

**2. The `Save` error branch is dead in tests.** `fakeRepo.saveErr` exists
(`fakes_test.go:16`) and is never set by any test. The second `errs = append(...)` at
`service.go:166` has no coverage.

**3. The fake's fresh row diverges from Postgres in the field the sweep reads.**
`fakeRepo.Ensure` inserts `defaultState(time.Time{})` — a **zero** `UpdatedAt` — while the DDL
stamps `updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP`
(`0001_init.up.sql:41`). The helper's own comment says it "mirrors the §3.2 column defaults", and
for this column it does not. Consequence: the realistic case "a user signs up at 23:50 and is
swept ten minutes later" — where production's `updated_at` is minutes old and the fake's is the
zero time — cannot be written against these fakes, and no test covers a newly created pet meeting
the sweep.

**4. No zone-boundary table.** The sweep is selected on `now.In(loc).Hour() == 0` and the only
zones tested are `UTC` and `Asia/Ho_Chi_Minh` (whole-hour offsets). Half-hour zones
(`Asia/Kolkata`, `Asia/Kathmandu`, `Pacific/Chatham`) and both DST directions are untested; the
reviewer's probe found that half-hour zones are in fact correct and that midnight-DST zones are
not (filed separately), which is exactly the kind of thing a table test is for.

Minor, same theme: `StatusHandler`/`ReviveHandler`'s `userID == ""` → 401 branch
(`handler.go:55-58`, `:72-75`) is never hit because `newPetRouter` always injects an id, and
`Repo.Timezone`'s error path in `Revive` (`service.go:77-80`) has no test.

## Expected output
Each named branch can fail:

- `TestSweepSkipsAUnreadableCounterAndContinues` seeds **two** candidates, makes `fakeStudy.err`
  per-user (`errFor map[string]error`), and asserts the healthy user *was* penalised while the
  failing one was not and the joined error names it — so removing the `continue` turns the test red;
- a test sets `fakeRepo.saveErr` for one user and asserts the same shape for the write branch;
- `fakeRepo.Ensure` stamps the harness clock (`defaultState(f.now())`) so a fresh row looks like
  Postgres's, plus a test for a pet created after the local midnight the sweep is judging
  (skipped) and one created just before it (penalised);
- a table test walks `Sweep` across every `:00` UTC of one day for `UTC`, `Asia/Kolkata`,
  `Asia/Kathmandu`, `Pacific/Chatham`, `America/Havana` (fall back) and `America/Santiago`
  (spring forward), asserting exactly one penalty per local day per user;
- `newPetRouter` gains a variant that injects no user id, and both handlers are asserted to answer
  401 `unauthorized`.

## Evidence
- Plan: `harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md` (Task 4
  fakes, Task 5 `Sweep` tests).
- `backend/internal/pet/service_test.go:287-299` — the single-candidate "and continues" test.
- `backend/internal/pet/service.go:157-168` — the two `errs`/`continue` branches under test.
- `backend/internal/pet/fakes_test.go:16` (`saveErr`, unused), `:24-26` and `:34`
  (`defaultState(now)` called with `time.Time{}`), `:122-138` (all-or-nothing `fakeStudy.err`).
- `backend/internal/store/migrations/0001_init.up.sql:41` — the real `updated_at` default.
- `backend/internal/pet/handler_test.go:18-25` — `newPetRouter` always sets `auth.ContextUserID`.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — low.** Test-only, but the sweep is the one piece of §8 nobody observes until a plant wilts, and the pet "durable day judgement" plan will rewrite `Sweep` and its fakes anyway — take these tests (per-user fake errors, realistic `updated_at`, zone table) in that plan.

**Planned (2026-09-23):** `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` — decision 5, Tasks 4, 6, 8 (per-user fake errors, clocked `Ensure`, zone table, 401 tests; mutation table in Verification).
