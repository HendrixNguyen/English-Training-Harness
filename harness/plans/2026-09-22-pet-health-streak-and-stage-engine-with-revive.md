---
idea: harness/ideas/2026-09-22-run-02/pet-health-streak-and-stage-engine-with-revive.md
status: draft
priority: high
merged: false
order: 4
---
# Pet: health, streak and stage engine with revive — Plan

**Idea:** `harness/ideas/2026-09-22-run-02/pet-health-streak-and-stage-engine-with-revive.md`
**Goal:** Add `backend/internal/pet` — `GET /api/v1/pet/status` and `POST /api/v1/pet/revive` behind `auth.Require()` with the **backend spec §6.3 DTOs field for field**, the §8 plant arithmetic split so the +20/streak++ fires once from the quests hook and the hourly cron applies only the −30 miss, idempotent `pet_states` row creation, and a Redis-tracked 15-minute revive challenge.

**Spec precedence:** the *Backend Technical Specification* wins for its layer (AGENTS.md → *Reading the spec*): §6.3 supplies the wire shapes, §8 the arithmetic (−30 on a miss, +20 on success, `wilted` at 0, revive resets to 50), §3.2 the table. The 1st-thinking doc supplies the flow (§5.2 steps 4–5) and the "15-minute revival challenge" wording (§7). Where the idea's *Expected output* said −20 and revive → 20, this plan follows §8/§6.3 (−30, → 50); the idea's `## Evaluation` records that.

**The §6.2-vs-§8 split, decided:** §6.2 says `POST /quests/progress` "increases plant health (+20%), and increments streak"; §8's cron carries the same "Success Logic". Doing both would double-bump. **Decision: the success arithmetic runs exactly once, at progress time, from `quests.Pet.OnTargetMet` (quests fires it once per user per local day); the hourly cron applies only §8's inactivity logic** (`Health = Max(0, Health - 30)`, `wilted` at 0, and streak → 0 because a streak with a missed day in it is not a streak — §8 is silent on the streak, recorded below).

**Architecture:** `engine.go` is pure arithmetic over a `State` value (`StageFor`, `ApplyTargetMet`, `ApplyMiss`, `ApplyRevive`). `Service` composes four interfaces: `Repo` (the `pet_states` row plus the two `users.timezone` reads the sweep needs), `ChallengeStore` (the `pet:revive:{user_id}` Hash), `StudyCounter` (read-only view of the daily counter — satisfied by `quests.RedisCounter`, so pet never touches quests' key or tables), and an injected clock. `QuestHook` adapts `*Service` to `quests.Pet`. `RunHourly` is the in-process cron (§2.1) that calls `Service.Sweep` at every `:00` UTC.

**Interface this slice implements (from the quests plan, Task 3 `pet.go`; verify against the merged file before starting):**
```go
type PetState struct{ Health, Streak int }
type Pet interface {
	OnTargetMet(ctx context.Context, userID, localDate string) error
	State(ctx context.Context, userID string) (PetState, error)
}
```
Also reused from `quests`: `Location(name string) *time.Location`, `LocalDate(now time.Time, loc *time.Location) string`, `TargetSeconds = 1800`, `RedisCounter.Total(ctx, userID, localDate string) (int64, error)`. `pet` imports `quests`; `quests` never imports `pet`.

**Tech stack:** Go 1.25 (`backend/go.mod`), Gin, `pgx/v5`, `go-redis/v9`. No new dependencies. No migration: every column used exists in `0001_init.up.sql`.

**Depends on:** `store` (1) and `auth` (2) merged on `main`; **`quests` (3) merged** — this plan edits `cmd/api/main.go` as the quests plan's Task 8 leaves it (`quests.NopPet{}` registered, `guarded := v1.Group("", auth.Require(tokens, sessions))`). If quests is not yet merged when this plan is executed, stop and say so.

**Run every command from `backend/`** unless the step says otherwise. `rg` is not installed — use `grep -n`.

## File structure

| Path | Responsibility |
| --- | --- |
| `backend/internal/store/keys.go` `keys_test.go` | add `PetReviveKey`, `PetReviveTTL` |
| `backend/internal/pet/engine.go` `engine_test.go` | `State`, stage thresholds, §8 arithmetic — pure |
| `backend/internal/pet/repo.go` | `Repo` interface + `PgRepo` (`Ensure`, `Get`, `Save`, `Timezone`, `Timezones`, `SweepCandidates`) |
| `backend/internal/pet/revive.go` | `Challenge`, `ChallengeStore` + `RedisChallengeStore`, `StudyCounter` |
| `backend/internal/pet/service.go` `service_test.go` | `Ensure`, `OnTargetMet`, `Revive`, `Sweep` |
| `backend/internal/pet/questhook.go` | `QuestHook` — the `quests.Pet` implementation |
| `backend/internal/pet/cron.go` `cron_test.go` | `NextTopOfHour`, `RunHourly` |
| `backend/internal/pet/handler.go` `handler_test.go` | the two §6.3 routes |
| `backend/internal/pet/fakes_test.go` | in-memory `Repo`, `ChallengeStore`, `StudyCounter` |
| `backend/internal/pet/integration_test.go` | `TestIntegrationEnsureCreatesExactlyOnePetRow` (gated on `TEST_DATABASE_URL`; CI counts it) |
| `backend/cmd/api/main.go` | register `QuestHook`, mount routes, start the cron |
| `harness/CODEMAP.md` | `pet` paragraph; `store` bullet gains the new key |

---

## Tasks

### Task 1: The revive key in `store`

**Files:**
- Modify: `backend/internal/store/keys.go`
- Test: `backend/internal/store/keys_test.go`

§4 has no key for the active revive challenge. This adds `pet:revive:{user_id}` (Hash, 24 h) beside the §4 builders so no package ever builds the string itself.

- [ ] **Step 1: Extend the failing tests**

In `keys_test.go`, add a row to the `tests` slice of `TestKeyBuilders`:
```go
		{"revive", PetReviveKey(uid), "pet:revive:3f0d1a7e-0000-4000-8000-000000000001"},
```
and a row to `TestTTLs`:
```go
		{"PetReviveTTL", PetReviveTTL, 24 * time.Hour},
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/store/... -run 'KeyBuilders|TTLs'
```
Expected: build failure, `undefined: PetReviveKey`.

- [ ] **Step 3: Implement**

In `keys.go`, add to the TTL const block:
```go
	// PetReviveTTL bounds the pet:revive hash. Not in spec §4 — chosen by the
	// pet slice: the challenge is bound to the local day it started, and 24h is
	// shorter than DailyAccumulatedTTL, whose counter the pass check reads.
	PetReviveTTL = 24 * time.Hour
```
and after `AIRateLimitKey`:
```go
// PetReviveKey is pet:revive:{user_id} — the active 15-minute revival
// challenge (Hash: started_at, local_date, start_seconds; TTL PetReviveTTL).
// Not in spec §4; added by the pet slice and documented in CODEMAP.
func PetReviveKey(userID string) string { return fmt.Sprintf("pet:revive:%s", userID) }
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/store/... -run 'KeyBuilders|TTLs' -v
```
Expected: `--- PASS` for both.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/store/keys.go backend/internal/store/keys_test.go && git commit -m "store: pet:revive:{user_id} key builder and 24h TTL"
```

---

### Task 2: Pure plant arithmetic

**Files:**
- Create: `backend/internal/pet/engine.go`
- Test: `backend/internal/pet/engine_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/pet/engine_test.go`:
```go
package pet

import (
	"testing"
	"time"
)

func TestStageForIsWiltedAtZeroHealthOtherwiseByStreak(t *testing.T) {
	tests := []struct {
		health, streak int
		want           string
	}{
		{0, 0, StageWilted},
		{0, 30, StageWilted}, // health wins over streak
		{100, 0, StageSprout},
		{100, 2, StageSprout},
		{100, 3, StageSapling},
		{100, 6, StageSapling},
		{100, 7, StageFlowering},
		{100, 13, StageFlowering},
		{100, 14, StageFruitful},
		{10, 100, StageFruitful},
	}
	for _, tt := range tests {
		if got := StageFor(tt.health, tt.streak); got != tt.want {
			t.Errorf("StageFor(%d, %d) = %q, want %q", tt.health, tt.streak, got, tt.want)
		}
	}
}

func TestApplyTargetMetAddsTwentyCapsAtHundredAndStampsPractice(t *testing.T) {
	now := time.Date(2026, time.September, 22, 20, 15, 0, 0, time.UTC)

	s := ApplyTargetMet(State{HealthPoints: 80, CurrentStreak: 4, Stage: StageSapling}, now)
	if s.HealthPoints != 100 || s.CurrentStreak != 5 {
		t.Errorf("state = %+v, want health 100 streak 5", s)
	}
	if s.LastPracticedAt == nil || !s.LastPracticedAt.Equal(now) || !s.UpdatedAt.Equal(now) {
		t.Errorf("timestamps = %v / %v, want both %v", s.LastPracticedAt, s.UpdatedAt, now)
	}

	capped := ApplyTargetMet(State{HealthPoints: 95, CurrentStreak: 13}, now)
	if capped.HealthPoints != 100 {
		t.Errorf("health = %d, want capped at 100", capped.HealthPoints)
	}
	if capped.Stage != StageFruitful {
		t.Errorf("stage = %q, want fruitful at streak 14", capped.Stage)
	}

	revived := ApplyTargetMet(State{HealthPoints: 0, Stage: StageWilted}, now)
	if revived.HealthPoints != 20 || revived.Stage != StageSprout || revived.CurrentStreak != 1 {
		t.Errorf("a wilted plant that meets the target = %+v, want 20/sprout/1", revived)
	}
}

func TestApplyMissSubtractsThirtyFloorsAtZeroAndWilts(t *testing.T) {
	now := time.Date(2026, time.September, 23, 0, 0, 0, 0, time.UTC)

	s := State{HealthPoints: 100, CurrentStreak: 9, Stage: StageFlowering}
	for i, want := range []int{70, 40, 10, 0} {
		s = ApplyMiss(s, now)
		if s.HealthPoints != want {
			t.Fatalf("miss %d: health = %d, want %d", i+1, s.HealthPoints, want)
		}
		if s.CurrentStreak != 0 {
			t.Errorf("miss %d: streak = %d, want 0", i+1, s.CurrentStreak)
		}
	}
	if s.Stage != StageWilted {
		t.Errorf("stage after four misses = %q, want wilted", s.Stage)
	}
	if !s.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", s.UpdatedAt, now)
	}

	again := ApplyMiss(s, now)
	if again.HealthPoints != 0 {
		t.Errorf("health below zero: %d", again.HealthPoints)
	}
}

