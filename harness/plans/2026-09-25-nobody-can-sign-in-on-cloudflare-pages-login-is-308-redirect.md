---
idea: harness/ideas/_inbox/nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect.md
status: done
priority: high
merged: false
branch: harness/2026-09-25-high-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect
worktree: .worktrees/nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect
pr: "https://github.com/HendrixNguyen/English-Training-Harness/pull/30"
---
# Nobody can sign in on Cloudflare Pages: /login is 308-redirected to /login/ and the auth middleware drops Google's code — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea (head):** `harness/ideas/_inbox/nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect.md`
**Also planned here (its frontmatter points at this plan):**
- `harness/ideas/_inbox/cloudflare-pages-serves-404-for-deep-links-and-no-cache-head.md` → Task 3 (and Task 4 if `deploy/` is present)

**Goal:** A Google sign-in on the Cloudflare Pages host completes — `GET /login?code=…&state=…` is served with a 200 and the query intact, the middleware never ejects a returning user because of a trailing slash — and Pages serves deep links with a 200 and the same cache headers as the Caddy image.

**Why now (`priority: high`):** On the live host (`https://english-learning-e6a.pages.dev`, owner, 2026-09-25) every sign-in dies: `curl -sI '/login?code=x&state=y'` → `308 location: /login/?code=x&state=y`, and the API log shows no `POST /api/v1/auth/google` ever. Sign-in is the only door into the product (spec §5.1). Verified read-only in the frontend source: `nuxt.config.ts` has no `nitro.prerender` block, so `nuxi generate` (Nitro 2.13.4, default `autoSubfolderIndex: true`) writes `login/index.html` and Pages canonicalises the directory to `/login/`; `middleware/auth.global.ts:7` checks `to.path === '/login'` exactly, so `/login/` falls into the protected branch at lines 15–18 — `auth.signOut()` then `navigateTo('/login', { replace: true })`, a bare path that discards `code`/`state`, and `pages/login.vue` (`const { code, state } = route.query`) never calls `finishSignIn`. The Caddy image (`frontend/Caddyfile`: `try_files {path} /index.html`) never redirects, which is why compose, Target B and CI's `docker-images` job all pass. The folded idea is the same mismatch: `frontend/public/` has no `_redirects`/`_headers`, so Pages answers deep links from `404.html` with a 404 status and serves every file `max-age=0, must-revalidate` (its smoke run fails 4/7).

**Architecture:** Two-line config change that makes `nuxi generate` emit flat `login.html`/`onboarding.html`/… (Pages then serves `/login` directly and redirects `/login/` → `/login`, the way it already redirects `/contact.html` → `/contact`); a normalised-path comparison in the route guard so no static host's trailing slash can ever route a returning user into the sign-out branch (defence in depth, pinned by a unit test); two static files in `frontend/public/` that Nuxt copies verbatim into `.output/public` and that only Pages reads (`_redirects`, `_headers`, both without extension, so the service worker's `globPatterns` never precaches them and the Caddy image serves them as inert text). A smoke-check line and a runbook correction are conditional on `deploy/` existing on the executor's base.

**Tech stack:** Nuxt 3.21 (`ssr: false`), Nitro 2.13 (`prerender.autoSubfolderIndex`), Vue Router, Pinia, Vitest + happy-dom (78 unit tests today, `tests/unit/authMiddleware.test.ts` stubs `defineNuxtRouteMiddleware`/`navigateTo`), Cloudflare Pages `_redirects`/`_headers`. No new dependencies.

**Base branch:** the executor bases its worktree on freshly fetched `origin/main` (harness-execute skill, step 5). **`deploy/` (`deploy/smoke-web.sh`, `deploy/README.md`) exists only on `harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook-` today** (`git ls-tree origin/main deploy` is empty); the daily PR carrying it is expected to merge tonight. Task 4 therefore begins with an existence check and is **skipped, not improvised**, when `deploy/smoke-web.sh` is absent — never merge or cherry-pick that branch to get it. Run every command from the worktree root unless a step says `frontend/`; `rg` is not installed — use `grep -n`.

