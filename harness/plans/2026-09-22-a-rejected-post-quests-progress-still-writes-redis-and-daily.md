---
idea: harness/ideas/_inbox/a-rejected-post-quests-progress-still-writes-redis-and-daily.md
status: done
priority: high
merged: true
amends: harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md
branch: harness/2026-09-22-high-quests-daily-quest-suite-and-progress-recording
worktree: .worktrees/quests-daily-quest-suite-and-progress-recording
---
# Quests amend: validate before writing, bound duration_seconds, DST-safe day_number — Plan

**Idea:** `harness/ideas/_inbox/a-rejected-post-quests-progress-still-writes-redis-and-daily.md` (blocker 1)
**Also fixes:** `harness/ideas/_inbox/duration-seconds-is-unbounded-so-one-request-bricks-a-user-s.md` (blocker 2),
`harness/ideas/_inbox/daynumber-loses-a-calendar-day-at-every-spring-forward-dst-t.md` (high, non-blocking),
`harness/ideas/_inbox/the-progress-rejection-tests-assert-only-the-error-and-the-f.md` (medium) — all four carry `plan:` → this file.
**Amends:** `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md`. This plan lands on that plan's branch
`harness/2026-09-22-high-quests-daily-quest-suite-and-progress-recording` in its worktree
`.worktrees/quests-daily-quest-suite-and-progress-recording` (HEAD `ef4b9fa`). No new branch, no new worktree. Every task
edits files that already exist there.
**Review that found them:** `harness/reviews/2026-09-22-quests-daily-quest-suite-and-progress-recording.md` (verdict `fail`).

**Goal:** Make `POST /api/v1/quests/progress` write nothing when it answers 404 or 400, make an oversized `duration_seconds` a
400 instead of a 48-hour outage for the user, and make `day_number` count calendar days so a DST learner never loses a day —
without changing the backend spec §6.2 request/response shapes.

**Architecture:** Three small changes inside `backend/internal/quests`, in the order the review ranked them. (1) `RecordProgress`
gains a read-only `QuestRepo.CheckExercise` lookup (exercise is on the caller's active roadmap **and** on today's `day_number`)
before any write, so the §5.2 write sequence INCRBY → `daily_progress` upsert → `MarkComplete` → pet hook runs only for a request
that will succeed; the ordering test is unchanged. (2) Two documented constants bound the input — `MaxDurationSeconds = 3600` per
call, `MaxDailySeconds = 86400` per local day, the latter checked against a `Counter.Total` read before the INCRBY — and both breaches
surface as a new sentinel `ErrInvalidDuration` that the handler maps to the existing `400 {"error":"invalid_request"}`. Reject, not
clamp: see *Notes*. (3) `DayNumber` normalises both local midnights onto UTC calendar dates before dividing, so a 23-hour
spring-forward day is still one day. The tests that let the first bug ship green — rejection tests that never looked at the call
log — gain the `len(h.log.calls) != 0` assertion, and the two dead fake error fields get a test each.

**Tech stack:** Go 1.25, Gin, `pgx/v5`, `go-redis/v9` — nothing new. `rg` is not installed: use `grep -n`.

