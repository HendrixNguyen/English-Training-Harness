---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-24-every-google-403-becomes-409-reauth-required-so-a-quota-erro.md
---
# Every Google 403 becomes 409 reauth_required, so a quota error forces a pointless re-consent

## Why
`doJSON` maps **every** 401 *and* 403 from Calendar and Tasks to `ErrReauthRequired`
(`backend/internal/google/client.go:74-75`), and `SyncHandler` turns that into
`409 {"error":"reauth_required"}` (`handler.go:34-37`). CODEMAP now states this as the contract
("`invalid_grant`/401/403 → 409 `reauth_required`").

But 403 is not only an authorization answer from Google's APIs. Calendar v3 and Tasks v1 both return
**403** for `rateLimitExceeded`, `userRateLimitExceeded`, `dailyLimitExceeded` and `quotaExceeded` —
throttling, not a revoked grant. A learner who hits a per-minute quota (trivially reachable: nothing
rate-limits this route, and one sync issues up to 30 Tasks calls in a burst — see the plan's own
*Notes → Rate/abuse*) is told to re-authenticate. Per the contract the frontend then sends them
through the whole Google consent screen, after which the very next sync fails identically, because
the refresh token was never the problem. That is a dead-end loop and it teaches users to distrust
the consent prompt.

The OAuth path gets this right — `oauth.go:70` only claims reauth when the body actually says
`invalid_grant` (or the status is 401). The Calendar/Tasks path should be equally specific.

## Expected output
`doJSON` distinguishes an authorization 403 from a throttling 403. Google returns a JSON error body
carrying `error.errors[].reason`; when that reason is one of `rateLimitExceeded`,
`userRateLimitExceeded`, `dailyLimitExceeded` or `quotaExceeded` (or the status is 429), the call
yields `*UpstreamError` — the handler already answers `502 google_unavailable` for that, which is the
honest "try again later" — and only a genuine 401, or a 403 whose reason is
`insufficientPermissions`/`forbidden`, yields `ErrReauthRequired`. A test in `calendar_test.go`
drives a 403 `rateLimitExceeded` body and asserts `*UpstreamError`, alongside the existing 403
`insufficient scopes` case (`calendar_test.go:110-113`) which must keep mapping to
`ErrReauthRequired`. CODEMAP's `google` paragraph is corrected to match.

## Evidence
- Plan: `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md` (reviewed 2026-09-23).
- `backend/internal/google/client.go:74-75` — `resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden` → `ErrReauthRequired`, with no inspection of the body.
- `backend/internal/google/handler.go:34-37` — that sentinel becomes `409 reauth_required`.
- `backend/internal/google/oauth.go:65-73` — the contrasting, correct treatment: the body is decoded and only `invalid_grant` (or a 401) is reauth.
- `backend/internal/google/calendar_test.go:104-113` — the only 403 test uses an `insufficient scopes` body, so the throttling case is neither covered nor distinguished.
- Backend spec §6.4 gives no error catalogue for this route, so the mapping is ours to get right.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — medium.** Confirmed at `client.go:74-75`: every 401/403 → `ErrReauthRequired` → 409 and a pointless full re-consent; Google's quota errors are 403s. A user who syncs twice quickly (28 Tasks inserts each) can hit this. Fix: decode `error.errors[].reason` and map the rate/quota reasons (and 429) to `*UpstreamError` → 502; keep `insufficientPermissions`/401 → reauth. Small `google` plan; can share a branch with the orphan fix if the owner prefers fewer `google` merges, but it is independent.

## Evaluation — 2026-09-24 daily planning (evaluator)
_Owner instruction 2026-09-24: pick ≤ 5 one-day tickets from the `selected` backlog, split Bug team / Feature team, write and approve the plans, one planning PR._

**Planned today — Bug team ticket B1 (head idea). Estimate 4 h.**
*Root cause (re-read on `main` @ 9517f25).* `backend/internal/google/client.go:74-75` — `doJSON` maps every 401 **and** 403 to `ErrReauthRequired` without decoding the body, while `oauth.go:65-73` decodes the body and claims reauth only on `invalid_grant`/401. Google's Calendar v3 / Tasks v1 quota answers (`rateLimitExceeded`, `userRateLimitExceeded`, `dailyLimitExceeded`, `quotaExceeded`) are 403s and so become `409 reauth_required`, sending the user through consent for nothing.
*Why today.* Happy-path for every user who syncs: one sync is up to 30 Tasks calls, nothing rate-limits the route, so a second tap can hit a per-minute quota and dead-end in a consent loop. Fix is confined to `client.go` (decode `error.errors[].reason`), `handler.go` (`ErrAlreadyExists` → 502, one log line) and their tests.
*Folded into this ticket* (same two files, same class — a `doJSON`/`SyncHandler` mapping too broad for the one call site that motivated it): `every-google-409-now-becomes-500-internal-error-instead-of-5.md` and `the-google-sync-route-logs-nothing-so-a-502-or-500-discards-.md`. One branch, one plan, one review.
*One-day check.* One package, two production files, no migration, no frontend, no new dependency.
