---
plan: harness/plans/2026-09-25-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/task-4-follow-up-smoke-web-has-no-login-return-trip-check-an.md, harness/ideas/_inbox/frontend-dist-symlink-left-by-nuxi-generate-is-not-gitignore.md, harness/ideas/_inbox/executor-worktree-nested-under-claude-worktrees-while-plan-f.md]
---
# Review — Nobody can sign in on Cloudflare Pages: /login is 308-redirected to /login/ and the auth middleware drops Google's code

**Plan:** `harness/plans/2026-09-25-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect.md`
**Branch/worktree:** `harness/2026-09-25-high-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect` / frontmatter says `.worktrees/nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect` (absent); the executor's is nested under `.claude/worktrees/harness-daily-execute-154026/.worktrees/…`. Reviewed in a fresh detached worktree of `origin/<branch>` @ `3b51a45` in the scratchpad (removed after).
**Diff:** `git diff main...harness/2026-09-25-high-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect --stat`

## Plan vs idea
**Head idea (sign-in) — delivered.** Both halves the idea asked for are in: flat generate output (`nitro.prerender.autoSubfolderIndex: false`) and a trailing-slash-normalised `/login` check in the route guard, pinned by a unit test. Read-only check of the live host (owner deployed this build): `GET /login?code=x&state=y` → `200` (was `308`); `GET /login/?code=x&state=y` → `308 → /login?code=x&state=y` (query preserved, as the plan's notes predicted). The idea's smoke-web check was not delivered because the plan's Task 4 was legitimately skipped (`deploy/` still not on `origin/main`) — filed as a tracked follow-up so it is not lost.

**Folded idea (`cloudflare-pages-serves-404-for-deep-links-and-no-cache-head.md`) — half delivered.** `_headers` works live (`/sw.js` → `cache-control: no-cache`). The `_redirects` SPA rewrite does not: live `GET /learn/abc` → `404` because Pages serves the generated `404.html` first. The plan anticipated exactly this in *Notes* and named the fallback; the owner already filed it on `main` as `harness/ideas/_inbox/pages-ignores-the-redirects-spa-rewrite-while-404-html-exist.md` (medium), so it is not re-filed here. Not a blocker: merging this branch fixes a total sign-in outage and makes deep links no worse than today.

## Code vs plan
- Task 1 (guard + 3 tests) — followed verbatim (`normalisePath`, line 15; test block identical to the plan).
- Task 2 (`autoSubfolderIndex: false`) — followed verbatim (`nuxt.config.ts:11`, with the comment).
- Task 3 (`_redirects`, `_headers`) — followed verbatim. The executor could not diff against `frontend/Caddyfile` (not on its base); I diffed against the deploy branch's `frontend/Caddyfile`: `/_nuxt/*` `public, max-age=31536000, immutable` and `/sw.js` `/manifest.webmanifest` `no-cache` match exactly.
- Task 4 — skipped per its precondition (correct; nothing cherry-picked). Follow-up filed.
- Task 5 (CODEMAP) — followed; the only `harness/` file on the branch. Accurate, except that it cites `frontend/Caddyfile`, which exists only once the deploy branch merges.
- Branch merges cleanly onto current `origin/main` (`git merge-tree` clean; `main` has no `frontend/` changes since the base).

Verification, re-run in the fresh worktree (`npm ci` from clean):
```
$ npm run lint                      -> eslint . (exit 0)
$ npm run typecheck                 -> exit 0, 0 error lines
$ npm run test:unit                 -> Test Files 16 passed (16); Tests 81 passed (81)
$ npx vitest run tests/unit/authMiddleware.test.ts --reporter=verbose
  ✓ … trailing slash from a static host … treats /login/?code=…&state=… like /login …
  ✓ … sends a signed-in user on /login/ home, exactly as on /login
  ✓ … still guards a protected route written with a trailing slash
  Tests 7 passed (7)
$ grep -n 'normalisePath(to.path)' middleware/auth.global.ts   -> 15:  if (normalisePath(to.path) === '/login') {
$ grep -n 'autoSubfolderIndex: false' nuxt.config.ts            -> 11:  nitro: { prerender: { autoSubfolderIndex: false } },
$ npm run build >/dev/null 2>&1 && echo build-ok                -> build-ok
$ rm -rf .output && npx nuxi generate; find .output/public -maxdepth 2 -name '*.html' | sort
  200.html 404.html index.html login.html onboarding.html revive.html roadmap.html settings.html
$ test ! -d .output/public/login && echo no-login-dir           -> no-login-dir
$ diff public/_redirects … && diff public/_headers … && echo copied -> copied
$ grep -c '_redirects\|_headers' .output/public/sw.js           -> 0
$ git status --short | grep -c '^?? .output'                    -> 0   (but `?? dist` — see bugs)
$ git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP -> only the " 1 file changed" summary line (plan's "no output" expectation is slightly off; only CODEMAP changed)
$ python3 tools/harness/cli.py validate                        -> exit 0
$ gh run list --branch <branch> --limit 1                       -> 36111481381 completed success; jobs: harness-tooling, frontend, backend-integration, backend-unit all success
```
Runtime proof re-run: `python3 -m http.server 18391 --directory .output/public` → `GET /login.html` `200`, `<title>Học 30 phút`; server killed. Live read-only (`https://english-learning-e6a.pages.dev`): `/login?code=x&state=y` 200, `/login/?code=x&state=y` 308 → `/login?code=x&state=y`, `/roadmap` 200, `/learn/abc` 404, `/sw.js` `cache-control: no-cache`. Everything the executor claimed reproduced — no gate failure.

## Quality
- **Test honesty — good.** I reverted `middleware/auth.global.ts` to `origin/main` and removed the `navigateTo` assertion from the first new test: it still fails on `expect(await caches.has(API_STATE_CACHE)).toBe(true)`, so the "signOut() was not called" check is independently meaningful, not decorative. Restored afterwards (`git status` clean apart from build output).
- **Correctness.** `normalisePath` handles repeated slashes and `/` → `/`; the protected branch still redirects `/roadmap/` (tested). The guard change is minimal and does not widen what counts as `/login`. Caddy image behaviour is unchanged (`try_files {path} /index.html`; `/login` is not a file, falls to the shell).
- **Boundaries/conventions.** Frontend-only, matches existing guard/test style and comment density; no new deps; the `as unknown as` cast is confined to the test as the plan specified.
- **Docs.** CODEMAP accurate modulo the `frontend/Caddyfile` reference (arrives with the deploy branch). `frontend/public/_redirects`' comment claims the rewrite "restores" SPA mode — live evidence shows it does not while `404.html` exists; that comment should be corrected by whoever fixes the existing deep-link bug.
- **Process.** Execution summary says the frontmatter `worktree` reflects the nested path; it does not (filed, low).

## Bugs filed
- `harness/ideas/_inbox/task-4-follow-up-smoke-web-has-no-login-return-trip-check-an.md` — medium — smoke-web login check + runbook correction still pending (Task 4 skipped; deploy branch unmerged).
- `harness/ideas/_inbox/frontend-dist-symlink-left-by-nuxi-generate-is-not-gitignore.md` — low — `nuxi generate` leaves an unignored `frontend/dist` symlink (pre-existing).
- `harness/ideas/_inbox/executor-worktree-nested-under-claude-worktrees-while-plan-f.md` — low — frontmatter worktree path is wrong; nested worktree escapes prune.
- Not re-filed (already on `main`, confirmed live): `harness/ideas/_inbox/pages-ignores-the-redirects-spa-rewrite-while-404-html-exist.md` — deep links still 404 on Pages.

No blockers.

## Verdict
**pass-with-bugs.** The sign-in outage fix is correct, tested honestly, CI-green, and confirmed on the live host; the folded deep-link half is tracked by an existing inbox bug.