---

## File structure

| Path | Change |
| --- | --- |
| `frontend/nuxt.config.ts` | add `nitro: { prerender: { autoSubfolderIndex: false } }` with a comment naming the Pages 308 |
| `frontend/middleware/auth.global.ts` | `normalisePath()` helper; the `/login` branch compares the normalised path |
| `frontend/tests/unit/authMiddleware.test.ts` | new `describe` block: `/login/` with `code`/`state` (no redirect, no sign-out), signed-in on `/login/` (→ `/`), protected `/roadmap/` (→ `/login`) |
| `frontend/public/_redirects` | new — `/*  /index.html  200` |
| `frontend/public/_headers` | new — `/_nuxt/*` immutable; `/sw.js`, `/manifest.webmanifest` `no-cache` (values copied from `frontend/Caddyfile`) |
| `deploy/smoke-web.sh` | **only if present** — one `check` for `GET /login?code=x&state=y` → `200` without `-L` |
| `deploy/README.md` | **only if present** — Target A paragraph: `_redirects`/`_headers` are required; `autoSubfolderIndex` note |
| `harness/CODEMAP.md` | frontend **shell** bullet: flat generate output, normalised route guard, the two Pages files |

---

## Tasks

### Task 1: Route guard compares a normalised path (test first)

**Files:**
- Modify: `frontend/middleware/auth.global.ts:7`
- Test: `frontend/tests/unit/authMiddleware.test.ts`

- [ ] **Step 1: Baseline** — in `frontend/`: `npm ci && npm run test:unit 2>&1 | tail -4`. Expected: `Tests  78 passed`. Record the number for the summary.
- [ ] **Step 2: Write the failing tests.** Append to `frontend/tests/unit/authMiddleware.test.ts` (same file — it already owns the `navigateTo` stub, `persistSession` and `installSeededCaches`):

```ts
describe('middleware/auth.global — trailing slash from a static host (Pages 308 → /login/, fix 2026-09-25)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    navigateTo.mockReset()
  })

  it('treats /login/?code=…&state=… like /login: no redirect, no sign-out, so the Google code survives', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    const login = { path: '/login/', query: { code: 'c', state: 's' } } as unknown as RouteLocationNormalized

    guard(login, login)
    await Promise.resolve()

    expect(navigateTo).not.toHaveBeenCalled()
    expect(await caches.has(API_STATE_CACHE)).toBe(true) // signOut() was not called
  })

  it('sends a signed-in user on /login/ home, exactly as on /login', () => {
    persistSession(Date.now() + 60_000)
    const login = { path: '/login/', query: {} } as RouteLocationNormalized

    guard(login, login)

    expect(navigateTo).toHaveBeenCalledWith('/', { replace: true })
  })

  it('still guards a protected route written with a trailing slash', () => {
    const roadmap = { path: '/roadmap/', query: {} } as RouteLocationNormalized

    guard(roadmap, roadmap)

    expect(navigateTo).toHaveBeenCalledWith('/login', { replace: true })
  })
})
```

- [ ] **Step 3: Run them and watch the first two fail.** `npx vitest run tests/unit/authMiddleware.test.ts`. Expected: the two `/login/` tests FAIL (first: `navigateTo` called with `'/login'`; second: `navigateTo` not called); the `/roadmap/` test and the four existing tests pass. If the first test passes before the fix, the test is not exercising the bug — stop and fix the test.
- [ ] **Step 4: Minimal implementation.** Replace the top of `frontend/middleware/auth.global.ts` so the whole file reads:

