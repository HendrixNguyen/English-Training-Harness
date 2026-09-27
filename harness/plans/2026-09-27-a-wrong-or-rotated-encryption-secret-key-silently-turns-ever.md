---
idea: harness/ideas/_inbox/a-wrong-or-rotated-encryption-secret-key-silently-turns-ever.md
status: done
priority: medium
merged: true
branch: harness/2026-09-27-medium-a-wrong-or-rotated-encryption-secret-key-silently-turns-ever
worktree: .worktrees/a-wrong-or-rotated-encryption-secret-key-silently-turns-ever
pr: "https://github.com/HendrixNguyen/English-Training-Harness/pull/53"
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

## Execution summary

**Built** on `harness/2026-09-27-medium-a-wrong-or-rotated-encryption-secret-key-silently-turns-ever` (from origin/main `0392e5d`): `740a0e2` google: log undecryptable and legacy refresh tokens once, never the value; `7b66edc` harness: CODEMAP — google logs undecryptable refresh tokens. Files: `backend/internal/google/token.go`, `backend/internal/google/token_test.go`, `harness/CODEMAP.md`.

**Deviations**
- Task 1.2: did not add `captureLog` to `token_test.go`. `handler_test.go` already declares it in package `google`, so a copy would not compile (a redeclaration). The new tests reuse that helper, and `handler_test.go` is still unedited.
- Plan premise (not a code deviation): the root cause says `SyncHandler` answers 409 without calling `logSyncFailure`. On origin/main it does log (commit `4e50505`: `logSyncFailure` runs for every error, before the switch). So before this fix, a wrong key already produced `google: sync user=… failed: … stored value unusable (secrets: ciphertext did not authenticate)`. It had no operator hint, and `errors.Is(err, secrets.ErrOpen)` was false. The fix still adds the wrapped sentinels and a distinct `did not decrypt — check ENCRYPTION_SECRET_KEY` line. A bad-key or legacy row now logs **two** lines: the openStored line and the handler's failure line. An empty column is silent in `openStored`, but the handler still logs its one failure line. The `handler.go` side is out of scope; reviewer/evaluator may want to de-duplicate it.

**Reproduction** (signature changed, body not yet changed):
```
--- FAIL: TestOpenStoredLogsAnUndecryptableTokenWithTheUserIDAndNoValue
    wrong key: err = google: no refresh token on file: stored value unusable (secrets: ciphertext did not authenticate), want it to wrap secrets.ErrOpen so a wrong key is distinguishable
    wrong key: log "", want it to contain "google: refresh token for user=u1 did not decrypt"
    tampered: (same two failures)
--- FAIL: TestOpenStoredNotesALegacyRowAndStaysQuietForAnEmptyColumn
    legacy: err = … (secrets: value is not a v1 ciphertext), want it to wrap secrets.ErrNotSealed
    legacy: log "", want user=u1 and the ErrNotSealed reason
```
After the fix, both tests PASS.

**Verification** (clean shell, DATABASE_URL/REDIS_URL/TEST_*/ENCRYPTION_SECRET_KEY/AI keys unset)
```
go test ./internal/google/ -count=1 -v -run 'OpenStored|SyncHandler'   → 14 PASS incl. TestOpenStoredMapsEmptyLegacyAndTamperedToErrNoRefreshToken, TestSyncHandlerLogsTheFailureServerSideOnly; ok
go test ./... -count=1        → ok for all 13 packages
gofmt -l .                    → (empty)
go vet ./...                  → ok
grep -n 'did not decrypt'     → internal/google/token.go:68, ../harness/CODEMAP.md:18
```

### Runtime proof
- `go build ./...` ok. `make check` (fmt-check, vet, `-race` tests) ok for all packages.
- Compose project `enc-key` (Postgres 55452, Redis 56402) came up healthy. `make test-integration` with TEST_* pointed at it: ok for all packages.
- The API binary was booted on :18102 with `ENCRYPTION_SECRET_KEY=bb…` (migrations 0001–0003 applied). Three users were seeded: a token sealed under key `aa…`, the legacy plaintext `1//legacy-plain`, and NULL. Each got a real JWT and session, then `POST /api/v1/integrations/google/sync`:
```
user=1111… -> {"error":"reauth_required"} HTTP 409
user=2222… -> {"error":"reauth_required"} HTTP 409
user=3333… -> {"error":"reauth_required"} HTTP 409
google: refresh token for user=1111… did not decrypt — check ENCRYPTION_SECRET_KEY if this repeats across users (secrets: ciphertext did not authenticate)
google: sync user=1111… failed: google: re-authentication required: google: no refresh token on file: stored value unusable: secrets: ciphertext did not authenticate
google: refresh token for user=2222… is a legacy unsealed value; reauth required (secrets: value is not a v1 ciphertext)
google: sync user=2222… failed: … stored value unusable: secrets: value is not a v1 ciphertext
google: sync user=3333… failed: google: re-authentication required: google: no refresh token on file
leak check (log lines containing the sealed value, 1//refresh-proof or 1//legacy-plain): 0
```
- Cleanup: API killed, `make down` (enc-key), scratch `.env` and helper removed. `docker ps` shows no enc-key containers.

**CI:** https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36296768886 — success (backend-unit, backend-integration, frontend, docker-images, harness-tooling).
