---
idea: harness/ideas/2026-09-22-run-02/quests-daily-quest-suite-and-progress-recording.md
status: draft
priority: high
merged: false
order: 3
---
# Quests: daily quest suite and progress recording — Plan

**Idea:** `harness/ideas/2026-09-22-run-02/quests-daily-quest-suite-and-progress-recording.md`
**Goal:** Add `backend/internal/quests` — `GET /api/v1/quests/daily` and `POST /api/v1/quests/progress` behind `auth.Require()` — implementing spec §5.2's Redis-counter-first ordering, the 1800-second target, and a `TargetMetListener` hook the pet slice will register against.

**Architecture:** A `Service` over four interfaces: `Counter` (Redis `INCRBY` + `EXPIRE` on `store.DailyAccumulatedKey`), `QuestRepo` (active roadmap, the day's exercises, marking one complete), `ProgressRepo` (the `daily_progress` upsert) and `TargetMetListener` (in-process hook, no-op by default). `day_number` and the user's local date are pure functions of a clock, a timezone and `roadmaps.created_at`, in their own file, so the fiddly arithmetic is tested without any I/O. The ordering contract from §5.2 is enforced by a shared call log in the fakes, not by reading the code.

**Tech stack:** Go 1.22, Gin, `pgx/v5`, `go-redis/v9` — all already in `go.mod` from slice 1. No new dependencies.

**Depends on:** slice 1 (`store`) and slice 2 (`auth`). Do not start until both are on `main`.

**Run every command from `backend/`** unless the step says otherwise. `rg` is not installed — use `grep -n`.

## File structure

| Path | Responsibility |
| --- | --- |
| `backend/internal/quests/day.go` `day_test.go` | local date, `day_number` (1..28 clamped) — pure |
| `backend/internal/quests/counter.go` `counter_test.go` | `Counter` interface + `RedisCounter` |
| `backend/internal/quests/repo.go` | `QuestRepo`/`ProgressRepo` interfaces + models + Postgres impls |
| `backend/internal/quests/listener.go` | `TargetMetListener`, `NopListener` |
| `backend/internal/quests/service.go` `service_test.go` | `Daily()` and `RecordProgress()` |
| `backend/internal/quests/handler.go` `handler_test.go` | the two routes |
| `backend/internal/quests/fakes_test.go` | shared in-memory fakes with the call log |
| `backend/internal/store/seed.go` `seed_test.go` | `SeedDemoRoadmap` — temporary, dies with onboarding |
| `backend/internal/quests/integration_test.go` | seed + upsert against a real DB, skipped without `DATABASE_URL` |
| `backend/cmd/api/main.go` | mount both routes behind `auth.Require()` |
| `harness/CODEMAP.md` | `quests` paragraph |

---

## Tasks

### Task 1: Local date and `day_number` arithmetic

**Files:**
- Create: `backend/internal/quests/day.go`
- Test: `backend/internal/quests/day_test.go`

Pure functions. §6.1 fixes the roadmap at 4 modules × 7 days = 28 days.

- [ ] **Step 1: Write the failing test**

`backend/internal/quests/day_test.go`:
```go
package quests

import (
	"testing"
	"time"
)

func TestLocationFallsBackToUTC(t *testing.T) {
	for _, name := range []string{"", "Not/AZone", "Mars/Olympus"} {
		if got := Location(name); got != time.UTC {
			t.Errorf("Location(%q) = %v, want UTC", name, got)
		}
	}
	if got := Location("Asia/Ho_Chi_Minh"); got.String() != "Asia/Ho_Chi_Minh" {
		t.Errorf("Location(Asia/Ho_Chi_Minh) = %v", got)
	}
}

func TestLocalDateUsesTheUsersTimezone(t *testing.T) {
	// 2026-09-22T18:30Z is already 2026-09-23 in Ho Chi Minh City (UTC+7).
	now := time.Date(2026, time.September, 22, 18, 30, 0, 0, time.UTC)

	if got := LocalDate(now, Location("UTC")); got != "2026-09-22" {
		t.Errorf("LocalDate(UTC) = %q, want 2026-09-22", got)
	}
	if got := LocalDate(now, Location("Asia/Ho_Chi_Minh")); got != "2026-09-23" {
		t.Errorf("LocalDate(Asia/Ho_Chi_Minh) = %q, want 2026-09-23", got)
	}
}

func TestDayNumberCountsCalendarDaysFromCreation(t *testing.T) {
	loc := Location("UTC")
	created := time.Date(2026, time.September, 1, 22, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		now  time.Time
		want int
	}{
		{"same day", time.Date(2026, time.September, 1, 23, 59, 0, 0, time.UTC), 1},
		{"next calendar day, 2h later", time.Date(2026, time.September, 2, 0, 30, 0, 0, time.UTC), 2},
		{"a week in", time.Date(2026, time.September, 8, 9, 0, 0, 0, time.UTC), 8},
		{"last day", time.Date(2026, time.September, 28, 9, 0, 0, 0, time.UTC), 28},
		{"clamped past the end", time.Date(2026, time.November, 1, 9, 0, 0, 0, time.UTC), 28},
		{"clamped before creation (clock skew)", time.Date(2026, time.August, 30, 9, 0, 0, 0, time.UTC), 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DayNumber(created, tt.now, loc); got != tt.want {
				t.Errorf("DayNumber = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDayNumberCrossesTheBoundaryInTheUsersTimezone(t *testing.T) {
	loc := Location("Asia/Ho_Chi_Minh")
	// Created 2026-09-01T20:00 local (= 13:00Z).
	created := time.Date(2026, time.September, 1, 13, 0, 0, 0, time.UTC)

	// 2026-09-01T23:30 local — still day 1.
	if got := DayNumber(created, time.Date(2026, time.September, 1, 16, 30, 0, 0, time.UTC), loc); got != 1 {
		t.Errorf("before local midnight: DayNumber = %d, want 1", got)
	}
	// 2026-09-02T00:30 local — day 2, even though it is still 2026-09-01 in UTC.
	if got := DayNumber(created, time.Date(2026, time.September, 1, 17, 30, 0, 0, time.UTC), loc); got != 2 {
		t.Errorf("after local midnight: DayNumber = %d, want 2", got)
	}
}

func TestRoadmapLength(t *testing.T) {
	if RoadmapDays != 28 {
		t.Errorf("RoadmapDays = %d, want 28 (spec §6.1: 4 modules x 7 days)", RoadmapDays)
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
mkdir -p internal/quests && go test ./internal/quests/...
```
Expected: build failure, `undefined: Location`.

- [ ] **Step 3: Implement**

`backend/internal/quests/day.go`:
```go
// Package quests serves the daily 30-minute exercise suite and records progress
// against it (spec §5.2, §6.1, §7).
package quests

import "time"

// RoadmapDays is the fixed roadmap length: 4 modules of 7 daily quests (§6.1).
const RoadmapDays = 28

// TargetSeconds is the daily goal from §1: at least 30 minutes.
const TargetSeconds = 1800

// Location resolves users.timezone (§3.2, default 'UTC'). An unknown name falls
// back to UTC rather than erroring — a bad timezone string must never lock a
// learner out of their quests.
func Location(name string) *time.Location {
	if name == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

// LocalDate is the YYYY-MM-DD the user is currently living in. It is the date
// component of the daily:accumulated key and of daily_progress.date.
func LocalDate(now time.Time, loc *time.Location) string {
	return now.In(loc).Format("2006-01-02")
}

// DayNumber is the 1-based day of the roadmap, counted in calendar days in the
// user's own timezone and clamped to 1..RoadmapDays. Clamping at the top means a
// learner past day 28 keeps seeing day 28 — see the plan's open questions.
func DayNumber(createdAt, now time.Time, loc *time.Location) int {
	start := startOfDay(createdAt, loc)
	today := startOfDay(now, loc)

	days := int(today.Sub(start).Hours()/24) + 1
	if days < 1 {
		return 1
	}
	if days > RoadmapDays {
		return RoadmapDays
	}
	return days
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/quests/... -v
```
Expected: every `--- PASS`, including both boundary subtests.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "quests: local date and clamped day_number arithmetic"
```

---

### Task 2: The Redis counter

**Files:**
- Create: `backend/internal/quests/counter.go`
- Test: `backend/internal/quests/counter_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/quests/counter_test.go`:
```go
package quests

import (
	"strings"
	"testing"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

func TestCounterUsesTheStoreKeyAndTTL(t *testing.T) {
	// The §4 key topology lives in store; quests must never build key strings.
	key := store.DailyAccumulatedKey("u1", mustDate(t, "2026-09-22"))
	if key != "daily:accumulated:u1:2026-09-22" {
		t.Fatalf("store.DailyAccumulatedKey = %q", key)
	}
	if store.DailyAccumulatedTTL.Hours() != 48 {
		t.Errorf("DailyAccumulatedTTL = %v, want 48h", store.DailyAccumulatedTTL)
	}
}

func TestRedisCounterKeyIsBuiltFromTheLocalDate(t *testing.T) {
	// counterKey takes the already-localised YYYY-MM-DD string so the counter
	// cannot silently use server time.
	got := counterKey("u1", "2026-09-22")
	if !strings.HasSuffix(got, ":2026-09-22") || !strings.HasPrefix(got, "daily:accumulated:u1:") {
		t.Errorf("counterKey = %q", got)
	}
}
```

Add a small helper at the bottom of the file:
```go
func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parsing %s: %v", s, err)
	}
	return d
}
```
and import `time`.

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/quests/... -run Counter
```
Expected: build failure, `undefined: counterKey`.