**Run every command from `backend/` inside the worktree** (`cd .worktrees/quests-daily-quest-suite-and-progress-recording/backend`)
unless the step says otherwise. Commit per task with the trailer `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>` after a
blank line. Do not rebase or merge anything into the branch; `main` is unchanged since it was cut.

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/quests/day.go` | `DayNumber` counts calendar days; new constants `MaxDurationSeconds`, `MaxDailySeconds` beside `TargetSeconds` |
| `backend/internal/quests/day_test.go` | DST table test — Europe/London, America/New_York, Australia/Sydney, both directions |
| `backend/internal/quests/repo.go` | `QuestRepo.CheckExercise` (interface, SQL, `PgRepo` impl) |
| `backend/internal/quests/service.go` | `ErrInvalidDuration`; `RecordProgress` validates duration → profile/roadmap → exercise → daily ceiling, then writes |
| `backend/internal/quests/handler.go` | maps `ErrInvalidDuration` → 400 `invalid_request` |
| `backend/internal/quests/fakes_test.go` | `fakeQuestRepo.CheckExercise`; `markErr` removed (no longer a path) |
| `backend/internal/quests/service_test.go` | rejection tests assert zero writes; out-of-range and daily-ceiling tests; `counter.err` / `progress.err` tests |
| `backend/internal/quests/handler_test.go` | 404 test asserts zero writes; 400 table gains the oversized magnitudes |
| `backend/internal/quests/integration_test.go` | the reviewer's two reproductions against real Postgres/Redis: 404 leaves both stores untouched, `1e14` is rejected, the day is not bricked |
| `harness/CODEMAP.md` | `quests` paragraph: validate-before-write, the two bounds, DST-safe `day_number` |

---

## Tasks

### Task 1: `DayNumber` counts calendar days, not elapsed hours

**Files:**
- Modify: `backend/internal/quests/day.go:33-48`
- Test: `backend/internal/quests/day_test.go`

Root cause (`day.go:40`): `int(today.Sub(start).Hours()/24) + 1`. `Sub` is absolute elapsed time; between two local midnights that
straddle a spring-forward only 23 hours elapse, the quotient truncates, and the learner is one day behind for the rest of the roadmap.
Fall-back days are 25 hours and truncate correctly by luck, so they are regression cases, not failing cases.

- [ ] **Step 1: Write the failing table test**

Append to `backend/internal/quests/day_test.go`:

```go
func TestDayNumberCountsCalendarDaysAcrossDSTTransitions(t *testing.T) {
	// DayNumber must count calendar days in the user's zone. A spring-forward
	// day is 23h long and a fall-back day 25h; dividing elapsed hours by 24
	// loses a day at every spring-forward (reviewer 2026-09-22). All 2026
	// transitions: Europe/London 03-29 / 10-25, America/New_York 03-08 / 11-01,
	// Australia/Sydney 10-04 (forward) / 04-05 (back).
	at := func(loc *time.Location, y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, 9, 0, 0, 0, loc)
	}
	tests := []struct {
		zone    string
		created [3]int // y, m, d — 09:00 local
		now     [3]int // y, m, d — 09:00 local
		want    int
	}{
		// Europe/London, spring-forward 2026-03-29
		{"Europe/London", [3]int{2026, 3, 25}, [3]int{2026, 3, 28}, 4},
		{"Europe/London", [3]int{2026, 3, 25}, [3]int{2026, 3, 29}, 5},
		{"Europe/London", [3]int{2026, 3, 25}, [3]int{2026, 3, 30}, 6},  // was 5
		{"Europe/London", [3]int{2026, 3, 25}, [3]int{2026, 4, 21}, 28}, // was 27: day 28 reached a day late
		// Europe/London, fall-back 2026-10-25 (25h day — unchanged, regression guard)
		{"Europe/London", [3]int{2026, 10, 20}, [3]int{2026, 10, 25}, 6},
		{"Europe/London", [3]int{2026, 10, 20}, [3]int{2026, 10, 26}, 7},
		// America/New_York, spring-forward 2026-03-08
		{"America/New_York", [3]int{2026, 3, 5}, [3]int{2026, 3, 7}, 3},
		{"America/New_York", [3]int{2026, 3, 5}, [3]int{2026, 3, 8}, 4},
		{"America/New_York", [3]int{2026, 3, 5}, [3]int{2026, 3, 9}, 5}, // was 4
		// America/New_York, fall-back 2026-11-01
		{"America/New_York", [3]int{2026, 10, 28}, [3]int{2026, 11, 1}, 5},
		{"America/New_York", [3]int{2026, 10, 28}, [3]int{2026, 11, 2}, 6},
		// Australia/Sydney, spring-forward 2026-10-04 (southern hemisphere)
		{"Australia/Sydney", [3]int{2026, 9, 29}, [3]int{2026, 10, 3}, 5},
		{"Australia/Sydney", [3]int{2026, 9, 29}, [3]int{2026, 10, 4}, 6},
		{"Australia/Sydney", [3]int{2026, 9, 29}, [3]int{2026, 10, 5}, 7}, // was 6
		// Australia/Sydney, fall-back 2026-04-05
		{"Australia/Sydney", [3]int{2026, 4, 1}, [3]int{2026, 4, 5}, 5},
		{"Australia/Sydney", [3]int{2026, 4, 1}, [3]int{2026, 4, 6}, 6},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("%s %04d-%02d-%02d -> %04d-%02d-%02d", tt.zone,
			tt.created[0], tt.created[1], tt.created[2], tt.now[0], tt.now[1], tt.now[2])
		t.Run(name, func(t *testing.T) {
			loc := Location(tt.zone)
			if loc == time.UTC {
				t.Fatalf("Location(%q) fell back to UTC — tzdata missing on this machine", tt.zone)
			}
			created := at(loc, tt.created[0], time.Month(tt.created[1]), tt.created[2])
			now := at(loc, tt.now[0], time.Month(tt.now[1]), tt.now[2])
			if got := DayNumber(created, now, loc); got != tt.want {
				t.Errorf("DayNumber = %d, want %d", got, tt.want)
			}
		})
	}

	// Inside the skipped hour itself: 01:30Z on 2026-03-29 is 02:30 BST, still day 5.
	loc := Location("Europe/London")
	created := at(loc, 2026, time.March, 25)
	if got := DayNumber(created, time.Date(2026, time.March, 29, 1, 30, 0, 0, time.UTC), loc); got != 5 {
		t.Errorf("DayNumber inside the transition hour = %d, want 5", got)
	}
}
```

Add `"fmt"` to the file's import block:

```go
import (
	"fmt"
	"testing"
	"time"
)
```

- [ ] **Step 2: Run it and confirm it fails on exactly the spring-forward cases**

```sh
go test ./internal/quests/... -run 'DSTTransitions' -v 2>&1 | grep -E '^(=== RUN|\s+--- (FAIL|PASS)|.*DayNumber =)' | grep -E 'FAIL|DayNumber ='
```
Expected: four `--- FAIL` subtests and only those — `Europe/London 2026-03-25 -> 2026-03-30` (`DayNumber = 5, want 6`),
`… -> 2026-04-21` (`27, want 28`), `America/New_York 2026-03-05 -> 2026-03-09` (`4, want 5`),
`Australia/Sydney 2026-09-29 -> 2026-10-05` (`6, want 7`). Every fall-back and pre-transition case passes. If a `tzdata missing`
fatal appears instead, stop: the machine lacks zoneinfo and the existing `Asia/Ho_Chi_Minh` tests would be failing too.

- [ ] **Step 3: Count calendar days**

Replace `DayNumber` in `backend/internal/quests/day.go:33-48` with:

```go
// DayNumber is the 1-based day of the roadmap, counted in calendar days in the
// user's own timezone and clamped to 1..RoadmapDays. Clamping at the top means a
// learner past day 28 keeps seeing day 28 — see the plan's open questions.
//
// It counts calendar days, not elapsed hours: the two local midnights are
// re-expressed as UTC dates before subtracting, so a 23-hour spring-forward day
// or a 25-hour fall-back day is still exactly one day. Dividing time.Sub by 24h
// lost a day at every spring-forward (reviewer 2026-09-22).
func DayNumber(createdAt, now time.Time, loc *time.Location) int {
	start := startOfDay(createdAt, loc)
	today := startOfDay(now, loc)

	su := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	tu := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	days := int(tu.Sub(su)/(24*time.Hour)) + 1
	if days < 1 {
		return 1
	}
	if days > RoadmapDays {
		return RoadmapDays
	}
	return days
}
```

`startOfDay` (`day.go:50-53`) is unchanged; `LocalDate` was already correct.

- [ ] **Step 4: Run the whole package and confirm everything passes**

```sh
go test ./internal/quests/... -count=1 -run 'DayNumber|LocalDate|Location|Daily' -v 2>&1 | grep -c -- '--- PASS'
go test ./internal/quests/... -count=1
```
Expected: the first line prints at least 30 (16 DST subtests + the transition-hour check inside the same test + the existing
`DayNumber`/`Daily` cases); the second prints `ok`. `TestDayNumberCountsCalendarDaysFromCreation`,
`TestDayNumberCrossesTheBoundaryInTheUsersTimezone` and `TestDailyClampsPastDay28` must still pass — the UTC and Ho Chi Minh
cases are unaffected by the formula change.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/quests/day.go backend/internal/quests/day_test.go && git commit -m "quests: DayNumber counts calendar days, not elapsed hours

A spring-forward day is 23h long, so dividing time.Sub by 24h truncated a
day for every DST learner (Europe/London, America/New_York, Australia/Sydney
table tests). Both local midnights are now compared as UTC dates.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>" && cd backend
```

---

### Task 2: Validate the exercise before any write

**Files:**
- Modify: `backend/internal/quests/repo.go:44-52` (interface), `:60-89` (SQL), `:135-144` (impl — add the new method beside it)
- Modify: `backend/internal/quests/fakes_test.go:48-88`
- Modify: `backend/internal/quests/service.go:47-90`
- Modify: `backend/internal/quests/service_test.go:203-211`
- Modify: `backend/internal/quests/handler_test.go:151-162`

Root cause (`service.go:75-90`): `counter.Add` → `progress.Upsert` → `quests.MarkComplete`; `MarkComplete` (`repo.go:135-144`) is
the only ownership check and runs last. Validation is a read, so it can move ahead of the INCRBY without touching the §5.2 write
order that `TestRecordProgressIncrementsRedisBeforeWritingPostgres` pins.

- [ ] **Step 1: Add the read-only lookup to the interface, the Postgres repo and the fake**

`backend/internal/quests/repo.go` — extend `QuestRepo` (`:44-52`):

```go
// QuestRepo is the Postgres read side plus the one exercise write.
type QuestRepo interface {
	Profile(ctx context.Context, userID string) (Profile, error)
	ActiveRoadmap(ctx context.Context, userID string) (Roadmap, error)
	ExercisesForDay(ctx context.Context, roadmapID string, day int) ([]Exercise, error)
	// CheckExercise is the read-only ownership check RecordProgress runs
	// before it writes anything: nil when exerciseID is on roadmapID for
	// day, ErrExerciseNotFound otherwise. Unknown ids and other users' ids
	// are deliberately indistinguishable.
	CheckExercise(ctx context.Context, roadmapID, exerciseID string, day int) error
	// MarkComplete sets is_completed and returns ErrExerciseNotFound when the
	// exercise is not on roadmapID.
	MarkComplete(ctx context.Context, roadmapID, exerciseID string) error
}
```

Add to the `const (...)` block after `exercisesForDaySQL`:

```go
	checkExerciseSQL = `
SELECT 1
FROM exercises
WHERE id = $1 AND roadmap_id = $2 AND day_number = $3`
```

Add the method after `ExercisesForDay` (before `MarkComplete`):

```go
func (r *PgRepo) CheckExercise(ctx context.Context, roadmapID, exerciseID string, day int) error {
	var one int
	err := r.Pool.QueryRow(ctx, checkExerciseSQL, exerciseID, roadmapID, day).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrExerciseNotFound
	}
	if err != nil {
		return fmt.Errorf("quests: checking exercise: %w", err)
	}
	return nil
}
```

