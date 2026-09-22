---
idea: harness/ideas/2026-09-22-run-02/store-go-module-postgres-and-redis-clients-migration-0001.md
status: draft
priority: high
merged: false
order: 1
---
# Store: Go module, Postgres and Redis clients, migration 0001 — Plan

**Idea:** `harness/ideas/2026-09-22-run-02/store-go-module-postgres-and-redis-clients-migration-0001.md`
**Goal:** Stand up `backend/` as a Go module with a Gin `/healthz` binary, pgx + go-redis clients, an embedded migration 0001 carrying the §3.2 DDL verbatim, §4 Redis key builders, and a `go test ./...` baseline that passes with no live Postgres or Redis.

**Architecture:** One Go module rooted at `backend/`. `internal/config` reads env. `internal/store` owns both clients, the embedded `migrations/` FS and the migration runner. `internal/health` owns the `/healthz` handler so it can be tested without booting the binary. `cmd/api/main.go` is wiring only. Everything that touches a live service sits behind a small interface (`store.Migrator`, `health.Pinger`) so the default test run uses fakes; the two assertions that genuinely need Postgres live in one file that `t.Skip`s when `DATABASE_URL` is unset.

**Tech stack:** Go 1.22, module path `github.com/HendrixNguyen/English-Training-Harness/backend`, `github.com/gin-gonic/gin`, `github.com/jackc/pgx/v5` (`pgxpool`), `github.com/redis/go-redis/v9`. No ORM — §6.2 and the later slices are raw SQL. No `golang-migrate`: the runner is ~60 lines and unit-testable against a fake.

**Run every command from `backend/`** unless the step says otherwise. `rg` is not installed on this machine — use `grep -n`.

## File structure

| Path | Responsibility |
| --- | --- |
| `backend/go.mod`, `backend/go.sum` | module + pinned deps |
| `backend/Makefile` | `make test` → `go test ./...`; `make run`; `make up`/`make down` |
| `backend/.gitignore` | built binary |
| `backend/docker-compose.yml` | dev-only Postgres 16 + Redis 7 |
| `backend/internal/config/config.go` `_test.go` | `DATABASE_URL`, `REDIS_URL`, `PORT` |
| `backend/internal/store/keys.go` `keys_test.go` | §4 key builders + TTL constants |
| `backend/internal/store/migrations/0001_init.up.sql` `.down.sql` | §3.2 DDL |
| `backend/internal/store/migrations.go` `migrations_test.go` | embedded FS + `Migrate` + `Migrator` |
| `backend/internal/store/postgres.go` | `pgxpool` client, `Ping`, `Close`, `PgMigrator` |
| `backend/internal/store/redis.go` `redis_test.go` | go-redis client, `Ping`, `Close` |
| `backend/internal/store/integration_test.go` | live-DB assertions, skipped without `DATABASE_URL` |
| `backend/internal/health/health.go` `health_test.go` | `GET /healthz` handler over `Pinger` |
| `backend/cmd/api/main.go` | wiring only |
| `harness/CODEMAP.md` | `store` paragraph |

---

## Tasks

### Task 1: Module skeleton and config loader

**Files:**
- Create: `backend/go.mod` (via `go mod init`), `backend/.gitignore`, `backend/Makefile`
- Create: `backend/internal/config/config.go`
- Test: `backend/internal/config/config_test.go`

- [ ] **Step 1: Initialise the module**

```sh
mkdir -p backend/internal/config
cd backend
go mod init github.com/HendrixNguyen/English-Training-Harness/backend
go mod edit -go=1.22
cat go.mod
```
Expected: `module github.com/HendrixNguyen/English-Training-Harness/backend` and `go 1.22`.

- [ ] **Step 2: Write the failing test**

`backend/internal/config/config_test.go`:
```go
package config

import "testing"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when DATABASE_URL is unset, got nil")
	}
}

func TestLoadRequiresRedisURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when REDIS_URL is unset, got nil")
	}
}

func TestLoadDefaultsPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil error", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.DatabaseURL != "postgres://u:p@localhost:5432/db" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("RedisURL = %q", cfg.RedisURL)
	}
}

func TestLoadHonoursPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("PORT", "9999")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil error", err)
	}
	if cfg.Port != "9999" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9999")
	}
}
```

- [ ] **Step 3: Run the test and confirm it fails**

```sh
go test ./internal/config/...
```
Expected: `FAIL` with `undefined: Load` (a build failure counts as the failing step).

- [ ] **Step 4: Write the minimal implementation**

`backend/internal/config/config.go`:
```go
// Package config loads the process environment described in spec §8.
package config

import (
	"fmt"
	"os"
)

// Config holds the environment this slice needs. Later slices add fields
// (GOOGLE_CLIENT_ID, JWT_SECRET, VAPID_*) as they are introduced.
type Config struct {
	DatabaseURL string
	RedisURL    string
	Port        string
}

// Load reads the environment and validates the required variables.
func Load() (Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		Port:        os.Getenv("PORT"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: DATABASE_URL is required")
	}
	if cfg.RedisURL == "" {
		return Config{}, fmt.Errorf("config: REDIS_URL is required")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	return cfg, nil
}
```

- [ ] **Step 5: Run the test and confirm it passes**

```sh
go test ./internal/config/...
```
Expected: `ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/config`

- [ ] **Step 6: Add the Makefile and .gitignore**

