---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# JWT verify does not require exp or bind iss/aud and Bearer is case-sensitive

## Why
Three small hardening gaps in `TokenIssuer.Verify` and the middleware's header parsing. None is
exploitable on its own today; each removes a layer that costs one line to keep.

- **`exp` is validated but not required.** `jwt.ParseWithClaims` is called with
  `WithValidMethods` and `WithTimeFunc` but not `jwt.WithExpirationRequired()`. golang-jwt/v5 only
  enforces `exp` when the claim is present, so a token signed with the correct secret but carrying
  no `exp` verifies forever. `Issue` always sets one, so this only matters if a token is ever minted
  elsewhere with the same secret — which is exactly the scenario the next point is about.
- **No `iss` / `aud` binding.** The claims are bare `RegisteredClaims` with only `sub`, `iat`, `exp`,
  and `Verify` checks neither issuer nor audience. If `JWT_SECRET` is ever shared with another
  service — reused across environments, copied into a worker, or set to the same value in staging
  and production, all easy to do given that `JWT_SECRET` is absent from the spec §8 env list — that
  service's tokens authenticate here. Setting and checking `iss` and `aud` makes the secret's blast
  radius explicit.
- **`Bearer` is matched case-sensitively.** `bearerToken` does
  `strings.HasPrefix(header, "Bearer ")`, so `bearer <tok>` — which RFC 6750 §2.1 and RFC 7235 §2.1
  make legal, the scheme being case-insensitive — is rejected as unauthenticated. Some HTTP clients
  and proxies normalise the scheme to lowercase. This will read as a mysterious 401 to whoever hits it.

Worth noting what is already right, so a fix does not disturb it: the algorithm **is** pinned twice
(the keyfunc rejects non-HMAC methods and `WithValidMethods` restricts to HS256), and
`TestVerifyRejectsTheNoneAlgorithm` and `TestVerifyRejectsAnotherSecret` prove it. That part is good.

## Expected output
- `jwt.ParseWithClaims` also gets `jwt.WithExpirationRequired()`, with a test that a same-secret token
  carrying no `exp` is rejected.
- `Issue` sets `Issuer` and `Audience` to a package constant, and `Verify` passes
  `jwt.WithIssuer(...)` / `jwt.WithAudience(...)`, with a test that a token from another issuer is rejected.
- `bearerToken` matches the scheme case-insensitively (`strings.EqualFold` on the first field), with
  `bearer`, `BEARER` and `Bearer` in the existing table at `middleware_test.go:59`.
- Consider a small `jwt.WithLeeway` (30-60s) so a client whose clock is slightly ahead of the server
  is not rejected at the edge of the 24h window; strict is defensible, but it should be a decision.

## Evidence
- Plan: `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md`
- `backend/internal/auth/token.go:47-56` — parse options: `WithValidMethods`, `WithTimeFunc`; no `WithExpirationRequired`, no issuer/audience, no leeway.
- `backend/internal/auth/token.go:33-37` — claims are `sub`, `iat`, `exp` only.
- `backend/internal/auth/middleware.go:52-58` — `const prefix = "Bearer "`, `strings.HasPrefix`.
- `backend/internal/auth/middleware_test.go:59` — the malformed-header table has no lowercase-scheme case.
- RFC 6750 §2.1 / RFC 7235 §2.1 — the auth scheme is case-insensitive.
- Related: `harness/ideas/_inbox/jwt-secret-is-accepted-at-any-length-including-one-character.md`.
