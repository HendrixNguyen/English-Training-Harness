---
idea: harness/ideas/_inbox/nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment.md
status: draft
priority: high
merged: false
---
# /onboarding: delete the stub and wire the page to the real onboarding endpoints — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/_inbox/nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment.md`
**Goal:** `/onboarding` always calls `GET /api/v1/onboarding/quiz` and `POST /api/v1/onboarding/assessment`; `NUXT_PUBLIC_STUB_ONBOARDING`, `stubs/onboarding.ts` and the "Bản thử" note are gone, so no build can fabricate a level and a roadmap id.

**Why now (`priority: high`):** the backend onboarding slice is merged and live on `main` (`backend/cmd/api/main.go:135-136` mounts both routes behind `auth.Require()`), so the stub is dead code with a loaded default — `nuxt.config.ts:34` says `'true'`, `.env.example` tells a deployer to copy `true`, and a build that forgets the variable runs every new user through five fake questions, creates no roadmap, and strands them on `no_active_roadmap` forever. Flipping the default to `'false'` would leave the trap in place; deleting the branch removes it.

**Root cause (confirmed on `main`):** `composables/useOnboardingApi.ts:6` selects `stubs/onboarding.ts` when `useRuntimeConfig().public.stubOnboarding === 'true'`, and `nuxt.config.ts:34` defaults `stubOnboarding` to `'true'`. Nothing else reads the flag (`grep -rn stubOnboarding frontend --exclude-dir=node_modules` → `nuxt.config.ts`, `useOnboardingApi.ts`, `playwright.config.ts`, `.env.example`; the page reads only `api.isStub` for the note at `pages/onboarding.vue:93`).

**Real wire shapes vs. the stub (verified against `backend/internal/onboarding/{bank,types,handler,service}.go`; backend spec §6.1 is the contract for the assessment):**

| | Stub (`stubs/onboarding.ts`) | Real backend | Differs? |
| --- | --- | --- | --- |
| `GET /onboarding/quiz` 200 | `{questions: [{id, prompt, options: {A,B,C,D}}]}` × 5 | `bank.go` `QuizResponse{Questions: []PublicQuestion{id, prompt, options}}` × **10** (two per level A1–C1; `correct`/`level` never serialised) | field names identical; count 5 → 10, ids `q1..q10` |
| `POST /onboarding/assessment` body | `{target_goal, notification_time, timezone, answers: [{question_id, selected_option}]}` | `types.go` `AssessmentRequest` — same four fields, same names | identical |
| assessment success | always `201`-shaped `{status:'success', assessed_level, roadmap_id:'stub-roadmap-…', pet_state:{plant_name, health_points, stage}}` | `201` on a new roadmap **or `200`** when one is already active (returns the existing `roadmap_id`, no AI call, no write); same field names; `roadmap_id` a real UUID | field names identical; **status code 200 or 201** — `utils/apiClient.ts` accepts any `res.ok`, so no client change |
| assessment errors | never fails | `400 invalid_request`, `429 rate_limited`, `502 ai_bad_output` / `ai_upstream_failed`, `503 ai_unavailable`, `500 internal_error` (`handler.go:44-58`) | **new**: the page currently shows one generic message for any failure |
| server validation | none | `service.go validate()`: `target_goal` 1..255 chars; `timezone` a valid IANA zone; `notification_time` **`HH:MM:SS`**; `answers` non-empty, every `question_id` in the bank, every `selected_option` one of its keys, no duplicates | the page already satisfies all of these: it sends `` `${time}:00` `` (`<input type="time">` yields `HH:MM`), `Intl.DateTimeFormat().resolvedOptions().timeZone`, and one answer per served question |

So: **no request or response field differs**; the differences are the question count, the 200-vs-201 status, and the error codes the page should name honestly.

**Architecture:** the four wire types move from `stubs/onboarding.ts` into `composables/useOnboardingApi.ts` (exported, the way `stores/quest.ts` co-locates its §6.2 types). `useOnboardingApi()` loses `isStub` and the `#app` import and becomes two thin calls on `useApi()`, which also makes it mockable from Vitest with the suite's existing `vi.mock('~/composables/useApi')` idiom. The page keeps its three steps and copy; the only behavioural additions are an `ApiError.code`-aware message for the assess step (`rate_limited`, the three `ai_*` codes, everything else) and the removal of the stub note. Playwright's `serviceWorkers: 'block'` and `page.route` stubs are unaffected; the config just stops setting the dead variable.

