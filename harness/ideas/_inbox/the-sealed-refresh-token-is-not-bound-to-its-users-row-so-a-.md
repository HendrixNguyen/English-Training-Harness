---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# The sealed refresh token is not bound to its users row, so a ciphertext can be moved to another account

## Why
`secrets.Box.Seal` calls `aead.Seal(nonce, nonce, plain, nil)` with **no additional authenticated data**. The ciphertext is authenticated but not bound to the row it belongs to. Anyone who can write `users` but lacks the key (SQL injection elsewhere, a leaked read-write DB credential, a restored backup edited by hand) can copy user A's `v1:` value into their own row and call `POST /api/v1/integrations/google/sync`. `Open` authenticates it, and the app then spends **A's** Google refresh token to push the attacker's roadmap into A's primary calendar and task lists. Encryption at rest is meant to make the column useless without the key. Transplanting is the standard gap an AEAD closes with AAD.

## Expected output
- `Seal`/`Open` take an associated-data argument, and both call sites pass a stable row identity that is known at write time and at read time. `google_id` fits: `auth.UpsertByGoogleID` has it before the insert, and `google.PgRefreshTokenSource` can `SELECT google_id, google_refresh_token`. A value moved to another row then fails to open (`ErrOpen` → 409).
- The format change goes behind a new prefix (`v2:`) so the `v1:` rows written by this plan still open, or follow the plan's self-healing cutover.
- A test seals for row A and asserts that opening it as row B fails.

## Evidence
- Plan: `harness/plans/2026-09-24-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md` (review: `harness/reviews/2026-09-24-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md`).
- `backend/internal/secrets/secrets.go:71`: `b.aead.Seal(nonce, nonce, []byte(plain), nil)`, and `:86` `b.aead.Open(nil, raw[:n], raw[n:], nil)`. The AAD is nil in both.
- `backend/internal/google/token.go:41`: `refreshTokenSQL` reads only the ciphertext by `id`, with nothing that ties it to the row.
