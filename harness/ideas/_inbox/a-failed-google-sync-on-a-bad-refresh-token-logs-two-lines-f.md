---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# A failed Google sync on a bad refresh token logs two lines for one event

## Why
Since the wrong-key logging plan, one undecryptable or legacy refresh token writes **two** log lines per sync:
- `openStored`'s `google: refresh token for user=… did not decrypt — check ENCRYPTION_SECRET_KEY …`;
- `SyncHandler`'s `logSyncFailure` line `google: sync user=… failed: … stored value unusable: secrets: ciphertext did not authenticate`, which already runs for every error.

The plan assumed the handler did not log a 409, but it has since commit `4e50505`. A mass-reauth burst after a key rotation therefore doubles the log volume. Operators also get two differently worded lines for one event, which makes "count the `did not decrypt` lines" the only reliable signal. An empty column logs nothing in `openStored` but one line in the handler, so the three cases are not symmetrical either.

## Expected output
- One line per failed sync that names the cause. Either `openStored` stops logging and `logSyncFailure` gains the key hint when `errors.Is(err, secrets.ErrOpen)`, or `logSyncFailure` skips `ErrReauthRequired` causes that `openStored` already logged.
- The test asserts exactly one line for the wrong-key case through `SyncHandler`.

## Evidence
- Plan: `harness/plans/2026-09-27-a-wrong-or-rotated-encryption-secret-key-silently-turns-ever.md`. The execution summary, "Plan premise", flags the duplicate itself.
- `backend/internal/google/token.go` (`openStored` log calls) and `backend/internal/google/handler.go` (`logSyncFailure` before the switch).
- Reviewer live run (API with key `bb…`, row sealed under `aa…`, `POST /api/v1/integrations/google/sync` → `409 reauth_required`), two lines:
  `google: refresh token for user=2222… did not decrypt — check ENCRYPTION_SECRET_KEY if this repeats across users (secrets: ciphertext did not authenticate)`
  `google: sync user=2222… failed: google: re-authentication required: google: no refresh token on file: stored value unusable: secrets: ciphertext did not authenticate`