**Test approach:** a Vitest component test of `pages/onboarding.vue` following `tests/unit/revivePage.test.ts` (`setActivePinia`, `vi.mock('~/composables/useApi')` with `get`/`post` routed by path, explicit component registration, `navigateTo` stubbed). The mock serves ten questions whose prompts exist only in the real bank (never in the stub) and an assessment whose values differ from the stub's constants (`assessed_level: 'C1'`, `plant_name: 'Cây Thử'`), so a test that passes proves the page rendered *server* data, not a literal. `tests/unit/onboardingStub.test.ts` is deleted with its subject.

**Tech stack:** Nuxt 3 / Vue 3 / Pinia on `main`; Vitest + `@vue/test-utils` + `happy-dom`. No new dependencies. Frontend-only: **`backend/` is not touched, and `backend/cmd/api/main.go` in particular is owned by another plan.**

**Run every command from `frontend/` inside the worktree.** `npm ci` first. `rg` is not installed — use `grep -n`. Strict TypeScript; lint + prettier-style must be clean in one pass (`npm run lint` is ESLint via `@nuxt/eslint` stylistic rules — no semicolons, single quotes, trailing commas as in the neighbouring files).

---

## File structure

| Path | Change |
| --- | --- |
| `frontend/composables/useOnboardingApi.ts` | Rewrite: exports the four wire types; `quiz()`/`assess()` call `useApi()` unconditionally; no `isStub`, no `#app` |
| `frontend/pages/onboarding.vue` | Import types from the composable; delete the `api.isStub` note; map assess `ApiError.code` to a message |
| `frontend/stubs/onboarding.ts` | **Delete** (and the now-empty `stubs/` directory) |
| `frontend/tests/unit/onboardingStub.test.ts` | **Delete** |
| `frontend/tests/unit/onboardingPage.test.ts` | **New**: three component tests |
| `frontend/nuxt.config.ts` | Remove `stubOnboarding` from `runtimeConfig.public` and from the comment above it |
| `frontend/.env.example` | Remove the two `NUXT_PUBLIC_STUB_ONBOARDING` lines |
| `frontend/playwright.config.ts` | Remove `NUXT_PUBLIC_STUB_ONBOARDING: 'true'` from `webServer.env` |
| `harness/CODEMAP.md` | `shell` bullet: `/onboarding` now against the real endpoints; drop the variable from the runtime-config list |

---

## Tasks

### Task 1: Component test that fails against the stub

**Files:**
- Create: `frontend/tests/unit/onboardingPage.test.ts`

- [ ] **Step 1: Write the test**

