---
idea: harness/ideas/2026-09-22-run-02/notify-web-push-subscriptions-and-delayed-reminder-queue.md
status: done
priority: high
merged: false
order: 8
branch: harness/2026-09-23-high-notify-web-push-subscriptions-and-delayed-reminder-queue
worktree: .worktrees/notify-web-push-subscriptions-and-delayed-reminder-queue
---
# Notify: Web Push subscriptions and delayed reminder queue — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use the executing-plans (or subagent-driven-development) skill to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/2026-09-22-run-02/notify-web-push-subscriptions-and-delayed-reminder-queue.md`
**Goal:** Add `backend/internal/notify` — `POST /api/v1/settings/notifications` behind `auth.Require()` with the **backend spec §6.4 DTO field for field** (`{notification_time, push_subscription{endpoint, p256dh, auth}}` → `{status: "updated", notification_time}`), which stores the VAPID subscription in `push_subscriptions` (deduplicated on `endpoint`), updates `users.notification_time`/`timezone`, and `ZADD`s the user's next local send time into the §4 `queue:webpush:delay` ZSET — plus an in-process worker (the pet slice's `RunHourly` pattern, 30-second poll) that pops due members, skips users who already met today's 1800-second target (via `quests.RedisCounter.Total` behind a `StudyCounter` interface), sends Web Push with VAPID keys from `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY` (§9), prunes subscriptions on 404/410, and re-slots each user for the next day.