`backend/internal/quests/fakes_test.go` — the fake looks the exercise up in the day's fixtures. Reads are not logged (neither are
`Profile` and `ActiveRoadmap`), so the call log stays a log of writes. Remove `markErr`: with ownership checked first there is no
longer a test that needs `MarkComplete` to fail with `ErrExerciseNotFound`, and a dead field is what the fourth idea is about.

```go
type fakeQuestRepo struct {
	log       *callLog
	timezone  string
	roadmap   *Roadmap
	exercises map[int][]Exercise // by day_number
	completed map[string]bool
}
```

```go
func (f *fakeQuestRepo) CheckExercise(_ context.Context, _, exerciseID string, day int) error {
	for _, e := range f.exercises[day] {
		if e.ID == exerciseID {
			return nil
		}
	}
	return ErrExerciseNotFound
}

func (f *fakeQuestRepo) MarkComplete(_ context.Context, _, exerciseID string) error {
	f.log.add("MARK COMPLETE %s", exerciseID)
	f.completed[exerciseID] = true
	return nil
}
```

```sh
go build ./... && go vet ./...
```
Expected: no output — the service does not call the new method yet, and the two tests that set `markErr` still compile only
because Step 2 rewrites them; if `go vet` reports `h.quests.markErr undefined`, do Step 2 before re-running.

- [ ] **Step 2: Make the rejection tests assert zero writes (they fail against the current ordering)**

Replace `TestRecordProgressRejectsAnExerciseOutsideTheActiveRoadmap` (`service_test.go:203-211`) with two tests:

```go
func TestRecordProgressRejectsAnExerciseOutsideTheActiveRoadmap(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	_, err := h.svc.RecordProgress(context.Background(), "u1", "someone-elses-exercise", 600)
	if !errors.Is(err, ErrExerciseNotFound) {
		t.Fatalf("err = %v, want ErrExerciseNotFound", err)
	}
	// A 404 must leave Redis and daily_progress byte for byte as they were:
	// this line was missing when the write-before-validate defect shipped.
	if len(h.log.calls) != 0 {
		t.Errorf("a rejected call touched Redis/Postgres: %v", h.log.calls)
	}
}

func TestRecordProgressRejectsAnExerciseFromAnotherDay(t *testing.T) {
	// The roadmap is on day 2; ex-1-reading is a real, owned exercise from
	// day 1. Progress is only recordable against today's tasks.
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	_, err := h.svc.RecordProgress(context.Background(), "u1", "ex-1-reading", 600)
	if !errors.Is(err, ErrExerciseNotFound) {
		t.Fatalf("err = %v, want ErrExerciseNotFound", err)
	}
	if len(h.log.calls) != 0 {
		t.Errorf("a rejected call touched Redis/Postgres: %v", h.log.calls)
	}
}
```

Replace `TestProgressHandlerReturns404ForAnUnknownExercise` (`handler_test.go:151-162`) with:

```go
func TestProgressHandlerReturns404ForAnUnknownExercise(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))

	w := postJSON(newQuestRouter(h.svc, "u1"), "/api/v1/quests/progress", `{"exercise_id":"nope","duration_seconds":600}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"error":"exercise_not_found"`) {
		t.Errorf("body = %s, want the exercise_not_found error", w.Body.String())
	}
	if len(h.log.calls) != 0 {
		t.Errorf("a 404 touched Redis/Postgres: %v", h.log.calls)
	}
}
```

```sh
go test ./internal/quests/... -count=1 -run 'RejectsAnExerciseOutsideTheActiveRoadmap|RejectsAnExerciseFromAnotherDay|Returns404ForAnUnknownExercise' -v 2>&1 | grep -E -- '--- (FAIL|PASS)|err = |touched'
```
Expected: all three `--- FAIL`. The two service tests fail at the `Fatalf` (`err = <nil>, want ErrExerciseNotFound` — the fake's
`MarkComplete` no longer errors, so the request succeeds and writes). The handler test fails with `status = 200, want 404`. This is
the defect made visible: nothing rejects the exercise before the writes.

- [ ] **Step 3: Move the check ahead of the writes**

Replace `RecordProgress` and its doc comment in `backend/internal/quests/service.go:47-90` (keep the hook / pet-state tail as is):

```go
// RecordProgress validates, then implements §5.2 steps 2-4 in that order:
//
//  0. Reads only — resolve the profile and active roadmap, then CheckExercise:
//     the exercise must be on the caller's active roadmap and on today's
//     day_number, or the call is rejected with ErrExerciseNotFound before a
//     single write. A rejected request leaves Redis and daily_progress
//     untouched (reviewer 2026-09-22).
//  1. INCRBY the Redis counter (+ EXPIRE) and read the running total back.
//  2. Upsert daily_progress from that total — Redis is the single source of
//     truth for the day, so bursts of calls cannot disagree.
//  3. Mark the exercise complete.
//  4. Fire Pet.OnTargetMet, but only on the call that crossed 1800s, then read
//     Pet.State so the §6.2 response carries pet_health / streak_count.
//
// The ordering is a contract, not an implementation detail: see the call-log
// test in service_test.go.
func (s *Service) RecordProgress(ctx context.Context, userID, exerciseID string, seconds int64) (ProgressResult, error) {
	if seconds <= 0 {
		return ProgressResult{}, fmt.Errorf("quests: duration_seconds must be positive, got %d", seconds)
	}

	profile, err := s.quests.Profile(ctx, userID)
	if err != nil {
		return ProgressResult{}, err
	}
	roadmap, err := s.quests.ActiveRoadmap(ctx, userID)
	if err != nil {
		return ProgressResult{}, err
	}

	loc := Location(profile.Timezone)
	now := s.now()
	date := LocalDate(now, loc)
	day := DayNumber(roadmap.CreatedAt, now, loc)

	if err := s.quests.CheckExercise(ctx, roadmap.ID, exerciseID, day); err != nil {
		return ProgressResult{}, err
	}

	total, err := s.counter.Add(ctx, userID, date, seconds)
	if err != nil {
		return ProgressResult{}, err
	}

	// newly_met is derived from the counter alone: this call crossed the target
	// iff the new total is at or past it and the previous total was not.
	targetMet := total >= TargetSeconds
	newlyMet := targetMet && total-seconds < TargetSeconds

	if err := s.progress.Upsert(ctx, userID, date, int(total/60), targetMet); err != nil {
		return ProgressResult{}, err
	}
	// MarkComplete keeps its own ErrExerciseNotFound for the race where the
	// row vanished between CheckExercise and here; the handler still maps it
	// to 404.
	if err := s.quests.MarkComplete(ctx, roadmap.ID, exerciseID); err != nil {
		return ProgressResult{}, err
	}
```

(The `seconds <= 0` guard is replaced in Task 3; leave it as is here so this commit is one change.)

- [ ] **Step 4: Run the package and confirm the ordering contract still holds**

```sh
go test ./internal/quests/... -count=1 -run 'RejectsAnExerciseOutsideTheActiveRoadmap|RejectsAnExerciseFromAnotherDay|Returns404ForAnUnknownExercise|IncrementsRedisBeforeWritingPostgres' -v 2>&1 | grep -E -- '--- (FAIL|PASS)'
go test ./internal/quests/... -count=1
grep -n 'markErr' internal/quests/*_test.go
```
Expected: four `--- PASS` (the call-log `want` list in `TestRecordProgressIncrementsRedisBeforeWritingPostgres` is unchanged —
`CheckExercise` is a read and is not logged); `ok`; the grep prints nothing.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/quests/repo.go backend/internal/quests/service.go backend/internal/quests/fakes_test.go backend/internal/quests/service_test.go backend/internal/quests/handler_test.go && git commit -m "quests: check exercise ownership and day before any progress write

