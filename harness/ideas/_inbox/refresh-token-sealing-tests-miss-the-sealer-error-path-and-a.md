---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Refresh-token sealing tests miss the sealer-error path and a short-but-valid v1 payload, and one can panic instead of fail

## Why
The new tests kill every unit mutation in the plan's table (the reviewer ran six), but a few branches added by this plan have no test, and one test can crash instead of reporting:
1. **The sealer-error path in `auth.PgUserRepo.UpsertByGoogleID` (`backend/internal/auth/repo.go:67-70`) is never exercised.** The only caller in tests uses a real `secrets.Box`, which cannot fail. Deleting the `if err != nil { return … }` would not be noticed, and a nil `sealer` panics at sign-in instead of failing at wiring.
2. **`secrets.Open`'s `len(raw) < n` branch (`backend/internal/secrets/secrets.go:83`) is untested.** Every existing case is either bad base64 or a full sealed value. A valid base64url payload of 1–11 bytes (shorter than the nonce), or of exactly 12 bytes (nonce with no tag), is not pinned to `ErrOpen`.
3. **`backend/internal/google/token_test.go:41` calls `err.Error()` without a nil check.** If `openStored` ever returned `(plain, nil)` for the legacy value, the suite would panic rather than fail on that assertion.
4. **`TestVerifyRejectsATokenWithoutExp` (`backend/internal/auth/token_test.go:85`) asserts only `err != nil`,** not `errors.Is(err, jwt.ErrTokenRequiredClaimMissing)`. It currently kills the mutation, but any unrelated verify failure would also satisfy it.
5. Minor: `google/token_test.go` ignores `Seal`'s error in two places. `ParseHexKey` has no case for uppercase hex (accepted by `hex.DecodeString`) or for whitespace inside the value.

## Expected output
- A fake `Sealer` that returns an error, and a test that `UpsertByGoogleID` returns it wrapped (`auth: sealing refresh token`) without touching the DB. `NewPgUserRepo` either documents that `sealer` must be non-nil or checks it.
- `TestOpenRejectsATruncatedPayload`: `v1:` plus base64url of 0, 11, 12 and 27 bytes, each giving `ErrOpen` with no panic.
- `google/token_test.go:40-43` checks `err != nil` before `err.Error()`. The no-exp test asserts the specific jwt sentinel.

## Evidence
- Plan: `harness/plans/2026-09-24-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md` (review: `harness/reviews/2026-09-24-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md`), and the reviewer's test-gap pass over `git diff origin/main...harness/2026-09-24-high-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r`.
- `grep -n 'Sealer' backend/internal/auth/*_test.go` finds no fake.