- [ ] **Step 3: Implement**

`backend/internal/quests/counter.go`:
```go
package quests

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// Counter is the Redis side of §5.2 step 2. Add returns the running total for
// the day *after* adding, which is what decides whether the 30-minute target has
// just been crossed.
type Counter interface {
	Add(ctx context.Context, userID, localDate string, seconds int64) (total int64, err error)
	Total(ctx context.Context, userID, localDate string) (int64, error)
}

// counterKey builds the §4 daily:accumulated key from an already-localised date,
// so the counter can never fall back to server time by accident.
func counterKey(userID, localDate string) string {
	d, err := time.Parse("2006-01-02", localDate)
	if err != nil {
		// Callers always pass LocalDate output; a bad value would silently
		// merge days, so fail loudly rather than guess.
		panic(fmt.Sprintf("quests: counterKey got a malformed date %q", localDate))
	}
	return store.DailyAccumulatedKey(userID, d)
}

// RedisCounter is the real Counter.
type RedisCounter struct{ Client *redis.Client }

// NewRedisCounter builds a counter over an existing client.
func NewRedisCounter(r *store.Redis) *RedisCounter { return &RedisCounter{Client: r.Client} }

// Add does INCRBY then EXPIRE, in that order, in one pipeline. EXPIRE is
// re-applied on every write so the 48h window slides with activity.
func (c *RedisCounter) Add(ctx context.Context, userID, localDate string, seconds int64) (int64, error) {
	key := counterKey(userID, localDate)

	pipe := c.Client.TxPipeline()
	incr := pipe.IncrBy(ctx, key, seconds)
	pipe.Expire(ctx, key, store.DailyAccumulatedTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("quests: incrementing daily counter: %w", err)
	}
	return incr.Val(), nil
}

// Total reads the counter without changing it. A missing key means zero.
func (c *RedisCounter) Total(ctx context.Context, userID, localDate string) (int64, error) {
	v, err := c.Client.Get(ctx, counterKey(userID, localDate)).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("quests: reading daily counter: %w", err)
	}
	return v, nil
}

var _ Counter = (*RedisCounter)(nil)
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/quests/... -run Counter -v
```
Expected: two `--- PASS` lines.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "quests: Redis daily counter with INCRBY then EXPIRE"
```

---

### Task 3: Repository interfaces, models and the listener hook

**Files:**
- Create: `backend/internal/quests/repo.go`
- Create: `backend/internal/quests/listener.go`

No tests of their own — these are interfaces and SQL constants, exercised by Tasks 4–6 and by the
integration test in Task 7. Commit them together so the next task compiles.

- [ ] **Step 1: Write `listener.go`**

```go
package quests

import "context"

// TargetMetListener is the in-process hook fired the first time a user crosses
// the 30-minute target on a given day (§5.2 step 4). The pet slice registers the
// implementation that bumps health and streak.
//
// It is called AFTER the daily_progress upsert commits, and its error is logged
// rather than returned: a failing pet update must never roll back a recorded
// study session.
type TargetMetListener interface {
	OnTargetMet(ctx context.Context, userID, localDate string) error
}

// NopListener is the default: quests works standalone until the pet slice lands.
type NopListener struct{}

func (NopListener) OnTargetMet(context.Context, string, string) error { return nil }

var _ TargetMetListener = NopListener{}
```

- [ ] **Step 2: Write `repo.go`**

```go
package quests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoActiveRoadmap means the user has no roadmaps row with is_active = TRUE.
// Until the onboarding slice exists, this is the normal state for a new user.
var ErrNoActiveRoadmap = errors.New("quests: no active roadmap")

// ErrExerciseNotFound means the exercise id is unknown or belongs to another
// user's roadmap. The two are deliberately indistinguishable to the client.
var ErrExerciseNotFound = errors.New("quests: exercise not found")