`backend/Makefile` (recipes must be TAB-indented):
```make
.PHONY: test run up down tidy

test:
	go test ./...

run:
	go run ./cmd/api

tidy:
	go mod tidy

up:
	docker compose up -d

down:
	docker compose down
```

`backend/.gitignore`:
```
/api
```

Verify:
```sh
make test
```
Expected: `ok` for `internal/config`, and `?   ... [no test files]` for nothing else yet.

- [ ] **Step 7: Commit**

```sh
cd .. && git add backend && git commit -m "store: go module skeleton, config loader, Makefile"
```

---

### Task 2: Redis key builders and TTL constants (spec §4)

**Files:**
- Create: `backend/internal/store/keys.go`
- Test: `backend/internal/store/keys_test.go`

Pure functions — no Redis involved. These are the exact strings every later slice must use.

- [ ] **Step 1: Write the failing test**

`backend/internal/store/keys_test.go`:
```go
package store

import (
	"testing"
	"time"
)

func TestKeyBuilders(t *testing.T) {
	const uid = "3f0d1a7e-0000-4000-8000-000000000001"
	day := time.Date(2026, time.September, 22, 23, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"session", SessionKey(uid), "sess:3f0d1a7e-0000-4000-8000-000000000001:token"},
		{"placement", PlacementQuizKey(uid), "quiz:placement:3f0d1a7e-0000-4000-8000-000000000001"},
		{"daily", DailyAccumulatedKey(uid, day), "daily:accumulated:3f0d1a7e-0000-4000-8000-000000000001:2026-09-22"},
		{"ratelimit", AIRateLimitKey(uid), "ratelimit:ai:3f0d1a7e-0000-4000-8000-000000000001"},
		{"webpush", WebPushDelayQueueKey, "queue:webpush:delay"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestDailyAccumulatedKeyUsesTheGivenLocation(t *testing.T) {
	// 2026-09-23T00:30 in Asia/Ho_Chi_Minh is still 2026-09-22 in UTC.
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	day := time.Date(2026, time.September, 23, 0, 30, 0, 0, loc)

	if got, want := DailyAccumulatedKey("u", day), "daily:accumulated:u:2026-09-23"; got != want {
		t.Errorf("DailyAccumulatedKey = %q, want %q", got, want)
	}
}

func TestTTLs(t *testing.T) {
	tests := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"SessionTTL", SessionTTL, 24 * time.Hour},
		{"PlacementQuizTTL", PlacementQuizTTL, 2 * time.Hour},
		{"DailyAccumulatedTTL", DailyAccumulatedTTL, 48 * time.Hour},
		{"AIRateLimitTTL", AIRateLimitTTL, time.Minute},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Run the test and confirm it fails**

```sh
mkdir -p internal/store && go test ./internal/store/...
```
Expected: build failure, `undefined: SessionKey`.

- [ ] **Step 3: Write the minimal implementation**

`backend/internal/store/keys.go`:
```go
// Package store owns the Postgres and Redis clients, the migration runner and
// the Redis key topology from spec §4.
package store

import (
	"fmt"
	"time"
)

// TTLs from spec §4. queue:webpush:delay is persistent and has no TTL.
const (
	SessionTTL          = 24 * time.Hour
	PlacementQuizTTL    = 2 * time.Hour
	DailyAccumulatedTTL = 48 * time.Hour
	AIRateLimitTTL      = time.Minute
)

// WebPushDelayQueueKey is the single ZSET of scheduled reminders (spec §4).
const WebPushDelayQueueKey = "queue:webpush:delay"

// SessionKey is sess:{user_id}:token — the active JWT session (TTL SessionTTL).
func SessionKey(userID string) string { return fmt.Sprintf("sess:%s:token", userID) }

// PlacementQuizKey is quiz:placement:{user_id} (TTL PlacementQuizTTL).
func PlacementQuizKey(userID string) string { return fmt.Sprintf("quiz:placement:%s", userID) }

// DailyAccumulatedKey is daily:accumulated:{user_id}:{YYYY-MM-DD}
// (TTL DailyAccumulatedTTL). The date is formatted in day's own location, so
// callers must pass a time already converted to the user's timezone.
func DailyAccumulatedKey(userID string, day time.Time) string {
	return fmt.Sprintf("daily:accumulated:%s:%s", userID, day.Format("2006-01-02"))
}

// AIRateLimitKey is ratelimit:ai:{user_id} (TTL AIRateLimitTTL, max 5 req/min).
func AIRateLimitKey(userID string) string { return fmt.Sprintf("ratelimit:ai:%s", userID) }
```

- [ ] **Step 4: Run the test and confirm it passes**

```sh
go test ./internal/store/...
```
Expected: `ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/store`

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "store: Redis key builders and TTL constants from spec §4"
```

---

### Task 3: Migration 0001 SQL, transcribed verbatim from spec §3.2

**Files:**
- Create: `backend/internal/store/migrations/0001_init.up.sql`
- Create: `backend/internal/store/migrations/0001_init.down.sql`
- Create: `backend/internal/store/migrations.go` (embed only — the runner arrives in Task 4)
- Test: `backend/internal/store/migrations_test.go`

The spec backslash-escapes underscores (`pet\_states`); the SQL files must **not**. Read the source with
`sed -n '144,256p' ../1st-thinking-architecture-doc.md` and strip the backslashes.

- [ ] **Step 1: Write the failing test**