```ts
import { useAuthStore } from '~/stores/auth'

// Static hosts may canonicalise a route to its trailing-slash form — Cloudflare
// Pages answered `/login?code=…` with `308 /login/?code=…` (2026-09-25) — and
// `/login/` must still be the login page, or a returning Google user lands in
// the protected branch below and loses the code.
function normalisePath(path: string): string {
  return path.replace(/\/+$/, '') || '/'
}

export default defineNuxtRouteMiddleware((to) => {
  const auth = useAuthStore()
  if (!auth.hydrated) auth.hydrate()

  if (normalisePath(to.path) === '/login') {
    // A signed-in user has no business on /login unless Google just sent them back.
    if (auth.isAuthenticated && !to.query.code) return navigateTo('/', { replace: true })
    // A session that is present but expired is dropped here too — with its
    // api-state cache — so the next account never inherits it (fix 2026-09-24).
    if (auth.accessToken !== null && !auth.isAuthenticated) void auth.signOut()
    return
  }
  if (!auth.isAuthenticated) {
    void auth.signOut()
    return navigateTo('/login', { replace: true })
  }
})
```

The protected-branch `navigateTo('/login', …)` is unchanged on purpose: it is reached only when there is no valid session, and a Google return trip never arrives there once `/login/` is recognised.

- [ ] **Step 5: Green.** `npx vitest run tests/unit/authMiddleware.test.ts` → 7 passed. Then `npm run test:unit 2>&1 | tail -4` → `Tests  81 passed`.
- [ ] **Step 6: Lint + types.** `npm run lint && npm run typecheck` → both clean (the `as unknown as RouteLocationNormalized` cast is needed because `query` is `LocationQuery`; keep it, do not widen the type elsewhere).
- [ ] **Step 7: Commit.** `git add frontend/middleware/auth.global.ts frontend/tests/unit/authMiddleware.test.ts && git commit -m "fix(frontend): auth guard treats /login/ as /login so a static host's trailing slash cannot drop Google's code"`.

### Task 2: `nuxi generate` emits flat `login.html` (no directory index, no Pages 308)

**Files:**
- Modify: `frontend/nuxt.config.ts` (insert after the `devtools` line)

- [ ] **Step 1: Reproduce the shape.** In `frontend/`: `npx nuxi generate >/dev/null 2>&1; find .output/public -maxdepth 2 -name '*.html' | sort`. Expected today: `./index.html`, `./200.html`, `./404.html`, and `login/index.html`, `onboarding/index.html`, `revive/index.html`, `roadmap/index.html`, `settings/index.html`. Record it.
- [ ] **Step 2: Add the Nitro option.** In `frontend/nuxt.config.ts`, after `devtools: { enabled: false },` insert:

```ts
  // Emit `login.html`, not `login/index.html`: a directory index makes
  // Cloudflare Pages 308 `/login?code=…` to `/login/?code=…`, which broke every
  // Google sign-in on the live host (2026-09-25). Pages serves `/login` from
  // the flat file with no redirect; the Caddy image's try_files is unaffected.
  nitro: { prerender: { autoSubfolderIndex: false } },
```

- [ ] **Step 3: Verify the output.** `rm -rf .output && npx nuxi generate >/dev/null 2>&1; find .output/public -maxdepth 2 -name '*.html' | sort`. Expected: `index.html`, `200.html`, `404.html`, `login.html`, `onboarding.html`, `revive.html`, `roadmap.html`, `settings.html` — and `test ! -d .output/public/login && echo no-login-dir` prints `no-login-dir`. `.output/` is untracked (`git status --short` shows nothing under it); if it does appear, do **not** add it.
- [ ] **Step 4: Nothing else moved.** `npm run build >/dev/null 2>&1 && echo build-ok` → `build-ok` (CI runs `build`, not `generate`; both must stay green). `npm run typecheck` → clean (the option is typed in `nitropack`; if `typecheck` rejects the key, the installed Nitro is older than 2.13 — stop and report, do not cast).
- [ ] **Step 5: Commit.** `git add frontend/nuxt.config.ts && git commit -m "fix(frontend): generate flat login.html so Cloudflare Pages serves /login without a 308"`.