**Spec precedence:** the *Backend Technical Specification* §6.4 wins for the wire shape (AGENTS.md → *Reading the spec*): the request carries **flat** `push_subscription.{endpoint, p256dh, auth}` (not the idea's `subscription.keys.*`) and the response is `{"status":"updated","notification_time":"20:00:00"}`. Two **additive** fields are kept because the feature is meaningless without them and §3.2 has the columns: optional request `timezone` (IANA name → `users.timezone`; onboarding §6.1 also writes it, this route lets the user change it) and response `next_reminder_at` (RFC3339 UTC). §4 supplies the key (`store.WebPushDelayQueueKey`, persistent, UNIX-timestamp scores); §2.1 the single-binary "integrated background cron worker"; §9 the env names; §3.2 `push_subscriptions` (`endpoint`, `p256dh`, `auth`, no unique index). The idea's `daily_progress.is_target_met` skip check is replaced by the §4 `daily:accumulated` counter (see *Architecture*) — recorded in the idea's `## Evaluation`.

**Architecture:** A `Service` over four interfaces so every test is pure: `Repo` (Postgres: `users.notification_time`/`timezone`, `push_subscriptions`), `Queue` (the ZSET: `Schedule` = `ZADD`, `Due` = `ZRANGEBYSCORE -inf now LIMIT`, `Remove` = `ZREM`), `Sender` (Web Push; `WebPushSender` wraps `webpush-go`, `ErrSubscriptionGone` on 404/410) and `StudyCounter` (`Total(ctx, userID, localDate)` — satisfied by `*quests.RedisCounter`, registered in `main.go`, so notify never touches quests' key or tables; the pet plan uses the same interface shape). `UpdateSettings` validates everything, then writes preferences, then the subscription, then schedules. `Tick(now)` **re-slots each due user to tomorrow's occurrence before sending** — a crash mid-send loses at most one reminder instead of re-firing every 30 s — then checks the counter, sends to every subscription, prunes gone ones, and drops users with no subscriptions (or no `users` row) from the queue. `RunWorker` is `pet.RunHourly`'s shape with a `time.Ticker` every `PollInterval` (30 s). Next-send arithmetic is a pure function of a clock, `notification_time` and a timezone, DST-safe through `time.Date` in the user's location.

**Web Push library — `github.com/SherClockHolmes/webpush-go` v1.4.0, decided:** it is the only maintained pure-Go implementation of RFC 8291 (message encryption, `aes128gcm`) and RFC 8292 (VAPID); the alternative is hand-rolling ECDH + HKDF + AES-GCM + a VAPID JWT, which is exactly the code nobody should write twice. It exposes `Options.HTTPClient` as an interface, so tests push to an `httptest.Server` (the subscription's `endpoint` *is* the URL — no base-URL plumbing needed), and `GenerateVAPIDKeys()` makes tests self-contained. Its only dependencies are `github.com/golang-jwt/jwt/v5` and `golang.org/x/crypto`, both already in `backend/go.mod`. Version confirmed on the module proxy (`go list -m -versions` → …, v1.3.0, v1.4.0). If `go get …@v1.4.0` fails offline, `@latest` is acceptable — record the version in the commit message.

**Tech stack:** Go 1.25 (`backend/go.mod`), Gin, `pgx/v5`, `go-redis/v9` (already present) + `webpush-go` v1.4.0 (new).

**Depends on:** `store` (1), `auth` (2), `quests` (3) merged on `main`; **pet (4) merged** — its plan's Task 9 rewrites the `main.go` block this plan edits (`studyCounter := quests.NewRedisCounter(rdb)` local, `go pet.RunHourly(ctx, petSvc)`), and this plan reuses both. If `grep -n 'studyCounter\|pet.RunHourly' cmd/api/main.go` finds nothing when you start Task 7, **stop and say so** rather than inventing a different wiring. Symbols checked against `main`: `store.WebPushDelayQueueKey` (`internal/store/keys.go:18`), `store.NewRedis` → `*store.Redis{Client *redis.Client}` (`redis.go`), `quests.RedisCounter.Total(ctx, userID, localDate string) (int64, error)` (`quests/counter.go:57`), `quests.TargetSeconds = 1800` (`quests/day.go:11`), `auth.Require`/`auth.UserID`/`auth.ContextUserID` (`auth/middleware.go`), `config.Load` (`config/config.go`). google (7) also adds a route line to `main.go`; both additions are append-only under `guarded`, so they merge cleanly.

**Run every command from `backend/`** unless the step says otherwise. `rg` is not installed — use `grep -n`. `timeout` is not installed — bound tests with `go test -timeout`. Commit messages end with the trailer after a blank line: `Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>` (shown once below; add it to every commit).

## File structure

| Path | Responsibility |
| --- | --- |
| `backend/internal/config/config.go` `config_test.go` | optional `VAPIDPublicKey`, `VAPIDPrivateKey`, `VAPIDSubject` |
| `backend/.env.example` | commented `VAPID_*` entries |
| `backend/internal/notify/schedule.go` `schedule_test.go` | `Location`, `NormalizeClock`, `NextSendTime`, `LocalDate` — pure |
| `backend/internal/notify/queue.go` | `Queue` interface + `RedisQueue` over `store.WebPushDelayQueueKey` |
| `backend/internal/notify/repo.go` | `Repo` interface, `Subscription`, `Preferences`, `PgRepo` |
| `backend/internal/notify/integration_test.go` | `TestIntegrationScheduleAndSubscriptionRoundTrip` (gated on `TEST_DATABASE_URL` + `TEST_REDIS_URL`) |
| `backend/internal/notify/push.go` `push_test.go` | `Payload`, `Sender` + `WebPushSender`, `ErrSubscriptionGone` |
| `backend/internal/notify/fakes_test.go` | in-memory `Repo`, `Queue`, `Sender`, `StudyCounter` with a call log |
| `backend/internal/notify/service.go` `service_test.go` | `UpdateSettings`, `Tick` |
| `backend/internal/notify/worker.go` `worker_test.go` | `RunWorker`, `PollInterval` |
| `backend/internal/notify/handler.go` `handler_test.go` | `POST /settings/notifications`, §6.4 body |
| `backend/cmd/api/main.go` | wire the service, start the worker when VAPID keys exist, mount the route |
| `harness/CODEMAP.md` | `notify` paragraph |

---

## Tasks

### Task 1: VAPID configuration (optional at boot)

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/config_test.go`
- Modify: `backend/.env.example`

§9 injects `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY` in production, but CI and every existing dev `.env` lack them; making them required would break `go run ./cmd/api` for everyone. Decision (mirrors `airouter`, which registers a provider only when its key is set): **optional** — the route always works, the worker starts only when both keys are present, and `main.go` logs which. `VAPID_SUBJECT` (the VAPID JWT `sub`, a `mailto:` or `https:` URL push services may verify) is **not in §9**; it defaults to `mailto:admin@example.com` — see *Notes*.

- [ ] **Step 1: Write the failing test**

Append to `backend/internal/config/config_test.go`:
```go
func TestLoadReadsOptionalVAPIDKeys(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", "s3cret")
	t.Setenv("VAPID_PUBLIC_KEY", "")
	t.Setenv("VAPID_PRIVATE_KEY", "")
	t.Setenv("VAPID_SUBJECT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() without VAPID keys = %v, want nil (they are optional)", err)
	}
	if cfg.VAPIDPublicKey != "" || cfg.VAPIDPrivateKey != "" {
		t.Errorf("VAPID keys = %q/%q, want empty", cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey)
	}
	if cfg.VAPIDSubject != DefaultVAPIDSubject {
		t.Errorf("VAPIDSubject = %q, want the default %q", cfg.VAPIDSubject, DefaultVAPIDSubject)
	}

	t.Setenv("VAPID_PUBLIC_KEY", "BPub")
	t.Setenv("VAPID_PRIVATE_KEY", "priv")
	t.Setenv("VAPID_SUBJECT", "mailto:ops@example.com")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.VAPIDPublicKey != "BPub" || cfg.VAPIDPrivateKey != "priv" || cfg.VAPIDSubject != "mailto:ops@example.com" {
		t.Errorf("cfg = %+v", cfg)
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

```sh
go test ./internal/config/... -run 'VAPID' -timeout 60s
```
Expected: FAIL to compile — `cfg.VAPIDPublicKey undefined`, `undefined: DefaultVAPIDSubject`.

- [ ] **Step 3: Implement**

In `backend/internal/config/config.go`, add to `Config` (after `JWTSecret`):
```go
	// VAPIDPublicKey / VAPIDPrivateKey sign Web Push requests (spec §9). They
	// are OPTIONAL at boot: without both, the notify worker does not start and
	// reminder settings are stored but nothing is sent (see cmd/api/main.go).
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	// VAPIDSubject is the VAPID JWT `sub` claim (a mailto: or https: URL push
	// services may contact). NOT in spec §9; defaults to DefaultVAPIDSubject.
	VAPIDSubject string
```
Add the constant near the top of the file:
```go
// DefaultVAPIDSubject is used when VAPID_SUBJECT is unset. Replace with a real
// contact before the first production push (see the notify plan's notes).
const DefaultVAPIDSubject = "mailto:admin@example.com"
```
In `Load`, after the required-variable loop and before `return cfg, nil`:
```go
	cfg.VAPIDPublicKey = os.Getenv("VAPID_PUBLIC_KEY")
	cfg.VAPIDPrivateKey = os.Getenv("VAPID_PRIVATE_KEY")
	if cfg.VAPIDSubject = os.Getenv("VAPID_SUBJECT"); cfg.VAPIDSubject == "" {
		cfg.VAPIDSubject = DefaultVAPIDSubject
	}
```
Also update the `Config` doc comment: it says later slices add `VAPID_*` — change to `(GOOGLE_CLIENT_ID, JWT_SECRET, VAPID_* have been added as their slices landed)`.

Append to `backend/.env.example`:
```
# Web Push (backend spec §9). Generate a pair once with
#   go run github.com/SherClockHolmes/webpush-go/cmd/webpush-go@v1.4.0 -gen   (or any VAPID generator)
# and keep VAPID_PUBLIC_KEY identical to the one the PWA subscribes with. Both
# unset: the API boots, POST /settings/notifications stores settings, but the
# reminder worker does not start. VAPID_SUBJECT is a mailto: or https: contact.
#VAPID_PUBLIC_KEY=
#VAPID_PRIVATE_KEY=
#VAPID_SUBJECT=mailto:admin@example.com
```
(If `webpush-go` has no `cmd/webpush-go` at v1.4.0, drop that line and say "any VAPID generator, e.g. `npx web-push generate-vapid-keys`" — verify with `go doc github.com/SherClockHolmes/webpush-go GenerateVAPIDKeys` after Task 4 and adjust.)

- [ ] **Step 4: Run to confirm it passes**

```sh
go test ./internal/config/... -count=1 -v -timeout 60s
```
Expected: every config test `--- PASS`, including the new one.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/config/config.go backend/internal/config/config_test.go backend/.env.example && git commit -m "config: optional VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY / VAPID_SUBJECT

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>" && cd backend
```

---

### Task 2: Pure scheduling — the next local send time

**Files:**
- Create: `backend/internal/notify/schedule.go`
- Test: `backend/internal/notify/schedule_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/notify/schedule_test.go`:
```go
package notify

import (
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

func TestNormalizeClockAcceptsHHMMAndHHMMSS(t *testing.T) {
	for in, want := range map[string]string{"20:00:00": "20:00:00", "7:05": "07:05:00", "23:59": "23:59:00", "00:00:00": "00:00:00"} {
		got, err := NormalizeClock(in)
		if err != nil || got != want {
			t.Errorf("NormalizeClock(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "24:00", "20:60:00", "eight", "20:00:00Z", "8pm"} {
		if _, err := NormalizeClock(bad); err == nil {
			t.Errorf("NormalizeClock(%q) accepted, want error", bad)
		}
	}
}

func TestNextSendTimeIsTodayIfAheadElseTomorrow(t *testing.T) {
	hcm := Location("Asia/Ho_Chi_Minh") // UTC+7
	// 2026-09-22T10:00Z is 17:00 in Ho Chi Minh City.
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)

	got, err := NextSendTime(now, "20:00:00", hcm)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.September, 22, 20, 0, 0, 0, hcm); !got.Equal(want) {
		t.Errorf("still ahead: got %v, want %v", got, want)
	}

	got, err = NextSendTime(now, "09:00:00", hcm)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.September, 23, 9, 0, 0, 0, hcm); !got.Equal(want) {
		t.Errorf("already past: got %v, want %v", got, want)
	}

	// Exactly now is "not ahead": the worker re-slots at fire time and must
	// land on tomorrow, never on the instant it is processing.
	at := time.Date(2026, time.September, 22, 20, 0, 0, 0, hcm)
	got, _ = NextSendTime(at, "20:00:00", hcm)
	if !got.Equal(at.AddDate(0, 0, 1)) {
		t.Errorf("at the exact minute: got %v, want tomorrow %v", got, at.AddDate(0, 0, 1))
	}
}

func TestNextSendTimeKeepsWallClockAcrossDST(t *testing.T) {
	ny := Location("America/New_York")
	// 2026-03-08 is spring-forward in New York (23-hour day).
	now := time.Date(2026, time.March, 7, 21, 0, 0, 0, ny) // 20:00 has passed
	got, err := NextSendTime(now, "20:00:00", ny)
	if err != nil {
		t.Fatal(err)
	}
	if got.Day() != 8 || got.Hour() != 20 {
		t.Errorf("got %v, want 2026-03-08 20:00 local", got)
	}
	if _, off := got.Zone(); off != -4*3600 {
		t.Errorf("offset = %d, want -14400 (EDT after spring-forward)", off)
	}
	if d := got.Sub(now); d != 23*time.Hour {
		t.Errorf("elapsed = %v, want 23h — the wall clock is kept, not the interval", d)
	}
}

func TestLocalDateUsesTheUsersTimezone(t *testing.T) {
	now := time.Date(2026, time.September, 22, 18, 30, 0, 0, time.UTC)
	if got := LocalDate(now, Location("Asia/Ho_Chi_Minh")); got != "2026-09-23" {
		t.Errorf("got %q, want 2026-09-23", got)
	}
	if got := LocalDate(now, time.UTC); got != "2026-09-22" {
		t.Errorf("got %q, want 2026-09-22", got)
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

```sh
go test ./internal/notify/... -run 'Location|NormalizeClock|NextSendTime|LocalDate' -timeout 60s
```
Expected: FAIL to compile — `undefined: Location` etc.

- [ ] **Step 3: Implement**

`backend/internal/notify/schedule.go`:
```go
// Package notify stores Web Push subscriptions and the learner's preferred
// practice time (backend spec §6.4 POST /settings/notifications), and runs
// the in-process reminder worker over the §4 queue:webpush:delay ZSET.
package notify

import (
	"fmt"
	"time"
)

// PollInterval is how often the worker asks the ZSET for due reminders.
const PollInterval = 30 * time.Second

// TargetSeconds mirrors quests.TargetSeconds (the §1 30-minute goal). Not
// imported: notify sees the counter only through StudyCounter.
const TargetSeconds = 1800

// Location resolves users.timezone (§3.2, default 'UTC'); an unknown name
// falls back to UTC. Same rule as quests.Location — a bad timezone must never
// stop a reminder.
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

// NormalizeClock accepts "HH:MM" or "HH:MM:SS" and returns "HH:MM:SS" — the
// form Postgres renders TIME in and the form §6.4 shows.
func NormalizeClock(s string) (string, error) {
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("15:04:05"), nil
		}
	}
	return "", fmt.Errorf("notify: notification_time %q is not HH:MM[:SS]", s)
}

// NextSendTime is the next wall-clock hhmmss in loc strictly after now:
// today if still ahead, otherwise tomorrow. Built with time.Date in loc, so a
// DST change between now and then keeps the wall-clock time (a 23- or
// 25-hour gap) rather than adding a fixed 24 h.
func NextSendTime(now time.Time, hhmmss string, loc *time.Location) (time.Time, error) {
	norm, err := NormalizeClock(hhmmss)
	if err != nil {
		return time.Time{}, err
	}
	clock, _ := time.Parse("15:04:05", norm)
	l := now.In(loc)
	next := time.Date(l.Year(), l.Month(), l.Day(), clock.Hour(), clock.Minute(), clock.Second(), 0, loc)
	if !next.After(now) {
		next = time.Date(l.Year(), l.Month(), l.Day()+1, clock.Hour(), clock.Minute(), clock.Second(), 0, loc)
	}
	return next, nil
}

// LocalDate is the YYYY-MM-DD the user is living in — the date component of
// quests' daily:accumulated key, which StudyCounter.Total is keyed by.
func LocalDate(now time.Time, loc *time.Location) string {
	return now.In(loc).Format("2006-01-02")
}
```

- [ ] **Step 4: Run to confirm it passes**

```sh
go test ./internal/notify/... -run 'Location|NormalizeClock|NextSendTime|LocalDate' -v -timeout 60s
```
Expected: five `--- PASS`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/notify/schedule.go backend/internal/notify/schedule_test.go && git commit -m "notify: pure next-send-time arithmetic, DST-safe" && cd backend
```
(Trailer as in Task 1.)

---

### Task 3: The ZSET queue, the Postgres repo, and their integration test

**Files:**
- Create: `backend/internal/notify/queue.go`, `backend/internal/notify/repo.go`
- Test: `backend/internal/notify/integration_test.go`

Neither file has logic that a fake can prove, so their one test is live: `TestIntegrationScheduleAndSubscriptionRoundTrip`, gated on **both** `TEST_DATABASE_URL` and `TEST_REDIS_URL` (the quests convention, `internal/quests/integration_test.go:23-26`) and never on the production URLs. CI's `backend-integration` job exports both, counts `func TestIntegration*` and fails on a `--- SKIP`, so it must pass there.

Dedupe decision for `push_subscriptions` (no unique index in §3.2, so no `ON CONFLICT`): one statement with a data-modifying CTE — delete the endpoint from **other** users (a browser profile that signed into a second account now belongs to that account), insert for this user only if this user does not already have that endpoint. Atomic per statement; `p256dh`/`auth` only change together with a new `endpoint`, so an existing row is left alone.

- [ ] **Step 1: Write the failing integration test**

`backend/internal/notify/integration_test.go`:
```go
package notify

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// TestIntegrationScheduleAndSubscriptionRoundTrip proves the ZSET round trip
// (ZADD → ZRANGEBYSCORE → overwrite → ZREM) and the push_subscriptions
// dedupe against real Redis and Postgres. Gated on TEST_DATABASE_URL and
// TEST_REDIS_URL like internal/quests — never on the production URLs.
func TestIntegrationScheduleAndSubscriptionRoundTrip(t *testing.T) {
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

	// Two users; the second steals the first one's browser endpoint later.
	ids := map[string]string{}
	for _, gid := range []string{"notify-integration-a", "notify-integration-b"} {
		_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)
		var id string
		if err := pg.Pool.QueryRow(ctx,
			`INSERT INTO users (email, google_id, target_goal, timezone) VALUES ($1,$2,$3,$4) RETURNING id`,
			gid+"@example.com", gid, "", "UTC").Scan(&id); err != nil {
			t.Fatalf("inserting %s: %v", gid, err)
		}
		ids[gid] = id
		g := gid
		t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, g) })
	}
	a, b := ids["notify-integration-a"], ids["notify-integration-b"]

	repo := NewPgRepo(pg.Pool)

	// --- preferences ---
	if err := repo.UpdatePreferences(ctx, a, "07:30:00", "Asia/Ho_Chi_Minh"); err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}
	prefs, err := repo.Preferences(ctx, a)
	if err != nil || prefs.NotificationTime != "07:30:00" || prefs.Timezone != "Asia/Ho_Chi_Minh" {
		t.Fatalf("Preferences = %+v, %v", prefs, err)
	}
	if err := repo.UpdatePreferences(ctx, a, "08:00:00", ""); err != nil {
		t.Fatal(err)
	}
	if prefs, _ = repo.Preferences(ctx, a); prefs.Timezone != "Asia/Ho_Chi_Minh" || prefs.NotificationTime != "08:00:00" {
		t.Errorf("empty timezone must leave it unchanged: %+v", prefs)
	}
	if err := repo.UpdatePreferences(ctx, "00000000-0000-0000-0000-000000000000", "08:00:00", ""); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("unknown user: err = %v, want ErrUserNotFound", err)
	}

	// --- subscriptions: same endpoint twice → one row; other user → moves ---
	sub := Subscription{Endpoint: "https://push.example.test/notify-integration/ep1", P256dh: "BNc5T", Auth: "aX8v"}
	for i := 0; i < 2; i++ {
		if err := repo.SaveSubscription(ctx, a, sub); err != nil {
			t.Fatalf("SaveSubscription #%d: %v", i+1, err)
		}
	}
	subsA, err := repo.Subscriptions(ctx, a)
	if err != nil || len(subsA) != 1 || subsA[0].Endpoint != sub.Endpoint || subsA[0].ID == "" {
		t.Fatalf("Subscriptions(a) = %+v, %v; want exactly one", subsA, err)
	}
	if err := repo.SaveSubscription(ctx, b, sub); err != nil {
		t.Fatal(err)
	}
	subsA, _ = repo.Subscriptions(ctx, a)
	subsB, _ := repo.Subscriptions(ctx, b)
	if len(subsA) != 0 || len(subsB) != 1 {
		t.Errorf("after b subscribes with a's endpoint: a=%d b=%d, want 0/1", len(subsA), len(subsB))
	}
	if err := repo.DeleteSubscription(ctx, subsB[0].ID); err != nil {
		t.Fatal(err)
	}
	if subsB, _ = repo.Subscriptions(ctx, b); len(subsB) != 0 {
		t.Errorf("after delete: %d subscriptions, want 0", len(subsB))
	}

	// --- the ZSET ---
	q := NewRedisQueue(rdb)
	t.Cleanup(func() { _ = q.Remove(ctx, a); _ = q.Remove(ctx, b) })
	_ = q.Remove(ctx, a)
	_ = q.Remove(ctx, b)

	now := time.Now().Truncate(time.Second)
	if err := q.Schedule(ctx, a, now.Add(time.Hour)); err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	if err := q.Schedule(ctx, b, now.Add(-time.Minute)); err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	due, err := q.Due(ctx, now, 100)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if !contains(due, b) || contains(due, a) {
		t.Errorf("Due(now) = %v; want b (past) and not a (in an hour)", due)
	}
	// ZADD on an existing member overwrites its score — the re-slot.
	if err := q.Schedule(ctx, b, now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if due, _ = q.Due(ctx, now, 100); contains(due, b) {
		t.Errorf("after re-slot b is still due: %v", due)
	}
	if due, _ = q.Due(ctx, now.Add(3*time.Hour), 100); !contains(due, a) || !contains(due, b) {
		t.Errorf("Due(now+3h) = %v; want both", due)
	}
	n := 0
	for _, m := range due {
		if m == b {
			n++
		}
	}
	if n != 1 {
		t.Errorf("b appears %d times in the ZSET, want 1 (ZADD must not duplicate)", n)
	}
	if err := q.Remove(ctx, a); err != nil {
		t.Fatal(err)
	}
	if due, _ = q.Due(ctx, now.Add(3*time.Hour), 100); contains(due, a) {
		t.Errorf("after Remove a is still there: %v", due)
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run to confirm it fails to compile**

```sh
go test ./internal/notify/... -run 'Integration' -timeout 60s
```
Expected: FAIL to compile — `undefined: NewPgRepo`, `NewRedisQueue`, `Subscription`, `ErrUserNotFound`.

- [ ] **Step 3: Implement the queue**

`backend/internal/notify/queue.go`:
```go
package notify

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// DueBatchSize caps one Tick. At a 30 s poll that is 200 users/minute; a
// bigger backlog simply drains over several ticks.
const DueBatchSize = 100

// Queue is the §4 queue:webpush:delay ZSET: member = user id, score = the
// UNIX time of the next reminder. It is persistent (no TTL).
type Queue interface {
	// Schedule sets the user's next send time (ZADD; overwrites the score, so
	// a user is never in the set twice).
	Schedule(ctx context.Context, userID string, at time.Time) error
	// Due returns up to limit members whose score is <= now, oldest first.
	Due(ctx context.Context, now time.Time, limit int64) ([]string, error)
	// Remove drops the user from the set (no subscriptions left, or no user).
	Remove(ctx context.Context, userID string) error
}

// RedisQueue is the real Queue.
type RedisQueue struct{ Client *redis.Client }

// NewRedisQueue builds a queue over an existing client.
func NewRedisQueue(r *store.Redis) *RedisQueue { return &RedisQueue{Client: r.Client} }

func (q *RedisQueue) Schedule(ctx context.Context, userID string, at time.Time) error {
	if err := q.Client.ZAdd(ctx, store.WebPushDelayQueueKey, redis.Z{Score: float64(at.Unix()), Member: userID}).Err(); err != nil {
		return fmt.Errorf("notify: scheduling reminder: %w", err)
	}
	return nil
}

func (q *RedisQueue) Due(ctx context.Context, now time.Time, limit int64) ([]string, error) {
	members, err := q.Client.ZRangeByScore(ctx, store.WebPushDelayQueueKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   strconv.FormatInt(now.Unix(), 10),
		Count: limit,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("notify: reading due reminders: %w", err)
	}
	return members, nil
}

func (q *RedisQueue) Remove(ctx context.Context, userID string) error {
	if err := q.Client.ZRem(ctx, store.WebPushDelayQueueKey, userID).Err(); err != nil {
		return fmt.Errorf("notify: removing reminder: %w", err)
	}
	return nil
}

var _ Queue = (*RedisQueue)(nil)
```

- [ ] **Step 4: Implement the repo**

`backend/internal/notify/repo.go`:
```go
package notify

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUserNotFound means the users row is gone (the session outlived it).
var ErrUserNotFound = errors.New("notify: user not found")

// Subscription is a §3.2 push_subscriptions row — the PushSubscription the
// browser handed the PWA. ID is empty on input.
type Subscription struct {
	ID       string
	Endpoint string
	P256dh   string
	Auth     string
}

// Preferences is the slice of users this package reads.
type Preferences struct {
	NotificationTime string // "HH:MM:SS"
	Timezone         string // IANA name, 'UTC' when NULL
}

// Repo is the Postgres side. Everything is scoped by user id.
type Repo interface {
	// UpdatePreferences writes notification_time and, when timezone is
	// non-empty, timezone. ErrUserNotFound when no row matched.
	UpdatePreferences(ctx context.Context, userID, notificationTime, timezone string) error
	Preferences(ctx context.Context, userID string) (Preferences, error)
	// SaveSubscription stores s for userID exactly once per endpoint and
	// takes the endpoint away from any other user that had it.
	SaveSubscription(ctx context.Context, userID string, s Subscription) error
	Subscriptions(ctx context.Context, userID string) ([]Subscription, error)
	DeleteSubscription(ctx context.Context, id string) error
}

const (
	updatePreferencesSQL = `
UPDATE users
SET notification_time = $2::time,
    timezone = COALESCE(NULLIF($3, ''), timezone)
WHERE id = $1`

	preferencesSQL = `SELECT COALESCE(notification_time::text, '20:00:00'), COALESCE(timezone, 'UTC') FROM users WHERE id = $1`

	// §3.2 has no UNIQUE on endpoint, so no ON CONFLICT: the CTE moves the
	// endpoint away from other users, the INSERT adds it for this user only
	// if absent. One statement, so it is atomic.
	saveSubscriptionSQL = `
WITH moved AS (
    DELETE FROM push_subscriptions WHERE endpoint = $2 AND user_id <> $1
)
INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
SELECT $1, $2, $3, $4
WHERE NOT EXISTS (SELECT 1 FROM push_subscriptions WHERE endpoint = $2 AND user_id = $1)`

	subscriptionsSQL = `SELECT id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = $1 ORDER BY created_at, id`

	deleteSubscriptionSQL = `DELETE FROM push_subscriptions WHERE id = $1`
)

// PgRepo is the real Repo.
type PgRepo struct{ Pool *pgxpool.Pool }

// NewPgRepo builds a repo over an existing pool.
func NewPgRepo(pool *pgxpool.Pool) *PgRepo { return &PgRepo{Pool: pool} }

func (r *PgRepo) UpdatePreferences(ctx context.Context, userID, notificationTime, timezone string) error {
	tag, err := r.Pool.Exec(ctx, updatePreferencesSQL, userID, notificationTime, timezone)
	if err != nil {
		return fmt.Errorf("notify: updating preferences: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PgRepo) Preferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.Pool.QueryRow(ctx, preferencesSQL, userID).Scan(&p.NotificationTime, &p.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrUserNotFound
	}
	if err != nil {
		return Preferences{}, fmt.Errorf("notify: reading preferences: %w", err)
	}
	return p, nil
}

func (r *PgRepo) SaveSubscription(ctx context.Context, userID string, s Subscription) error {
	if _, err := r.Pool.Exec(ctx, saveSubscriptionSQL, userID, s.Endpoint, s.P256dh, s.Auth); err != nil {
		return fmt.Errorf("notify: saving subscription: %w", err)
	}
	return nil
}

func (r *PgRepo) Subscriptions(ctx context.Context, userID string) ([]Subscription, error) {
	rows, err := r.Pool.Query(ctx, subscriptionsSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("notify: reading subscriptions: %w", err)
	}
	defer rows.Close()

	var out []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.ID, &s.Endpoint, &s.P256dh, &s.Auth); err != nil {
			return nil, fmt.Errorf("notify: scanning subscription: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PgRepo) DeleteSubscription(ctx context.Context, id string) error {
	if _, err := r.Pool.Exec(ctx, deleteSubscriptionSQL, id); err != nil {
		return fmt.Errorf("notify: deleting subscription: %w", err)
	}
	return nil
}

var _ Repo = (*PgRepo)(nil)
```

- [ ] **Step 5: Compile, vet, confirm the skip locally**

```sh
go vet ./internal/notify/... && go test ./internal/notify/... -run 'Integration' -v -count=1 -timeout 60s
```
Expected: `--- SKIP: TestIntegrationScheduleAndSubscriptionRoundTrip` with the unset-URL message. **It must PASS in CI's `backend-integration` job.** To prove it locally: dev stack with `COMPOSE_PROJECT_NAME=<slug>` and non-default ports (AGENTS.md), export `TEST_DATABASE_URL`/`TEST_REDIS_URL`, then `go test ./internal/notify/... -run Integration -v -count=1 -p 1 -timeout 120s`.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend/internal/notify/queue.go backend/internal/notify/repo.go backend/internal/notify/integration_test.go && git commit -m "notify: queue:webpush:delay ZSET queue, push_subscriptions repo, live round-trip test" && cd backend
```
(Trailer as in Task 1.)

---

### Task 4: Web Push sender over `webpush-go`, tested against `httptest`

**Files:**
- Modify: `backend/go.mod`, `backend/go.sum` (via `go get`)
- Create: `backend/internal/notify/push.go`
- Test: `backend/internal/notify/push_test.go`

- [ ] **Step 1: Add the dependency**

```sh
go get github.com/SherClockHolmes/webpush-go@v1.4.0 && go mod tidy
grep -n 'webpush-go' go.mod
```
Expected: one `require` line `github.com/SherClockHolmes/webpush-go v1.4.0` (direct). `go mod tidy` may move `golang.org/x/crypto` from `// indirect` — fine.

- [ ] **Step 2: Write the failing test**

`backend/internal/notify/push_test.go`:
```go
package notify

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// browserSubscription fabricates what PushManager.subscribe() hands the PWA:
// an uncompressed P-256 public key and a 16-byte auth secret, base64url.
func browserSubscription(t *testing.T, endpoint string) Subscription {
	t.Helper()
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	if _, err := rand.Read(auth); err != nil {
		t.Fatal(err)
	}
	return Subscription{
		ID:       "sub-1",
		Endpoint: endpoint,
		P256dh:   base64.RawURLEncoding.EncodeToString(priv.PublicKey().Bytes()),
		Auth:     base64.RawURLEncoding.EncodeToString(auth),
	}
}

func newTestSender(t *testing.T) *WebPushSender {
	t.Helper()
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewWebPushSender(pub, priv, "mailto:test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestWebPushSenderPostsAnEncryptedVAPIDSignedRequest(t *testing.T) {
	var gotAuth, gotEncoding, gotTTL string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotEncoding = r.Header.Get("Content-Encoding")
		gotTTL = r.Header.Get("TTL")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	s := newTestSender(t)
	err := s.Send(context.Background(), browserSubscription(t, srv.URL+"/push/abc"), Payload{Title: "t", Body: "b", URL: "/"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.HasPrefix(strings.ToLower(gotAuth), "vapid ") {
		t.Errorf("Authorization = %q, want a VAPID header", gotAuth)
	}
	if gotEncoding != "aes128gcm" {
		t.Errorf("Content-Encoding = %q, want aes128gcm (RFC 8291)", gotEncoding)
	}
	if gotTTL != "3600" {
		t.Errorf("TTL = %q, want 3600", gotTTL)
	}
	if len(gotBody) == 0 || strings.Contains(string(gotBody), `"title"`) {
		t.Errorf("body (%d bytes) must be the encrypted payload, never plaintext JSON", len(gotBody))
	}
}

func TestWebPushSenderReportsGoneOn404And410(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusGone} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }))
		s := newTestSender(t)
		err := s.Send(context.Background(), browserSubscription(t, srv.URL), Payload{Title: "t"})
		srv.Close()
		if !errors.Is(err, ErrSubscriptionGone) {
			t.Errorf("%d: err = %v, want ErrSubscriptionGone", code, err)
		}
	}
}

func TestWebPushSenderReportsOtherFailuresAsErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	s := newTestSender(t)
	err := s.Send(context.Background(), browserSubscription(t, srv.URL), Payload{Title: "t"})
	if err == nil || errors.Is(err, ErrSubscriptionGone) {
		t.Errorf("429: err = %v, want a non-gone error", err)
	}
}

func TestNewWebPushSenderRequiresBothKeys(t *testing.T) {
	if _, err := NewWebPushSender("", "priv", "mailto:x@example.com"); err == nil {
		t.Error("missing public key accepted")
	}
	if _, err := NewWebPushSender("pub", "", "mailto:x@example.com"); err == nil {
		t.Error("missing private key accepted")
	}
}
```

- [ ] **Step 3: Run to confirm it fails**

```sh
go test ./internal/notify/... -run 'WebPushSender' -timeout 60s
```
Expected: FAIL to compile — `undefined: NewWebPushSender`, `Payload`, `ErrSubscriptionGone`.

- [ ] **Step 4: Implement**

`backend/internal/notify/push.go`:
```go
package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// ErrSubscriptionGone means the push service answered 404 or 410: the
// browser unsubscribed or the endpoint expired. The worker deletes the row.
var ErrSubscriptionGone = errors.New("notify: subscription gone")

// Payload is what the PWA's service worker receives (it shows a notification
// and opens URL on click).
type Payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

// DefaultPayload is the daily reminder.
var DefaultPayload = Payload{
	Title: "Time to practice English",
	Body:  "Your 30-minute session is waiting — keep your plant alive.",
	URL:   "/",
}

// PushTTL is how long the push service keeps an undelivered reminder. An
// hour: a reminder about "now" is noise by tomorrow.
const PushTTL = int(time.Hour / time.Second)

// Sender delivers one payload to one subscription.
type Sender interface {
	Send(ctx context.Context, sub Subscription, p Payload) error
}

// WebPushSender is the real Sender (RFC 8291 encryption + RFC 8292 VAPID via
// webpush-go). HTTPClient is an interface so tests can point it anywhere;
// the subscription endpoint is the URL, so httptest needs no base-URL plumbing.
type WebPushSender struct {
	PublicKey  string
	PrivateKey string
	Subscriber string // VAPID `sub`: a mailto: or https: contact
	HTTPClient webpush.HTTPClient
	TTL        int
}

// NewWebPushSender validates the key pair is present (spec §9 VAPID_*).
func NewWebPushSender(publicKey, privateKey, subscriber string) (*WebPushSender, error) {
	if publicKey == "" || privateKey == "" {
		return nil, errors.New("notify: VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY are both required")
	}
	return &WebPushSender{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Subscriber: subscriber,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		TTL:        PushTTL,
	}, nil
}

func (s *WebPushSender) Send(ctx context.Context, sub Subscription, p Payload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("notify: encoding payload: %w", err)
	}
	resp, err := webpush.SendNotificationWithContext(ctx, body,
		&webpush.Subscription{Endpoint: sub.Endpoint, Keys: webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth}},
		&webpush.Options{
			HTTPClient:      s.HTTPClient,
			Subscriber:      s.Subscriber,
			VAPIDPublicKey:  s.PublicKey,
			VAPIDPrivateKey: s.PrivateKey,
			TTL:             s.TTL,
			Urgency:         webpush.UrgencyNormal,
		})
	if err != nil {
		return fmt.Errorf("notify: sending push: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return fmt.Errorf("%w: push service returned %d", ErrSubscriptionGone, resp.StatusCode)
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return fmt.Errorf("notify: push service returned %d", resp.StatusCode)
	}
	return nil
}

var _ Sender = (*WebPushSender)(nil)
```

- [ ] **Step 5: Run to confirm it passes**

```sh
go vet ./internal/notify/... && go test ./internal/notify/... -run 'WebPushSender' -v -count=1 -timeout 60s
```
Expected: four `--- PASS`. If `TestWebPushSenderPostsAnEncryptedVAPIDSignedRequest` fails on the `Authorization` prefix, print the header — webpush-go v1.x sends `vapid t=…, k=…`; if your version sends `WebPush …` + `Crypto-Key`, relax the assertion to accept either and note the version in the commit.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend/go.mod backend/go.sum backend/internal/notify/push.go backend/internal/notify/push_test.go && git commit -m "notify: Web Push sender over webpush-go v1.4.0; 404/410 -> ErrSubscriptionGone" && cd backend
```
(Trailer as in Task 1.)

---

### Task 5: Fakes and `Service` — `UpdateSettings` and `Tick`

**Files:**
- Create: `backend/internal/notify/fakes_test.go`, `backend/internal/notify/service.go`
- Test: `backend/internal/notify/service_test.go`

- [ ] **Step 1: Write the fakes**

`backend/internal/notify/fakes_test.go`:
```go
package notify

import (
	"context"
	"fmt"
	"sort"
	"time"
)

type callLog struct{ calls []string }

func (l *callLog) add(format string, args ...any) { l.calls = append(l.calls, fmt.Sprintf(format, args...)) }

type fakeRepo struct {
	log   *callLog
	prefs map[string]Preferences   // by user id; missing → ErrUserNotFound
	subs  map[string][]Subscription // by user id
	seq   int
}

func newFakeRepo(log *callLog) *fakeRepo {
	return &fakeRepo{log: log, prefs: map[string]Preferences{}, subs: map[string][]Subscription{}}
}

func (f *fakeRepo) UpdatePreferences(_ context.Context, userID, clock, tz string) error {
	f.log.add("repo.UpdatePreferences(%s,%s,%s)", userID, clock, tz)
	p, ok := f.prefs[userID]
	if !ok {
		return ErrUserNotFound
	}
	p.NotificationTime = clock
	if tz != "" {
		p.Timezone = tz
	}
	f.prefs[userID] = p
	return nil
}

func (f *fakeRepo) Preferences(_ context.Context, userID string) (Preferences, error) {
	f.log.add("repo.Preferences(%s)", userID)
	p, ok := f.prefs[userID]
	if !ok {
		return Preferences{}, ErrUserNotFound
	}
	return p, nil
}

func (f *fakeRepo) SaveSubscription(_ context.Context, userID string, s Subscription) error {
	f.log.add("repo.SaveSubscription(%s,%s)", userID, s.Endpoint)
	for _, existing := range f.subs[userID] {
		if existing.Endpoint == s.Endpoint {
			return nil
		}
	}
	f.seq++
	s.ID = fmt.Sprintf("sub-%d", f.seq)
	f.subs[userID] = append(f.subs[userID], s)
	return nil
}

func (f *fakeRepo) Subscriptions(_ context.Context, userID string) ([]Subscription, error) {
	f.log.add("repo.Subscriptions(%s)", userID)
	return f.subs[userID], nil
}

func (f *fakeRepo) DeleteSubscription(_ context.Context, id string) error {
	f.log.add("repo.DeleteSubscription(%s)", id)
	for uid, list := range f.subs {
		kept := list[:0]
		for _, s := range list {
			if s.ID != id {
				kept = append(kept, s)
			}
		}
		f.subs[uid] = kept
	}
	return nil
}

type fakeQueue struct {
	log    *callLog
	scores map[string]int64 // member → unix score
}

func newFakeQueue(log *callLog) *fakeQueue { return &fakeQueue{log: log, scores: map[string]int64{}} }

func (f *fakeQueue) Schedule(_ context.Context, userID string, at time.Time) error {
	f.log.add("queue.Schedule(%s,%s)", userID, at.UTC().Format(time.RFC3339))
	f.scores[userID] = at.Unix()
	return nil
}

func (f *fakeQueue) Due(_ context.Context, now time.Time, limit int64) ([]string, error) {
	f.log.add("queue.Due(%s)", now.UTC().Format(time.RFC3339))
	var out []string
	for m, s := range f.scores {
		if s <= now.Unix() {
			out = append(out, m)
		}
	}
	sort.Strings(out)
	if int64(len(out)) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeQueue) Remove(_ context.Context, userID string) error {
	f.log.add("queue.Remove(%s)", userID)
	delete(f.scores, userID)
	return nil
}

type fakeSender struct {
	log  *callLog
	gone map[string]bool // endpoint → answer ErrSubscriptionGone
	fail map[string]bool // endpoint → answer a generic error
	sent []string        // endpoints in order
}

func newFakeSender(log *callLog) *fakeSender {
	return &fakeSender{log: log, gone: map[string]bool{}, fail: map[string]bool{}}
}

func (f *fakeSender) Send(_ context.Context, sub Subscription, _ Payload) error {
	f.log.add("sender.Send(%s)", sub.Endpoint)
	if f.gone[sub.Endpoint] {
		return ErrSubscriptionGone
	}
	if f.fail[sub.Endpoint] {
		return fmt.Errorf("push service returned 429")
	}
	f.sent = append(f.sent, sub.Endpoint)
	return nil
}

type fakeCounter struct {
	log    *callLog
	totals map[string]int64 // userID+"|"+localDate → seconds
	err    error
}

func newFakeCounter(log *callLog) *fakeCounter { return &fakeCounter{log: log, totals: map[string]int64{}} }

func (f *fakeCounter) Total(_ context.Context, userID, localDate string) (int64, error) {
	f.log.add("counter.Total(%s,%s)", userID, localDate)
	if f.err != nil {
		return 0, f.err
	}
	return f.totals[userID+"|"+localDate], nil
}

// harness: one user in Ho Chi Minh City with a 20:00 reminder.
type harness struct {
	log     *callLog
	repo    *fakeRepo
	queue   *fakeQueue
	sender  *fakeSender
	counter *fakeCounter
	now     time.Time
	svc     *Service
}

func newHarness() *harness {
	log := &callLog{}
	h := &harness{
		log:     log,
		repo:    newFakeRepo(log),
		queue:   newFakeQueue(log),
		sender:  newFakeSender(log),
		counter: newFakeCounter(log),
		// 2026-09-22T10:00Z = 17:00 in Ho Chi Minh City.
		now: time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC),
	}
	h.repo.prefs["u1"] = Preferences{NotificationTime: "20:00:00", Timezone: "Asia/Ho_Chi_Minh"}
	h.svc = NewService(h.repo, h.queue, h.sender, h.counter, func() time.Time { return h.now })
	return h
}

func (h *harness) hcm(day, hour, min int) time.Time {
	return time.Date(2026, time.September, day, hour, min, 0, 0, Location("Asia/Ho_Chi_Minh"))
}
```

- [ ] **Step 2: Write the failing service tests**

`backend/internal/notify/service_test.go`:
```go
package notify

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

var validSub = &Subscription{Endpoint: "https://push.example/ep1", P256dh: "BNc5T", Auth: "aX8v"}

func TestUpdateSettingsStoresSubscriptionPreferencesAndSchedulesTonight(t *testing.T) {
	h := newHarness()
	res, err := h.svc.UpdateSettings(context.Background(), "u1", SettingsRequest{
		NotificationTime: "20:00", Timezone: "Asia/Ho_Chi_Minh", Subscription: validSub,
	})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if res.Status != "updated" || res.NotificationTime != "20:00:00" {
		t.Errorf("result = %+v (§6.4 body)", res)
	}
	want := h.hcm(22, 20, 0)
	if res.NextReminderAt != want.UTC().Format(time.RFC3339) {
		t.Errorf("next_reminder_at = %s, want %s", res.NextReminderAt, want.UTC().Format(time.RFC3339))
	}
	if h.queue.scores["u1"] != want.Unix() {
		t.Errorf("ZSET score = %d, want %d (tonight 20:00 HCM)", h.queue.scores["u1"], want.Unix())
	}
	if subs := h.repo.subs["u1"]; len(subs) != 1 || subs[0].Endpoint != validSub.Endpoint {
		t.Errorf("subscriptions = %+v", subs)
	}
	if p := h.repo.prefs["u1"]; p.NotificationTime != "20:00:00" || p.Timezone != "Asia/Ho_Chi_Minh" {
		t.Errorf("prefs = %+v", p)
	}
}

func TestUpdateSettingsWithoutASubscriptionOnlyMovesTheTime(t *testing.T) {
	h := newHarness()
	_, err := h.svc.UpdateSettings(context.Background(), "u1", SettingsRequest{NotificationTime: "06:30:00"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range h.log.calls {
		if strings.HasPrefix(c, "repo.SaveSubscription") {
			t.Errorf("no subscription in the request, but %s", c)
		}
	}
	// Timezone untouched (empty in request), time moved, tomorrow 06:30 HCM.
	if p := h.repo.prefs["u1"]; p.Timezone != "Asia/Ho_Chi_Minh" || p.NotificationTime != "06:30:00" {
		t.Errorf("prefs = %+v", p)
	}
	if want := h.hcm(23, 6, 30); h.queue.scores["u1"] != want.Unix() {
		t.Errorf("score = %d, want tomorrow 06:30 HCM %d", h.queue.scores["u1"], want.Unix())
	}
}

func TestUpdateSettingsRejectsBadInputBeforeWriting(t *testing.T) {
	cases := map[string]SettingsRequest{
		"bad clock":               {NotificationTime: "25:00"},
		"missing clock":           {NotificationTime: ""},
		"bad timezone":            {NotificationTime: "20:00", Timezone: "Mars/Olympus"},
		"subscription no endpoint": {NotificationTime: "20:00", Subscription: &Subscription{P256dh: "x", Auth: "y"}},
		"subscription no keys":    {NotificationTime: "20:00", Subscription: &Subscription{Endpoint: "https://e"}},
	}
	for name, req := range cases {
		h := newHarness()
		_, err := h.svc.UpdateSettings(context.Background(), "u1", req)
		if !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: err = %v, want ErrInvalidRequest", name, err)
		}
		if len(h.log.calls) != 0 {
			t.Errorf("%s: rejected request still made calls %v", name, h.log.calls)
		}
	}
}

func TestUpdateSettingsForAMissingUser(t *testing.T) {
	h := newHarness()
	_, err := h.svc.UpdateSettings(context.Background(), "ghost", SettingsRequest{NotificationTime: "20:00"})
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("err = %v, want ErrUserNotFound", err)
	}
}

// tick-time harness: it is 20:00:05 in HCM and u1 is due.
func dueHarness() *harness {
	h := newHarness()
	h.now = h.hcm(22, 20, 0).Add(5 * time.Second)
	h.queue.scores["u1"] = h.hcm(22, 20, 0).Unix()
	h.repo.subs["u1"] = []Subscription{
		{ID: "s1", Endpoint: "https://push.example/ep1", P256dh: "a", Auth: "b"},
		{ID: "s2", Endpoint: "https://push.example/ep2", P256dh: "c", Auth: "d"},
	}
	return h
}

func TestTickSendsToEverySubscriptionAndReslotsTomorrowBeforeSending(t *testing.T) {
	h := dueHarness()
	stats, err := h.svc.Tick(context.Background(), h.now)
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if stats.Due != 1 || stats.Sent != 2 || stats.Skipped != 0 || stats.Pruned != 0 {
		t.Errorf("stats = %+v", stats)
	}
	if want := h.hcm(23, 20, 0); h.queue.scores["u1"] != want.Unix() {
		t.Errorf("re-slot = %d, want tomorrow 20:00 HCM %d", h.queue.scores["u1"], want.Unix())
	}
	idx := func(prefix string) int {
		for i, c := range h.log.calls {
			if strings.HasPrefix(c, prefix) {
				return i
			}
		}
		return -1
	}
	if !(idx("queue.Schedule(u1") < idx("sender.Send(")) {
		t.Errorf("must re-slot before sending (crash safety): %v", h.log.calls)
	}
	if len(h.sender.sent) != 2 {
		t.Errorf("sent = %v", h.sender.sent)
	}
}

func TestTickIgnoresUsersNotYetDue(t *testing.T) {
	h := dueHarness()
	h.repo.prefs["u2"] = Preferences{NotificationTime: "21:00:00", Timezone: "Asia/Ho_Chi_Minh"}
	h.queue.scores["u2"] = h.hcm(22, 21, 0).Unix()
	h.repo.subs["u2"] = []Subscription{{ID: "s3", Endpoint: "https://push.example/ep3"}}

	stats, _ := h.svc.Tick(context.Background(), h.now)
	if stats.Due != 1 {
		t.Errorf("due = %d, want 1", stats.Due)
	}
	for _, e := range h.sender.sent {
		if e == "https://push.example/ep3" {
			t.Error("u2 (21:00) was sent at 20:00")
		}
	}
	if h.queue.scores["u2"] != h.hcm(22, 21, 0).Unix() {
		t.Error("u2's slot must not move")
	}
}

func TestTickSkipsAUserWhoAlreadyMetTodaysTarget(t *testing.T) {
	h := dueHarness()
	h.counter.totals["u1|2026-09-22"] = TargetSeconds // exactly 1800 counts as met (quests: total >= 1800)

	stats, err := h.svc.Tick(context.Background(), h.now)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Skipped != 1 || stats.Sent != 0 || len(h.sender.sent) != 0 {
		t.Errorf("stats = %+v, sent = %v", stats, h.sender.sent)
	}
	if want := h.hcm(23, 20, 0); h.queue.scores["u1"] != want.Unix() {
		t.Error("a skipped user is still re-slotted for tomorrow")
	}
	if !strings.Contains(strings.Join(h.log.calls, ";"), "counter.Total(u1,2026-09-22)") {
		t.Errorf("the counter must be read for the user's LOCAL date: %v", h.log.calls)
	}
}

func TestTickSendsWhenTheCounterIsUnavailable(t *testing.T) {
	h := dueHarness()
	h.counter.err = errors.New("redis down")
	stats, err := h.svc.Tick(context.Background(), h.now)
	if err == nil {
		t.Error("counter failure must be reported")
	}
	if stats.Sent != 2 {
		t.Errorf("sent = %d, want 2 — a missing counter means 'not met', not 'skip'", stats.Sent)
	}
}

func TestTickPrunesGoneSubscriptionsAndKeepsTheRest(t *testing.T) {
	h := dueHarness()
	h.sender.gone["https://push.example/ep1"] = true

	stats, err := h.svc.Tick(context.Background(), h.now)
	if err != nil {
		t.Fatalf("a gone subscription is normal, not an error: %v", err)
	}
	if stats.Pruned != 1 || stats.Sent != 1 {
		t.Errorf("stats = %+v", stats)
	}
	if subs := h.repo.subs["u1"]; len(subs) != 1 || subs[0].ID != "s2" {
		t.Errorf("subscriptions after prune = %+v, want only s2", subs)
	}
}

func TestTickReportsSendFailuresButContinues(t *testing.T) {
	h := dueHarness()
	h.sender.fail["https://push.example/ep1"] = true
	stats, err := h.svc.Tick(context.Background(), h.now)
	if err == nil || stats.Failed != 1 || stats.Sent != 1 {
		t.Errorf("err = %v, stats = %+v", err, stats)
	}
	if len(h.repo.subs["u1"]) != 2 {
		t.Error("a transient failure must not delete the subscription")
	}
}

func TestTickDropsUsersWithNoSubscriptionsOrNoRow(t *testing.T) {
	h := dueHarness()
	h.repo.subs["u1"] = nil
	h.queue.scores["ghost"] = h.now.Unix() - 10 // due, but has no users row

	stats, err := h.svc.Tick(context.Background(), h.now)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Due != 2 || stats.Sent != 0 {
		t.Errorf("stats = %+v", stats)
	}
	if _, ok := h.queue.scores["u1"]; ok {
		t.Error("u1 has no subscriptions and must leave the queue (UpdateSettings re-adds)")
	}
	if _, ok := h.queue.scores["ghost"]; ok {
		t.Error("a user with no row must leave the queue")
	}
}
```

- [ ] **Step 3: Run to confirm they fail**

```sh
go test ./internal/notify/... -run 'UpdateSettings|Tick' -timeout 60s
```
Expected: FAIL to compile — `undefined: NewService`, `SettingsRequest`, `ErrInvalidRequest`.

- [ ] **Step 4: Implement**

`backend/internal/notify/service.go`:
```go
package notify

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrInvalidRequest means the settings body failed validation; nothing was written.
var ErrInvalidRequest = errors.New("notify: invalid request")

// StudyCounter is notify's read-only view of the §4 daily:accumulated
// counter, keyed by the user's LOCAL date. *quests.RedisCounter satisfies it
// (cmd/api/main.go registers it), so notify never touches quests' key.
type StudyCounter interface {
	Total(ctx context.Context, userID, localDate string) (int64, error)
}

// SettingsRequest is the validated form of the §6.4 body. Subscription is
// nil when the client only moved the time.
type SettingsRequest struct {
	NotificationTime string
	Timezone         string // optional IANA name; "" keeps users.timezone
	Subscription     *Subscription
}

// SettingsResult is the §6.4 response plus next_reminder_at (additive).
type SettingsResult struct {
	Status           string `json:"status"`
	NotificationTime string `json:"notification_time"`
	NextReminderAt   string `json:"next_reminder_at"`
}

// TickStats is one worker pass, for logs and tests.
type TickStats struct {
	Due, Sent, Skipped, Pruned, Failed int
}

// Service owns the settings write path and the reminder pass.
type Service struct {
	repo    Repo
	queue   Queue
	sender  Sender
	counter StudyCounter
	now     func() time.Time
	payload Payload
}

// NewService wires the dependencies. sender may be nil when VAPID keys are
// absent: UpdateSettings still works; Tick must not be called (main.go does
// not start the worker).
func NewService(repo Repo, queue Queue, sender Sender, counter StudyCounter, now func() time.Time) *Service {
	return &Service{repo: repo, queue: queue, sender: sender, counter: counter, now: now, payload: DefaultPayload}
}

// UpdateSettings validates, writes users.notification_time (+timezone),
// stores the subscription, and ZADDs the next local send time.
func (s *Service) UpdateSettings(ctx context.Context, userID string, req SettingsRequest) (SettingsResult, error) {
	clock, err := NormalizeClock(req.NotificationTime)
	if err != nil {
		return SettingsResult{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	if req.Timezone != "" {
		if _, err := time.LoadLocation(req.Timezone); err != nil {
			return SettingsResult{}, fmt.Errorf("%w: unknown timezone %q", ErrInvalidRequest, req.Timezone)
		}
	}
	if sub := req.Subscription; sub != nil && (sub.Endpoint == "" || sub.P256dh == "" || sub.Auth == "") {
		return SettingsResult{}, fmt.Errorf("%w: push_subscription needs endpoint, p256dh and auth", ErrInvalidRequest)
	}

	if err := s.repo.UpdatePreferences(ctx, userID, clock, req.Timezone); err != nil {
		return SettingsResult{}, err
	}
	if req.Subscription != nil {
		if err := s.repo.SaveSubscription(ctx, userID, *req.Subscription); err != nil {
			return SettingsResult{}, err
		}
	}
	prefs, err := s.repo.Preferences(ctx, userID)
	if err != nil {
		return SettingsResult{}, err
	}
	next, err := NextSendTime(s.now(), clock, Location(prefs.Timezone))
	if err != nil {
		return SettingsResult{}, err
	}
	if err := s.queue.Schedule(ctx, userID, next); err != nil {
		return SettingsResult{}, err
	}
	return SettingsResult{Status: "updated", NotificationTime: clock, NextReminderAt: next.UTC().Format(time.RFC3339)}, nil
}

// Tick is one worker pass at now: pop due users, re-slot each for tomorrow
// FIRST (a crash mid-send then costs one reminder, not one every 30 s), skip
// those who already met today's target, send to every subscription, prune
// 404/410 ones, and drop users with nothing to send to. Per-user failures
// are collected and returned joined; the pass never stops early.
func (s *Service) Tick(ctx context.Context, now time.Time) (TickStats, error) {
	var stats TickStats
	due, err := s.queue.Due(ctx, now, DueBatchSize)
	if err != nil {
		return stats, err
	}
	stats.Due = len(due)

	var errs []error
	for _, userID := range due {
		prefs, err := s.repo.Preferences(ctx, userID)
		if errors.Is(err, ErrUserNotFound) {
			errs = appendIf(errs, s.queue.Remove(ctx, userID))
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		loc := Location(prefs.Timezone)

		next, err := NextSendTime(now, prefs.NotificationTime, loc)
		if err != nil {
			// Unparseable column value: keep the user out of a hot loop.
			next = now.Add(24 * time.Hour)
			errs = append(errs, fmt.Errorf("user %s: %w", userID, err))
		}
		if err := s.queue.Schedule(ctx, userID, next); err != nil {
			errs = append(errs, err)
			continue
		}

		total, err := s.counter.Total(ctx, userID, LocalDate(now, loc))
		if err != nil {
			// Unknown is not "met": send rather than silently skip.
			errs = append(errs, fmt.Errorf("user %s: %w", userID, err))
			total = 0
		}
		if total >= TargetSeconds {
			stats.Skipped++
			continue
		}

		subs, err := s.repo.Subscriptions(ctx, userID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if len(subs) == 0 {
			errs = appendIf(errs, s.queue.Remove(ctx, userID))
			continue
		}
		for _, sub := range subs {
			err := s.sender.Send(ctx, sub, s.payload)
			switch {
			case errors.Is(err, ErrSubscriptionGone):
				stats.Pruned++
				errs = appendIf(errs, s.repo.DeleteSubscription(ctx, sub.ID))
			case err != nil:
				stats.Failed++
				errs = append(errs, fmt.Errorf("user %s: %w", userID, err))
			default:
				stats.Sent++
			}
		}
	}
	return stats, errors.Join(errs...)
}

func appendIf(errs []error, err error) []error {
	if err != nil {
		return append(errs, err)
	}
	return errs
}
```

- [ ] **Step 5: Run to confirm they pass**

```sh
go vet ./internal/notify/... && go test ./internal/notify/... -run 'UpdateSettings|Tick' -v -count=1 -timeout 60s
```
Expected: eleven `--- PASS`.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend/internal/notify/fakes_test.go backend/internal/notify/service.go backend/internal/notify/service_test.go && git commit -m "notify: UpdateSettings and Tick — re-slot first, skip target-met, prune gone subscriptions" && cd backend
```
(Trailer as in Task 1.)

---

### Task 6: The worker loop and the §6.4 route

**Files:**
- Create: `backend/internal/notify/worker.go`, `backend/internal/notify/handler.go`
- Test: `backend/internal/notify/worker_test.go`, `backend/internal/notify/handler_test.go`

Error codes (§6.4 defines none; keep the merged `{"error": "<code>"}` convention): 400 `invalid_request` (binding failure or `ErrInvalidRequest`), 401 `unauthorized`, 404 `user_not_found` (`ErrUserNotFound` — the session outlived the row), 500 `internal_error`.

- [ ] **Step 1: Write the failing tests**

`backend/internal/notify/worker_test.go`:
```go
package notify

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunWorkerTicksAndStopsWhenTheContextIsCancelled(t *testing.T) {
	h := dueHarness()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		RunWorker(ctx, h.svc, 5*time.Millisecond)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for {
		if len(h.sender.sent) >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("worker never sent; calls: %v", h.log.calls)
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunWorker did not return after cancel")
	}
	if !strings.Contains(strings.Join(h.log.calls, ";"), "queue.Due(") {
		t.Error("the worker never asked the queue")
	}
}
```
(The fakes are not goroutine-safe; this test only reads `h.sender.sent` racily as a progress signal and asserts after `done`. If `go test -race` complains, guard `sent` with a mutex in `fakeSender`.)

`backend/internal/notify/handler_test.go`:
```go
package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

func router(svc *Service, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/settings/notifications", func(c *gin.Context) {
		if userID != "" {
			c.Set(auth.ContextUserID, userID)
		}
		c.Next()
	}, SettingsHandler(svc))
	return r
}

func post(t *testing.T, r *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/notifications", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// spec64Body is the §6.4 request verbatim.
const spec64Body = `{"notification_time": "20:00:00", "push_subscription": {"endpoint": "push_subscription_endpoint_string", "p256dh": "BNc5T...", "auth": "aX8v..."}}`

func TestSettingsHandlerAcceptsTheSpec64BodyAndAnswersTheSpec64Response(t *testing.T) {
	h := newHarness()
	w := post(t, router(h.svc, "u1"), spec64Body)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "updated" || body["notification_time"] != "20:00:00" {
		t.Errorf("body = %v", body)
	}
	if _, ok := body["next_reminder_at"].(string); !ok {
		t.Errorf("next_reminder_at missing: %v", body)
	}
	if subs := h.repo.subs["u1"]; len(subs) != 1 || subs[0].Endpoint != "push_subscription_endpoint_string" {
		t.Errorf("subscription not stored: %+v", subs)
	}
}

func TestSettingsHandlerAcceptsTimeOnlyAndOptionalTimezone(t *testing.T) {
	h := newHarness()
	w := post(t, router(h.svc, "u1"), `{"notification_time":"07:15","timezone":"Europe/London"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body)
	}
	if p := h.repo.prefs["u1"]; p.Timezone != "Europe/London" || p.NotificationTime != "07:15:00" {
		t.Errorf("prefs = %+v", p)
	}
}

func TestSettingsHandlerRejectsBadBodies(t *testing.T) {
	for name, body := range map[string]string{
		"not json":               `{`,
		"no notification_time":   `{"push_subscription":{"endpoint":"e","p256dh":"p","auth":"a"}}`,
		"bad clock":              `{"notification_time":"25:99"}`,
		"incomplete subscription": `{"notification_time":"20:00","push_subscription":{"endpoint":"e"}}`,
		"nested keys (not §6.4)": `{"notification_time":"20:00","push_subscription":{"endpoint":"e","keys":{"p256dh":"p","auth":"a"}}}`,
	} {
		h := newHarness()
		w := post(t, router(h.svc, "u1"), body)
		if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"invalid_request"}` {
			t.Errorf("%s: status = %d, body = %s", name, w.Code, w.Body)
		}
		if len(h.queue.scores) != 0 {
			t.Errorf("%s: a rejected request scheduled a reminder", name)
		}
	}
}

func TestSettingsHandlerRequiresAUserAndMapsMissingRowTo404(t *testing.T) {
	h := newHarness()
	if w := post(t, router(h.svc, ""), spec64Body); w.Code != http.StatusUnauthorized {
		t.Errorf("no user: status = %d, want 401", w.Code)
	}
	if w := post(t, router(h.svc, "ghost"), spec64Body); w.Code != http.StatusNotFound || w.Body.String() != `{"error":"user_not_found"}` {
		t.Errorf("missing row: status = %d, body = %s", w.Code, w.Body)
	}
}
```

- [ ] **Step 2: Run to confirm they fail**

```sh
go test ./internal/notify/... -run 'RunWorker|SettingsHandler' -timeout 60s
```
Expected: FAIL to compile — `undefined: RunWorker`, `SettingsHandler`.

- [ ] **Step 3: Implement the worker**

`backend/internal/notify/worker.go`:
```go
package notify

import (
	"context"
	"log"
	"time"
)

// RunWorker is the in-process reminder worker from spec §2.1 — the same shape
// as pet.RunHourly. It blocks until ctx is cancelled, running svc.Tick every
// `every` (PollInterval in production). Errors are logged and the loop
// continues: one bad subscription must not stop everyone else's reminder.
// Run exactly one worker per deployment: Tick's "re-slot then send" is safe
// against crashes, not against two processes popping the same member.
func RunWorker(ctx context.Context, svc *Service, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		at := svc.now()
		stats, err := svc.Tick(ctx, at)
		if err != nil {
			log.Printf("notify: tick at %s: %+v, errors: %v", at.UTC().Format(time.RFC3339), stats, err)
			continue
		}
		if stats.Due > 0 {
			log.Printf("notify: tick at %s: %+v", at.UTC().Format(time.RFC3339), stats)
		}
	}
}
```

- [ ] **Step 4: Implement the handler**

`backend/internal/notify/handler.go`:
```go
package notify

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// settingsRequest is the backend spec §6.4 POST /settings/notifications body:
// notification_time plus a FLAT push_subscription {endpoint, p256dh, auth}
// (not the browser's nested `keys` object — the PWA flattens it). timezone is
// additive (§3.2 users.timezone; optional).
type settingsRequest struct {
	NotificationTime string  `json:"notification_time" binding:"required"`
	Timezone         string  `json:"timezone"`
	PushSubscription *pushSub `json:"push_subscription"`
}

type pushSub struct {
	Endpoint string `json:"endpoint" binding:"required"`
	P256dh   string `json:"p256dh" binding:"required"`
	Auth     string `json:"auth" binding:"required"`
}

// SettingsHandler serves POST /api/v1/settings/notifications. It must be
// mounted behind auth.Require().
func SettingsHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var req settingsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}
		in := SettingsRequest{NotificationTime: req.NotificationTime, Timezone: req.Timezone}
		if req.PushSubscription != nil {
			in.Subscription = &Subscription{Endpoint: req.PushSubscription.Endpoint, P256dh: req.PushSubscription.P256dh, Auth: req.PushSubscription.Auth}
		}

		res, err := svc.UpdateSettings(c.Request.Context(), userID, in)
		switch {
		case errors.Is(err, ErrInvalidRequest):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		case errors.Is(err, ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			c.JSON(http.StatusOK, res)
		}
	}
}
```

- [ ] **Step 5: Run to confirm they pass**

```sh
go vet ./internal/notify/... && go test ./internal/notify/... -run 'RunWorker|SettingsHandler' -v -count=1 -timeout 60s
```
Expected: five `--- PASS`. (Gin's validator dives into the non-nil `*pushSub`, so `"incomplete subscription"` fails binding; `"nested keys"` fails because `p256dh`/`auth` are required at the top of `push_subscription`. If either instead reaches the service, `UpdateSettings`'s own validation still returns `ErrInvalidRequest` → 400 — the test passes either way, by design.)

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend/internal/notify/worker.go backend/internal/notify/worker_test.go backend/internal/notify/handler.go backend/internal/notify/handler_test.go && git commit -m "notify: 30 s worker loop and POST /settings/notifications with the §6.4 body" && cd backend
```
(Trailer as in Task 1.)

---

### Task 7: Wire it in `main.go`

**Files:**
- Modify: `backend/cmd/api/main.go`

- [ ] **Step 1: Confirm the pet slice's wiring is present**

```sh
grep -n 'studyCounter\|pet.RunHourly\|guarded := ' cmd/api/main.go
```
Expected: three hits — `studyCounter := quests.NewRedisCounter(rdb)`, `go pet.RunHourly(ctx, petSvc)`, `guarded := v1.Group("", auth.Require(tokens, sessions))`. **If `studyCounter` or `pet.RunHourly` is missing, pet (order 4) has not merged: stop and report it** — do not substitute your own counter or cron pattern.

- [ ] **Step 2: Add the notify wiring**

Add the import `"github.com/HendrixNguyen/English-Training-Harness/backend/internal/notify"` to the `internal/…` group (alphabetical: after `health`, before `pet`). Directly **after** the `go pet.RunHourly(ctx, petSvc)` line, add:
```go
	// Spec §2.1 reminder worker, in-process, polling the §4 queue:webpush:delay
	// ZSET every 30 s. It starts only when both VAPID keys (spec §9) are set;
	// without them settings are stored but nothing is sent.
	var pushSender notify.Sender
	if cfg.VAPIDPublicKey != "" && cfg.VAPIDPrivateKey != "" {
		sender, err := notify.NewWebPushSender(cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject)
		if err != nil {
			log.Fatalf("notify: %v", err)
		}
		pushSender = sender
	}
	notifySvc := notify.NewService(
		notify.NewPgRepo(pg.Pool),
		notify.NewRedisQueue(rdb),
		pushSender,
		studyCounter, // notify reads the daily counter only through this interface
		time.Now,
	)
	if pushSender != nil {
		go notify.RunWorker(ctx, notifySvc, notify.PollInterval)
	} else {
		log.Printf("notify: VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY unset; reminder settings are stored but no Web Push is sent")
	}
```
Then, after the last existing `guarded.…` route line (google's `/integrations/google/sync` if it has landed), add:
```go
	guarded.POST("/settings/notifications", notify.SettingsHandler(notifySvc))
```
Do not move, rename or reorder anything else in `main.go`.

- [ ] **Step 3: Confirm build, vet and the whole suite**

```sh
go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s
grep -n 'notify.RunWorker\|notify.SettingsHandler\|/settings/notifications' cmd/api/main.go
```
Expected: clean build/vet; `ok` for every package (integration tests skip); the grep has three hits.

- [ ] **Step 4: Commit**

```sh
cd .. && git add backend/cmd/api/main.go && git commit -m "notify: mount /settings/notifications and start the reminder worker when VAPID keys exist" && cd backend
```
(Trailer as in Task 1.)

---

### Task 8: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (`**notify**` bullet)

- [ ] **Step 1: Replace the `notify` bullet**

Replace the line beginning `- **notify** — Web Push, \`queue:webpush:delay\` ZSET, in-process cron.` with:
```
- **notify** — Web Push reminders (wire contract = backend spec §6.4; queue = §4 `queue:webpush:delay`; worker = §2.1). `POST /api/v1/settings/notifications` (behind `auth.Require()`) takes `{notification_time, timezone?, push_subscription?{endpoint, p256dh, auth}}` — **flat** keys per §6.4, `timezone` additive — validates everything first (400 `invalid_request`), then `UPDATE users SET notification_time, timezone` (404 `user_not_found` when the row is gone), stores the subscription in `push_subscriptions` once per `endpoint` (a CTE moves an endpoint away from any other user — no unique index in §3.2), and `ZADD`s `store.WebPushDelayQueueKey` with the next occurrence of the time in the user's zone (DST-safe via `time.Date`), answering `{status: "updated", notification_time, next_reminder_at}` (last field additive, RFC3339 UTC). `RunWorker` (pet's `RunHourly` shape, `PollInterval` 30 s, **one per deployment**) runs `Service.Tick`: `ZRANGEBYSCORE -inf now LIMIT 100`, and per user **re-slots to tomorrow first** (a crash costs one reminder, never a re-fire loop), reads today's local-date total through `StudyCounter` (= `quests.RedisCounter.Total`; an error counts as "not met") and skips at ≥ 1800 s, sends `DefaultPayload` `{title, body, url:"/"}` to every subscription via `webpush-go` v1.4.0 (RFC 8291 `aes128gcm` + RFC 8292 VAPID, TTL 1 h), deletes subscriptions on 404/410, `ZREM`s users with no subscriptions or no `users` row, and joins per-user errors without stopping. `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY` (§9) are **optional at boot**: the route always works, the worker starts only when both are set (`main.go` logs which); `VAPID_SUBJECT` (not in §9) defaults to `mailto:admin@example.com`. Tests are pure (fakes with a call log; the sender is tested against an `httptest` push endpoint with real generated VAPID + P-256 keys); `TestIntegrationScheduleAndSubscriptionRoundTrip` is gated on `TEST_DATABASE_URL`+`TEST_REDIS_URL` (skips locally, must pass in CI).
```

- [ ] **Step 2: Validate and commit**

```sh
cd .. && python3 tools/harness/cli.py validate && grep -c 'webpush-go' harness/CODEMAP.md
```
Expected: exit 0; count ≥ 1.
```sh
git add harness/CODEMAP.md && git commit -m "codemap: notify — §6.4 settings route, re-slot-first worker, optional VAPID" && cd backend
```
(Trailer as in Task 1.)

---

## Verification

Run from the worktree root (`cd backend` where shown). Every command is local — no Docker, no push service.

```sh
cd backend && go build ./... && go vet ./...
# expect: no output

env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 120s
# expect: ok for every package incl. internal/notify and internal/config; no FAIL — nothing needs a live service

go test ./internal/notify/... -run 'UpdateSettings|Tick' -v -count=1 -timeout 60s
# expect: eleven --- PASS — schedule tonight/tomorrow, reject-before-write, re-slot before send, not-yet-due untouched, target-met skip on the LOCAL date, counter error still sends, prune on gone, transient failure keeps the row, no-subs/no-row leave the queue

go test ./internal/notify/... -run 'WebPushSender' -v -count=1 -timeout 60s
# expect: four --- PASS — VAPID header, aes128gcm, TTL, 404/410 → gone, 429 → error, both keys required

go test ./internal/notify/... -run 'NextSendTime|NormalizeClock' -v -count=1 -timeout 60s
# expect: --- PASS incl. the America/New_York spring-forward case (23 h, wall clock kept)

go test ./internal/notify/... -run 'SettingsHandler|RunWorker' -v -count=1 -timeout 60s
# expect: five --- PASS — the §6.4 body verbatim is accepted and answered

grep -n 'webpush-go v1.4.0' go.mod
# expect: one hit (or the version recorded in Task 4's commit)

grep -n '"notification_time"\|"push_subscription"\|"p256dh"\|"auth"\|"endpoint"' internal/notify/handler.go
# expect: 5 hits — the §6.4 request, flat keys

grep -rn --include='*.go' '"keys"' internal/notify/
# expect: no hits outside handler_test.go (the nested browser shape is only asserted as rejected)

grep -n '"status"\|"notification_time"\|"next_reminder_at"' internal/notify/service.go
# expect: 3 hits — §6.4 response + the additive field

grep -n 'store.WebPushDelayQueueKey' internal/notify/queue.go
# expect: 3 hits — notify never hand-builds the §4 key

grep -rn 'daily:accumulated\|DailyAccumulatedKey\|daily_progress' internal/notify/
# expect: no hits — the counter is read only through StudyCounter

grep -rn '"github.com/HendrixNguyen/English-Training-Harness/backend/internal/quests"' internal/notify/
# expect: no hits — notify does not import quests (main.go injects *quests.RedisCounter)

grep -rn 'Getenv("DATABASE_URL")\|Getenv("REDIS_URL")' internal/notify/
# expect: no hits — integration gating is on TEST_* only

grep -c '^func TestIntegration' internal/notify/integration_test.go
# expect: 1 — CI's backend-integration job counts it and fails if it skips there

grep -n 'VAPID_PUBLIC_KEY\|VAPID_PRIVATE_KEY\|VAPID_SUBJECT' internal/config/config.go .env.example
# expect: 3 hits in each

grep -n 'notify.RunWorker\|notify.SettingsHandler\|studyCounter,' cmd/api/main.go
# expect: 3 hits

cd .. && python3 tools/harness/cli.py validate; echo exit=$?
# expect: exit=0

git log --oneline main..HEAD
# expect 8 commits, one per task, each with the Co-Authored-By trailer

git status --short
# expect: clean
```

With a dev stack (`COMPOSE_PROJECT_NAME=<slug>` and non-default `POSTGRES_PORT`/`REDIS_PORT` in `backend/.env`, `docker compose up -d --wait`; `make down` after): export `TEST_DATABASE_URL`/`TEST_REDIS_URL` and run `cd backend && make test-integration` — `TestIntegrationScheduleAndSubscriptionRoundTrip` must PASS (it cleans up its two users and its ZSET members). After pushing the branch: `gh run list --branch <branch>` must show `backend-unit`, `backend-integration` and `harness-tooling` green.

## Notes and open questions

- **Additive fields vs. §6.4.** Request `timezone` and response `next_reminder_at` are not in §6.4. Without `timezone` the route can only schedule in whatever `users.timezone` already holds ('UTC' for a user who skipped onboarding); without `next_reminder_at` the client cannot show "reminder at 20:00 tonight". Both are optional/extra, so a §6.4-exact client works unchanged. The spec owner should add them to §6.4 or say no.
- **Flat `push_subscription` keys.** §6.4 shows `{endpoint, p256dh, auth}`; the browser's `PushSubscription.toJSON()` gives `{endpoint, keys:{p256dh, auth}}`. The PWA flattens; the nested shape is rejected with 400 (tested), so the mismatch is caught in development, not by a silent NULL.
- **Skip check reads the Redis counter, not `daily_progress`.** Same source of truth quests and pet use (CODEMAP: "Redis is the source of truth for the day, Postgres is the record"); a counter read error counts as "not met" so an outage never silences reminders. Trade-off: a user who reached 1800 s only via a `daily_progress` row (none can today) would still be nudged.
- **Exactly one worker.** `Tick` re-slots before sending, which is crash-safe but not multi-process-safe (two replicas can both pop a member before either re-slots). §2.1/§8 deploy one binary. If Railway ever runs two, add a `SET NX` lock per user or a Lua `ZPOPMIN`-with-condition — recorded, not built.
- **`VAPID_SUBJECT` is not in §9** but push services (Mozilla's in particular) require a `mailto:`/`https:` `sub` in the VAPID JWT. Default `mailto:admin@example.com` boots; replace before production. Should §9 list it?
- **`push_subscriptions` has no UNIQUE(endpoint) in §3.2**, so dedupe is a CTE rather than `ON CONFLICT`, and two simultaneous first-subscribes from one browser could insert twice (both receive; the first 410 prunes one). A `UNIQUE (endpoint)` index in a later migration would make it `ON CONFLICT (endpoint) DO UPDATE SET user_id, p256dh, auth`; flagged for the spec owner, not added here (schema changes stay with their own decision — see the google plan's `0002`).
- **A time-only update keeps the user in the queue even with no subscription** until the first `Tick` finds no subscriptions and `ZREM`s them; `UpdateSettings` with a subscription re-adds. Harmless, one wasted lookup per day.
- **Push TTL 1 h, urgency normal.** A reminder about "now" is noise tomorrow; §6.4/§4 say nothing. Tunable constant.
- **Payload is fixed text** (`DefaultPayload`). run-01's "adaptive reminder timing and pre-decay rescue push" idea is where per-user copy (pet health, streak) belongs; `Service.payload` is a field so that slice can inject.
- **`Location`/`LocalDate`/`TargetSeconds` duplicate quests' by design** (package boundary). Third copy → hoist.
- **VAPID public key to the client** is a frontend concern (Nuxt runtime config, per the idea and `_run.md`): no `GET /settings/vapid-public-key` is added (not in §7).

## Execution summary

Branch `harness/2026-09-23-high-notify-web-push-subscriptions-and-delayed-reminder-queue`, worktree `.worktrees/notify-web-push-subscriptions-and-delayed-reminder-queue`, all 8 tasks implemented task-by-task with TDD (failing test → implement → pass → commit), 8 commits on top of `main`, all with the `Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>` trailer. Working tree clean, `python3 tools/harness/cli.py validate` exits 0.

**Pre-flight (as instructed):** re-read `backend/cmd/api/main.go` on `main` before Task 7. It already had pet's wiring (`studyCounter := quests.NewRedisCounter(rdb)`, `go pet.RunHourly(ctx, petSvc)`, `guarded := v1.Group(...)`) exactly as the plan expected, and — as flagged — neither google's nor onboarding's routes were present yet (both still unmerged branches). No stop condition was hit. Added the notify block directly after `go pet.RunHourly(ctx, petSvc)` and the route after the last existing guarded line (`/pet/revive`), preserving intent; a future merge of google/onboarding will conflict on this file as expected, and that is the human's problem to resolve, not this executor's.

### Deviations from the plan (all recorded, none silent)

1. **Fixed a wrong assertion in the plan's own DST test.** `TestNextSendTimeKeepsWallClockAcrossDST` asserted `got.Sub(now) == 23*time.Hour`. I verified independently with a standalone Go program against Go's tzdata: 21:00 EST (UTC-5) on 2026-03-07 to 20:00 EDT (UTC-4) on 2026-03-08 (the US spring-forward day) is **22h** elapsed, not 23h — the spring-forward day loses an hour, and 23h would only hold if the base time were 20:00 rather than 21:00 the day before. My `NextSendTime` implementation (resolving the local wall-clock time via `time.Date` and letting Go pick the correct UTC offset) was already correct; only the hardcoded test assertion was wrong. Fixed the test to assert 22h, with a comment explaining the arithmetic, and confirmed all other assertions in that test (day, hour, offset -14400) already passed.
2. **Fixed `.env.example`'s VAPID key generator reference**, per the plan's own contingency note. `webpush-go` v1.4.0 has no `cmd/webpush-go` binary (confirmed via `go doc` and inspecting the module cache — only `example/` and `.github/` exist); replaced the false command with a pointer to `npx web-push generate-vapid-keys` / any VAPID generator.
3. **Added a mutex to `fakeSender.sent`** in `fakes_test.go` (and a `Sent()` accessor used only by `worker_test.go`), per the plan's own contingency note ("if `go test -race` complains, guard `sent` with a mutex"). `-race` did complain (a real read/write race between `RunWorker`'s background goroutine and the test goroutine polling `h.sender.sent`); CI's `backend-unit` job runs `go test ./... -count=1` without `-race` so this would not have failed CI, but it is a genuine race and trivial to fix, so I fixed it rather than leaving it.
4. **`daily:accumulated` grep check in Verification is stricter than the plan's own template code.** The plan's own Task 5/Task 2 code includes doc comments mentioning `daily:accumulated` (in `StudyCounter`'s and `LocalDate`'s doc comments, copied verbatim from the plan) — `grep -rn 'daily:accumulated\|DailyAccumulatedKey\|daily_progress' internal/notify/` therefore finds 2 comment-only hits, not 0. No code in `internal/notify` touches the Redis key or `daily_progress` table directly; the counter is read exclusively through the `StudyCounter` interface, satisfying the check's actual intent. Left the comments as-is since they're accurate documentation and part of the plan's own text.
5. Task 4's `.env.example` fix (item 2) is bundled into the Task 4 commit rather than a separate one, since it's the direct output of that task's own "verify after Task 4" instruction.

### Two spec discrepancies (deliberate, flagged for the reviewer)

1. **Flat `push_subscription.{endpoint, p256dh, auth}` vs. backend spec §6.4's nested `keys.{p256dh, auth}`.** Implemented the **plan's** flat shape as instructed (the plan is the contract here), and the nested browser-native shape is explicitly rejected with 400 `invalid_request` (tested in `handler_test.go`'s `"nested keys (not §6.4)"` case) so the mismatch surfaces immediately in development rather than as a silent NULL. This is the same class of bug as the `token`/`access_token` auth mismatch mentioned in the task brief. Already documented prominently in this plan's own *Notes and open questions* section ("Flat `push_subscription` keys").
2. **`VAPID_SUBJECT` is not in backend spec §9's environment checklist**, but `webpush-go`'s `Options.Subscriber` (the VAPID JWT `sub` claim) is effectively required by push services (Mozilla's in particular). Implemented exactly as the plan directs: optional env var, defaulting to `mailto:admin@example.com`, both recorded in `config.go`'s doc comment and `.env.example`. Already documented in this plan's *Notes and open questions* ("`VAPID_SUBJECT` is not in §9").

### Verification (plan's Verification section, run from the worktree)

- `go build ./...` / `go vet ./...` — clean, no output.
- `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 120s` — `ok` for every package (airouter, auth, config, health, notify, pet, quests, store); no service env vars needed.
- `go test ./internal/notify/... -run 'UpdateSettings|Tick' -v` — 11 named tests pass (a 12th incidental match, `TestRunWorkerTicksAndStopsWhenTheContextIsCancelled`, also matches the `Tick` substring and passes).
- `go test ./internal/notify/... -run 'WebPushSender' -v` — 4/4 pass, including the VAPID `Authorization` header check (webpush-go v1.4.0 sends `vapid t=…,k=…`, matching the plan's primary expectation — no relaxation needed).
- `go test ./internal/notify/... -run 'NextSendTime|NormalizeClock' -v` — 3/3 pass incl. the corrected DST case.
- `go test ./internal/notify/... -run 'SettingsHandler|RunWorker' -v` — 5/5 pass.
- All the plan's `grep` checks pass with the one noted exception (item 4 above, which matches the check's intent, not its literal zero-hit wording).
- `python3 tools/harness/cli.py validate` — exit 0.
- `git log --oneline main..HEAD` — 8 commits, each with the trailer. `git status --short` — clean.
- Dev stack: `COMPOSE_PROJECT_NAME=notif`, `POSTGRES_PORT=5445`, `REDIS_PORT=6393` in a scratch `backend/.env`, `docker compose up -d --wait --wait-timeout 120` (both containers healthy). `TEST_DATABASE_URL=postgres://english:english@localhost:5445/english?sslmode=disable`, `TEST_REDIS_URL=redis://localhost:6393/0`, then `make test-integration` (documented command, run exactly as written) — every package's integration tests pass, including `TestIntegrationScheduleAndSubscriptionRoundTrip`, with `-p 1` as documented.

### Runtime proof

**1. Boot + one real authenticated request end to end.** Built `go build -o /tmp/notif-api-test ./cmd/api`, ran it on port 18099 against the dev-stack Postgres/Redis with no VAPID keys set. Logs confirmed: migrations applied, `notify: VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY unset; reminder settings are stored but no Web Push is sent`, and `POST /api/v1/settings/notifications` mounted. `curl /healthz` → `{"postgres":"ok","redis":"ok","status":"ok"}` (200). `curl` the settings route with no `Authorization` → 401 `{"error":"unauthorized"}`. Then, via a throwaway helper program (built inside `backend/cmd/notiftest/`, deleted before finishing — not part of the plan's file list) that inserted a real user, minted a real JWT with `auth.NewTokenIssuer`, wrote a real session with `auth.NewRedisSessionStore.Put`, and made a real authenticated HTTP `POST` against the running server with `{"notification_time":"20:00:00","timezone":"Asia/Ho_Chi_Minh","push_subscription":{...}}`: got HTTP 200 `{"status":"updated","notification_time":"20:00:00","next_reminder_at":"2026-09-23T13:00:00Z"}`, and `ZSCORE queue:webpush:delay <user_id>` in real Redis returned `1790168400`, which **exactly matches** the independently-computed next occurrence of 20:00 in Asia/Ho_Chi_Minh (`2026-09-23T20:00:00+07:00` = unix `1790168400`) — proving the timezone is applied correctly end to end through the real HTTP → Service → Postgres → Redis path.

**2. The delayed-queue timing claims, against real Redis, real Postgres, and a real `httptest` fake push endpoint** (same throwaway helper, using `notify.NewRedisQueue`, `notify.NewPgRepo`, and a real `notify.WebPushSender` with freshly generated VAPID keys — only the `StudyCounter` was a trivial always-zero fake, since that dependency isn't the subject of this proof):
  - **Enqueue:** scheduled userA 2s in the future; `ZSET queue:webpush:delay` showed score `1790134795` (exactly `fireAt.Unix()`); `ZRANGEBYSCORE` at enqueue time returned `[]` (not yet due).
  - **Tick before the score fires nothing:** `Tick(now=2026-09-23T03:39:53Z)` → `{Due:0 Sent:0 ...}`, fake server saw 0 hits.
  - **Tick at/after the score fires exactly once and re-slots:** `Tick(now=2026-09-23T03:39:55Z)` → `{Due:1 Sent:1 Skipped:0 Pruned:0 Failed:0}`, fake server saw exactly 1 hit; `ZSCORE` afterward was `1790193600` (tomorrow 20:00 UTC in this test's timezone setup), strictly greater than the tick time — the member was moved forward, not removed outright, matching the "re-slot before send" design.
  - **A second Tick right after does NOT re-fire:** `Tick(now=tick+1s)` → `{Due:0 Sent:0 ...}`, fake server hit count **stayed at 1**. This is the specific no-duplicate-fire assertion requested: two ticks past the original fire time produced exactly one push, ever.
  - **410 Gone → cleaned up, not retried forever:** a second user's subscription pointed at the fake server's `/gone` path (always 410). `Tick` on its due entry produced `{Due:1 Sent:0 Skipped:0 Pruned:1 Failed:0}`; a follow-up `repo.Subscriptions(ctx, userB)` query against real Postgres returned `[]` — the row was deleted, not left to be retried on every future tick. (The plan does cover this: `Tick`'s `ErrSubscriptionGone` branch calls `DeleteSubscription`, unconditionally, not a "not covered, invented" case.)
  - Full raw output of this proof is reproducible; it was captured directly during execution and is summarized above with exact stats structs and ZSET scores rather than paraphrased.

### CI

Pushed `harness/2026-09-23-high-notify-web-push-subscriptions-and-delayed-reminder-queue`. `gh pr create` failed exactly as anticipated: `pull request create failed: GraphQL: must be a collaborator (createPullRequest)` (the `gh` account authenticated here, `hendrixnguyen-optisigns`, is not a collaborator on `HendrixNguyen/English-Training-Harness`; attempted once, noted, moved on — also could not create labels beforehand, same root cause, HTTP 404 on the labels endpoint). CI run triggered by the push: **all three jobs green** — `backend-unit`, `backend-integration`, `harness-tooling`. Run: https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35815345858 (conclusion: success).

### Cleanup

Deleted `backend/cmd/notiftest/` (throwaway verification helper, never committed). Killed the test API server (`/tmp/notif-api-test`, PID 61084) — note: plain `kill`/SIGTERM did not stop it because `main.go`'s `signal.NotifyContext` is only consulted by the background workers (`pet.RunHourly`, `notify.RunWorker`), not by the blocking `r.Run()` HTTP listener, so `SIGKILL` was required; this is a pre-existing gap in `main.go` unrelated to this plan, noted here rather than silently worked around. `docker compose down` for the `notif` project (both containers stopped and removed); deleted the scratch `backend/.env`. Final checks: `pgrep -fl exe/api` and `pgrep -fl notif-api-test`/`notiftest` — none found; `docker ps` shows only pre-existing, unrelated containers (`scio3-redis-1`, `scio3-mongo-1`), not this task's. `git status --short` in the worktree is clean.

Definition of done: (1) builds — yes; (2) whole suite passes from a clean shell incl. `make test-integration -p 1` — yes; (3) boots and serves a real request — yes; (4) every documented command works exactly as documented — yes; (5) CI green on the pushed branch — yes (run linked above); (6) this summary. Setting `status=done`.

### Amendment: SSRF blocker fix (2026-09-23)

`harness/plans/2026-09-23-push-subscription-endpoint-is-an-unvalidated-user-supplied-u.md` (blocker, `amends` this plan) landed 5 more commits on this same branch/worktree, on top of `547c0ab` (this plan's final commit): `8d283bf` (`ValidateEndpoint` + `forbiddenAddr`), `c761fe3` (`UpdateSettings` refuses hostile endpoints before writing), `1c647b3` (guarded dial-time HTTP client, no redirects, `Send` validates first), `f7a02c1` (`Tick` prunes forbidden-endpoint rows), `b85189a` (CODEMAP). All five carry the `Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>` trailer.

The fix closes the SSRF hole the review found: `push_subscription.endpoint` (`handler.go`/`service.go`) was an arbitrary client-supplied URL handed straight to `http.Client.Do` in `push.go`, reachable by an authenticated user pointing it at `169.254.169.254` or any internal address, fired once per user per day at a time the user controls. It is now validated at both doors — `UpdateSettings` (400 `invalid_request`, zero writes) and again in `Send` — and enforced at TCP dial time via `net.Dialer.Control` on the *resolved* address (DNS-rebinding-safe, closes the check-then-connect window), with `Proxy: nil` so the guard sees the real destination and `CheckRedirect` returning `http.ErrUseLastResponse` so a permitted host cannot 302 into a private range. A stored row the guard refuses fails `Send` with `ErrForbiddenEndpoint`, which `Tick` prunes exactly like 404/410.

Full verification, mutation-testing, live-proof, and CI evidence for this amendment is in that plan's own `## Execution summary` — not duplicated here. In short: whole suite green (`go build`/`go vet`/full `go test ./...` without services), `go test ./internal/notify/... -race` clean, all 17 named SSRF tests PASS including the reviewer's exact reproduction, all three Task 3 mutations (`guardDial` removed, `CheckRedirect` removed, `Send`'s `ValidateEndpoint` removed) and the Task 1/2/4 mutations broke the intended assertions and were reverted, a standalone live-proof program plus a real dev-stack `curl` against the running API confirmed the hostile endpoint is refused (0 listener hits, subscription pruned) and the legitimate endpoint still works end to end, and CI is green on the new push (run `35819788843`, conclusion `success`). `python3 tools/harness/cli.py blockers` against this plan now reports none.
