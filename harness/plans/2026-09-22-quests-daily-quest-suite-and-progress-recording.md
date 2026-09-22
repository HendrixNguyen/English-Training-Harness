---
idea: harness/ideas/2026-09-22-run-02/quests-daily-quest-suite-and-progress-recording.md
status: approved
priority: high
merged: false
order: 3
---
# Quests: daily quest suite and progress recording — Plan

**Idea:** `harness/ideas/2026-09-22-run-02/quests-daily-quest-suite-and-progress-recording.md`
**Goal:** Add `backend/internal/quests` — `GET /api/v1/quests/daily` and `POST /api/v1/quests/progress` behind `auth.Require()` — implementing the 1st-thinking doc's §5.2 Redis-counter-first ordering and 1800-second target, the **backend spec's §6.2 request/response DTOs field for field**, and a `quests.Pet` hook (`OnTargetMet` + `State`) the pet slice will register against.

**Spec precedence:** where the 1st-thinking doc (§5.2, §7 endpoint list) and the *Backend Technical Specification* (§6.2) differ, the backend spec wins for the wire contract (AGENTS.md → *Reading the spec*). The §6.2 JSON blocks are the DTOs this plan implements; §5.2 supplies the ordering; §4 supplies the Redis key and TTL; §8 supplies the pet arithmetic the hook triggers.