// Roadmap is the slice of spec §3.2 `roadmaps` this package reads.
type Roadmap struct {
	ID        string
	CreatedAt time.Time
}

// Exercise is one of a day's three tasks (§3.2 `exercises`, §6.1).
type Exercise struct {
	ID          string
	DayNumber   int
	TaskType    string          // vocabulary | reading | practice
	ContentJSON json.RawMessage
	IsCompleted bool
}

// Profile is the slice of `users` quests needs: the timezone decides which day
// a progress call belongs to.
type Profile struct {
	Timezone string
}

// QuestRepo is the Postgres read side plus the one exercise write.
type QuestRepo interface {
	Profile(ctx context.Context, userID string) (Profile, error)
	ActiveRoadmap(ctx context.Context, userID string) (Roadmap, error)
	ExercisesForDay(ctx context.Context, roadmapID string, day int) ([]Exercise, error)
	// MarkComplete sets is_completed and returns ErrExerciseNotFound when the
	// exercise is not on roadmapID.
	MarkComplete(ctx context.Context, roadmapID, exerciseID string) error
}

// ProgressRepo owns the daily_progress upsert.
type ProgressRepo interface {
	// Upsert writes minutes_spent and is_target_met for (userID, localDate).
	Upsert(ctx context.Context, userID, localDate string, minutes int, targetMet bool) error
}

const (
	profileSQL = `SELECT COALESCE(timezone, 'UTC') FROM users WHERE id = $1`

	activeRoadmapSQL = `
SELECT id, created_at
FROM roadmaps
WHERE user_id = $1 AND is_active = TRUE
ORDER BY created_at DESC
LIMIT 1`

	exercisesForDaySQL = `
SELECT id, day_number, task_type, content_json, is_completed
FROM exercises
WHERE roadmap_id = $1 AND day_number = $2
ORDER BY task_type`

	markCompleteSQL = `
UPDATE exercises
SET is_completed = TRUE
WHERE id = $1 AND roadmap_id = $2`

	// daily_progress.date defaults to CURRENT_DATE, which is the *server's*
	// date — always pass the user's local date explicitly.
	upsertProgressSQL = `
INSERT INTO daily_progress (user_id, date, minutes_spent, is_target_met)
VALUES ($1, $2::date, $3, $4)
ON CONFLICT (user_id, date) DO UPDATE SET
    minutes_spent = EXCLUDED.minutes_spent,
    is_target_met = EXCLUDED.is_target_met`
)

// PgRepo implements both QuestRepo and ProgressRepo over one pool.
type PgRepo struct{ Pool *pgxpool.Pool }

// NewPgRepo builds a repo over an existing pool.
func NewPgRepo(pool *pgxpool.Pool) *PgRepo { return &PgRepo{Pool: pool} }

func (r *PgRepo) Profile(ctx context.Context, userID string) (Profile, error) {
	var p Profile
	if err := r.Pool.QueryRow(ctx, profileSQL, userID).Scan(&p.Timezone); err != nil {
		return Profile{}, fmt.Errorf("quests: reading profile: %w", err)
	}
	return p, nil
}

func (r *PgRepo) ActiveRoadmap(ctx context.Context, userID string) (Roadmap, error) {
	var rm Roadmap
	err := r.Pool.QueryRow(ctx, activeRoadmapSQL, userID).Scan(&rm.ID, &rm.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Roadmap{}, ErrNoActiveRoadmap
	}
	if err != nil {
		return Roadmap{}, fmt.Errorf("quests: reading active roadmap: %w", err)
	}
	return rm, nil
}