`backend/internal/store/migrations_test.go`:
```go
package store

import (
	"io/fs"
	"strings"
	"testing"
)

func readMigration(t *testing.T, name string) string {
	t.Helper()
	b, err := fs.ReadFile(MigrationsFS, "migrations/"+name)
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(b)
}

func TestMigration0001MatchesSpec32(t *testing.T) {
	up := readMigration(t, "0001_init.up.sql")

	want := []string{
		"CREATE TYPE cefr_level AS ENUM ('A1', 'A2', 'B1', 'B2', 'C1', 'C2');",
		"CREATE TYPE pet_stage AS ENUM ('seed', 'sprout', 'sapling', 'flowering', 'fruitful', 'wilted');",
		"CREATE TYPE task_category AS ENUM ('vocabulary', 'reading', 'practice');",
		"CREATE TABLE users (",
		"CREATE TABLE push_subscriptions (",
		"CREATE TABLE pet_states (",
		"CREATE TABLE daily_progress (",
		"CREATE TABLE roadmaps (",
		"CREATE TABLE exercises (",
		"email VARCHAR(255) UNIQUE NOT NULL",
		"google_id VARCHAR(255) UNIQUE NOT NULL",
		"target_goal VARCHAR(255) NOT NULL",
		// The 1:1 constraint merged by plan add-unique-user-id-to-pet-states-ddl.
		"user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"health_points INT DEFAULT 100 CHECK (health_points BETWEEN 0 AND 100)",
		"stage pet_stage DEFAULT 'sprout'",
		"UNIQUE(user_id, date)",
		"task_type task_category NOT NULL",
		"content_json JSONB NOT NULL",
	}
	for _, w := range want {
		if !strings.Contains(up, w) {
			t.Errorf("0001_init.up.sql is missing %q", w)
		}
	}
}

func TestOnlyPetStatesHasUniqueUserID(t *testing.T) {
	up := readMigration(t, "0001_init.up.sql")

	if got := strings.Count(up, "user_id UUID UNIQUE NOT NULL"); got != 1 {
		t.Errorf("UNIQUE user_id appears %d times, want exactly 1 (pet_states only)", got)
	}
	// push_subscriptions, daily_progress and roadmaps stay 1:N.
	if got := strings.Count(up, "user_id UUID REFERENCES users(id) ON DELETE CASCADE"); got != 3 {
		t.Errorf("plain 1:N user_id appears %d times, want 3", got)
	}
}

func TestMigration0001DownDropsEverything(t *testing.T) {
	down := readMigration(t, "0001_init.down.sql")

	for _, w := range []string{
		"DROP TABLE IF EXISTS exercises;",
		"DROP TABLE IF EXISTS roadmaps;",
		"DROP TABLE IF EXISTS daily_progress;",
		"DROP TABLE IF EXISTS pet_states;",
		"DROP TABLE IF EXISTS push_subscriptions;",
		"DROP TABLE IF EXISTS users;",
		"DROP TYPE IF EXISTS task_category;",
		"DROP TYPE IF EXISTS pet_stage;",
		"DROP TYPE IF EXISTS cefr_level;",
	} {
		if !strings.Contains(down, w) {
			t.Errorf("0001_init.down.sql is missing %q", w)
		}
	}
	// exercises references roadmaps, so it must be dropped first.
	if strings.Index(down, "DROP TABLE IF EXISTS exercises;") > strings.Index(down, "DROP TABLE IF EXISTS roadmaps;") {
		t.Error("down migration must drop exercises before roadmaps")
	}
}
```

- [ ] **Step 2: Run the test and confirm it fails**

```sh
go test ./internal/store/...
```
Expected: build failure, `undefined: MigrationsFS`.

- [ ] **Step 3: Write the SQL**

`backend/internal/store/migrations/0001_init.up.sql`:
```sql
-- Migration 0001 — initial schema, transcribed verbatim from
-- 1st-thinking-architecture-doc.md §3.2 (the spec escapes underscores; this does not).
-- gen_random_uuid() is core in PostgreSQL 13+; docker-compose.yml pins postgres:16.

CREATE TYPE cefr_level AS ENUM ('A1', 'A2', 'B1', 'B2', 'C1', 'C2');

CREATE TYPE pet_stage AS ENUM ('seed', 'sprout', 'sapling', 'flowering', 'fruitful', 'wilted');

CREATE TYPE task_category AS ENUM ('vocabulary', 'reading', 'practice');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    full_name VARCHAR(255),
    google_id VARCHAR(255) UNIQUE NOT NULL,
    google_refresh_token TEXT,
    cefr_current cefr_level DEFAULT 'A1',
    target_goal VARCHAR(255) NOT NULL,
    notification_time TIME DEFAULT '20:00:00',
    timezone VARCHAR(50) DEFAULT 'UTC',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE push_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE pet_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plant_name VARCHAR(100) DEFAULT 'My Green Buddy',
    health_points INT DEFAULT 100 CHECK (health_points BETWEEN 0 AND 100),
    stage pet_stage DEFAULT 'sprout',
    current_streak INT DEFAULT 0,
    last_practiced_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE daily_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    minutes_spent INT DEFAULT 0,
    is_target_met BOOLEAN DEFAULT FALSE,
    UNIQUE(user_id, date)
);

CREATE TABLE roadmaps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    roadmap_json JSONB NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE exercises (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    roadmap_id UUID REFERENCES roadmaps(id) ON DELETE CASCADE,
    day_number INT NOT NULL,
    task_type task_category NOT NULL,
    content_json JSONB NOT NULL,
    is_completed BOOLEAN DEFAULT FALSE
);
```

