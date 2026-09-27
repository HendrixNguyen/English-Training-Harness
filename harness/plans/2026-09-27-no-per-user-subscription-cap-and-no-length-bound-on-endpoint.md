---
idea: harness/ideas/_inbox/no-per-user-subscription-cap-and-no-length-bound-on-endpoint.md
status: approved
priority: medium
merged: false
---
# Push subscriptions: validate `p256dh`/`auth`, cap 10 per user (evict oldest), prune after 3 consecutive send failures — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Bug team — ticket **B3** of 2026-09-27. **Estimate:** 3.5 h. **Branch:** `harness/2026-09-27-medium-no-per-user-subscription-cap-and-no-length-bound-on-endpoint`.

**Idea:** `harness/ideas/_inbox/no-per-user-subscription-cap-and-no-length-bound-on-endpoint.md` — backend only; **no design doc**. Its `## Evaluation` (2026-09-27) records what is still open on `origin/main` and the decisions below.

**Goal:** One authenticated account can no longer store unbounded or undeliverable push rows: `POST /api/v1/settings/notifications` refuses key material that cannot encrypt (`400 invalid_request`, nothing written), a user holds at most 10 subscriptions (the oldest is evicted), and `Tick` deletes a subscription after 3 consecutive non-gone send failures instead of retrying it every day forever.

**Already done on `main` (regression test only, no code):** `endpoint` length — `notify.MaxEndpointLength` (2048) is enforced by `ValidateEndpoint`, which `UpdateSettings` wraps in `ErrInvalidRequest`; the JSON body is bounded globally by `middleware.BodyLimit` (`cmd/api/main.go:170`). Task 1 adds the missing handler-level test.