func (r *PgRepo) ExercisesForDay(ctx context.Context, roadmapID string, day int) ([]Exercise, error) {
	rows, err := r.Pool.Query(ctx, exercisesForDaySQL, roadmapID, day)
	if err != nil {
		return nil, fmt.Errorf("quests: reading exercises: %w", err)
	}
	defer rows.Close()

	var out []Exercise
	for rows.Next() {
		var e Exercise
		if err := rows.Scan(&e.ID, &e.DayNumber, &e.TaskType, &e.ContentJSON, &e.IsCompleted); err != nil {
			return nil, fmt.Errorf("quests: scanning exercise: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PgRepo) MarkComplete(ctx context.Context, roadmapID, exerciseID string) error {
	tag, err := r.Pool.Exec(ctx, markCompleteSQL, exerciseID, roadmapID)
	if err != nil {
		return fmt.Errorf("quests: marking exercise complete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrExerciseNotFound
	}
	return nil
}

func (r *PgRepo) Upsert(ctx context.Context, userID, localDate string, minutes int, targetMet bool) error {
	if _, err := r.Pool.Exec(ctx, upsertProgressSQL, userID, localDate, minutes, targetMet); err != nil {
		return fmt.Errorf("quests: upserting daily_progress: %w", err)
	}
	return nil
}

var (
	_ QuestRepo    = (*PgRepo)(nil)
	_ ProgressRepo = (*PgRepo)(nil)
)
```

- [ ] **Step 3: Confirm it compiles**

```sh
go build ./... && go vet ./internal/quests/...
```
Expected: no output.

- [ ] **Step 4: Commit**

```sh
cd .. && git add backend && git commit -m "quests: repository interfaces, models and the TargetMet hook"
```

---

### Task 4: Shared test fakes with an ordering call log

**Files:**
- Create: `backend/internal/quests/fakes_test.go`

The call log is what turns "Redis first, then Postgres" from a comment into a test.

- [ ] **Step 1: Write the fakes**

`backend/internal/quests/fakes_test.go`:
```go
package quests

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// callLog records the order of side effects across all fakes in one test, so
// the §5.2 ordering contract (counter first, then Postgres) is asserted rather
// than assumed.
type callLog struct{ calls []string }

func (l *callLog) add(format string, args ...any) {
	l.calls = append(l.calls, fmt.Sprintf(format, args...))
}

type fakeCounter struct {
	log    *callLog
	totals map[string]int64 // keyed by userID|localDate
	err    error
}

func newFakeCounter(l *callLog) *fakeCounter {
	return &fakeCounter{log: l, totals: map[string]int64{}}
}

func (f *fakeCounter) key(userID, date string) string { return userID + "|" + date }

func (f *fakeCounter) Add(_ context.Context, userID, localDate string, seconds int64) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.log.add("INCRBY %s %d", f.key(userID, localDate), seconds)
	f.totals[f.key(userID, localDate)] += seconds
	f.log.add("EXPIRE %s", f.key(userID, localDate))
	return f.totals[f.key(userID, localDate)], nil
}

func (f *fakeCounter) Total(_ context.Context, userID, localDate string) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.totals[f.key(userID, localDate)], nil
}

type fakeQuestRepo struct {
	log       *callLog
	timezone  string
	roadmap   *Roadmap
	exercises map[int][]Exercise // by day_number
	completed map[string]bool
	markErr   error
}

func newFakeQuestRepo(l *callLog) *fakeQuestRepo {
	return &fakeQuestRepo{
		log:       l,
		timezone:  "UTC",
		exercises: map[int][]Exercise{},
		completed: map[string]bool{},
	}
}

func (f *fakeQuestRepo) Profile(context.Context, string) (Profile, error) {
	return Profile{Timezone: f.timezone}, nil
}

func (f *fakeQuestRepo) ActiveRoadmap(context.Context, string) (Roadmap, error) {
	if f.roadmap == nil {
		return Roadmap{}, ErrNoActiveRoadmap
	}
	return *f.roadmap, nil
}

func (f *fakeQuestRepo) ExercisesForDay(_ context.Context, _ string, day int) ([]Exercise, error) {
	return f.exercises[day], nil
}

func (f *fakeQuestRepo) MarkComplete(_ context.Context, _, exerciseID string) error {
	if f.markErr != nil {
		return f.markErr
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
	err error
}

func newFakeProgressRepo(l *callLog) *fakeProgressRepo {
	return &fakeProgressRepo{log: l, rows: map[string]struct {
		minutes   int
		targetMet bool
	}{}}
}

func (f *fakeProgressRepo) Upsert(_ context.Context, userID, localDate string, minutes int, targetMet bool) error {
	if f.err != nil {
		return f.err
	}
	f.log.add("UPSERT daily_progress %s|%s minutes=%d target=%t", userID, localDate, minutes, targetMet)
	f.rows[userID+"|"+localDate] = struct {
		minutes   int
		targetMet bool
	}{minutes, targetMet}
	return nil
}

type recordingListener struct {
	log   *callLog
	fired int
	err   error
}

func (l *recordingListener) OnTargetMet(_ context.Context, userID, localDate string) error {
	l.fired++
	l.log.add("ON TARGET MET %s|%s", userID, localDate)
	return l.err
}

// demoExercises builds a day's three tasks in the §6.1 categories.
func demoExercises(day int) []Exercise {
	return []Exercise{
		{ID: fmt.Sprintf("ex-%d-practice", day), DayNumber: day, TaskType: "practice", ContentJSON: json.RawMessage(`{"n":1}`)},
		{ID: fmt.Sprintf("ex-%d-reading", day), DayNumber: day, TaskType: "reading", ContentJSON: json.RawMessage(`{"n":2}`)},
		{ID: fmt.Sprintf("ex-%d-vocabulary", day), DayNumber: day, TaskType: "vocabulary", ContentJSON: json.RawMessage(`{"n":3}`)},
	}
}

// fixedClock returns a clock pinned to t.
func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }
```

- [ ] **Step 2: Confirm the package still builds**

```sh
go vet ./internal/quests/...
```
Expected: no output (the fakes are unused until Task 5 — `go vet` does not complain about unused
package-level funcs).

- [ ] **Step 3: Commit**

```sh
cd .. && git add backend && git commit -m "quests: shared test fakes with a cross-fake call log"
```

---

### Task 5: `RecordProgress` — the §5.2 write path

**Files:**
- Create: `backend/internal/quests/service.go`
- Test: `backend/internal/quests/service_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/quests/service_test.go`:
```go
package quests

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type harness struct {
	svc      *Service
	log      *callLog
	counter  *fakeCounter
	quests   *fakeQuestRepo
	progress *fakeProgressRepo
	listener *recordingListener
}

func newHarness(t *testing.T, now time.Time) *harness {
	t.Helper()
	log := &callLog{}
	h := &harness{
		log:      log,
		counter:  newFakeCounter(log),
		quests:   newFakeQuestRepo(log),
		progress: newFakeProgressRepo(log),
		listener: &recordingListener{log: log},
	}
	h.quests.roadmap = &Roadmap{ID: "rm-1", CreatedAt: now.Add(-24 * time.Hour)}
	h.quests.exercises[1] = demoExercises(1)
	h.quests.exercises[2] = demoExercises(2)
	h.svc = NewService(h.counter, h.quests, h.progress, h.listener, fixedClock(now))
	return h
}

func TestRecordProgressIncrementsRedisBeforeWritingPostgres(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 600)
	if err != nil {
		t.Fatalf("RecordProgress() = %v", err)
	}
	if out.TotalSeconds != 600 || out.TargetMet || out.NewlyMet {
		t.Errorf("out = %+v, want total=600 targetMet=false newlyMet=false", out)
	}

	want := []string{
		"INCRBY u1|2026-09-22 600",
		"EXPIRE u1|2026-09-22",
		"UPSERT daily_progress u1|2026-09-22 minutes=10 target=false",
		"MARK COMPLETE ex-2-reading",
	}
	if !reflect.DeepEqual(h.log.calls, want) {
		t.Errorf("call order =\n  %v\nwant\n  %v", h.log.calls, want)
	}
}

func TestCrossingExactly1800SecondsMeetsTheTargetAndFiresOnce(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	ctx := context.Background()

	for i, seconds := range []int64{600, 600} {
		out, err := h.svc.RecordProgress(ctx, "u1", "ex-2-reading", seconds)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if out.TargetMet {
			t.Fatalf("call %d: TargetMet true at %ds, want false", i, out.TotalSeconds)
		}
	}

	out, err := h.svc.RecordProgress(ctx, "u1", "ex-2-practice", 600)
	if err != nil {
		t.Fatalf("third call: %v", err)
	}
	if out.TotalSeconds != 1800 {
		t.Errorf("TotalSeconds = %d, want 1800", out.TotalSeconds)
	}
	if !out.TargetMet || !out.NewlyMet {
		t.Errorf("out = %+v, want targetMet and newlyMet both true at exactly 1800s", out)
	}
	if h.listener.fired != 1 {
		t.Errorf("listener fired %d times, want 1", h.listener.fired)
	}
	if row := h.progress.rows["u1|2026-09-22"]; row.minutes != 30 || !row.targetMet {
		t.Errorf("daily_progress row = %+v, want minutes=30 target=true", row)
	}
}

func TestFurtherProgressTheSameDayDoesNotRefire(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	ctx := context.Background()

	if _, err := h.svc.RecordProgress(ctx, "u1", "ex-2-reading", 1800); err != nil {
		t.Fatalf("first call: %v", err)
	}
	out, err := h.svc.RecordProgress(ctx, "u1", "ex-2-practice", 600)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if !out.TargetMet {
		t.Error("TargetMet = false after the target was already met")
	}
	if out.NewlyMet {
		t.Error("NewlyMet = true on a second call the same day")
	}
	if h.listener.fired != 1 {
		t.Errorf("listener fired %d times, want 1", h.listener.fired)
	}
	if row := h.progress.rows["u1|2026-09-22"]; row.minutes != 40 {
		t.Errorf("minutes_spent = %d, want 40 (2400s / 60)", row.minutes)
	}
}

func TestMinutesUseIntegerDivision(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	if _, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 1799); err != nil {
		t.Fatalf("RecordProgress: %v", err)
	}
	row := h.progress.rows["u1|2026-09-22"]
	if row.minutes != 29 || row.targetMet {
		t.Errorf("row = %+v, want minutes=29 target=false at 1799s", row)
	}
}