`backend/internal/store/migrations/0001_init.down.sql`:
```sql
-- Reverse of 0001_init.up.sql. Children before parents, tables before types.

DROP TABLE IF EXISTS exercises;
DROP TABLE IF EXISTS roadmaps;
DROP TABLE IF EXISTS daily_progress;
DROP TABLE IF EXISTS pet_states;
DROP TABLE IF EXISTS push_subscriptions;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS task_category;
DROP TYPE IF EXISTS pet_stage;
DROP TYPE IF EXISTS cefr_level;
```

`backend/internal/store/migrations.go`:
```go
package store

import "embed"

// MigrationsFS holds the numbered DDL files. Each version is a pair
// NNNN_name.up.sql / NNNN_name.down.sql; Migrate applies the .up.sql files in
// filename order. Never edit a migration that has been applied — add a new one.
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
```

- [ ] **Step 4: Run the test and confirm it passes**

```sh
go test ./internal/store/... -run 'Migration|UniqueUserID' -v
```
Expected: `--- PASS: TestMigration0001MatchesSpec32`, `--- PASS: TestOnlyPetStatesHasUniqueUserID`, `--- PASS: TestMigration0001DownDropsEverything`.

- [ ] **Step 5: Cross-check the SQL against the spec by eye**

```sh
cd .. && diff \
  <(sed -n '/^CREATE TYPE cefr/,/^);$/p' 1st-thinking-architecture-doc.md | tr -d '\\' | grep -v '^$') \
  <(grep -v '^--' backend/internal/store/migrations/0001_init.up.sql | grep -v '^$' | sed -n '1,/^);$/p')
```
Expected: the first `CREATE TYPE`/`CREATE TABLE users` block matches line for line. Any difference here is a transcription bug — fix the SQL, not the spec.

- [ ] **Step 6: Commit**

```sh
git add backend && git commit -m "store: migration 0001 with the spec §3.2 DDL"
```

---

### Task 4: Migration runner over a `Migrator` interface

**Files:**
- Modify: `backend/internal/store/migrations.go`
- Test: `backend/internal/store/migrations_test.go` (append)

`Migrate` never touches pgx: it talks to a `Migrator`. The Postgres implementation lands in Task 5,
the tests here use a fake, so idempotency is proven without a database.

- [ ] **Step 1: Write the failing test** (append to `migrations_test.go`)

```go
type fakeMigrator struct {
	ensured  int
	applied  []string
	appliedS []string // the SQL passed to Apply, in order
	failOn   string
}

func (f *fakeMigrator) EnsureVersionTable(_ context.Context) error {
	f.ensured++
	return nil
}

func (f *fakeMigrator) AppliedVersions(_ context.Context) (map[string]bool, error) {
	out := map[string]bool{}
	for _, v := range f.applied {
		out[v] = true
	}
	return out, nil
}

func (f *fakeMigrator) Apply(_ context.Context, version, sql string) error {
	if version == f.failOn {
		return errors.New("boom")
	}
	f.applied = append(f.applied, version)
	f.appliedS = append(f.appliedS, sql)
	return nil
}

func TestMigrateAppliesPendingVersions(t *testing.T) {
	m := &fakeMigrator{}

	got, err := Migrate(context.Background(), m, MigrationsFS)
	if err != nil {
		t.Fatalf("Migrate() = %v, want nil error", err)
	}
	if want := []string{"0001_init"}; !reflect.DeepEqual(got, want) {
		t.Errorf("applied = %v, want %v", got, want)
	}
	if m.ensured != 1 {
		t.Errorf("EnsureVersionTable called %d times, want 1", m.ensured)
	}
	if !strings.Contains(m.appliedS[0], "CREATE TABLE users (") {
		t.Error("Apply did not receive the up SQL")
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	m := &fakeMigrator{}

	if _, err := Migrate(context.Background(), m, MigrationsFS); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	got, err := Migrate(context.Background(), m, MigrationsFS)
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("second run applied %v, want nothing", got)
	}
}

func TestMigrateIgnoresDownFiles(t *testing.T) {
	m := &fakeMigrator{}

	if _, err := Migrate(context.Background(), m, MigrationsFS); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	for _, sql := range m.appliedS {
		if strings.Contains(sql, "DROP TABLE") {
			t.Fatal("Migrate applied a .down.sql file")
		}
	}
}

func TestMigrateReportsApplyFailure(t *testing.T) {
	m := &fakeMigrator{failOn: "0001_init"}

	if _, err := Migrate(context.Background(), m, MigrationsFS); err == nil {
		t.Fatal("expected an error when Apply fails, got nil")
	}
}
```

Add the imports `context`, `errors`, `reflect` to the test file's import block.

- [ ] **Step 2: Run the test and confirm it fails**

```sh
go test ./internal/store/... -run Migrate
```
Expected: build failure, `undefined: Migrate`.

- [ ] **Step 3: Write the minimal implementation** (append to `migrations.go`)