**Decisions (from the idea's Evaluation):**
- Keys: `p256dh` must base64url-decode — **padded and unpadded both accepted**, standard base64 (`+`/`/`) refused — to exactly 65 bytes, first byte `0x04`, and be a valid P-256 point (`ecdh.P256().NewPublicKey`); `auth` must decode to exactly 16 bytes. The PWA sends `PushSubscription.toJSON().keys`, the browser's unpadded base64url, so real subscriptions pass.
- Cap: **`MaxSubscriptionsPerUser = 10`, evict the oldest by `created_at` (tie: `id`)**, not reject. There is no unsubscribe endpoint, so under a reject-400 cap a learner with dead browser profiles could never subscribe a new device; eviction keeps the device they are on working and bounds `Tick`'s fan-out to 10 requests per user per day just the same. Re-saving an endpoint the user already has is still a no-op (existing dedupe) and evicts nothing.
- Prune: **`MaxConsecutiveFailures = 3`**. Each subscription is attempted once per day, so three consecutive failures is three days broken; a success resets the count. Counter in Redis at `push:fail:{subscription_id}`, `INCR` + `EXPIRE` 7 days on failure, `DEL` on success and on prune. **No migration** (two unmerged branches already add `0004_*`); the key builder and TTL live in `store/keys.go` in §4 style like `PetReviveKey`.
- Wiring: `cmd/api/main.go` is **off-limits** this run (unmerged branches edit it), so the counter is two methods on the existing `Queue` interface — `RedisQueue` already holds the Redis client — and `NewService`'s signature is unchanged. Nothing outside `backend/internal/notify/`, `backend/internal/store/keys.go` + `keys_test.go`, and the CODEMAP `notify`/`store` bullets may change.

## Global Constraints
- Work in `.worktrees/<slug>`; Go from `backend/`; `rg`/`timeout` are not installed (use `grep -n`, `go test -timeout`); integration tests via `COMPOSE_PROJECT_NAME=<slug> make up` … `make down`.
- **Do not touch** `backend/cmd/api/*`, `backend/internal/store/migrations/*`, `store/migrations_test.go`, `store/integration_test.go`, `backend/internal/middleware/*`, or anything under `project-base/`. If a step seems to need one of them, stop and report instead.
- Validation order in `UpdateSettings` stays "everything validated before the first write": clock → timezone → endpoint → keys → `UpdatePreferences`. A rejected request leaves the fake repo/queue call log empty (the existing handler tests assert `len(h.log.calls) == 0`).
- `Tick`'s error classification order stays gone → forbidden → other; only "other" is counted. A counter read/write error never prunes (treat as count 0, append the error) — Redis being down must not delete subscriptions.
- Existing test fixtures with placeholder keys (`handler_test.go` `spec64Body` and the hostile-endpoint body, `service_test.go` `validSub`) must move to real generated key material; `integration_test.go:75`'s repo-level fixture may stay (the repo does not validate).
- `gofmt -l internal/notify internal/store` empty; `go vet ./...` clean; `go test ./... -race` green with no `TEST_*`/`DATABASE_URL`/`REDIS_URL` exported.

## Review Focus
1. Key validation: unpadded and padded base64url of a real 65-byte `0x04…` P-256 point pass; 64-byte, 66-byte, `0x02`-prefixed (compressed), off-curve (e.g. `0x04` + 64 zero bytes), standard-base64 and non-base64 inputs → `ErrInvalidSubscriptionKeys`; `auth` of 15/17 bytes or non-base64 → the same. `UpdateSettings` returns `ErrInvalidRequest` (wrapping it) with **no** repo/queue call; the handler answers exactly `400 {"error":"invalid_request"}`.
2. Cap: the 11th distinct endpoint for one user leaves exactly 10 rows, the oldest (`ep1`) gone and `ep11` present — asserted on the fake (service test) **and** on real Postgres (integration test). Both statements run in one `pgx.BeginFunc` transaction. Re-saving `ep11` leaves 10 rows and evicts nothing.
3. Prune: two failures keep the row (`Failed` counted, count 1 then 2); the third deletes it (`Pruned++`, `repo.DeleteSubscription`, counter cleared); a success between failures resets to 0; a `RecordFailure` error → row kept, error joined. A gone (404/410) or forbidden row is pruned on the first tick exactly as before — the counter is not consulted for them.
4. `NewService(repo, queue, sender, counter, now)` is unchanged; `cmd/api/main.go` compiles untouched (`go build ./...`).
5. `store.PushFailKey("s1") == "push:fail:s1"`, `store.PushFailTTL == 7 * 24 * time.Hour`, both in `keys_test.go`'s tables; the integration test sees a positive `TTL` on the key after `RecordFailure`.

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/notify/subscription_keys.go` (new) + `subscription_keys_test.go` (new) | `ErrInvalidSubscriptionKeys`, `ValidateSubscriptionKeys(p256dh, auth string) error`, `decodeBase64URL`; the `validKeys(t)` test helper |
| `backend/internal/notify/service.go` + `service_test.go` | key check in `UpdateSettings`; prune-after-N in `Tick`; `MaxConsecutiveFailures` |
| `backend/internal/notify/handler_test.go` | fixtures on real keys; malformed-key and over-length-endpoint rows |
| `backend/internal/notify/repo.go` + `integration_test.go` | `MaxSubscriptionsPerUser`, trim statement inside a transaction; cap assertions |
| `backend/internal/notify/queue.go` + `fakes_test.go` | `Queue.RecordFailure` / `Queue.ClearFailures`, `RedisQueue` impl, `fakeQueue.failures`; `fakeRepo.SaveSubscription` mirrors the cap |
| `backend/internal/store/keys.go` + `keys_test.go` | `PushFailKey`, `PushFailTTL` |
| `harness/CODEMAP.md` | `notify` bullet (keys, cap, prune) and `store` bullet (`PushFailKey` beside `PetReviveKey`) |

## Tasks

### Task 1: Validate `p256dh` / `auth` before anything is written

**Files:** `backend/internal/notify/subscription_keys.go` (new), `subscription_keys_test.go` (new), `service.go`, `service_test.go`, `handler_test.go`.

- [ ] **Step 1 (tests first):** `subscription_keys_test.go` — `validKeys(t *testing.T) (p256dh, auth string)` generating a real pair the same way `browserSubscription` in `push_test.go` does (`ecdh.P256().GenerateKey` → `PublicKey().Bytes()`, 16 random bytes; `base64.RawURLEncoding`); `TestValidateSubscriptionKeysAcceptsBrowserKeys` (unpadded, and the same strings re-encoded with `base64.URLEncoding` i.e. padded); `TestValidateSubscriptionKeysRefusesBadMaterial` table per Review Focus 1, every case `errors.Is(err, ErrInvalidSubscriptionKeys)`.
- [ ] **Step 2 (tests first):** `service_test.go` — replace `var validSub` with `func validSubscription(t) *Subscription` using `validKeys`; update its callers; add to `TestUpdateSettingsRejectsBadInputBeforeWriting` two cases (`p256dh` = `"BNc5T"`, `auth` = 15 bytes) expecting `ErrInvalidRequest` and an empty call log. `handler_test.go` — turn `spec64Body` into `func spec64Body(t) string` built with `validKeys` (keep the FCM endpoint and the §6.4 comment); give the hostile-endpoints test real keys too; add `TestSettingsHandlerRejectsMalformedKeysAndOverlongEndpointsWith400` with rows: `p256dh` `"BNc5T..."`, `p256dh` standard-base64 of a valid point, `auth` 17 bytes, and an endpoint of `"https://fcm.googleapis.com/fcm/send/" + strings.Repeat("a", MaxEndpointLength)` — each `400 {"error":"invalid_request"}` and `len(h.log.calls) == 0`.
- [ ] **Step 3:** `subscription_keys.go` — `ErrInvalidSubscriptionKeys`; `decodeBase64URL(s)` = `base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))` (refuse empty); `ValidateSubscriptionKeys`: p256dh → 65 bytes, `b[0] == 0x04`, `ecdh.P256().NewPublicKey(b)` succeeds; auth → 16 bytes; every failure wraps `ErrInvalidSubscriptionKeys` with which field and why (never echo the value). `service.go` `UpdateSettings`: after `ValidateEndpoint`, `if err := ValidateSubscriptionKeys(sub.P256dh, sub.Auth); err != nil { return SettingsResult{}, fmt.Errorf("%w: %w", ErrInvalidRequest, err) }`.
- [ ] **Step 4:** `go test ./internal/notify -run 'Keys|UpdateSettings|SettingsHandler' -count=1 -v`. Commit: `notify: refuse push subscriptions whose p256dh/auth cannot encrypt`.

### Task 2: Cap subscriptions at 10 per user, evicting the oldest

**Files:** `backend/internal/notify/repo.go`, `fakes_test.go`, `service_test.go`, `integration_test.go`.

- [ ] **Step 1 (tests first):** `service_test.go` — `TestUpdateSettingsKeepsAtMostTenSubscriptionsPerUser`: save `ep1…ep11` (distinct endpoints, valid keys) through `UpdateSettings`; assert `len(h.repo.subs["u1"]) == MaxSubscriptionsPerUser`, `subs[0].Endpoint == ".../ep2"`, last is `ep11`; re-save `ep11` → still 10, unchanged order. `integration_test.go` — after the existing dedupe block, insert 11 distinct endpoints for `a` (row-by-row so `created_at` orders them; if two share a timestamp the `id` tie-break makes the outcome deterministic only in count — assert count 10 and that `ep11` is present, and that `ep1` is absent), then re-save `ep11` and assert 10 again.
- [ ] **Step 2:** `fakes_test.go` `fakeRepo.SaveSubscription`: after appending, `if len(list) > MaxSubscriptionsPerUser { list = list[len(list)-MaxSubscriptionsPerUser:] }` (oldest first in the slice already).
- [ ] **Step 3:** `repo.go` — `const MaxSubscriptionsPerUser = 10` with a comment stating the eviction decision; `trimSubscriptionsSQL = DELETE FROM push_subscriptions WHERE user_id = $1 AND id NOT IN (SELECT id FROM push_subscriptions WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2)`; `PgRepo.SaveSubscription` runs `saveSubscriptionSQL` then `trimSubscriptionsSQL` inside `pgx.BeginFunc(ctx, r.Pool, func(tx pgx.Tx) error {…})` (pgx v5 `tx.go:402`). Update the `Repo.SaveSubscription` doc comment: "…and keeps at most `MaxSubscriptionsPerUser` rows for userID, evicting the oldest by `created_at`."
- [ ] **Step 4:** `go test ./internal/notify -run 'AtMostTen|Integration' -count=1` (integration skips without `TEST_*`; run it for real once via the Verification block). Commit: `notify: cap push subscriptions at 10 per user, evicting the oldest`.

### Task 3: Prune a subscription after 3 consecutive send failures (Redis counter, no migration)

**Files:** `backend/internal/store/keys.go`, `keys_test.go`, `backend/internal/notify/queue.go`, `fakes_test.go`, `service.go`, `service_test.go`, `integration_test.go`.

- [ ] **Step 1 (tests first):** `store/keys_test.go` — add `{"pushfail", PushFailKey("3f0d…0001"), "push:fail:3f0d…0001"}` to `TestKeyBuilders` and `{"PushFailTTL", PushFailTTL, 7 * 24 * time.Hour}` to `TestTTLs`. `notify/service_test.go` — `TestTickPrunesASubscriptionAfterThreeConsecutiveFailures`: `dueHarness`, `h.sender.fail[ep1] = true`; run `Tick` on three successive days (advance `now` by 24 h and re-slot — or call `Tick` with the queue score reset each time), asserting after ticks 1 and 2 `stats.Failed == 1`, `stats.Pruned == 0`, `len(h.repo.subs["u1"]) == 2`, `h.queue.failures["s1"] == 1` then `2`; after tick 3 `stats.Pruned == 1`, `stats.Failed == 0`, only `s2` remains, `h.queue.failures["s1"]` absent. `TestTickResetsTheFailureCountOnSuccess`: two failures, then `delete(h.sender.fail, ep1)`, one success → `failures["s1"]` absent; then two more failures still keep the row. `TestTickKeepsTheRowWhenTheCounterIsUnavailable`: `h.queue.failErr = errors.New("redis down")` → `Failed == 1`, row kept, error joined. Keep `TestTickReportsSendFailuresButContinues` as is (one failure = kept) — it now documents the N-1 side.
- [ ] **Step 2:** `store/keys.go` — `PushFailTTL = 7 * 24 * time.Hour` in the TTL const block (comment: not in spec §4; one attempt per day, so the window must outlive `MaxConsecutiveFailures` days with slack; an expired counter simply restarts at 1) and `func PushFailKey(subscriptionID string) string { return fmt.Sprintf("push:fail:%s", subscriptionID) }` (comment: consecutive non-gone send failures for one push_subscriptions row, String, INCR+EXPIRE; added by notify hardening, documented in CODEMAP).
- [ ] **Step 3:** `notify/queue.go` — extend `Queue` (doc: "notify's Redis side: the §4 ZSET plus the per-subscription failure counter, so `NewService` needs no extra wiring") with `RecordFailure(ctx, subscriptionID string) (int64, error)` (`TxPipelined`: `INCR` + `EXPIRE store.PushFailTTL`; return the `INCR` value) and `ClearFailures(ctx, subscriptionID string) error` (`DEL`). `fakes_test.go` `fakeQueue`: `failures map[string]int64`, `failErr error`; log both calls. `service.go` — `const MaxConsecutiveFailures = 3`; in `Tick`'s `case err != nil:` branch: `n, cerr := s.queue.RecordFailure(ctx, sub.ID)`; if `cerr != nil` → `stats.Failed++`, join both errors, continue; if `n >= MaxConsecutiveFailures` → `stats.Pruned++`, `errs = append(errs, fmt.Errorf("user %s: subscription %s pruned after %d consecutive failures: %w", …))`, `appendIf(errs, s.repo.DeleteSubscription(ctx, sub.ID))`, `appendIf(errs, s.queue.ClearFailures(ctx, sub.ID))`; else `stats.Failed++` and the existing error line. In the `default:` (sent) branch: `errs = appendIf(errs, s.queue.ClearFailures(ctx, sub.ID))`. Update `Tick`'s doc comment and `TickStats.Pruned`'s meaning ("gone, forbidden, or 3× failed").
- [ ] **Step 4:** `integration_test.go` — after the ZSET block: `q.RecordFailure(ctx, "notify-integration-sub")` ×3 returns 1, 2, 3; `rdb.Client.TTL(ctx, store.PushFailKey(...))` > 0 and ≤ `store.PushFailTTL`; `ClearFailures` then `RecordFailure` returns 1; cleanup `DEL`s the key.
- [ ] **Step 5:** `go test ./internal/store ./internal/notify -run 'Key|TTL|Tick|Integration' -count=1 -race`. Commit: `notify: prune a subscription after 3 consecutive send failures (push:fail counter)`.

### Task 4: CODEMAP and the full gate

**Files:** `harness/CODEMAP.md` only.

- [ ] **Step 1:** `notify` bullet — after the SSRF sentence add: keys are validated at subscribe (`ValidateSubscriptionKeys`: base64url padded/unpadded → 65-byte `0x04` P-256 point on the curve, 16-byte auth; → 400, nothing written); `MaxSubscriptionsPerUser` 10 with oldest-by-`created_at` eviction inside one transaction (why evict, not reject); `Tick` prunes after `MaxConsecutiveFailures` (3) non-gone failures counted at `store.PushFailKey` (`INCR`+`EXPIRE` `PushFailTTL` 7 d, `DEL` on success; a counter error never prunes), carried on the `Queue` interface so `main.go` did not change. `store` bullet — extend the `PetReviveKey` sentence: `PushFailKey`/`PushFailTTL` (`push:fail:{subscription_id}`, 7 d) is the other key not in spec §4.
- [ ] **Step 2:** Run the Verification block in full (unit + integration + push). Commit: `docs: CODEMAP — push key validation, 10-subscription cap, prune after 3 failures`.

## Notes
- **Spec follow-up (not in this plan — `project-base/` is unsafe today):** backend spec §4 Redis key table gains `push:fail:{subscription_id}` (String, 7 d), and §6.4 gains "p256dh/auth are validated (base64url, 65-byte P-256 point / 16 bytes); at most 10 subscriptions per user, oldest evicted". The executor lists this under *Follow-ups* in the Execution summary for the reviewer to file.
- Re-saving an endpoint the user already has keeps its original `created_at`, so a device that re-subscribes every settings visit still ages against newer devices. With a cap of 10 this is theoretical; if it ever matters, bump `created_at` on the dedupe path (one `UPDATE` in the same transaction).
- A stored row with bad keys that predates this fix now falls to the prune path (three failed days) rather than living forever — no backfill needed.

## Verification
```
cd backend && go build ./... && gofmt -l . && go vet ./... && go test -timeout 120s ./... -count=1 -race
go test -timeout 60s ./internal/notify -run 'Keys|SettingsHandler|UpdateSettings|AtMostTen|Tick' -count=1 -v
go test -timeout 60s ./internal/store -run 'KeyBuilders|TTLs' -count=1 -v
COMPOSE_PROJECT_NAME=<slug> make up && go test -timeout 300s ./internal/notify -run Integration -p 1 -count=1 -v && make down
git diff --stat origin/main -- cmd/api internal/store/migrations internal/middleware ../project-base   # must be empty
grep -n 'MaxSubscriptionsPerUser\|MaxConsecutiveFailures\|ValidateSubscriptionKeys' internal/notify/*.go ../harness/CODEMAP.md
grep -n 'PushFailKey\|PushFailTTL' internal/store/keys.go internal/store/keys_test.go internal/notify/queue.go ../harness/CODEMAP.md
git push -u origin harness/2026-09-27-medium-no-per-user-subscription-cap-and-no-length-bound-on-endpoint
```