```ts
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import GoalCard from '~/components/onboarding/GoalCard.vue'
import PlantSvg from '~/components/plant/PlantSvg.vue'
import AppButton from '~/components/ui/AppButton.vue'
import AppCard from '~/components/ui/AppCard.vue'
import StateBlock from '~/components/ui/StateBlock.vue'
import { ApiError } from '~/utils/apiClient'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))
const navigateTo = vi.fn()
vi.stubGlobal('navigateTo', navigateTo)

const { default: OnboardingPage } = await import('~/pages/onboarding.vue')

/** Ten items shaped like backend bank.go's PublicBank(); prompts exist only in the real bank. */
const QUIZ = {
  questions: Array.from({ length: 10 }, (_, i) => ({
    id: `q${i + 1}`,
    prompt: i === 0 ? 'She ___ a teacher.' : `Real bank item ${i + 1}`,
    options: { A: 'am', B: 'is', C: 'are', D: 'be' },
  })),
}
/** §6.1 201 body with values the stub never produced. */
const ASSESSED = {
  status: 'success',
  assessed_level: 'C1',
  roadmap_id: 'b11c22d3-44e5-66f7-88a9-00bbccddeeff',
  pet_state: { plant_name: 'Cây Thử', health_points: 100, stage: 'sprout' },
}

function mountPage() {
  api.get.mockImplementation((path: string) =>
    path.endsWith('/onboarding/quiz') ? Promise.resolve(QUIZ) : Promise.reject(new ApiError(404, 'no_active_roadmap')))
  return mount(OnboardingPage, {
    global: {
      components: { AppButton, AppCard, GoalCard, PlantSvg, StateBlock },
      stubs: { AppHeader: true },
      mocks: { navigateTo },
    },
  })
}

async function click(w: VueWrapper, text: string) {
  const b = w.findAll('button').find(x => x.text().includes(text))
  if (!b) throw new Error(`no button "${text}"`)
  await b.trigger('click')
  await flushPromises()
}

/** Goal → start → answer option A on every item → submit. */
async function completeQuiz(w: VueWrapper) {
  await click(w, 'IELTS 7.0')
  await click(w, 'Bắt đầu bài kiểm tra')
  for (let i = 0; i < QUIZ.questions.length; i++) {
    await w.find('[role="radio"]').trigger('click')
    await click(w, i === QUIZ.questions.length - 1 ? 'Hoàn thành' : 'Tiếp tục')
  }
}

describe('/onboarding against the real endpoints (backend spec §6.1)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    api.get.mockReset()
    api.post.mockReset()
    navigateTo.mockReset()
  })

  it('loads the placement items from GET /api/v1/onboarding/quiz and renders what the server sent', async () => {
    const w = mountPage()
    await flushPromises()
    await click(w, 'IELTS 7.0')
    await click(w, 'Bắt đầu bài kiểm tra')

    expect(api.get).toHaveBeenCalledWith('/api/v1/onboarding/quiz')
    expect(w.text()).toContain('Câu 1 / 10')
    expect(w.text()).toContain('She ___ a teacher.')
    expect(w.text()).not.toContain('Bản thử')
  })

  it('posts the §6.1 assessment body and renders the assessed level and pet from the response', async () => {
    api.post.mockResolvedValue(ASSESSED)
    const w = mountPage()
    await flushPromises()
    await completeQuiz(w)

    expect(api.post).toHaveBeenCalledTimes(1)
    const [path, body] = api.post.mock.calls[0] as [string, Record<string, unknown>]
    expect(path).toBe('/api/v1/onboarding/assessment')
    expect(body).toEqual({
      target_goal: 'IELTS 7.0 Preparation',
      notification_time: '20:00:00',
      timezone: expect.stringMatching(/.+/),
      answers: QUIZ.questions.map(q => ({ question_id: q.id, selected_option: 'A' })),
    })
    expect(w.text()).toContain('Trình độ của bạn: C1')
    expect(w.text()).toContain('Cây Thử đã nảy mầm')
  })

  it('names a 429 rate_limited honestly and keeps the learner on the quiz with their answers', async () => {
    api.post.mockRejectedValue(new ApiError(429, 'rate_limited'))
    const w = mountPage()
    await flushPromises()
    await completeQuiz(w)

    expect(w.find('[role="alert"]').text()).toContain('quá nhiều lần')
    expect(w.text()).toContain('Câu 10 / 10')
    expect(w.text()).not.toContain('Trình độ của bạn')
  })
})
```

- [ ] **Step 2: Run it and confirm it fails for the right reason**

Run: `npx vitest run tests/unit/onboardingPage.test.ts`
Expected: the file fails to load — `pages/onboarding.vue` imports `useOnboardingApi`, which imports `useRuntimeConfig` from `#app`, unresolvable outside Nuxt (or, if it does resolve, case 1 fails on `Câu 1 / 10` because the stub serves five items). Either failure is the stub's dependency chain; if all three cases pass, stop — you are not on the code this plan was written against.

- [ ] **Step 3: Commit the failing test**

```bash
git add tests/unit/onboardingPage.test.ts
git commit -m "onboarding: component test — /onboarding must load and submit against the real §6.1 endpoints"
```

---

### Task 2: Replace the stub with the real client

**Files:**
- Modify: `frontend/composables/useOnboardingApi.ts`
- Modify: `frontend/pages/onboarding.vue`
- Delete: `frontend/stubs/onboarding.ts`, `frontend/tests/unit/onboardingStub.test.ts`

- [ ] **Step 1: Rewrite `composables/useOnboardingApi.ts`**

```ts
import { useApi } from '~/composables/useApi'

/** GET /api/v1/onboarding/quiz item (backend bank.go PublicQuestion — no answer, no level). */
export interface QuizQuestion {
  id: string
  prompt: string
  options: Record<string, string>
}
export interface QuizResponse {
  questions: QuizQuestion[]
}
/** Backend spec §6.1 POST /onboarding/assessment body. */
export interface AssessmentRequest {
  target_goal: string
  notification_time: string
  timezone: string
  answers: { question_id: string, selected_option: string }[]
}
/** §6.1 response — 201 on a new roadmap, 200 when one was already active. */
export interface AssessmentResponse {
  status: 'success'
  assessed_level: string
  roadmap_id: string
  pet_state: { plant_name: string, health_points: number, stage: string }
}

export function useOnboardingApi() {
  return {
    quiz: (): Promise<QuizResponse> => useApi().get<QuizResponse>('/api/v1/onboarding/quiz'),
    assess: (req: AssessmentRequest): Promise<AssessmentResponse> =>
      useApi().post<AssessmentResponse>('/api/v1/onboarding/assessment', req),
  }
}
```