```go
import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// Migrator is the storage side of the migration runner. PgMigrator is the
// Postgres implementation; tests use a fake.
type Migrator interface {
	// EnsureVersionTable creates the bookkeeping table if it is missing.
	EnsureVersionTable(ctx context.Context) error
	// AppliedVersions returns the set of versions already applied.
	AppliedVersions(ctx context.Context) (map[string]bool, error)
	// Apply runs one migration and records its version atomically.
	Apply(ctx context.Context, version, sql string) error
}

// Migrate applies every *.up.sql in fsys that has not been applied yet, in
// filename order, and returns the versions it applied. It is safe to call on
// every boot: a second call with no new files applies nothing.
func Migrate(ctx context.Context, m Migrator, fsys fs.FS) ([]string, error) {
	if err := m.EnsureVersionTable(ctx); err != nil {
		return nil, fmt.Errorf("store: ensuring version table: %w", err)
	}
	done, err := m.AppliedVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: reading applied versions: %w", err)
	}

	names, err := fs.Glob(fsys, "migrations/*.up.sql")
	if err != nil {
		return nil, fmt.Errorf("store: listing migrations: %w", err)
	}
	sort.Strings(names)

	var applied []string
	for _, name := range names {
		version := strings.TrimSuffix(strings.TrimPrefix(name, "migrations/"), ".up.sql")
		if done[version] {
			continue
		}
		body, err := fs.ReadFile(fsys, name)
		if err != nil {
			return applied, fmt.Errorf("store: reading %s: %w", name, err)
		}
		if err := m.Apply(ctx, version, string(body)); err != nil {
			return applied, fmt.Errorf("store: applying %s: %w", version, err)
		}
		applied = append(applied, version)
	}
	return applied, nil
}
```

- [ ] **Step 4: Run the test and confirm it passes**

```sh
go test ./internal/store/... -run Migrate -v
```
Expected: four `--- PASS` lines.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "store: migration runner with version bookkeeping"
```

---

### Task 5: Postgres client and `PgMigrator`

**Files:**
- Create: `backend/internal/store/postgres.go`
- Test: `backend/internal/store/postgres_test.go`

- [ ] **Step 1: Add the dependency**

```sh
go get github.com/jackc/pgx/v5@latest
```
Expected: `go: added github.com/jackc/pgx/v5 v5.x.y`.

- [ ] **Step 2: Write the failing test**

`backend/internal/store/postgres_test.go`:
```go
package store

import (
	"context"
	"strings"
	"testing"
)

func TestNewPostgresRejectsAnUnparseableURL(t *testing.T) {
	if _, err := NewPostgres(context.Background(), "://not a url"); err == nil {
		t.Fatal("expected an error for a malformed DATABASE_URL, got nil")
	}
}

func TestVersionTableDDLIsIdempotent(t *testing.T) {
	if !strings.Contains(versionTableDDL, "CREATE TABLE IF NOT EXISTS schema_migrations") {
		t.Errorf("versionTableDDL must be IF NOT EXISTS, got:\n%s", versionTableDDL)
	}
	if !strings.Contains(versionTableDDL, "version TEXT PRIMARY KEY") {
		t.Errorf("schema_migrations needs a primary key on version, got:\n%s", versionTableDDL)
	}
}
```

- [ ] **Step 3: Run the test and confirm it fails**

```sh
go test ./internal/store/... -run 'Postgres|VersionTable'
```
Expected: build failure, `undefined: NewPostgres`.

- [ ] **Step 4: Write the minimal implementation**

`backend/internal/store/postgres.go`:
```go
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// versionTableDDL is the migration runner's own bookkeeping table.
const versionTableDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
)`

// Postgres is the process-wide connection pool built from DATABASE_URL.
type Postgres struct {
	Pool *pgxpool.Pool
}

// NewPostgres parses url and creates a lazy pool; it does not dial. Call Ping
// to confirm the database is reachable.
func NewPostgres(ctx context.Context, url string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("store: parsing DATABASE_URL: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: creating pool: %w", err)
	}
	return &Postgres{Pool: pool}, nil
}

// Ping reports whether the database is reachable (used by GET /healthz).
func (p *Postgres) Ping(ctx context.Context) error { return p.Pool.Ping(ctx) }

// Close releases the pool.
func (p *Postgres) Close() { p.Pool.Close() }

// Migrator returns the Migrator backed by this pool.
func (p *Postgres) Migrator() Migrator { return &PgMigrator{pool: p.Pool} }

// PgMigrator applies migrations to Postgres, one transaction per version.
type PgMigrator struct{ pool *pgxpool.Pool }

func (m *PgMigrator) EnsureVersionTable(ctx context.Context) error {
	_, err := m.pool.Exec(ctx, versionTableDDL)
	return err
}

func (m *PgMigrator) AppliedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := m.pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

// Apply runs the migration body and records the version in one transaction, so
// a failed migration leaves no half-applied schema and no version row.
func (m *PgMigrator) Apply(ctx context.Context, version, sql string) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, sql); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// compile-time check
var _ Migrator = (*PgMigrator)(nil)
```

- [ ] **Step 5: Run the test and confirm it passes**

```sh
go mod tidy && go test ./...
```
Expected: `ok` for `internal/config` and `internal/store`.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend && git commit -m "store: pgx pool client and Postgres migrator"
```

---

### Task 6: Redis client

**Files:**
- Create: `backend/internal/store/redis.go`
- Test: `backend/internal/store/redis_test.go`

- [ ] **Step 1: Add the dependency**

```sh
go get github.com/redis/go-redis/v9@latest
```

- [ ] **Step 2: Write the failing test**

`backend/internal/store/redis_test.go`:
```go
package store

