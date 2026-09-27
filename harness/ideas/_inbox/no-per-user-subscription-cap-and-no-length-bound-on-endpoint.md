---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-27-no-per-user-subscription-cap-and-no-length-bound-on-endpoint.md
---
# No per-user subscription cap and no length bound on endpoint p256dh auth

## Why
`SaveSubscription` deduplicates on `endpoint`, so the same browser is stored once — but a
*different* endpoint string is always a new row, and nothing caps how many rows one user may
accumulate. `endpoint`, `p256dh` and `auth` are `TEXT` columns with no length limit and no
format check, and the handler puts no bound on the JSON body either.

Two consequences. First, storage: one authenticated account can write unbounded rows of
unbounded size into `push_subscriptions`. Second, and worse, `Tick` fans out over
`repo.Subscriptions(userID)` and sends to *every* row, so N stored rows means N outbound HTTP
requests per day for that one user — an amplification multiplier on top of the SSRF finding,
and a way for a normal user with several stale browser profiles to get several duplicate
notifications a day.

`p256dh` and `auth` also go unvalidated: garbage key material makes `webpush-go` fail at
encryption time with an error that is *not* `ErrSubscriptionGone`, so `Tick` counts it as
`Failed` and never prunes it — the bad row is retried every day forever.

## Expected output
- `p256dh` must decode as base64url to a 65-byte uncompressed P-256 point and `auth` to a
  16-byte secret; otherwise `400 invalid_request` and nothing is written. This also turns the
  permanently-failing-row case into a request that never gets stored.
- `endpoint` has a maximum length (2048 is the usual URL cap) enforced at the handler.
- A per-user subscription cap (5–10 devices is generous) — on exceeding it, either reject with
  400 or evict the oldest row by `created_at`.
- `Tick` treats a persistently failing subscription as prunable after a bounded number of
  consecutive non-gone failures, rather than retrying it daily forever.
- Tests: over-length endpoint → 400; malformed `p256dh`/`auth` → 400; the (cap+1)-th distinct
  endpoint for one user → 400 or eviction, with the row count asserted.

## Evidence
- Plan under review: `harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md`.
- `backend/internal/notify/handler.go:22-26` — `binding:"required"` only; no `max`, no format.
- `backend/internal/notify/service.go:69-71` — non-empty is the entire validation for all three fields.
- `backend/internal/notify/repo.go:55-61` — `saveSubscriptionSQL` dedupes on `(endpoint, user_id)` but has no per-user count check.
- `backend/internal/store/migrations/0001_init.up.sql:24-31` — `endpoint TEXT NOT NULL, p256dh TEXT NOT NULL, auth TEXT NOT NULL`, no lengths, no unique index.
- `backend/internal/notify/service.go:143-164` — `Tick` iterates every subscription of the user; `service.go:158-160` classifies a non-gone error as `Failed` and leaves the row in place.
- `backend/internal/notify/service_test.go:194-204` (`TestTickReportsSendFailuresButContinues`) pins exactly this behaviour: "a transient failure must not delete the subscription" — correct for a transient failure, but there is no bound for a permanent one.
- Sibling finding: `harness/ideas/_inbox/push-subscription-endpoint-is-an-unvalidated-user-supplied-u.md`.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — medium.** Notify is `done` but unmerged; plan after it lands. Unbounded rows per user, unvalidated key material that fails encryption forever, and N-fold outbound amplification are all real. Fix in `notify` only (validation + cap + prune-after-N-failures). Pairs naturally with `tick-sends-serially…` and `due-plus-re-slot…` as one notify-hardening plan if the owner wants fewer merges.

_Evaluator, 2026-09-27 — daily decide (bug queue, planned today as **B3**)._

**Still real on `origin/main`; keep `selected` / medium.** Checked on this checkout (= `origin/main`), with `go test ./internal/notify ./internal/store` green as the baseline:
- **Keys unvalidated** — `backend/internal/notify/service.go:69-76`: non-empty plus `ValidateEndpoint`; `p256dh`/`auth` are handed straight to `webpush.Keys` in `push.go:79`. Garbage still stores, and the tests themselves pin the gap: `handler_test.go:39` (`spec64Body`) and `service_test.go:11` (`validSub`) use `"BNc5T..."`/`"aX8v..."` placeholders that no decoder accepts — those fixtures must move to real key material with the fix.
- **No per-user cap** — `repo.go:55-61`: the CTE dedupes on `(endpoint, user_id)` and nothing counts rows; `Tick` (`service.go:158`) still sends to every row.
- **Permanent failures never pruned** — `service.go:170-173`: any non-gone, non-forbidden error is `Failed++` and the row stays; `TestTickReportsSendFailuresButContinues` (`service_test.go:194`) pins "kept", correctly for a transient error, with no bound for a permanent one.
- **Already fixed by the sibling SSRF plan** — `endpoint` length: `endpoint.go:26` `MaxEndpointLength = 2048` is enforced in `ValidateEndpoint`, which `UpdateSettings` wraps as `ErrInvalidRequest` → 400; `endpoint_test.go:50` covers the unit, but no handler-level test does. The JSON body is already bounded globally by `middleware.BodyLimit` (`cmd/api/main.go:170`). So this part is a regression test only, no new code.

**Decisions.** (1) `p256dh` must base64url-decode (padded or unpadded — the PWA sends `PushSubscription.toJSON().keys`, the browser's unpadded form) to 65 bytes starting `0x04` and lie on P-256 (`ecdh.P256().NewPublicKey`); `auth` to exactly 16 bytes; else `400 invalid_request`, nothing written. (2) Cap **10 per user, evict the oldest by `created_at`** rather than reject: there is no unsubscribe endpoint, so a learner whose old browser profiles are gone could never get back under a reject-400 cap, while eviction keeps the device they are on working and bounds the fan-out just the same. (3) Prune after **3 consecutive** non-gone failures (one attempt per day → three days broken), counted in Redis at `push:fail:{subscription_id}` with a 7-day TTL, `INCR`+`EXPIRE` on failure, `DEL` on success — **no migration** today (two unmerged branches already add `0004_*`); the key builder and TTL go in `store/keys.go` in §4 style like `PetReviveKey`. (4) `cmd/api/main.go` is off-limits this run, so the counter rides on the existing `Queue` interface (`RedisQueue` already holds the Redis client) instead of a new constructor argument; nothing in `NewService`'s signature changes. Files stay inside `backend/internal/notify/`, `backend/internal/store/keys.go` (+ `keys_test.go`) and the CODEMAP `notify`/`store` bullets; the §4/§6.4 spec sentences are a follow-up in the plan's Notes. `tick-sends-serially…` and `due-plus-re-slot…` stay separate.
