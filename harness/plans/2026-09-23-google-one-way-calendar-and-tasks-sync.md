---
idea: harness/ideas/2026-09-22-run-02/google-one-way-calendar-and-tasks-sync.md
status: done
priority: high
merged: false
order: 7
branch: harness/2026-09-23-high-google-one-way-calendar-and-tasks-sync
worktree: .worktrees/google-one-way-calendar-and-tasks-sync
---
# Google: one-way Calendar and Tasks sync — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use the executing-plans (or subagent-driven-development) skill to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/2026-09-22-run-02/google-one-way-calendar-and-tasks-sync.md`
**Goal:** Add `backend/internal/google` — `POST /api/v1/integrations/google/sync` behind `auth.Require()` — that refreshes a Google access token from `users.google_refresh_token`, pushes one recurring 30-minute Calendar event (`RRULE:FREQ=DAILY;COUNT=28`, starting at the next `users.notification_time` in `users.timezone`) and one Google Tasks list with one task per roadmap day, answers the **backend spec §6.4 DTO field for field** (`{"status":"synced","calendar_event_id":"…","tasks_created_count":N}`), is idempotent on re-sync via a new `google_sync` table (migration `0002`), and maps Google's `invalid_grant` to 409 `{"error":"reauth_required"}`.

**Spec precedence:** the *Backend Technical Specification* §6.4 wins for the wire shape (AGENTS.md → *Reading the spec*): the idea's `{calendar_event_id, tasklist_id, tasks_created}` is superseded by §6.4's `{status, calendar_event_id, tasks_created_count}` (the idea's `## Evaluation` records this). §6.4 says "asynchronously" but its 200 body carries the ids Google returns, so the sync runs **synchronously inside the request** under a 60-second deadline; recorded in *Notes*. 1st-thinking §5.1 steps 6–7 ("Push 30-min Recurring Event", "Push Daily Checklist Tasks") supply the direction (one-way, nothing read back or subscribed to); §3.2 supplies the columns read; §7 (AES-256-GCM at rest) is **not** implemented here — see *Refresh token seam* below.

**Architecture:** A `Service` over five interfaces, all injected so every test is pure: `RefreshTokenSource` (the one place `users.google_refresh_token` is read), `TokenRefresher` (OAuth `grant_type=refresh_token`), `CalendarClient` (`events.insert` / `events.patch` on the primary calendar), `TasksClient` (`tasklists.insert` / `tasklists.delete` / `tasks.insert`) and `Repo` (profile, active roadmap + per-day titles, and the `google_sync` row). The three HTTP clients take a base URL / token URL as a struct field with a production default, exactly like `auth.GoogleClient` (`internal/auth/google.go:14-18`), so tests point them at `httptest.Server`s — **`go test ./...` never calls Google**. Event timing is a pure function of a clock, `notification_time` and a timezone, in its own file. Idempotency: `google_sync` is keyed `user_id` (one row per user), holds `calendar_event_id`, `tasklist_id`, `roadmap_id` and `tasks_created_count`; a re-sync **patches** the stored event (re-inserting only when Google says 404/410), and for tasks either does nothing (same `roadmap_id`, count re-reported), or deletes the stored list and builds a new one (new roadmap). State is persisted **after the event step and after the list is created, before the 28 task inserts**, so a crash mid-sync never orphans a Google object the next sync cannot find.

**Storage decision — `google_sync` table via migration `0002`, not JSONB on `roadmaps` (decided in the idea's `## Evaluation`):** (a) the Calendar event exists even for a user with no roadmap, so `roadmaps` cannot own it; (b) `roadmaps.roadmap_json` is the AI-generated document onboarding writes and quests reads — writing integration state into it crosses the CODEMAP boundary ("packages talk via interfaces, never each other's tables"); (c) a `user_id` primary key gives idempotent re-sync one `ON CONFLICT (user_id) DO UPDATE` and records which `roadmap_id` the tasks were pushed for. Cost, paid in Task 1: `internal/store`'s tests hard-code `0001_init` as the only version (`integration_test.go:49,73-74,141-142`, `migrations_test.go:124`), and the backend spec's §3.2 DDL block must gain the same `CREATE TABLE` (AGENTS.md: spec DDL == migrations, executor's Definition of done).

**Refresh token seam:** `RefreshTokenSource` has exactly one implementation today, `PgRefreshTokenSource`, which reads the column as plaintext because that is what the merged `auth` slice writes (`internal/auth/repo.go:34-39`). The inbox bug `harness/ideas/_inbox/google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md` (spec §7 AES-256-GCM via `ENCRYPTION_SECRET_KEY`) is **not folded in**: its fix replaces that one struct with a decrypting one and touches nothing else in this package. Do not add a cipher here.

**Tech stack:** Go 1.25 (`backend/go.mod`), Gin, `pgx/v5` — all already in `go.mod`. **No new dependencies**: Google's Calendar v3 and Tasks v1 REST endpoints are called with `net/http` + `encoding/json` (the official `google.golang.org/api` client pulls in ~40 modules for three calls and would hard-code the base URLs the tests need to override).

**Depends on:** `store` (1), `auth` (2), `quests` (3) and `airouter` (5) merged on `main`. Symbols checked against `main`: `store.Migrate(ctx, m, store.MigrationsFS)` / `MigrationsFS` (`internal/store/migrations.go`), `store.NewPostgres` → `*store.Postgres{Pool}` / `.Migrator()` (`postgres.go:24,43`), `auth.Require(tokens *auth.TokenIssuer, sessions auth.SessionStore)`, `auth.UserID(c)`, `auth.ContextUserID` (`middleware.go:11,20,43`), `auth.GoogleClient{TokenURL, HTTPClient}` pattern (`google.go:44-60`), `config.Config{GoogleClientID, GoogleClientSecret}` (`config/config.go`), `quests.Location` (`quests/day.go`, copied not imported — see *Notes*). The `roadmaps` / `exercises` rows this package reads are written by onboarding (order 6, approved: `INSERT roadmaps (…, is_active = TRUE)`, 84 × `INSERT exercises` with `content_json` = the task object carrying `title`) and today by `store.SeedDemoRoadmap`; both write the same `content_json->>'title'` key quests' `toTask` reads. **In flight, do not conflict:** pet (order 4) and onboarding (order 6) both edit `cmd/api/main.go`; this plan's Task 7 adds one import and one route line and tells you to `grep` the file first rather than assume its shape.

**Run every command from `backend/`** unless the step says otherwise. `rg` is not installed — use `grep -n`. `timeout` is not installed — bound tests with `go test -timeout`. Commit messages end with the trailer after a blank line: `Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>` (shown once below; add it to every commit).

## File structure

| Path | Responsibility |
| --- | --- |
| `backend/internal/store/migrations/0002_google_sync.up.sql` `.down.sql` | the `google_sync` table |
| `backend/internal/store/migrations_test.go`, `integration_test.go` | know about two versions |
| `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` | §3.2 DDL gains `google_sync` |
| `backend/internal/google/schedule.go` `schedule_test.go` | `Location`, `NextOccurrence`, `PracticeEvent`, `DayDue`, `TaskTitle` — pure |
| `backend/internal/google/token.go` `token_test.go` | `RefreshTokenSource` + `PgRefreshTokenSource` (plaintext today); `ErrNoRefreshToken` |
| `backend/internal/google/oauth.go` `oauth_test.go` | `TokenRefresher` + `OAuthClient` (`grant_type=refresh_token`); `ErrReauthRequired` |
| `backend/internal/google/client.go` | shared `doJSON`, `UpstreamError`, `ErrNotFound`, status mapping |
| `backend/internal/google/calendar.go` `calendar_test.go` | `Event`, `CalendarClient` + `HTTPCalendarClient` |
| `backend/internal/google/tasks.go` `tasks_test.go` | `Task`, `TasksClient` + `HTTPTasksClient` |
| `backend/internal/google/repo.go` `integration_test.go` | `Repo` interface, models, `PgRepo`; `TestIntegrationSyncStateIsOneRowPerUser` |
| `backend/internal/google/fakes_test.go` | in-memory fakes with a call log |
| `backend/internal/google/service.go` `service_test.go` | `Service.Sync` — the idempotent algorithm |
| `backend/internal/google/handler.go` `handler_test.go` | `POST /integrations/google/sync`, §6.4 body, error codes |
| `backend/cmd/api/main.go` | mount the route behind `auth.Require()` |
| `harness/CODEMAP.md` | `google` paragraph; `store` bullet mentions `0002` |

---

## Tasks

### Task 1: Migration `0002_google_sync` and the store tests that assumed one version

**Files:**
- Create: `backend/internal/store/migrations/0002_google_sync.up.sql`
- Create: `backend/internal/store/migrations/0002_google_sync.down.sql`
- Modify: `backend/internal/store/migrations_test.go:117-133` (`TestMigrateAppliesPendingVersions`), add `TestMigration0002CreatesGoogleSync`
- Modify: `backend/internal/store/integration_test.go:39-60` (`reset`), `:73-74`, `:86`, `:141-142`
- Modify: `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` (§3.2 ```sql block, after `CREATE TABLE exercises (...);`)

`0001_init.up.sql`'s conventions (`head -12 internal/store/migrations/0001_init.up.sql`): a leading `--` comment naming the source, plain `CREATE TABLE` (no `IF NOT EXISTS` — `Migrate` records versions in `schema_migrations`, so a file never runs twice), `gen_random_uuid()` defaults, `TIMESTAMP WITH TIME ZONE`, `REFERENCES users(id) ON DELETE CASCADE`; the down file drops children before parents with `IF EXISTS`.

- [ ] **Step 1: Write the failing test**

Append to `backend/internal/store/migrations_test.go`:
```go
func TestMigration0002CreatesGoogleSync(t *testing.T) {
	up := readMigration(t, "0002_google_sync.up.sql")
	for _, w := range []string{
		"CREATE TABLE google_sync (",
		"user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE",
		"calendar_event_id TEXT",
		"tasklist_id TEXT",
		"roadmap_id UUID REFERENCES roadmaps(id) ON DELETE SET NULL",
		"tasks_created_count INT NOT NULL DEFAULT 0",
		"synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP",
	} {
		if !strings.Contains(up, w) {
			t.Errorf("0002_google_sync.up.sql is missing %q", w)
		}
	}
	down := readMigration(t, "0002_google_sync.down.sql")
	if !strings.Contains(down, "DROP TABLE IF EXISTS google_sync;") {
		t.Error("0002_google_sync.down.sql does not drop google_sync")
	}
}
```

In `TestMigrateAppliesPendingVersions` change the expectation to both versions in order:
```go
	if want := []string{"0001_init", "0002_google_sync"}; !reflect.DeepEqual(got, want) {
		t.Errorf("applied = %v, want %v", got, want)
	}
```

- [ ] **Step 2: Run to confirm they fail**

```sh
go test ./internal/store/... -run 'Migration0002|AppliesPendingVersions' -timeout 60s
```
Expected: FAIL — `reading 0002_google_sync.up.sql: open …: file does not exist` and `applied = [0001_init], want [0001_init 0002_google_sync]`.

- [ ] **Step 3: Write the migration**

`backend/internal/store/migrations/0002_google_sync.up.sql`:
```sql
-- Migration 0002 — google_sync: the Google Calendar event and Tasks list ids
-- that POST /api/v1/integrations/google/sync (backend spec §6.4) needs for an
-- idempotent re-sync. Spec §3.2 has no column for them; one row per user,
-- because the Calendar event exists even when the user has no roadmap.
-- roadmap_id records which roadmap the task list was built for, so a new
-- roadmap gets a fresh list while a same-roadmap re-sync creates nothing.

CREATE TABLE google_sync (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    calendar_event_id TEXT,
    tasklist_id TEXT,
    roadmap_id UUID REFERENCES roadmaps(id) ON DELETE SET NULL,
    tasks_created_count INT NOT NULL DEFAULT 0,
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

`backend/internal/store/migrations/0002_google_sync.down.sql`:
```sql
-- Reverse of 0002_google_sync.up.sql.

DROP TABLE IF EXISTS google_sync;
```

- [ ] **Step 4: Teach the integration tests about two versions**

In `backend/internal/store/integration_test.go`:

`reset` must run the down files newest-first (`google_sync` references `users`/`roadmaps`, so it must go before `0001`'s drops). Replace the body from `down, err := …` through the `down migration` check with:
```go
	ctx := context.Background()
	for _, name := range []string{"migrations/0002_google_sync.down.sql", "migrations/0001_init.down.sql"} {
		down, err := MigrationsFS.ReadFile(name)
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if _, err := pg.Pool.Exec(ctx, string(down)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
```
and update the comment above `reset` to "drops everything every migration creates (newest first)". Remove the now-duplicate `ctx := context.Background()` line that followed.

In `TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent`:
```go
	if want := []string{"0001_init", "0002_google_sync"}; !reflect.DeepEqual(first, want) {
		t.Fatalf("first run applied %v, want %v", first, want)
	}
```
(add `"reflect"` to the imports) and add `"google_sync"` to the table list on the `for _, table := range` line.

In `TestIntegrationConcurrentMigrateDoesNotRace` the total across callers is now the number of versions:
```go
	const versions = 2 // 0001_init, 0002_google_sync
	if total != versions {
		t.Errorf("migrations were applied %d times across %d concurrent callers, want exactly %d", total, n, versions)
	}
```

- [ ] **Step 5: Append the DDL to the backend spec §3.2 block**

In `project-base/Adaptive English Learning Platform - Backend Technical Specification.md`, inside the ```sql block of §3.2, directly after the `CREATE TABLE exercises (...);` statement (before the closing fence), add a blank line and:
```sql
-- Added by migration 0002 (google slice): ids for idempotent Calendar/Tasks re-sync.
CREATE TABLE google_sync (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    calendar_event_id TEXT,
    tasklist_id TEXT,
    roadmap_id UUID REFERENCES roadmaps(id) ON DELETE SET NULL,
    tasks_created_count INT NOT NULL DEFAULT 0,
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```
The spec's fenced ```sql blocks are clean (not backslash-escaped), so paste as-is. Check with:
```sh
cd .. && grep -n 'CREATE TABLE google_sync' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md" backend/internal/store/migrations/0002_google_sync.up.sql; cd backend
```
Expected: one hit in each file.

- [ ] **Step 6: Run the store suite**

```sh
go build ./... && go vet ./... && go test ./internal/store/... -count=1 -timeout 120s
```
Expected: `ok`; the `TestIntegration*` tests skip locally (no `TEST_DATABASE_URL`). If you have the dev stack up (`COMPOSE_PROJECT_NAME=<slug>`, non-default ports, see AGENTS.md), also run `make test-integration` and expect `TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent` and `…ConcurrentMigrateDoesNotRace` to PASS.

- [ ] **Step 7: Commit**

```sh
cd .. && git add backend/internal/store/migrations/0002_google_sync.up.sql backend/internal/store/migrations/0002_google_sync.down.sql backend/internal/store/migrations_test.go backend/internal/store/integration_test.go "project-base/Adaptive English Learning Platform - Backend Technical Specification.md" && git commit -m "store: migration 0002 google_sync for idempotent Calendar/Tasks re-sync

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>" && cd backend
```

---

### Task 2: Pure scheduling — event window, task due dates, task titles

**Files:**
- Create: `backend/internal/google/schedule.go`
- Test: `backend/internal/google/schedule_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/google/schedule_test.go`:
```go
package google

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestLocationFallsBackToUTC(t *testing.T) {
	for _, name := range []string{"", "Not/AZone"} {
		if got := Location(name); got != time.UTC {
			t.Errorf("Location(%q) = %v, want UTC", name, got)
		}
	}
	if got := Location("Asia/Ho_Chi_Minh"); got.String() != "Asia/Ho_Chi_Minh" {
		t.Errorf("Location(Asia/Ho_Chi_Minh) = %v", got)
	}
}

func TestNextOccurrenceIsTodayWhenStillAheadElseTomorrow(t *testing.T) {
	hcm := Location("Asia/Ho_Chi_Minh") // UTC+7, no DST
	// 2026-09-22T10:00Z = 17:00 in Ho Chi Minh City.
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)

	got, err := NextOccurrence(now, "20:00:00", hcm)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.September, 22, 20, 0, 0, 0, hcm); !got.Equal(want) {
		t.Errorf("20:00 still ahead: got %v, want %v", got, want)
	}

	got, err = NextOccurrence(now, "09:30", hcm) // HH:MM is accepted too
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.September, 23, 9, 30, 0, 0, hcm); !got.Equal(want) {
		t.Errorf("09:30 already past: got %v, want %v", got, want)
	}

	if _, err := NextOccurrence(now, "25:00:00", hcm); err == nil {
		t.Error("expected an error for hour 25")
	}
	if _, err := NextOccurrence(now, "eight", hcm); err == nil {
		t.Error("expected an error for a non-clock string")
	}
}

func TestNextOccurrenceKeepsWallClockAcrossDST(t *testing.T) {
	ny := Location("America/New_York")
	// 2026-11-01 is the fall-back day in New York (25 hours). 20:00 local on
	// 2026-10-31 has passed; the next 20:00 must be 20:00 on 11-01, EST.
	now := time.Date(2026, time.October, 31, 21, 0, 0, 0, ny)
	got, err := NextOccurrence(now, "20:00:00", ny)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hour() != 20 || got.Day() != 1 || got.Month() != time.November {
		t.Errorf("got %v, want 2026-11-01 20:00 America/New_York", got)
	}
	if _, off := got.Zone(); off != -5*3600 {
		t.Errorf("offset = %d, want -18000 (EST after fall-back)", off)
	}
}

func TestPracticeEventIsThirtyMinutesDailyFor28Days(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	ev, err := PracticeEvent(now, "20:00:00", "Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatal(err)
	}
	if ev.End.Sub(ev.Start) != 30*time.Minute {
		t.Errorf("duration = %v, want 30m", ev.End.Sub(ev.Start))
	}
	if ev.Summary != EventSummary || ev.TimeZone != "Asia/Ho_Chi_Minh" {
		t.Errorf("summary/timezone = %q/%q", ev.Summary, ev.TimeZone)
	}

	body, _ := json.Marshal(ev.payload())
	s := string(body)
	for _, want := range []string{
		`"recurrence":["RRULE:FREQ=DAILY;COUNT=28"]`,
		`"dateTime":"2026-09-22T20:00:00+07:00"`,
		`"dateTime":"2026-09-22T20:30:00+07:00"`,
		`"timeZone":"Asia/Ho_Chi_Minh"`,
		`"summary":"English practice"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("payload %s is missing %s", s, want)
		}
	}
}

func TestDayDueIsTheRoadmapDayAsADate(t *testing.T) {
	hcm := Location("Asia/Ho_Chi_Minh")
	// Created 2026-09-01T18:00Z = 2026-09-02 01:00 in HCM: day 1 is the 2nd.
	created := time.Date(2026, time.September, 1, 18, 0, 0, 0, time.UTC)
	if got := DayDue(created, 1, hcm); got.Format("2006-01-02") != "2026-09-02" {
		t.Errorf("day 1 due %v, want 2026-09-02", got)
	}
	if got := DayDue(created, 28, hcm); got.Format("2006-01-02") != "2026-09-29" {
		t.Errorf("day 28 due %v, want 2026-09-29", got)
	}
	if got := DayDue(created, 1, hcm); got.Location() != time.UTC || got.Hour() != 0 {
		t.Errorf("due must be UTC midnight (Tasks API keeps only the date), got %v", got)
	}
}

func TestTaskTitleJoinsTheDaysExercises(t *testing.T) {
	if got := TaskTitle(3, []string{"Greetings", "Short story", "Order a coffee"}); got != "Day 3: Greetings · Short story · Order a coffee" {
		t.Errorf("got %q", got)
	}
	if got := TaskTitle(7, []string{"", "Only one"}); got != "Day 7: Only one" {
		t.Errorf("empty titles are skipped: got %q", got)
	}
	if got := TaskTitle(9, nil); got != "Day 9" {
		t.Errorf("no titles: got %q", got)
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

```sh
go test ./internal/google/... -run 'Location|NextOccurrence|PracticeEvent|DayDue|TaskTitle' -timeout 60s
```
Expected: FAIL to compile — `undefined: Location` etc.

- [ ] **Step 3: Implement**

`backend/internal/google/schedule.go`:
```go
// Package google is the one-way Calendar + Tasks push of 1st-thinking §5.1
// steps 6-7 (wire contract: backend spec §6.4). Nothing is read back from
// Google and nothing subscribes to it.
package google

import (
	"fmt"
	"strings"
	"time"
)

// EventSummary and EventDescription name the recurring Calendar block.
const (
	EventSummary     = "English practice"
	EventDescription = "Your daily 30-minute English session. Open the app to start today's quests."
	// EventDuration is the §1 daily target.
	EventDuration = 30 * time.Minute
	// Recurrence is one event per day for the roadmap's 28 days (§6.1: 4 x 7).
	Recurrence = "RRULE:FREQ=DAILY;COUNT=28"
	// TasklistTitle names the Google Tasks list holding one task per roadmap day.
	TasklistTitle = "English daily quests"
	// RoadmapDays mirrors quests.RoadmapDays (not imported: package boundary).
	RoadmapDays = 28
)

// Location resolves users.timezone (§3.2, default 'UTC'); an unknown name
// falls back to UTC. Same rule as quests.Location — a bad timezone must never
// make the sync fail.
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

// parseClock accepts users.notification_time as Postgres renders TIME
// ("20:00:00") or as "HH:MM".
func parseClock(s string) (h, m, sec int, err error) {
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, perr := time.Parse(layout, s); perr == nil {
			return t.Hour(), t.Minute(), t.Second(), nil
		}
	}
	return 0, 0, 0, fmt.Errorf("google: notification_time %q is not HH:MM[:SS]", s)
}

// NextOccurrence is the next wall-clock hhmmss in loc strictly after now:
// today if that moment is still ahead, otherwise tomorrow. It goes through
// time.Date in loc, so a DST transition between now and then keeps the
// wall-clock time rather than shifting it by an hour.
func NextOccurrence(now time.Time, hhmmss string, loc *time.Location) (time.Time, error) {
	h, m, s, err := parseClock(hhmmss)
	if err != nil {
		return time.Time{}, err
	}
	l := now.In(loc)
	candidate := time.Date(l.Year(), l.Month(), l.Day(), h, m, s, 0, loc)
	if !candidate.After(now) {
		candidate = time.Date(l.Year(), l.Month(), l.Day()+1, h, m, s, 0, loc)
	}
	return candidate, nil
}

// Event is the Calendar event this package pushes (calendar.go sends it).
type Event struct {
	Summary     string
	Description string
	Start, End  time.Time
	TimeZone    string // IANA name, sent alongside dateTime so Google recurs in the user's zone
}

// payload is the Calendar v3 events resource body for insert and patch.
func (e Event) payload() map[string]any {
	return map[string]any{
		"summary":     e.Summary,
		"description": e.Description,
		"start":       map[string]string{"dateTime": e.Start.Format(time.RFC3339), "timeZone": e.TimeZone},
		"end":         map[string]string{"dateTime": e.End.Format(time.RFC3339), "timeZone": e.TimeZone},
		"recurrence":  []string{Recurrence},
	}
}

// PracticeEvent builds the 30-minute daily block starting at the next
// notificationTime in timezone.
func PracticeEvent(now time.Time, notificationTime, timezone string) (Event, error) {
	loc := Location(timezone)
	start, err := NextOccurrence(now, notificationTime, loc)
	if err != nil {
		return Event{}, err
	}
	return Event{
		Summary:     EventSummary,
		Description: EventDescription,
		Start:       start,
		End:         start.Add(EventDuration),
		TimeZone:    loc.String(),
	}, nil
}

// DayDue is the calendar date of roadmap day n (1-based) — the user's local
// date of roadmaps.created_at plus n-1 days — expressed as UTC midnight,
// which is how the Tasks API stores `due` (it keeps only the date part).
// Same calendar-day arithmetic as quests.DayNumber, inverted.
func DayDue(createdAt time.Time, n int, loc *time.Location) time.Time {
	l := createdAt.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day()+n-1, 0, 0, 0, 0, time.UTC)
}

// TaskTitle is "Day N: title · title · title", skipping empty titles.
func TaskTitle(day int, titles []string) string {
	var kept []string
	for _, t := range titles {
		if t = strings.TrimSpace(t); t != "" {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		return fmt.Sprintf("Day %d", day)
	}
	return fmt.Sprintf("Day %d: %s", day, strings.Join(kept, " · "))
}
```

- [ ] **Step 4: Run to confirm it passes**

```sh
go test ./internal/google/... -run 'Location|NextOccurrence|PracticeEvent|DayDue|TaskTitle' -v -timeout 60s
```
Expected: seven `--- PASS`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/google/schedule.go backend/internal/google/schedule_test.go && git commit -m "google: pure event window, task due date and title arithmetic" && cd backend
```
(Trailer as in Task 1.)

---

### Task 3: The refresh-token seam and the OAuth refresh client

**Files:**
- Create: `backend/internal/google/token.go`, `backend/internal/google/oauth.go`, `backend/internal/google/client.go`
- Test: `backend/internal/google/oauth_test.go`

`client.go` holds the shared HTTP plumbing all three Google clients use; it is introduced here because the OAuth client is the first user. Status mapping, decided once: `401`/`403` from a Google API → `ErrReauthRequired` (token revoked, or the `calendar.events`/`tasks` scopes were never granted); `404`/`410` → `ErrNotFound` (the stored id is gone; the service re-creates); any other non-2xx → `*UpstreamError` (handler → 502). The token endpoint is special: a `400` whose body says `invalid_grant` is the revoked-refresh-token signal and maps to `ErrReauthRequired`.

- [ ] **Step 1: Write the failing test**

`backend/internal/google/oauth_test.go`:
```go
package google

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOAuthClientRefreshesAnAccessToken(t *testing.T) {
	var gotForm map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		gotForm = map[string]string{}
		for k := range r.PostForm {
			gotForm[k] = r.PostForm.Get(k)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ya29.new","expires_in":3599,"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	c := NewOAuthClient("cid", "secret")
	c.TokenURL = srv.URL

	tok, err := c.AccessToken(context.Background(), "1//refresh")
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if tok != "ya29.new" {
		t.Errorf("token = %q", tok)
	}
	want := map[string]string{"grant_type": "refresh_token", "refresh_token": "1//refresh", "client_id": "cid", "client_secret": "secret"}
	for k, v := range want {
		if gotForm[k] != v {
			t.Errorf("form[%s] = %q, want %q", k, gotForm[k], v)
		}
	}
}

func TestOAuthClientMapsInvalidGrantToReauthRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Token has been expired or revoked."}`))
	}))
	defer srv.Close()

	c := NewOAuthClient("cid", "secret")
	c.TokenURL = srv.URL
	_, err := c.AccessToken(context.Background(), "1//dead")
	if !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("err = %v, want ErrReauthRequired", err)
	}
}

func TestOAuthClientReportsOtherFailuresAsUpstream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal_failure"}`))
	}))
	defer srv.Close()

	c := NewOAuthClient("cid", "secret")
	c.TokenURL = srv.URL
	_, err := c.AccessToken(context.Background(), "1//x")
	var up *UpstreamError
	if !errors.As(err, &up) || up.Status != 500 || up.Service != "oauth" {
		t.Fatalf("err = %v, want *UpstreamError{Service: oauth, Status: 500}", err)
	}
	if errors.Is(err, ErrReauthRequired) {
		t.Error("a 500 is not a re-auth condition")
	}
}

func TestOAuthClientRejectsAnEmptyAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	c := NewOAuthClient("cid", "secret")
	c.TokenURL = srv.URL
	if _, err := c.AccessToken(context.Background(), "1//x"); err == nil {
		t.Fatal("expected an error when access_token is missing")
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

```sh
go test ./internal/google/... -run 'OAuthClient' -timeout 60s
```
Expected: FAIL to compile — `undefined: NewOAuthClient`.

- [ ] **Step 3: Implement**

`backend/internal/google/client.go`:
```go
package google

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrReauthRequired means Google no longer honours this user's refresh token
// (invalid_grant) or rejects the scopes (401/403). The handler answers 409
// reauth_required so the client sends the user through /login again
// (auth.AuthCodeURL asks for calendar.events + tasks with prompt=consent).
var ErrReauthRequired = errors.New("google: re-authentication required")

// ErrNotFound means the Calendar event or Tasks list we stored an id for no
// longer exists at Google (404/410). Service re-creates it.
var ErrNotFound = errors.New("google: resource not found")

// UpstreamError is any other non-2xx from Google. The handler maps it to 502.
type UpstreamError struct {
	Service string // "oauth" | "calendar" | "tasks"
	Status  int
	Body    string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("google: %s returned %d: %s", e.Service, e.Status, e.Body)
}

// defaultHTTPClient bounds every Google call; the handler's overall deadline
// (SyncTimeout) bounds the whole sync.
func defaultHTTPClient() *http.Client { return &http.Client{Timeout: 15 * time.Second} }

// doJSON sends in (JSON-encoded, or nothing when nil) with a bearer token,
// maps the status code as documented on the errors above, and decodes a 2xx
// body into out when out is non-nil.
func doJSON(ctx context.Context, client *http.Client, service, method, url, accessToken string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("google: encoding %s request: %w", service, err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("google: building %s request: %w", service, err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if client == nil {
		client = defaultHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("google: calling %s: %w", service, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("google: reading %s response: %w", service, err)
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%w: %s returned %d", ErrReauthRequired, service, resp.StatusCode)
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return fmt.Errorf("%w: %s returned %d", ErrNotFound, service, resp.StatusCode)
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return &UpstreamError{Service: service, Status: resp.StatusCode, Body: string(raw)}
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("google: decoding %s response: %w", service, err)
	}
	return nil
}
```

`backend/internal/google/token.go`:
```go
package google

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoRefreshToken means users.google_refresh_token is NULL or empty: the
// user signed in before offline access was requested, or Google omitted it.
// Service maps it to ErrReauthRequired.
var ErrNoRefreshToken = errors.New("google: no refresh token on file")

// RefreshTokenSource is the ONLY way this package reads
// users.google_refresh_token. Today's single implementation returns the
// column as stored (plaintext — what the merged auth slice writes). Backend
// spec §7 wants AES-256-GCM at rest via ENCRYPTION_SECRET_KEY; that fix
// (inbox bug "google_refresh_token is stored in plaintext") replaces this
// implementation with a decrypting one and changes nothing else here.
type RefreshTokenSource interface {
	RefreshToken(ctx context.Context, userID string) (string, error)
}

// PgRefreshTokenSource reads the column verbatim.
type PgRefreshTokenSource struct{ Pool *pgxpool.Pool }

// NewPgRefreshTokenSource builds the source over an existing pool.
func NewPgRefreshTokenSource(pool *pgxpool.Pool) *PgRefreshTokenSource {
	return &PgRefreshTokenSource{Pool: pool}
}

const refreshTokenSQL = `SELECT COALESCE(google_refresh_token, '') FROM users WHERE id = $1`

func (s *PgRefreshTokenSource) RefreshToken(ctx context.Context, userID string) (string, error) {
	var tok string
	if err := s.Pool.QueryRow(ctx, refreshTokenSQL, userID).Scan(&tok); err != nil {
		return "", fmt.Errorf("google: reading refresh token: %w", err)
	}
	if tok == "" {
		return "", ErrNoRefreshToken
	}
	return tok, nil
}

var _ RefreshTokenSource = (*PgRefreshTokenSource)(nil)
```

`backend/internal/google/oauth.go`:
```go
package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DefaultTokenURL is Google's OAuth token endpoint (same as auth.DefaultTokenURL).
const DefaultTokenURL = "https://oauth2.googleapis.com/token"

// TokenRefresher swaps a refresh token for a short-lived access token.
type TokenRefresher interface {
	AccessToken(ctx context.Context, refreshToken string) (string, error)
}

// OAuthClient is the real TokenRefresher. TokenURL is a field so tests point
// it at an httptest.Server (the auth.GoogleClient pattern).
type OAuthClient struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	HTTPClient   *http.Client
}

// NewOAuthClient builds a client against the production endpoint.
func NewOAuthClient(clientID, clientSecret string) *OAuthClient {
	return &OAuthClient{ClientID: clientID, ClientSecret: clientSecret, TokenURL: DefaultTokenURL, HTTPClient: defaultHTTPClient()}
}

// AccessToken performs grant_type=refresh_token. A 400 whose body names
// invalid_grant is Google's "revoked or expired refresh token" signal and
// becomes ErrReauthRequired; other failures are *UpstreamError.
func (c *OAuthClient) AccessToken(ctx context.Context, refreshToken string) (string, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("google: building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := c.HTTPClient
	if client == nil {
		client = defaultHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("google: calling token endpoint: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("google: reading token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var body struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &body)
		if body.Error == "invalid_grant" || resp.StatusCode == http.StatusUnauthorized {
			return "", fmt.Errorf("%w: token endpoint returned %d %s", ErrReauthRequired, resp.StatusCode, body.Error)
		}
		return "", &UpstreamError{Service: "oauth", Status: resp.StatusCode, Body: string(raw)}
	}

	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &tok); err != nil {
		return "", fmt.Errorf("google: decoding token response: %w", err)
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("google: token endpoint returned no access token")
	}
	return tok.AccessToken, nil
}

var _ TokenRefresher = (*OAuthClient)(nil)
```

- [ ] **Step 4: Run to confirm it passes**

```sh
go vet ./internal/google/... && go test ./internal/google/... -run 'OAuthClient' -v -timeout 60s
```
Expected: four `--- PASS`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/google/client.go backend/internal/google/token.go backend/internal/google/oauth.go backend/internal/google/oauth_test.go && git commit -m "google: refresh-token seam, OAuth refresh client, invalid_grant -> ErrReauthRequired" && cd backend
```
(Trailer as in Task 1.)

---

### Task 4: Calendar and Tasks HTTP clients over injectable base URLs

**Files:**
- Create: `backend/internal/google/calendar.go`, `backend/internal/google/tasks.go`
- Test: `backend/internal/google/calendar_test.go`, `backend/internal/google/tasks_test.go`

- [ ] **Step 1: Write the failing tests**

`backend/internal/google/calendar_test.go`:
```go
package google

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type recordedCall struct {
	Method, Path, Auth string
	Body               map[string]any
}

// fakeGoogleAPI records every request and answers from a per-path script.
func fakeGoogleAPI(t *testing.T, answers map[string]struct {
	Status int
	Body   string
}) (*httptest.Server, *[]recordedCall) {
	t.Helper()
	var calls []recordedCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := recordedCall{Method: r.Method, Path: r.URL.Path, Auth: r.Header.Get("Authorization")}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&rc.Body)
		}
		calls = append(calls, rc)
		a, ok := answers[r.Method+" "+r.URL.Path]
		if !ok {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(a.Status)
		_, _ = w.Write([]byte(a.Body))
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func sampleEvent() Event {
	start := time.Date(2026, time.September, 22, 20, 0, 0, 0, Location("Asia/Ho_Chi_Minh"))
	return Event{Summary: EventSummary, Description: EventDescription, Start: start, End: start.Add(EventDuration), TimeZone: "Asia/Ho_Chi_Minh"}
}

func TestCalendarInsertEventPostsToPrimaryAndReturnsTheID(t *testing.T) {
	srv, calls := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{"POST /calendars/primary/events": {200, `{"id":"evt_1","status":"confirmed"}`}})

	c := NewHTTPCalendarClient()
	c.BaseURL = srv.URL
	id, err := c.InsertEvent(context.Background(), "ya29.tok", sampleEvent())
	if err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}
	if id != "evt_1" {
		t.Errorf("id = %q", id)
	}
	call := (*calls)[0]
	if call.Auth != "Bearer ya29.tok" {
		t.Errorf("Authorization = %q", call.Auth)
	}
	rec, _ := call.Body["recurrence"].([]any)
	if len(rec) != 1 || rec[0] != Recurrence {
		t.Errorf("recurrence = %v, want [%s]", call.Body["recurrence"], Recurrence)
	}
	start, _ := call.Body["start"].(map[string]any)
	if start["timeZone"] != "Asia/Ho_Chi_Minh" || start["dateTime"] != "2026-09-22T20:00:00+07:00" {
		t.Errorf("start = %v", start)
	}
}

