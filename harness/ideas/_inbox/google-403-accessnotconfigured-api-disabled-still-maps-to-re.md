---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Google 403 accessNotConfigured (API disabled) still maps to reauth_required, looping users through re-consent

## Why
`doJSON` now treats a 403 as `ErrReauthRequired` unless its first `error.errors[].reason` is one of four throttle reasons (a deny-list). Every other 403 still sends the user back through `/login` with `prompt=consent`. Some 403s cannot be fixed by re-consenting: `accessNotConfigured` / `SERVICE_DISABLED` (Calendar or Tasks API not enabled in the GCP project — plausible on the free-tier deployment), `forbiddenForServiceAccounts`, or a quota reason outside the list. For those, every user who taps Sync is bounced to Google consent, comes back, taps Sync, and is bounced again — a loop with no way out on the client. The new server log line now shows the reason, so it is diagnosable, but the user-facing answer is wrong.

The idea's *Expected output* asked for the opposite shape: "only a genuine 401, or a 403 whose reason is `insufficientPermissions`/`forbidden`, yields `ErrReauthRequired`" (an allow-list). The plan chose a deny-list without saying why; the deviation is reasonable as a default for an empty/non-JSON 403, but not for a 403 that carries a known non-auth reason.

## Expected output
A 403 whose reason is known to be a project/configuration problem (at least `accessNotConfigured`, and `error.status == "PERMISSION_DENIED"` with `SERVICE_DISABLED` details if Google sends that form) yields `*UpstreamError` → `502 google_unavailable`, not `409 reauth_required`. Empty-reason / non-JSON 403 and `insufficientPermissions` / `forbidden` keep mapping to reauth. A `calendar_test.go` case pins a 403 `accessNotConfigured` body to `*UpstreamError`; CODEMAP's `google` error sentence is updated.

## Evidence
- Plan: `harness/plans/2026-09-24-every-google-403-becomes-409-reauth-required-so-a-quota-erro.md` (Task 1)
- `backend/internal/google/client.go` on branch `harness/2026-09-24-medium-every-google-403-becomes-409-reauth-required-so-a-quota-erro`: `case resp.StatusCode == http.StatusForbidden && !throttleReasons[googleErrorReason(raw)]:` → `ErrReauthRequired`
- Head idea's *Expected output*: `harness/ideas/_inbox/every-google-403-becomes-409-reauth-required-so-a-quota-erro.md`
- Review: `harness/reviews/` entry for the plan above (2026-09-25)

## Evaluation
_Evaluator, 2026-09-26 — **deferred** (bug cap of 5 reached; status left `proposed`)._ Real loop if the Calendar/Tasks API is disabled in the GCP project; the owner's project has them enabled (sync works live). Plan with the 409 test item as one `google` plan when a slot frees.