- [ ] **Step 2: Edit `pages/onboarding.vue` (script and one template line)**

In `<script setup>`:
- Replace `import type { AssessmentResponse, QuizQuestion } from '~/stubs/onboarding'` with `import { useOnboardingApi, type AssessmentResponse, type QuizQuestion } from '~/composables/useOnboardingApi'` (drop the separate `useOnboardingApi` import line) and add `import { ApiError } from '~/utils/apiClient'`.
- Add, after `GOALS`:

```ts
/** Honest copy per backend error code (handler.go); answers are kept in every case. */
function assessErrorMessage(e: unknown): string {
  if (e instanceof ApiError && e.code === 'rate_limited') return 'Bạn vừa gửi quá nhiều lần. Đợi một phút rồi thử lại.'
  if (e instanceof ApiError && e.code.startsWith('ai_')) return 'Máy chủ AI đang bận, chưa chấm được bài. Thử lại sau ít phút.'
  return 'Không tạo được lộ trình. Thử lại.'
}
```

- In `next()`, change `catch {` / `error.value = 'Không tạo được lộ trình. Thử lại.' // answers are kept` to `catch (e) {` / `error.value = assessErrorMessage(e)`.

In the template: delete the `<p v-if="api.isStub" …>Bản thử…</p>` block (lines 93–95 on `main`). Nothing else in the template changes.

- [ ] **Step 3: Delete the stub and its test**

```bash
git rm stubs/onboarding.ts tests/unit/onboardingStub.test.ts
```
`stubs/` is then empty and disappears from git; confirm `grep -rn 'stubs/onboarding\|isStub' --include='*.ts' --include='*.vue' . --exclude-dir=node_modules --exclude-dir=.nuxt --exclude-dir=.output` → no hits.

- [ ] **Step 4: Run the new test, then the whole suite**

Run: `npx vitest run tests/unit/onboardingPage.test.ts` → 3 passed.
Run: `npm run test:unit` → all passed (`onboardingStub.test.ts` gone, `onboardingPage.test.ts` in; the other files are untouched).

- [ ] **Step 5: Commit**

```bash
git add composables/useOnboardingApi.ts pages/onboarding.vue
git commit -m "onboarding: call the real /onboarding/quiz and /onboarding/assessment; delete the stub"
```
(The `git rm` from step 3 is already staged.)

---

### Task 3: Remove the flag from config, env template and Playwright

**Files:**
- Modify: `frontend/nuxt.config.ts`, `frontend/.env.example`, `frontend/playwright.config.ts`

- [ ] **Step 1: `nuxt.config.ts`** — delete `stubOnboarding: 'true',` from `runtimeConfig.public`, and shorten the comment above it to name only the three remaining variables:

```ts
  // NUXT_PUBLIC_API_BASE, NUXT_PUBLIC_GOOGLE_CLIENT_ID, NUXT_PUBLIC_VAPID_PUBLIC_KEY
  // override these at runtime (Nuxt convention; the idea's bare API_BASE/
  // GOOGLE_CLIENT_ID/VAPID_PUBLIC_KEY names are the same values under the
  // NUXT_PUBLIC_ prefix).
```

- [ ] **Step 2: `.env.example`** — delete the comment line and `NUXT_PUBLIC_STUB_ONBOARDING=true`. Three variables remain.

- [ ] **Step 3: `playwright.config.ts`** — delete `NUXT_PUBLIC_STUB_ONBOARDING: 'true',` from `webServer.env`.

- [ ] **Step 4: Confirm the name is gone from source, then lint, typecheck, build**

```bash
grep -rn 'STUB_ONBOARDING\|stubOnboarding' . --exclude-dir=node_modules --exclude-dir=.nuxt --exclude-dir=.output   # → no output
npm run lint && npm run typecheck && npm run build
grep -rl 'stub-roadmap\|stubOnboarding' .output   # → no output: the fabricated roadmap id is not in any shipped bundle
```
Expected: exit 0 each; the two greps print nothing.