**Architecture:** A `Service` over four interfaces: `Counter` (Redis `INCRBY` + `EXPIRE` on `store.DailyAccumulatedKey`), `QuestRepo` (active roadmap, the day's exercises, marking one complete), `ProgressRepo` (the `daily_progress` upsert) and `Pet` (in-process hook `OnTargetMet` plus `State`, because the §6.2 progress response carries `pet_health` and `streak_count`; `NopPet` by default). `day_number` and the user's local date are pure functions of a clock, a timezone and `roadmaps.created_at`, in their own file, so the fiddly arithmetic is tested without any I/O. The ordering contract from §5.2 is enforced by a shared call log in the fakes, not by reading the code.

**Tech stack:** Go 1.25 (`backend/go.mod`), Gin, `pgx/v5`, `go-redis/v9` — all already in `go.mod` from slice 1. No new dependencies.

**Depends on:** slice 1 (`store`) and slice 2 (`auth`) — both merged on `main` (`f607282`). The symbols this plan calls were checked against that code: `store.DailyAccumulatedKey(userID string, day time.Time)`, `store.DailyAccumulatedTTL`, `store.NewPostgres`/`*store.Postgres{Pool}`/`.Migrator()`, `store.Migrate(ctx, migrator, store.MigrationsFS)`, `store.NewRedis`/`*store.Redis{Client}`, `auth.Require(tokens *auth.TokenIssuer, sessions auth.SessionStore)`, `auth.UserID(c)`, `auth.ContextUserID`.

**Run every command from `backend/`** unless the step says otherwise. `rg` is not installed — use `grep -n`.

## File structure

| Path | Responsibility |
| --- | --- |
| `backend/internal/quests/day.go` `day_test.go` | local date, `day_number` (1..28 clamped) — pure |
| `backend/internal/quests/counter.go` `counter_test.go` | `Counter` interface + `RedisCounter` |
| `backend/internal/quests/repo.go` | `QuestRepo`/`ProgressRepo` interfaces + models + Postgres impls |
| `backend/internal/quests/pet.go` | `Pet` interface (`OnTargetMet`, `State`), `PetState`, `NopPet` |
| `backend/internal/quests/service.go` `service_test.go` | `Daily()` and `RecordProgress()` |
| `backend/internal/quests/handler.go` `handler_test.go` | the two routes |
| `backend/internal/quests/fakes_test.go` | shared in-memory fakes with the call log |
| `backend/internal/store/seed.go` `seed_test.go` | `SeedDemoRoadmap` — temporary, dies with onboarding |
| `backend/internal/quests/integration_test.go` | `TestIntegration…` — seed + both endpoints against real Postgres/Redis; gated on `TEST_DATABASE_URL`/`TEST_REDIS_URL` (skips locally, **must pass** in CI's `backend-integration` job) |
| `backend/cmd/api/main.go` | hoist `tokens`/`sessions`, mount both routes behind `auth.Require()` |
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
// against it (1st-thinking doc §5.2, §6.1; backend spec §6.2 for the wire DTOs).
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

### Task 3: Repository interfaces, models and the Pet hook

**Files:**
- Create: `backend/internal/quests/repo.go`
- Create: `backend/internal/quests/pet.go`

No tests of their own — these are interfaces and SQL constants, exercised by Tasks 4–6 and by the
integration test in Task 7. Commit them together so the next task compiles.

- [ ] **Step 1: Write `pet.go`**

```go
package quests

import "context"

// PetState is the slice of pet_states (§3.2) that the §6.2 progress response
// reports back as pet_health / streak_count.
type PetState struct {
	Health int // pet_states.health_points
	Streak int // pet_states.current_streak
}

// Pet is quests' view of the pet slice. Packages talk via interfaces, never
// each other's tables (CODEMAP), so quests never reads pet_states itself.
//
// OnTargetMet is the in-process hook fired the first time a user crosses the
// 30-minute target on a given local day — the trigger backend spec §6.2
// describes for POST /quests/progress ("increases plant health (+20%), and
// increments streak"; the arithmetic is §8's success logic). It is called
// AFTER the daily_progress upsert, and its error is logged rather than
// returned: a failing pet update must never roll back a recorded study session.
//
// State is read after the hook so the response carries the post-bump values.
type Pet interface {
	OnTargetMet(ctx context.Context, userID, localDate string) error
	State(ctx context.Context, userID string) (PetState, error)
}

// NopPet is the default until the pet slice registers the real implementation.
// It reports the §3.2 pet_states column defaults (health_points 100,
// current_streak 0) — the state a freshly onboarded pet has — so the §6.2
// response shape is complete from day one. See the plan's Reconciliation notes.
type NopPet struct{}

func (NopPet) OnTargetMet(context.Context, string, string) error { return nil }

func (NopPet) State(context.Context, string) (PetState, error) {
	return PetState{Health: 100, Streak: 0}, nil
}

var _ Pet = NopPet{}
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

// Exercise is a §3.2 `exercises` row. It is the storage model; the §6.2 wire
// DTO is `Task` in service.go (see toTask).
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
cd .. && git add backend && git commit -m "quests: repository interfaces, models and the Pet hook"
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

type fakePet struct {
	log      *callLog
	fired    int
	hookErr  error
	state    PetState
	stateErr error
}

// newFakePet starts mid-course (health 80, streak 4) so a +20/+1 bump is visible.
func newFakePet(l *callLog) *fakePet {
	return &fakePet{log: l, state: PetState{Health: 80, Streak: 4}}
}

// OnTargetMet applies §8's success arithmetic to the fake state so a test can
// prove State() is read after the hook, not before.
func (p *fakePet) OnTargetMet(_ context.Context, userID, localDate string) error {
	p.fired++
	p.log.add("ON TARGET MET %s|%s", userID, localDate)
	if p.hookErr != nil {
		return p.hookErr
	}
	p.state.Health = min(100, p.state.Health+20)
	p.state.Streak++
	return nil
}

func (p *fakePet) State(context.Context, string) (PetState, error) {
	if p.stateErr != nil {
		return PetState{}, p.stateErr
	}
	return p.state, nil
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

### Task 5: `RecordProgress` — the §5.2 write path, §6.2 response

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
	pet      *fakePet
}

func newHarness(t *testing.T, now time.Time) *harness {
	t.Helper()
	log := &callLog{}
	h := &harness{
		log:      log,
		counter:  newFakeCounter(log),
		quests:   newFakeQuestRepo(log),
		progress: newFakeProgressRepo(log),
		pet:      newFakePet(log),
	}
	h.quests.roadmap = &Roadmap{ID: "rm-1", CreatedAt: now.Add(-24 * time.Hour)}
	h.quests.exercises[1] = demoExercises(1)
	h.quests.exercises[2] = demoExercises(2)
	h.svc = NewService(h.counter, h.quests, h.progress, h.pet, fixedClock(now))
	return h
}

func TestRecordProgressIncrementsRedisBeforeWritingPostgres(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 600)
	if err != nil {
		t.Fatalf("RecordProgress() = %v", err)
	}
	if out.DailySecondsSpent != 600 || out.DailyMinutesSpent != 10 || out.IsTargetMet || out.NewlyMet {
		t.Errorf("out = %+v, want seconds=600 minutes=10 isTargetMet=false newlyMet=false", out)
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

func TestProgressBelowTheTargetStillReportsThePetState(t *testing.T) {
	// §6.2: pet_health and streak_count are on every progress response, not
	// only the one that crosses the target.
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 600)
	if err != nil {
		t.Fatalf("RecordProgress() = %v", err)
	}
	if out.PetHealth != 80 || out.StreakCount != 4 {
		t.Errorf("pet = (%d, %d), want the unbumped (80, 4)", out.PetHealth, out.StreakCount)
	}
	if h.pet.fired != 0 {
		t.Errorf("hook fired %d times below the target, want 0", h.pet.fired)
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
		if out.IsTargetMet {
			t.Fatalf("call %d: IsTargetMet true at %ds, want false", i, out.DailySecondsSpent)
		}
	}

	out, err := h.svc.RecordProgress(ctx, "u1", "ex-2-practice", 600)
	if err != nil {
		t.Fatalf("third call: %v", err)
	}
	if out.DailySecondsSpent != 1800 || out.DailyMinutesSpent != 30 {
		t.Errorf("spent = %ds/%dm, want 1800/30", out.DailySecondsSpent, out.DailyMinutesSpent)
	}
	if !out.IsTargetMet || !out.NewlyMet {
		t.Errorf("out = %+v, want isTargetMet and newlyMet both true at exactly 1800s", out)
	}
	if h.pet.fired != 1 {
		t.Errorf("hook fired %d times, want 1", h.pet.fired)
	}
	// State is read AFTER the hook: 80+20 = 100, 4+1 = 5 (§8 success logic).
	if out.PetHealth != 100 || out.StreakCount != 5 {
		t.Errorf("pet = (%d, %d), want the post-hook (100, 5)", out.PetHealth, out.StreakCount)
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

	if !out.IsTargetMet {
		t.Error("IsTargetMet = false after the target was already met")
	}
	if out.NewlyMet {
		t.Error("NewlyMet = true on a second call the same day")
	}
	if h.pet.fired != 1 {
		t.Errorf("hook fired %d times, want 1", h.pet.fired)
	}
	if out.PetHealth != 100 || out.StreakCount != 5 {
		t.Errorf("pet = (%d, %d), want (100, 5) — no second bump", out.PetHealth, out.StreakCount)
	}
	if row := h.progress.rows["u1|2026-09-22"]; row.minutes != 40 {
		t.Errorf("minutes_spent = %d, want 40 (2400s / 60)", row.minutes)
	}
}

func TestMinutesUseIntegerDivision(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 1799)
	if err != nil {
		t.Fatalf("RecordProgress: %v", err)
	}
	row := h.progress.rows["u1|2026-09-22"]
	if row.minutes != 29 || row.targetMet || out.DailyMinutesSpent != 29 {
		t.Errorf("row = %+v, out.minutes = %d; want minutes=29 target=false at 1799s", row, out.DailyMinutesSpent)
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

func TestAPetHookFailureDoesNotFailTheRequest(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.pet.hookErr = errors.New("pet is on fire")

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 1800)
	if err != nil {
		t.Fatalf("RecordProgress() = %v, want nil: a hook failure must not fail the write", err)
	}
	if !out.NewlyMet {
		t.Error("NewlyMet = false despite crossing the target")
	}
}

func TestAPetStateFailureDoesNotFailTheRequest(t *testing.T) {
	// The write is already committed; a 500 here would make the client retry
	// and double-count. Pet fields fall back to zero and are logged.
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.pet.stateErr = errors.New("pet_states unreachable")

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 600)
	if err != nil {
		t.Fatalf("RecordProgress() = %v, want nil", err)
	}
	if out.DailySecondsSpent != 600 || out.PetHealth != 0 || out.StreakCount != 0 {
		t.Errorf("out = %+v, want the progress recorded and zero pet fields", out)
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
go test ./internal/quests/... -run 'Record|Crossing|Further|Minutes|Pet|Timezone'
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

// ProgressResult is the POST /api/v1/quests/progress 200 body — backend spec
// §6.2, field for field.
type ProgressResult struct {
	DailySecondsSpent int64 `json:"daily_seconds_spent"`
	DailyMinutesSpent int   `json:"daily_minutes_spent"`
	IsTargetMet       bool  `json:"is_target_met"`
	PetHealth         int   `json:"pet_health"`
	StreakCount       int   `json:"streak_count"`

	// NewlyMet is true only on the call that crossed TargetSeconds. It is the
	// once-only trigger for Pet.OnTargetMet and is asserted by tests; §6.2 has
	// no such field, so it never reaches the wire.
	NewlyMet bool `json:"-"`
}

// Service implements the daily loop (1st-thinking §5.2; backend spec §6.2).
type Service struct {
	counter  Counter
	quests   QuestRepo
	progress ProgressRepo
	pet      Pet
	now      func() time.Time
}

// NewService wires the collaborators. now is injectable so day boundaries are
// testable; pet may be NopPet{} until the pet slice registers the real one.
func NewService(counter Counter, quests QuestRepo, progress ProgressRepo, pet Pet, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	if pet == nil {
		pet = NopPet{}
	}
	return &Service{counter: counter, quests: quests, progress: progress, pet: pet, now: now}
}

// RecordProgress implements §5.2 steps 2-4, in that order:
//
//	1. INCRBY the Redis counter (+ EXPIRE) and read the running total back.
//	2. Upsert daily_progress from that total — Redis is the single source of
//	   truth for the day, so bursts of calls cannot disagree.
//	3. Mark the exercise complete.
//	4. Fire Pet.OnTargetMet, but only on the call that crossed 1800s, then read
//	   Pet.State so the §6.2 response carries pet_health / streak_count.
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
		// rolled back by a pet failure.
		if err := s.pet.OnTargetMet(ctx, userID, date); err != nil {
			log.Printf("quests: pet target-met hook failed for user %s on %s: %v", userID, date, err)
		}
	}

	// Also best-effort: the write is done, and a 500 here would make the client
	// retry and double-count. GET /pet/status (pet slice) is the authoritative read.
	pet, err := s.pet.State(ctx, userID)
	if err != nil {
		log.Printf("quests: reading pet state for user %s: %v", userID, err)
		pet = PetState{}
	}

	return ProgressResult{
		DailySecondsSpent: total,
		DailyMinutesSpent: int(total / 60),
		IsTargetMet:       targetMet,
		PetHealth:         pet.Health,
		StreakCount:       pet.Streak,
		NewlyMet:          newlyMet,
	}, nil
}
```

(Task 6 adds `"encoding/json"` to this import block when it introduces `Task`/`toTask`; importing it now would be an unused-import compile error.)

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/quests/... -v
```
Expected: every service test `--- PASS`, including the exact call-order assertion and the (100, 5) post-hook pet state.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "quests: RecordProgress with Redis-first ordering, once-only pet hook and the §6.2 response"
```

---

### Task 6: `Daily()` and both HTTP handlers — §6.2 wire shapes

**Files:**
- Modify: `backend/internal/quests/service.go` (add `Task`, `DailySuite`, `toTask`, `Daily`)
- Create: `backend/internal/quests/handler.go`
- Test: `backend/internal/quests/service_test.go` (append)
- Test: `backend/internal/quests/handler_test.go`

The two response bodies and the request body are copied from the backend spec §6.2 JSON blocks. Do
not rename, add or drop a field without updating the Reconciliation section.

- [ ] **Step 1: Write the failing tests**

Append to `service_test.go` (and add `"encoding/json"` to its import block):
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
	if got.Date != "2026-09-22" {
		t.Errorf("Date = %q, want 2026-09-22", got.Date)
	}
	if got.DayNumber != 2 {
		t.Errorf("DayNumber = %d, want 2", got.DayNumber)
	}
	if got.TotalMinutesRequired != 30 {
		t.Errorf("TotalMinutesRequired = %d, want 30 (§6.2)", got.TotalMinutesRequired)
	}
	if len(got.Tasks) != 3 {
		t.Fatalf("len(Tasks) = %d, want 3", len(got.Tasks))
	}
	if got.AccumulatedSeconds != 900 || got.IsTargetMet {
		t.Errorf("accumulated = %d isTargetMet = %t, want 900/false", got.AccumulatedSeconds, got.IsTargetMet)
	}
	seen := map[string]bool{}
	for _, task := range got.Tasks {
		seen[task.TaskType] = true
		if task.DurationMinutes != DefaultTaskMinutes {
			t.Errorf("task %s DurationMinutes = %d, want the %d-minute default", task.ID, task.DurationMinutes, DefaultTaskMinutes)
		}
	}
	for _, want := range []string{"vocabulary", "reading", "practice"} {
		if !seen[want] {
			t.Errorf("missing the %s task", want)
		}
	}
}

func TestDailyDateIsTheUsersLocalDate(t *testing.T) {
	// 18:30Z is already the 23rd in Ho Chi Minh City; §6.2's `date` must agree
	// with the day the counter and daily_progress are keyed on.
	h := newHarness(t, time.Date(2026, time.September, 22, 18, 30, 0, 0, time.UTC))
	h.quests.timezone = "Asia/Ho_Chi_Minh"

	got, err := h.svc.Daily(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Daily() = %v", err)
	}
	if got.Date != "2026-09-23" {
		t.Errorf("Date = %q, want 2026-09-23", got.Date)
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
	if len(got.Tasks) != 3 {
		t.Errorf("len(Tasks) = %d, want 3", len(got.Tasks))
	}
}

func TestDailyWithNoRowsForTheDayReturnsAnEmptyList(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	delete(h.quests.exercises, 2)

	got, err := h.svc.Daily(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Daily() = %v", err)
	}
	if got.Tasks == nil || len(got.Tasks) != 0 {
		t.Errorf("Tasks = %#v, want an empty non-nil slice (serialises as [])", got.Tasks)
	}
}

func TestTaskTitleAndDurationComeFromContentJSON(t *testing.T) {
	// §6.2 puts title and duration_minutes on each task; §3.2 has no such
	// columns, so they ride inside content_json.
	rich := toTask(Exercise{
		ID: "ex-1", TaskType: "vocabulary", IsCompleted: true,
		ContentJSON: json.RawMessage(`{"title":"10 Key Business Email Phrasings","duration_minutes":15,"words":[]}`),
	})
	if rich.ID != "ex-1" || rich.TaskType != "vocabulary" || !rich.IsCompleted {
		t.Errorf("identity fields not copied: %+v", rich)
	}
	if rich.Title != "10 Key Business Email Phrasings" || rich.DurationMinutes != 15 {
		t.Errorf("title/duration = %q/%d, want the content_json values", rich.Title, rich.DurationMinutes)
	}
	if string(rich.ContentJSON) != `{"title":"10 Key Business Email Phrasings","duration_minutes":15,"words":[]}` {
		t.Errorf("ContentJSON was altered: %s", rich.ContentJSON)
	}

	bare := toTask(Exercise{ID: "ex-2", TaskType: "reading", ContentJSON: json.RawMessage(`{"passage":"..."}`)})
	if bare.Title != "" || bare.DurationMinutes != DefaultTaskMinutes {
		t.Errorf("bare task = %q/%d, want \"\"/%d", bare.Title, bare.DurationMinutes, DefaultTaskMinutes)
	}

	broken := toTask(Exercise{ID: "ex-3", TaskType: "practice", ContentJSON: json.RawMessage(`not json`)})
	if broken.DurationMinutes != DefaultTaskMinutes {
		t.Errorf("malformed content_json must still yield the default duration, got %d", broken.DurationMinutes)
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

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// newQuestRouter mounts the routes with a stub that injects the authenticated
// user under auth.ContextUserID, standing in for auth.Require() (covered in the
// auth slice, middleware_test.go).
func newQuestRouter(svc *Service, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	inject := func(c *gin.Context) {
		c.Set(auth.ContextUserID, userID)
		c.Next()
	}
	g := r.Group("/api/v1", inject)
	g.GET("/quests/daily", DailyHandler(svc))
	g.POST("/quests/progress", ProgressHandler(svc))
	return r
}

func postJSON(r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDailyHandlerReturnsTheSpec62Body(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	w := httptest.NewRecorder()
	newQuestRouter(h.svc, "u1").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/quests/daily", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	// Exactly the backend spec §6.2 GET /quests/daily shape.
	var body struct {
		Date                 string `json:"date"`
		DayNumber            int    `json:"day_number"`
		TotalMinutesRequired int    `json:"total_minutes_required"`
		AccumulatedSeconds   int64  `json:"accumulated_seconds"`
		IsTargetMet          bool   `json:"is_target_met"`
		Tasks                []struct {
			ID              string          `json:"id"`
			TaskType        string          `json:"task_type"`
			Title           string          `json:"title"`
			DurationMinutes int             `json:"duration_minutes"`
			IsCompleted     bool            `json:"is_completed"`
			ContentJSON     json.RawMessage `json:"content_json"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding: %v (%s)", err, w.Body.String())
	}
	if body.Date != "2026-09-22" || body.DayNumber != 2 || body.TotalMinutesRequired != 30 || len(body.Tasks) != 3 {
		t.Errorf("body = %+v", body)
	}
	if body.Tasks[0].DurationMinutes != DefaultTaskMinutes || len(body.Tasks[0].ContentJSON) == 0 {
		t.Errorf("task[0] = %+v, want duration_minutes=%d and content_json present", body.Tasks[0], DefaultTaskMinutes)
	}
	for _, stale := range []string{`"exercises"`, `"total_seconds"`, `"target_met"`} {
		if strings.Contains(w.Body.String(), stale) {
			t.Errorf("body still uses pre-reconciliation field %s: %s", stale, w.Body.String())
		}
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

func TestProgressHandlerReturnsTheSpec62Body(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	// The full §6.2 request, user_answers included.
	w := postJSON(newQuestRouter(h.svc, "u1"), "/api/v1/quests/progress",
		`{"exercise_id":"ex-2-reading","duration_seconds":1800,"user_answers":{"q1":"A"}}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	for _, want := range []string{
		`"daily_seconds_spent":1800`, `"daily_minutes_spent":30`, `"is_target_met":true`,
		`"pet_health":100`, `"streak_count":5`,
	} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("body = %s, missing %s", w.Body.String(), want)
		}
	}
	for _, stale := range []string{`newly_met`, `"total_seconds"`, `"target_met"`} {
		if strings.Contains(w.Body.String(), stale) {
			t.Errorf("body leaks non-§6.2 field %s: %s", stale, w.Body.String())
		}
	}
}

func TestProgressHandlerAcceptsABodyWithoutUserAnswers(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))

	w := postJSON(newQuestRouter(h.svc, "u1"), "/api/v1/quests/progress",
		`{"exercise_id":"ex-2-reading","duration_seconds":600}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — user_answers is optional; body = %s", w.Code, w.Body.String())
	}
}

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
	} {
		if w := postJSON(r, "/api/v1/quests/progress", body); w.Code != http.StatusBadRequest {
			t.Errorf("body %q → status %d, want 400", body, w.Code)
		}
	}
}

func TestProgressHandlerReturns404ForAnUnknownExercise(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	h.quests.markErr = ErrExerciseNotFound

	w := postJSON(newQuestRouter(h.svc, "u1"), "/api/v1/quests/progress", `{"exercise_id":"nope","duration_seconds":600}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"error":"exercise_not_found"`) {
		t.Errorf("body = %s, want the exercise_not_found error", w.Body.String())
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/quests/... -run 'Daily|Progress|Task'
```
Expected: build failure, `undefined: DailyHandler` (and `toTask`, `DefaultTaskMinutes`).

- [ ] **Step 3: Implement the §6.2 daily body and `Daily` in `service.go`**

Add `"encoding/json"` to `service.go`'s import block, then append:
```go
// DefaultTaskMinutes is the §6.2 task length ("3x 10-min tasks"), used when an
// exercise's content_json carries no duration_minutes.
const DefaultTaskMinutes = 10

// Task is one entry of the GET /api/v1/quests/daily `tasks` array (backend spec
// §6.2). title and duration_minutes are not §3.2 columns; toTask reads them
// from content_json.
type Task struct {
	ID              string          `json:"id"`
	TaskType        string          `json:"task_type"`
	Title           string          `json:"title"`
	DurationMinutes int             `json:"duration_minutes"`
	IsCompleted     bool            `json:"is_completed"`
	ContentJSON     json.RawMessage `json:"content_json"`
}

// DailySuite is the GET /api/v1/quests/daily 200 body — backend spec §6.2,
// field for field.
type DailySuite struct {
	Date                 string `json:"date"`
	DayNumber            int    `json:"day_number"`
	TotalMinutesRequired int    `json:"total_minutes_required"`
	AccumulatedSeconds   int64  `json:"accumulated_seconds"`
	IsTargetMet          bool   `json:"is_target_met"`
	Tasks                []Task `json:"tasks"`
}

// toTask maps a §3.2 exercises row onto the §6.2 task DTO. A missing title is
// "", a missing or non-positive duration is DefaultTaskMinutes. Malformed
// content_json is the generator's bug, not a reason to 500 the whole day, so the
// unmarshal error is deliberately ignored and the defaults apply.
func toTask(e Exercise) Task {
	var meta struct {
		Title           string `json:"title"`
		DurationMinutes int    `json:"duration_minutes"`
	}
	_ = json.Unmarshal(e.ContentJSON, &meta)
	if meta.DurationMinutes <= 0 {
		meta.DurationMinutes = DefaultTaskMinutes
	}
	return Task{
		ID:              e.ID,
		TaskType:        e.TaskType,
		Title:           meta.Title,
		DurationMinutes: meta.DurationMinutes,
		IsCompleted:     e.IsCompleted,
		ContentJSON:     e.ContentJSON,
	}
}

// Daily resolves the active roadmap, computes today's day_number in the user's
// timezone and returns that day's tasks plus today's running total.
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
	date := LocalDate(now, loc)
	day := DayNumber(roadmap.CreatedAt, now, loc)

	exercises, err := s.quests.ExercisesForDay(ctx, roadmap.ID, day)
	if err != nil {
		return DailySuite{}, err
	}
	total, err := s.counter.Total(ctx, userID, date)
	if err != nil {
		return DailySuite{}, err
	}

	tasks := make([]Task, 0, len(exercises)) // never nil: serialises as []
	for _, e := range exercises {
		tasks = append(tasks, toTask(e))
	}

	return DailySuite{
		Date:                 date,
		DayNumber:            day,
		TotalMinutesRequired: TargetSeconds / 60,
		AccumulatedSeconds:   total,
		IsTargetMet:          total >= TargetSeconds,
		Tasks:                tasks,
	}, nil
}
```

`Exercise` in `repo.go` keeps no JSON tags — it is the storage model and never reaches the wire.

- [ ] **Step 4: Implement the handlers**

`backend/internal/quests/handler.go`:
```go
package quests

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// DailyHandler serves GET /api/v1/quests/daily (backend spec §6.2). It must be
// mounted behind auth.Require().
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
		c.JSON(http.StatusOK, suite)
	}
}

// progressRequest is the backend spec §6.2 POST /api/v1/quests/progress body.
type progressRequest struct {
	ExerciseID      string `json:"exercise_id" binding:"required"`
	DurationSeconds int64  `json:"duration_seconds" binding:"required,gt=0"`
	// UserAnswers is part of the §6.2 request and is accepted so a
	// spec-conformant client is never rejected, but it is not persisted: no
	// §3.2 table stores answers and §6.2 does not say what becomes of them.
	// See the plan's Reconciliation section.
	UserAnswers json.RawMessage `json:"user_answers"`
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

		out, err := svc.RecordProgress(c.Request.Context(), userID, req.ExerciseID, req.DurationSeconds)
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

Error bodies follow the merged auth slice's `{"error": "<snake_case_code>"}` convention
(`auth/handler.go`, `auth/middleware.go`); §6.2 defines no error shapes, so that convention is the
contract here. `auth.ContextUserID` is `"user_id"` on `main` (`auth/middleware.go:11`) — the test uses
the constant, not the literal.

- [ ] **Step 5: Run and confirm it passes**

```sh
go test ./internal/quests/... -v
```
Expected: every test `--- PASS`.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend && git commit -m "quests: GET /quests/daily and POST /quests/progress with the §6.2 DTOs"
```

---

### Task 7: Demo roadmap seed and the integration test

**Files:**
- Create: `backend/internal/store/seed.go`
- Test: `backend/internal/store/seed_test.go`
- Create: `backend/internal/quests/integration_test.go`

The seed exists only because `onboarding` has no slice in this run. Its doc comment says so, so that
whoever builds onboarding deletes it rather than extending it.

The integration test is named `TestIntegration…`, so CI's `backend-integration` job **counts it and
fails if it skips** (`.github/workflows/ci.yml`, "Integration tests must run, not skip"). It is gated on
`TEST_DATABASE_URL` / `TEST_REDIS_URL` — never on the production `DATABASE_URL` / `REDIS_URL` — exactly
as `internal/store/integration_test.go` and `internal/auth/integration_test.go` are, because
`internal/store`'s tests drop every table in whatever database they are pointed at.

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
// user and returns the roadmap id. Each content_json carries the `title` and
// `duration_minutes` that the backend spec §6.2 daily response exposes per
// task — §3.2 has no columns for them, so quests reads them from here.
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
			content := fmt.Sprintf(`{"title":"Day %d %s","duration_minutes":10,"day":%d,"task":"%s"}`, day, taskType, day, taskType)
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

// TestIntegration* names are counted by CI's backend-integration job, which
// exports TEST_DATABASE_URL/TEST_REDIS_URL and fails on any --- SKIP, so this
// test must pass there. Locally it skips without them — `go test ./...` stays
// green with no services. It deliberately does NOT read DATABASE_URL/REDIS_URL:
// those are the production variables (spec §9), and internal/store's tests
// drop every table in the database they are pointed at (see
// internal/store/integration_test.go and internal/auth/integration_test.go for
// the same convention). Run with -p 1 (make test-integration): all packages
// share the one database.
func TestIntegrationDailyAndProgressAgainstRealServices(t *testing.T) {
	dbURL, redisURL := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_REDIS_URL")
	if dbURL == "" || redisURL == "" {
		t.Skip("TEST_DATABASE_URL/TEST_REDIS_URL unset; run `make up` and export them to run integration tests")
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

	svc := NewService(counter, repo, repo, NopPet{}, func() time.Time { return now })

	suite, err := svc.Daily(ctx, userID)
	if err != nil {
		t.Fatalf("Daily: %v", err)
	}
	if suite.Date != LocalDate(now, time.UTC) {
		t.Errorf("Date = %q, want %q", suite.Date, LocalDate(now, time.UTC))
	}
	if suite.DayNumber != 1 {
		t.Errorf("DayNumber = %d, want 1 on a freshly seeded roadmap", suite.DayNumber)
	}
	if suite.TotalMinutesRequired != 30 || suite.AccumulatedSeconds != 0 || suite.IsTargetMet {
		t.Errorf("suite = %+v, want 30 required, 0 accumulated, target unmet", suite)
	}
	if len(suite.Tasks) != 3 {
		t.Fatalf("len(Tasks) = %d, want 3", len(suite.Tasks))
	}
	if suite.Tasks[0].Title == "" || suite.Tasks[0].DurationMinutes != 10 {
		t.Errorf("task = %+v, want the seed's title and 10-minute duration", suite.Tasks[0])
	}

	out, err := svc.RecordProgress(ctx, userID, suite.Tasks[0].ID, 1800)
	if err != nil {
		t.Fatalf("RecordProgress: %v", err)
	}
	if out.DailySecondsSpent != 1800 || out.DailyMinutesSpent != 30 || !out.IsTargetMet || !out.NewlyMet {
		t.Errorf("out = %+v, want 1800s/30m and the target newly met", out)
	}
	if out.PetHealth != 100 || out.StreakCount != 0 {
		t.Errorf("pet = (%d, %d), want NopPet's §3.2 defaults (100, 0)", out.PetHealth, out.StreakCount)
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

	var completed bool
	if err := pg.Pool.QueryRow(ctx, `SELECT is_completed FROM exercises WHERE id = $1`, suite.Tasks[0].ID).Scan(&completed); err != nil {
		t.Fatalf("reading exercise: %v", err)
	}
	if !completed {
		t.Error("exercises.is_completed = false after progress was recorded against it")
	}

	again, err := svc.Daily(ctx, userID)
	if err != nil {
		t.Fatalf("second Daily: %v", err)
	}
	if again.AccumulatedSeconds != 1800 || !again.IsTargetMet || !again.Tasks[0].IsCompleted {
		t.Errorf("second Daily = %+v, want 1800s accumulated, target met, task[0] completed", again)
	}
}
```

- [ ] **Step 3: Confirm it skips without services, and the suite stays green**

```sh
env -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1
go test ./internal/quests/... -count=1 -run Integration -v
```
Expected: `ok` everywhere; the second command prints `--- SKIP: TestIntegrationDailyAndProgressAgainstRealServices` with the `TEST_DATABASE_URL/TEST_REDIS_URL unset` message.

- [ ] **Step 4: Run it for real (CI will; do this locally if Docker is available)**

```sh
POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose up -d --wait
export TEST_DATABASE_URL='postgres://english:english@localhost:5433/english?sslmode=disable'
export TEST_REDIS_URL='redis://localhost:6380/0'
make test-integration      # = go test ./... -count=1 -v -run Integration -p 1
docker compose down
```
Expected: `--- PASS` for every `TestIntegration*` (store's, auth's and this one) and no `--- SKIP`. Keep
`-p 1`: every package's integration tests share this one database and `internal/store` drops the
schema around its own. If Docker is unavailable, say so in the execution summary — CI's
`backend-integration` job is the gate either way.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "quests: demo roadmap seed and end-to-end integration test"
```

---

### Task 8: Mount the routes behind `auth.Require()`

**Files:**
- Modify: `backend/cmd/api/main.go`

On `main`, `cmd/api/main.go:49-54` builds the token issuer and session store *inline* inside the
`auth.NewService(...)` call — there are **no** `tokens` / `sessions` locals to reuse, and lines 59-60
are a placeholder comment (`// Later slices mount their routes on this group: // guarded := ...`).
Hoist the two values so `Require()` shares them with sign-in, and replace the placeholder.

- [ ] **Step 1: Wire it**

Add `"github.com/HendrixNguyen/English-Training-Harness/backend/internal/quests"` to the imports, then
replace the block from `authSvc := auth.NewService(` through the placeholder comment with:
```go
	tokens := auth.NewTokenIssuer(cfg.JWTSecret, time.Now)
	sessions := auth.NewRedisSessionStore(rdb)
	authSvc := auth.NewService(
		auth.NewGoogleClient(cfg.GoogleClientID, cfg.GoogleClientSecret),
		auth.NewPgUserRepo(pg.Pool),
		sessions,
		tokens,
	)

	questRepo := quests.NewPgRepo(pg.Pool) // satisfies both QuestRepo and ProgressRepo
	questSvc := quests.NewService(
		quests.NewRedisCounter(rdb),
		questRepo,
		questRepo,
		quests.NopPet{}, // the pet slice replaces this
		time.Now,
	)

	v1 := r.Group("/api/v1")
	v1.POST("/auth/google", auth.Handler(authSvc))

	guarded := v1.Group("", auth.Require(tokens, sessions))
	guarded.GET("/quests/daily", quests.DailyHandler(questSvc))
	guarded.POST("/quests/progress", quests.ProgressHandler(questSvc))
```

- [ ] **Step 2: Confirm build, vet and tests**

```sh
go build ./... && go vet ./... && go test ./... -count=1
grep -n 'auth.Require(tokens, sessions)' cmd/api/main.go
```
Expected: clean build/vet; `ok` for `internal/auth`, `internal/config`, `internal/health`,
`internal/quests`, `internal/store`; one grep hit.

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
- **quests** — the daily loop (1st-thinking §5.2; wire contract = backend spec §6.2). `GET /api/v1/quests/daily` resolves `roadmaps.is_active`, computes `day_number` = calendar days since `roadmaps.created_at` in `users.timezone`, +1, clamped to 1..28, and returns `{date, day_number, total_minutes_required: 30, accumulated_seconds, is_target_met, tasks[]}`, each task `{id, task_type, title, duration_minutes, is_completed, content_json}` — `title`/`duration_minutes` are read from `content_json` (no §3.2 column; default 10 min); 404 `no_active_roadmap` when there is none. `POST /api/v1/quests/progress` (`{exercise_id, duration_seconds, user_answers?}` — `user_answers` accepted, not persisted) does `INCRBY daily:accumulated:{user_id}:{local-date}` + `EXPIRE` 48h **first**, then upserts `daily_progress` (`minutes_spent = total/60`, `is_target_met = total >= 1800`) from that Redis total, then sets `exercises.is_completed`, and answers `{daily_seconds_spent, daily_minutes_spent, is_target_met, pet_health, streak_count}`; a `service_test.go` call-log test pins that order. Crossing 1800 is derived from the counter alone (`total >= 1800 && total-delta < 1800`), so `quests.Pet.OnTargetMet` fires exactly once per user per local day — the pet slice registers the implementation (§6.2/§8: health +20, streak +1) and `Pet.State` supplies `pet_health`/`streak_count`; until then `NopPet` reports the §3.2 defaults (100, 0). Hook and state errors are logged, never returned. Known accepted gap: a crash between the INCRBY and the upsert loses the Postgres row but keeps the Redis count, and the next progress call re-derives and re-upserts it. Both routes sit behind `auth.Require()`. Tests are pure (in-memory `Counter`/`QuestRepo`/`ProgressRepo`/`Pet`); `TestIntegrationDailyAndProgressAgainstRealServices` is gated on `TEST_DATABASE_URL`+`TEST_REDIS_URL` (skips locally, must pass in CI's `backend-integration` job).
```

- [ ] **Step 2: Append to the store bullet**

Add at the end of the `**store**` bullet:
```
`store.SeedDemoRoadmap` inserts a 28-day x 3-exercise demo roadmap whose `content_json` carries the §6.2 `title`/`duration_minutes`; it is TEMPORARY scaffolding for the missing onboarding slice and should be deleted, not extended, when onboarding lands.
```

- [ ] **Step 3: Verify**

```sh
grep -n 'quests.Pet\|SeedDemoRoadmap' harness/CODEMAP.md
```
Expected: two hits.

- [ ] **Step 4: Commit**

```sh
git add harness/CODEMAP.md && git commit -m "codemap: quests — daily loop, §6.2 DTOs, ordering contract, pet hook"
```

---

## Verification

Run from the worktree root.

```sh
cd backend && go build ./... && go vet ./...
# expect: no output

go test ./... -count=1
# expect: ok for internal/auth, internal/config, internal/health, internal/quests, internal/store; no FAIL

env -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1
# expect: still ok — no test needs a live service

go test ./internal/quests/... -run 'IncrementsRedisBeforeWritingPostgres' -v
# expect: --- PASS — the §5.2 ordering contract

go test ./internal/quests/... -run 'CrossingExactly1800|FurtherProgress' -v
# expect: two --- PASS — the target fires once and only once, and State is read after the hook

go test ./internal/quests/... -run 'Timezone|Boundary|Clamps|LocalDate' -v
# expect: --- PASS for the timezone-boundary, local-date and day-28 clamp cases

go test ./internal/quests/... -run 'Handler|Spec62|Task' -v
# expect: --- PASS — the §6.2 wire shapes, request validation and content_json mapping

grep -n '"accumulated_seconds"\|"total_minutes_required"\|"tasks"' internal/quests/service.go
# expect 3 hits — §6.2 GET /quests/daily body

grep -n '"daily_seconds_spent"\|"daily_minutes_spent"\|"pet_health"\|"streak_count"' internal/quests/service.go
# expect 4 hits — §6.2 POST /quests/progress body

grep -n '"duration_seconds"\|"user_answers"' internal/quests/handler.go
# expect 2 hits — §6.2 POST /quests/progress request

grep -rn --include='*.go' --exclude='*_test.go' '"total_seconds"\|"target_met"\|"newly_met"\|"exercises"\|"seconds"' internal/quests/
# expect: no hits — the pre-reconciliation field names are gone from the wire (the tests mention them only to assert their absence)

grep -n 'store.DailyAccumulatedKey\|store.DailyAccumulatedTTL' internal/quests/counter.go
# expect two hits — quests never hand-builds a §4 key

grep -n 'TargetSeconds = 1800' internal/quests/day.go
# expect one hit

grep -n 'TEMPORARY' internal/store/seed.go
# expect one hit — the seed is marked for deletion when onboarding lands

grep -rn 'Getenv("DATABASE_URL")\|Getenv("REDIS_URL")' internal/quests/ internal/store/seed*.go
# expect: no hits — integration gating is on TEST_DATABASE_URL / TEST_REDIS_URL only

grep -c '^func TestIntegration' internal/quests/integration_test.go
# expect: 1 — CI counts it and fails if it skips there

grep -n 'auth.Require(tokens, sessions)' cmd/api/main.go
# expect one hit

cd .. && python3 tools/harness/cli.py validate; echo exit=$?
# expect: exit=0

git log --oneline main..HEAD
# expect 9 commits, one per task, each with the Co-Authored-By trailer

git status --short
# expect: clean
```

After pushing the branch: `gh run list --branch <branch>` must show `backend-unit`, `backend-integration`
and `harness-tooling` green; `backend-integration` is where the new `TestIntegration…` actually runs.

## Notes and open questions

- **The seed is a stopgap.** `store.SeedDemoRoadmap` exists only because `onboarding` was left out of this run (`_run.md`, first Note). Delete it when onboarding lands. Whatever generates real roadmaps must write `title` and `duration_minutes` into each exercise's `content_json`, or the §6.2 daily response will show empty titles and 10-minute defaults.
- **`day_number` clamps at 28** rather than ending the roadmap, because the spec never says what day 29 is. A returning learner past day 28 keeps seeing day 28's quests. The real fix (regenerate, or a "course complete" state) needs a product decision.
- **Nothing resets `exercises.is_completed`.** §3.2 makes it per-exercise, not per-day, so a learner revisiting day 5 sees it already ticked. Flagged, not changed — changing it would mean a schema change.
- **`Pet.OnTargetMet` is best-effort and fires after the commit.** §6.2 says the progress call "increases plant health (+20%), and increments streak" but not whether that is transactional with the progress write. This plan chooses "never lose a recorded study session"; if the human wants them atomic, the hook has to move inside the upsert transaction, which couples quests to pet's tables and breaks the CODEMAP boundary rule. **For the pet slice:** §8's hourly cron *also* has a "Success Logic" (+20 / +1 at local midnight). Applying it both on the progress call and in the cron would double-bump; the pet slice must pick one (this plan assumes the progress-time hook, per §6.2, and the cron handles only the inactivity decay).
- **`pet_health` / `streak_count` before the pet slice exists** come from `NopPet` — the §3.2 `pet_states` defaults (100, 0), i.e. a freshly onboarded pet. This is a placeholder, not a read of anything; it is replaced when the pet slice registers its `Pet`. A `Pet.State` error is logged and reported as (0, 0) rather than failing the request, because the progress write has already committed and a retry would double-count.
- **`user_answers` is accepted and dropped.** §6.2 puts it on the request but no §3.2 table stores answers and no endpoint reads them back. Persisting them would be a schema change and a product decision (essay grading in §5.1's `TaskEssayGrading` is the likely consumer). Accepting the field keeps a spec-conformant client from being rejected.
- **Redis is the source of truth for the day, Postgres is the record.** A crash between the two leaves Postgres behind until the next progress call re-derives the total. Self-healing within the 48h TTL; recorded in CODEMAP so a reviewer does not file it as a bug.
- **`POST /quests/progress` trusts the client's `duration_seconds`.** Nothing stops a client posting 1800 instantly. §5.2 shows the client reporting duration, so this matches the spec, but it means the 30-minute metric is client-asserted. Server-side plausibility limits (e.g. cap per call, cap per day) would be a separate, product-level decision.

## Reconciliation

Reconciled on 2026-09-22 by the evaluator against the *Backend Technical Specification* (§4, §6.2, §8) and
the code merged on `main` at `f607282` (`internal/store`, `internal/auth`, `cmd/api`, `.github/workflows/ci.yml`).
The plan had been written against the 1st-thinking doc only and diverged from the §6.2 DTO contract on
nearly every wire field, so it went `approved → draft` and needs re-approval. Every change and its
citation:

**Wire contract (backend spec §6.2 — the JSON blocks are the contract per AGENTS.md → *Reading the spec*)**

1. `GET /quests/daily` response renamed/extended to `{date, day_number, total_minutes_required, accumulated_seconds, is_target_met, tasks[]}` (was `{day_number, total_seconds, target_met, exercises[]}`). §6.2 `Response (200 OK)`. Tasks 6, 9, Verification.
2. Each task is `{id, task_type, title, duration_minutes, is_completed, content_json}` (was `{id, day_number, task_type, content_json, is_completed}`). `title` and `duration_minutes` are not §3.2 `exercises` columns, so they are read from `content_json` with defaults `""` / 10 (`DefaultTaskMinutes`, §6.2 "3x 10-min tasks"); a new `Task` DTO separates the wire shape from the `Exercise` storage model. §6.2 `tasks[]` item; §3.2 DDL. Tasks 3, 6, 7 (seed writes both keys).
3. `POST /quests/progress` request is `{exercise_id, duration_seconds, user_answers?}` (was `{exercise_id, seconds}`). `user_answers` is accepted (`json.RawMessage`, optional) and not persisted — see open questions. The old `seconds` name is rejected with 400, not aliased. §6.2 `Request Body`. Task 6.
4. `POST /quests/progress` response is `{daily_seconds_spent, daily_minutes_spent, is_target_met, pet_health, streak_count}` (was `{total_seconds, target_met, newly_met}`). `newly_met` stays as an internal, `json:"-"` field because it is the once-only hook trigger the tests assert. §6.2 `Response (200 OK)`. Tasks 5, 6.
5. Because the response carries `pet_health` / `streak_count`, the `TargetMetListener` hook became a `quests.Pet` interface with `OnTargetMet` **and** `State`; `NopPet` reports the §3.2 `pet_states` defaults until the pet slice registers. Fakes, service and tests updated; new tests prove `State` is read after the hook (80→100, 4→5) and that pet fields appear on sub-target calls too. §6.2 description ("increases plant health (+20%), and increments streak"), §8 success logic, §3.2 `pet_states` defaults, CODEMAP boundary rule. Tasks 3, 4, 5, 8, 9.
6. `total_minutes_required` is `TargetSeconds / 60` = 30 (§6.2 value; §1/§8 "30 mins"). Task 6.
7. Error shapes: §6.2 defines none; the plan keeps the merged auth slice's `{"error": "<code>"}` convention (400 `invalid_request`, 401 `unauthorized`, 404 `no_active_roadmap` / `exercise_not_found`, 500 `internal_error`). Stated explicitly in Task 6 rather than implied.

**§8 — what "target met" triggers**

8. The hook's doc comment and the open questions now cite §8's arithmetic (`Health = Min(100, Health + 20)`, `Streak + 1`) and warn the pet slice that §8's hourly cron carries the same "Success Logic", so the bump must be applied once, not both at progress time and at local midnight. No behaviour change in quests; it fires the hook once per user per local day as before.

**§4 — Redis keys**

9. Checked, unchanged: `daily:accumulated:{user_id}:{YYYY-MM-DD}` String(Int) 48h maps to `store.DailyAccumulatedKey` / `store.DailyAccumulatedTTL` on `main` (`internal/store/keys.go:14,30-32`); the counter test asserts both. `counterKey` parses the localised date string back to `time.Time` because the merged builder takes `(userID string, day time.Time)`.

**Conventions that changed since the plan was written (merged `main`)**

10. Integration test gates on `TEST_DATABASE_URL` / `TEST_REDIS_URL`, never `DATABASE_URL` / `REDIS_URL` — `internal/store/integration_test.go:16-18`, `internal/auth/integration_test.go:18-20`, `Makefile` `test-integration`. Its `TestIntegration…` name means CI's `backend-integration` job counts it and fails on `--- SKIP` (`ci.yml:84-104`), so it must pass there, not skip. The "skip locally" check, the "run for real" step (`--wait`, non-default ports, `make test-integration` with `-p 1`) and the CODEMAP wording were rewritten accordingly. Task 7, Task 9, Verification, File structure.
11. `cmd/api/main.go` on `main` (`:49-54`) builds `auth.NewTokenIssuer` / `auth.NewRedisSessionStore` inline inside `auth.NewService(...)`; there are no `tokens` / `sessions` locals as the plan assumed, and `:59-60` is a placeholder comment. Task 8 now hoists them and replaces the placeholder. `auth.Require(tokens *TokenIssuer, sessions SessionStore)` signature confirmed at `internal/auth/middleware.go:20`.
12. `auth.ContextUserID = "user_id"` and `auth.UserID(c)` confirmed (`middleware.go:11,43`); the handler test now sets the constant instead of the literal and the "if the auth slice renamed it" hedge is gone. Task 6.
13. `store.Migrate(ctx, pg.Migrator(), store.MigrationsFS)` returns `(applied, err)` and `pg.Migrator()` is the advisory-lock `PgMigrator` (`postgres.go:43`, `migrations.go`); `store.NewRedis` returns `*store.Redis{Client}` with `Close() error`. The integration test already matched; noted in *Depends on*.
14. Go version corrected to 1.25 (`backend/go.mod:3`). Header.
15. Verification gained greps that fail if any pre-reconciliation field name (`total_seconds`, `target_met`, `newly_met`, `exercises`, `seconds`) reappears on the wire, and that the §6.2 names are present.

**Open questions for the human (recorded, not blocking re-approval)**

- `user_answers`: accept-and-drop (this plan) vs persist. Persisting needs a table that §3.2 does not have.
- `pet_health` / `streak_count` placeholder from `NopPet` (100, 0) until the pet slice: acceptable, or should the fields be omitted/`null` until then? The spec types them as integers, so the plan keeps integers.
- `Pet.State` failure → log and report (0, 0) with a 200, rather than a 500 that would invite a double-counting retry. Confirm.
- §6.2 progress-time pet bump vs §8 cron "Success Logic": which one the pet slice implements (plan assumes progress-time).
- `title` / `duration_minutes` sourced from `content_json` keys of the same name — the onboarding/AI generator must agree on those key names.

The idea file (`## Evaluation`, *Reconciliation note*) records that this plan's DTOs supersede the field names in its *Expected output*.
