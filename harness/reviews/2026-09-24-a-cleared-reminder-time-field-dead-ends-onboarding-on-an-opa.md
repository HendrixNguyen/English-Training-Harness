---
plan: harness/plans/2026-09-24-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/onboarding-s-invalid-request-copy-asks-the-learner-to-fix-fi.md, harness/ideas/_inbox/a-rejected-cachestorage-delete-now-fails-sign-in-for-a-user-.md, harness/ideas/_inbox/no-test-pins-that-login-awaits-signin-s-cache-clear-before-n.md]
---
# Review — frontend: close the 2026-09-23 review follow-ups on `/onboarding`, sign-in and `/revive`

**Plan:** `harness/plans/2026-09-24-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa.md`
**Branch/worktree:** `harness/2026-09-24-medium-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa` / `.worktrees/a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa`
**Diff:** `git diff origin/main...harness/2026-09-24-medium-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa --stat` — 10 files, +173 / −9 (below the 200-line threshold for a separate test-gap agent). Base is freshly fetched `origin/main`; branch head `60227f0` equals `origin/<branch>`.

## Plan vs idea
All six ideas are delivered.
- **Head (cleared reminder time):** the start button is gated on a goal and an `HH:MM` time, with a hint under the field, and `invalid_request` has its own copy. The idea also asked for a test that "clears the time field, completes the quiz and asserts the learner is told what to fix". The gate makes that path impossible, so the plan tests the gate itself (button disabled, no quiz `GET`, hint shown) and tests `invalid_request` separately. That substitution is justified. One gap remains: the new copy tells the learner to fix fields that the quiz step doesn't show (bug 1).
- **`ai_*` copy untested:** `it.each` covers all three `ai_*` codes the handler returns, not only the two the idea named.
- **Stale `api-state` into the next account:** the idea offered two alternatives and the branch does both: `signIn()` clears the cache, and the guard drops an expired session on `/login`. Both are tested against the Map-backed fake, and `assets` survives.
- **authStore `caches` stub:** the per-file `afterEach(vi.unstubAllGlobals)`, plus a trailing sentinel test. Design decision 4 explains why the suite-wide `unstubGlobals: true` option was rejected, and the reasoning is sound: it would revert the top-level `navigateTo` and middleware stubs.
- **"bỏ học2 ngày":** the sentence is now computed in script. A rendered-text test covers both the 3-day case and the `null` case.
- **Stale data wins on `/revive`:** covers both the idea's fourth case and its optional fifth.

## Code vs plan
Every task was followed exactly: code, copy and test bodies match the plan text line for line.
- **Task 6 (CODEMAP):** one deviation. The executor replaced the old "`rate_limited` / `ai_*` get their own copy" clause instead of adding a second clause next to it. That is justified, and the result is accurate.
- **Commits:** 6, one per task, each with the plan's message and a `Co-Authored-By` trailer.

Re-run in the worktree's `frontend/` (condensed):
```
npm ci                         # ok
npm run lint                   # eslint . — clean
npm run typecheck              # nuxi typecheck — clean
npm run test                   # "Missing script: test" — the suite script is test:unit (as the plan and CI use); not a branch defect
npm run test:unit              # Test Files 16 passed (16) · Tests 78 passed (78)
                               #   onboardingPage 8 · authStore 9 · authMiddleware 4 · revivePage 6
npx vitest run <the 4 files>   # 4 files, 27 tests passed
npm run build                  # ✨ Build complete! (client + nitro + injectManifest SW)
grep canStart|invalid_request pages/onboarding.vue   # 16, 29, 46, 103
grep 'return clearApiCache()' stores/auth.ts         # 73 (signIn), 81 (signOut)
grep 'await auth.signIn' pages/login.vue             # 35
grep unstubAllGlobals tests/unit/authStore.test.ts   # 19
grep 'bỏ học<template' pages/revive.vue              # no output
git log origin/main..HEAD                            # 6 commits, all with Co-Authored-By
cli.py validate                                      # exit 0
```
Runtime proof, reproduced on the built artifact (`node .output/server/index.mjs`, `PORT=13002`, no backend):
- **Serving:** `curl` returns `200` and `<title>Học 30 phút</title>` for `/`, `/onboarding`, `/login` and `/revive`.
- **`/onboarding`, in the in-app browser with a seeded valid session:** after selecting IELTS 7.0 with the default `20:00`, the button reads `disabled=false` and there is no note. After clearing the time input, it reads `time=""`, `aria-invalid="true"`, `disabled=true`, and the note says "Chọn một giờ nhắc học để tiếp tục.". After setting `07:30`, it reads `disabled=false` and the note is gone.
- **`/login` with a seeded *expired* session:** the page stays on `/login` and `localStorage['aelp.auth']` is `null`. The new guard branch works live.
- **`/revive`:** renders the load-error StateBlock ("Không tải được trạng thái cây…") with a retry. That is expected with no backend. The wilted sentence is covered by the unit test, not live.
- **Cleanup:** server stopped (port 13002 free), seeded `localStorage` cleared. No containers were started.

