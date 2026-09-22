---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# signing in on a second device silently logs the first one out

## Why
`sess:{user_id}:token` holds exactly one token per user, and `Require` accepts a presented token only
if it equals the stored one byte for byte. So the session model is strictly one device per user:
signing in on a phone invalidates the laptop's session immediately and invisibly. The laptop's next
request returns `401 unauthorized`, indistinguishable from a revoked or expired session, and the user
is bounced to the consent screen — which, because `AuthCodeURL` sets `prompt=consent`, shows the full
Google permission dialog again rather than a silent re-auth.

This was a deliberate design decision (the plan records it, `TestRequireRejectsASupersededToken`
asserts it, and CODEMAP documents it), and single-session is a defensible choice — but it was taken
to make revocation work, not because the product wanted it. Spec §4 describes
`sess:{user_id}:token` as "Active JWT session context and user metadata" and says nothing about
limiting a user to one device. The product is an installable PWA whose §5.1 daily loop is built around
a notification the user taps — plausibly on a phone, while their actual study session is on a laptop.
Two devices ping-ponging each other's sessions all day is a bad first impression, and it will be
reported as a login bug, not recognised as a design choice.

Better to settle it now, before the six remaining slices all mount behind `Require` and the semantics
harden.

## Expected output
A decision, recorded in CODEMAP, and the code matching it. The two credible options:

1. **Multi-device (recommended).** Key each session by a token id rather than by user: give the JWT a
   `jti`, store `sess:{user_id}:{jti}` (or a Redis SET of live `jti`s under the §4 key), and have
   `Require` check membership rather than equality. Revoking one device deletes one member; revoking
   everything deletes the set. The §4 key shape is preserved as a prefix and the TTL is unchanged.
2. **Single-device, but honestly.** Keep the current behaviour, return a response the client can tell
   apart (a distinct error code for "superseded" vs "expired" vs "revoked"), and state the limit in
   CODEMAP and in whatever the frontend-shell slice shows the user.

Either way: a test covering two concurrent sign-ins for the same user asserting the intended outcome,
rather than only the current `TestRequireRejectsASupersededToken`.

## Evidence
- Plan: `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md` (design decision recorded in the idea's Evaluation, and in the plan's Notes).
- `backend/internal/auth/service.go:50` — `s.sessions.Put(ctx, user.ID, jwtToken, TokenTTL)` overwrites unconditionally.
- `backend/internal/auth/middleware.go:32-36` — `stored != raw` → 401.
- `backend/internal/auth/middleware_test.go:95-105` — `TestRequireRejectsASupersededToken` pins the single-device behaviour.
- `backend/internal/store/keys.go:21-22` — `SessionKey(userID) = sess:{userID}:token`, one key per user.
- Spec §4 line 262 — describes the key, does not require a single active session.
