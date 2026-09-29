---
type: bug
status: planned
source: reviewer
run: _inbox
priority: low
plan: harness/plans/2026-09-27-google-403-accessnotconfigured-api-disabled-still-maps-to-re.md
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

_Evaluator, 2026-09-27 — daily decide (bug queue, planned today as **B2**)._ **Select — low; planned today.** Confirmed on `origin/main` by reading the code (not reproduced live — the owner's project has both APIs enabled): `backend/internal/google/client.go:113` — `case resp.StatusCode == http.StatusForbidden && !throttleReasons[googleErrorReason(raw)]` → `ErrReauthRequired`, with `throttleReasons` (`client.go:27-32`) holding only the four quota reasons and `googleErrorReason` (`client.go:36-48`) reading only `error.errors[0].reason`. So a 403 `accessNotConfigured` (Calendar/Tasks API disabled in the GCP project — Google's body carries `errors[0].reason: "accessNotConfigured"` plus `status: "PERMISSION_DENIED"` and an `ErrorInfo` detail `reason: "SERVICE_DISABLED"`) is `ErrReauthRequired` → `handler.go:39` 409 `reauth_required`, and the PWA loops the user through consent. `calendar_test.go:148-186` pins only `forbidden`/`noscope`/`notjson403` → reauth and two throttle reasons → upstream; nothing pins a configuration 403. **Root cause:** reauth is the default branch of a deny-list, so every reason nobody listed is treated as "re-consent fixes it". **Fix decision — allow-list.** A 403 is reauth only when a parsed reason is a scope/permission reason (`insufficientPermissions`, `forbidden`, or the `ErrorInfo` detail `ACCESS_TOKEN_SCOPE_INSUFFICIENT`) or when no reason can be parsed at all (empty / HTML 403 — the legacy shape, kept as reauth as this idea asks); any other parsed reason is `*UpstreamError` → 502, and `throttleReasons` disappears (subsumed). Why not extend the deny-list: 409 is a directive the client cannot recover from when it is wrong (consent → sync → consent), while 502 is an honest "Google refused, retry" whose server log already carries the reason (`logSyncFailure`), so an unknown reason must land on the recoverable side; the scopes are fixed by `auth.Scopes` and Google documents exactly those reasons for scope problems on Calendar v3 / Tasks v1, so the allow-list is short and stable, whereas the set of non-auth 403 reasons (`accessNotConfigured`, `SERVICE_DISABLED`, `forbiddenForServiceAccounts`, `domainPolicy`, the quota family, …) is open-ended. Folds `no-test-pins-a-409-from-calendar-events-patch-…` (Task 3), `fakecalendar-returns-the-same-nextid-…` (Task 2) and `practiceeventid-is-a-lossy-filter-…` (Task 4): all live in `backend/internal/google/` outside `token*.go` / `integration_test.go`, which no unmerged branch edits, so the plan merges cleanly into today's integration branch. Plan: `harness/plans/2026-09-27-google-403-accessnotconfigured-api-disabled-still-maps-to-re.md`.
