---
type: bug
status: planned
source: human
run: _inbox
priority: high
plan: harness/plans/2026-09-25-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect.md
---
# Nobody can sign in on Cloudflare Pages: /login is 308-redirected to /login/ and the auth middleware drops Google's code

## Why
First production sign-in attempt (owner, 2026-09-25, `https://english-learning-e6a.pages.dev`): after Google consent the user lands back on the plain login page and the API log shows **no** `POST /api/v1/auth/google` at all. Sign-in is the front door of the whole retention loop (spec §5.1); on the live host it is impossible for every user.

Root cause (two halves, both needed):
1. `nuxi generate` writes `login/index.html` (and `onboarding/index.html`, …). Cloudflare Pages therefore answers `GET /login?code=…&state=…` with `308 Location: /login/?code=…&state=…` (directory-index canonicalisation). The registered Google redirect URI is `<origin>/login` (`frontend/pages/login.vue` `redirectUri`), so every return trip goes through this redirect.
2. `frontend/middleware/auth.global.ts` special-cases only `to.path === '/login'`. With `to.path === '/login/'` it falls into the protected-route branch: `auth.signOut()` then `navigateTo('/login', {replace: true})` — the query (code, state) is discarded, so the page renders the sign-in button and nothing calls the API.

The Caddy image (`frontend/Caddyfile`, `try_files … /index.html`) serves `/login` directly, so local compose and Target B never hit this; the runbook's "Pages serves the SPA fallback by itself" masked it.

## Expected output
- `GET /login?code=x&state=y` on Pages loads the login page with the query intact, `finishSignIn` runs and `POST /api/v1/auth/google` appears in the Railway log; a real sign-in lands on `/` (or onboarding).
- Fix both halves: (a) `nuxt.config.ts` → `nitro: { prerender: { autoSubfolderIndex: false } }` so generate emits `login.html` etc. and Pages serves `/login` without a redirect (Pages then redirects `/login/` → `/login`, which the middleware handles); (b) middleware compares a normalised path (`to.path.replace(/\/+$/, '') || '/'`) so a trailing slash from any static host can never eject a returning user. Unit test for (b); a smoke-web check `GET /login?code=x&state=y` → 200 with no redirect (`curl -o /dev/null -w %{http_code}` without `-L`).
- Related inbox bug "Cloudflare Pages serves 404 for deep links and no-cache headers…" (`_redirects`/`_headers`) is the same hosting mismatch and can be planned together.

## Evidence
```
$ curl -sI 'https://english-learning-e6a.pages.dev/login?code=x&state=y' | grep -iE '^(HTTP|location)'
HTTP/2 308
location: /login/?code=x&state=y
$ find frontend/.output/public -maxdepth 2 -name '*.html'
/index.html /404.html /200.html /settings/index.html /roadmap/index.html /revive/index.html /login/index.html /onboarding/index.html
$ railway logs --service api --deployment | grep -v healthz | tail
2026/09/25 07:46:52 listening on [::]:8080 (GIN_MODE=release)      # nothing after boot — no auth POST ever arrived
```
`frontend/middleware/auth.global.ts` lines 7–18 (exact-path check, then `navigateTo('/login')` without the query). Nuxt/Nitro `prerender.autoSubfolderIndex`: https://nitro.build/config#prerender ; Pages trailing-slash behaviour: https://developers.cloudflare.com/pages/configuration/serving-pages/

## Evaluation
**Verdict: select, `priority: high`, planned together with the `_redirects`/`_headers` inbox bug.**

*Is the Why real?* Yes — sign-in is the only door into the product (spec §5.1) and it is broken for every user of the live host. The evidence is first-hand (a `curl -sI` showing `308 → /login/?code=…`, the generate output listing `login/index.html`, and an API log with no auth POST). Nothing else on the roadmap matters while this stands, and it is the definition of a `high` bug: blocks users on the happy path.

*Root cause, verified read-only against the frontend source on the deploy branch's worktree (identical to `origin/main` for `frontend/`):*
1. `frontend/nuxt.config.ts` has no `nitro.prerender` block, so Nitro 2.13.4 (the resolved `nitropack`) keeps its default `autoSubfolderIndex: true` and `nuxi generate` writes `login/index.html`. Cloudflare Pages canonicalises a directory index to the trailing-slash form, hence `GET /login?code=…` → `308 /login/?code=…`. The Caddy image (`try_files {path} /index.html`) never redirects, which is why compose/Target B and the CI `docker-images` job all pass.
2. `frontend/middleware/auth.global.ts` lines 7–18: the `/login` branch is `if (to.path === '/login')`; `/login/` misses it, falls to `if (!auth.isAuthenticated)` (true — no session yet), calls `auth.signOut()` and `navigateTo('/login', { replace: true })` — a path string with no query, so `code`/`state` are gone and `pages/login.vue` (`const { code, state } = route.query`) never calls `finishSignIn`. `tests/unit/authMiddleware.test.ts` exercises only `/` and `/login`, so no test could catch this.

*Smallest correct fix:* both halves, exactly as the idea proposes. (a) `nitro: { prerender: { autoSubfolderIndex: false } }` → flat `login.html`, `onboarding.html`, … which Pages serves at the extension-less path with no redirect. (b) Compare a normalised path in the middleware (`to.path.replace(/\/+$/, '') || '/'`) so any static host that adds a trailing slash cannot eject a returning user — a defence for half (a), and it makes the existing `/login` tests cover `/login/` too. Regression unit test for (b): `/login/?code=…&state=…` with no session must not navigate and must not sign out.

*Dependencies:* none on the frontend. The smoke-web check the idea asks for (`GET /login?code=x&state=y` → 200 without `-L`) belongs in `deploy/smoke-web.sh`, which exists only on branch `harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook-` (not on `origin/main` — `git ls-tree origin/main deploy` is empty). The executor bases its worktree on `origin/main`, so that step is conditional in the plan: do it only if `deploy/smoke-web.sh` exists on the executor's base; otherwise it is recorded as a follow-up for the deploy branch's daily PR.

*Folded in:* `harness/ideas/_inbox/cloudflare-pages-serves-404-for-deep-links-and-no-cache-head.md` — same hosting mismatch (Pages does not apply the Caddy rules), fixed by two files in `frontend/public/` that Nuxt copies verbatim into `.output/public` and the Caddy image ignores. One plan, one branch, one review.

*Priority rationale:* `high` — no user can sign in on the production host; the fix is two small edits plus two static files, finishable well inside two hours, and CI-provable (`npm run lint && npm run typecheck && npm run test:unit && npm run build`, plus `npx nuxi generate` output inspection).