- [ ] **Step 5: Commit**

```bash
git add nuxt.config.ts .env.example playwright.config.ts
git commit -m "frontend: drop NUXT_PUBLIC_STUB_ONBOARDING — onboarding always hits the backend"
```

---

### Task 4: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (`shell` bullet only)

- [ ] **Step 1: Two edits in the `**shell**` bullet**

- Replace `` `/onboarding` (7.1, against `stubs/onboarding.ts` while `NUXT_PUBLIC_STUB_ONBOARDING=true`) `` with `` `/onboarding` (7.1, `GET /onboarding/quiz` → ten items, `POST /onboarding/assessment` per backend spec §6.1 — both 200 and 201 are success; wire types live in `composables/useOnboardingApi.ts`; `rate_limited` / `ai_*` get their own copy) ``.
- In the runtime-config list, delete `, `NUXT_PUBLIC_STUB_ONBOARDING``, leaving the three variables.

- [ ] **Step 2: Commit**

```bash
git add ../harness/CODEMAP.md
git commit -m "CODEMAP: /onboarding is wired to the real endpoints; stub flag retired"
```

---

## Verification

From `frontend/` in the worktree, clean shell (`env -u NUXT_PUBLIC_API_BASE -u PORT -u HOST`):

1. `npm ci && npm run lint && npm run typecheck` → exit 0 each, in one pass (no fix-up commits for lint/prettier/type errors).
2. `npm run test:unit` → all pass, including the three cases in `tests/unit/onboardingPage.test.ts`; `tests/unit/onboardingStub.test.ts` no longer exists.
3. **Mutation check (paste the output into the execution summary):**
   - In `composables/useOnboardingApi.ts`, temporarily make `quiz()` return `Promise.resolve({ questions: [{ id: 'q1', prompt: 'She ___ to work every day.', options: { A: 'go', B: 'goes', C: 'going', D: 'gone' } }] })` instead of calling `useApi()`. Run `npx vitest run tests/unit/onboardingPage.test.ts` → **case 1 must fail** on `expect(api.get).toHaveBeenCalledWith('/api/v1/onboarding/quiz')` (and on `Câu 1 / 10`). Restore.
   - Temporarily make `assess()` return `Promise.resolve({ status: 'success', assessed_level: 'B1', roadmap_id: 'x', pet_state: { plant_name: 'My Green Buddy', health_points: 100, stage: 'sprout' } })`. Run again → **case 2 must fail** on `expect(api.post).toHaveBeenCalledTimes(1)` and **case 3 must fail** (no alert). Restore; `git diff --quiet composables/useOnboardingApi.ts` clean; re-run → 3 passed.
4. `grep -rn 'STUB_ONBOARDING\|stubOnboarding\|stubs/onboarding\|isStub' . --exclude-dir=node_modules --exclude-dir=.nuxt --exclude-dir=.output` → no output.
5. `npm run build` → exit 0; `grep -rl 'stub-roadmap' .output` → no output.
6. `git diff --stat main..HEAD -- ../backend` → empty (frontend-only; `backend/cmd/api/main.go` untouched).
7. Push; `gh run list --branch <branch>` — `frontend`, `backend-unit`, `backend-integration`, `harness-tooling` all green.

## Notes and open questions

- **Why delete instead of default `'false'`:** a `'false'` default keeps a second code path that is never exercised in production and can be flipped back by one env line; the shell plan (Task 12) and the idea both said the stub goes when onboarding merges. It has.
- **Existing roadmap (200):** if a user reaches `/onboarding` with an active roadmap, `onMounted` already redirects to `/` once `quest.load()` succeeds; if they submit anyway, the backend returns the existing roadmap with 200 and the page shows it — the client treats both statuses as success, so nothing to add.
- **Playwright** keeps stubbing `**/api/v1/**` with `page.route` (`tests/e2e/login.spec.ts`); no onboarding e2e is added — Playwright is local-only (not in `ci.yml`), and the component test is the CI-effective proof. If an executor wants a browser check, `page.route` `/onboarding/quiz` and `/onboarding/assessment` the same way the spec stubs `/quests/daily`.
- **Not in scope:** the `GET /onboarding/quiz` endpoint is still absent from both specs (CODEMAP already flags it for the spec's owner); a dedicated `stores/onboarding.ts` is not warranted for a one-page flow.