### Task 3: `_redirects` and `_headers` for Cloudflare Pages

**Files:**
- Create: `frontend/public/_redirects`
- Create: `frontend/public/_headers`

- [ ] **Step 1: `_redirects`** — create `frontend/public/_redirects` with exactly this content (SPA fallback with a 200; Pages would otherwise answer unknown routes from the generated `404.html` with a 404 status):

```
# Cloudflare Pages only (the Caddy image has its own try_files rule).
# `nuxi generate` emits 404.html, which turns off Pages' implicit SPA mode;
# this rewrite restores it: every unmatched route is the app shell, status 200.
/*  /index.html  200
```

- [ ] **Step 2: `_headers`** — create `frontend/public/_headers`; the three values are copied verbatim from `frontend/Caddyfile` so both hosts stay identical (frontend spec §3: the worker and manifest must never be served stale; hashed assets are safe forever):

```
# Cloudflare Pages only — mirrors the Caddyfile so both targets behave the same.
/_nuxt/*
  Cache-Control: public, max-age=31536000, immutable

/sw.js
  Cache-Control: no-cache

/manifest.webmanifest
  Cache-Control: no-cache
```

- [ ] **Step 3: They reach the build output unchanged.** In `frontend/`: `rm -rf .output && npx nuxi generate >/dev/null 2>&1; diff public/_redirects .output/public/_redirects && diff public/_headers .output/public/_headers && echo copied`. Expected: `copied`. Then `grep -c '_redirects\|_headers' .output/public/sw.js` → `0` (no extension, so `injectManifest.globPatterns` never precaches them).
- [ ] **Step 4: Lint** — `npm run lint` → clean (the files are not linted, but confirm nothing else regressed).
- [ ] **Step 5: Commit.** `git add frontend/public/_redirects frontend/public/_headers && git commit -m "fix(frontend): Pages _redirects (SPA 200) and _headers (immutable assets, no-cache sw/manifest) to match the Caddy image"`.

### Task 4 (conditional): smoke-web login check and runbook correction

**Precondition:** `test -f deploy/smoke-web.sh` at the worktree root. If it is **absent** (the deploy branch has not merged into `origin/main` yet), skip this task entirely, write `Task 4 skipped: deploy/ not on base` in the execution summary, and stop here — do **not** merge, cherry-pick or copy files from `harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook-`. The evaluator will re-file the two edits below as a follow-up inbox item if they are still missing after that branch merges.

**Files:**
- Modify: `deploy/smoke-web.sh` (after the `spa fallback` check line)
- Modify: `deploy/README.md` (the **PWA on Cloudflare Pages** paragraph)

- [ ] **Step 1: The check.** In `deploy/smoke-web.sh`, directly after `check "spa fallback (/learn/abc)" "200" "$(status "$web/learn/abc")"`, add:

```sh
# Google's return trip: /login with the code and state must be served as-is —
# a 308 to /login/ (directory-index canonicalisation) lost every sign-in on
# Pages (2026-09-25). `status` never follows redirects, so a 3xx fails here.
check "login return trip (/login?code&state)" "200" "$(status "$web/login?code=x&state=y")"
```

`status()` calls `curl` without `-L`, so a 308 is reported as `308` and fails the check — that is the point.

- [ ] **Step 2: The runbook.** In `deploy/README.md`, in the *PWA on Cloudflare Pages* paragraph, replace the sentence that begins `Pages serves the SPA fallback and hashed-asset caching by itself` with:

