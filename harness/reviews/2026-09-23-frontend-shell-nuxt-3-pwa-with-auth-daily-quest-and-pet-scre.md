---
plan: harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/frontend-shell-plan-claims-the-6-1-auth-shape-is-live-on-mai.md, harness/ideas/_inbox/revive-shows-a-false-your-plant-is-dead-alarm-whenever-get-p.md, harness/ideas/_inbox/expired-session-sign-out-leaves-per-user-api-responses-in-th.md, harness/ideas/_inbox/nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment.md, harness/ideas/_inbox/authstore-test-does-not-accept-the-pre-spec-token-shape-pass.md, harness/ideas/_inbox/codemap-ci-section-still-says-three-parallel-jobs-after-the-.md]
---
# Review — Frontend Shell: Nuxt 3 PWA with auth, daily quest and pet screens

**Plan:** `harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md`
**Branch/worktree:** `harness/2026-09-23-high-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre` / `.worktrees/frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre`
**Diff:** `git diff main...harness/2026-09-23-high-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre --stat`

## Plan vs idea

Delivered, and then some. The idea (`harness/ideas/2026-09-22-run-02/frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md`) asked for a Nuxt 3 PWA shell with `/login`, `/`, `/pet`, a `$fetch` wrapper, stores, a route guard and a CI job. What exists is the Frontend spec §7 set of five wireframes plus a `/settings` placeholder — `/login`, `/onboarding`, `/`, `/learn/:id`, `/roadmap`, `/revive` — with the plant folded into the dashboard hub as §7.2 draws it. The plan's *Notes* justified the expansion (spec wins for its own layer) and the idea's `## Evaluation` records it; that is the right call, not scope creep.