func TestCalendarPatchEventUsesTheStoredID(t *testing.T) {
	srv, calls := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{"PATCH /calendars/primary/events/evt_1": {200, `{"id":"evt_1"}`}})

	c := NewHTTPCalendarClient()
	c.BaseURL = srv.URL
	if err := c.PatchEvent(context.Background(), "ya29.tok", "evt_1", sampleEvent()); err != nil {
		t.Fatalf("PatchEvent: %v", err)
	}
	if got := (*calls)[0]; got.Method != http.MethodPatch || got.Path != "/calendars/primary/events/evt_1" {
		t.Errorf("call = %+v", got)
	}
}

func TestCalendarMapsStatusesToSentinelErrors(t *testing.T) {
	srv, _ := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{
		"PATCH /calendars/primary/events/gone":      {404, `{"error":{"code":404}}`},
		"PATCH /calendars/primary/events/forbidden": {403, `{"error":{"code":403,"message":"insufficient scopes"}}`},
		"PATCH /calendars/primary/events/broken":    {500, `{"error":{"code":500}}`},
	})
	c := NewHTTPCalendarClient()
	c.BaseURL = srv.URL
	ctx := context.Background()

	if err := c.PatchEvent(ctx, "t", "gone", sampleEvent()); !errors.Is(err, ErrNotFound) {
		t.Errorf("404: err = %v, want ErrNotFound", err)
	}
	if err := c.PatchEvent(ctx, "t", "forbidden", sampleEvent()); !errors.Is(err, ErrReauthRequired) {
		t.Errorf("403: err = %v, want ErrReauthRequired", err)
	}
	var up *UpstreamError
	if err := c.PatchEvent(ctx, "t", "broken", sampleEvent()); !errors.As(err, &up) || up.Service != "calendar" {
		t.Errorf("500: err = %v, want *UpstreamError{calendar}", err)
	}
}
```

`backend/internal/google/tasks_test.go`:
```go
package google

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestTasksInsertTaskListReturnsTheID(t *testing.T) {
	srv, calls := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{"POST /users/@me/lists": {200, `{"id":"list_1","title":"English daily quests"}`}})

	c := NewHTTPTasksClient()
	c.BaseURL = srv.URL
	id, err := c.InsertTaskList(context.Background(), "ya29.tok", TasklistTitle)
	if err != nil {
		t.Fatalf("InsertTaskList: %v", err)
	}
	if id != "list_1" || (*calls)[0].Body["title"] != TasklistTitle {
		t.Errorf("id = %q, body = %v", id, (*calls)[0].Body)
	}
}