> Pages does **not** apply the `Caddyfile` rules: `frontend/public/_redirects` (`/* /index.html 200`) gives deep links a 200 and `frontend/public/_headers` gives `/_nuxt/*` the immutable cache and `sw.js`/`manifest.webmanifest` `no-cache`; both are copied into `.output/public` by `nuxi generate`. `nuxt.config.ts` sets `nitro.prerender.autoSubfolderIndex: false` so the build emits `login.html` rather than `login/index.html` — a directory index made Pages 308 `/login?code=…` to `/login/?code=…` and lost every Google sign-in.

Also extend the *Smoke check* bullet for `smoke-web.sh` with `; \`/login?code=x&state=y\` is \`200\` with no redirect`.

- [ ] **Step 3: Prove the script still parses and the check is reachable.** `sh -n deploy/smoke-web.sh && echo parses`; then against the local Caddy image (Target B semantics, no cloud): in `frontend/`, `docker build -t aelp-web:login -f Dockerfile --build-arg NUXT_PUBLIC_API_BASE=https://api.example.test . && docker run -d --rm --name web-login -p 18082:80 aelp-web:login`, wait for `curl -sf --max-time 2 http://127.0.0.1:18082/healthz`, then `sh deploy/smoke-web.sh http://127.0.0.1:18082`. Expected: 8/8 `ok` including `ok   login return trip (/login?code&state): 200`. Then `docker rm -f web-login`.
- [ ] **Step 4: Commit.** `git add deploy/smoke-web.sh deploy/README.md && git commit -m "deploy: smoke-web checks the /login return trip; runbook says Pages needs _redirects/_headers"`.

### Task 5: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` — the frontend **shell** bullet

- [ ] **Step 1:** In the **shell** bullet, after the sentence that names `Route guard \`middleware/auth.global.ts\``, add: `(compares a trailing-slash-normalised path, so \`/login/\` from a static host is still the login page — Cloudflare Pages 308ed \`/login?code=…\` to \`/login/\` and every sign-in died, fix 2026-09-25; \`nuxt.config.ts\` sets \`nitro.prerender.autoSubfolderIndex: false\` so \`nuxi generate\` emits flat \`login.html\` and Pages never redirects; \`public/_redirects\` (SPA 200) and \`public/_headers\` (immutable \`/_nuxt/*\`, \`no-cache\` worker/manifest) are read by Pages only and mirror \`frontend/Caddyfile\`)`.
- [ ] **Step 2:** `python3 tools/harness/cli.py validate` → exit 0. Commit: `git add harness/CODEMAP.md && git commit -m "harness: CODEMAP — flat generate output, normalised route guard, Pages _redirects/_headers"`. This is the only `harness/` file the branch may change.

## Verification

```bash
cd frontend
npm run lint && npm run typecheck && npm run test:unit 2>&1 | tail -4
# expect: lint clean, typecheck clean, Tests  81 passed (78 + 3 in authMiddleware.test.ts)
npx vitest run tests/unit/authMiddleware.test.ts 2>&1 | grep -E 'trailing slash|passed'
# expect: the new describe block listed, 7 passed
grep -n 'normalisePath(to.path)' middleware/auth.global.ts
# expect: line 15 — the /login branch
grep -n 'autoSubfolderIndex: false' nuxt.config.ts
# expect: one line
npm run build >/dev/null 2>&1 && echo build-ok
# expect: build-ok (the CI frontend job's last step)
rm -rf .output && npx nuxi generate >/dev/null 2>&1
find .output/public -maxdepth 2 -name '*.html' | sort
# expect: 200.html 404.html index.html login.html onboarding.html revive.html roadmap.html settings.html — no */index.html
test ! -d .output/public/login && echo no-login-dir
# expect: no-login-dir
diff public/_redirects .output/public/_redirects && diff public/_headers .output/public/_headers && echo copied
# expect: copied
grep -c '_redirects\|_headers' .output/public/sw.js
# expect: 0
git status --short | grep -c '^?? .output' ; true
# expect: 0 (never committed)
cd ..
if [ -f deploy/smoke-web.sh ]; then grep -n 'login return trip' deploy/smoke-web.sh && sh -n deploy/smoke-web.sh && echo parses; else echo 'Task 4 skipped: deploy/ not on base'; fi
git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP
# expect: no output
python3 tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0
gh run list --branch "$(git branch --show-current)" --limit 1
# expect: every job green; the frontend job's log shows 81 unit tests
```