Two substitutions are documented and defensible: the platform `fetch` with an injectable override instead of `$fetch` (the client unit-tests without ofetch internals, same behaviour), and `utils/progress.ts`/`plant.ts`/`roadmap.ts` carrying the arithmetic so it is testable without a Nuxt runtime. Offline progress queueing is explicitly deferred with a reason (it would change the backend's once-per-day hook semantics).

## Code vs plan

All 15 tasks present, one commit each (`git log --oneline main..HEAD` → 15 commits, plus nothing else). TDD ordering is visible in the commit sequence. The six declared deviations are each real and each harmless — I checked the one that was worth checking rather than accepting:

1. **`@vite-pwa/assets-generator` `^0.2.6` → `^1.0.0`.** Genuine. `node_modules/@vite-pwa/nuxt/package.json` → `"peerDependencies": {"@vite-pwa/assets-generator":"^1.0.0"}`. Optional peer, but npm still fails `ERESOLVE` when a conflicting version is a direct devDependency. Resolved `1.0.4`.
2. **`tailwind.config.ts` needs `content: []`.** Genuine — `node_modules/tailwindcss/types/config.d.ts:359` makes `content: ContentConfig` required on `RequiredConfig`. And it does *not* disable scanning: the built CSS contains `.bg-alert`, `.text-growth`, `.rounded-card` and `.font-display`, so `@nuxtjs/tailwindcss` is supplying its own globs as claimed.
3. **`Reflect.deleteProperty` over dynamic `delete`** (`stores/quest.ts:108`) — `@typescript-eslint/no-dynamic-delete`. Identical behaviour.
4. **`duration-300` not `duration-400`** (`components/ui/SegmentedProgress.vue:39`) — `duration-400` is not a Tailwind 3.4 default; the plan's own Step 4 predicted this. The unused `DAILY_TARGET_SECONDS` import is gone, also as the plan allowed.
5. **`:id` before `:to`** (`components/roadmap/RoadmapNode.vue:19-20`) — `vue/attributes-order`. Cosmetic.
6. **`test-results/` + `playwright-report/` in `.gitignore`** — correct; without it the documented `npm run test:e2e` leaves the tree dirty.

**CODEMAP.** The resolved-version list is accurate, not aspirational — I checked every entry against `package-lock.json`: `nuxt@3.21.11`, `vue@3.5.43`, `pinia@3.0.4`, `@pinia/nuxt@0.11.3`, `@vite-pwa/nuxt@1.1.1`, `@nuxtjs/tailwindcss@6.14.0`, `@nuxt/eslint@1.17.0`, `eslint@9.39.5`, `vitest@3.2.7`, `@vue/test-utils@2.5.1`, `happy-dom@17.6.3`, `typescript@5.9.3`, `vue-tsc@2.2.12`, `@playwright/test@1.63.0`, `workbox-precaching@7.4.1`, `@vite-pwa/assets-generator@1.0.4`. All match. Two CODEMAP defects: the CI section still opens "Three parallel GitHub Actions jobs" while listing four (filed), and the shell paragraph repeats the plan's wrong "the spec shape, now live on `main`" (covered by the blocker).

**CI job.** No path filter, no `if:` condition — `.github/workflows/ci.yml:122-144` runs on every push to `main` or `harness/**` and on every PR, and its steps are exactly `npm ci`, `npm run lint`, `npm run typecheck`, `npm run test:unit`, `npm run build` from `working-directory: frontend`. It cannot silently skip. `gh run list --branch harness/2026-09-23-high-frontend-shell-…` → run `35818226698`, `completed success`, all four jobs green.

### Verification re-run (clean shell, from `frontend/` in the worktree)

```
$ rm -rf node_modules  (the worktree had none; .nuxt/.output only)
$ npm ci                    exit 0
$ npm run lint              exit 0, no output
$ npm run typecheck         exit 0
$ npm run test:unit
   Test Files  14 passed (14)
        Tests  58 passed (58)
$ npx playwright install chromium && npx playwright test --reporter=list
   3 passed (1.3s)
$ git status --short        clean
```

Everything the executor claimed reproduces. No executor gate failure. Lint, typecheck and tests are genuinely clean from a clean install, not only green in CI.

## Quality

**Token storage — defensible and deliberate.** The JWT sits in `localStorage['aelp.auth']`. That is the Frontend spec §4 design (`useAuthStore` manages the JWT), the backend issues a bearer JWT with no cookie path and no CSRF defence, and the app is `ssr: false` against a cross-origin API — so httpOnly cookies are not a drop-in alternative, they are a backend redesign. The choice is documented at `nuxt.config.ts:3-4` and in the plan's *Architecture*. What matters more is whether the surrounding code keeps the blast radius small, and here it does:

- `grep -rn "v-html\|innerHTML\|outerHTML\|eval("` over `pages components stores utils composables service-worker` → **no hits**. AI-generated `content_json` is rendered through text interpolation only (`utils/content.ts` classifies into words/questions/raw and `JSON.stringify`s the fallback). The main XSS vector a localStorage token would hand an attacker is absent.
- `grep -rn "console\.\|alert("` over the same tree → **no hits**. The token is never logged.
- The token never enters a URL. The only reader is `composables/useApi.ts:15` (`getToken: () => auth.accessToken`).
- **The bearer cannot leak to a third-party origin.** `utils/apiClient.ts:48` builds every request as `` `${opts.baseURL}${path}` `` and `baseURL` is bound once to `config.public.apiBase` (`useApi.ts:14`). There is no code path that hands the client an absolute URL, and even a hostile `path` concatenates onto the configured base rather than replacing it. This is the "blindly attaches the bearer to whatever URL it's handed" failure mode, and it is not present.
- OAuth hygiene is right: `randomState()` uses `crypto.getRandomValues` (16 bytes), the state round-trips through `sessionStorage` and is compared before the exchange (`pages/login.vue:26-31`), and `window.history.replaceState(null, '', '/login')` strips the `code` from the address bar *before* the exchange runs (`pages/login.vue:54`).

**The auth contract.** `stores/auth.ts:59-65` reads `access_token` and rejects anything else; `grep -rn '"token"' stores/auth.ts pages/login.vue` → no hits. `expires_in` is **not** parsed and discarded — it becomes `expiresAt = now + expires_in * 1000`, which `isAuthenticated` enforces (`auth.ts:39`) and the global guard evaluates on every navigation. There is no proactive re-auth scheduler, but the Backend spec exposes no refresh endpoint (§6 has no `/auth/refresh`; `expires_in` is 86400 and the only renewal is a fresh Google consent), so expiry-checking is the whole of what that field can drive here. The real defect around this contract is not in the code at all — it is the plan's claim that the §6.1 shape is live on `main`, which it is not (blocker below).

**Stub boundary — comparison right, default wrong.** `useOnboardingApi.ts:6` is `=== 'true'`, so no `Boolean("false") === true` bug: `"false"`, `"0"` and `""` all disable the stub. But `nuxt.config.ts:34` defaults it to `'true'`, so *unset* means stub-on and a deploy that forgets the variable serves a fake placement quiz to real users. Filed medium.

**Service worker.** `sw.ts:15-18` NetworkFirst for `/api/v1/(quests/daily|pet/status)` and `:22-25` StaleWhileRevalidate for `style|script|font|image`, which is exactly Frontend spec §3 ("User Progress & Pet Status: NetworkFirst", "Static Assets & Exercises: StaleWhileRevalidate"). `parsePushPayload` is defensive in the right place — `push.ts:14` only accepts a `url` starting with `/`, so a push payload cannot drive `notificationclick` to a third-party origin. I verified the `api-state` cache name is usable from the page: workbox's `cacheNames.getRuntimeName` returns an explicitly-passed name verbatim (`node_modules/workbox-core/_private/cacheNames.js`), so `caches.delete('api-state')` really does clear the worker's cache. The flaw is that one of three sign-out paths forgets to call it — filed medium.

**Offline and error states — good, with one bad outlier.** `/` , `/roadmap`, `/learn/:id` and `/onboarding` all distinguish loading / error / empty and offer a retry; `stores/*.load()` always clear `loading` in a `finally`, so there is no infinite-spinner path anywhere. `/revive` is the exception and it fails in the worst direction: with `/pet/status` unreachable its final `v-else` renders a wilted plant, a red "your plant has withered" banner and the revival CTA to a user whose plant is fine. Reproduced in a real browser. Filed high.

**Test honesty — mutation-tested, three of four assertions carried their claim.**

| Mutation | Test | Result |
| --- | --- | --- |
| `stores/auth.ts` `signIn` reads `(res as {token}).token` in guard and assignment | `tests/unit/authStore.test.ts` | **caught** — line 21 `expect(auth.accessToken).toBe('eyJ.test')` failed (1 failed / 4 passed) |
| same mutation | `authStore.test.ts:31` "does not accept the pre-spec `{token}` shape" | **not caught** — still passed; the fixture omits `expires_in`, so the guard's other half rejects it regardless. Filed low |
| `utils/progress.ts` `segmentFills` drains `segmentSeconds / 2` per segment | `tests/unit/progress.test.ts` | **caught** — "fills segments left to right": `expected [1,1,1] to deeply equal [1,1,+0]` |
| `utils/plant.ts` `healthTone` threshold `60 → 50` | `tests/unit/plant.test.ts` | **caught** — "tones health: growth ≥ 60…": `expected 'growth' to be 'streak'`. The suite pins the exact boundary |
| `middleware/auth.global.ts` returns without `navigateTo('/login')` | `tests/e2e/login.spec.ts:50` | **caught** — rebuilt and re-ran Playwright; "a guarded route without a token redirects to /login" failed at `page.waitForURL(/\/login$/)` (1 failed / 2 passed) |

All five mutations reverted; `git diff --quiet` clean on each file, `git status --short` empty in the worktree, and `.output` rebuilt from the restored source.

**Other quality notes (no bug filed).** `pages/learn/[id].vue` shows the "not in today's quests" empty state when `quest.load()` fails outright rather than an error state — the action button recovers the user, so it is a nit not a defect. `AppHeader`'s menu has no outside-click dismissal, which the plan already flagged as MVP scope. `stores/pet.ts`'s client-side revive anchor in `localStorage['aelp.revive']` is honest about being an anchor and re-posts for the server's truth; the plan records the DTO extension as a pet follow-up.

## Bugs filed

- **BLOCKER** `harness/ideas/_inbox/frontend-shell-plan-claims-the-6-1-auth-shape-is-live-on-mai.md` (high) — the plan's *Merge blocker* header and its `POST /auth/google` matrix row now assert the §6.1 shape is "live on `main`". It is not: `git show main:backend/internal/auth/handler.go` line 32 is still `"token": out.Token`, and the fix plan is `status: done, merged: false`. The plan's own *Notes* still say the opposite and instruct the reviewer to file a blocker if the dependency is open. It is. Merging this branch first ships a frontend that cannot sign in.
- `harness/ideas/_inbox/revive-shows-a-false-your-plant-is-dead-alarm-whenever-get-p.md` (high) — `/revive` has no error branch; a failed `GET /pet/status` renders the wilted UI. Reproduced in Chromium against a dead API base.
- `harness/ideas/_inbox/expired-session-sign-out-leaves-per-user-api-responses-in-th.md` (medium) — `middleware/auth.global.ts:13` signs out without `clearApiCache()`, unlike the other two sign-out paths; the `api-state` cache (keyed on URL only, no expiry) can then serve one account's quests and pet state to the next.
- `harness/ideas/_inbox/nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment.md` (medium) — the stub flag's comparison is correct but its default is `'true'`, so a forgotten env var ships the fake placement quiz.
- `harness/ideas/_inbox/authstore-test-does-not-accept-the-pre-spec-token-shape-pass.md` (low) — that test passes under the mutation it is named for.
- `harness/ideas/_inbox/codemap-ci-section-still-says-three-parallel-jobs-after-the-.md` (low) — CODEMAP says three jobs, lists four. I prepared the one-word fix on the branch but the sandbox denied the commit, so it is filed rather than fixed.

## Verdict

**pass-with-bugs**, with one merge blocker outstanding.

The slice is delivered and the code is good: every plan task landed, all six deviations are real and justified, lint / typecheck / 58 unit tests / 3 Playwright specs / the build all reproduce clean from a clean install, CI is four-for-four green on the pushed branch, and the security questions this slice actually raises come out well — the bearer token cannot reach a foreign origin, it is never logged or put in a URL, there is no `v-html` anywhere, and the OAuth `state` and `code` handling is correct. The mutation tests confirm the suite is honest about its central claims.

It must not merge yet, and not because of the code. The plan asserts that the §6.1 sign-in contract this frontend targets is live on `main`; it is not, and merging in the wrong order ships an app nobody can sign into. Land `harness/2026-09-23-high-auth-google-response-returns-token-and-omits-token-type-and-` on `main` first, correct the plan's two passages and the CODEMAP sentence that repeat the claim, and this branch is ready. The `/revive` false alarm is the one code bug worth fixing soon; the rest can wait for the next ideation run.

**PR:** none — `gh pr create` 403s because the `gh` CLI is authenticated as the owner's work account, which is not a collaborator on this repo. No PR to comment on or mark ready; expected and not a finding.
