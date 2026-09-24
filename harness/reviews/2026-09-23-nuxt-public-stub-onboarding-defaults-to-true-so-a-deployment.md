---
plan: harness/plans/2026-09-23-nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa.md, harness/ideas/_inbox/onboarding-s-ai-error-copy-is-untested-deleting-the-whole-br.md, harness/ideas/_inbox/the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own.md]
---
# Review — /onboarding: delete the stub and wire the page to the real onboarding endpoints

**Plan:** `harness/plans/2026-09-23-nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment.md`
**Branch/worktree:** `harness/2026-09-23-high-nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment` / `.worktrees/nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment`
**Diff:** `git diff main...harness/2026-09-23-high-nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment --stat` — 9 files, +154 / −97, frontend + CODEMAP only.
**CI:** `gh run list --branch …` → `completed success … 35847895562`, on the branch tip `0be853d`. Green.

## Plan vs idea

The idea asked for the fail-open default to stop fabricating a level and a roadmap id; its own *Expected output* said that **when the onboarding slice merges, the flag and `stubs/onboarding.ts` are deleted**. The slice is merged, and that is what shipped — deletion, not a flipped default. Delivered in full.

- `frontend/stubs/` is gone (`find . -maxdepth 1 -name stubs` → nothing), and `tests/unit/onboardingStub.test.ts` with it.
- `grep -rn 'STUB_ONBOARDING\|stubOnboarding\|stubs/onboarding\|isStub' . --exclude-dir=node_modules --exclude-dir=.nuxt --exclude-dir=.output` → no output. No orphaned import.
- After `npm run build`: `grep -rl 'stub-roadmap' .output` → nothing; `grep -rl 'stubOnboarding\|STUB_ONBOARDING' .output` → nothing. The fabricated roadmap id is not in any shipped bundle.
- `nuxt.config.ts`, `.env.example` and `playwright.config.ts` each lost the variable, and the `nuxt.config.ts` comment above `runtimeConfig.public` was narrowed to the three surviving names rather than left stale.

## Code vs plan

All four tasks followed, no deviations. The four wire types moved into `composables/useOnboardingApi.ts` as exported interfaces; `quiz()`/`assess()` are unconditional `useApi()` calls with no `isStub` and no `#app` import; the page imports its types from the composable, gained `assessErrorMessage`, changed `catch {` → `catch (e)`, and dropped the `api.isStub` "Bản thử" paragraph. CODEMAP's `shell` bullet was updated on both counts.

### Re-run of the plan's Verification (worktree, `env -u NUXT_PUBLIC_API_BASE -u PORT -u HOST`)

```
npm run lint       → exit 0
npm run typecheck  → exit 0
npm run test:unit  → Test Files 15 passed (15) / Tests 62 passed (62)
npm run build      → exit 0
git status --short → clean
git diff --stat main..HEAD -- ../backend → empty   (backend/cmd/api/main.go untouched)
```

### The central question: does the page match the merged backend?

I did **not** take the executor's hand-written Node stub as evidence. I ran the real Go binary — `COMPOSE_PROJECT_NAME=onbrev`, Postgres 5453 / Redis 6401 / API 8107, migrations `0001_init` + `0002_google_sync` applied, `/healthz` `{"postgres":"ok","redis":"ok","status":"ok"}` — seeded two users, minted HS256 session JWTs against `JWT_SECRET` and the §4 `sess:{user_id}:token` keys, and drove the branch's production build (`node .output/server/index.mjs`) against it in a real Chromium tab.

**Contract diff: no drift.** Field for field, `composables/useOnboardingApi.ts` matches what the binary actually emits.

