---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# JWT_SECRET is accepted at any length including one character

## Why
`config.Load` validates that `JWT_SECRET` is non-empty and nothing else. `JWT_SECRET=x` boots the
API cleanly — verified by running the branch's `cmd/api` binary with exactly that value.

The session token is HS256. Its security rests entirely on the secret being long and random: a short
or dictionary secret can be brute-forced offline from a single captured JWT in seconds with standard
tooling, after which anyone can mint a token for any `sub` and `Require` will accept it — the Redis
check does not help, because the attacker can sign a token matching whatever is stored, and in any
case can sign one for a user whose session they can observe. This is the whole authentication system
for a product where every other endpoint in §7 is per-user.

The risk is not theoretical for this project specifically: `JWT_SECRET` is **absent from the spec §8
environment list** (the plan and CODEMAP both flag this), so whoever provisions Railway will be
inventing the variable on the spot, from a spec that never mentions it, with no stated requirements.
That is exactly the situation in which someone types `secret` and moves on. The one place that could
catch it is the config loader, and it does not.

## Expected output
`config.Load` rejects a `JWT_SECRET` shorter than 32 bytes with a message that says what is wrong and
what to do — for example:
`config: JWT_SECRET must be at least 32 bytes (generate one with: openssl rand -base64 32)`.

32 bytes is the HMAC-SHA256 block-size floor that RFC 7518 §3.2 requires for HS256 ("a key of the same
size as the hash output or larger"). A table test over the boundary (31 rejected, 32 accepted) keeps it
honest. Optionally warn on a secret with very low character diversity, but length is the load-bearing
check.

While there: `backend/.env.example` and the CODEMAP note should state the requirement, and the
follow-up spec bug about adding `JWT_SECRET` to §8 should carry the minimum length with it.

## Evidence
- Plan: `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md`
- `backend/internal/config/config.go:43-51` — presence-only loop over `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `JWT_SECRET`.
- `backend/internal/config/config_test.go:69-98` — asserts only that each is required, never that a weak value is rejected.
- `backend/internal/auth/token.go:23-28` — `NewTokenIssuer(secret string, ...)` accepts any string as HMAC key material.
- Verified in the plan's worktree: `JWT_SECRET=x` (one character) booted the API and served both routes.
- RFC 7518 §3.2 — HS256 keys MUST be at least the size of the hash output (256 bits / 32 bytes).
