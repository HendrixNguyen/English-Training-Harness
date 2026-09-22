---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
---
# auth/google response returns token and omits token_type and expires_in required by backend spec 6.1

## Why
The backend spec's §6.1 is the REST contract for `POST /api/v1/auth/google` (AGENTS.md → *Reading the
spec*: "its §6 request/response shapes are the contract"). Its 200 body is

```json
{"access_token": "eyJ...", "token_type": "Bearer", "expires_in": 86400,
 "user": {"id": "...", "email": "...", "full_name": "...", "cefr_current": "B1"}}
```

The merged handler returns `{"token": "...", "user": {...}}`: the JWT is under a different key and
`token_type` / `expires_in` are absent. The `user` object matches (`id`, `email`, `full_name`,
`cefr_current`).

User impact: every sign-in. The frontend spec's `useAuthStore` (Frontend spec §4) is written against
the backend contract, so a client built to spec reads `access_token`, gets `undefined`, and stores no
token — the user is bounced to login on the first guarded call. `expires_in` is also what lets the
client schedule a silent re-login before the 24h session (§7) lapses instead of discovering it by a 401.
The request body (`{"code", "redirect_uri"}`) already matches §6.1, so this is a one-shape fix.

## Expected output
- `auth.Handler` writes exactly the §6.1 shape: `access_token` (the JWT), `token_type: "Bearer"`,
  `expires_in: int(auth.TokenTTL.Seconds())` = 86400, and the existing `user` object.
- No other field (in particular, do not keep `token` as an alias — one contract, one shape).
- `handler_test.go` decodes `access_token`, asserts `token_type == "Bearer"` and `expires_in == 86400`,
  and asserts the body has no `token` key.
- The `auth` paragraph in `harness/CODEMAP.md` names the response fields.

## Evidence
- Found by the evaluator's spec reconciliation of the merged `auth` slice against the backend spec
  (this run), not by a code review; filed with `source: reviewer` because it is a conformance finding.
- Spec: `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` §6.1,
  the `Response (200 OK)` JSON under `POST /api/v1/auth/google` (`access_token`, `token_type`,
  `expires_in`, `user`); §7 second bullet (24-hour TTL → `expires_in: 86400`).
- Code: `backend/internal/auth/handler.go:31-39` — `c.JSON(http.StatusOK, gin.H{"token": out.Token, "user": ...})`.
- Test pinning the wrong shape: `backend/internal/auth/handler_test.go:47` — the decoded struct field `Token` carries the tag `json:"token"`.
- TTL source for `expires_in`: `backend/internal/auth/token.go:14` — `const TokenTTL = store.SessionTTL` (24h).
- Frontend consumer: Frontend spec §4 — `useAuthStore: Manages JWT tokens, user profile metadata (full_name, cefr_current)`.
- Plan that shipped it: `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md`
  (written against the 1st-thinking doc only, before the backend spec existed).