import (
	"context"
	"testing"
)

func TestNewRedisRejectsAnUnparseableURL(t *testing.T) {
	if _, err := NewRedis(context.Background(), "http://localhost:6379"); err == nil {
		t.Fatal("expected an error for a non-redis REDIS_URL, got nil")
	}
}

func TestNewRedisAcceptsAValidURLWithoutDialling(t *testing.T) {
	// No server is running; NewRedis must not connect.
	r, err := NewRedis(context.Background(), "redis://localhost:6379/0")
	if err != nil {
		t.Fatalf("NewRedis() = %v, want nil error", err)
	}
	defer r.Close()

	if r.Client == nil {
		t.Fatal("Client is nil")
	}
}
```

- [ ] **Step 3: Run the test and confirm it fails**

```sh
go test ./internal/store/... -run Redis
```
Expected: build failure, `undefined: NewRedis`.

- [ ] **Step 4: Write the minimal implementation**

`backend/internal/store/redis.go`:
```go
package store

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Redis is the process-wide client built from REDIS_URL. Key names and TTLs
// live in keys.go (spec §4).
type Redis struct {
	Client *redis.Client
}

// NewRedis parses url and creates a client; it does not dial. Call Ping to
// confirm the server is reachable.
func NewRedis(_ context.Context, url string) (*Redis, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("store: parsing REDIS_URL: %w", err)
	}
	return &Redis{Client: redis.NewClient(opt)}, nil
}

// Ping reports whether Redis is reachable (used by GET /healthz).
func (r *Redis) Ping(ctx context.Context) error { return r.Client.Ping(ctx).Err() }

// Close releases the client.
func (r *Redis) Close() error { return r.Client.Close() }
```

- [ ] **Step 5: Run the test and confirm it passes**

```sh
go mod tidy && go test ./internal/store/... -run Redis -v
```
Expected: two `--- PASS` lines.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend && git commit -m "store: go-redis client"
```

---

### Task 7: `GET /healthz` handler

**Files:**
- Create: `backend/internal/health/health.go`
- Test: `backend/internal/health/health_test.go`

The handler lives in its own package so it can be tested with fakes; `main.go` stays wiring-only.

- [ ] **Step 1: Add the dependency**

```sh
go get github.com/gin-gonic/gin@latest
```

- [ ] **Step 2: Write the failing test**

`backend/internal/health/health_test.go`:
```go
package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(_ context.Context) error { return f.err }

func newRouter(db, cache Pinger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/healthz", Handler(db, cache))
	return r
}

func do(t *testing.T, r *gin.Engine) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	return w
}

func TestHealthzOKWhenBothServicesRespond(t *testing.T) {
	w := do(t, newRouter(fakePinger{}, fakePinger{}))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Errorf("body = %s", w.Body.String())
	}
}

func TestHealthzUnavailableWhenPostgresIsDown(t *testing.T) {
	w := do(t, newRouter(fakePinger{err: errors.New("no pg")}, fakePinger{}))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"postgres":"no pg"`) {
		t.Errorf("body = %s, want the postgres error reported", w.Body.String())
	}
}