RecordProgress ran INCRBY and the daily_progress upsert before MarkComplete,
the only ownership check, so a 404 exercise_not_found still moved both
stores. QuestRepo.CheckExercise is a read-only lookup (roadmap + today's
day_number) run first; the §5.2 write order is unchanged. The rejection
tests now assert an empty call log — the missing line that let this ship.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>" && cd backend
```

---

### Task 3: Bound `duration_seconds` per call and per day

**Files:**
- Modify: `backend/internal/quests/day.go:10-11` (constants beside `TargetSeconds`)
- Modify: `backend/internal/quests/service.go` (`ErrInvalidDuration`, the guard, the ceiling read)
- Modify: `backend/internal/quests/handler.go:64-75`
- Modify: `backend/internal/quests/service_test.go:213-225`
- Modify: `backend/internal/quests/handler_test.go:133-149`

Root cause: `handler.go:39` `binding:"required,gt=0"` has no maximum; `service.go` guards `seconds <= 0` only; `int(total/60)` is
written to `minutes_spent INT` (`internal/store/migrations/0001_init.up.sql:48`). Past ~1.29e11 seconds the upsert fails, the INCRBY
has already committed, and every later call for that user fails for the 48h TTL. Decision (evaluator, in the idea's `## Evaluation`):
reject above a per-call cap and at a per-day ceiling, both **before** the INCRBY, so the failure is unreachable and no compensating
write is needed.

- [ ] **Step 1: Write the failing service tests**

Replace `TestRecordProgressRejectsNonPositiveSeconds` (`service_test.go:213-225`) with:

```go
func TestRecordProgressRejectsAnOutOfRangeDuration(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	for _, seconds := range []int64{0, -1, -600, MaxDurationSeconds + 1, 1_000_000_000, 100_000_000_000_000} {
		_, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", seconds)
		if !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("seconds = %d: err = %v, want ErrInvalidDuration", seconds, err)
		}
	}
	if len(h.log.calls) != 0 {
		t.Errorf("a rejected call touched Redis/Postgres: %v", h.log.calls)
	}
}

func TestRecordProgressAcceptsTheMaximumPerCallDuration(t *testing.T) {
	// The bound is inclusive: one full hour in a single report is allowed.
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", MaxDurationSeconds)
	if err != nil {
		t.Fatalf("RecordProgress(%d) = %v, want nil", MaxDurationSeconds, err)
	}
	if out.DailySecondsSpent != MaxDurationSeconds || out.DailyMinutesSpent != MaxDurationSeconds/60 {
		t.Errorf("out = %+v, want %ds/%dm", out, MaxDurationSeconds, MaxDurationSeconds/60)
	}
}

func TestRecordProgressRejectsCrossingTheDailyCeiling(t *testing.T) {
	// minutes_spent is INT (§3.2); the ceiling keeps the counter far from it
	// and is checked against the running total BEFORE the INCRBY.
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.counter.totals["u1|2026-09-22"] = MaxDailySeconds - 100

	_, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 101)
	if !errors.Is(err, ErrInvalidDuration) {
		t.Fatalf("101s over the ceiling: err = %v, want ErrInvalidDuration", err)
	}
	if len(h.log.calls) != 0 {
		t.Fatalf("a rejected call touched Redis/Postgres: %v", h.log.calls)
	}

	// Landing exactly on the ceiling is allowed …
	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 100)
	if err != nil {
		t.Fatalf("100s to reach the ceiling exactly: %v", err)
	}
	if out.DailySecondsSpent != MaxDailySeconds {
		t.Errorf("DailySecondsSpent = %d, want %d", out.DailySecondsSpent, MaxDailySeconds)
	}
	// … and one more second is not.
	if _, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-practice", 1); !errors.Is(err, ErrInvalidDuration) {
		t.Errorf("1s past the ceiling: err = %v, want ErrInvalidDuration", err)
	}
	if got := h.counter.totals["u1|2026-09-22"]; got != MaxDailySeconds {
		t.Errorf("counter = %d after the rejection, want it untouched at %d", got, MaxDailySeconds)
	}
}
```

Extend `TestProgressHandlerRejectsABadBody` (`handler_test.go:133-149`) — add the oversized magnitudes and assert the error code:

```go
func TestProgressHandlerRejectsABadBody(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	r := newQuestRouter(h.svc, "u1")

	for _, body := range []string{
		`{}`,
		`{"exercise_id":"ex-2-reading"}`,
		`{"exercise_id":"ex-2-reading","duration_seconds":0}`,
		`{"exercise_id":"ex-2-reading","duration_seconds":-5}`,
		`{"exercise_id":"ex-2-reading","seconds":600}`, // the pre-§6.2 field name is not an alias
		`nonsense`,
		// Above MaxDurationSeconds (3600) — a task is 10 minutes:
		`{"exercise_id":"ex-2-reading","duration_seconds":3601}`,
		`{"exercise_id":"ex-2-reading","duration_seconds":1000000000}`,
		`{"exercise_id":"ex-2-reading","duration_seconds":100000000000000}`, // the reviewer's 48h brick
		`{"exercise_id":"ex-2-reading","duration_seconds":99999999999999999999}`, // does not fit int64
	} {
		w := postJSON(r, "/api/v1/quests/progress", body)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %q → status %d, want 400", body, w.Code)
		}
		if !strings.Contains(w.Body.String(), `"error":"invalid_request"`) {
			t.Errorf("body %q → %s, want the invalid_request error", body, w.Body.String())
		}
	}
	if len(h.log.calls) != 0 {
		t.Errorf("a 400 touched Redis/Postgres: %v", h.log.calls)
	}
}
```

```sh
go vet ./internal/quests/ 2>&1 | head -3
```
Expected: `undefined: MaxDurationSeconds`, `undefined: ErrInvalidDuration` — the tests do not compile yet.

- [ ] **Step 2: Add the constants and the sentinel**

`backend/internal/quests/day.go` — after `TargetSeconds` (`:10-11`):

```go
// TargetSeconds is the daily goal from §1: at least 30 minutes.
const TargetSeconds = 1800

// MaxDurationSeconds bounds one POST /quests/progress report. A §6.2 task is
// 10 minutes and the whole day is 30; one hour is a learner who left a task
// open, not a plausible single sitting. Anything larger is a client bug
// (milliseconds, an overflowed Number) or abuse, and is answered 400 before
// the INCRBY. Chosen so that even the per-day ceiling below cannot be crossed
// by less than 24 max-size reports.
const MaxDurationSeconds = 3600

// MaxDailySeconds bounds the counter for one local day: nobody studies more
// than a day in a day. RecordProgress rejects, before the INCRBY, any report
// that would push the running total past it. daily_progress.minutes_spent is
// INT (§3.2); without a ceiling ~1.29e11 accumulated seconds overflow the
// upsert and every later call for that user fails for the 48h TTL (reviewer
// 2026-09-22). The check races with concurrent calls, but each can overshoot
// by at most MaxDurationSeconds, so the INT limit stays ~10^6 concurrent
// max-size requests away.
const MaxDailySeconds = 86400
```

`backend/internal/quests/service.go` — after the imports, before `ProgressResult`:

```go
// ErrInvalidDuration means duration_seconds is outside 1..MaxDurationSeconds
// or would push the day past MaxDailySeconds. The handler maps it to 400.
var ErrInvalidDuration = errors.New("quests: invalid duration_seconds")
```