func TestApplyReviveResetsToFiftySproutZeroStreak(t *testing.T) {
	now := time.Date(2026, time.September, 23, 9, 0, 0, 0, time.UTC)
	s := ApplyRevive(State{HealthPoints: 0, Stage: StageWilted, CurrentStreak: 0}, now)
	if s.HealthPoints != ReviveHealth || s.HealthPoints != 50 || s.Stage != StageSprout || s.CurrentStreak != 0 {
		t.Errorf("revived = %+v, want 50/sprout/0 (backend spec §6.3)", s)
	}
	if !s.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", s.UpdatedAt, now)
	}
}

func TestSpec8Constants(t *testing.T) {
	if TargetMetHealthBonus != 20 || MissPenalty != 30 || MaxHealth != 100 || ReviveSeconds != 900 {
		t.Errorf("constants = (+%d, -%d, max %d, revive %ds), want (+20, -30, 100, 900)", TargetMetHealthBonus, MissPenalty, MaxHealth, ReviveSeconds)
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
mkdir -p internal/pet && go test ./internal/pet/...
```
Expected: build failure, `undefined: StageFor`.

- [ ] **Step 3: Implement**

`backend/internal/pet/engine.go`:
```go
// Package pet is the virtual-plant engine: health, streak and stage
// (1st-thinking §5.2 steps 4-5; backend spec §6.3 for the wire DTOs, §8 for
// the arithmetic). Success (+20, streak+1) is applied once, at progress time,
// through the quests hook; the hourly cron applies only the miss penalty.
package pet

import "time"

// Arithmetic from backend spec §8 and §6.3.
const (
	MaxHealth            = 100
	TargetMetHealthBonus = 20  // §8 success logic: Health = Min(100, Health + 20)
	MissPenalty          = 30  // §8 inactivity logic: Health = Max(0, Health - 30)
	ReviveHealth         = 50  // §6.3: "Resets health to 50% upon passing"
	ReviveSeconds        = 900 // §7: "15-minute revival challenge"
)

// Stages are the §3.2 pet_stage enum values. StageSeed is never produced —
// the DDL default is 'sprout' (see the reconcile-pet-states-stage bug).
const (
	StageSeed      = "seed"
	StageSprout    = "sprout"
	StageSapling   = "sapling"
	StageFlowering = "flowering"
	StageFruitful  = "fruitful"
	StageWilted    = "wilted"
)

// State is the pet_states row (§3.2) minus its ids.
type State struct {
	PlantName       string
	HealthPoints    int
	Stage           string
	CurrentStreak   int
	LastPracticedAt *time.Time
	UpdatedAt       time.Time
}

// StageFor derives the stage from health and streak. The spec gives no
// thresholds; these are the pet slice's decision, recorded in CODEMAP:
// wilted at 0 health, otherwise sprout (0-2), sapling (3-6), flowering (7-13),
// fruitful (14+) by streak.
func StageFor(health, streak int) string {
	switch {
	case health <= 0:
		return StageWilted
	case streak >= 14:
		return StageFruitful
	case streak >= 7:
		return StageFlowering
	case streak >= 3:
		return StageSapling
	default:
		return StageSprout
	}
}

// ApplyTargetMet is §8's success logic, run once per local day from the
// quests hook: +20 capped at 100, streak+1, last_practiced_at = now.
func ApplyTargetMet(s State, now time.Time) State {
	s.HealthPoints = min(MaxHealth, s.HealthPoints+TargetMetHealthBonus)
	s.CurrentStreak++
	s.Stage = StageFor(s.HealthPoints, s.CurrentStreak)
	t := now
	s.LastPracticedAt = &t
	s.UpdatedAt = now
	return s
}

// ApplyMiss is §8's inactivity logic, run by the hourly cron for a user whose
// previous local day stayed under 1800s: -30 floored at 0, wilted at 0. §8 is
// silent on the streak; a streak with a missed day in it is not a streak, so
// it resets.
func ApplyMiss(s State, now time.Time) State {
	s.HealthPoints = max(0, s.HealthPoints-MissPenalty)
	s.CurrentStreak = 0
	s.Stage = StageFor(s.HealthPoints, s.CurrentStreak)
	s.UpdatedAt = now
	return s
}

// ApplyRevive is the §6.3 pass: health 50, sprout, streak 0.
func ApplyRevive(s State, now time.Time) State {
	s.HealthPoints = ReviveHealth
	s.CurrentStreak = 0
	s.Stage = StageFor(s.HealthPoints, s.CurrentStreak)
	s.UpdatedAt = now
	return s
}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/pet/... -v
```
Expected: five `--- PASS` lines.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/pet && git commit -m "pet: pure §8 arithmetic — stage thresholds, target-met bonus, miss penalty, revive"
```

---

### Task 3: Repository, challenge store and the study-counter view

**Files:**
- Create: `backend/internal/pet/repo.go`
- Create: `backend/internal/pet/revive.go`

Interfaces plus their real implementations. No tests of their own — the fakes in Task 4 and the
integration test in Task 8 exercise them. Commit together so Task 4 compiles.

- [ ] **Step 1: Write `repo.go`**

```go
package pet

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoPet means Get ran for a user with no pet_states row. Service.Ensure
// always inserts first, so callers only see this on a deleted user.
var ErrNoPet = errors.New("pet: no pet_states row")

// Candidate is one pet the hourly sweep may decay: the row plus the timezone
// that decides which local day just ended.
type Candidate struct {
	UserID   string
	Timezone string
	State    State
}

// Repo is the Postgres side. Ensure/Get/Save touch pet_states (this package's
// table). Timezone/Timezones/SweepCandidates read users.timezone — the shared
// root table, read the same way quests reads it for day_number.
type Repo interface {
	// Ensure creates the 1:1 row idempotently: INSERT ... ON CONFLICT DO NOTHING.
	Ensure(ctx context.Context, userID string) error
	Get(ctx context.Context, userID string) (State, error)
	// Save writes health, stage, streak, last_practiced_at and updated_at.
	Save(ctx context.Context, userID string, s State) error
	Timezone(ctx context.Context, userID string) (string, error)
	// Timezones lists the distinct users.timezone values of users who have a pet.
	Timezones(ctx context.Context) ([]string, error)
	// SweepCandidates returns every pet whose user is in one of the timezones.
	SweepCandidates(ctx context.Context, timezones []string) ([]Candidate, error)
}

const (
	ensureSQL = `INSERT INTO pet_states (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`

	// Every pet_states column except user_id is nullable in §3.2 (DEFAULT
	// without NOT NULL), so COALESCE to the DDL defaults.
	stateColumns = `COALESCE(p.plant_name, 'My Green Buddy'), COALESCE(p.health_points, 100), COALESCE(p.stage::text, 'sprout'),
	COALESCE(p.current_streak, 0), p.last_practiced_at, COALESCE(p.updated_at, CURRENT_TIMESTAMP)`

	getSQL = `SELECT ` + stateColumns + ` FROM pet_states p WHERE p.user_id = $1`

	saveSQL = `
UPDATE pet_states
SET health_points = $2, stage = $3::pet_stage, current_streak = $4, last_practiced_at = $5, updated_at = $6
WHERE user_id = $1`

	timezoneSQL = `SELECT COALESCE(timezone, 'UTC') FROM users WHERE id = $1`

	timezonesSQL = `
SELECT DISTINCT COALESCE(u.timezone, 'UTC')
FROM users u JOIN pet_states p ON p.user_id = u.id`

	candidatesSQL = `
SELECT p.user_id, COALESCE(u.timezone, 'UTC'), ` + stateColumns + `
FROM pet_states p JOIN users u ON u.id = p.user_id
WHERE COALESCE(u.timezone, 'UTC') = ANY($1)`
)

// PgRepo is the real Repo.
type PgRepo struct{ Pool *pgxpool.Pool }

// NewPgRepo builds a repo over an existing pool.
func NewPgRepo(pool *pgxpool.Pool) *PgRepo { return &PgRepo{Pool: pool} }

func (r *PgRepo) Ensure(ctx context.Context, userID string) error {
	if _, err := r.Pool.Exec(ctx, ensureSQL, userID); err != nil {
		return fmt.Errorf("pet: ensuring pet_states row: %w", err)
	}
	return nil
}

func scanState(row pgx.Row, dst ...any) (State, error) {
	var s State
	var last *time.Time
	targets := append(dst, &s.PlantName, &s.HealthPoints, &s.Stage, &s.CurrentStreak, &last, &s.UpdatedAt)
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

func (r *PgRepo) Get(ctx context.Context, userID string) (State, error) {
	s, err := scanState(r.Pool.QueryRow(ctx, getSQL, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, ErrNoPet
	}
	if err != nil {
		return State{}, fmt.Errorf("pet: reading pet_states: %w", err)
	}
	return s, nil
}

func (r *PgRepo) Save(ctx context.Context, userID string, s State) error {
	tag, err := r.Pool.Exec(ctx, saveSQL, userID, s.HealthPoints, s.Stage, s.CurrentStreak, s.LastPracticedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("pet: saving pet_states: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNoPet
	}
	return nil
}

func (r *PgRepo) Timezone(ctx context.Context, userID string) (string, error) {
	var tz string
	if err := r.Pool.QueryRow(ctx, timezoneSQL, userID).Scan(&tz); err != nil {
		return "", fmt.Errorf("pet: reading timezone: %w", err)
	}
	return tz, nil
}

func (r *PgRepo) Timezones(ctx context.Context) ([]string, error) {
	rows, err := r.Pool.Query(ctx, timezonesSQL)
	if err != nil {
		return nil, fmt.Errorf("pet: listing timezones: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var tz string
		if err := rows.Scan(&tz); err != nil {
			return nil, fmt.Errorf("pet: scanning timezone: %w", err)
		}
		out = append(out, tz)
	}
	return out, rows.Err()
}

func (r *PgRepo) SweepCandidates(ctx context.Context, timezones []string) ([]Candidate, error) {
	if len(timezones) == 0 {
		return nil, nil
	}
	rows, err := r.Pool.Query(ctx, candidatesSQL, timezones)
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

var _ Repo = (*PgRepo)(nil)
```

- [ ] **Step 2: Write `revive.go`**

```go
package pet

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// Challenge is the active revival attempt: the user must record ReviveSeconds
// of study (via POST /quests/progress) on LocalDate, counted from StartSeconds
// — the daily counter's value when the challenge began.
type Challenge struct {
	StartedAt    time.Time
	LocalDate    string
	StartSeconds int64
}

// ChallengeStore keeps the one active challenge per user under
// store.PetReviveKey with store.PetReviveTTL.
type ChallengeStore interface {
	Start(ctx context.Context, userID string, c Challenge) error
	// Get returns ok=false when there is no active challenge.
	Get(ctx context.Context, userID string) (c Challenge, ok bool, err error)
	Clear(ctx context.Context, userID string) error
}

// StudyCounter is pet's read-only view of the daily study counter. It is
// satisfied by *quests.RedisCounter (Total), so pet never builds quests' key
// or reads its tables — packages talk via interfaces (CODEMAP).
type StudyCounter interface {
	Total(ctx context.Context, userID, localDate string) (int64, error)
}

// RedisChallengeStore is the real ChallengeStore: a Hash with three fields.
type RedisChallengeStore struct{ Client *redis.Client }

// NewRedisChallengeStore builds a store over an existing client.
func NewRedisChallengeStore(r *store.Redis) *RedisChallengeStore {
	return &RedisChallengeStore{Client: r.Client}
}

func (s *RedisChallengeStore) Start(ctx context.Context, userID string, c Challenge) error {
	key := store.PetReviveKey(userID)
	pipe := s.Client.TxPipeline()
	pipe.HSet(ctx, key,
		"started_at", c.StartedAt.UTC().Format(time.RFC3339),
		"local_date", c.LocalDate,
		"start_seconds", strconv.FormatInt(c.StartSeconds, 10),
	)
	pipe.Expire(ctx, key, store.PetReviveTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("pet: starting revive challenge: %w", err)
	}
	return nil
}

func (s *RedisChallengeStore) Get(ctx context.Context, userID string) (Challenge, bool, error) {
	m, err := s.Client.HGetAll(ctx, store.PetReviveKey(userID)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return Challenge{}, false, fmt.Errorf("pet: reading revive challenge: %w", err)
	}
	if len(m) == 0 {
		return Challenge{}, false, nil
	}
	started, err := time.Parse(time.RFC3339, m["started_at"])
	if err != nil {
		return Challenge{}, false, fmt.Errorf("pet: malformed started_at %q: %w", m["started_at"], err)
	}
	start, err := strconv.ParseInt(m["start_seconds"], 10, 64)
	if err != nil {
		return Challenge{}, false, fmt.Errorf("pet: malformed start_seconds %q: %w", m["start_seconds"], err)
	}
	return Challenge{StartedAt: started, LocalDate: m["local_date"], StartSeconds: start}, true, nil
}

func (s *RedisChallengeStore) Clear(ctx context.Context, userID string) error {
	if err := s.Client.Del(ctx, store.PetReviveKey(userID)).Err(); err != nil {
		return fmt.Errorf("pet: clearing revive challenge: %w", err)
	}
	return nil
}

var _ ChallengeStore = (*RedisChallengeStore)(nil)
```

- [ ] **Step 3: Confirm it compiles**

```sh
go build ./... && go vet ./internal/pet/...
```
Expected: no output.

- [ ] **Step 4: Commit**

```sh
cd .. && git add backend/internal/pet && git commit -m "pet: pet_states repository, Redis revive challenge store, StudyCounter view"
```

---

### Task 4: Test fakes

**Files:**
- Create: `backend/internal/pet/fakes_test.go`

- [ ] **Step 1: Write the fakes**

```go
package pet

import (
	"context"
	"errors"
	"sort"
	"time"
)

type fakeRepo struct {
	states    map[string]State  // by userID; present == row exists
	timezones map[string]string // by userID; missing == "UTC"
	ensured   int
	saved     int
	ensureErr error
	saveErr   error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{states: map[string]State{}, timezones: map[string]string{}}
}

// defaultState mirrors the §3.2 column defaults a fresh INSERT produces.
func defaultState(now time.Time) State {
	return State{PlantName: "My Green Buddy", HealthPoints: 100, Stage: StageSprout, UpdatedAt: now}
}

func (f *fakeRepo) Ensure(_ context.Context, userID string) error {
	if f.ensureErr != nil {
		return f.ensureErr
	}
	f.ensured++
	if _, ok := f.states[userID]; !ok {
		f.states[userID] = defaultState(time.Time{})
	}
	return nil
}

func (f *fakeRepo) Get(_ context.Context, userID string) (State, error) {
	s, ok := f.states[userID]
	if !ok {
		return State{}, ErrNoPet
	}
	return s, nil
}

func (f *fakeRepo) Save(_ context.Context, userID string, s State) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	if _, ok := f.states[userID]; !ok {
		return ErrNoPet
	}
	f.saved++
	s.PlantName = f.states[userID].PlantName
	f.states[userID] = s
	return nil
}

func (f *fakeRepo) Timezone(_ context.Context, userID string) (string, error) {
	if tz, ok := f.timezones[userID]; ok {
		return tz, nil
	}
	return "UTC", nil
}

func (f *fakeRepo) Timezones(context.Context) ([]string, error) {
	seen := map[string]bool{}
	for userID := range f.states {
		tz, _ := f.Timezone(context.Background(), userID)
		seen[tz] = true
	}
	out := make([]string, 0, len(seen))
	for tz := range seen {
		out = append(out, tz)
	}
	sort.Strings(out)
	return out, nil
}

func (f *fakeRepo) SweepCandidates(_ context.Context, timezones []string) ([]Candidate, error) {
	want := map[string]bool{}
	for _, tz := range timezones {
		want[tz] = true
	}
	var out []Candidate
	for userID, s := range f.states {
		tz, _ := f.Timezone(context.Background(), userID)
		if want[tz] {
			out = append(out, Candidate{UserID: userID, Timezone: tz, State: s})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UserID < out[j].UserID })
	return out, nil
}

type fakeChallenges struct {
	items   map[string]Challenge
	started int
	cleared int
}

func newFakeChallenges() *fakeChallenges { return &fakeChallenges{items: map[string]Challenge{}} }

func (f *fakeChallenges) Start(_ context.Context, userID string, c Challenge) error {
	f.started++
	f.items[userID] = c
	return nil
}

func (f *fakeChallenges) Get(_ context.Context, userID string) (Challenge, bool, error) {
	c, ok := f.items[userID]
	return c, ok, nil
}

func (f *fakeChallenges) Clear(_ context.Context, userID string) error {
	f.cleared++
	delete(f.items, userID)
	return nil
}

type fakeStudy struct {
	totals map[string]int64 // keyed by userID|localDate
	err    error
}

func newFakeStudy() *fakeStudy { return &fakeStudy{totals: map[string]int64{}} }

func (f *fakeStudy) set(userID, localDate string, seconds int64) {
	f.totals[userID+"|"+localDate] = seconds
}

func (f *fakeStudy) Total(_ context.Context, userID, localDate string) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.totals[userID+"|"+localDate], nil
}

var errBoom = errors.New("boom")

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }
```

- [ ] **Step 2: Confirm the package still vets**

```sh
go vet ./internal/pet/...
```
Expected: no output.

- [ ] **Step 3: Commit**

```sh
cd .. && git add backend/internal/pet/fakes_test.go && git commit -m "pet: in-memory fakes for Repo, ChallengeStore and StudyCounter"
```

---

### Task 5: `Service` — ensure, target-met, revive, sweep — and the `quests.Pet` adapter

**Files:**
- Create: `backend/internal/pet/service.go`
- Create: `backend/internal/pet/questhook.go`
- Test: `backend/internal/pet/service_test.go`

- [ ] **Step 1: Write the failing tests**

`backend/internal/pet/service_test.go`:
```go
package pet

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/quests"
)

type harness struct {
	svc        *Service
	repo       *fakeRepo
	challenges *fakeChallenges
	study      *fakeStudy
}

func newHarness(now time.Time) *harness {
	h := &harness{repo: newFakeRepo(), challenges: newFakeChallenges(), study: newFakeStudy()}
	h.svc = NewService(h.repo, h.challenges, h.study, fixedClock(now))
	return h
}

var (
	ctx     = context.Background()
	sept22  = time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	midnite = time.Date(2026, time.September, 23, 0, 0, 0, 0, time.UTC) // 00:00 UTC = a :00 sweep
)

func TestEnsureCreatesTheRowOnceAndReturnsTheDefaults(t *testing.T) {
	h := newHarness(sept22)

	first, err := h.svc.Ensure(ctx, "u1")
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if first.PlantName != "My Green Buddy" || first.HealthPoints != 100 || first.Stage != StageSprout || first.CurrentStreak != 0 || first.LastPracticedAt != nil {
		t.Errorf("fresh pet = %+v, want the §3.2 defaults", first)
	}
	if _, err := h.svc.Ensure(ctx, "u1"); err != nil {
		t.Fatalf("second Ensure: %v", err)
	}
	if h.repo.ensured != 2 || len(h.repo.states) != 1 {
		t.Errorf("ensured %d times into %d rows, want 2 calls, 1 row", h.repo.ensured, len(h.repo.states))
	}
}

func TestOnTargetMetAppliesSpec8SuccessOnceAndPersists(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{PlantName: "Fern", HealthPoints: 80, CurrentStreak: 4, Stage: StageSapling}

	if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-22"); err != nil {
		t.Fatalf("OnTargetMet: %v", err)
	}
	got := h.repo.states["u1"]
	if got.HealthPoints != 100 || got.CurrentStreak != 5 || got.Stage != StageSapling {
		t.Errorf("state = %+v, want 100/5/sapling", got)
	}
	if got.LastPracticedAt == nil || !got.LastPracticedAt.Equal(sept22) {
		t.Errorf("LastPracticedAt = %v, want %v", got.LastPracticedAt, sept22)
	}
	if got.PlantName != "Fern" {
		t.Errorf("PlantName = %q, want untouched", got.PlantName)
	}
	if h.repo.saved != 1 {
		t.Errorf("saved %d times, want 1", h.repo.saved)
	}
}

func TestOnTargetMetForAUserWithoutARowCreatesIt(t *testing.T) {
	h := newHarness(sept22)
	if err := h.svc.OnTargetMet(ctx, "new", "2026-09-22"); err != nil {
		t.Fatalf("OnTargetMet: %v", err)
	}
	if got := h.repo.states["new"]; got.HealthPoints != 100 || got.CurrentStreak != 1 {
		t.Errorf("state = %+v, want the defaults bumped: 100 (capped) / streak 1", got)
	}
}

func TestQuestHookSatisfiesQuestsPet(t *testing.T) {
	var _ quests.Pet = (*QuestHook)(nil)

	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 40, CurrentStreak: 2}
	hook := NewQuestHook(h.svc)

	st, err := hook.State(ctx, "u1")
	if err != nil {
		t.Fatalf("State: %v", err)
	}
	if st != (quests.PetState{Health: 40, Streak: 2}) {
		t.Errorf("State = %+v, want {40 2}", st)
	}
	if err := hook.OnTargetMet(ctx, "u1", "2026-09-22"); err != nil {
		t.Fatalf("OnTargetMet: %v", err)
	}
	st, _ = hook.State(ctx, "u1")
	if st != (quests.PetState{Health: 60, Streak: 3}) {
		t.Errorf("State after hook = %+v, want {60 3}", st)
	}
}

func TestReviveIsRejectedWhileHealthIsAboveZero(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 10, Stage: StageSprout}

	if _, err := h.svc.Revive(ctx, "u1"); !errors.Is(err, ErrNotWilted) {
		t.Fatalf("err = %v, want ErrNotWilted", err)
	}
	if h.challenges.started != 0 {
		t.Error("a rejected revive started a challenge")
	}
}

func TestReviveStartsAChallengeAtTheCurrentCounterValue(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}
	h.study.set("u1", "2026-09-22", 300) // already studied 5 min today

	out, err := h.svc.Revive(ctx, "u1")
	if err != nil {
		t.Fatalf("Revive: %v", err)
	}
	if out.Passed {
		t.Error("Passed = true on the call that starts the challenge")
	}
	if out.State.HealthPoints != 0 || out.State.Stage != StageWilted {
		t.Errorf("state = %+v, want still wilted", out.State)
	}
	c, ok := h.challenges.items["u1"]
	if !ok || c.LocalDate != "2026-09-22" || c.StartSeconds != 300 || !c.StartedAt.Equal(sept22) {
		t.Errorf("challenge = %+v ok=%t, want {2026-09-22, 300, %v}", c, ok, sept22)
	}
}

func TestReviveUsesTheUsersTimezoneForTheLocalDate(t *testing.T) {
	// 18:30Z is already the 23rd in Ho Chi Minh City.
	h := newHarness(time.Date(2026, time.September, 22, 18, 30, 0, 0, time.UTC))
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}
	h.repo.timezones["u1"] = "Asia/Ho_Chi_Minh"

	if _, err := h.svc.Revive(ctx, "u1"); err != nil {
		t.Fatalf("Revive: %v", err)
	}
	if c := h.challenges.items["u1"]; c.LocalDate != "2026-09-23" {
		t.Errorf("LocalDate = %q, want 2026-09-23", c.LocalDate)
	}
}

func TestRevivePassesOnceNineHundredSecondsWereStudiedSinceTheStart(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted, CurrentStreak: 0}
	h.study.set("u1", "2026-09-22", 300)
	if _, err := h.svc.Revive(ctx, "u1"); err != nil {
		t.Fatalf("start: %v", err)
	}

	h.study.set("u1", "2026-09-22", 300+899)
	out, err := h.svc.Revive(ctx, "u1")
	if err != nil {
		t.Fatalf("check at 899s: %v", err)
	}
	if out.Passed || out.State.HealthPoints != 0 {
		t.Errorf("out = %+v at 899s, want not passed", out)
	}
	if h.challenges.started != 1 {
		t.Errorf("challenge restarted (%d starts) while still active on the same day", h.challenges.started)
	}

	h.study.set("u1", "2026-09-22", 300+900)
	out, err = h.svc.Revive(ctx, "u1")
	if err != nil {
		t.Fatalf("check at 900s: %v", err)
	}
	if !out.Passed || out.State.HealthPoints != 50 || out.State.Stage != StageSprout || out.State.CurrentStreak != 0 {
		t.Errorf("out = %+v at 900s, want passed with 50/sprout/0 (§6.3)", out)
	}
	if h.challenges.cleared != 1 || len(h.challenges.items) != 0 {
		t.Error("passing did not clear the challenge")
	}
	if got := h.repo.states["u1"]; got.HealthPoints != 50 {
		t.Errorf("persisted health = %d, want 50", got.HealthPoints)
	}
}

func TestReviveOnANewLocalDayRestartsTheChallenge(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}
	h.challenges.items["u1"] = Challenge{StartedAt: sept22.Add(-20 * time.Hour), LocalDate: "2026-09-21", StartSeconds: 0}
	h.study.set("u1", "2026-09-21", 5000) // yesterday's total is irrelevant now
	h.study.set("u1", "2026-09-22", 120)

	out, err := h.svc.Revive(ctx, "u1")
	if err != nil {
		t.Fatalf("Revive: %v", err)
	}
	if out.Passed {
		t.Error("a stale challenge from yesterday must not pass on today's call")
	}
	if c := h.challenges.items["u1"]; c.LocalDate != "2026-09-22" || c.StartSeconds != 120 {
		t.Errorf("challenge = %+v, want restarted for today at 120s", c)
	}
}

func TestSweepPenalisesOnlyUsersAtLocalMidnightWhoMissedYesterday(t *testing.T) {
	h := newHarness(midnite) // 00:00 UTC: UTC users just hit midnight; Ho Chi Minh (UTC+7) is at 07:00
	h.repo.states["utc-missed"] = State{HealthPoints: 100, CurrentStreak: 9, Stage: StageFlowering, UpdatedAt: midnite.Add(-30 * time.Hour)}
	h.repo.states["utc-met"] = State{HealthPoints: 100, CurrentStreak: 9, Stage: StageFlowering, UpdatedAt: midnite.Add(-3 * time.Hour)}
	h.repo.states["utc-short"] = State{HealthPoints: 40, CurrentStreak: 1, Stage: StageSprout, UpdatedAt: midnite.Add(-30 * time.Hour)}
	h.repo.states["hcm-missed"] = State{HealthPoints: 100, CurrentStreak: 9, Stage: StageFlowering, UpdatedAt: midnite.Add(-30 * time.Hour)}
	h.repo.timezones["hcm-missed"] = "Asia/Ho_Chi_Minh"
	h.study.set("utc-met", "2026-09-22", 1800)
	h.study.set("utc-short", "2026-09-22", 1799)

	n, err := h.svc.Sweep(ctx, midnite)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if n != 2 {
		t.Errorf("penalised %d pets, want 2 (utc-missed, utc-short)", n)
	}
	if got := h.repo.states["utc-missed"]; got.HealthPoints != 70 || got.CurrentStreak != 0 || got.Stage != StageSprout || !got.UpdatedAt.Equal(midnite) {
		t.Errorf("utc-missed = %+v, want 70/0/sprout stamped %v", got, midnite)
	}
	if got := h.repo.states["utc-short"]; got.HealthPoints != 10 || got.Stage != StageSprout {
		t.Errorf("utc-short = %+v, want 10/sprout (1799s is a miss)", got)
	}
	if got := h.repo.states["utc-met"]; got.HealthPoints != 100 || got.CurrentStreak != 9 {
		t.Errorf("utc-met = %+v, want untouched", got)
	}
	if got := h.repo.states["hcm-missed"]; got.HealthPoints != 100 {
		t.Errorf("hcm-missed = %+v, want untouched — it is 07:00 in Ho Chi Minh City", got)
	}
}

func TestSweepAtSeventeenUTCCatchesHoChiMinhMidnight(t *testing.T) {
	at := time.Date(2026, time.September, 22, 17, 0, 0, 0, time.UTC) // 00:00 on the 23rd in UTC+7
	h := newHarness(at)
	h.repo.states["hcm"] = State{HealthPoints: 100, UpdatedAt: at.Add(-30 * time.Hour)}
	h.repo.timezones["hcm"] = "Asia/Ho_Chi_Minh"
	h.repo.states["utc"] = State{HealthPoints: 100, UpdatedAt: at.Add(-30 * time.Hour)}

	if _, err := h.svc.Sweep(ctx, at); err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if got := h.repo.states["hcm"]; got.HealthPoints != 70 {
		t.Errorf("hcm = %+v, want 70 — local day 2026-09-22 ended with no study", got)
	}
	if got := h.repo.states["utc"]; got.HealthPoints != 100 {
		t.Errorf("utc = %+v, want untouched at 17:00 UTC", got)
	}
}

func TestSweepIsIdempotentWithinTheSameLocalDay(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-30 * time.Hour)}

	if _, err := h.svc.Sweep(ctx, midnite); err != nil {
		t.Fatalf("first: %v", err)
	}
	n, err := h.svc.Sweep(ctx, midnite.Add(20*time.Minute)) // a restart re-running the hour
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if n != 0 || h.repo.states["u1"].HealthPoints != 70 {
		t.Errorf("second sweep penalised %d, health = %d; want 0 and 70 — updated_at is already past local midnight", n, h.repo.states["u1"].HealthPoints)
	}
}

func TestSweepWiltsAfterFourMissedDays(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, CurrentStreak: 20, Stage: StageFruitful, UpdatedAt: midnite.Add(-30 * time.Hour)}

	for day := 0; day < 4; day++ {
		at := midnite.AddDate(0, 0, day)
		h.svc.now = fixedClock(at)
		if _, err := h.svc.Sweep(ctx, at); err != nil {
			t.Fatalf("day %d: %v", day, err)
		}
	}
	if got := h.repo.states["u1"]; got.HealthPoints != 0 || got.Stage != StageWilted || got.CurrentStreak != 0 {
		t.Errorf("after four misses = %+v, want 0/wilted/0", got)
	}
}

func TestSweepSkipsAUnreadableCounterAndContinues(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-30 * time.Hour)}
	h.study.err = errBoom

	n, err := h.svc.Sweep(ctx, midnite)
	if err == nil {
		t.Fatal("Sweep returned nil with an unreadable counter; want the error surfaced")
	}
	if n != 0 || h.repo.states["u1"].HealthPoints != 100 {
		t.Errorf("a pet was penalised on an unreadable counter: n=%d health=%d", n, h.repo.states["u1"].HealthPoints)
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/pet/... -run 'Ensure|OnTargetMet|QuestHook|Revive|Sweep'
```
Expected: build failure, `undefined: NewService`.

- [ ] **Step 3: Implement `service.go`**

```go
package pet

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/quests"
)

// ErrNotWilted means POST /pet/revive was called while health_points > 0.
var ErrNotWilted = errors.New("pet: revive requires a wilted plant")

// ReviveResult is what POST /pet/revive reports (backend spec §6.3).
type ReviveResult struct {
	Passed bool
	State  State
}

// Service is the plant engine over its collaborators.
type Service struct {
	repo       Repo
	challenges ChallengeStore
	study      StudyCounter
	now        func() time.Time
}

// NewService wires the collaborators; now is injectable for tests.
func NewService(repo Repo, challenges ChallengeStore, study StudyCounter, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, challenges: challenges, study: study, now: now}
}

// Ensure creates the user's pet_states row if it is missing (idempotent:
// INSERT ... ON CONFLICT (user_id) DO NOTHING) and returns the current state.
// It is what GET /pet/status and onboarding call first, which answers the
// "who creates the row" gap in the spec for the code.
func (s *Service) Ensure(ctx context.Context, userID string) (State, error) {
	if err := s.repo.Ensure(ctx, userID); err != nil {
		return State{}, err
	}
	return s.repo.Get(ctx, userID)
}

// OnTargetMet is §8's success logic, applied once per local day: quests fires
// it on the progress call that crosses 1800s (backend spec §6.2). localDate
// is informational — the row is not keyed by day.
func (s *Service) OnTargetMet(ctx context.Context, userID, localDate string) error {
	st, err := s.Ensure(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.Save(ctx, userID, ApplyTargetMet(st, s.now()))
}

// Revive implements the 15-minute revival challenge behind POST /pet/revive.
//
// Only a wilted plant (health 0) may be revived; otherwise ErrNotWilted (409).
// The first call on a local day starts a challenge, recording the daily
// counter's current value; each later call the same day checks whether
// ReviveSeconds more have been recorded through POST /quests/progress. On pass
// the state becomes 50 / sprout / 0 (§6.3) and the challenge is cleared. A
// challenge left over from an earlier local day is replaced.
func (s *Service) Revive(ctx context.Context, userID string) (ReviveResult, error) {
	st, err := s.Ensure(ctx, userID)
	if err != nil {
		return ReviveResult{}, err
	}
	if st.HealthPoints > 0 {
		return ReviveResult{}, ErrNotWilted
	}

	tz, err := s.repo.Timezone(ctx, userID)
	if err != nil {
		return ReviveResult{}, err
	}
	now := s.now()
	today := quests.LocalDate(now, quests.Location(tz))

	c, ok, err := s.challenges.Get(ctx, userID)
	if err != nil {
		return ReviveResult{}, err
	}
	if !ok || c.LocalDate != today {
		start, err := s.study.Total(ctx, userID, today)
		if err != nil {
			return ReviveResult{}, err
		}
		if err := s.challenges.Start(ctx, userID, Challenge{StartedAt: now, LocalDate: today, StartSeconds: start}); err != nil {
			return ReviveResult{}, err
		}
		return ReviveResult{Passed: false, State: st}, nil
	}

	total, err := s.study.Total(ctx, userID, c.LocalDate)
	if err != nil {
		return ReviveResult{}, err
	}
	if total-c.StartSeconds < ReviveSeconds {
		return ReviveResult{Passed: false, State: st}, nil
	}

	st = ApplyRevive(st, now)
	if err := s.repo.Save(ctx, userID, st); err != nil {
		return ReviveResult{}, err
	}
	if err := s.challenges.Clear(ctx, userID); err != nil {
		// The pass is already persisted; a stale key only expires later.
		log.Printf("pet: clearing revive challenge for %s: %v", userID, err)
	}
	return ReviveResult{Passed: true, State: st}, nil
}

// Sweep is the body of the §8 hourly cron. It runs at :00 UTC; a user "hits
// local midnight" when now in their timezone is in hour 0. For each such pet
// whose previous local day recorded fewer than 1800s it applies ApplyMiss.
//
// Idempotency: a pet whose updated_at is already at or after that local
// midnight has been touched this local day (by an earlier run of this same
// sweep after a restart, or by a target met after midnight) and is skipped.
// Errors on one pet are collected and the rest are still processed; the
// count returned is the number of pets penalised.
func (s *Service) Sweep(ctx context.Context, now time.Time) (int, error) {
	zones, err := s.repo.Timezones(ctx)
	if err != nil {
		return 0, err
	}
	var atMidnight []string
	for _, tz := range zones {
		if now.In(quests.Location(tz)).Hour() == 0 {
			atMidnight = append(atMidnight, tz)
		}
	}
	if len(atMidnight) == 0 {
		return 0, nil
	}

	cands, err := s.repo.SweepCandidates(ctx, atMidnight)
	if err != nil {
		return 0, err
	}

	penalised := 0
	var errs []error
	for _, c := range cands {
		loc := quests.Location(c.Timezone)
		local := now.In(loc)
		midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
		if !c.State.UpdatedAt.Before(midnight) {
			continue // already handled this local day
		}
		yesterday := midnight.AddDate(0, 0, -1).Format("2006-01-02")
		total, err := s.study.Total(ctx, c.UserID, yesterday)
		if err != nil {
			errs = append(errs, fmt.Errorf("user %s: %w", c.UserID, err))
			continue
		}
		if total >= quests.TargetSeconds {
			continue
		}
		if err := s.repo.Save(ctx, c.UserID, ApplyMiss(c.State, now)); err != nil {
			errs = append(errs, fmt.Errorf("user %s: %w", c.UserID, err))
			continue
		}
		penalised++
	}
	return penalised, errors.Join(errs...)
}
```

- [ ] **Step 4: Implement `questhook.go`**

```go
package pet

import (
	"context"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/quests"
)

// QuestHook adapts *Service to quests.Pet — the hook quests fires once per
// user per local day when the 30-minute target is crossed, and the state read
// that fills pet_health / streak_count on the §6.2 progress response.
// cmd/api/main.go registers it in place of quests.NopPet{}.
type QuestHook struct{ svc *Service }

// NewQuestHook wraps a Service.
func NewQuestHook(svc *Service) *QuestHook { return &QuestHook{svc: svc} }

func (h *QuestHook) OnTargetMet(ctx context.Context, userID, localDate string) error {
	return h.svc.OnTargetMet(ctx, userID, localDate)
}

func (h *QuestHook) State(ctx context.Context, userID string) (quests.PetState, error) {
	st, err := h.svc.Ensure(ctx, userID)
	if err != nil {
		return quests.PetState{}, err
	}
	return quests.PetState{Health: st.HealthPoints, Streak: st.CurrentStreak}, nil
}

var _ quests.Pet = (*QuestHook)(nil)
```

- [ ] **Step 5: Run and confirm it passes**

```sh
go test ./internal/pet/... -v
```
Expected: every `--- PASS`, including the 17:00 UTC Ho Chi Minh case and the idempotency case.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend/internal/pet && git commit -m "pet: Service — ensure, once-only target-met, revive challenge, midnight sweep; quests.Pet adapter"
```

---

### Task 6: The in-process hourly cron

**Files:**
- Create: `backend/internal/pet/cron.go`
- Test: `backend/internal/pet/cron_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/pet/cron_test.go`:
```go
package pet

import (
	"context"
	"testing"
	"time"
)

func TestNextTopOfHour(t *testing.T) {
	tests := []struct {
		now, want time.Time
	}{
		{time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC), time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC)},
		{time.Date(2026, 9, 22, 10, 0, 0, 1, time.UTC), time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC)},
		{time.Date(2026, 9, 22, 10, 59, 59, 0, time.UTC), time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC)},
		{time.Date(2026, 9, 22, 23, 30, 0, 0, time.UTC), time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		if got := NextTopOfHour(tt.now); !got.Equal(tt.want) {
			t.Errorf("NextTopOfHour(%v) = %v, want %v", tt.now, got, tt.want)
		}
	}
}

