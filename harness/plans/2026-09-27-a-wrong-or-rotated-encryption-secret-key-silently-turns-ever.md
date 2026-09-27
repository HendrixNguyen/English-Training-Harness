---
idea: harness/ideas/_inbox/a-wrong-or-rotated-encryption-secret-key-silently-turns-ever.md
status: approved
priority: medium
merged: false
---
# A wrong or rotated ENCRYPTION_SECRET_KEY silently turns every Google sync into reauth_required with no log line — Plan

**Idea:** `harness/ideas/_inbox/a-wrong-or-rotated-encryption-secret-key-silently-turns-ever.md`
**Goal:** `openStored` logs one server-side line when a stored refresh token fails to authenticate (`secrets.ErrOpen`: wrong key, rotated key, tampered row), notes a legacy plaintext row (`secrets.ErrNotSealed`) once at info, stays silent for an empty column, and never logs the stored value. The `409 reauth_required` contract does not change.

**Root cause:** `backend/internal/google/token.go` `openStored` wraps every `Opener.Open` failure as `fmt.Errorf("%w: stored value unusable (%v)", ErrNoRefreshToken, err)`. `Service.Sync` (`service.go:46`) turns that into `ErrReauthRequired`, and `SyncHandler` (`handler.go:38-42`) answers 409 without calling `logSyncFailure`, because that case returns before the logging branch. Nothing on the path logs, so a key mismatch looks exactly like normal re-consent traffic.

**Scope / files:** `backend/internal/google/token.go`, `backend/internal/google/token_test.go`, `harness/CODEMAP.md` (google bullet). No other file. Today's unmerged google branch (`harness/2026-09-27-low-google-403-accessnotconfigured…`) edits `client.go`, `schedule.go`, `fakes_test.go`, `handler_test.go`, `calendar_test.go`, `schedule_test.go` and `tasks_test.go`, none of which this plan touches. Leave `handler.go`, `service.go` and `backend/.env.example` alone: `.env.example:59-60` already says rotating the key forces a mass re-consent.

**Out of scope (the idea's "optional" boot-time key fingerprint):** it needs a migration or a canary row. It is a later step.

## Tasks

### Task 1: Failing tests for the log line (token_test.go)
Files: `backend/internal/google/token_test.go`.
1. Change the existing `openStored(box, stored)` calls to the new signature `openStored(box, "u1", stored)`. They will not compile until Task 2.
2. Add `captureLog(t) *bytes.Buffer`, which swaps `log.SetOutput` and restores it in `t.Cleanup` (same pattern as `handler_test.go:118-121`, copied, not shared, so this plan does not edit `handler_test.go`).
3. `TestOpenStoredLogsAnUndecryptableTokenWithTheUserIDAndNoValue`: seal under `testBox`, then open with a second box whose key is `bytes.Repeat([]byte{7}, secrets.KeyBytes)`. That is the wrong-key case. Also run a tampered value. For each, assert:
   - `errors.Is(err, ErrNoRefreshToken)` still holds, so the 409 is unchanged.
   - `errors.Is(err, secrets.ErrOpen)` is now true, so a caller can tell a wrong key from re-consent.
   - The log contains `google: refresh token for user=u1 did not decrypt` and the sentinel text `ciphertext did not authenticate`.
   - The log does **not** contain the sealed string or `1//refresh`.
4. `TestOpenStoredNotesALegacyRowAndStaysQuietForAnEmptyColumn`: the legacy plaintext row `"1//refresh"` logs a line containing `user=u1` and `not a v1 ciphertext` and does not contain `1//refresh`, and `errors.Is(err, secrets.ErrNotSealed)` holds. The empty value `""` logs nothing (`buf.Len() == 0`) and returns exactly `ErrNoRefreshToken`.
5. Run `cd backend && go test ./internal/google/ -run OpenStored -count=1`. Expect a compile failure or FAIL.

### Task 2: Implement (token.go)
Files: `backend/internal/google/token.go`.
1. Change the signature to `openStored(opener Opener, userID, stored string) (string, error)`. `RefreshToken` passes `userID`.
2. Keep empty → `ErrNoRefreshToken`, with no log.
3. On an `Open` error:
   - `errors.Is(err, secrets.ErrNotSealed)`: `log.Printf("google: refresh token for user=%s is a legacy unsealed value; reauth required (%v)", userID, err)`.
   - Any other error, including `secrets.ErrOpen`: `log.Printf("google: refresh token for user=%s did not decrypt — check ENCRYPTION_SECRET_KEY if this repeats across users (%v)", userID, err)`.
   - Return `fmt.Errorf("%w: stored value unusable: %w", ErrNoRefreshToken, err)`. The double `%w` (Go ≥ 1.20) keeps both sentinels matchable.
   - The `err` values from `secrets` are fixed sentinels that never carry the input. Say so in a one-line comment, so the "never the value" rule is visibly upheld.
4. Update the `ErrNoRefreshToken` doc comment: the reason is logged server-side, and the wrapped `secrets` sentinel tells the cases apart.
5. Run `cd backend && go test ./internal/google/ -count=1`. Expect PASS, including the existing `TestOpenStoredMapsEmptyLegacyAndTamperedToErrNoRefreshToken` and `TestSyncHandlerLogsTheFailureServerSideOnly`.
6. Run `gofmt -l internal/google` (expect empty output) and `go vet ./internal/google/`.
7. Commit: `google: log undecryptable and legacy refresh tokens once, never the value`.

### Task 3: CODEMAP
Files: `harness/CODEMAP.md`, google bullet. After "an empty, unsealed (pre-encryption) or undecryptable value is `ErrNoRefreshToken` → `409 reauth_required`", add this sentence: "an undecryptable value (`secrets.ErrOpen`: wrong/rotated `ENCRYPTION_SECRET_KEY` or a tampered row) logs `google: refresh token for user=… did not decrypt` and a legacy plaintext row logs once at info; an empty column is silent; the stored value is never logged". Commit: `harness: CODEMAP — google logs undecryptable refresh tokens`.

## Verification
```bash
cd backend
go test ./internal/google/ -count=1 -v -run 'OpenStored|SyncHandler'
go test ./... -count=1
gofmt -l . && go vet ./...
grep -n 'did not decrypt' internal/google/token.go ../harness/CODEMAP.md
```
- Wrong-key and tampered rows: 409 is unchanged, one log line names `user=` and the sentinel, and neither the sealed value nor the plaintext appears in the log.
- Legacy row: one info line. Empty column: no log.
- CI (`backend-unit`, `backend-integration`) is green on the pushed branch.