and add `"errors"` to the import block:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
)
```

- [ ] **Step 3: Guard per call, then read the counter and guard per day, then write**

In `RecordProgress` replace the opening guard:

```go
	if seconds <= 0 || seconds > MaxDurationSeconds {
		return ProgressResult{}, fmt.Errorf("%w: must be in 1..%d, got %d", ErrInvalidDuration, MaxDurationSeconds, seconds)
	}
```

and replace the `total, err := s.counter.Add(...)` block (immediately after the `CheckExercise` check from Task 2) with:

```go
	// Still a read: the running total decides whether this report fits under
	// the daily ceiling. Only then does the §5.2 write sequence start.
	total, err := s.counter.Total(ctx, userID, date)
	if err != nil {
		return ProgressResult{}, err
	}
	if total+seconds > MaxDailySeconds {
		return ProgressResult{}, fmt.Errorf("%w: %d + %d would exceed the daily ceiling of %d", ErrInvalidDuration, total, seconds, MaxDailySeconds)
	}

	total, err = s.counter.Add(ctx, userID, date, seconds)
	if err != nil {
		return ProgressResult{}, err
	}
```

Update the doc comment's step 0 to end with: `Then read the counter and reject with ErrInvalidDuration if seconds is outside
1..MaxDurationSeconds or the day would pass MaxDailySeconds.`

`backend/internal/quests/handler.go` — add a case to the `switch` in `ProgressHandler` (`:64-75`), first:

```go
		switch {
		case errors.Is(err, ErrInvalidDuration):
			// Same code as a binding failure: the request is malformed, not
			// the state. Wire shape per §6.2 is unchanged.
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		case errors.Is(err, ErrNoActiveRoadmap):
```

- [ ] **Step 4: Run the tests**

```sh
go test ./internal/quests/... -count=1 -run 'OutOfRangeDuration|MaximumPerCallDuration|DailyCeiling|RejectsABadBody' -v 2>&1 | grep -E -- '--- (FAIL|PASS)'
go test ./internal/quests/... -count=1 -run 'IncrementsRedisBeforeWritingPostgres|CrossingExactly1800|FurtherProgress|MinutesUseIntegerDivision' -v 2>&1 | grep -E -- '--- (FAIL|PASS)'
go test ./internal/quests/... -count=1
grep -n 'MaxDurationSeconds = 3600\|MaxDailySeconds = 86400' internal/quests/day.go
grep -n 'ErrInvalidDuration' internal/quests/handler.go
```
Expected: four `--- PASS`, then four `--- PASS` (the `Total` read is not logged, so the ordering `want` list is unchanged), `ok`,
two hits, one hit.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/quests/day.go backend/internal/quests/service.go backend/internal/quests/handler.go backend/internal/quests/service_test.go backend/internal/quests/handler_test.go && git commit -m "quests: bound duration_seconds per call (3600) and per day (86400)

duration_seconds had no upper bound; one 1e14 report overflowed
daily_progress.minutes_spent INT after the INCRBY had committed, and every
later call for that user failed for the 48h TTL. Both bounds are checked
before the INCRBY and answer the existing 400 invalid_request via
ErrInvalidDuration; the §6.2 shapes are unchanged.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>" && cd backend
```

---

### Task 4: Exercise the two dead fake error paths

**Files:**
- Modify: `backend/internal/quests/service_test.go` (append)

`fakeCounter.err` (`fakes_test.go:22`) and `fakeProgressRepo.err` (`fakes_test.go:96`) are set by no test. These two tests make
them live and pin the behaviour this plan relies on: a Redis failure writes nothing, and a Postgres failure after the INCRBY does not
leave the counter unusable (the second blocker's "counter stays usable" ask — now trivially true because the overflow is unreachable,
but proven rather than argued).

- [ ] **Step 1: Write the tests**

Append to `backend/internal/quests/service_test.go`:

```go
func TestARedisFailureWritesNothing(t *testing.T) {
	// Counter.Total and Counter.Add both fail: the error surfaces and neither
	// Postgres write runs — the §5.2 order means Redis failing first is clean.
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	boom := errors.New("redis down")
	h.counter.err = boom

	_, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 600)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the redis error", err)
	}
	if len(h.log.calls) != 0 {
		t.Errorf("a failed Redis call still reached Postgres: %v", h.log.calls)
	}
}

func TestAnUpsertFailureAfterTheIncrbyLeavesTheCounterUsable(t *testing.T) {
	// Redis is the source of truth for the day (plan, Notes): if the
	// daily_progress upsert fails the increment stays, the error surfaces,
	// MarkComplete does not run, and the next call re-derives the row from the
	// counter and succeeds. (Recovering the once-only hook on the crossing
	// call is a separate inbox bug, not asserted here.)
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	boom := errors.New("postgres down")
	h.progress.err = boom

	_, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 600)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the postgres error", err)
	}
	want := []string{"INCRBY u1|2026-09-22 600", "EXPIRE u1|2026-09-22"}
	if !reflect.DeepEqual(h.log.calls, want) {
		t.Fatalf("calls = %v, want only the Redis increment %v", h.log.calls, want)
	}

	h.progress.err = nil
	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-practice", 600)
	if err != nil {
		t.Fatalf("the call after recovery failed: %v", err)
	}
	if out.DailySecondsSpent != 1200 || out.DailyMinutesSpent != 20 {
		t.Errorf("out = %+v, want the counter's 1200s/20m — the first increment was kept", out)
	}
	if row := h.progress.rows["u1|2026-09-22"]; row.minutes != 20 {
		t.Errorf("daily_progress minutes = %d, want 20 re-derived from the counter", row.minutes)
	}
	if !h.quests.completed["ex-2-practice"] || h.quests.completed["ex-2-reading"] {
		t.Errorf("completed = %v, want only ex-2-practice (the failed call must not mark its exercise)", h.quests.completed)
	}
}
```

- [ ] **Step 2: Run them**

```sh
go test ./internal/quests/... -count=1 -run 'RedisFailureWritesNothing|UpsertFailureAfterTheIncrby' -v 2>&1 | grep -E -- '--- (FAIL|PASS)'
go test ./internal/quests/... -count=1
```
Expected: two `--- PASS` (these pin current behaviour — they should pass first time; if the first one fails with a non-empty log,
Task 3's `Total` read is in the wrong place), then `ok`.

- [ ] **Step 3: Commit**

```sh
cd .. && git add backend/internal/quests/service_test.go && git commit -m "quests: test the Redis-down and upsert-failure paths

fakeCounter.err and fakeProgressRepo.err were never set. A Redis failure
now provably writes nothing; an upsert failure after the INCRBY provably
leaves the counter usable for the next call.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>" && cd backend
```

---

### Task 5: Reproduce the review's evidence against real services, and update CODEMAP

**Files:**
- Modify: `backend/internal/quests/integration_test.go:85-122`
- Modify: `harness/CODEMAP.md:12`

The reviewer's two reproductions were live HTTP calls; the integration test asserts the same facts one layer down (the service
against real Postgres/Redis), and the runtime proof in *Verification* repeats them over HTTP. The rejected id must be a **real
exercise of another user** (as the reviewer did), not a random string: `exercises.id` is `UUID`, and a non-UUID string makes
Postgres error out with a 500 in `MarkComplete` and `CheckExercise` alike — pre-existing, tracked by the inbox's
`progress-request-validation-is-incomplete-outside-the-gin-bi.md`, not this plan.

- [ ] **Step 1: Extend the integration test**

In `backend/internal/quests/integration_test.go`, after the block ending `t.Error("exercises.is_completed = false after progress
was recorded against it")` (`:107-113`) and before `again, err := svc.Daily(ctx, userID)` (`:115`), insert:

```go
	// --- Review reproduction 1: a 404 must leave both stores untouched. ---
	// Another user's real exercise (ids are UUIDs; a random string would be a
	// Postgres type error, not a 404 — see the plan).
	const otherGid = "google-quests-integration-other"
	var otherID string
	_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, otherGid)
	if err := pg.Pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, target_goal, timezone) VALUES ($1,$2,$3,$4) RETURNING id`,
		"quests-other@example.com", otherGid, "", "UTC").Scan(&otherID); err != nil {
		t.Fatalf("inserting the other user: %v", err)
	}
	t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, otherGid) })
	if _, err := store.SeedDemoRoadmap(ctx, pg.Pool, otherID); err != nil {
		t.Fatalf("SeedDemoRoadmap(other): %v", err)
	}
	otherSuite, err := svc.Daily(ctx, otherID)
	if err != nil {
		t.Fatalf("Daily(other): %v", err)
	}

	_, err = svc.RecordProgress(ctx, userID, otherSuite.Tasks[0].ID, 999)
	if !errors.Is(err, ErrExerciseNotFound) {
		t.Fatalf("progress against another user's exercise: err = %v, want ErrExerciseNotFound", err)
	}
	assertUntouched := func(t *testing.T, what string) {
		t.Helper()
		total, err := counter.Total(ctx, userID, LocalDate(now, time.UTC))
		if err != nil {
			t.Fatalf("%s: reading counter: %v", what, err)
		}
		if total != 1800 {
			t.Errorf("%s: daily:accumulated = %d, want 1800 untouched", what, total)
		}
		var m int
		var met bool
		if err := pg.Pool.QueryRow(ctx,
			`SELECT minutes_spent, is_target_met FROM daily_progress WHERE user_id = $1 AND date = $2::date`,
			userID, LocalDate(now, time.UTC)).Scan(&m, &met); err != nil {
			t.Fatalf("%s: reading daily_progress: %v", what, err)
		}
		if m != 30 || !met {
			t.Errorf("%s: daily_progress = (%d, %t), want (30, true) untouched", what, m, met)
		}
	}
	assertUntouched(t, "after the 404")
	var otherCompleted bool
	if err := pg.Pool.QueryRow(ctx, `SELECT is_completed FROM exercises WHERE id = $1`, otherSuite.Tasks[0].ID).Scan(&otherCompleted); err != nil {
		t.Fatalf("reading the other user's exercise: %v", err)
	}
	if otherCompleted {
		t.Error("the other user's exercise was marked complete by a rejected request")
	}

	// --- Review reproduction 2: an oversized report is a 400, not a 48h brick. ---
	for _, seconds := range []int64{MaxDurationSeconds + 1, 100_000_000_000_000} {
		if _, err := svc.RecordProgress(ctx, userID, suite.Tasks[1].ID, seconds); !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("duration %d: err = %v, want ErrInvalidDuration", seconds, err)
		}
	}
	assertUntouched(t, "after the oversized reports")

	// The day is still writable afterwards — the reviewer's ordinary 60s call
	// was a 500 before this fix.
	out2, err := svc.RecordProgress(ctx, userID, suite.Tasks[1].ID, 60)
	if err != nil {
		t.Fatalf("an ordinary 60s report after the rejections failed: %v", err)
	}
	if out2.DailySecondsSpent != 1860 || out2.DailyMinutesSpent != 31 || out2.NewlyMet {
		t.Errorf("out = %+v, want 1860s/31m and not newly met", out2)
	}