func TestRunHourlyStopsWhenTheContextIsCancelled(t *testing.T) {
	h := newHarness(sept22)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		RunHourly(ctx, h.svc)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunHourly did not return after cancel")
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/pet/... -run 'TopOfHour|RunHourly'
```
Expected: build failure, `undefined: NextTopOfHour`.

- [ ] **Step 3: Implement**

`backend/internal/pet/cron.go`:
```go
package pet

import (
	"context"
	"log"
	"time"
)

// NextTopOfHour is the next :00 strictly after now (spec §8: the cron runs
// at :00 UTC every hour).
func NextTopOfHour(now time.Time) time.Time {
	return now.Truncate(time.Hour).Add(time.Hour)
}

// RunHourly is the in-process cron worker from spec §2.1. It blocks until ctx
// is cancelled, calling svc.Sweep at every :00 UTC. Sweep errors are logged
// and the loop continues — one bad row must not stop tomorrow's decay.
func RunHourly(ctx context.Context, svc *Service) {
	for {
		now := svc.now()
		timer := time.NewTimer(NextTopOfHour(now).Sub(now))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		at := svc.now()
		n, err := svc.Sweep(ctx, at)
		if err != nil {
			log.Printf("pet: sweep at %s: penalised %d, errors: %v", at.UTC().Format(time.RFC3339), n, err)
			continue
		}
		if n > 0 {
			log.Printf("pet: sweep at %s: penalised %d", at.UTC().Format(time.RFC3339), n)
		}
	}
}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/pet/... -run 'TopOfHour|RunHourly' -v
```
Expected: two `--- PASS`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/pet/cron.go backend/internal/pet/cron_test.go && git commit -m "pet: in-process hourly cron running the midnight sweep"
```

---

### Task 7: The two §6.3 routes

**Files:**
- Create: `backend/internal/pet/handler.go`
- Test: `backend/internal/pet/handler_test.go`

Bodies are copied from the backend spec §6.3 JSON blocks. Error bodies follow the merged
`{"error": "<snake_case_code>"}` convention (`auth/handler.go`, quests plan Task 6).

- [ ] **Step 1: Write the failing tests**

`backend/internal/pet/handler_test.go`:
```go
package pet

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

// newPetRouter stands in for auth.Require() by injecting the user id under
// auth.ContextUserID (Require itself is covered in the auth slice).
func newPetRouter(svc *Service, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1", func(c *gin.Context) { c.Set(auth.ContextUserID, userID); c.Next() })
	g.GET("/pet/status", StatusHandler(svc))
	g.POST("/pet/revive", ReviveHandler(svc))
	return r
}

func do(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestStatusReturnsTheSpec63BodyAndCreatesTheRow(t *testing.T) {
	h := newHarness(sept22)

	w := do(newPetRouter(h.svc, "u1"), http.MethodGet, "/api/v1/pet/status", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var body struct {
		PlantName       string  `json:"plant_name"`
		Stage           string  `json:"stage"`
		HealthPoints    int     `json:"health_points"`
		CurrentStreak   int     `json:"current_streak"`
		LastPracticedAt *string `json:"last_practiced_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding: %v (%s)", err, w.Body.String())
	}
	if body.PlantName != "My Green Buddy" || body.Stage != "sprout" || body.HealthPoints != 100 || body.CurrentStreak != 0 || body.LastPracticedAt != nil {
		t.Errorf("body = %+v, want the fresh-pet defaults with last_practiced_at null", body)
	}
	if !strings.Contains(w.Body.String(), `"last_practiced_at":null`) {
		t.Errorf("last_practiced_at must be present and null on a fresh pet: %s", w.Body.String())
	}
	if h.repo.ensured != 1 {
		t.Errorf("Ensure called %d times, want 1", h.repo.ensured)
	}
}

