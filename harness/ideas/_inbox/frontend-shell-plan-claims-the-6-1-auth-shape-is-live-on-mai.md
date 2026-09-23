---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md
rejected_reason: "Resolved by reality: PR #1 merged the auth §6.1 shape to main on 2026-09-23, so the plan's claim is now true. No fix to write."
---
# frontend-shell plan claims the §6.1 auth shape is live on main; it is not, so the merged frontend cannot sign in

## Why
The whole daily loop is behind sign-in. `stores/auth.ts` deliberately reads the Backend spec §6.1 body `{access_token, token_type, expires_in, user}` and has no fallback to the merged handler's `{token, user}`. That shape is **not** on `main`: the fix lives on an unmerged branch. If `harness/2026-09-23-high-frontend-shell-...` is merged before the auth-shape branch, every sign-in against `main` dies at `pages/login.vue:36-39` with "Máy chủ trả về phiên đăng nhập không hợp lệ" and no user can reach any screen.

The plan's *Notes* section is correct and still says so: "This slice cannot merge until ... is fixed on `main` ... The reviewer should file it as a blocker against this plan at review time if it is still open." It is still open. But the executor's "stale-passage correction" rewrote two other passages of the same plan to assert the opposite, so a human reading the top of the plan would conclude the dependency is discharged.

## Expected output
1. The plan's two rewritten passages state what is true. The header currently reads **"Merge blocker — RESOLVED before execution"** and "That shape is **now live on `main`**: ... and is awaiting merge on `harness/2026-09-23-high-auth-google-response-...`" — self-contradictory ("live on `main`" *and* "awaiting merge"). The API→UI matrix row for `POST /api/v1/auth/google` reads "**live on `main`** — the auth response-shape fix landed". Both must say the fix is **done but unmerged on its own branch**, and that this branch must not merge first.
2. Merge order is honoured: `harness/2026-09-23-high-auth-google-response-returns-token-and-omits-token-type-and-` merges to `main` before this frontend branch.
No implementation change is needed — `stores/auth.ts` already targets the spec shape correctly.

## Evidence
- Plan: `harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md` — header *Merge blocker* paragraph, the `POST /api/v1/auth/google` row of *API → UI mapping matrix*, and the contradicting *Notes and open questions* → "Merge blocker, restated".
- `main` still emits the old shape: `git show main:backend/internal/auth/handler.go` line 32 → `"token": out.Token,`. No `access_token` / `token_type` / `expires_in` anywhere in that file on `main`.
- The fix is unmerged: `harness/plans/2026-09-23-auth-google-response-returns-token-and-omits-token-type-and-.md` frontmatter is `status: done`, `merged: false`.
- Frontend behaviour under the `main` shape is by design: `stores/auth.ts:59` returns early unless `access_token` is a string, and `tests/unit/authStore.test.ts:31` pins that.
- `harness/CODEMAP.md` carries the same wrong claim: "§6.1 `{access_token, expires_in, user}` — **the spec shape, now live on `main`**".