```

Then change the final assertion (`:119`) so it reflects the extra 60 seconds:

```go
	if again.AccumulatedSeconds != 1860 || !again.IsTargetMet || !again.Tasks[0].IsCompleted || !again.Tasks[1].IsCompleted {
		t.Errorf("second Daily = %+v, want 1860s accumulated, target met, tasks[0] and [1] completed", again)
	}
```

Add `"errors"` to the import block:

```go
import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)
```

- [ ] **Step 2: Confirm it still skips without services, then run it for real**

```sh
env -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./internal/quests/... -count=1 -run Integration -v 2>&1 | grep -E 'SKIP|ok|FAIL'
```
Expected: `--- SKIP` and `ok`.

Bring up a scratch stack on non-default ports (the owner's containers hold 5432/6379/6380 — see the original plan's execution
summary), then run the integration job the way CI does:

```sh
POSTGRES_PORT=5433 REDIS_PORT=6381 docker compose up -d --wait
export TEST_DATABASE_URL='postgres://english:english@localhost:5433/english?sslmode=disable'
export TEST_REDIS_URL='redis://localhost:6381/0'
make test-integration 2>&1 | grep -E -- '--- (PASS|FAIL|SKIP)|^ok|^FAIL'
```
Expected: `--- PASS: TestIntegrationDailyAndProgressAgainstRealServices` and every other `TestIntegration*` `--- PASS`, no
`--- SKIP`, no `FAIL`. Leave the stack up for the runtime proof in *Verification*; `docker compose down` afterwards.

- [ ] **Step 3: Update CODEMAP**

In `harness/CODEMAP.md` line 12 (the `**quests**` bullet) make these three edits, keeping the rest of the paragraph verbatim:

1. Replace `computes `day_number` = calendar days since `roadmaps.created_at` in `users.timezone`, +1, clamped to 1..28,` with
   `computes `day_number` = calendar days since `roadmaps.created_at` in `users.timezone`, +1, clamped to 1..28 (calendar days, not
   elapsed hours ÷ 24 — DST-safe; `day_test.go` covers London/New York/Sydney across both transitions),`.
2. Replace `does `INCRBY daily:accumulated:{user_id}:{local-date}` + `EXPIRE` 48h **first**,` with
   `**validates before it writes** — `duration_seconds` must be in 1..`MaxDurationSeconds` (3600) and must not push the day past
   `MaxDailySeconds` (86400, checked against a `Counter.Total` read; both → 400 `invalid_request`), and `QuestRepo.CheckExercise`
   must find the exercise on the caller's active roadmap **and** today's `day_number` (else 404 `exercise_not_found`) — so a rejected
   request leaves Redis and `daily_progress` untouched; only then does it `INCRBY daily:accumulated:{user_id}:{local-date}` +
   `EXPIRE` 48h **first**,`.
3. Replace `a `service_test.go` call-log test pins that order.` with `a `service_test.go` call-log test pins that order and the
   rejection tests assert an empty call log.`

```sh
cd .. && grep -c 'CheckExercise\|MaxDailySeconds\|DST-safe' harness/CODEMAP.md && cd backend
```
Expected: `1` (all three phrases are on the one `quests` line; `grep -c` counts lines).

- [ ] **Step 4: Commit**

```sh
cd .. && git add backend/internal/quests/integration_test.go harness/CODEMAP.md && git commit -m "quests: integration-test the review's reproductions; CODEMAP

A 404 against another user's exercise leaves daily:accumulated and
daily_progress untouched; 3601 and 1e14 are rejected and the next ordinary
report still succeeds. CODEMAP records validate-before-write, the two
bounds and the DST-safe day_number.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>" && cd backend
```

---

## Verification

Run from the worktree root (`.worktrees/quests-daily-quest-suite-and-progress-recording`) unless noted. The first block is the
standard gate; the second re-runs the review's evidence; the third is the live runtime proof the review demanded.

```sh
cd backend && go build ./... && go vet ./...
# expect: no output

env -u TEST_DATABASE_URL -u TEST_REDIS_URL -u DATABASE_URL -u REDIS_URL go test ./... -count=1
# expect: ok for internal/auth, internal/config, internal/health, internal/quests, internal/store; no FAIL, no service needed