The proof on the live host is the owner's (no cloud accounts for agents): after the daily PR merges and Pages rebuilds, `curl -sI 'https://english-learning-e6a.pages.dev/login?code=x&state=y' | head -1` → `HTTP/2 200`, and `sh deploy/smoke-web.sh https://english-learning-e6a.pages.dev` → all `ok`.

## Notes and open questions

- **Why both halves?** (a) alone fixes Pages today; (b) alone would let the app survive the 308 but with `/login/` in the address bar and a second redirect hop. Together, no static host's slash convention can strand a returning user, and the unit test pins it.
- **`/login/` after the fix.** Pages redirects `/login/` → `/login` (the same canonicalisation it applies to `/contact.html` → `/contact`); the query survives that hop, and the guard now accepts either spelling anyway. Vue Router's default `strict: false` already matches `/login/` to the `login` page.
- **`_redirects` `/*` and real files.** The Pages docs do not spell out asset-vs-rewrite precedence; the `/* /index.html 200` rewrite is Cloudflare's documented SPA idiom and is what the folded idea's owner asked for. If the owner's live smoke shows a real file (`/login`, a `/_nuxt/*.js`) being answered with `index.html`, the fallback is to drop `_redirects` and instead delete `404.html` from the generate output (Pages then falls into its implicit SPA mode) — a follow-up inbox item, not a change to this plan.
- **Caddy image unchanged.** `try_files {path} /index.html` serves `login.html` for `/login` only because Caddy's `try_files` tries `{path}` literally — `/login` is not a file, so it falls to `index.html`, which is the SPA shell and renders the login page as before. Behaviour is identical to today; `_redirects`/`_headers` are served as plain files if requested and nothing links to them.
- **Out of scope:** the e2e `tests/e2e/login.spec.ts` (Playwright, local-only), Google Console settings (owner), any change to `pages/login.vue`'s `redirectUri` (it is already `origin + '/login'`, which is the correct flat path).

## Execution summary

Built in `.worktrees/nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect` on branch `harness/2026-09-25-high-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect`, based on freshly fetched `origin/main` (`959cb5f`). All tasks executed task-by-task with tests first, one commit per task:

- `5a9dd2b` — Task 1: `normalisePath()` in `middleware/auth.global.ts`; 3 new tests in `authMiddleware.test.ts`.
- `b20bf3f` — Task 2: `nitro.prerender.autoSubfolderIndex: false` in `nuxt.config.ts`.
- `88e7c4a` — Task 3: `frontend/public/_redirects` and `frontend/public/_headers`.
- `3b51a45` — Task 5: `harness/CODEMAP.md` shell bullet updated (the only `harness/` file touched on the branch).

**Deviations:**
- **Task 4 skipped**, exactly per the plan's conditional: `test -f deploy/smoke-web.sh` was false on this base (`git ls-tree origin/main deploy` is empty — the containerised-deploy branch carrying it has not merged yet). Not improvised, nothing merged or cherry-picked from that branch.
- **Development worktree location**: the orchestrator's instruction specified `WT=/Users/hendrixnguyen/Workspaces/self/Learning-English-Project/.worktrees/$SLUG` (a sibling of the main checkout). This session's sandbox only permits edits inside its own worktree subtree (`.claude/worktrees/harness-daily-execute-154026/…`), so the executor worktree was created nested at `.claude/worktrees/harness-daily-execute-154026/.worktrees/$SLUG` instead — still branched from the same freshly fetched `origin/main` commit (`959cb5f`), matching what other concurrent sessions in this repo already do. The plan's `worktree` frontmatter reflects this nested path.
- **`frontend/Caddyfile` not present on this base** (same non-merge as `deploy/`), so the `_redirects`/`_headers` values could not be independently diffed against it; they were written verbatim from the plan's own Step 1/Step 2 content blocks, which the plan states are copies of the Caddyfile values.