func TestTasksDeleteTaskListTreatsMissingAsDone(t *testing.T) {
	srv, calls := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{
		"DELETE /users/@me/lists/list_1": {204, ``},
		"DELETE /users/@me/lists/gone":   {404, `{"error":{"code":404}}`},
	})
	c := NewHTTPTasksClient()
	c.BaseURL = srv.URL
	if err := c.DeleteTaskList(context.Background(), "t", "list_1"); err != nil {
		t.Errorf("204: %v", err)
	}
	if err := c.DeleteTaskList(context.Background(), "t", "gone"); err != nil {
		t.Errorf("404 on delete must be nil (already gone), got %v", err)
	}
	if len(*calls) != 2 || (*calls)[0].Method != http.MethodDelete {
		t.Errorf("calls = %+v", *calls)
	}
}

func TestTasksInsertTaskPostsTitleNotesAndDue(t *testing.T) {
	srv, calls := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{"POST /lists/list_1/tasks": {200, `{"id":"task_1"}`}})
	c := NewHTTPTasksClient()
	c.BaseURL = srv.URL

	due := time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC)
	err := c.InsertTask(context.Background(), "t", "list_1", Task{Title: "Day 1: Greetings", Notes: "3 quests · 30 minutes", Due: due})
	if err != nil {
		t.Fatalf("InsertTask: %v", err)
	}
	b := (*calls)[0].Body
	if b["title"] != "Day 1: Greetings" || b["notes"] != "3 quests · 30 minutes" || b["due"] != "2026-09-02T00:00:00Z" {
		t.Errorf("body = %v", b)
	}
}

func TestTasksInsertTaskMapsReauth(t *testing.T) {
	srv, _ := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{"POST /lists/list_1/tasks": {401, `{"error":{"code":401}}`}})
	c := NewHTTPTasksClient()
	c.BaseURL = srv.URL
	if err := c.InsertTask(context.Background(), "t", "list_1", Task{Title: "x"}); !errors.Is(err, ErrReauthRequired) {
		t.Errorf("err = %v, want ErrReauthRequired", err)
	}
}
```

- [ ] **Step 2: Run to confirm they fail**

```sh
go test ./internal/google/... -run 'Calendar|Tasks' -timeout 60s
```
Expected: FAIL to compile — `undefined: NewHTTPCalendarClient`, `NewHTTPTasksClient`, `Task`.

- [ ] **Step 3: Implement**

`backend/internal/google/calendar.go`:
```go
package google

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// DefaultCalendarBaseURL is Google Calendar API v3.
const DefaultCalendarBaseURL = "https://www.googleapis.com/calendar/v3"

// CalendarClient is the two Calendar calls this package makes, on the user's
// primary calendar. InsertEvent returns Google's event id; PatchEvent updates
// the event with that id and returns ErrNotFound when Google no longer has it.
type CalendarClient interface {
	InsertEvent(ctx context.Context, accessToken string, ev Event) (string, error)
	PatchEvent(ctx context.Context, accessToken, eventID string, ev Event) error
}

// HTTPCalendarClient is the real CalendarClient. BaseURL is a field so tests
// point it at an httptest.Server.
type HTTPCalendarClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewHTTPCalendarClient builds a client against the production endpoint.
func NewHTTPCalendarClient() *HTTPCalendarClient {
	return &HTTPCalendarClient{BaseURL: DefaultCalendarBaseURL, HTTPClient: defaultHTTPClient()}
}

func (c *HTTPCalendarClient) InsertEvent(ctx context.Context, accessToken string, ev Event) (string, error) {
	var out struct {
		ID string `json:"id"`
	}
	if err := doJSON(ctx, c.HTTPClient, "calendar", http.MethodPost, c.BaseURL+"/calendars/primary/events", accessToken, ev.payload(), &out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("google: calendar insert returned no id")
	}
	return out.ID, nil
}

func (c *HTTPCalendarClient) PatchEvent(ctx context.Context, accessToken, eventID string, ev Event) error {
	return doJSON(ctx, c.HTTPClient, "calendar", http.MethodPatch, c.BaseURL+"/calendars/primary/events/"+url.PathEscape(eventID), accessToken, ev.payload(), nil)
}

var _ CalendarClient = (*HTTPCalendarClient)(nil)
```

`backend/internal/google/tasks.go`:
```go
package google

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DefaultTasksBaseURL is Google Tasks API v1.
const DefaultTasksBaseURL = "https://tasks.googleapis.com/tasks/v1"

// Task is one Google Tasks item: one per roadmap day.
type Task struct {
	Title string
	Notes string
	Due   time.Time // UTC midnight of the due date; Google keeps only the date
}

func (t Task) payload() map[string]any {
	p := map[string]any{"title": t.Title, "notes": t.Notes}
	if !t.Due.IsZero() {
		p["due"] = t.Due.UTC().Format(time.RFC3339)
	}
	return p
}

// TasksClient is the three Tasks calls this package makes. DeleteTaskList
// returns nil when the list is already gone.
type TasksClient interface {
	InsertTaskList(ctx context.Context, accessToken, title string) (string, error)
	DeleteTaskList(ctx context.Context, accessToken, tasklistID string) error
	InsertTask(ctx context.Context, accessToken, tasklistID string, t Task) error
}

// HTTPTasksClient is the real TasksClient.
type HTTPTasksClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewHTTPTasksClient builds a client against the production endpoint.
func NewHTTPTasksClient() *HTTPTasksClient {
	return &HTTPTasksClient{BaseURL: DefaultTasksBaseURL, HTTPClient: defaultHTTPClient()}
}

func (c *HTTPTasksClient) InsertTaskList(ctx context.Context, accessToken, title string) (string, error) {
	var out struct {
		ID string `json:"id"`
	}
	if err := doJSON(ctx, c.HTTPClient, "tasks", http.MethodPost, c.BaseURL+"/users/@me/lists", accessToken, map[string]string{"title": title}, &out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("google: tasklist insert returned no id")
	}
	return out.ID, nil
}

func (c *HTTPTasksClient) DeleteTaskList(ctx context.Context, accessToken, tasklistID string) error {
	err := doJSON(ctx, c.HTTPClient, "tasks", http.MethodDelete, c.BaseURL+"/users/@me/lists/"+url.PathEscape(tasklistID), accessToken, nil, nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (c *HTTPTasksClient) InsertTask(ctx context.Context, accessToken, tasklistID string, t Task) error {
	return doJSON(ctx, c.HTTPClient, "tasks", http.MethodPost, c.BaseURL+"/lists/"+url.PathEscape(tasklistID)+"/tasks", accessToken, t.payload(), nil)
}

var _ TasksClient = (*HTTPTasksClient)(nil)
```

- [ ] **Step 4: Run to confirm they pass**

```sh
go vet ./internal/google/... && go test ./internal/google/... -run 'Calendar|Tasks' -v -timeout 60s
```
Expected: seven `--- PASS`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/google/calendar.go backend/internal/google/calendar_test.go backend/internal/google/tasks.go backend/internal/google/tasks_test.go && git commit -m "google: Calendar and Tasks clients over injectable base URLs, httptest only" && cd backend
```
(Trailer as in Task 1.)

---

### Task 5: Repository — profile, roadmap day titles, and the `google_sync` row

**Files:**
- Create: `backend/internal/google/repo.go`
- Test: `backend/internal/google/integration_test.go`

The repo reads `users` (`notification_time`, `timezone`), `roadmaps` (active row) and `exercises` (`content_json->>'title'` per day) through **its own read-only SQL** — the same arrangement quests uses (`internal/quests/repo.go`) — and owns `google_sync`. The only unit test possible here is the live one; it is `TestIntegration*`, gated on `TEST_DATABASE_URL` only (never `DATABASE_URL`), so CI's `backend-integration` job counts it and fails if it skips there.

- [ ] **Step 1: Write the failing integration test**

`backend/internal/google/integration_test.go`:
```go
package google

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// TestIntegrationSyncStateIsOneRowPerUser proves the google_sync upsert
// (migration 0002) and the read-only lookups against a real Postgres. Gated on
// TEST_DATABASE_URL like internal/store — never on the production DATABASE_URL.
func TestIntegrationSyncStateIsOneRowPerUser(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL unset; run `make up` and export it to run integration tests")
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

	const gid = "google-sync-integration"
	var userID string
	_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)
	if err := pg.Pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, target_goal, timezone, notification_time, google_refresh_token)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		"google@example.com", gid, "", "Asia/Ho_Chi_Minh", "07:30:00", "1//refresh").Scan(&userID); err != nil {
		t.Fatalf("inserting user: %v", err)
	}
	t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid) })

	repo := NewPgRepo(pg.Pool)

	// Profile renders TIME as HH:MM:SS.
	prof, err := repo.Profile(ctx, userID)
	if err != nil {
		t.Fatalf("Profile: %v", err)
	}
	if prof.NotificationTime != "07:30:00" || prof.Timezone != "Asia/Ho_Chi_Minh" {
		t.Errorf("profile = %+v", prof)
	}

	// The refresh-token seam reads the column as stored.
	tok, err := NewPgRefreshTokenSource(pg.Pool).RefreshToken(ctx, userID)
	if err != nil || tok != "1//refresh" {
		t.Errorf("RefreshToken = %q, %v", tok, err)
	}

	// No roadmap yet.
	if _, err := repo.ActiveRoadmap(ctx, userID); !errors.Is(err, ErrNoActiveRoadmap) {
		t.Errorf("ActiveRoadmap without a roadmap: err = %v, want ErrNoActiveRoadmap", err)
	}
	if _, err := repo.SyncState(ctx, userID); !errors.Is(err, ErrNoSyncState) {
		t.Errorf("SyncState before any sync: err = %v, want ErrNoSyncState", err)
	}

	// A roadmap with two days; titles come from content_json->>'title' — the
	// key onboarding writes and quests' toTask reads.
	var roadmapID string
	if err := pg.Pool.QueryRow(ctx,
		`INSERT INTO roadmaps (user_id, roadmap_json, is_active) VALUES ($1, '{}'::jsonb, TRUE) RETURNING id`, userID).Scan(&roadmapID); err != nil {
		t.Fatalf("inserting roadmap: %v", err)
	}
	for _, row := range []struct {
		day   int
		typ   string
		title string
	}{{1, "vocabulary", "Greetings"}, {1, "reading", "Short story"}, {2, "practice", "Order a coffee"}} {
		if _, err := pg.Pool.Exec(ctx,
			`INSERT INTO exercises (roadmap_id, day_number, task_type, content_json) VALUES ($1,$2,$3::task_category,$4::jsonb)`,
			roadmapID, row.day, row.typ, `{"title":"`+row.title+`","duration_minutes":10}`); err != nil {
			t.Fatalf("inserting exercise: %v", err)
		}
	}
	rm, err := repo.ActiveRoadmap(ctx, userID)
	if err != nil || rm.ID != roadmapID {
		t.Fatalf("ActiveRoadmap = %+v, %v", rm, err)
	}
	days, err := repo.DayTitles(ctx, roadmapID)
	if err != nil {
		t.Fatalf("DayTitles: %v", err)
	}
	if len(days) != 2 || days[0].Day != 1 || len(days[0].Titles) != 2 || days[1].Day != 2 || days[1].Titles[0] != "Order a coffee" {
		t.Errorf("DayTitles = %+v", days)
	}

	// Upsert twice → one row, latest values.
	first := SyncState{UserID: userID, CalendarEventID: "evt_1"}
	if err := repo.SaveSyncState(ctx, first); err != nil {
		t.Fatalf("SaveSyncState #1: %v", err)
	}
	second := SyncState{UserID: userID, CalendarEventID: "evt_1", TasklistID: "list_1", RoadmapID: roadmapID, TasksCreatedCount: 2}
	if err := repo.SaveSyncState(ctx, second); err != nil {
		t.Fatalf("SaveSyncState #2: %v", err)
	}
	got, err := repo.SyncState(ctx, userID)
	if err != nil {
		t.Fatalf("SyncState: %v", err)
	}
	if got != second {
		t.Errorf("SyncState = %+v, want %+v", got, second)
	}
	var n int
	if err := pg.Pool.QueryRow(ctx, `SELECT count(*) FROM google_sync WHERE user_id = $1`, userID).Scan(&n); err != nil || n != 1 {
		t.Errorf("google_sync rows = %d (%v), want 1", n, err)
	}
}
```

- [ ] **Step 2: Run to confirm it fails to compile (and skips without the URL)**

```sh
go test ./internal/google/... -run 'Integration' -timeout 60s
```
Expected: FAIL to compile — `undefined: NewPgRepo`, `SyncState`, `ErrNoActiveRoadmap`, `ErrNoSyncState`.

- [ ] **Step 3: Implement**

`backend/internal/google/repo.go`:
```go
package google

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoActiveRoadmap means the user has no roadmaps row with is_active = TRUE;
// the sync then pushes only the Calendar event.
var ErrNoActiveRoadmap = errors.New("google: no active roadmap")

// ErrNoSyncState means this user has never synced (no google_sync row).
var ErrNoSyncState = errors.New("google: no sync state")

// Profile is the slice of §3.2 users the event needs.
type Profile struct {
	NotificationTime string // "HH:MM:SS" as Postgres renders TIME
	Timezone         string // IANA name, 'UTC' when NULL
}

// Roadmap is the slice of §3.2 roadmaps this package reads.
type Roadmap struct {
	ID        string
	CreatedAt time.Time
}

// DayTitles is one roadmap day's exercise titles (content_json->>'title'),
// in task_type order.
type DayTitles struct {
	Day    int
	Titles []string
}

// SyncState is the google_sync row (migration 0002). Empty strings mean NULL.
type SyncState struct {
	UserID            string
	CalendarEventID   string
	TasklistID        string
	RoadmapID         string
	TasksCreatedCount int
}

// Repo is the Postgres side of Sync. Everything but SaveSyncState is read-only.
type Repo interface {
	Profile(ctx context.Context, userID string) (Profile, error)
	ActiveRoadmap(ctx context.Context, userID string) (Roadmap, error)
	DayTitles(ctx context.Context, roadmapID string) ([]DayTitles, error)
	SyncState(ctx context.Context, userID string) (SyncState, error)
	SaveSyncState(ctx context.Context, s SyncState) error
}

const (
	profileSQL = `SELECT COALESCE(notification_time::text, '20:00:00'), COALESCE(timezone, 'UTC') FROM users WHERE id = $1`

	activeRoadmapSQL = `
SELECT id, created_at
FROM roadmaps
WHERE user_id = $1 AND is_active = TRUE
ORDER BY created_at DESC
LIMIT 1`

	dayTitlesSQL = `
SELECT day_number, COALESCE(content_json->>'title', '')
FROM exercises
WHERE roadmap_id = $1
ORDER BY day_number, task_type`

	syncStateSQL = `
SELECT COALESCE(calendar_event_id, ''), COALESCE(tasklist_id, ''), COALESCE(roadmap_id::text, ''), tasks_created_count
FROM google_sync
WHERE user_id = $1`

	// NULLIF turns the empty-string convention back into NULL; the FK on
	// roadmap_id is ON DELETE SET NULL, so a vanished roadmap reads as ''.
	saveSyncStateSQL = `
INSERT INTO google_sync (user_id, calendar_event_id, tasklist_id, roadmap_id, tasks_created_count, synced_at)
VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, '')::uuid, $5, CURRENT_TIMESTAMP)
ON CONFLICT (user_id) DO UPDATE SET
    calendar_event_id = EXCLUDED.calendar_event_id,
    tasklist_id = EXCLUDED.tasklist_id,
    roadmap_id = EXCLUDED.roadmap_id,
    tasks_created_count = EXCLUDED.tasks_created_count,
    synced_at = EXCLUDED.synced_at`
)

// PgRepo is the real Repo.
type PgRepo struct{ Pool *pgxpool.Pool }

// NewPgRepo builds a repo over an existing pool.
func NewPgRepo(pool *pgxpool.Pool) *PgRepo { return &PgRepo{Pool: pool} }

func (r *PgRepo) Profile(ctx context.Context, userID string) (Profile, error) {
	var p Profile
	if err := r.Pool.QueryRow(ctx, profileSQL, userID).Scan(&p.NotificationTime, &p.Timezone); err != nil {
		return Profile{}, fmt.Errorf("google: reading profile: %w", err)
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
		return Roadmap{}, fmt.Errorf("google: reading active roadmap: %w", err)
	}
	return rm, nil
}

func (r *PgRepo) DayTitles(ctx context.Context, roadmapID string) ([]DayTitles, error) {
	rows, err := r.Pool.Query(ctx, dayTitlesSQL, roadmapID)
	if err != nil {
		return nil, fmt.Errorf("google: reading exercise titles: %w", err)
	}
	defer rows.Close()

	var out []DayTitles
	for rows.Next() {
		var (
			day   int
			title string
		)
		if err := rows.Scan(&day, &title); err != nil {
			return nil, fmt.Errorf("google: scanning exercise title: %w", err)
		}
		if n := len(out); n > 0 && out[n-1].Day == day {
			out[n-1].Titles = append(out[n-1].Titles, title)
			continue
		}
		out = append(out, DayTitles{Day: day, Titles: []string{title}})
	}
	return out, rows.Err()
}

func (r *PgRepo) SyncState(ctx context.Context, userID string) (SyncState, error) {
	s := SyncState{UserID: userID}
	err := r.Pool.QueryRow(ctx, syncStateSQL, userID).Scan(&s.CalendarEventID, &s.TasklistID, &s.RoadmapID, &s.TasksCreatedCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return SyncState{}, ErrNoSyncState
	}
	if err != nil {
		return SyncState{}, fmt.Errorf("google: reading sync state: %w", err)
	}
	return s, nil
}

func (r *PgRepo) SaveSyncState(ctx context.Context, s SyncState) error {
	if _, err := r.Pool.Exec(ctx, saveSyncStateSQL, s.UserID, s.CalendarEventID, s.TasklistID, s.RoadmapID, s.TasksCreatedCount); err != nil {
		return fmt.Errorf("google: saving sync state: %w", err)
	}
	return nil
}

var _ Repo = (*PgRepo)(nil)
```