func TestProgressUsesTheUsersTimezoneForTheDate(t *testing.T) {
	// 18:30Z is already the 23rd in Ho Chi Minh City.
	now := time.Date(2026, time.September, 22, 18, 30, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.quests.timezone = "Asia/Ho_Chi_Minh"

	if _, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 600); err != nil {
		t.Fatalf("RecordProgress: %v", err)
	}
	if _, ok := h.progress.rows["u1|2026-09-23"]; !ok {
		t.Errorf("rows = %v, want a row for the user's local date 2026-09-23", h.progress.rows)
	}
}

func TestAListenerFailureDoesNotFailTheRequest(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.listener.err = errors.New("pet is on fire")

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 1800)
	if err != nil {
		t.Fatalf("RecordProgress() = %v, want nil: a listener failure must not fail the write", err)
	}
	if !out.NewlyMet {
		t.Error("NewlyMet = false despite crossing the target")
	}
}

func TestRecordProgressRejectsAnExerciseOutsideTheActiveRoadmap(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.quests.markErr = ErrExerciseNotFound

	if _, err := h.svc.RecordProgress(context.Background(), "u1", "someone-elses-exercise", 600); !errors.Is(err, ErrExerciseNotFound) {
		t.Fatalf("err = %v, want ErrExerciseNotFound", err)
	}
}

func TestRecordProgressRejectsNonPositiveSeconds(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	for _, seconds := range []int64{0, -1, -600} {
		if _, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", seconds); err == nil {
			t.Errorf("seconds = %d: err = nil, want an error", seconds)
		}
	}
	if len(h.log.calls) != 0 {
		t.Errorf("a rejected call touched Redis/Postgres: %v", h.log.calls)
	}
}

func TestRecordProgressWithoutAnActiveRoadmap(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.quests.roadmap = nil

	if _, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 600); !errors.Is(err, ErrNoActiveRoadmap) {
		t.Fatalf("err = %v, want ErrNoActiveRoadmap", err)
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/quests/... -run 'Record|Crossing|Further|Minutes|Listener|Timezone'
```
Expected: build failure, `undefined: NewService`.

- [ ] **Step 3: Implement**

`backend/internal/quests/service.go`:
```go
package quests

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ProgressResult is the POST /quests/progress response body.
type ProgressResult struct {
	TotalSeconds int64 `json:"total_seconds"`
	TargetMet    bool  `json:"target_met"`
	NewlyMet     bool  `json:"newly_met"`
}

// Service implements the §5.2 daily loop.
type Service struct {
	counter  Counter
	quests   QuestRepo
	progress ProgressRepo
	listener TargetMetListener
	now      func() time.Time
}

// NewService wires the collaborators. now is injectable so day boundaries are
// testable; listener may be NopListener{} until the pet slice registers one.
func NewService(counter Counter, quests QuestRepo, progress ProgressRepo, listener TargetMetListener, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	if listener == nil {
		listener = NopListener{}
	}
	return &Service{counter: counter, quests: quests, progress: progress, listener: listener, now: now}
}

// RecordProgress implements §5.2 steps 2-4, in that order:
//
//	1. INCRBY the Redis counter (+ EXPIRE) and read the running total back.
//	2. Upsert daily_progress from that total — Redis is the single source of
//	   truth for the day, so bursts of calls cannot disagree.
//	3. Mark the exercise complete.
//	4. Fire OnTargetMet, but only on the call that crossed 1800s.
//
// The ordering is a contract, not an implementation detail: see the call-log
// test in service_test.go.
func (s *Service) RecordProgress(ctx context.Context, userID, exerciseID string, seconds int64) (ProgressResult, error) {
	if seconds <= 0 {
		return ProgressResult{}, fmt.Errorf("quests: seconds must be positive, got %d", seconds)
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
	date := LocalDate(s.now(), loc)

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
	if err := s.quests.MarkComplete(ctx, roadmap.ID, exerciseID); err != nil {
		return ProgressResult{}, err
	}

	if newlyMet {
		// Best-effort: the study session is already recorded and must not be
		// rolled back by a listener failure.
		if err := s.listener.OnTargetMet(ctx, userID, date); err != nil {
			log.Printf("quests: target-met listener failed for user %s on %s: %v", userID, date, err)
		}
	}

	return ProgressResult{TotalSeconds: total, TargetMet: targetMet, NewlyMet: newlyMet}, nil
}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/quests/... -v
```
Expected: every service test `--- PASS`, including the exact call-order assertion.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "quests: RecordProgress with Redis-first ordering and a once-only target hook"
```

---

### Task 6: `Daily()` and both HTTP handlers

**Files:**
- Modify: `backend/internal/quests/service.go` (add `Daily`)
- Create: `backend/internal/quests/handler.go`
- Test: `backend/internal/quests/service_test.go` (append)
- Test: `backend/internal/quests/handler_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `service_test.go`:
```go
func TestDailyReturnsTodaysThreeTasksAndTheRunningTotal(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	// Roadmap created 24h ago → day 2.
	if _, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 900); err != nil {
		t.Fatalf("seeding progress: %v", err)
	}

	got, err := h.svc.Daily(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Daily() = %v", err)
	}
	if got.DayNumber != 2 {
		t.Errorf("DayNumber = %d, want 2", got.DayNumber)
	}
	if len(got.Exercises) != 3 {
		t.Fatalf("len(Exercises) = %d, want 3", len(got.Exercises))
	}
	if got.TotalSeconds != 900 || got.TargetMet {
		t.Errorf("total = %d targetMet = %t, want 900/false", got.TotalSeconds, got.TargetMet)
	}
	seen := map[string]bool{}
	for _, e := range got.Exercises {
		seen[e.TaskType] = true
	}
	for _, want := range []string{"vocabulary", "reading", "practice"} {
		if !seen[want] {
			t.Errorf("missing the %s task", want)
		}
	}
}

func TestDailyWithoutAnActiveRoadmap(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	h.quests.roadmap = nil

	if _, err := h.svc.Daily(context.Background(), "u1"); !errors.Is(err, ErrNoActiveRoadmap) {
		t.Fatalf("err = %v, want ErrNoActiveRoadmap", err)
	}
}

func TestDailyClampsPastDay28(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.quests.roadmap = &Roadmap{ID: "rm-1", CreatedAt: now.Add(-90 * 24 * time.Hour)}
	h.quests.exercises[28] = demoExercises(28)

	got, err := h.svc.Daily(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Daily() = %v", err)
	}
	if got.DayNumber != 28 {
		t.Errorf("DayNumber = %d, want 28 (clamped)", got.DayNumber)
	}
	if len(got.Exercises) != 3 {
		t.Errorf("len(Exercises) = %d, want 3", len(got.Exercises))
	}
}
```

`backend/internal/quests/handler_test.go`:
```go
package quests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// withUser mounts the routes with a stub that injects the authenticated user,
// standing in for auth.Require() (which is covered in the auth slice).
func newQuestRouter(svc *Service, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	inject := func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
	g := r.Group("/api/v1", inject)
	g.GET("/quests/daily", DailyHandler(svc))
	g.POST("/quests/progress", ProgressHandler(svc))
	return r
}

func TestDailyHandlerReturns200WithTheSuite(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	w := httptest.NewRecorder()
	newQuestRouter(h.svc, "u1").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/quests/daily", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var body struct {
		DayNumber    int  `json:"day_number"`
		TotalSeconds int  `json:"total_seconds"`
		TargetMet    bool `json:"target_met"`
		Exercises    []struct {
			ID          string          `json:"id"`
			TaskType    string          `json:"task_type"`
			ContentJSON json.RawMessage `json:"content_json"`
			IsCompleted bool            `json:"is_completed"`
		} `json:"exercises"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding: %v (%s)", err, w.Body.String())
	}
	if body.DayNumber != 2 || len(body.Exercises) != 3 {
		t.Errorf("body = %+v", body)
	}
}

func TestDailyHandlerReturns404WithoutARoadmap(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	h.quests.roadmap = nil

	w := httptest.NewRecorder()
	newQuestRouter(h.svc, "u1").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/quests/daily", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"error":"no_active_roadmap"`) {
		t.Errorf("body = %s, want the no_active_roadmap error", w.Body.String())
	}
}

