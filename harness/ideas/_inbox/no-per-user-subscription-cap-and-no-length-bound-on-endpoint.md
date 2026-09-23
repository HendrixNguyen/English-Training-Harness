---
type: bug
status: selected
source: reviewer
run: _inbox
priority: medium
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