func TestHealthzUnavailableWhenRedisIsDown(t *testing.T) {
	w := do(t, newRouter(fakePinger{}, fakePinger{err: errors.New("no redis")}))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"redis":"no redis"`) {
		t.Errorf("body = %s, want the redis error reported", w.Body.String())
	}
}
```

- [ ] **Step 3: Run the test and confirm it fails**

```sh
mkdir -p internal/health && go test ./internal/health/...
```
Expected: build failure, `undefined: Handler`.

- [ ] **Step 4: Write the minimal implementation**

`backend/internal/health/health.go`:
```go
// Package health serves GET /healthz: a liveness probe that reports whether
// Postgres and Redis are reachable.
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Pinger is anything that can be checked for reachability. store.Postgres and
// store.Redis both satisfy it.
type Pinger interface {
	Ping(ctx context.Context) error
}

const pingTimeout = 2 * time.Second

// Handler returns 200 when both dependencies answer, 503 otherwise, naming the
// failing dependency in the body.
func Handler(db, cache Pinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), pingTimeout)
		defer cancel()

		body := gin.H{"status": "ok", "postgres": "ok", "redis": "ok"}
		healthy := true

		if err := db.Ping(ctx); err != nil {
			body["postgres"] = err.Error()
			healthy = false
		}
		if err := cache.Ping(ctx); err != nil {
			body["redis"] = err.Error()
			healthy = false
		}
		if !healthy {
			body["status"] = "unavailable"
			c.JSON(http.StatusServiceUnavailable, body)
			return
		}
		c.JSON(http.StatusOK, body)
	}
}
```

- [ ] **Step 5: Run the test and confirm it passes**

```sh
go mod tidy && go test ./internal/health/... -v
```
Expected: three `--- PASS` lines.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend && git commit -m "store: GET /healthz handler over a Pinger interface"
```

---

### Task 8: `cmd/api/main.go` wiring

**Files:**
- Create: `backend/cmd/api/main.go`

No test: this file is wiring, and every piece it wires is covered above.

- [ ] **Step 1: Write it**

`backend/cmd/api/main.go`:
```go
// Command api is the single Go process described in spec §2.1. This slice
// serves only GET /healthz; later slices mount their own route groups here.
package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/config"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/health"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	pg, err := store.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pg.Close()

	rdb, err := store.NewRedis(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer func() { _ = rdb.Close() }()

	applied, err := store.Migrate(ctx, pg.Migrator(), store.MigrationsFS)
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if len(applied) > 0 {
		log.Printf("migrations applied: %v", applied)
	}

	r := gin.Default()
	r.GET("/healthz", health.Handler(pg, rdb))

	log.Printf("listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
```

- [ ] **Step 2: Confirm it builds and the suite is still green**

```sh
go build ./... && go vet ./... && go test ./...
```
Expected: no output from build/vet; `ok` for `internal/config`, `internal/health`, `internal/store`,
and `?   ... [no test files]` for `cmd/api`.

- [ ] **Step 3: Commit**

```sh
cd .. && git add backend && git commit -m "store: cmd/api boots Gin with /healthz and runs migrations"
```

---

### Task 9: docker-compose and the skipped integration tests

**Files:**
- Create: `backend/docker-compose.yml`
- Create: `backend/internal/store/integration_test.go`

These are the only tests that need a live database. They **skip**, not fail, when `DATABASE_URL` is unset —
so `go test ./...` stays green on a machine with no services.

- [ ] **Step 1: Write the compose file**

`backend/docker-compose.yml`:
```yaml
# Dev only. Not used in deployment: spec §8 provisions Postgres and Redis as
# Railway plugins and injects DATABASE_URL / REDIS_URL.
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: english
      POSTGRES_PASSWORD: english
      POSTGRES_DB: english
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U english"]
      interval: 2s
      timeout: 3s
      retries: 15

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 2s
      timeout: 3s
      retries: 15
```

Document the URLs in the Makefile so nobody has to guess — append to `backend/Makefile`:
```make
# Matches docker-compose.yml:
#   export DATABASE_URL=postgres://english:english@localhost:5432/english?sslmode=disable
#   export REDIS_URL=redis://localhost:6379/0
.PHONY: test-integration
test-integration:
	go test ./... -count=1 -v -run Integration
```

- [ ] **Step 2: Write the integration tests**

`backend/internal/store/integration_test.go`:
```go
package store

import (
	"context"
	"os"
	"testing"
)

// requirePostgres skips the test when no database is configured. This is
// deliberate: the default `go test ./...` run must not need live services.
func requirePostgres(t *testing.T) *Postgres {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is unset; run `make up` and export it to run integration tests")
	}
	pg, err := NewPostgres(context.Background(), url)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	t.Cleanup(pg.Close)
	return pg
}

// reset drops everything 0001 creates plus the bookkeeping table, so each test
// starts from an empty database.
func reset(t *testing.T, pg *Postgres) {
	t.Helper()
	down, err := MigrationsFS.ReadFile("migrations/0001_init.down.sql")
	if err != nil {
		t.Fatalf("reading down migration: %v", err)
	}
	ctx := context.Background()
	if _, err := pg.Pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("down migration: %v", err)
	}
	if _, err := pg.Pool.Exec(ctx, `DROP TABLE IF EXISTS schema_migrations`); err != nil {
		t.Fatalf("dropping schema_migrations: %v", err)
	}
}

func TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent(t *testing.T) {
	pg := requirePostgres(t)
	reset(t, pg)
	t.Cleanup(func() { reset(t, pg) })

	ctx := context.Background()

	first, err := Migrate(ctx, pg.Migrator(), MigrationsFS)
	if err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if len(first) != 1 || first[0] != "0001_init" {
		t.Fatalf("first run applied %v, want [0001_init]", first)
	}

	second, err := Migrate(ctx, pg.Migrator(), MigrationsFS)
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if len(second) != 0 {
		t.Errorf("second run applied %v, want nothing", second)
	}

	for _, table := range []string{"users", "push_subscriptions", "pet_states", "daily_progress", "roadmaps", "exercises"} {
		var exists bool
		err := pg.Pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)`, table).Scan(&exists)
		if err != nil {
			t.Fatalf("checking %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s was not created", table)
		}
	}
}

func TestIntegrationPetStatesRejectsASecondRowForTheSameUser(t *testing.T) {
	pg := requirePostgres(t)
	reset(t, pg)
	t.Cleanup(func() { reset(t, pg) })

	ctx := context.Background()
	if _, err := Migrate(ctx, pg.Migrator(), MigrationsFS); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var userID string
	err := pg.Pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, target_goal) VALUES ($1, $2, $3) RETURNING id`,
		"a@example.com", "google-1", "").Scan(&userID)
	if err != nil {
		t.Fatalf("inserting user: %v", err)
	}

	if _, err := pg.Pool.Exec(ctx, `INSERT INTO pet_states (user_id) VALUES ($1)`, userID); err != nil {
		t.Fatalf("first pet_states insert: %v", err)
	}
	if _, err := pg.Pool.Exec(ctx, `INSERT INTO pet_states (user_id) VALUES ($1)`, userID); err == nil {
		t.Fatal("second pet_states insert succeeded; user_id is not UNIQUE")
	}
}

