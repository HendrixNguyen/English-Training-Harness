---
type: bug
status: selected
source: reviewer
run: _inbox
priority: high
---
# google_refresh_token is stored in plaintext; backend spec 7 requires AES-256-GCM via ENCRYPTION_SECRET_KEY

## Why
The backend spec says three times that the Google refresh token is encrypted at rest:

- §6.1, `POST /api/v1/auth/google` description: "Swaps Google OAuth authorization code for tokens,
  **encrypts refresh token with AES-256-GCM**, and issues signed JWT."
- §7, first bullet: "`users.google_refresh_token` encrypted with [AES-256-GCM] via
  `ENCRYPTION_SECRET_KEY` (32-byte hex)."
- §9, step 2: `ENCRYPTION_SECRET_KEY` is in the deployment env list.

The merged `auth` slice writes the token Google returns straight into `users.google_refresh_token`
(`service.go:39` passes `tok.RefreshToken` to the upsert; `repo.go:34-39` stores it as-is) and
`grep -rn 'aes\|AES\|encrypt\|ENCRYPTION' backend/` finds nothing — no cipher, no key, no config field,
no `.env.example` entry.

User impact: this is the most sensitive secret the product holds. The consent set (`auth.Scopes`)
includes `calendar.events` and `tasks`, so a plaintext refresh token is standing write access to a
user's Google Calendar and Tasks for as long as it is valid. Any database dump, backup, log line or
read-replica leak exposes it. It has no visible symptom today, which is exactly why it should be fixed
before the first real user row exists: retrofitting means an in-place re-encryption migration of live
rows, and the `google` slice (spec §6.4 `POST /integrations/google/sync`) will start *reading* this
column, so the decrypt path should exist before that slice is planned.

## Expected output
- `config.Load` reads `ENCRYPTION_SECRET_KEY` (required; 32-byte hex → 32 raw bytes, else a clear
  startup error, matching how `JWT_SECRET` is handled) and `backend/.env.example` documents it.
- A small `auth` helper (`Encrypt(plain) → base64(nonce||ciphertext)`, `Decrypt`) over
  `crypto/aes` + `crypto/cipher.NewGCM` with a random 12-byte nonce per call; unit tests for round-trip,
  tampered ciphertext rejected, and two encryptions of the same plaintext differing.
- `Service.SignIn` encrypts before `UpsertByGoogleID`; the empty-token-keeps-stored-value rule in
  `upsertUserSQL` still holds (encrypt only when Google returned a token).
- The `TEST_DATABASE_URL`-gated integration test asserts the stored column is not equal to the
  plaintext and decrypts back to it.
- CODEMAP `auth` paragraph and the spec §8/§9 env-list note updated; `ENCRYPTION_SECRET_KEY` added to
  the CI job env if the integration test needs it.

## Evidence
- Found by the evaluator's spec reconciliation of the merged `auth` slice against the backend spec
  (this run), not by a code review; filed with `source: reviewer` because it is a conformance finding.
  Priority `high` is a judgment: no user-visible symptom, but a spec-mandated security control on the
  most sensitive column, cheapest before any real rows exist and before the `google` slice reads it.
  Downgrade to `medium` if deployment is far off.
- Spec: `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` §6.1
  (description of `POST /api/v1/auth/google`), §7 first bullet, §9 step 2 (`ENCRYPTION_SECRET_KEY`).
- Code: `backend/internal/auth/service.go:39` — `s.users.UpsertByGoogleID(ctx, profile.Sub, profile.Email, profile.Name, tok.RefreshToken)`;
  `backend/internal/auth/repo.go:33-40` — `upsertUserSQL` writes `google_refresh_token` verbatim;
  `backend/internal/auth/integration_test.go:78-80` — asserts the stored value equals the plaintext `"rt-1"`.
- Config: `backend/internal/config/config.go:41-46` — reads `JWT_SECRET`, nothing for `ENCRYPTION_SECRET_KEY`.
- Related: `harness/ideas/_inbox/jwt-secret-is-accepted-at-any-length-including-one-character.md`
  (the same config validation gap for the other secret).
- Plan that shipped it: `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md`
  (written against the 1st-thinking doc only, before the backend spec existed).

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — high (top 10).** Spec §7 mandates AES-256-GCM via `ENCRYPTION_SECRET_KEY`; `auth/repo.go` stores the token verbatim and `google.PgRefreshTokenSource` reads it verbatim. It is standing write access to the user's Calendar and Tasks, and the fix gets strictly more expensive after the first real row (in-place re-encryption). Plan: `config` reads the key; `auth` encrypts on sign-in; `google.NewPgRefreshTokenSource` takes a decrypter (the seam the google plan left for exactly this). Touches `cmd/api/main.go` (one constructor argument) — schedule after the cmd/api hardening plan merges.