func TestProgressHandlerReturnsTheTotals(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/quests/progress",
		strings.NewReader(`{"exercise_id":"ex-2-reading","seconds":1800}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newQuestRouter(h.svc, "u1").ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	for _, want := range []string{`"total_seconds":1800`, `"target_met":true`, `"newly_met":true`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("body = %s, missing %s", w.Body.String(), want)
		}
	}
}

func TestProgressHandlerRejectsABadBody(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	r := newQuestRouter(h.svc, "u1")

	for _, body := range []string{`{}`, `{"exercise_id":"ex-2-reading"}`, `{"exercise_id":"ex-2-reading","seconds":0}`, `{"exercise_id":"ex-2-reading","seconds":-5}`, `nonsense`} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/quests/progress", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %q → status %d, want 400", body, w.Code)
		}
	}
}

func TestProgressHandlerReturns404ForAnUnknownExercise(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	h.quests.markErr = ErrExerciseNotFound

	req := httptest.NewRequest(http.MethodPost, "/api/v1/quests/progress",
		strings.NewReader(`{"exercise_id":"nope","seconds":600}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newQuestRouter(h.svc, "u1").ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/quests/... -run 'Daily|Progress'
```
Expected: build failure, `undefined: DailyHandler`.

- [ ] **Step 3: Implement `Daily` in `service.go`**

```go
// DailySuite is the GET /quests/daily response.
type DailySuite struct {
	DayNumber    int        `json:"day_number"`
	TotalSeconds int64      `json:"total_seconds"`
	TargetMet    bool       `json:"target_met"`
	Exercises    []Exercise `json:"exercises"`
}

// Daily resolves the active roadmap, computes today's day_number in the user's
// timezone and returns that day's three tasks plus today's running total.
func (s *Service) Daily(ctx context.Context, userID string) (DailySuite, error) {
	profile, err := s.quests.Profile(ctx, userID)
	if err != nil {
		return DailySuite{}, err
	}
	roadmap, err := s.quests.ActiveRoadmap(ctx, userID)
	if err != nil {
		return DailySuite{}, err
	}

	loc := Location(profile.Timezone)
	now := s.now()
	day := DayNumber(roadmap.CreatedAt, now, loc)

	exercises, err := s.quests.ExercisesForDay(ctx, roadmap.ID, day)
	if err != nil {
		return DailySuite{}, err
	}
	total, err := s.counter.Total(ctx, userID, LocalDate(now, loc))
	if err != nil {
		return DailySuite{}, err
	}

	return DailySuite{
		DayNumber:    day,
		TotalSeconds: total,
		TargetMet:    total >= TargetSeconds,
		Exercises:    exercises,
	}, nil
}
```

Add the JSON tags to `Exercise` in `repo.go` so it serialises as §7 expects:
```go
type Exercise struct {
	ID          string          `json:"id"`
	DayNumber   int             `json:"day_number"`
	TaskType    string          `json:"task_type"`
	ContentJSON json.RawMessage `json:"content_json"`
	IsCompleted bool            `json:"is_completed"`
}
```

- [ ] **Step 4: Implement the handlers**

`backend/internal/quests/handler.go`:
```go
package quests

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// DailyHandler serves GET /api/v1/quests/daily (spec §7). It must be mounted
// behind auth.Require().
func DailyHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		suite, err := svc.Daily(c.Request.Context(), userID)
		if errors.Is(err, ErrNoActiveRoadmap) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no_active_roadmap"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		if suite.Exercises == nil {
			suite.Exercises = []Exercise{} // serialise as [] rather than null
		}
		c.JSON(http.StatusOK, suite)
	}
}

// progressRequest is the §7 POST /api/v1/quests/progress body.
type progressRequest struct {
	ExerciseID string `json:"exercise_id" binding:"required"`
	Seconds    int64  `json:"seconds" binding:"required,gt=0"`
}