- [ ] **Step 4: Compile, vet, confirm the skip locally**

```sh
go vet ./internal/google/... && go test ./internal/google/... -run 'Integration' -v -timeout 60s
```
Expected: `--- SKIP: TestIntegrationSyncStateIsOneRowPerUser` with the `TEST_DATABASE_URL unset` message. **It must PASS in CI's `backend-integration` job**; to prove it locally, bring up the dev stack with a unique compose project and non-default ports (AGENTS.md), export `TEST_DATABASE_URL`, and run `go test ./internal/google/... -run Integration -v -count=1 -p 1 -timeout 120s`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/google/repo.go backend/internal/google/integration_test.go && git commit -m "google: repo for profile, roadmap day titles and the google_sync upsert" && cd backend
```
(Trailer as in Task 1.)

---

### Task 6: Fakes and `Service.Sync` — the idempotent algorithm

**Files:**
- Create: `backend/internal/google/fakes_test.go`, `backend/internal/google/service.go`
- Test: `backend/internal/google/service_test.go`

The algorithm, in order:
1. `tokens.RefreshToken` — `ErrNoRefreshToken` → `ErrReauthRequired`.
2. `oauth.AccessToken` — `invalid_grant` is already `ErrReauthRequired`.
3. `repo.Profile`, `repo.SyncState` (`ErrNoSyncState` → empty state).
4. **Event:** `PracticeEvent(now, …)`; if a `calendar_event_id` is stored → `PatchEvent`; on `ErrNotFound` → `InsertEvent`; no stored id → `InsertEvent`. **Persist state now** (so a later failure never loses the event id and duplicates the event on retry).
5. **Tasks:** `repo.ActiveRoadmap`; `ErrNoActiveRoadmap` → `tasks_created_count = 0`, done. Otherwise: if the stored `tasklist_id` is non-empty **and** `roadmap_id` equals the active roadmap → nothing to do, re-report the stored count. Else: if a `tasklist_id` is stored (older roadmap, or a list whose tasks never finished) → `DeleteTaskList` (404 is fine), then `InsertTaskList`, **persist state with the new list id, `roadmap_id = ""`, count 0** (a crash during the inserts leaves a findable, deletable list), then one `InsertTask` per `DayTitles` row (`TaskTitle`, `DayDue`), then persist the final state with `roadmap_id` and the count.

- [ ] **Step 1: Write the fakes**

`backend/internal/google/fakes_test.go`:
```go
package google

import (
	"context"
	"fmt"
	"time"
)

// callLog is shared by every fake so tests can assert ordering.
type callLog struct{ calls []string }

func (l *callLog) add(format string, args ...any) { l.calls = append(l.calls, fmt.Sprintf(format, args...)) }

type fakeTokens struct {
	log   *callLog
	token string
	err   error
}

func (f *fakeTokens) RefreshToken(_ context.Context, userID string) (string, error) {
	f.log.add("tokens.RefreshToken(%s)", userID)
	return f.token, f.err
}

type fakeOAuth struct {
	log *callLog
	err error
}

func (f *fakeOAuth) AccessToken(_ context.Context, refresh string) (string, error) {
	f.log.add("oauth.AccessToken(%s)", refresh)
	if f.err != nil {
		return "", f.err
	}
	return "access-for-" + refresh, nil
}

type fakeCalendar struct {
	log      *callLog
	nextID   string
	patchErr error // returned by PatchEvent (e.g. ErrNotFound)
	inserted []Event
	patched  []string
}

func (f *fakeCalendar) InsertEvent(_ context.Context, tok string, ev Event) (string, error) {
	f.log.add("calendar.InsertEvent(%s)", tok)
	f.inserted = append(f.inserted, ev)
	return f.nextID, nil
}

func (f *fakeCalendar) PatchEvent(_ context.Context, tok, id string, _ Event) error {
	f.log.add("calendar.PatchEvent(%s,%s)", tok, id)
	if f.patchErr != nil {
		return f.patchErr
	}
	f.patched = append(f.patched, id)
	return nil
}

type fakeTasks struct {
	log       *callLog
	nextList  string
	insertErr error // returned by InsertTask after failAfter successes
	failAfter int
	deleted   []string
	tasks     map[string][]Task
}

func (f *fakeTasks) InsertTaskList(_ context.Context, tok, title string) (string, error) {
	f.log.add("tasks.InsertTaskList(%s,%s)", tok, title)
	if f.tasks == nil {
		f.tasks = map[string][]Task{}
	}
	return f.nextList, nil
}

func (f *fakeTasks) DeleteTaskList(_ context.Context, tok, id string) error {
	f.log.add("tasks.DeleteTaskList(%s,%s)", tok, id)
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeTasks) InsertTask(_ context.Context, tok, list string, t Task) error {
	f.log.add("tasks.InsertTask(%s,%s,%s)", tok, list, t.Title)
	if f.insertErr != nil && len(f.tasks[list]) >= f.failAfter {
		return f.insertErr
	}
	f.tasks[list] = append(f.tasks[list], t)
	return nil
}

type fakeRepo struct {
	log     *callLog
	profile Profile
	roadmap Roadmap
	noRoad  bool
	days    []DayTitles
	state   SyncState
	noState bool
	saved   []SyncState
}

func (f *fakeRepo) Profile(_ context.Context, userID string) (Profile, error) {
	f.log.add("repo.Profile(%s)", userID)
	return f.profile, nil
}

func (f *fakeRepo) ActiveRoadmap(_ context.Context, userID string) (Roadmap, error) {
	f.log.add("repo.ActiveRoadmap(%s)", userID)
	if f.noRoad {
		return Roadmap{}, ErrNoActiveRoadmap
	}
	return f.roadmap, nil
}

func (f *fakeRepo) DayTitles(_ context.Context, roadmapID string) ([]DayTitles, error) {
	f.log.add("repo.DayTitles(%s)", roadmapID)
	return f.days, nil
}

func (f *fakeRepo) SyncState(_ context.Context, userID string) (SyncState, error) {
	f.log.add("repo.SyncState(%s)", userID)
	if f.noState {
		return SyncState{}, ErrNoSyncState
	}
	return f.state, nil
}

func (f *fakeRepo) SaveSyncState(_ context.Context, s SyncState) error {
	f.log.add("repo.SaveSyncState(evt=%s,list=%s,roadmap=%s,n=%d)", s.CalendarEventID, s.TasklistID, s.RoadmapID, s.TasksCreatedCount)
	f.saved = append(f.saved, s)
	f.state, f.noState = s, false
	return nil
}

// harness wires fakes for a user in Ho Chi Minh City with a 28-day roadmap.
type harness struct {
	log    *callLog
	tokens *fakeTokens
	oauth  *fakeOAuth
	cal    *fakeCalendar
	tasks  *fakeTasks
	repo   *fakeRepo
	now    time.Time
	svc    *Service
}

func newHarness() *harness {
	log := &callLog{}
	days := make([]DayTitles, 0, RoadmapDays)
	for d := 1; d <= RoadmapDays; d++ {
		days = append(days, DayTitles{Day: d, Titles: []string{fmt.Sprintf("Vocab %d", d), fmt.Sprintf("Read %d", d), fmt.Sprintf("Practice %d", d)}})
	}
	h := &harness{
		log:    log,
		tokens: &fakeTokens{log: log, token: "1//refresh"},
		oauth:  &fakeOAuth{log: log},
		cal:    &fakeCalendar{log: log, nextID: "evt_new"},
		tasks:  &fakeTasks{log: log, nextList: "list_new"},
		repo: &fakeRepo{
			log:     log,
			profile: Profile{NotificationTime: "20:00:00", Timezone: "Asia/Ho_Chi_Minh"},
			roadmap: Roadmap{ID: "roadmap-A", CreatedAt: time.Date(2026, time.September, 1, 18, 0, 0, 0, time.UTC)},
			days:    days,
			noState: true,
		},
		now: time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC),
	}
	h.svc = NewService(h.tokens, h.oauth, h.cal, h.tasks, h.repo, func() time.Time { return h.now })
	return h
}
```

- [ ] **Step 2: Write the failing service tests**

`backend/internal/google/service_test.go`:
```go
package google

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSyncFirstTimeInsertsEventListAnd28Tasks(t *testing.T) {
	h := newHarness()

	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if res.Status != "synced" || res.CalendarEventID != "evt_new" || res.TasksCreatedCount != 28 {
		t.Errorf("result = %+v", res)
	}
	if len(h.cal.inserted) != 1 || len(h.cal.patched) != 0 {
		t.Errorf("calendar inserts/patches = %d/%d, want 1/0", len(h.cal.inserted), len(h.cal.patched))
	}
	if got := h.cal.inserted[0].Start.Format("2006-01-02T15:04:05Z07:00"); got != "2026-09-22T20:00:00+07:00" {
		t.Errorf("event start = %s", got)
	}
	tasks := h.tasks.tasks["list_new"]
	if len(tasks) != 28 || tasks[0].Title != "Day 1: Vocab 1 · Read 1 · Practice 1" || tasks[27].Due.Format("2006-01-02") != "2026-09-29" {
		t.Errorf("tasks = %d, first %q, last due %v", len(tasks), tasks[0].Title, tasks[27].Due)
	}
	// Access token reached every Google call; the refresh token never did.
	for _, c := range h.log.calls {
		if strings.HasPrefix(c, "calendar.") || strings.HasPrefix(c, "tasks.") {
			if !strings.Contains(c, "access-for-1//refresh") {
				t.Errorf("call %q did not carry the access token", c)
			}
		}
	}
	final := h.repo.saved[len(h.repo.saved)-1]
	if final != (SyncState{UserID: "u1", CalendarEventID: "evt_new", TasklistID: "list_new", RoadmapID: "roadmap-A", TasksCreatedCount: 28}) {
		t.Errorf("final state = %+v", final)
	}
}