**Plan Verification section — full output:**
```
$ npm run lint && npm run typecheck && npm run test:unit 2>&1 | tail -4
ESLint: No issues found
typecheck: clean (nuxi typecheck)
Test Files  16 passed (16)
     Tests  81 passed (81)

$ npx vitest run tests/unit/authMiddleware.test.ts --reporter=verbose 2>&1 | grep -E 'trailing slash|passed'
✓ … middleware/auth.global — trailing slash from a static host (Pages 308 → /login/, fix 2026-09-25) > treats /login/?code=…&state=… like /login … (x3)
Test Files  1 passed (1)
     Tests  7 passed (7)

$ grep -n 'normalisePath(to.path)' middleware/auth.global.ts
15:  if (normalisePath(to.path) === '/login') {

$ grep -n 'autoSubfolderIndex: false' nuxt.config.ts
11:  nitro: { prerender: { autoSubfolderIndex: false } },

$ npm run build >/dev/null 2>&1 && echo build-ok
build-ok

$ rm -rf .output && npx nuxi generate >/dev/null 2>&1
$ find .output/public -maxdepth 2 -name '*.html' | sort
200.html 404.html index.html login.html onboarding.html revive.html roadmap.html settings.html
(no */index.html)

$ test ! -d .output/public/login && echo no-login-dir
no-login-dir

$ diff public/_redirects .output/public/_redirects && diff public/_headers .output/public/_headers && echo copied
copied

$ grep -c '_redirects\|_headers' .output/public/sw.js
0

$ git status --short | grep -c '^?? .output' ; true
0

$ cd .. ; test -f deploy/smoke-web.sh || echo 'Task 4 skipped: deploy/ not on base'
Task 4 skipped: deploy/ not on base

$ git diff --stat origin/main...HEAD -- harness/
harness/CODEMAP.md | 2 +-
 1 file changed, 1 insertion(+), 1 deletion(-)
(only CODEMAP.md changed)

$ python3 tools/harness/cli.py validate; echo "exit=$?"
exit=0

$ gh run list --branch harness/2026-09-25-high-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect --limit 1
CI · 36111481381 — all 4 jobs green (harness-tooling, frontend, backend-integration, backend-unit)
```

**Runtime proof (harness-execute step 8 / executor role definition-of-done, frontend layer):**
- Build: `npm run build` → `build-ok`, no errors/warnings introduced.
- Whole suite (not just new tests): `npm run test:unit` → 16 files, 81 passed, from a clean `npm ci` install.
- Boot and answer a real path: `rm -rf .output dist && npx nuxi generate`, then `python3 -m http.server 18091 --directory .output/public &`; `curl -s --max-time 5 -o /tmp/login_resp.html -w "HTTP_STATUS:%{http_code}"  http://127.0.0.1:18091/login.html` → `HTTP_STATUS:200`, body is the Nuxt app shell (`<title>Học 30 phút</title>`). `.output/public/login/` does not exist (`no-login-dir-confirmed`); `.output/public/_redirects` and `.output/public/_headers` are present (228 B / 247 B). Server killed afterward; `pgrep -fl "http.server"` confirmed empty — no leftover process.
- Every documented command run as documented: all commands in the plan's Verification block above, run verbatim, in this worktree.
- CI green on the branch: run `36111481381`, https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36111481381 — `harness-tooling`, `frontend`, `backend-integration`, `backend-unit` all passed.
- No backend boot was needed (frontend-only plan); no cloud accounts used. The live-host proof (`curl` against `https://english-learning-e6a.pages.dev`) is explicitly the owner's, per the plan, and was not attempted here.