go test ./internal/quests/... -count=1 -run 'DSTTransitions' -v 2>&1 | grep -c -- '--- PASS'
# expect: 17 (16 table cases + the parent test); the transition-hour check is inside the parent

go test ./internal/quests/... -count=1 -run 'RejectsAnExerciseOutsideTheActiveRoadmap|RejectsAnExerciseFromAnotherDay|Returns404ForAnUnknownExercise' -v 2>&1 | grep -c -- '--- PASS'
# expect: 3 — blocker 1: each asserts len(h.log.calls) == 0

go test ./internal/quests/... -count=1 -run 'OutOfRangeDuration|MaximumPerCallDuration|DailyCeiling|RejectsABadBody' -v 2>&1 | grep -c -- '--- PASS'
# expect: 4 — blocker 2: per-call cap, inclusive bound, daily ceiling, handler 400 table

go test ./internal/quests/... -count=1 -run 'IncrementsRedisBeforeWritingPostgres|CrossingExactly1800|FurtherProgress' -v 2>&1 | grep -c -- '--- PASS'
# expect: 3 — the §5.2 ordering contract and the once-only hook are unchanged

go test ./internal/quests/... -count=1 -run 'RedisFailureWritesNothing|UpsertFailureAfterTheIncrby' -v 2>&1 | grep -c -- '--- PASS'
# expect: 2 — the formerly dead fake error fields

grep -n 'MaxDurationSeconds = 3600\|MaxDailySeconds = 86400' internal/quests/day.go
# expect: 2 hits — the documented bounds

grep -c 'CheckExercise' internal/quests/repo.go internal/quests/service.go
# expect: repo.go ≥ 3 (interface, SQL const, impl), service.go ≥ 1

grep -n 'ErrInvalidDuration' internal/quests/handler.go
# expect: 1 hit — mapped to 400

grep -c 'len(h.log.calls) != 0' internal/quests/service_test.go internal/quests/handler_test.go
# expect: service_test.go ≥ 5, handler_test.go ≥ 2