CI: `gh run list --branch <branch>` → run 35959491527 on head `60227f0`, **success**: frontend, backend-unit, harness-tooling and backend-integration are all green.

Mutations: I did not re-run the plan's mutation table, because this session is not allowed to change app code even temporarily. Reading the tests statically, every row's named test does exercise the mutated line. One exception is `authMiddleware` "leaves /login alone when there is no session": it relies on `FakeCacheStorage.delete` removing the entry synchronously inside its async body, which holds for this fake.

## Quality
- **Design and boundaries:** `signIn()` owning the clear, symmetrical with `signOut()`, is the right place: callers still never touch the cache. The guard change is minimal and keeps `/login` navigation-free. There is no cross-layer reach.
- **Error handling (bug 2):** `signIn()` now couples a successful sign-in to `caches.delete` resolving. `clearApiCache()` does not catch, and `login.vue` awaits it inside the same `try` as the sign-in `POST`. A rejecting CacheStorage therefore shows "Không đăng nhập được" to a user who is in fact signed in. This is low priority, and it is my inference; I did not reproduce it in a named browser.
- **Test honesty:** the new tests assert state (`has()`, rendered text, `disabled`), not call counts, which matches the `fakeCaches.ts` convention. `vi.setSystemTime` without fake timers is restored by `vi.useRealTimers()` in `finally`; vitest 3.2.7 calls `resetDate()` there, so nothing leaks. Two soft spots:
  - The trailing `'caches' in globalThis` sentinel will go red spuriously if a future happy-dom ships CacheStorage natively. The plan documents that case.
  - Nothing pins `login.vue`'s `await` (bug 3), which is what Design decision 3's ordering relies on.
- **Correctness on untried inputs:** `TIME_RE` accepts `99:99`. That value is unreachable from a supporting `<input type="time">`, and the backend rejects it as `invalid_request` anyway. Only a browser that falls back to a text input could produce it, so I did not file it.
- **Accessibility:** the hint uses `role="note"`, which is not a live region, so clearing the field is not announced. The hint is not wired through `aria-describedby`. Because the hint sits inside the `<label>`, the input's accessible name becomes label text plus hint text: it works, but it is clumsy. Not filed; worth folding into a future a11y pass.
- **Idiom:** the explicit `Promise<void>` return and the JSDoc on `signIn` are good. `goal.value!` (`onboarding.vue:68`, pre-existing) is now provably safe through `canStart`.
- **Performance:** no concerns. The only added work is one regex per keystroke and one `caches.delete` per sign-in.
- **CODEMAP:** the `shell` paragraph accurately records the gate, the per-code copy, the `signIn()` clear and the `/login` guard branch. No correction needed.

## Bugs filed
- `harness/ideas/_inbox/onboarding-s-invalid-request-copy-asks-the-learner-to-fix-fi.md` (low): the `invalid_request` copy names goal and reminder time, but the quiz step has no way back to them, so retry still loops.
- `harness/ideas/_inbox/a-rejected-cachestorage-delete-now-fails-sign-in-for-a-user-.md` (low): `clearApiCache()` is not best-effort, so a rejected `caches.delete` breaks `/login` for a user who is signed in.
- `harness/ideas/_inbox/no-test-pins-that-login-awaits-signin-s-cache-clear-before-n.md` (low): no test mounts `pages/login.vue`, so dropping the `await` on line 35 stays green.

No blockers.

## Verdict
**pass-with-bugs.** All six ideas are delivered as planned. The executor's evidence reproduces: lint, typecheck, 78/78 unit tests and the build pass; the gate, the hint and the expired-session drop work live on the built app; CI is green on head. The three bugs filed are low-priority follow-ups, none of which holds up this branch.