| | Real backend response | Page's types / body |
| --- | --- | --- |
| `GET /onboarding/quiz` | `200`, `{questions:[…]}`, **10** items `q1..q10`, per-item keys exactly `id`/`prompt`/`options`, options keyed `A`–`D`, no `correct`/`level` leaked | `QuizResponse{questions: QuizQuestion[]}`, `QuizQuestion{id, prompt, options: Record<string,string>}` — identical |
| `POST /onboarding/assessment` body | `types.go AssessmentRequest{target_goal, notification_time, timezone, answers[{question_id, selected_option}]}` | same four names; `HH:MM:SS` sent as `` `${time}:00` ``, IANA zone from `Intl` |
| `201` / `200` | `{status, assessed_level, roadmap_id, pet_state{plant_name, health_points, stage}}` — byte-identical on both codes | `AssessmentResponse` — identical; `apiClient` branches on `res.ok`, so both succeed |
| error envelope | `{"error":"<code>"}` (a bare string) | `apiClient.errorCode()` reads `typeof body.error === 'string'` → `ApiError.code` |

**Every status code driven on the real binary**, not simulated:

| Condition | Real response | What the page does |
| --- | --- | --- |
| `notification_time: "20:00"` / `":00"` | `400 invalid_request` | generic "Không tạo được lộ trình. Thử lại." — **filed as a bug** |
| 5 assessments inside a minute (`ratelimit:ai:{user}`) | `429 rate_limited` | "Bạn vừa gửi quá nhiều lần…" |
| no provider API key set | `503 ai_unavailable` | AI-specific copy |
| provider returns non-JSON (2 attempts) | `502 ai_bad_output` | AI-specific copy — **seen rendered in the browser** |
| provider returns 500 | `502 ai_upstream_failed` | AI-specific copy |
| valid roadmap from the provider | `201` | result card |
| resubmit with a roadmap already active | `200`, same `roadmap_id`, no AI call | result card (identical handling) |

**Browser run against the real binary** (fresh user, quiz fetched from `bank.go`):

- `/onboarding` rendered **"Câu 1 / 10"** with the bank's own first item, `"She ___ a teacher."`, options `(A) am (B) is (C) are (D) be` in A–D order; no "Bản thử" note anywhere. Question 10 was the bank's real `q10`, `"Had it not been for the delay, we ___ the deadline."`
- First submit (provider in garbage mode) → `POST … => [502]` in `browser_network_requests`, page rendered **"Máy chủ AI đang bận, chưa chấm được bài. Thử lại sau ít phút."** and stayed on "Câu 10 / 10" with answers intact.
- Resubmit (provider healthy) → `POST … => [201]`, page rendered **"Trình độ của bạn: C1"** and **"My Green Buddy đã nảy mầm…"** — the server's level and the pet row's real default name, not a client constant.
- Postgres afterwards: `users` = `C1 | IELTS 7.0 Preparation | 20:00:00 | Asia/Saigon`; **1** active roadmap; **84** exercises; a `pet_states` row `My Green Buddy|100|sprout`; Redis `quiz:placement:{user}` cleared. The page's request body is what landed in the database.
- **200 already-active**: revisiting `/onboarding` as that user redirected to `/`, which rendered day 1 of the roadmap the page had just created. The client-side 200 branch itself I drove by curl (request G above): `HTTP 200` with the existing `roadmap_id` and the same body shape, which `res.ok` accepts.

This supersedes the executor's stub-server proof; the shared-browser question is moot for this review. (For the record, the executor has since appended a re-run note to its summary saying the disrupted attempt was discarded and the flow redone twice more in a dedicated tab.)

## Quality

**Mutations.** The executor's two reproduce exactly; my third does not, and that is the finding.

1. `quiz()` → literal single-question response: `AssertionError: expected "spy" to be called with arguments: [ '/api/v1/onboarding/quiz' ]` at `onboardingPage.test.ts:77`; **3 failed**. Restored, `git diff --quiet` clean.
2. `assess()` → literal `B1` / `My Green Buddy`: case 2 `expected "spy" to be called 1 times, but got 0 times`; case 3 `Cannot call text on an empty DOMWrapper`; case 1 still passes. **2 failed | 1 passed**. Restored clean.
3. **Mine** — delete `pages/onboarding.vue:15` outright, so all three `ai_*` codes collapse to the generic message: **3 passed**. The suite does not hold the AI copy in place at all. Filed. (A 200-already-active mutation is not meaningful at this layer — the page has no code that distinguishes 200 from 201, which is exactly why the real-backend curl above is the right instrument for it; I used the untested `ai_*` branch instead.)