// ProgressHandler serves POST /api/v1/quests/progress. It must be mounted
// behind auth.Require().
func ProgressHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req progressRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}

		out, err := svc.RecordProgress(c.Request.Context(), userID, req.ExerciseID, req.Seconds)
		switch {
		case errors.Is(err, ErrNoActiveRoadmap):
			c.JSON(http.StatusNotFound, gin.H{"error": "no_active_roadmap"})
		case errors.Is(err, ErrExerciseNotFound):
			// 404 rather than 403 so other users' exercise ids stay unprobeable.
			c.JSON(http.StatusNotFound, gin.H{"error": "exercise_not_found"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			c.JSON(http.StatusOK, out)
		}
	}
}
```

The handler test's `inject` stub sets `"user_id"`, which is `auth.ContextUserID` — if the auth slice
renamed that constant, use `c.Set(auth.ContextUserID, userID)` in the test instead of the literal.

- [ ] **Step 5: Run and confirm it passes**

```sh
go test ./internal/quests/... -v
```
Expected: every test `--- PASS`.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend && git commit -m "quests: GET /quests/daily and POST /quests/progress handlers"
```

---

### Task 7: Demo roadmap seed and the integration test

**Files:**
- Create: `backend/internal/store/seed.go`
- Test: `backend/internal/store/seed_test.go`
- Create: `backend/internal/quests/integration_test.go`

The seed exists only because `onboarding` has no slice in this run. Its doc comment says so, so that
whoever builds onboarding deletes it rather than extending it.

- [ ] **Step 1: Write the seed helper**

`backend/internal/store/seed.go`:
```go
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DemoRoadmapDays and DemoTaskTypes mirror spec §6.1: 4 modules of 7 days,
// three 10-minute tasks each.
const DemoRoadmapDays = 28

// DemoTaskTypes are the three task_category values (§3.2).
var DemoTaskTypes = []string{"vocabulary", "reading", "practice"}

// SeedDemoRoadmap inserts one active roadmap with 28 days x 3 exercises for a
// user and returns the roadmap id.
//
// TEMPORARY. This exists only because the onboarding slice (which generates a
// real roadmap via the AI router, spec §5.1 steps 4-5) is not part of this MVP
// run. When onboarding lands, DELETE this file rather than extending it.
func SeedDemoRoadmap(ctx context.Context, pool *pgxpool.Pool, userID string) (string, error) {
	var roadmapID string
	err := pool.QueryRow(ctx,
		`INSERT INTO roadmaps (user_id, roadmap_json, is_active) VALUES ($1, $2::jsonb, TRUE) RETURNING id`,
		userID, `{"source":"demo-seed","modules":4,"days":28}`,
	).Scan(&roadmapID)
	if err != nil {
		return "", fmt.Errorf("store: seeding roadmap: %w", err)
	}

	for day := 1; day <= DemoRoadmapDays; day++ {
		for _, taskType := range DemoTaskTypes {
			content := fmt.Sprintf(`{"day":%d,"task":"%s","minutes":10}`, day, taskType)
			if _, err := pool.Exec(ctx,
				`INSERT INTO exercises (roadmap_id, day_number, task_type, content_json)
				 VALUES ($1, $2, $3::task_category, $4::jsonb)`,
				roadmapID, day, taskType, content,
			); err != nil {
				return "", fmt.Errorf("store: seeding day %d %s: %w", day, taskType, err)
			}
		}
	}
	return roadmapID, nil
}
```

`backend/internal/store/seed_test.go`:
```go
package store

import "testing"

func TestDemoRoadmapShapeMatchesSpec61(t *testing.T) {
	if DemoRoadmapDays != 28 {
		t.Errorf("DemoRoadmapDays = %d, want 28 (4 modules x 7 days)", DemoRoadmapDays)
	}
	want := []string{"vocabulary", "reading", "practice"}
	if len(DemoTaskTypes) != len(want) {
		t.Fatalf("DemoTaskTypes = %v, want %v", DemoTaskTypes, want)
	}
	for i, w := range want {
		if DemoTaskTypes[i] != w {
			t.Errorf("DemoTaskTypes[%d] = %q, want %q", i, DemoTaskTypes[i], w)
		}
	}
}
```

- [ ] **Step 2: Write the integration test**

`backend/internal/quests/integration_test.go`:
```go
package quests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// Skipped unless both services are configured — `go test ./...` stays green
// without them.
func TestIntegrationDailyAndProgressAgainstRealServices(t *testing.T) {
	dbURL, redisURL := os.Getenv("DATABASE_URL"), os.Getenv("REDIS_URL")
	if dbURL == "" || redisURL == "" {
		t.Skip("DATABASE_URL/REDIS_URL unset; run `make up` and export them")
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
	rdb, err := store.NewRedis(ctx, redisURL)
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	const gid = "google-quests-integration"
	var userID string
	_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)
	if err := pg.Pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, target_goal, timezone) VALUES ($1,$2,$3,$4) RETURNING id`,
		"quests@example.com", gid, "", "UTC").Scan(&userID); err != nil {
		t.Fatalf("inserting user: %v", err)
	}
	t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid) })

	if _, err := store.SeedDemoRoadmap(ctx, pg.Pool, userID); err != nil {
		t.Fatalf("SeedDemoRoadmap: %v", err)
	}

	now := time.Now().UTC()
	repo := NewPgRepo(pg.Pool)
	counter := NewRedisCounter(rdb)
	t.Cleanup(func() {
		rdb.Client.Del(ctx, store.DailyAccumulatedKey(userID, now))
	})

	svc := NewService(counter, repo, repo, NopListener{}, func() time.Time { return now })

	suite, err := svc.Daily(ctx, userID)
	if err != nil {
		t.Fatalf("Daily: %v", err)
	}
	if suite.DayNumber != 1 {
		t.Errorf("DayNumber = %d, want 1 on a freshly seeded roadmap", suite.DayNumber)
	}
	if len(suite.Exercises) != 3 {
		t.Fatalf("len(Exercises) = %d, want 3", len(suite.Exercises))
	}

	out, err := svc.RecordProgress(ctx, userID, suite.Exercises[0].ID, 1800)
	if err != nil {
		t.Fatalf("RecordProgress: %v", err)
	}
	if !out.TargetMet || !out.NewlyMet {
		t.Errorf("out = %+v, want target met on the first 1800s", out)
	}

	var minutes int
	var met bool
	if err := pg.Pool.QueryRow(ctx,
		`SELECT minutes_spent, is_target_met FROM daily_progress WHERE user_id = $1 AND date = $2::date`,
		userID, LocalDate(now, time.UTC)).Scan(&minutes, &met); err != nil {
		t.Fatalf("reading daily_progress: %v", err)
	}
	if minutes != 30 || !met {
		t.Errorf("daily_progress = (%d, %t), want (30, true)", minutes, met)
	}
}
```

- [ ] **Step 3: Confirm it skips without services, and the suite stays green**

```sh
env -u DATABASE_URL -u REDIS_URL go test ./... -count=1
```
Expected: `ok` everywhere, with the integration tests reported as skipped under `-v`.

- [ ] **Step 4: Optionally run it for real**

```sh
docker compose up -d
export DATABASE_URL='postgres://english:english@localhost:5432/english?sslmode=disable'
export REDIS_URL='redis://localhost:6379/0'
go test ./internal/quests/... -count=1 -run Integration -v
docker compose down
```
Expected: `--- PASS`. If Docker is unavailable, note it in the execution summary.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "quests: demo roadmap seed and end-to-end integration test"
```