func TestStatusFormatsLastPracticedAtAsUTCRFC3339(t *testing.T) {
	h := newHarness(sept22)
	at := time.Date(2026, time.September, 21, 20, 15, 0, 0, time.UTC)
	h.repo.states["u1"] = State{PlantName: "My Green Buddy", HealthPoints: 80, Stage: StageSprout, CurrentStreak: 5, LastPracticedAt: &at}

	w := do(newPetRouter(h.svc, "u1"), http.MethodGet, "/api/v1/pet/status", "")
	// Exactly the §6.3 example.
	want := `{"plant_name":"My Green Buddy","stage":"sprout","health_points":80,"current_streak":5,"last_practiced_at":"2026-09-21T20:15:00Z"}`
	if strings.TrimSpace(w.Body.String()) != want {
		t.Errorf("body =\n%s\nwant\n%s", w.Body.String(), want)
	}
}

func TestReviveReturns409WhileNotWilted(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 30, Stage: StageSprout}

	w := do(newPetRouter(h.svc, "u1"), http.MethodPost, "/api/v1/pet/revive", `{"answers":{"q1":"C"}}`)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"error":"pet_not_wilted"`) {
		t.Errorf("status = %d body = %s, want 409 pet_not_wilted", w.Code, w.Body.String())
	}
}

func TestReviveStartsThenPassesWithTheSpec63Body(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}
	r := newPetRouter(h.svc, "u1")

	// §6.3 request body; an empty body is accepted too (the client may have no answers yet).
	w := do(r, http.MethodPost, "/api/v1/pet/revive", `{"answers":{"q1":"C","q2":"B"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("start: status = %d body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"revival_passed":false`) || !strings.Contains(w.Body.String(), `"health_points":0`) {
		t.Errorf("start body = %s, want revival_passed false and health 0", w.Body.String())
	}

	h.study.set("u1", "2026-09-22", 900)
	w = do(r, http.MethodPost, "/api/v1/pet/revive", "")
	if w.Code != http.StatusOK {
		t.Fatalf("pass: status = %d body = %s", w.Code, w.Body.String())
	}
	want := `{"revival_passed":true,"pet_state":{"health_points":50,"stage":"sprout","current_streak":0}}`
	if strings.TrimSpace(w.Body.String()) != want {
		t.Errorf("pass body =\n%s\nwant\n%s", w.Body.String(), want)
	}
}

func TestReviveRejectsMalformedJSON(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}

	w := do(newPetRouter(h.svc, "u1"), http.MethodPost, "/api/v1/pet/revive", `nonsense`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandlersReturn500OnRepoFailure(t *testing.T) {
	h := newHarness(sept22)
	h.repo.ensureErr = errBoom
	r := newPetRouter(h.svc, "u1")

	if w := do(r, http.MethodGet, "/api/v1/pet/status", ""); w.Code != http.StatusInternalServerError {
		t.Errorf("status: %d, want 500", w.Code)
	}
	if w := do(r, http.MethodPost, "/api/v1/pet/revive", ""); w.Code != http.StatusInternalServerError {
		t.Errorf("revive: %d, want 500", w.Code)
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/pet/... -run 'Status|Revive.*Body|409|Malformed|Handlers'
```
Expected: build failure, `undefined: StatusHandler`.

- [ ] **Step 3: Implement**

`backend/internal/pet/handler.go`:
```go
package pet

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// statusResponse is the GET /api/v1/pet/status 200 body — backend spec §6.3,
// field for field. last_practiced_at is null until the first target is met.
type statusResponse struct {
	PlantName       string     `json:"plant_name"`
	Stage           string     `json:"stage"`
	HealthPoints    int        `json:"health_points"`
	CurrentStreak   int        `json:"current_streak"`
	LastPracticedAt *time.Time `json:"last_practiced_at"`
}

func toStatus(s State) statusResponse {
	out := statusResponse{PlantName: s.PlantName, Stage: s.Stage, HealthPoints: s.HealthPoints, CurrentStreak: s.CurrentStreak}
	if s.LastPracticedAt != nil {
		u := s.LastPracticedAt.UTC()
		out.LastPracticedAt = &u
	}
	return out
}

// reviveRequest is the §6.3 POST /api/v1/pet/revive body. answers is accepted
// so a spec-conformant client is never rejected, but neither spec says how it
// is graded, so it is not read — the pass condition is 15 minutes of recorded
// study (see Service.Revive and the plan's open questions).
type reviveRequest struct {
	Answers json.RawMessage `json:"answers"`
}

// reviveResponse is the §6.3 200 body.
type reviveResponse struct {
	RevivalPassed bool `json:"revival_passed"`
	PetState      struct {
		HealthPoints  int    `json:"health_points"`
		Stage         string `json:"stage"`
		CurrentStreak int    `json:"current_streak"`
	} `json:"pet_state"`
}

// StatusHandler serves GET /api/v1/pet/status. Mount behind auth.Require().
func StatusHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		st, err := svc.Ensure(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		c.JSON(http.StatusOK, toStatus(st))
	}
}

// ReviveHandler serves POST /api/v1/pet/revive. Mount behind auth.Require().
func ReviveHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if c.Request.ContentLength != 0 {
			var req reviveRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
				return
			}
		}

		out, err := svc.Revive(c.Request.Context(), userID)
		switch {
		case errors.Is(err, ErrNotWilted):
			c.JSON(http.StatusConflict, gin.H{"error": "pet_not_wilted"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			var resp reviveResponse
			resp.RevivalPassed = out.Passed
			resp.PetState.HealthPoints = out.State.HealthPoints
			resp.PetState.Stage = out.State.Stage
			resp.PetState.CurrentStreak = out.State.CurrentStreak
			c.JSON(http.StatusOK, resp)
		}
	}
}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/pet/... -v
```
Expected: every `--- PASS`, including the two exact-body comparisons.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/pet/handler.go backend/internal/pet/handler_test.go && git commit -m "pet: GET /pet/status and POST /pet/revive with the §6.3 DTOs"
```

---

### Task 8: Integration test — one row per user under concurrent `Ensure`

**Files:**
- Create: `backend/internal/pet/integration_test.go`

Named `TestIntegration…` so CI's `backend-integration` job counts it and fails if it skips
(`.github/workflows/ci.yml`, "Integration tests must run, not skip"). Gated on `TEST_DATABASE_URL`
only — never `DATABASE_URL` (`internal/store/integration_test.go`, `internal/auth/integration_test.go`).

- [ ] **Step 1: Write the test**

```go
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
		if err := repo.Save(ctx, userID, ApplyMiss(st, time.Now())); err != nil {
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
```

- [ ] **Step 2: Confirm it skips without services and the suite is green**

```sh
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1
go test ./internal/pet/... -count=1 -run Integration -v
```
Expected: `ok` everywhere; the second prints `--- SKIP: TestIntegrationEnsureCreatesExactlyOnePetRow`.

- [ ] **Step 3: Run it for real if Docker is available**

```sh
POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose up -d --wait
export TEST_DATABASE_URL='postgres://english:english@localhost:5433/english?sslmode=disable'
export TEST_REDIS_URL='redis://localhost:6380/0'
make test-integration
docker compose down
```
Expected: `--- PASS` for every `TestIntegration*` (store, auth, quests, pet), no `--- SKIP`. If Docker is unavailable, say so in the execution summary; CI's `backend-integration` job is the gate.

- [ ] **Step 4: Commit**

```sh
cd .. && git add backend/internal/pet/integration_test.go && git commit -m "pet: integration test — concurrent Ensure yields one pet_states row"
```

---

### Task 9: Wire it in `main.go`

**Files:**
- Modify: `backend/cmd/api/main.go`

After the quests plan's Task 8, `main.go` contains (verify with `grep -n 'NopPet\|guarded' cmd/api/main.go`):
```go
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

- [ ] **Step 1: Replace the pet placeholder and mount the routes**

Add `"github.com/HendrixNguyen/English-Training-Harness/backend/internal/pet"` to the imports. Replace the block above with:
```go
	studyCounter := quests.NewRedisCounter(rdb)
	petSvc := pet.NewService(
		pet.NewPgRepo(pg.Pool),
		pet.NewRedisChallengeStore(rdb),
		studyCounter, // pet reads the daily counter only through this interface
		time.Now,
	)

	questRepo := quests.NewPgRepo(pg.Pool) // satisfies both QuestRepo and ProgressRepo
	questSvc := quests.NewService(
		studyCounter,
		questRepo,
		questRepo,
		pet.NewQuestHook(petSvc),
		time.Now,
	)

	// Spec §8 hourly cron, in-process (§2.1). Sweeps at every :00 UTC.
	go pet.RunHourly(ctx, petSvc)

	v1 := r.Group("/api/v1")
	v1.POST("/auth/google", auth.Handler(authSvc))

	guarded := v1.Group("", auth.Require(tokens, sessions))
	guarded.GET("/quests/daily", quests.DailyHandler(questSvc))
	guarded.POST("/quests/progress", quests.ProgressHandler(questSvc))
	guarded.GET("/pet/status", pet.StatusHandler(petSvc))
	guarded.POST("/pet/revive", pet.ReviveHandler(petSvc))
```

- [ ] **Step 2: Confirm build, vet and tests**

```sh
go build ./... && go vet ./... && go test ./... -count=1
grep -n 'NopPet' cmd/api/main.go
grep -n 'pet.RunHourly\|pet.NewQuestHook\|/pet/status\|/pet/revive' cmd/api/main.go
```
Expected: clean build/vet; `ok` for every package; the first grep has **no** hits; the second has four.

- [ ] **Step 3: Commit**

```sh
cd .. && git add backend/cmd/api/main.go && git commit -m "pet: register the quests hook, mount /pet routes, start the hourly sweep"
```

---

### Task 10: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (the `**pet**` bullet; one sentence on `**store**`)

- [ ] **Step 1: Replace the pet bullet**

```
- **pet** — the virtual-plant engine (1st-thinking §5.2 steps 4–5; wire contract = backend spec §6.3; arithmetic = §8). `GET /api/v1/pet/status` returns `{plant_name, stage, health_points, current_streak, last_practiced_at}` after `Service.Ensure` — `INSERT INTO pet_states (user_id) … ON CONFLICT (user_id) DO NOTHING` — so the 1:1 row is created idempotently on first read (also called by onboarding); `last_practiced_at` is `null` until the first target is met. **Success (+20 capped at 100, streak+1, `last_practiced_at = now`) is applied exactly once per local day from `quests.Pet.OnTargetMet`** via `pet.QuestHook` (registered in `main.go` in place of `quests.NopPet`); the §8 hourly cron (`pet.RunHourly`, in-process, fires at every `:00` UTC) applies **only** the inactivity logic: for users whose `users.timezone` is in local hour 0, if the previous local day's `daily:accumulated` total (read through the `StudyCounter` interface = `quests.RedisCounter.Total`) is `< 1800`, `health = max(0, health-30)`, `current_streak = 0`, `wilted` at 0; a pet whose `updated_at` is already past that local midnight is skipped, which makes a re-run in the same hour idempotent. Stage thresholds (spec gives none): `wilted` iff health 0, else by streak — 0–2 `sprout`, 3–6 `sapling`, 7–13 `flowering`, 14+ `fruitful`; `seed` is never produced (DDL default is `sprout`). `POST /api/v1/pet/revive` (`{answers?}` accepted, not graded — no spec for it) is 409 `pet_not_wilted` unless health is 0; the first call on a local day starts a 15-minute challenge in `pet:revive:{user_id}` (Hash `started_at, local_date, start_seconds`, TTL 24h — **not in §4**, added by this slice) recording the daily counter's value; a later call the same day passes once ≥ 900 s more were recorded via `POST /quests/progress`, setting `{health_points: 50, stage: sprout, current_streak: 0}` (§6.3) and clearing the key; a challenge from an earlier local day is replaced. Reads `users.timezone` like quests does. Pure tests with fakes and a fixed clock; `TestIntegrationEnsureCreatesExactlyOnePetRow` is gated on `TEST_DATABASE_URL` (skips locally, must pass in CI's `backend-integration` job).
```

- [ ] **Step 2: Append to the store bullet**

At the end of the `**store**` bullet:
```
`PetReviveKey`/`PetReviveTTL` (`pet:revive:{user_id}`, 24h) is the one key not in spec §4 — added by the pet slice for the revive challenge.
```

- [ ] **Step 3: Verify and commit**

```sh
grep -n 'pet.QuestHook\|PetReviveKey' harness/CODEMAP.md
git add harness/CODEMAP.md && git commit -m "codemap: pet — §6.3 routes, once-only success hook, miss-only cron, revive key"
```
Expected: two grep hits.

---

## Verification

Run from the worktree root.

```sh
cd backend && go build ./... && go vet ./...
# expect: no output

env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1
# expect: ok for internal/auth, internal/config, internal/health, internal/pet, internal/quests, internal/store — no live service needed

go test ./internal/pet/... -run 'ApplyMiss|ApplyTargetMet|ApplyRevive|StageFor|Spec8' -v
# expect: --- PASS — -30 floor 0 + wilted, +20 cap 100, revive 50/sprout/0, thresholds

go test ./internal/pet/... -run 'Sweep' -v
# expect: --- PASS — only local-midnight zones, 1799s is a miss, met is spared, idempotent, four misses wilt

go test ./internal/pet/... -run 'Revive' -v
# expect: --- PASS — 409 above 0, start records baseline, pass at +900s, stale day restarts

go test ./internal/pet/... -run 'QuestHook' -v
# expect: --- PASS — *QuestHook satisfies quests.Pet

go test ./internal/pet/... -run 'Status|Handler' -v
# expect: --- PASS — exact §6.3 bodies

grep -n '"plant_name"\|"health_points"\|"current_streak"\|"last_practiced_at"\|"revival_passed"\|"pet_state"' internal/pet/handler.go
# expect: 6+ hits — the §6.3 field names

grep -rn --include='*.go' --exclude='*_test.go' 'MissPenalty *= *30\|TargetMetHealthBonus *= *20\|ReviveHealth *= *50' internal/pet/
# expect: 3 hits — §8/§6.3 numbers, not the idea's -20/20

grep -rn --include='*.go' --exclude='*_test.go' 'ApplyTargetMet' internal/pet/
# expect: definition in engine.go and exactly one call site, in Service.OnTargetMet — never in Sweep

grep -n 'store.PetReviveKey\|store.PetReviveTTL' internal/pet/revive.go
# expect: 3 hits — pet never hand-builds the key

grep -rn --include='*.go' 'daily:accumulated\|DailyAccumulatedKey' internal/pet/
# expect: no hits — pet reads the counter only through StudyCounter

grep -rn 'Getenv("DATABASE_URL")\|Getenv("REDIS_URL")' internal/pet/
# expect: no hits

grep -c '^func TestIntegration' internal/pet/integration_test.go
# expect: 1

grep -n 'NopPet' cmd/api/main.go
# expect: no hits

cd .. && python3 tools/harness/cli.py validate; echo exit=$?
# expect: exit=0

git log --oneline main..HEAD
# expect: 10 commits, one per task, each with the Co-Authored-By trailer

git status --short
# expect: clean
```

After pushing: `gh run list --branch <branch>` must show `backend-unit`, `backend-integration`, `harness-tooling` green.

## Notes and open questions

- **Success once, from the hook; cron = miss only.** Decided above (§6.2 progress-time bump vs §8 cron "Success Logic"). Consequence: a user who never calls `POST /quests/progress` but somehow has ≥ 1800 s in Redis gets no bump — impossible today since the counter is only written by that route.
- **Streak resets on a miss.** §8 only specifies health. If the owner wants streaks to survive a missed day (e.g. for run-01's streak shield), `ApplyMiss` is the one place to change.
- **Revive `answers` are accepted and ignored.** §6.3 puts them on the request; nothing says what questions they answer or how they are graded. The pass condition is the one both §7 lines describe (15 minutes). If a graded mini-quiz is wanted, it slots into `Service.Revive` before `ApplyRevive`.
- **Revive response carries no progress.** §6.3 has only `revival_passed` and `pet_state`; a client wanting a countdown can read `accumulated_seconds` from `GET /quests/daily`. Adding `challenge{started_at, seconds_remaining}` would extend the DTO — not done.
- **Cron granularity vs half-hour timezones.** The sweep runs at `:00` UTC; a user in a `+05:30` zone hits midnight at `:30` and is swept up to 30 minutes later. A target met in that window (needs 30 minutes of study after midnight) would set `updated_at` past midnight and spare yesterday's miss. Accepted for MVP.
- **DST.** On the night a zone's clocks fall back, hour 0 occurs twice; the `updated_at` guard makes the second pass a no-op. On spring-forward there is still an hour 0. Not tested.
- **`Sweep` reads the counter per user** (N Redis `GET`s per midnight zone). Fine at MVP scale; `MGET` when it is not.
- **A wilted plant also recovers through a full 30-minute day** without the revive route (`ApplyTargetMet` on health 0 → 20 / sprout / streak 1). Intentional — revive is the faster path to 50.
- **`pet_states.updated_at` is set by Go, not `now()`**, so the idempotency guard and the tests share one clock. The DDL default still covers the INSERT.
- **`users.timezone` read by pet** — same as quests' `Profile`. If the owner wants a single owner for `users` reads, an `auth.ProfileReader` would serve both; not done here.
