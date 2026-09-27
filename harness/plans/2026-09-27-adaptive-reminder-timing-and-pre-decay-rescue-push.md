---
idea: harness/ideas/2026-09-22-run-01/adaptive-reminder-timing-and-pre-decay-rescue-push.md
status: executing
priority: low
merged: false
branch: harness/2026-09-27-low-adaptive-reminder-timing-and-pre-decay-rescue-push
worktree: .worktrees/adaptive-reminder-timing-and-pre-decay-rescue-push
---
# Pre-decay rescue push: one "your plant needs {n} more minutes" Web Push at local 22:00 on an unmet day — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Feature team — ticket **F1** of 2026-09-27. **Estimate:** 4 h. **Branch:** `harness/2026-09-27-low-adaptive-reminder-timing-and-pre-decay-rescue-push`.

**Idea:** `harness/ideas/2026-09-22-run-01/adaptive-reminder-timing-and-pre-decay-rescue-push.md` — **rescue-push half only** (its 2026-09-24 evaluation split it). The learned reminder time (`users.reminder_mode`, `learned_notification_time`, `daily_progress.first_activity_at`, the nightly median job, the settings-endpoint change) is a separate, later plan — **none of that is built here, and no migration is added today.** Backend only; **no design doc**.

**Depends on (must be on `origin/main` before execution):** the 2026-09-26 daily code PR (branches listed in the plan's Notes) — if missing when you start, stop and report. Tasks 1–3 touch only files no unmerged branch edits (`backend/internal/notify/rescue*.go`, new; `backend/internal/store/keys.go`, append); Task 4 (the one-line wiring in `backend/cmd/api/main.go`, which seven unmerged branches edit) is written to run after `origin/main` contains `harness/daily-2026-09-26`.

**Goal:** At local hour 22 on a day whose `daily:accumulated` total is below 1800 s, a subscribed learner receives exactly one Web Push — "Cây của cậu cần thêm {n} phút hôm nay" — and never one on a met day, never a second one that day, never one without a subscription.

**Architecture:** a second in-process hourly job in `notify` — `Rescue` (new file `rescue.go`) with `pet.RunHourly`'s shape (`:00` UTC timer, every candidate judged in Go by its own `users.timezone`) and `Service.Tick`'s delivery semantics (send to every subscription, prune 404/410/forbidden, join per-user errors, never stop early). It reads the day's seconds only through the existing `StudyCounter` interface (`quests.RedisCounter.Total`), claims a once-per-day flag `rescue:{user_id}:{YYYY-MM-DD}` with `SET NX EX` (48 h, new key builder in `store/keys.go`), and sends through the existing `Sender`. It does **not** touch `Service`, `Tick`, `Repo`, `Queue` or the fakes in `fakes_test.go` — B3's bug plan is editing `handler.go`, `service.go`, `repo.go` today; this plan adds only new files plus one method on `*PgRepo` declared in `rescue.go`. `pet` is not consulted: notify has no interface to it, and pet's only reader (`Service.Ensure`) also INSERTs — so the idea's `· sức khỏe {health}%` segment is dropped (Notes).

**Wire contract (spec §6.4 unchanged; push payload = `notify.Payload {title, body, url}`, same as `DefaultPayload`):**
```
Title: "Tớ cần cậu thêm {n} phút nữa"
Body:  "Cây của cậu cần thêm {n} phút hôm nay"
URL:   "/"
n = ceil((1800 − total) / 60), never below 1
```
Redis (§4 style, not yet in the spec — Notes): `rescue:{user_id}:{YYYY-MM-DD}` = `"1"`, `SET NX EX 172800`; the date is the user's local date, the same one `daily:accumulated` uses.

## Global Constraints
- Work in `.worktrees/<slug>`; Go from `backend/`; `rg`/`timeout` not installed (`grep -n`, `go test -timeout`); integration tests via `COMPOSE_PROJECT_NAME=<slug> POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose up -d --wait` … `make down` with the same project name.
- **Do not edit** `notify/service.go`, `handler.go`, `repo.go`, `queue.go`, `fakes_test.go`, `service_test.go`, `worker.go` — B3 is in them. New behaviour lives in `rescue.go`, `rescue_test.go`, `rescue_integration_test.go`; the only pre-existing files touched are `store/keys.go` (append only) and, in Task 4, `cmd/api/main.go` (add lines, move nothing).
- **No migration, no new columns, no spec-document edits** (the `project-base/` files are on unmerged branches — Task 4 puts the spec sentences in CODEMAP and the Execution summary's Follow-ups instead).
- One rescue per user per local day, enforced by the Redis flag **claimed after the "met" check and before the send** (a send failure after the claim costs that day's rescue — same trade-off as `Tick`'s re-slot-first; a claim before the met check would be wrong: it would burn the day's flag for a user who then never needed it — harmless but misleading in the stats).
- A Redis error (counter or flag) **skips the user** and is reported once in the sweep's joined error; it never sends (unlike `Tick`, whose "unknown is not met → send" rule is right for a scheduled reminder and wrong for a rescue that must fire at most once).
- Local hour 22 means `now.In(loc).Hour() == 22` — the tick between 22:00 and 22:59 local. Half-hour and quarter-hour zones (Kolkata, Kathmandu, Chatham) still hit exactly one `:00` UTC tick inside that hour; a zone that skips hour 22 on a DST day simply gets no rescue that day (accepted; note it in CODEMAP).
- Constants: `RescueLocalHour = 22`, `store.RescueTTL = 48 * time.Hour`; `TargetSeconds` (1800) reused from `schedule.go`.
- `gofmt -l internal/notify internal/store cmd/api` empty; `go vet` clean; `-race` clean (a third long-lived goroutine joins `pet.RunHourly` and `notify.RunWorker`).

## Review Focus
1. Hour window: a user in `Asia/Ho_Chi_Minh` is in the window at the 15:00 UTC tick and not at 14:00/16:00; a user in `America/New_York` (EDT in September) at the 02:00 UTC tick of the *next* UTC day, and the local date used for the counter and the flag is the New York date (`2026-09-22`, not `2026-09-23`).
2. Once per day: two sweeps in the same local hour 22 → one send, one `AlreadySent`; the flag key is `rescue:u1:2026-09-22`.
3. Met day → no send **and no claim** (the flag store's call log is empty).
4. Counter error → `Skipped`, no claim, no send, the sweep's error mentions the user; flag error → `Skipped`, no send.
5. No subscription → silent (candidates query already excludes them; the in-Go `len(subs) == 0` branch is still there for the race and is not an error).
6. Minutes: total 0 → 30; 1 → 30; 60 → 29; 1799 → 1; the copy is exactly the strings above.
7. `RunRescue` stops on context cancel and logs once per sweep with errors; the `:00` alignment mirrors `pet.NextTopOfHour`.
8. Nothing in `service.go` / `repo.go` / `handler.go` / `fakes_test.go` changed (`git diff --stat origin/main -- internal/notify` lists only `rescue*.go`).

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/store/keys.go` | append `RescueTTL`, `RescueKey(userID, localDate string)` |
| `backend/internal/notify/rescue.go` (new) | `RescueLocalHour`, `RescueCandidate`, `RescueRepo`, `(*PgRepo).RescueCandidates`, `RescueFlags` + `RedisRescueFlags`, `RescueMinutes`, `RescuePayload`, `RescueStats`, `Rescue`/`NewRescue`/`Sweep`, `NextTopOfHour`, `RunRescue` |
| `backend/internal/notify/rescue_test.go` (new) | `fakeRescueRepo` (embeds `*fakeRepo`), `fakeRescueFlags`, the unit tests below |
| `backend/internal/notify/rescue_integration_test.go` (new) | `TestIntegrationRescueCandidatesAndFlagClaim` (gated on `TEST_DATABASE_URL`/`TEST_REDIS_URL`) |
| `backend/cmd/api/main.go` (Task 4, gated) | build `Rescue`, `go notify.RunRescue(ctx, rescue)` beside `RunWorker` |
| `harness/CODEMAP.md` `notify` + `store` paragraphs | the job, the key, the dropped health, the DST note |

## Tasks

### Task 1: the `rescue:{user}:{date}` key builder

**Files:** `backend/internal/store/keys.go` (append), `backend/internal/store/keys_test.go` (add a case if the file exists — `ls internal/store/*_test.go`; otherwise create `keys_test.go` with only this test).

- [ ] **Step 1 (test first):** `TestRescueKey` — `RescueKey("u1", "2026-09-22") == "rescue:u1:2026-09-22"` and `RescueTTL == 48*time.Hour` (the same 48 h as `DailyAccumulatedTTL`, so the flag never outlives the counter it guards).
- [ ] **Step 2:** `go test ./internal/store -run RescueKey` → FAIL (undefined).
- [ ] **Step 3:** append to `keys.go`, after `PetReviveKey`:
  ```go
  // RescueTTL bounds the rescue flag. Not in spec §4 — added by the rescue
  // push (notify): 48h matches DailyAccumulatedTTL, the counter it guards.
  const RescueTTL = 48 * time.Hour

  // RescueKey is rescue:{user_id}:{YYYY-MM-DD} — set once (SET NX) when the
  // pre-decay rescue push for that LOCAL date was sent (TTL RescueTTL). The
  // date is the same local date DailyAccumulatedKey uses.
  func RescueKey(userID, localDate string) string {
  	return fmt.Sprintf("rescue:%s:%s", userID, localDate)
  }
  ```
  (Put `RescueTTL` in the existing `const (...)` block instead if you prefer; either is fine — keep it append-only.)
- [ ] **Step 4:** `go test ./internal/store -run RescueKey -count=1` → PASS; `gofmt -l internal/store` empty. Commit: `store: rescue:{user}:{date} key builder and 48h TTL`.

### Task 2: `Rescue.Sweep` — the decision and the send

**Files:** create `backend/internal/notify/rescue.go`, `backend/internal/notify/rescue_test.go`.

- [ ] **Step 1 (fakes):** in `rescue_test.go`:
  ```go
  // fakeRescueRepo adds the candidates query to the shared fakeRepo without
  // touching fakes_test.go (another plan is editing it today).
  type fakeRescueRepo struct{ *fakeRepo }

  func (f *fakeRescueRepo) RescueCandidates(_ context.Context) ([]RescueCandidate, error) {
  	f.log.add("repo.RescueCandidates()")
  	var out []RescueCandidate
  	for id, p := range f.prefs {
  		if len(f.subs[id]) > 0 { // the SQL joins push_subscriptions
  			out = append(out, RescueCandidate{UserID: id, Timezone: p.Timezone})
  		}
  	}
  	sort.Slice(out, func(i, j int) bool { return out[i].UserID < out[j].UserID })
  	return out, nil
  }

  type fakeRescueFlags struct {
  	log     *callLog
  	claimed map[string]bool // key → already set
  	err     error
  }

  func (f *fakeRescueFlags) Claim(_ context.Context, userID, localDate string) (bool, error) {
  	f.log.add("flags.Claim(%s,%s)", userID, localDate)
  	if f.err != nil {
  		return false, f.err
  	}
  	k := store.RescueKey(userID, localDate)
  	if f.claimed[k] {
  		return false, nil
  	}
  	f.claimed[k] = true
  	return true, nil
  }
  ```
  plus a `rescueHarness` built on `newHarness()`'s pieces (`h.repo`, `h.sender`, `h.counter`, `h.log`) with `u1` in `Asia/Ho_Chi_Minh` **with one subscription** (`h.repo.subs["u1"] = []Subscription{{ID: "sub-1", Endpoint: "https://push.example/1", P256dh: "p", Auth: "a"}}`) and `r := NewRescue(&fakeRescueRepo{h.repo}, h.counter, flags, h.sender, func() time.Time { return h.now })`. Helper `utc(day, hour int) time.Time`.
- [ ] **Step 2 (tests first)** — write all of these, run `go test ./internal/notify -run Rescue` → FAIL (undefined):
  - `TestRescueMinutes` — table: 0→30, 1→30, 60→29, 1740→1, 1799→1.
  - `TestRescuePayloadCopy` — `RescuePayload(12)` is exactly `{Title: "Tớ cần cậu thêm 12 phút nữa", Body: "Cây của cậu cần thêm 12 phút hôm nay", URL: "/"}`.
  - `TestRescueSweepSendsOnlyInLocalHour22` — u1 (HCM, 12 min done: `totals["u1|2026-09-22"] = 720`); sweeps at `utc(22,14)`, `utc(22,15)`, `utc(22,16)` → sends only at 15:00 UTC, with `sender.sent == ["https://push.example/1"]` once and `stats.InWindow == 1, Sent == 1` for that tick, `InWindow == 0` for the others; the flag log shows `flags.Claim(u1,2026-09-22)` exactly once; assert the payload the fake sender received (extend `fakeSender` **only if it already records payloads** — it does not, so capture via a tiny `recordingSender` wrapper in `rescue_test.go` that delegates to `fakeSender` and keeps `[]Payload`).
  - `TestRescueSweepUsesTheUsersLocalDateAcrossMidnightUTC` — u2 in `America/New_York`, no time done; sweep at `utc(23, 2)` (= 22:00 EDT on 2026-09-22) → one send, `counter.Total(u2,2026-09-22)` and `flags.Claim(u2,2026-09-22)` in the log (not `2026-09-23`).
  - `TestRescueSweepSendsAtMostOncePerDay` — two sweeps at `utc(22,15)` and `utc(22,15).Add(20*time.Minute)` → one send; second sweep `AlreadySent == 1, Sent == 0`.
  - `TestRescueSweepSkipsAMetDay` — `totals = 1800` → `Met == 1`, no send, **no `flags.Claim` in the log**.
  - `TestRescueSweepSkipsAUserWhenRedisFails` — `h.counter.err = errors.New("redis down")` → `Skipped == 1`, no claim, no send, `err != nil` and `strings.Contains(err.Error(), "u1")`; then `counter.err = nil`, `flags.err = errors.New("redis down")` → `Skipped == 1`, no send.
  - `TestRescueSweepIsSilentWithoutASubscription` — u3 with prefs but `subs["u3"] = nil` → not a candidate (fake mirrors the join), `stats.Candidates == 1` (u1 only); and a u1 whose subs are emptied between candidates and send (`repo.subs["u1"] = nil` after building the harness but with the candidate injected by a custom `RescueRepo` fake returning u1 anyway) → no send, `err == nil`.
  - `TestRescueSweepPrunesGoneSubscriptionsAndContinues` — two subs, first `gone` → `Pruned == 1, Sent == 1`, `repo.DeleteSubscription(sub-1)` logged; a `fail` endpoint → `Failed == 1`, error mentions `u1`, sweep continues to the next user.
- [ ] **Step 3 (implementation):** `rescue.go`:
  ```go
  package notify

  // RescueLocalHour is the local hour (T-2h before midnight) in which the
  // pre-decay rescue push may fire — once per user per local day.
  const RescueLocalHour = 22

  // RescueCandidate is a user with at least one push subscription and the
  // timezone that decides whether their local hour is RescueLocalHour.
  type RescueCandidate struct{ UserID, Timezone string }

  // RescueRepo is the Postgres side of the rescue job. *PgRepo satisfies it;
  // Subscriptions/DeleteSubscription are the ones Repo already has.
  type RescueRepo interface {
  	RescueCandidates(ctx context.Context) ([]RescueCandidate, error)
  	Subscriptions(ctx context.Context, userID string) ([]Subscription, error)
  	DeleteSubscription(ctx context.Context, id string) error
  }

  const rescueCandidatesSQL = `
  SELECT DISTINCT u.id::text, COALESCE(u.timezone, 'UTC')
  FROM users u JOIN push_subscriptions p ON p.user_id = u.id
  ORDER BY u.id::text`

  func (r *PgRepo) RescueCandidates(ctx context.Context) ([]RescueCandidate, error) { … rows.Scan(&c.UserID, &c.Timezone) … }

  // RescueFlags is the once-per-local-day guard: Claim is SET NX EX on
  // store.RescueKey and answers true only for the first caller that day.
  type RescueFlags interface {
  	Claim(ctx context.Context, userID, localDate string) (bool, error)
  }

  type RedisRescueFlags struct{ Client *redis.Client }

  func NewRedisRescueFlags(r *store.Redis) *RedisRescueFlags { return &RedisRescueFlags{Client: r.Client} }

  func (f *RedisRescueFlags) Claim(ctx context.Context, userID, localDate string) (bool, error) {
  	ok, err := f.Client.SetNX(ctx, store.RescueKey(userID, localDate), "1", store.RescueTTL).Result()
  	if err != nil {
  		return false, fmt.Errorf("notify: claiming rescue flag: %w", err)
  	}
  	return ok, nil
  }

  // RescueMinutes is how many whole minutes are still missing from the
  // TargetSeconds goal, never below 1 (the push is only sent when total < TargetSeconds).
  func RescueMinutes(total int64) int {
  	missing := int64(TargetSeconds) - total
  	if missing <= 0 { return 1 }
  	m := int((missing + 59) / 60)
  	if m < 1 { m = 1 }
  	return m
  }

  // RescuePayload is the rescue copy; the plant speaks as "tớ".
  func RescuePayload(minutes int) Payload {
  	return Payload{
  		Title: fmt.Sprintf("Tớ cần cậu thêm %d phút nữa", minutes),
  		Body:  fmt.Sprintf("Cây của cậu cần thêm %d phút hôm nay", minutes),
  		URL:   "/",
  	}
  }

  type RescueStats struct{ Candidates, InWindow, Met, AlreadySent, Skipped, Sent, Pruned, Failed int }

  type Rescue struct {
  	repo    RescueRepo
  	counter StudyCounter
  	flags   RescueFlags
  	sender  Sender
  	now     func() time.Time
  }

  func NewRescue(repo RescueRepo, counter StudyCounter, flags RescueFlags, sender Sender, now func() time.Time) *Rescue

  // Sweep is one hourly pass at now. Per candidate: skip unless the local
  // hour is RescueLocalHour; read the local day's seconds (an error SKIPS —
  // a rescue must never fire twice, so unknown is not "unmet" here, unlike
  // Tick); skip a met day without touching the flag; claim the flag (false →
  // already sent today; error → skip); send RescuePayload to every
  // subscription with Tick's prune rules. Per-user errors are joined; the
  // pass never stops early.
  func (r *Rescue) Sweep(ctx context.Context, now time.Time) (RescueStats, error) { … }
  ```
  The send loop is a copy of `Tick`'s `switch` (gone → prune; forbidden → prune + report once; other error → `Failed`; else `Sent`). Do **not** refactor `Tick` to share it today (B3 is in `service.go`); Notes carry the "fold both loops into one `deliver` helper" follow-up.
- [ ] **Step 4:** `go test ./internal/notify -run 'Rescue' -count=1 -v` → all PASS; `go test ./internal/notify -count=1 -race` (the whole package still green — proves nothing pre-existing moved). Commit: `notify: Rescue.Sweep — one pre-decay rescue push at local 22:00 on an unmet day`.

### Task 3: `RunRescue` (hourly loop) and the integration test

**Files:** `backend/internal/notify/rescue.go` (append), `rescue_test.go` (append), create `rescue_integration_test.go`.

- [ ] **Step 1 (tests first):** `TestNextTopOfHourRescue` — `10:17:30 → 11:00:00`, `10:00:00 → 11:00:00` (strictly after, like `pet.NextTopOfHour`); `TestRunRescueSweepsAndStopsWhenTheContextIsCancelled` — the `worker_test.go` pattern: inject `now` one second before the top of the hour, run `RunRescue` in a goroutine, wait for `sender.Sent()` to reach 1 (poll ≤ 3 s), cancel, assert return. Read `worker_test.go` first and mirror its waiting/`-race` discipline (the fake sender's `Sent()` exists for exactly this).
  Integration (`rescue_integration_test.go`, same gate and setup as `integration_test.go`): insert two users, subscribe only the first (`repo.SaveSubscription`), assert `RescueCandidates` returns only the first with its timezone; `RedisRescueFlags.Claim` → `true` then `false` for the same `(user, date)`, `true` for the next date; `TTL` of the key is within `(47h, 48h]`; cleanup deletes the users and the keys.
- [ ] **Step 2:** append to `rescue.go`:
  ```go
  // NextTopOfHour is the next :00 strictly after now — pet.RunHourly's rule,
  // copied rather than imported (notify never imports pet).
  func NextTopOfHour(now time.Time) time.Time { return now.Truncate(time.Hour).Add(time.Hour) }

  // RunRescue is the in-process hourly rescue job, pet.RunHourly's shape:
  // blocks until ctx is cancelled, sweeping at every :00 UTC (the tick at
  // hh:00 UTC catches every zone whose local hour is RescueLocalHour —
  // half-hour zones included). One per deployment, like RunWorker: the SET NX
  // flag makes a second sweeper harmless but not useful.
  func RunRescue(ctx context.Context, r *Rescue) {
  	for {
  		now := r.now()
  		timer := time.NewTimer(NextTopOfHour(now).Sub(now))
  		select {
  		case <-ctx.Done():
  			timer.Stop()
  			return
  		case <-timer.C:
  		}
  		at := r.now()
  		stats, err := r.Sweep(ctx, at)
  		if err != nil {
  			log.Printf("notify: rescue sweep at %s: %+v, errors: %v", at.UTC().Format(time.RFC3339), stats, err)
  			continue
  		}
  		if stats.InWindow > 0 {
  			log.Printf("notify: rescue sweep at %s: %+v", at.UTC().Format(time.RFC3339), stats)
  		}
  	}
  }
  ```
- [ ] **Step 3:** `go test ./internal/notify -run 'Rescue|NextTopOfHour' -count=1 -race -v` → PASS; then the integration run from `## Verification` (the CI `backend-integration` job counts `TestIntegration*` functions and requires a `--- PASS` for each — a skip in CI is a failure). Commit: `notify: RunRescue hourly loop; integration test for candidates and the SET NX flag`.

### Task 4: wiring in `main.go` and CODEMAP — after `origin/main` contains `harness/daily-2026-09-26`

**Files:** `backend/cmd/api/main.go`, `harness/CODEMAP.md`.

**Gate (repeat of the header):** `git fetch origin && git merge-base --is-ancestor $(git rev-parse origin/harness/daily-2026-09-26) origin/main && echo ok` must print `ok` **and** the branch must have been merged onto `origin/main` before you `git merge origin/main` into this branch; if the daily branch does not exist or is not an ancestor, stop after Task 3, push, and report "Task 4 blocked: 2026-09-26 daily PR not on origin/main" in the Execution summary.

- [ ] **Step 1:** `git fetch origin main && git merge origin/main --no-edit` (this branch syncs only from `origin/main`); `go build ./... && go test ./internal/notify -count=1` still green.
- [ ] **Step 2:** in `main.go`, inside the existing `if pushSender != nil { … }` block right after `go notify.RunWorker(ctx, notifySvc, notify.PollInterval)`, add:
  ```go
  		// Pre-decay rescue push: one Web Push at local 22:00 on an unmet day
  		// (notify.Rescue; once per user per local day via rescue:{user}:{date}).
  		rescue := notify.NewRescue(
  			notify.NewPgRepo(pg.Pool),
  			studyCounter, // the same read-only view of daily:accumulated
  			notify.NewRedisRescueFlags(rdb),
  			pushSender,
  			time.Now,
  		)
  		go notify.RunRescue(ctx, rescue)
  ```
  Add lines only; do not hoist or rename `notify.NewPgRepo(pg.Pool)` in `NewService` (keeps the diff to one insertion so the daily merge cannot conflict on lines other branches moved). If, after the merge, `main.go` already has a `notifyRepo` variable, use it instead of a second `NewPgRepo` and say so in the Execution summary.
- [ ] **Step 3:** `cd backend && go build ./... && go vet ./... && gofmt -l cmd/api` empty; `go run ./cmd/api` is not required (no `.env` in CI) — the `cmd/api` tests must still pass: `go test ./cmd/api -count=1`.
- [ ] **Step 4:** CODEMAP — `notify` paragraph: add the rescue job (hourly `:00` UTC, `RescueLocalHour` 22, the order counter → met → `SET NX` flag → send, Redis error skips, no-subscription silent, the exact copy, health dropped and why, the DST-skips-hour-22 note, one per deployment); `store` paragraph: `RescueKey`/`RescueTTL` join `PetReviveKey` as keys "not in spec §4". Commit: `api: start notify.RunRescue beside RunWorker; CODEMAP`.
- [ ] **Step 5:** Execution summary *Follow-ups* (for the reviewer to file as inbox ideas, since `project-base/` is on unmerged branches today): (a) backend spec §4 gains `rescue:{user_id}:{date}` (String, 48 h) and §8 a "pre-decay rescue push at local 22:00" sentence; 1st-thinking §4 the same key; (b) restore `· sức khỏe {health}%` once `pet` exposes a read-only `Health(ctx, userID)` (no INSERT) that `main.go` can adapt into a `notify.PetHealth` interface; (c) fold `Tick`'s and `Sweep`'s send loops into one `deliver` helper after B3 lands; (d) the learned reminder time — the idea's other half — is its own plan; (e) skip the rescue while a shield/revive is pending once `pet` exposes that state (idea text; `pet-streak-shield` is on an unmerged branch today).

## Verification
```
cd backend && go build ./... && gofmt -l . && go vet ./... && go test -timeout 120s ./... -count=1 -race
go test -timeout 60s ./internal/notify -run 'Rescue|NextTopOfHour' -count=1 -race -v
COMPOSE_PROJECT_NAME=<slug> POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose up -d --wait \
  && TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable TEST_REDIS_URL=redis://localhost:6380/0 \
     go test -timeout 300s ./... -run Integration -p 1 -count=1 -v | grep -c -- '--- PASS' \
  && COMPOSE_PROJECT_NAME=<slug> make down
git diff --stat origin/main -- internal/notify           # only rescue.go, rescue_test.go, rescue_integration_test.go
git diff origin/main -- internal/store/keys.go | grep -c '^-[^-]'   # 0: append-only
grep -n 'RescueKey\|RescueTTL' internal/store/keys.go internal/notify/rescue.go
grep -n 'Cây của cậu cần thêm' internal/notify/rescue.go internal/notify/rescue_test.go
grep -n 'RunRescue' cmd/api/main.go ../harness/CODEMAP.md   # Task 4 only — absent if the gate held
git push -u origin harness/2026-09-27-low-adaptive-reminder-timing-and-pre-decay-rescue-push
```
(Use the credentials/ports your `backend/.env` for the scratch project actually sets; the compose defaults are in `backend/docker-compose.yml` — check with `grep -n POSTGRES_ backend/docker-compose.yml`.)

## Notes
- **Scoped out (the idea's other half, a later plan):** `users.reminder_mode`, `users.learned_notification_time`, `daily_progress.first_activity_at`, the nightly median job, `reminder_mode` on `POST /settings/notifications`, the "Auto (learned …)" settings copy. No migration today — two unmerged `0004_*` migrations (`pet_shields`, `rls`) already collide.
- **Health dropped from the copy:** notify has no interface to pet; pet's only state reader `Service.Ensure` INSERTs the row; adding a `notify.PetHealth` interface plus a `main.go` adapter would widen the gated `main.go` diff. Follow-up (Task 4 Step 5 b).
- **Dependency gate — the 2026-09-26 daily code PR** must contain these `done` branches (all unmerged at planning time; seven of them edit `main.go`): `harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut`, `harness/2026-09-25-medium-settings-screen-wires-web-push-reminders-and-google-calendar` (creates the first `push_subscriptions` rows — without it no user is a rescue candidate), `harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a`, `harness/2026-09-25-medium-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-`, `harness/2026-09-25-medium-auth-reports-postgres-and-redis-failures-as-401-and-logs-not`, `harness/2026-09-25-medium-cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r`, `harness/2026-09-26-medium-growth-moment-after-every-task-health-gain-streak-and-target`, `harness/2026-09-26-medium-name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o`, `harness/2026-09-25-medium-parseorigins-accepts-frontend-origin-entries-no-browser-send`, `harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days`, `harness/2026-09-26-medium-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-`, `harness/2026-09-25-high-ship-on-merge-a-deploy-workflow-that-builds-the-pwa-and-uplo`, `harness/2026-09-25-high-task-timer-keeps-counting-through-reloads-and-background-tab`, `harness/2026-09-25-medium-the-documented-set-a-env-export-also-exports-test-database-u`, `harness/2026-09-26-high-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a`, `harness/2026-09-26-high-a-session-a-learner-wants-to-finish-level-true-content-do-to`, `harness/2026-09-26-medium-caddyfile-serves-index-html-with-a-one-year-immutable-cache-`, `harness/2026-09-26-high-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-`, `harness/2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase`, `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n`, `harness/2026-09-26-high-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve`. Tasks 1–3 are safe to execute before the gate (their files are untouched by all of them); only Task 4 waits.
- **Coordination with B3 (2026-09-27 `notify` bug plan):** B3 adds subscription validation, a per-user cap and a failure-prune counter in `handler.go`, `service.go`, `repo.go`, `keys.go`. This plan never edits those notify files; both append to `store/keys.go` — an append-only conflict the daily merge resolves by keeping both hunks (if the merge does conflict there, keep both key builders and re-run `go build`).
- **Why the flag is claimed after the met check, not before the counter read:** a met day must leave no trace; a Redis failure must not consume the day's single rescue.
- **Why not the ZSET:** `queue:webpush:delay` carries one score per user — the *scheduled* reminder — and `Tick` sends `DefaultPayload`; enqueueing the rescue there would overwrite the user's reminder slot and send the wrong copy. The rescue goes straight through `Sender`, which is the existing send path.
- **Runtime prerequisite (not a code dependency):** VAPID keys set on the API host (`deploy/README.md`) — like `RunWorker`, `RunRescue` starts only inside the `pushSender != nil` branch.