func TestIntegrationRedisRoundTrip(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL is unset; run `make up` and export it to run integration tests")
	}
	rdb, err := NewRedis(context.Background(), url)
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	if err := rdb.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	key := SessionKey("integration-test-user")
	t.Cleanup(func() { rdb.Client.Del(ctx, key) })

	if err := rdb.Client.Set(ctx, key, "token", SessionTTL).Err(); err != nil {
		t.Fatalf("Set: %v", err)
	}
	ttl, err := rdb.Client.TTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if ttl <= 0 || ttl > SessionTTL {
		t.Errorf("TTL = %v, want (0, %v]", ttl, SessionTTL)
	}
}
```

- [ ] **Step 3: Confirm they skip with no services**

```sh
env -u DATABASE_URL -u REDIS_URL go test ./internal/store/... -run Integration -v
```
Expected: three `--- SKIP` lines and `ok` (not `FAIL`).

- [ ] **Step 4: Confirm they pass with services (optional but strongly preferred)**

```sh
docker compose up -d
export DATABASE_URL='postgres://english:english@localhost:5432/english?sslmode=disable'
export REDIS_URL='redis://localhost:6379/0'
go test ./internal/store/... -count=1 -run Integration -v
docker compose down
```
Expected: three `--- PASS` lines. If Docker is unavailable in the worktree, record that in the execution
summary and leave the tests skipping — the plan does not require Docker to be green.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "store: dev docker-compose and skippable integration tests"
```

---

### Task 10: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (the `## Planned backend packages` heading and the `**store**` bullet)

- [ ] **Step 1: Rewrite the heading and the store bullet**

The heading currently reads `## Planned backend packages (\`backend/internal/\`) — none exist yet`.
Change it to `## Backend packages (\`backend/internal/\`)` and replace the `**store**` bullet with:

```
- **store** — Go module `github.com/HendrixNguyen/English-Training-Harness/backend` (Go 1.22, Gin, pgx/v5, go-redis/v9). Postgres + Redis clients from `DATABASE_URL` / `REDIS_URL`; `store.Migrate(ctx, pg.Migrator(), store.MigrationsFS)` applies `internal/store/migrations/*.up.sql` in filename order and records each in `schema_migrations` (idempotent, run on every boot from `cmd/api/main.go`). `0001_init` is the spec §3.2 DDL verbatim; `pet_states.user_id` is UNIQUE NOT NULL (1:1 with `users`). Redis key builders and §4 TTLs live in `keys.go` — use them, never literal key strings. Tests: `cd backend && make test` (no live services needed); `make up` + exported `DATABASE_URL`/`REDIS_URL` then `make test-integration` for the live-database assertions, which skip otherwise. Everything else depends on this package.
```

Leave the other seven bullets as "planned" — they are still unbuilt.

- [ ] **Step 2: Verify**

```sh
grep -n 'English-Training-Harness' harness/CODEMAP.md
grep -n 'Backend packages' harness/CODEMAP.md
```
Expected: one hit each.

- [ ] **Step 3: Commit**

```sh
git add harness/CODEMAP.md && git commit -m "codemap: store — module path, migration runner, test commands"
```

---

## Verification

Run from the worktree root.

```sh
cd backend && go build ./... && go vet ./...
# expect: no output

go test ./...
# expect (order may vary, no FAIL lines):
# ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/config
# ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/health
# ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/store
# ?   	github.com/HendrixNguyen/English-Training-Harness/backend/cmd/api	[no test files]

env -u DATABASE_URL -u REDIS_URL go test ./... -count=1
# expect: still ok — the default run needs no live services

make test
# expect: identical to `go test ./...`

grep -c 'CREATE TABLE' internal/store/migrations/0001_init.up.sql
# expect: 6

grep -c 'CREATE TYPE' internal/store/migrations/0001_init.up.sql
# expect: 3

grep -n 'user_id UUID UNIQUE NOT NULL' internal/store/migrations/0001_init.up.sql
# expect exactly one hit, inside CREATE TABLE pet_states

grep -n 'sess:%s:token\|quiz:placement:%s\|daily:accumulated:%s:%s\|ratelimit:ai:%s\|queue:webpush:delay' internal/store/keys.go
# expect 5 hits — all five §4 keys

cd .. && python3 tools/harness/cli.py validate; echo exit=$?
# expect: exit=0

git log --oneline main..HEAD
# expect 10 commits, one per task, each with the Co-Authored-By trailer

git status --short
# expect: clean (no untracked files under backend/ — the binary is gitignored)
```

## Notes and open questions

- **`users.target_goal` is `NOT NULL` with no default.** Migration 0001 transcribes §3.2 as written, so the auth slice (2) must insert `''` at login. Changing this to `DEFAULT ''` is a spec edit and belongs in its own bug file, not here.
- **`gen_random_uuid()`** is core from PostgreSQL 13; `docker-compose.yml` pins `postgres:16-alpine`. If the Railway plugin turns out to be PG 12 or older, migration 0002 adds `CREATE EXTENSION IF NOT EXISTS pgcrypto;` — do not edit 0001 after it has been applied anywhere.
- **The two open `pet_states` spec bugs** in this run (`reconcile-pet-states-stage-between-erd-and-ddl-wilted-defaul`, `spec-never-states-when-the-pet-states-row-is-created`) are not resolved here. 0001 ships `stage pet_stage DEFAULT 'sprout'` because that is what §3.2 says today; any correction lands as 0002.
- **No `JWT_SECRET` in `config.Config`** — §8 does not list it and this slice does not need it. Slice 2 adds the field and the required-variable check.