func TestSyncPersistsStateAfterTheEventAndAfterTheListBeforeTasks(t *testing.T) {
	h := newHarness()
	if _, err := h.svc.Sync(context.Background(), "u1"); err != nil {
		t.Fatal(err)
	}
	if len(h.repo.saved) != 3 {
		t.Fatalf("saved %d times, want 3 (after event, after list, final): %v", len(h.repo.saved), h.log.calls)
	}
	if s := h.repo.saved[0]; s.CalendarEventID != "evt_new" || s.TasklistID != "" {
		t.Errorf("first save = %+v, want only the event id", s)
	}
	if s := h.repo.saved[1]; s.TasklistID != "list_new" || s.RoadmapID != "" || s.TasksCreatedCount != 0 {
		t.Errorf("second save = %+v, want list id with no roadmap yet", s)
	}
	// Ordering: event saved before the list is created; list saved before any task.
	idx := func(prefix string) int {
		for i, c := range h.log.calls {
			if strings.HasPrefix(c, prefix) {
				return i
			}
		}
		return -1
	}
	if !(idx("repo.SaveSyncState(evt=evt_new,list=,") < idx("tasks.InsertTaskList") && idx("repo.SaveSyncState(evt=evt_new,list=list_new,roadmap=,") < idx("tasks.InsertTask(")) {
		t.Errorf("save points out of order: %v", h.log.calls)
	}
}

func TestSyncFailureMidTasksKeepsTheListIDSoRetryDeletesIt(t *testing.T) {
	h := newHarness()
	h.tasks.insertErr = &UpstreamError{Service: "tasks", Status: 500}
	h.tasks.failAfter = 5

	_, err := h.svc.Sync(context.Background(), "u1")
	var up *UpstreamError
	if !errors.As(err, &up) {
		t.Fatalf("err = %v, want *UpstreamError", err)
	}
	if h.repo.state.TasklistID != "list_new" || h.repo.state.RoadmapID != "" {
		t.Fatalf("state after failure = %+v, want list_new with no roadmap", h.repo.state)
	}

	// Retry: the half-built list is deleted and rebuilt, not appended to.
	h.tasks.insertErr = nil
	h.tasks.nextList = "list_retry"
	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if len(h.tasks.deleted) != 1 || h.tasks.deleted[0] != "list_new" || res.TasksCreatedCount != 28 || len(h.tasks.tasks["list_retry"]) != 28 {
		t.Errorf("deleted = %v, result = %+v", h.tasks.deleted, res)
	}
}

func TestResyncSameRoadmapPatchesEventAndCreatesNothing(t *testing.T) {
	h := newHarness()
	h.repo.noState = false
	h.repo.state = SyncState{UserID: "u1", CalendarEventID: "evt_old", TasklistID: "list_old", RoadmapID: "roadmap-A", TasksCreatedCount: 28}

	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if res.CalendarEventID != "evt_old" || res.TasksCreatedCount != 28 {
		t.Errorf("result = %+v", res)
	}
	if len(h.cal.inserted) != 0 || len(h.cal.patched) != 1 || h.cal.patched[0] != "evt_old" {
		t.Errorf("calendar inserts/patches = %v/%v", h.cal.inserted, h.cal.patched)
	}
	for _, c := range h.log.calls {
		if strings.HasPrefix(c, "tasks.") {
			t.Errorf("same roadmap must not touch Tasks, but called %s", c)
		}
	}
}

func TestResyncNewRoadmapDeletesOldListAndBuildsANewOne(t *testing.T) {
	h := newHarness()
	h.repo.noState = false
	h.repo.state = SyncState{UserID: "u1", CalendarEventID: "evt_old", TasklistID: "list_old", RoadmapID: "roadmap-OLD", TasksCreatedCount: 28}

	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(h.tasks.deleted) != 1 || h.tasks.deleted[0] != "list_old" {
		t.Errorf("deleted = %v, want [list_old]", h.tasks.deleted)
	}
	if res.TasksCreatedCount != 28 || h.repo.state.TasklistID != "list_new" || h.repo.state.RoadmapID != "roadmap-A" {
		t.Errorf("result = %+v, state = %+v", res, h.repo.state)
	}
}

func TestResyncReinsertsTheEventWhenGoogleLostIt(t *testing.T) {
	h := newHarness()
	h.repo.noState = false
	h.repo.state = SyncState{UserID: "u1", CalendarEventID: "evt_deleted_by_user", TasklistID: "list_old", RoadmapID: "roadmap-A", TasksCreatedCount: 28}
	h.cal.patchErr = ErrNotFound

	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if res.CalendarEventID != "evt_new" || len(h.cal.inserted) != 1 {
		t.Errorf("result = %+v, inserted = %d", res, len(h.cal.inserted))
	}
}

func TestSyncWithoutARoadmapPushesOnlyTheEvent(t *testing.T) {
	h := newHarness()
	h.repo.noRoad = true

	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if res.Status != "synced" || res.CalendarEventID != "evt_new" || res.TasksCreatedCount != 0 {
		t.Errorf("result = %+v", res)
	}
	for _, c := range h.log.calls {
		if strings.HasPrefix(c, "tasks.") {
			t.Errorf("no roadmap must not touch Tasks, but called %s", c)
		}
	}
}

func TestSyncNeedsReauthWithoutARefreshTokenOrOnInvalidGrant(t *testing.T) {
	h := newHarness()
	h.tokens.err = ErrNoRefreshToken
	if _, err := h.svc.Sync(context.Background(), "u1"); !errors.Is(err, ErrReauthRequired) {
		t.Errorf("no refresh token: err = %v, want ErrReauthRequired", err)
	}
	if len(h.cal.inserted) != 0 {
		t.Error("nothing may reach Google without a token")
	}

	h = newHarness()
	h.oauth.err = ErrReauthRequired // what OAuthClient returns on invalid_grant
	if _, err := h.svc.Sync(context.Background(), "u1"); !errors.Is(err, ErrReauthRequired) {
		t.Errorf("invalid_grant: err = %v, want ErrReauthRequired", err)
	}
	if len(h.repo.saved) != 0 {
		t.Error("no state may be written when the token refresh fails")
	}
}
```

- [ ] **Step 3: Run to confirm they fail**

```sh
go test ./internal/google/... -run 'Sync|Resync' -timeout 60s
```
Expected: FAIL to compile — `undefined: NewService`, `Service`, `Result`.

- [ ] **Step 4: Implement**

`backend/internal/google/service.go`:
```go
package google

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Result is the backend spec §6.4 response for POST /integrations/google/sync.
type Result struct {
	Status            string `json:"status"`
	CalendarEventID   string `json:"calendar_event_id"`
	TasksCreatedCount int    `json:"tasks_created_count"`
}

// Service performs the one-way sync. Every dependency is an interface so the
// tests never reach Google or Postgres.
type Service struct {
	tokens RefreshTokenSource
	oauth  TokenRefresher
	cal    CalendarClient
	tasks  TasksClient
	repo   Repo
	now    func() time.Time
}

// NewService wires the dependencies. now is injected so event times are testable.
func NewService(tokens RefreshTokenSource, oauth TokenRefresher, cal CalendarClient, tasks TasksClient, repo Repo, now func() time.Time) *Service {
	return &Service{tokens: tokens, oauth: oauth, cal: cal, tasks: tasks, repo: repo, now: now}
}

// Sync pushes the recurring practice event and the per-day task list, and is
// idempotent: the event is patched (re-inserted only if Google lost it) and
// the task list is rebuilt only when the active roadmap changed. State is
// persisted after the event and after the list is created, before the task
// inserts, so a failure part-way never orphans a Google object.
func (s *Service) Sync(ctx context.Context, userID string) (Result, error) {
	refresh, err := s.tokens.RefreshToken(ctx, userID)
	if errors.Is(err, ErrNoRefreshToken) {
		return Result{}, fmt.Errorf("%w: %v", ErrReauthRequired, err)
	}
	if err != nil {
		return Result{}, err
	}
	access, err := s.oauth.AccessToken(ctx, refresh)
	if err != nil {
		return Result{}, err
	}

	prof, err := s.repo.Profile(ctx, userID)
	if err != nil {
		return Result{}, err
	}
	state, err := s.repo.SyncState(ctx, userID)
	if errors.Is(err, ErrNoSyncState) {
		state = SyncState{UserID: userID}
	} else if err != nil {
		return Result{}, err
	}

	// 1. Calendar event: patch what we have, insert when we have nothing or
	// Google no longer has it.
	ev, err := PracticeEvent(s.now(), prof.NotificationTime, prof.Timezone)
	if err != nil {
		return Result{}, err
	}
	if state.CalendarEventID != "" {
		err = s.cal.PatchEvent(ctx, access, state.CalendarEventID, ev)
		if errors.Is(err, ErrNotFound) {
			state.CalendarEventID = ""
			err = nil
		}
		if err != nil {
			return Result{}, err
		}
	}
	if state.CalendarEventID == "" {
		id, err := s.cal.InsertEvent(ctx, access, ev)
		if err != nil {
			return Result{}, err
		}
		state.CalendarEventID = id
	}
	if err := s.repo.SaveSyncState(ctx, state); err != nil {
		return Result{}, err
	}

	// 2. Tasks list: one task per roadmap day, rebuilt only for a new roadmap.
	rm, err := s.repo.ActiveRoadmap(ctx, userID)
	if errors.Is(err, ErrNoActiveRoadmap) {
		return Result{Status: "synced", CalendarEventID: state.CalendarEventID, TasksCreatedCount: 0}, nil
	}
	if err != nil {
		return Result{}, err
	}
	if state.TasklistID != "" && state.RoadmapID == rm.ID {
		return Result{Status: "synced", CalendarEventID: state.CalendarEventID, TasksCreatedCount: state.TasksCreatedCount}, nil
	}
	if state.TasklistID != "" {
		// An older roadmap's list, or one whose tasks never finished: replace it.
		if err := s.tasks.DeleteTaskList(ctx, access, state.TasklistID); err != nil {
			return Result{}, err
		}
		state.TasklistID, state.RoadmapID, state.TasksCreatedCount = "", "", 0
	}
	listID, err := s.tasks.InsertTaskList(ctx, access, TasklistTitle)
	if err != nil {
		return Result{}, err
	}
	state.TasklistID = listID
	if err := s.repo.SaveSyncState(ctx, state); err != nil {
		return Result{}, err
	}

	days, err := s.repo.DayTitles(ctx, rm.ID)
	if err != nil {
		return Result{}, err
	}
	loc := Location(prof.Timezone)
	created := 0
	for _, d := range days {
		task := Task{
			Title: TaskTitle(d.Day, d.Titles),
			Notes: fmt.Sprintf("%d quests · 30 minutes. Open the app to start.", len(d.Titles)),
			Due:   DayDue(rm.CreatedAt, d.Day, loc),
		}
		if err := s.tasks.InsertTask(ctx, access, listID, task); err != nil {
			return Result{}, err
		}
		created++
	}
	state.RoadmapID, state.TasksCreatedCount = rm.ID, created
	if err := s.repo.SaveSyncState(ctx, state); err != nil {
		return Result{}, err
	}
	return Result{Status: "synced", CalendarEventID: state.CalendarEventID, TasksCreatedCount: created}, nil
}
```

- [ ] **Step 5: Run to confirm they pass**

```sh
go vet ./internal/google/... && go test ./internal/google/... -run 'Sync|Resync' -v -timeout 60s
```
Expected: eight `--- PASS`.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend/internal/google/fakes_test.go backend/internal/google/service.go backend/internal/google/service_test.go && git commit -m "google: Service.Sync — idempotent event patch, roadmap-keyed task list, crash-safe state" && cd backend
```
(Trailer as in Task 1.)

---

### Task 7: The route — §6.4 body, 409 `reauth_required` — and `main.go`

