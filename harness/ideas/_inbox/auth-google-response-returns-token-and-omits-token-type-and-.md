---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
plan: harness/plans/2026-09-23-auth-google-response-returns-token-and-omits-token-type-and-.md
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

## Evaluation
**Verdict: select, `priority: high`** — this is MVP-enabling work, not an ordinary inbox bug. The
frontend-shell plan (`harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md`,
MVP order 9, approved) reads `access_token` / `expires_in` with **no** fallback to `token` (its header
records the merge blocker; its `useAuthStore.signIn` returns early when `res.access_token` is not a
string). Ranking rule 2 — an MVP slice would build on something broken — puts this ahead of the
remaining slices; it must land on `main` before frontend-shell can merge.

**Claims verified on `main` @ `60456e8` (2026-09-23), nothing has drifted:**
- `backend/internal/auth/handler.go:31-39` — `c.JSON(http.StatusOK, gin.H{"token": out.Token, "user": gin.H{…}})`.
- `backend/internal/auth/handler_test.go:47` — decoded struct field `Token string \`json:"token"\`` pins the wrong key.
- `backend/internal/auth/token.go:14` — `const TokenTTL = store.SessionTTL`; `store/keys.go:12` — `SessionTTL = 24 * time.Hour` → `expires_in: 86400`.
- Backend spec §6.1 (`Backend Technical Specification.md:251`), the 200 body being implemented:
  `{"access_token": "eyJ…", "token_type": "Bearer", "expires_in": 86400, "user": {"id": "…", "email": "…", "full_name": "…", "cefr_current": "B1"}}`.
- The request body `{code, redirect_uri}` (`handler.go:10-13`) already matches §6.1 — one-shape fix confirmed.

**Root cause (systematic-debugging):** the auth plan
(`harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md`) was written against the
1st-thinking doc alone, whose §7 line 670 says only "OAuth code token swap & JWT issuance" with no body; the
executor chose `token`, and the review (`harness/reviews/2026-09-22-auth-google-…md:22`) checked the handler
against the *plan*, which it matched. The backend spec that defines the shape arrived afterwards. Contract
drift, not a logic defect — `SignInResult{Token, User}` (`service.go:9-12`) carries everything needed.

**Sibling-drift check (same class, same slice): none.** The auth slice registers one route
(`backend/cmd/api/main.go:95`); spec §6 defines no refresh/logout/me endpoint. The only other
`access_token` / `expires_in` / `token_type` in Go are Google's own token DTO (`google.go:25-28`), which is
correct there. So this is one plan, one shape fix.

**Dependencies / conflicts:** none unbuilt. `git log --all -- backend/internal/auth/handler.go` shows only
the original commit; the one unmerged worktree (onboarding) does not touch it. Ordinary plan — no `amends:`,
no `blocks:`, own worktree and branch.

**Decisions:** no `token` alias (one contract, per spec); `expires_in` is derived from `TokenTTL`
(`int(TokenTTL / time.Second)`) so it can never disagree with the JWT `exp` or the Redis TTL, while the test
asserts the spec literal `86400` so a TTL change forces a spec conversation. Follow a typed response struct,
as the later `pet` handler does (`pet/handler.go:16,42`).