**Test honesty** — mostly good, one overstatement. The three cases do prove server data rendered: `Câu 1 / 10` cannot come from the five-item stub, and the assertions on `C1` / `Cây Thử` are values the stub never produced. But `onboardingPage.test.ts:18` claims "prompts exist only in the real bank" while 9 of the 10 are `Real bank item N`, which exist in neither the bank nor the stub; only `q1`'s prompt is genuine. The assertion that carries the weight is still sound — noting it as comment accuracy, not filing it.

**Error handling.** `assessErrorMessage` covers `rate_limited` and `ai_*` honestly, and I confirmed those are the codes the binary emits. Nothing is swallowed silently — every catch sets a visible `role="alert"` and keeps the answers. Two gaps: `invalid_request` has no copy of its own and is reachable from the UI (filed), and `startQuiz()`'s bare `catch {` at `pages/onboarding.vue:47` discards the error object entirely, so an unexpected quiz failure is indistinguishable from a network one and nothing is logged — low, unchanged from the shell plan, not filed separately.

**Design, boundaries, conventions.** The composable is now two thin calls with the wire types co-located, matching how `stores/quest.ts` carries its §6.2 types; dropping `#app` is what makes the page testable under plain Vitest, which is the suite's existing idiom. `useApi()` is called per request rather than at setup — harmless, since `useApi()` memoises its client. Naming, quoting and comment density match the neighbouring files; lint is clean in one pass. CODEMAP's new `shell` text is accurate against what I observed (ten items, both 200 and 201 success, types in the composable, `rate_limited`/`ai_*` copy), and the runtime-config list correctly lists three variables.

**Concurrent edits.** No overlap with the session-cache branch: it touches `composables/useApi.ts`, `stores/auth.ts`, `middleware/auth.global.ts`, `components/AppHeader.vue` and three tests; this branch touches none of them. Semantically safe too — that branch changes only the internals of `useApi()`'s 401 handler, leaving the signature and the memoised-singleton behaviour this composable depends on intact, and the onboarding test mocks `~/composables/useApi` wholesale. `git merge-tree --write-tree main HEAD` against the current `main` (`084b72a`) produces a tree with no conflicts.

**Out of scope but found here:** the API sends no CORS headers at all, so the deployed PWA on its own Railway origin cannot reach a single endpoint. I only got a browser onto the real backend by putting a same-origin proxy in front of it. Pre-existing, affects every page; filed separately.

## Bugs filed

- `harness/ideas/_inbox/a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa.md` — medium. A cleared `<input type="time">` posts `notification_time: ":00"`, the real backend answers `400 invalid_request`, and the learner sees only the generic message after ten answered questions, with every retry failing the same way.
- `harness/ideas/_inbox/onboarding-s-ai-error-copy-is-untested-deleting-the-whole-br.md` — medium. Deleting the `ai_*` branch leaves the suite green, so the copy for `503 ai_unavailable` / `502 ai_bad_output` / `502 ai_upstream_failed` — the path any key-less deploy takes — is unguarded.
- `harness/ideas/_inbox/the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own.md` — medium, pre-existing and not caused by this branch. Preflight `OPTIONS` 404s and no `Access-Control-Allow-Origin` is ever sent.

None is a blocker. The branch is a strict improvement over what is on `main`, and every one of these is a smaller problem than the stub it replaces.

## Verdict

**pass-with-bugs.** The plan delivered the idea in full, and the page genuinely matches the merged backend — proven against the real Go binary, not a reproduction: ten real bank items, the §6.1 body landing verbatim in Postgres with 84 exercises, and all six status codes (`400`/`429`/`502`×2/`503`/`201`/`200`) driven live. Three medium bugs filed, none merge-blocking.