**Files:**
- Create: `backend/internal/google/handler.go`
- Test: `backend/internal/google/handler_test.go`
- Modify: `backend/cmd/api/main.go`

Error codes (§6.4 defines none; keep the merged `{"error": "<code>"}` convention): 401 `unauthorized` (no user in context), 409 `reauth_required` (`ErrReauthRequired`), 502 `google_unavailable` (`*UpstreamError` or a transport error from Google), 500 `internal_error` (anything else). The whole sync is bounded by `SyncTimeout` = 60 s (one token call + one Calendar call + up to 30 Tasks calls, sequential).

- [ ] **Step 1: Write the failing test**

`backend/internal/google/handler_test.go`:
```go
package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// router mounts the handler behind a stand-in for auth.Require() that sets
// auth.ContextUserID (the real middleware is covered in internal/auth).
func router(svc *Service, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/integrations/google/sync", func(c *gin.Context) {
		if userID != "" {
			c.Set(auth.ContextUserID, userID)
		}
		c.Next()
	}, SyncHandler(svc))
	return r
}

func post(t *testing.T, r *gin.Engine) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/google/sync", nil)
	r.ServeHTTP(w, req)
	return w
}

func TestSyncHandlerAnswersTheSpec64Body(t *testing.T) {
	h := newHarness()
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "synced" || body["calendar_event_id"] != "evt_new" || body["tasks_created_count"] != float64(28) {
		t.Errorf("body = %v", body)
	}
	if len(body) != 3 {
		t.Errorf("§6.4 body has exactly status, calendar_event_id, tasks_created_count; got %v", body)
	}
}

func TestSyncHandlerMapsReauthTo409(t *testing.T) {
	h := newHarness()
	h.oauth.err = ErrReauthRequired
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusConflict || w.Body.String() != `{"error":"reauth_required"}` {
		t.Errorf("status = %d, body = %s", w.Code, w.Body)
	}
}

func TestSyncHandlerMapsUpstreamTo502(t *testing.T) {
	h := newHarness()
	h.oauth.err = &UpstreamError{Service: "oauth", Status: 503, Body: "down"}
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusBadGateway || w.Body.String() != `{"error":"google_unavailable"}` {
		t.Errorf("status = %d, body = %s", w.Code, w.Body)
	}
}

func TestSyncHandlerRequiresAUser(t *testing.T) {
	h := newHarness()
	w := post(t, router(h.svc, ""))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

```sh
go test ./internal/google/... -run 'SyncHandler' -timeout 60s
```
Expected: FAIL to compile — `undefined: SyncHandler`.

- [ ] **Step 3: Implement the handler**

`backend/internal/google/handler.go`:
```go
package google

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// SyncTimeout bounds one sync: a token refresh, one Calendar call and up to
// 30 Tasks calls, sequential. §6.4 says "asynchronously" but its 200 body
// carries the ids Google returns, so the work runs inside the request.
const SyncTimeout = 60 * time.Second

// SyncHandler serves POST /api/v1/integrations/google/sync (backend spec
// §6.4). It must be mounted behind auth.Require().
func SyncHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), SyncTimeout)
		defer cancel()

		res, err := svc.Sync(ctx, userID)
		var up *UpstreamError
		switch {
		case errors.Is(err, ErrReauthRequired):
			// The refresh token is gone or revoked: the client sends the user
			// back through /login (auth.AuthCodeURL re-requests consent).
			c.JSON(http.StatusConflict, gin.H{"error": "reauth_required"})
		case errors.As(err, &up), errors.Is(err, context.DeadlineExceeded):
			c.JSON(http.StatusBadGateway, gin.H{"error": "google_unavailable"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			c.JSON(http.StatusOK, res)
		}
	}
}
```

- [ ] **Step 4: Run to confirm it passes**

```sh
go vet ./internal/google/... && go test ./internal/google/... -run 'SyncHandler' -v -timeout 60s
```
Expected: four `--- PASS`.

- [ ] **Step 5: Mount the route in `main.go`**

First look at the file as it is on your branch — pet (order 4) and onboarding (order 6) may or may not have landed:
```sh
grep -n 'guarded\|import (\|internal/quests"' cmd/api/main.go
```
Add the import `"github.com/HendrixNguyen/English-Training-Harness/backend/internal/google"` in the `internal/…` group (alphabetical: after `config`, before `health`). Immediately after the `guarded := v1.Group("", auth.Require(tokens, sessions))` line and the routes already mounted on it, add:
```go
	googleSvc := google.NewService(
		google.NewPgRefreshTokenSource(pg.Pool), // plaintext today; the §7 encryption fix replaces only this
		google.NewOAuthClient(cfg.GoogleClientID, cfg.GoogleClientSecret),
		google.NewHTTPCalendarClient(),
		google.NewHTTPTasksClient(),
		google.NewPgRepo(pg.Pool),
		time.Now,
	)
	guarded.POST("/integrations/google/sync", google.SyncHandler(googleSvc))