---

### Task 8: Mount the routes behind `auth.Require()`

**Files:**
- Modify: `backend/cmd/api/main.go`

- [ ] **Step 1: Wire it**

Extend the `/api/v1` group added by the auth slice:
```go
	questSvc := quests.NewService(
		quests.NewRedisCounter(rdb),
		quests.NewPgRepo(pg.Pool),
		quests.NewPgRepo(pg.Pool),
		quests.NopListener{}, // the pet slice replaces this
		time.Now,
	)

	guarded := v1.Group("", auth.Require(tokens, sessions))
	guarded.GET("/quests/daily", quests.DailyHandler(questSvc))
	guarded.POST("/quests/progress", quests.ProgressHandler(questSvc))
```
where `tokens` and `sessions` are the `*auth.TokenIssuer` and `auth.SessionStore` the auth slice left in
local variables. Construct `NewPgRepo` once and pass it twice rather than building it twice, if you
prefer — it satisfies both interfaces.

- [ ] **Step 2: Confirm build, vet and tests**

```sh
go build ./... && go vet ./... && go test ./...
```
Expected: clean build/vet; `ok` for `internal/auth`, `internal/config`, `internal/health`,
`internal/quests`, `internal/store`.

- [ ] **Step 3: Commit**

```sh
cd .. && git add backend && git commit -m "quests: mount daily and progress routes behind auth.Require()"
```

---

### Task 9: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (the `**quests**` bullet, and one sentence on the `**store**` bullet)

- [ ] **Step 1: Replace the quests bullet**

```
- **quests** — the daily loop (§5.2). `GET /api/v1/quests/daily` resolves `roadmaps.is_active`, computes `day_number` = calendar days since `roadmaps.created_at` in `users.timezone`, +1, clamped to 1..28, and returns that day's three `exercises` with today's `total_seconds`/`target_met`; 404 `no_active_roadmap` when there is none. `POST /api/v1/quests/progress` ({exercise_id, seconds}) does `INCRBY daily:accumulated:{user_id}:{local-date}` + `EXPIRE` 48h **first**, then upserts `daily_progress` (`minutes_spent = total/60`, `is_target_met = total >= 1800`) from that Redis total, then sets `exercises.is_completed`; a `service_test.go` call-log test pins that order. `newly_met` is derived from the counter alone (`total >= 1800 && total-delta < 1800`), so `quests.TargetMetListener.OnTargetMet` fires exactly once per user per local day — the pet slice registers the implementation; a listener error is logged, never returned. Known accepted gap: a crash between the INCRBY and the upsert loses the Postgres row but keeps the Redis count, and the next progress call re-derives and re-upserts it. Both routes sit behind `auth.Require()`. Tests are pure (in-memory `Counter`/`QuestRepo`/`ProgressRepo`); one `DATABASE_URL`+`REDIS_URL`-gated test skips.
```

- [ ] **Step 2: Append to the store bullet**

Add at the end of the `**store**` bullet:
```
`store.SeedDemoRoadmap` inserts a 28-day x 3-exercise demo roadmap; it is TEMPORARY scaffolding for the missing onboarding slice and should be deleted, not extended, when onboarding lands.
```

- [ ] **Step 3: Verify**

```sh
grep -n 'TargetMetListener\|SeedDemoRoadmap' harness/CODEMAP.md
```
Expected: two hits.

- [ ] **Step 4: Commit**

```sh
git add harness/CODEMAP.md && git commit -m "codemap: quests — daily loop, ordering contract, target-met hook"
```

---

## Verification

Run from the worktree root.

```sh
cd backend && go build ./... && go vet ./...
# expect: no output

go test ./...
# expect: ok for internal/auth, internal/config, internal/health, internal/quests, internal/store; no FAIL

env -u DATABASE_URL -u REDIS_URL go test ./... -count=1
# expect: still ok — no test needs a live service

go test ./internal/quests/... -run 'IncrementsRedisBeforeWritingPostgres' -v
# expect: --- PASS — the §5.2 ordering contract

go test ./internal/quests/... -run 'CrossingExactly1800|FurtherProgress' -v
# expect: two --- PASS — the target fires once and only once

go test ./internal/quests/... -run 'Timezone|Boundary|Clamps' -v
# expect: --- PASS for the timezone-boundary and day-28 clamp cases

grep -n 'store.DailyAccumulatedKey\|store.DailyAccumulatedTTL' internal/quests/counter.go
# expect two hits — quests never hand-builds a §4 key

grep -n 'TargetSeconds = 1800' internal/quests/day.go
# expect one hit

grep -n 'TEMPORARY' internal/store/seed.go
# expect one hit — the seed is marked for deletion when onboarding lands

cd .. && python3 tools/harness/cli.py validate; echo exit=$?
# expect: exit=0

git log --oneline main..HEAD
# expect 9 commits, one per task, each with the Co-Authored-By trailer

git status --short
# expect: clean
```

## Notes and open questions

- **The seed is a stopgap.** `store.SeedDemoRoadmap` exists only because `onboarding` was left out of this run (`_run.md`, first Note). Delete it when onboarding lands.
- **`day_number` clamps at 28** rather than ending the roadmap, because the spec never says what day 29 is. A returning learner past day 28 keeps seeing day 28's quests. The real fix (regenerate, or a "course complete" state) needs a product decision.
- **Nothing resets `exercises.is_completed`.** §3.2 makes it per-exercise, not per-day, so a learner revisiting day 5 sees it already ticked. Flagged, not changed — changing it would mean a schema change.
- **`OnTargetMet` is best-effort and fires after the commit.** The spec (§5.2 step 4) does not say whether the pet update is transactional with the progress write. This plan chooses "never lose a recorded study session"; if the human wants them atomic, the listener has to move inside the upsert transaction, which couples quests to pet's tables and breaks the CODEMAP boundary rule.
- **Redis is the source of truth for the day, Postgres is the record.** A crash between the two leaves Postgres behind until the next progress call re-derives the total. Self-healing within the 48h TTL; recorded in CODEMAP so a reviewer does not file it as a bug.
- **`POST /quests/progress` trusts the client's `seconds`.** Nothing stops a client posting 1800 instantly. §5.2 shows the client reporting duration, so this matches the spec, but it means the 30-minute metric is client-asserted. Server-side plausibility limits (e.g. cap per call, cap per day) would be a separate, product-level decision.