grep -n 'markErr' internal/quests/*_test.go
# expect: no hits

grep -n 'Hours()/24' internal/quests/day.go
# expect: no hits — the elapsed-hours formula is gone

grep -n '"duration_seconds"\|"user_answers"' internal/quests/handler.go
# expect: 2 hits — §6.2 request unchanged

grep -n '"daily_seconds_spent"\|"daily_minutes_spent"\|"pet_health"\|"streak_count"' internal/quests/service.go
# expect: 4 hits — §6.2 response unchanged

grep -rn --include='*.go' --exclude='*_test.go' '"total_seconds"\|"target_met"\|"newly_met"\|"exercises"\|"seconds"' internal/quests/
# expect: no hits

cd .. && python3 tools/harness/cli.py validate; echo exit=$?
# expect: exit=0

git log --oneline ef4b9fa..HEAD
# expect: 5 commits, one per task, each carrying the Co-Authored-By trailer

git status --short
# expect: clean
```

**Integration (the review's reproductions, service level):**

```sh
cd backend
POSTGRES_PORT=5433 REDIS_PORT=6381 docker compose up -d --wait
export TEST_DATABASE_URL='postgres://english:english@localhost:5433/english?sslmode=disable'
export TEST_REDIS_URL='redis://localhost:6381/0'
make test-integration 2>&1 | grep -E -- '--- (PASS|FAIL|SKIP)'
# expect: --- PASS for every TestIntegration* including TestIntegrationDailyAndProgressAgainstRealServices; no SKIP, no FAIL
```

**Runtime proof (the review's reproductions, over HTTP):** boot the real binary against the same scratch stack on a free port
(the original plan's execution summary used 18099; the reviewer 18123 — pick one that `lsof -i :PORT` shows free), mint a real HS256
bearer + Redis session and seed **two** users' demo roadmaps exactly as the executor's throwaway `cmd/devtoken` did (delete it again
before committing; it is not part of this plan's file structure). With `$TOKEN` for user A and `$OTHER_EX` = one of user B's
exercise ids, `$EX0`/`$EX1` = user A's day-1 exercise ids, `$UID` = user A's id, `$DAY` = today's UTC date:

```sh
curl -s -X POST localhost:$PORT/api/v1/quests/progress -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"exercise_id\":\"$EX0\",\"duration_seconds\":1800}"
# expect: 200 {"daily_seconds_spent":1800,"daily_minutes_spent":30,"is_target_met":true,"pet_health":100,"streak_count":0}

# Blocker 1 — the reviewer's 404 must leave both stores untouched:
curl -s -o /dev/null -w '%{http_code}\n' -X POST localhost:$PORT/api/v1/quests/progress -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"exercise_id\":\"$OTHER_EX\",\"duration_seconds\":999}"
# expect: 404 (body {"error":"exercise_not_found"})
curl -s localhost:$PORT/api/v1/quests/daily -H "Authorization: Bearer $TOKEN" | grep -o '"accumulated_seconds":[0-9]*'
# expect: "accumulated_seconds":1800  (the review saw 2799 here)
redis-cli -p 6381 GET "daily:accumulated:$UID:$DAY"
# expect: "1800"
psql "$TEST_DATABASE_URL" -Atc "SELECT minutes_spent, is_target_met FROM daily_progress WHERE user_id='$UID' AND date='$DAY'"
# expect: 30|t  (the review saw 46|t here)

# Blocker 2 — the reviewer's brick must be a 400, and the day must stay writable:
for d in 3601 1000000000 100000000000000; do
  curl -s -w ' %{http_code}\n' -X POST localhost:$PORT/api/v1/quests/progress -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
    -d "{\"exercise_id\":\"$EX1\",\"duration_seconds\":$d}"
done
# expect: three lines, each {"error":"invalid_request"} 400  (the review saw 200 for 1e9 and 500 for 1e14)
redis-cli -p 6381 GET "daily:accumulated:$UID:$DAY"
# expect: "1800" — still
curl -s -X POST localhost:$PORT/api/v1/quests/progress -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"exercise_id\":\"$EX1\",\"duration_seconds\":60}"
# expect: 200 {"daily_seconds_spent":1860,"daily_minutes_spent":31,...}  (the review saw 500 here — the day was bricked)
```

Paste the actual outputs into this plan's *Execution summary*. Then `docker compose down`, remove the scratch `.env` and
`cmd/devtoken`, confirm `git status --short` is clean, push the branch, and check `gh run list --branch
harness/2026-09-22-high-quests-daily-quest-suite-and-progress-recording` shows `backend-unit`, `backend-integration` and
`harness-tooling` green — `backend-integration` is where the extended `TestIntegration…` actually runs.

## Notes and open questions

- **Reject, not clamp, at the daily ceiling.** Clamping would make `daily_seconds_spent` disagree with what the client sent, and
  clamping *after* the INCRBY would need a compensating `SET`/`DECRBY` on the rejection path — the very class of write blocker 1
  removes. Nobody legitimately reaches 24h of study in a local day, so the ceiling is an abuse guard, not a state the UI needs to
  render; the frontend sees the same `400 invalid_request` it already handles for a malformed body. If the product later wants a
  distinguishable code (`daily_limit_reached`), it is a one-line handler change.
- **The ceiling check is a read-then-write and races.** Two concurrent calls can both pass and overshoot by up to
  `MaxDurationSeconds` each. That is fine: the bound is what matters, and overflowing `minutes_spent INT` would need roughly
  `(1.29e11 − 86400) / 3600 ≈ 3.6e7` max-size requests in flight simultaneously. No compensation path (`DECRBY`) is added, and the
  post-INCRBY failure the second blocker described becomes unreachable rather than handled.
- **Progress is recordable only against today's tasks.** `CheckExercise` scopes to today's `day_number` (owner's call). A learner
  who finishes day 3's last task at 00:05 local on day 4 gets `404 exercise_not_found`. This is consistent with the counter, which
  already attributes those seconds to day 4's date. It partially overlaps the low inbox bug
  `progress-request-validation-is-incomplete-outside-the-gin-bi.md` ("`MarkComplete` not scoped to today"); that idea is left
  `proposed` and not folded in (MVP first) — the evaluator should re-check it against this branch when it comes up.
- **`MarkComplete` keeps its `ErrExerciseNotFound` branch** for the race where the row disappears between `CheckExercise` and the
  update; the handler still maps it to 404. It is no longer the ownership check.
- **A non-UUID `exercise_id` still yields a 500** from Postgres (`invalid input syntax for type uuid`) in `CheckExercise`, as it did
  in `MarkComplete` before. Pre-existing; same low inbox bug as above. That is why the integration test and the runtime proof use
  another user's *real* exercise id, exactly as the reviewer did.
- **The once-only hook is still lost if a write after the INCRBY fails on the crossing call** —
  `ontargetmet-is-lost-forever-if-a-write-after-the-incrby-fail.md`, not folded in. Task 4's upsert-failure test deliberately
  asserts only that the counter stays usable, not that the hook is recovered.
- **One extra Redis `GET` per progress call** (`Counter.Total` for the ceiling). Negligible; it is a read and is not in the call log.
- **The blocked plan's `## Notes`** ("Server-side plausibility limits … would be a separate, product-level decision") is now
  decided by this plan: 3600 per call, 86400 per day. The executor should not edit the original plan; the amend relation is in
  this plan's frontmatter.

## Execution summary

Executed in the amended plan's existing worktree/branch as instructed — no new branch or worktree. All five tasks landed exactly
as written, no deviations, five commits on top of `ef4b9fa`:

```
4d4b00c quests: integration-test the review's reproductions; CODEMAP
89dd878 quests: test the Redis-down and upsert-failure paths
551770a quests: bound duration_seconds per call (3600) and per day (86400)
4f39f85 quests: check exercise ownership and day before any progress write
3c760b3 quests: DayNumber counts calendar days, not elapsed hours
```

One process note, not a plan deviation: the harness's single global lock (`harness/.lock`) was held by a concurrent
executor (the airouter plan) when this task started; execution was paused and resumed once it was released, per the
skill's "another executor is running; report and stop" rule.

**Plan Verification output** (run from the worktree; all matched the plan's expected values):

```
go build ./... && go vet ./...                     # no output
env -u TEST_DATABASE_URL -u TEST_REDIS_URL -u DATABASE_URL -u REDIS_URL go test ./... -count=1
  # ok: auth, config, health, quests, store — no service needed
go test ./internal/quests/... -run 'DSTTransitions' -v | grep -c -- '--- PASS'                          # 17
go test ./internal/quests/... -run 'RejectsAnExerciseOutsideTheActiveRoadmap|RejectsAnExerciseFromAnotherDay|Returns404ForAnUnknownExercise' -v | grep -c -- '--- PASS'   # 3
go test ./internal/quests/... -run 'OutOfRangeDuration|MaximumPerCallDuration|DailyCeiling|RejectsABadBody' -v | grep -c -- '--- PASS'   # 4
go test ./internal/quests/... -run 'IncrementsRedisBeforeWritingPostgres|CrossingExactly1800|FurtherProgress' -v | grep -c -- '--- PASS'  # 3
go test ./internal/quests/... -run 'RedisFailureWritesNothing|UpsertFailureAfterTheIncrby' -v | grep -c -- '--- PASS'   # 2
grep -n 'MaxDurationSeconds = 3600\|MaxDailySeconds = 86400' internal/quests/day.go     # 2 hits
grep -c 'CheckExercise' internal/quests/repo.go internal/quests/service.go             # repo.go:3 service.go:3
grep -n 'ErrInvalidDuration' internal/quests/handler.go                                # 1 hit
grep -c 'len(h.log.calls) != 0' internal/quests/service_test.go internal/quests/handler_test.go   # service_test.go:5 handler_test.go:2
grep -n 'markErr' internal/quests/*_test.go                                            # no hits
grep -n 'Hours()/24' internal/quests/day.go                                            # no hits
grep -n '"duration_seconds"\|"user_answers"' internal/quests/handler.go                # 2 hits
grep -n '"daily_seconds_spent"\|"daily_minutes_spent"\|"pet_health"\|"streak_count"' internal/quests/service.go   # 4 hits
grep -rn --include='*.go' --exclude='*_test.go' '"total_seconds"\|"target_met"\|"newly_met"\|"exercises"\|"seconds"' internal/quests/   # no hits
python3 tools/harness/cli.py validate; echo exit=$?     # exit=0
git log --oneline ef4b9fa..HEAD    # 5 commits, each carrying the trailer
git status --short                 # clean
```

**Integration (service level, real Postgres/Redis on scratch ports POSTGRES_PORT=5434 REDIS_PORT=6382 — 5433/6381 and
8099/18099 were in use by another concurrent agent; 6379/6380 are the owner's unrelated containers and were never touched):**

```
POSTGRES_PORT=5434 REDIS_PORT=6382 docker compose up -d --wait
TEST_DATABASE_URL='postgres://english:english@localhost:5434/english?sslmode=disable'
TEST_REDIS_URL='redis://localhost:6382/0'
make test-integration
```
Result: `--- PASS` for every `TestIntegration*` (auth, quests — including `TestIntegrationDailyAndProgressAgainstRealServices`
with both review reproductions, store), no `--- SKIP`, no `FAIL`.

**Runtime proof** (real binary on port 18201, real HS256 bearer + Redis session via a throwaway `cmd/devtoken` seeding two
users' demo roadmaps, deleted before committing — mirrors the review's reproductions live):

- `GET /healthz` → `200`.
- `POST /quests/progress` (user A's own day-1 exercise, 1800s) → `200 {"daily_seconds_spent":1800,"daily_minutes_spent":30,"is_target_met":true,"pet_health":100,"streak_count":0}`.
- **Blocker 1** — `POST /quests/progress` against user B's real exercise → `404`; `GET /quests/daily` still shows
  `"accumulated_seconds":1800`; `redis-cli GET daily:accumulated:<uid>:<day>` → `"1800"`; `psql … daily_progress` →
  `30|t` — all unchanged (the review had seen `2799` / `46|t` here).
- **Blocker 2** — `duration_seconds` 3601, 1e9, 1e14 → three `400 {"error":"invalid_request"}`; Redis still `"1800"`
  afterward; a following ordinary 60s report → `200 {"daily_seconds_spent":1860,"daily_minutes_spent":31,"is_target_met":true,...}`
  (the review had seen `500` here — the day was bricked).

Server and scratch stack shut down afterward (`docker compose down`); `backend/.env` and `backend/cmd/devtoken/` deleted;
also verified `make up`/`make down` (the documented commands, not just raw `docker compose`) work correctly against the
scratch ports. `git status --short` in the worktree was clean before every commit and after cleanup.

**PR:** skipped per the amending-plan rule (branch already carries the original slice's PR relationship;
`pr=skipped-not-a-collaborator` on the original plan).

**CI:** pushed `harness/2026-09-22-high-quests-daily-quest-suite-and-progress-recording`
(`ee99b1a..4d4b00c`). Green: <https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35755768316>
(`backend-unit` 19s ✓, `backend-integration` 33s ✓, `harness-tooling` 6s ✓, conclusion `success`).

**Blockers check** (after `status=done`):

```
$ python3 tools/harness/cli.py blockers --plan harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md
exit=0
```