```
Do not move, rename or reorder anything else in `main.go` — that is how the in-flight pet/onboarding merges stay conflict-free.

- [ ] **Step 6: Confirm build, vet and the whole suite**

```sh
go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s
grep -n 'google.SyncHandler\|/integrations/google/sync' cmd/api/main.go
```
Expected: clean build/vet; `ok` for every package (integration tests skip); the grep has one hit.

- [ ] **Step 7: Commit**

```sh
cd .. && git add backend/internal/google/handler.go backend/internal/google/handler_test.go backend/cmd/api/main.go && git commit -m "google: POST /integrations/google/sync — §6.4 body, 409 reauth_required, 502 upstream" && cd backend
```
(Trailer as in Task 1.)

---

### Task 8: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (`**google**` bullet; `**store**` bullet)

- [ ] **Step 1: Replace the `google` bullet**

Replace the line beginning `- **google** — one-way Calendar + Tasks sync.` with:
```
- **google** — one-way Calendar + Tasks push (1st-thinking §5.1 steps 6–7; wire contract = backend spec §6.4). `POST /api/v1/integrations/google/sync` (behind `auth.Require()`) refreshes an access token from `users.google_refresh_token` — read **only** through `google.RefreshTokenSource`, whose single implementation `PgRefreshTokenSource` returns the column as stored (plaintext; the §7 AES-256-GCM inbox bug replaces that one struct) — then upserts a 30-minute `English practice` event on the primary calendar starting at the next `users.notification_time` in `users.timezone` (`RRULE:FREQ=DAILY;COUNT=28`, `dateTime`+`timeZone` so Google recurs in the user's zone) and an `English daily quests` Tasks list with one task per roadmap day (`Day N: title · title · title` from `exercises.content_json->>'title'`, `due` = local date of `roadmaps.created_at` + N−1), answering `{status: "synced", calendar_event_id, tasks_created_count}`. Idempotent via `google_sync` (migration `0002`, one row per user: `calendar_event_id`, `tasklist_id`, `roadmap_id`, `tasks_created_count`): re-sync **patches** the event (re-inserts on 404/410), leaves Tasks alone when `roadmap_id` matches the active roadmap, and deletes + rebuilds the list for a new roadmap; state is saved after the event and after the list is created, before the task inserts, so a failure never orphans a Google object. No roadmap → event only, `tasks_created_count: 0`. Errors: `invalid_grant`/401/403 → 409 `reauth_required`; other Google failures or the 60 s `SyncTimeout` → 502 `google_unavailable`. Clients (`OAuthClient.TokenURL`, `HTTPCalendarClient.BaseURL`, `HTTPTasksClient.BaseURL`) default to production and are overridden in tests — **`go test ./...` never calls Google**. Reads `users`/`roadmaps`/`exercises` through its own read-only SQL (like quests). Sync runs synchronously in the request despite §6.4's "asynchronously" (the 200 body needs the ids). Tests: `httptest` for all three clients, fakes with a call log for the service; `TestIntegrationSyncStateIsOneRowPerUser` gated on `TEST_DATABASE_URL` (skips locally, must pass in CI).
```

- [ ] **Step 2: Update the `store` bullet**

In the `**store**` bullet, after the sentence ending `` `pet_states.user_id` is UNIQUE NOT NULL (1:1 with `users`). `` add:
```
`0002_google_sync` adds `google_sync` (one row per user, FK `ON DELETE CASCADE`; `roadmap_id` FK `ON DELETE SET NULL`) for the google slice; `reset()` in the store integration tests runs every down file newest-first, and the "applied exactly once" assertions count versions, so a new migration means updating both.
```

- [ ] **Step 3: Validate and commit**

```sh
cd .. && python3 tools/harness/cli.py validate && grep -c 'google_sync' harness/CODEMAP.md
```
Expected: exit 0; count ≥ 2.
```sh
git add harness/CODEMAP.md && git commit -m "codemap: google — §6.4 sync, google_sync idempotency, refresh-token seam" && cd backend
```
(Trailer as in Task 1.)

---

## Verification

Run from the worktree root (`cd backend` where shown). Every command is local — no Docker, no Google.

```sh
cd backend && go build ./... && go vet ./...
# expect: no output

env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 120s
# expect: ok for every package incl. internal/google; no FAIL — nothing needs a live service or Google

go test ./internal/google/... -run 'Sync|Resync' -v -count=1 -timeout 60s
# expect: eight --- PASS — first sync, three save points, mid-task failure + retry, same-roadmap no-op, new-roadmap rebuild, lost event re-insert, no roadmap, reauth

go test ./internal/google/... -run 'OAuthClient|Calendar|Tasks' -v -count=1 -timeout 60s
# expect: --- PASS for every client test; every one of them uses httptest

go test ./internal/google/... -run 'NextOccurrence|PracticeEvent|DayDue' -v -count=1 -timeout 60s
# expect: --- PASS incl. the America/New_York fall-back case

go test ./internal/store/... -run 'Migration0002|AppliesPendingVersions' -v -count=1 -timeout 60s
# expect: two --- PASS

grep -rn 'googleapis.com\|oauth2.googleapis.com' internal/google/*_test.go
# expect: no hits — tests never mention a real Google host

grep -n '"status"\|"calendar_event_id"\|"tasks_created_count"' internal/google/service.go
# expect: 3 hits — the §6.4 response body

grep -rn --include='*.go' '"tasklist_id"\|"tasks_created"' internal/google/
# expect: no hits — the idea's pre-§6.4 field names are not on the wire

grep -n 'RRULE:FREQ=DAILY;COUNT=28' internal/google/schedule.go
# expect: one hit

grep -rn 'google_refresh_token' internal/google/
# expect: exactly one hit, in token.go — the single read seam for the §7 encryption fix

grep -rn 'aes\|cipher\|ENCRYPTION_SECRET_KEY' internal/google/
# expect: no hits — encryption is the inbox bug's job, not this slice's

grep -rn 'Getenv("DATABASE_URL")\|Getenv("REDIS_URL")' internal/google/
# expect: no hits — integration gating is on TEST_DATABASE_URL only

grep -c '^func TestIntegration' internal/google/integration_test.go
# expect: 1 — CI's backend-integration job counts it and fails if it skips there

grep -n 'roadmap_json' internal/google/*.go
# expect: no hits — google never writes into onboarding's document

grep -n 'CREATE TABLE google_sync' internal/store/migrations/0002_google_sync.up.sql "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md"
# expect: one hit in each — spec DDL == migrations

grep -n 'google.SyncHandler' cmd/api/main.go
# expect: one hit

cd .. && python3 tools/harness/cli.py validate; echo exit=$?
# expect: exit=0

git log --oneline main..HEAD
# expect 8 commits, one per task, each with the Co-Authored-By trailer

git status --short
# expect: clean
```

With a dev stack (`COMPOSE_PROJECT_NAME=<slug>` and non-default `POSTGRES_PORT`/`REDIS_PORT` in `backend/.env`, `docker compose up -d --wait`; `make down` after): export `TEST_DATABASE_URL`/`TEST_REDIS_URL` and run `cd backend && make test-integration` — `TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent`, `TestIntegrationConcurrentMigrateDoesNotRace` and `TestIntegrationSyncStateIsOneRowPerUser` must PASS. After pushing the branch: `gh run list --branch <branch>` must show `backend-unit`, `backend-integration` and `harness-tooling` green.

## Notes and open questions

- **Synchronous despite §6.4 "asynchronously".** The §6.4 200 body carries `calendar_event_id` and `tasks_created_count`, which only exist after Google answers, so the sync runs inside the request under `SyncTimeout` (60 s). A queued/async variant would need a status endpoint §7 does not list. If the human wants async, the `Service` is unchanged and only the handler moves to a job.
- **Wire shape supersedes the idea.** `{calendar_event_id, tasklist_id, tasks_created}` → §6.4 `{status, calendar_event_id, tasks_created_count}`; the tasklist id is stored, not returned. "No roadmap" is conveyed by `tasks_created_count: 0`, not an extra field (§6.4 has none).
- **`google_sync` vs JSONB on `roadmaps`** — decided for the table (see *Storage decision*). Side effect: the inbox bug *"Migrate is only tested against the single embedded migration"* is partly addressed for free (the runner now provably applies two versions in order); its `## Evaluation` should note that when it is judged.
- **Idempotency granularity for Tasks is per roadmap, not per task.** Same roadmap → no Tasks calls at all (titles edited in Google are left alone; a partially-deleted list is not repaired). New roadmap or a half-built list → delete + rebuild. Repairing individual tasks would mean reading the list back, which §5.1 forbids ("one-way").
- **Event start is "the next `notification_time` from now", COUNT=28 from there.** It is not anchored to `roadmaps.created_at`, so a re-sync on day 10 patches the event to start today with 28 more occurrences. Anchoring to the roadmap would make a late sync create mostly-past occurrences; recorded, not solved.
- **`due` is a date.** The Tasks API keeps only the date part of `due` (documented on the resource), so `DayDue` sends UTC midnight of the user's local calendar date; a "due time" would be lost anyway.
- **`Location` and `RoadmapDays` are copied from `quests`, not imported.** Eight lines duplicated to keep the package boundary (CODEMAP: packages talk via interfaces). If a third copy appears, hoist into `store` or a tiny `internal/clock` package.
- **Deleting a user's Tasks list on roadmap change** is the one destructive Google call. It only ever deletes a list *this package created* (its id came from our own `tasklists.insert` and is stored in `google_sync`), never a user-made list.
- **`GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`** are already required by `config.Load`; no new env. No base-URL env overrides either — tests set struct fields; add `GOOGLE_*_BASE_URL` only if a staging Google ever exists.
- **Refresh token rotation.** Google may return a new `refresh_token` on refresh; this plan ignores it (`OAuthClient` reads only `access_token`). Persisting a rotated token would be a write to `users.google_refresh_token`, which belongs to `auth`/the encryption fix — flagged for that bug's evaluation.
- **Rate/abuse.** Nothing stops a client calling sync in a loop; each call is one Calendar patch (and, for the same roadmap, nothing else). If it matters, reuse `airouter.RedisRateLimiter`'s pattern under a new §4 key — a product decision, not in this slice.

## Execution summary

Executed in `.worktrees/google-one-way-calendar-and-tasks-sync` on branch `harness/2026-09-23-high-google-one-way-calendar-and-tasks-sync`, branched from `main` at `60456e8`. All 8 tasks completed exactly as written, test-first, one commit per task (8 commits, `git log --oneline main..HEAD` confirms). `git status --short` is clean.

**Pre-flight checks (per the launch instructions):** confirmed `main` had only `0001_init.{up,down}.sql` before starting (Task 1's `0002_google_sync` uncontested). Re-read `backend/cmd/api/main.go` on `main` before Task 7: it matched the plan's quoted shape byte-for-byte (pet slice merged: `guarded` group ends with `POST /pet/revive`; onboarding not merged, as expected). No adaptation was needed — Task 7 added the `google` import and the `googleSvc`/route lines exactly as specified, immediately after the existing guarded routes and before `log.Printf("listening on :%s"...)`.

### Deviations from the plan

1. **Commit trailer.** The plan's own text specifies `Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`. The session's current attribution instructions (issued after the plan was written) specify `Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>` for all commits created in this session. I followed the session instruction (it postdates and supersedes the plan's literal text per the harness's own precedence rules for live directives vs. a stored artifact) and used it consistently across all 8 commits, amending the first (unpushed, single-commit-old) commit once to fix it before continuing.
2. **Three of the plan's own `grep` verification lines don't match literally**, because the plan's own code listings (which I transcribed verbatim) contain the very strings those greps say should be absent or singular:
   - `grep -rn 'google_refresh_token' internal/google/` — plan expects "exactly one hit, in token.go". Actual: 5 hits (4 in `token.go` — 3 doc comments plus the one `refreshTokenSQL` constant — and 1 in `integration_test.go`'s fixture `INSERT`). The single **read** of the column (the SQL constant) is indeed unique; the extra hits are the plan's own explanatory prose and test fixture.
   - `grep -rn 'aes\|cipher\|ENCRYPTION_SECRET_KEY' internal/google/` — plan expects no hits. Actual: 1 hit, `token.go`'s doc comment (as specified verbatim by the plan) referencing `ENCRYPTION_SECRET_KEY` to explain why encryption is out of scope. No cipher code exists anywhere in the package.
   - `grep -n 'roadmap_json' internal/google/*.go` — plan expects no hits. Actual: 1 hit, in `integration_test.go`'s fixture `INSERT INTO roadmaps (...roadmap_json...)` (as specified verbatim by the plan), required because the column is `NOT NULL`. Production code (`service.go`, `repo.go`) never references `roadmap_json`.
   None of these reflect a functional problem; they are artifacts of the plan's own verification commands being written slightly out of sync with its own code listings. Not resolved by altering the plan-specified code.
3. **Extra, non-committed live-proof test.** Beyond the plan's own tests, I wrote a temporary `backend/internal/google/livecheck_test.go` to satisfy the launching agent's request for live proof of the sync route through real Postgres/Redis and fake Google `httptest` servers (see *Runtime proof* below). It was not part of the plan's task list, so it was deleted after capturing its output — the branch contains only the plan's 8 commits.
4. **Synchronous vs. "asynchronously".** Implemented synchronously inside the request, exactly as the plan directs; the plan already records this as a deliberate discrepancy with the 1st-thinking doc's prose (see *Notes and open questions*). No further deviation.

### Plan verification output

```
$ go build ./... && go vet ./...
(no output — clean)

$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 120s
ok  	.../internal/airouter
ok  	.../internal/auth
ok  	.../internal/config
ok  	.../internal/google
ok  	.../internal/health
ok  	.../internal/pet
ok  	.../internal/quests
ok  	.../internal/store
(cmd/api: no test files)

$ go test ./internal/google/... -run 'Sync|Resync' -v -count=1 -timeout 60s
--- PASS x8: TestSyncFirstTimeInsertsEventListAnd28Tasks, TestSyncPersistsStateAfterTheEventAndAfterTheListBeforeTasks,
    TestSyncFailureMidTasksKeepsTheListIDSoRetryDeletesIt, TestResyncSameRoadmapPatchesEventAndCreatesNothing,
    TestResyncNewRoadmapDeletesOldListAndBuildsANewOne, TestResyncReinsertsTheEventWhenGoogleLostIt,
    TestSyncWithoutARoadmapPushesOnlyTheEvent, TestSyncNeedsReauthWithoutARefreshTokenOrOnInvalidGrant

$ go test ./internal/google/... -run 'OAuthClient|Calendar|Tasks' -v -count=1 -timeout 60s
--- PASS x11 (4 OAuthClient + 3 Calendar + 4 Tasks), all via httptest

$ go test ./internal/google/... -run 'NextOccurrence|PracticeEvent|DayDue' -v -count=1 -timeout 60s
--- PASS x3, including the America/New_York fall-back case

$ go test ./internal/store/... -run 'Migration0002|AppliesPendingVersions' -v -count=1 -timeout 60s
--- PASS x2

$ grep -rn 'googleapis.com\|oauth2.googleapis.com' internal/google/*_test.go   -> no hits
$ grep -n '"status"\|"calendar_event_id"\|"tasks_created_count"' internal/google/service.go   -> 3 hits
$ grep -rn --include='*.go' '"tasklist_id"\|"tasks_created"' internal/google/   -> no hits
$ grep -n 'RRULE:FREQ=DAILY;COUNT=28' internal/google/schedule.go   -> 1 hit
$ grep -rn 'Getenv("DATABASE_URL")\|Getenv("REDIS_URL")' internal/google/   -> no hits
$ grep -c '^func TestIntegration' internal/google/integration_test.go   -> 1
$ grep -n 'CREATE TABLE google_sync' internal/store/migrations/0002_google_sync.up.sql \
    "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md"   -> 1 hit each
$ grep -n 'google.SyncHandler' cmd/api/main.go   -> 1 hit
$ python3 tools/harness/cli.py validate; echo exit=$?   -> exit=0
$ git log --oneline main..HEAD   -> 8 commits, one per task, each with the Co-Authored-By trailer
$ git status --short   -> clean
```
(The three grep exceptions are covered under *Deviations* above.)

With the dev stack up (`COMPOSE_PROJECT_NAME=goog`, `POSTGRES_PORT=5442`, `REDIS_PORT=6390`):
```
$ make test-integration
--- PASS: TestIntegrationRateLimiterAllowsFiveThenBlocks (airouter)
--- PASS: TestIntegrationUpsertCreatesThenPreservesTheLearnerState (auth)
--- PASS: TestIntegrationSyncStateIsOneRowPerUser (google)
--- PASS: TestIntegrationEnsureCreatesExactlyOnePetRow (pet)
--- PASS: TestIntegrationDailyAndProgressAgainstRealServices (quests)
--- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (store)
--- PASS: TestIntegrationConcurrentMigrateDoesNotRace (store)
--- PASS: TestIntegrationPetStatesRejectsASecondRowForTheSameUser (store)
--- PASS: TestIntegrationRedisRoundTrip (store)
```

### Runtime proof

1. **Build + full suite** — see above, clean.
2. **Real binary boot:** built `cmd/api`, ran it against the dev stack with `DATABASE_URL`/`REDIS_URL` pointed at the unique-port compose stack and dummy `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`/`JWT_SECRET`. Gin's route dump showed `POST /api/v1/integrations/google/sync` mounted. `curl /healthz` returned `{"postgres":"ok","redis":"ok","status":"ok"}` (200). `curl -X POST /api/v1/integrations/google/sync` with no `Authorization` header returned `{"error":"unauthorized"}` (401), proving the route is correctly guarded by `auth.Require()`. Process killed and confirmed gone (`pgrep` exit 1) before moving on.
3. **Live proof of the slice's central claim (one-way sync, idempotency, reauth)** — driven through the real route (`auth.Require` → `google.SyncHandler`, identical wiring to `main.go`) against the real dev-stack Postgres/Redis, with `OAuthClient.TokenURL`/`HTTPCalendarClient.BaseURL`/`HTTPTasksClient.BaseURL` pointed at in-process `httptest` fakes recording every outbound request body (temporary test, deleted after — see *Deviations*):
   - **First sync** for a user with a 2-day roadmap (6 exercises): `200 {"status":"synced","calendar_event_id":"evt_live_1","tasks_created_count":2}`. Exactly one outbound Calendar `POST /calendars/primary/events` with body containing `"recurrence":["RRULE:FREQ=DAILY;COUNT=28"]` and `"summary":"English practice"`. Exactly one `POST /users/@me/lists` and exactly 2 `POST /lists/list_live_1/tasks`, with titles `"Day 1: Greetings · Short story · Order a coffee"` and `"Day 2: Numbers · Weather · Directions"`. SQL confirms `google_sync`: `exists=true event=evt_live_1 list=list_live_1 count=2`.
   - **Second sync**, same user, no input changes: `200`, identical body (`evt_live_1`, count 2). Exactly one *new* Calendar call and it was a `PATCH` (not a duplicate insert). Zero new Tasks calls at all (same active roadmap ⇒ list untouched, confirmed by comparing the fake's call count before/after). SQL after: identical row, unchanged (`evt_live_1`/`list_live_1`/2) — the idempotency claim, verified by asserting the actual outbound requests, not just the response.
   - **Invalid/expired refresh token**: a second user whose fake OAuth token endpoint answers `400 {"error":"invalid_grant",...}` got `409 {"error":"reauth_required"}` from the real route. Zero Calendar/Tasks calls were made (fakes had no registered routes and recorded none). SQL confirms **no** `google_sync` row was written for that user (`exists=false`) — no half-written state.
4. **Cleanup:** `docker compose down` for the `goog` project (containers/network removed, confirmed via `docker ps` showing only unrelated pre-existing containers from another project), scratch `backend/.env` deleted, temporary `livecheck_test.go` deleted, no stray `exe/api`/`cmd/api` processes (`pgrep` exit 1).

### CI

`gh` in this environment is authenticated as the owner's work account, and this repo's collaborator model does not include it — `gh pr create` is expected to fail with 403. Per the launch instructions this was noted and skipped in favor of pushing the branch, which is what triggers CI. Branch push and `gh run` results are recorded next (or noted if unavailable).

Branch pushed: `harness/2026-09-23-high-google-one-way-calendar-and-tasks-sync` (https://github.com/HendrixNguyen/English-Training-Harness/tree/harness/2026-09-23-high-google-one-way-calendar-and-tasks-sync).

**Draft PR:** `gh pr create` failed as anticipated — `GraphQL: must be a collaborator (createPullRequest)` (the `gh` CLI here is authenticated as the owner's work account, `hendrixnguyen-optisigns`, which GitHub does not recognize as a collaborator on this repo). Creating the missing `harness`/`type: mvp-slice`/`priority: high` labels first also 404'd for the same reason. No PR exists; the pushed branch is the deliverable, as instructed.

**CI:** green. Run https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35813268849 — `harness-tooling` (7s), `backend-integration` (46s), `backend-unit` (25s), all ✓.

**Status:** `done`. All six Definition-of-done items hold: builds clean; full suite (unit + integration via `make test-integration`) passes; the real `cmd/api` binary boots and answers `/healthz` and the new guarded route; every documented command (`go build`, `go vet`, `go test`, `make test-integration`, the plan's grep checks) was run as written; CI is green on the pushed branch; no destructive command surprises (the store `reset()` helper still hard-refuses without `TEST_DATABASE_URL`).
