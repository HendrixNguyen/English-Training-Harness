---
idea: harness/ideas/2026-09-22-run-02/frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md
status: draft
priority: high
merged: false
order: 9
design: harness/designs/frontend-shell.md
---
# Frontend Shell: Nuxt 3 PWA with auth, daily quest and pet screens — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/2026-09-22-run-02/frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md`
**Design:** `harness/designs/frontend-shell.md` — layout, states, component inventory, tokens. Every screen below cites the design section it implements; when this plan and the design disagree on a visual, the design wins.
**Goal:** Stand up `frontend/` — a Nuxt 3 SPA/PWA (Vue 3, Pinia, Tailwind, `@vite-pwa/nuxt`) with the six screens of the daily loop (login, onboarding, dashboard + plant hub, learning room, roadmap tree, revival mode) wired to the Backend spec §6 endpoints through three Pinia stores, unit-tested with Vitest, smoke-tested with Playwright against a stubbed API, and checked by a new `frontend` CI job.

**Spec precedence (AGENTS.md → *Reading the spec*):** the *Frontend Technical Specification* wins for this layer — §2 stack, §3 caching strategies, §4 the three stores, §6.1 palette, §7 the five wireframes. Wire shapes come from the *Backend Technical Specification* §6.1–6.3 **as written**, not from what the merged handlers emit (see *Merge blocker* below). The 1st-thinking doc supplies the flow (§5.1 step 8, §5.2 step 5) and the env list (§8).

**Merge blocker (state this in the PR description):** Backend spec §6.1 says `POST /api/v1/auth/google` returns `{access_token, token_type, expires_in, user}`; the handler on `main` returns `{token, user}`. This slice reads `access_token` and `expires_in` and **does not** fall back to `token`. Until the inbox bug `harness/ideas/_inbox/auth-google-response-returns-token-and-omits-token-type-and-.md` is fixed on `main`, sign-in against the real backend stores no token. **Do not fix the backend in this slice.** The reviewer should file this as a blocker at review time; the plan is complete when the frontend is correct against the spec.

**Architecture:** `ssr: false` — the app is a client-rendered PWA (the token lives in `localStorage`; nothing is rendered per-user on a server). One pure API client (`utils/apiClient.ts`, injectable `fetch`) wrapped by one composable (`useApi`) that adds `Authorization: Bearer` from `useAuthStore` and signs out on 401. Three Pinia stores exactly as §4 names them — `useAuthStore`, `useQuestStore`, `usePetStore` — hold all server state; pages are thin. Pure helpers (`utils/progress.ts`, `utils/plant.ts`, `utils/roadmap.ts`, `service-worker/push.ts`) carry the logic that is unit-tested without Nuxt. A custom `injectManifest` service worker implements §3 (NetworkFirst for progress/pet, StaleWhileRevalidate for assets) plus the `push`/`notificationclick` handlers the idea asks for.

**Tech stack:** Nuxt 3 (latest 3.x — **not** Nuxt 4; the spec says Nuxt 3), Vue 3.5, Pinia 3 via `@pinia/nuxt`, Tailwind 3 via `@nuxtjs/tailwindcss`, `@vite-pwa/nuxt` + `workbox-*` 7, `@nuxt/eslint` (ESLint 9 flat config), Vitest 3 + `happy-dom` + `@vue/test-utils`, `@playwright/test`, `vue-tsc` for `nuxi typecheck`. Node LTS (CI: `actions/setup-node@v7`, `node-version: lts/*`). Self-hosted fonts via `@fontsource-variable/fraunces` and `@fontsource/source-sans-3` so the shell renders offline (§3).

**Version ranges are floors, not pins.** This plan was written without running `npm`. Every range in `package.json` below is a caret on the major the author believes current; `npm install` resolves the latest matching. If a range does not resolve (package renamed, major bumped), move the caret to the current major, keep going, and record the resolved majors in the execution summary and the CODEMAP paragraph. Do not downgrade Nuxt below 3.

**Run every command from `frontend/`** unless the step says otherwise. `rg` and `timeout` are not installed — `grep -n`, and bound long commands with the tool's flags (`vitest run` exits; `playwright test --timeout`).

---

## API → UI mapping matrix

Frontend spec §5 is titled "API Data to UI Mapping Matrix" but its body is a pasted copy of the backend architecture diagram, so the matrix is supplied here from Backend spec §6 and the §7 wireframes.

| Endpoint (Backend spec) | Store / action | Screen (wireframe) | Status at execution |
| --- | --- | --- | --- |
| `POST /api/v1/auth/google` §6.1 `{code, redirect_uri}` → `{access_token, token_type, expires_in, user}` | `useAuthStore.signIn` | `/login` (7.1 upper) | **live on `main`, wrong shape** — merge blocker above |
| `GET /api/v1/quests/daily` §6.2 → `{date, day_number, total_minutes_required, accumulated_seconds, is_target_met, tasks[]}`; 404 `no_active_roadmap` | `useQuestStore.load` | `/` (7.2), `/learn/:id` (7.3), `/roadmap` (7.4 — `day_number` only), `/revive` (7.5 — challenge progress) | **live on `main`** |
| `POST /api/v1/quests/progress` §6.2 `{exercise_id, duration_seconds, user_answers?}` → `{daily_seconds_spent, daily_minutes_spent, is_target_met, pet_health, streak_count}`; 400 `invalid_request`, 404 `exercise_not_found` | `useQuestStore.complete` → `usePetStore.applyProgress` | `/learn/:id` (7.3) | **live on `main`** |
| `GET /api/v1/pet/status` §6.3 → `{plant_name, health_points, stage, current_streak, last_practiced_at}` | `usePetStore.load` | `/` (7.2), `/revive` (7.5) | **contract-wired; backend merged to branch, unreviewed** — shape confirmed in `.worktrees/pet-health-streak-and-stage-engine-with-revive/backend/internal/pet/handler.go` |
| `POST /api/v1/pet/revive` §6.3 `{answers}` → `{revival_passed, pet_state}`; 409 `pet_not_wilted` | `usePetStore.revive` | `/revive` (7.5) | **contract-wired; backend on branch, unreviewed** |
| `GET /api/v1/onboarding/quiz` (onboarding plan, not in spec) → `{questions[{id, prompt, options}]}` | `useOnboardingApi().quiz` | `/onboarding` (7.1 lower) | **stub** — `NUXT_PUBLIC_STUB_ONBOARDING=true` serves `stubs/onboarding.ts`; flip to `false` when the onboarding slice merges |
| `POST /api/v1/onboarding/assessment` §6.1 → 201 `{status, assessed_level, roadmap_id, pet_state}` | `useOnboardingApi().assess` | `/onboarding` (7.1 lower) | **stub** — same flag |
| `POST /api/v1/settings/notifications` §6.4, `POST /api/v1/integrations/google/sync` §6.4 | — | `/settings` placeholder | **unwired** — two disabled buttons so notify (7) and google (6) attach here |

All error bodies are `{"error": "<code>"}` (every merged handler uses `gin.H{"error": ...}`); `ApiError.code` carries the string.

## File structure

| Path | Responsibility |
| --- | --- |
| `frontend/package.json`, `package-lock.json`, `nuxt.config.ts`, `tsconfig.json`, `tailwind.config.ts`, `eslint.config.mjs`, `vitest.config.ts`, `playwright.config.ts`, `.env.example`, `app.vue`, `assets/css/main.css`, `public/logo.svg` | scaffold; `tailwind.config.ts` exports the §6.1 `tokens` |
| `utils/apiClient.ts` | pure fetch wrapper: bearer header, `{error}` envelope → `ApiError`, 401 hook |
| `utils/googleAuth.ts` | consent URL mirroring `backend/internal/auth/scopes.go`; random `state` |
| `utils/session.ts` | clears the service worker's API cache on sign-out |
| `utils/progress.ts` | 30-minute target math: segment fills, minutes, percent, `clampDuration`, `formatCountdown` |
| `utils/plant.ts` | stage normalisation, health tone, speech line, `daysSince` |
| `utils/roadmap.ts` | 28 nodes from `day_number` |
| `composables/useApi.ts` | the one `ApiClient` instance bound to runtime config + auth store |
| `composables/useOnboardingApi.ts` + `stubs/onboarding.ts` | quiz/assessment against real or stub |
| `stores/auth.ts`, `stores/quest.ts`, `stores/pet.ts` | the §4 stores |
| `middleware/auth.global.ts` | guard every route but `/login` |
| `components/ui/{AppButton,AppCard,StateBlock,SegmentedProgress,HealthBar}.vue` | design system (design §1, §4) |
| `components/plant/{PlantSvg,SpeechBubble}.vue` | the plant (design §3) |
| `components/{AppHeader,quest/QuestRow,learn/CountdownTimer,learn/ContentViewer,roadmap/RoadmapNode,onboarding/GoalCard}.vue` | screen parts (design §4) |
| `pages/{login,index,roadmap,revive,onboarding,settings}.vue`, `pages/learn/[id].vue` | the six screens + placeholder |
| `service-worker/sw.ts`, `service-worker/push.ts` | §3 caching + push handlers |
| `tests/unit/*.test.ts` | Vitest (no Nuxt runtime) |
| `tests/e2e/login.spec.ts` | Playwright smoke against `page.route` stubs |
| `.github/workflows/ci.yml` | new `frontend` job |
| `harness/CODEMAP.md`, `.gitignore` | `frontend` paragraph; `!.env.example` |

---

## Tasks

### Task 1: Scaffold, tokens, tooling

**Files:**
- Create: `frontend/package.json`, `frontend/nuxt.config.ts`, `frontend/tsconfig.json`, `frontend/tailwind.config.ts`, `frontend/eslint.config.mjs`, `frontend/vitest.config.ts`, `frontend/.env.example`, `frontend/app.vue`, `frontend/assets/css/main.css`, `frontend/public/logo.svg`
- Modify: `.gitignore` (repo root)
- Test: `frontend/tests/unit/tokens.test.ts`

- [ ] **Step 1: `package.json`**

```json
{
  "name": "aelp-frontend",
  "private": true,
  "type": "module",
  "engines": { "node": ">=20" },
  "scripts": {
    "dev": "nuxi dev",
    "build": "nuxi build",
    "preview": "nuxi preview",
    "postinstall": "nuxi prepare",
    "lint": "eslint .",
    "typecheck": "nuxi typecheck",
    "test:unit": "vitest run",
    "test:e2e": "playwright test",
    "pwa:assets": "pwa-assets-generator"
  },
  "dependencies": {
    "@fontsource-variable/fraunces": "^5.2.0",
    "@fontsource/source-sans-3": "^5.2.0",
    "@pinia/nuxt": "^0.11.0",
    "@vite-pwa/nuxt": "^1.0.0",
    "nuxt": "^3.17.0",
    "pinia": "^3.0.0",
    "vue": "^3.5.0",
    "vue-router": "^4.5.0",
    "workbox-precaching": "^7.3.0",
    "workbox-routing": "^7.3.0",
    "workbox-strategies": "^7.3.0"
  },
  "devDependencies": {
    "@nuxt/eslint": "^1.0.0",
    "@nuxtjs/tailwindcss": "^6.13.0",
    "@playwright/test": "^1.50.0",
    "@vite-pwa/assets-generator": "^0.2.6",
    "@vitejs/plugin-vue": "^5.2.0",
    "@vue/test-utils": "^2.4.6",
    "eslint": "^9.20.0",
    "happy-dom": "^17.0.0",
    "typescript": "^5.7.0",
    "vitest": "^3.0.0",
    "vue-tsc": "^2.2.0"
  }
}
```

- [ ] **Step 2: `nuxt.config.ts`**

```ts
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  // Client-rendered PWA: the JWT lives in localStorage and every screen is
  // per-user, so there is nothing to render on a server (design §5).
  ssr: false,
  devtools: { enabled: false },
  modules: ['@pinia/nuxt', '@nuxtjs/tailwindcss', '@nuxt/eslint', '@vite-pwa/nuxt'],
  components: [{ path: '~/components', pathPrefix: false }],
  css: [
    '@fontsource-variable/fraunces/index.css',
    '@fontsource/source-sans-3/400.css',
    '@fontsource/source-sans-3/600.css',
    '~/assets/css/main.css',
  ],
  app: {
    head: {
      title: 'Học 30 phút',
      htmlAttrs: { lang: 'vi' },
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1, viewport-fit=cover' },
        { name: 'theme-color', content: '#10B981' },
      ],
    },
  },
  // NUXT_PUBLIC_API_BASE, NUXT_PUBLIC_GOOGLE_CLIENT_ID, NUXT_PUBLIC_VAPID_PUBLIC_KEY,
  // NUXT_PUBLIC_STUB_ONBOARDING override these at runtime (Nuxt convention;
  // the idea's bare API_BASE/GOOGLE_CLIENT_ID/VAPID_PUBLIC_KEY names are the
  // same values under the NUXT_PUBLIC_ prefix).
  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8080',
      googleClientId: '',
      vapidPublicKey: '',
      stubOnboarding: 'true',
    },
  },
  pwa: {
    strategies: 'injectManifest',
    srcDir: 'service-worker',
    filename: 'sw.ts',
    registerType: 'autoUpdate',
    injectRegister: 'auto',
    manifest: {
      name: 'Học 30 phút',
      short_name: 'Học30',
      description: 'Học tiếng Anh 30 phút mỗi ngày và nuôi một cái cây.',
      lang: 'vi',
      display: 'standalone',
      orientation: 'portrait',
      start_url: '/',
      scope: '/',
      theme_color: '#10B981',
      background_color: '#F8FAFC',
      icons: [
        { src: 'pwa-64x64.png', sizes: '64x64', type: 'image/png' },
        { src: 'pwa-192x192.png', sizes: '192x192', type: 'image/png' },
        { src: 'pwa-512x512.png', sizes: '512x512', type: 'image/png' },
        { src: 'maskable-icon-512x512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
      ],
    },
    injectManifest: { globPatterns: ['**/*.{js,css,html,svg,png,woff2}'] },
    client: { installPrompt: true },
    devOptions: { enabled: false },
  },
  typescript: { strict: true },
})
```

If the backend's default `PORT` (see `backend/internal/config`) is not 8080, use that value for `apiBase` and in `.env.example`.

- [ ] **Step 3: `tsconfig.json`, `eslint.config.mjs`, `vitest.config.ts`**

`tsconfig.json`:
```json
{
  "extends": "./.nuxt/tsconfig.json",
  "exclude": ["node_modules", ".output", "dist", "service-worker/sw.ts", "tests/e2e"]
}
```
(`sw.ts` uses the `webworker` lib and is compiled by the PWA module's own Vite build; excluding it keeps `nuxi typecheck` on the app.)

`eslint.config.mjs`:
```js
import withNuxt from './.nuxt/eslint.config.mjs'

export default withNuxt({
  ignores: ['.output/**', '.nuxt/**', 'dist/**', 'dev-dist/**', 'public/**'],
})
```

`vitest.config.ts` — plain Vitest, no Nuxt runtime; stores and utils import everything explicitly so they run here:
```ts
import { fileURLToPath } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

const root = fileURLToPath(new URL('.', import.meta.url))

export default defineConfig({
  plugins: [vue()],
  resolve: { alias: { '~': root, '@': root } },
  test: {
    environment: 'happy-dom',
    include: ['tests/unit/**/*.test.ts'],
    restoreMocks: true,
  },
})
```

- [ ] **Step 4: `tailwind.config.ts`, `assets/css/main.css`, `app.vue`, `public/logo.svg`, `.env.example`, root `.gitignore`**

`tailwind.config.ts` (design §1 — the only file that may contain a hex colour):
```ts
import type { Config } from 'tailwindcss'

/** Frontend spec §6.1 palette plus derived neutrals (design §1). */
export const tokens = {
  growth: '#10B981',
  streak: '#F59E0B',
  alert: '#EF4444',
  ink: '#1E293B',
  paper: '#F8FAFC',
  'paper-dark': '#0F172A',
  mute: '#64748B',
} as const

export default {
  darkMode: 'media',
  theme: {
    extend: {
      colors: tokens,
      fontFamily: {
        display: ['"Fraunces Variable"', 'Georgia', 'serif'],
        body: ['"Source Sans 3"', 'system-ui', 'sans-serif'],
      },
      borderRadius: { card: '16px', btn: '12px' },
    },
  },
} satisfies Config
```

`assets/css/main.css`:
```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  html {
    @apply bg-paper font-body text-ink antialiased;
  }
  @media (prefers-color-scheme: dark) {
    html {
      @apply bg-paper-dark text-paper;
    }
  }
  :focus-visible {
    @apply outline-none ring-2 ring-growth ring-offset-2;
  }
}
```

`app.vue`:
```vue
<template>
  <div class="min-h-screen">
    <NuxtPwaManifest />
    <NuxtPage />
  </div>
</template>
```

`public/logo.svg` — the `sprout` mark the PWA icons are generated from (Task 13):
```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
  <rect width="512" height="512" rx="96" fill="#F8FAFC"/>
  <path d="M176 352h160l-16 64H192z" fill="#1E293B"/>
  <path d="M256 352V232" stroke="#10B981" stroke-width="22" stroke-linecap="round"/>
  <path d="M256 250c-70 0-100-50-100-90 60 0 100 30 100 90z" fill="#10B981"/>
  <path d="M256 220c70 0 100-50 100-90-60 0-100 30-100 90z" fill="#10B981"/>
</svg>
```

`.env.example`:
```sh
# Copy to .env for `npm run dev`. All values are public (baked into the client).
NUXT_PUBLIC_API_BASE=http://localhost:8080
NUXT_PUBLIC_GOOGLE_CLIENT_ID=
NUXT_PUBLIC_VAPID_PUBLIC_KEY=
# true = /onboarding renders against stubs/onboarding.ts until the onboarding slice lands
NUXT_PUBLIC_STUB_ONBOARDING=true
```

Root `.gitignore`: add one line after `.env*`:
```
!.env.example
```
(`backend/.env.example` is tracked only because it was force-added; this makes both examples tracked by rule.)

- [ ] **Step 5: Install and generate**

```sh
npm install
```
Expected: `package-lock.json` created, `.nuxt/` generated by `postinstall` (`nuxi prepare`). Commit the lockfile. If any range fails to resolve, apply the *Version ranges are floors* rule above.

- [ ] **Step 6: Write the failing token test**

`tests/unit/tokens.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { tokens } from '~/tailwind.config'

describe('design tokens (Frontend spec §6.1)', () => {
  it('registers the four spec colours under their token names', () => {
    expect(tokens.growth).toBe('#10B981')
    expect(tokens.streak).toBe('#F59E0B')
    expect(tokens.alert).toBe('#EF4444')
    expect(tokens.ink).toBe('#1E293B')
  })
})
```

- [ ] **Step 7: Run it**

```sh
npx vitest run tests/unit/tokens.test.ts
```
Expected: PASS (the config was written in Step 4; this test pins it). Then:
```sh
npm run lint && npm run typecheck && npm run build
```
Expected: lint clean, typecheck clean, build ends with the Nitro output summary and a `PWA` line from `vite-plugin-pwa`. The `injectManifest` strategy needs `service-worker/sw.ts` to exist, so create this minimal one now (Task 13 replaces it) and the build is green at every commit:
```ts
/// <reference lib="webworker" />
import { precacheAndRoute } from 'workbox-precaching'

declare let self: ServiceWorkerGlobalScope

precacheAndRoute(self.__WB_MANIFEST)
```
Task 13 replaces it.

- [ ] **Step 8: Commit**

```sh
cd .. && git add .gitignore frontend/package.json frontend/package-lock.json frontend/nuxt.config.ts frontend/tsconfig.json frontend/tailwind.config.ts frontend/eslint.config.mjs frontend/vitest.config.ts frontend/.env.example frontend/app.vue frontend/assets/css/main.css frontend/public/logo.svg frontend/service-worker/sw.ts frontend/tests/unit/tokens.test.ts && git commit -m "frontend: Nuxt 3 scaffold with §6.1 tokens, Pinia, Tailwind, PWA module and Vitest"
```

### Task 2: The API client

**Files:**
- Create: `frontend/utils/apiClient.ts`
- Test: `frontend/tests/unit/apiClient.test.ts`

A pure module: no Nuxt imports, `fetch` injectable, so the bearer header, the `{error}` envelope and the 401 hook are unit-tested without a browser or a backend. The idea said "`$fetch` wrapper"; this uses the platform `fetch` for the same behaviour with one fewer layer to mock (recorded in Notes).

- [ ] **Step 1: Write the failing tests**

`tests/unit/apiClient.test.ts`:
```ts
import { describe, expect, it, vi } from 'vitest'
import { ApiError, createApiClient } from '~/utils/apiClient'

function jsonResponse(status: number, body: unknown) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function setup(response: Response, token: string | null = 'jwt-1') {
  const fetchMock = vi.fn(async () => response)
  const onUnauthorized = vi.fn()
  const api = createApiClient({
    baseURL: 'http://api.test',
    getToken: () => token,
    onUnauthorized,
    fetch: fetchMock as unknown as typeof fetch,
  })
  return { api, fetchMock, onUnauthorized }
}

describe('createApiClient', () => {
  it('prefixes the base URL and sends Authorization: Bearer <jwt>', async () => {
    const { api, fetchMock } = setup(jsonResponse(200, { ok: true }))
    await expect(api.get('/api/v1/quests/daily')).resolves.toEqual({ ok: true })
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('http://api.test/api/v1/quests/daily')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer jwt-1')
    expect(init.method).toBe('GET')
  })

  it('omits the Authorization header when there is no token', async () => {
    const { api, fetchMock } = setup(jsonResponse(200, {}), null)
    await api.post('/api/v1/auth/google', { code: 'c', redirect_uri: 'r' })
    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    const headers = init.headers as Record<string, string>
    expect(headers.Authorization).toBeUndefined()
    expect(headers['Content-Type']).toBe('application/json')
    expect(init.body).toBe(JSON.stringify({ code: 'c', redirect_uri: 'r' }))
  })

  it('calls onUnauthorized and throws ApiError(401) on a 401', async () => {
    const { api, onUnauthorized } = setup(jsonResponse(401, { error: 'unauthorized' }))
    await expect(api.get('/api/v1/pet/status')).rejects.toMatchObject({ status: 401, code: 'unauthorized' })
    expect(onUnauthorized).toHaveBeenCalledTimes(1)
  })

  it('maps the {error} envelope to ApiError.code', async () => {
    const { api, onUnauthorized } = setup(jsonResponse(404, { error: 'no_active_roadmap' }))
    const err = await api.get('/api/v1/quests/daily').catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ status: 404, code: 'no_active_roadmap' })
    expect(onUnauthorized).not.toHaveBeenCalled()
  })

  it('falls back to unknown_error when the error body is not JSON', async () => {
    const { api } = setup(new Response('<html>502</html>', { status: 502 }))
    await expect(api.get('/healthz')).rejects.toMatchObject({ status: 502, code: 'unknown_error' })
  })
})
```

- [ ] **Step 2: Run and confirm failure**

```sh
npx vitest run tests/unit/apiClient.test.ts
```
Expected: FAIL — `Failed to resolve import "~/utils/apiClient"`.

- [ ] **Step 3: Implement**

`utils/apiClient.ts`:
```ts
/**
 * The one HTTP client. Every backend handler answers errors as
 * {"error": "<code>"}; that code becomes ApiError.code so screens can branch
 * on `no_active_roadmap`, `exercise_not_found`, `pet_not_wilted`, ... without
 * parsing bodies themselves.
 */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
  ) {
    super(`${status} ${code}`)
    this.name = 'ApiError'
  }
}

export interface ApiClientOptions {
  baseURL: string
  getToken: () => string | null
  /** Called once per 401 before the ApiError is thrown (sign out + redirect). */
  onUnauthorized: () => void
  fetch?: typeof globalThis.fetch
}

export interface ApiClient {
  get<T>(path: string): Promise<T>
  post<T>(path: string, body?: unknown): Promise<T>
}

async function errorCode(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as { error?: unknown }
    return typeof body.error === 'string' ? body.error : 'unknown_error'
  } catch {
    return 'unknown_error'
  }
}

export function createApiClient(opts: ApiClientOptions): ApiClient {
  const doFetch = opts.fetch ?? globalThis.fetch

  async function request<T>(method: 'GET' | 'POST', path: string, body?: unknown): Promise<T> {
    const headers: Record<string, string> = { Accept: 'application/json' }
    const token = opts.getToken()
    if (token) headers.Authorization = `Bearer ${token}`
    if (body !== undefined) headers['Content-Type'] = 'application/json'

    const res = await doFetch(`${opts.baseURL}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    })

    if (res.status === 401) {
      opts.onUnauthorized()
      throw new ApiError(401, 'unauthorized')
    }
    if (!res.ok) throw new ApiError(res.status, await errorCode(res))
    return (await res.json()) as T
  }

  return {
    get: <T>(path: string) => request<T>('GET', path),
    post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  }
}
```

- [ ] **Step 4: Run and confirm pass**

```sh
npx vitest run tests/unit/apiClient.test.ts
```
Expected: 5 passed.

- [ ] **Step 5: Commit**

```sh
cd .. && git add frontend/utils/apiClient.ts frontend/tests/unit/apiClient.test.ts && git commit -m "frontend: pure API client — bearer header, {error} envelope, 401 hook"
```

### Task 3: `useAuthStore`, the Google consent URL, `useApi`, the route guard

**Files:**
- Create: `frontend/utils/googleAuth.ts`, `frontend/utils/session.ts`, `frontend/stores/auth.ts`, `frontend/composables/useApi.ts`, `frontend/middleware/auth.global.ts`
- Test: `frontend/tests/unit/googleAuth.test.ts`, `frontend/tests/unit/authStore.test.ts`

The store holds the §6.1 response **as the spec shapes it** (`access_token`, `expires_in`, `user`) and persists `{accessToken, expiresAt, user}` to `localStorage` under `aelp.auth`. The consent URL mirrors `backend/internal/auth/scopes.go` (CODEMAP: "the frontend must build its login URL from `auth.AuthCodeURL`" — there is no endpoint exposing it, so the list is duplicated here and pinned by a test that names the file).

- [ ] **Step 1: Write the failing tests**

`tests/unit/googleAuth.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { GOOGLE_SCOPES, googleAuthUrl, randomState } from '~/utils/googleAuth'

describe('googleAuthUrl (mirrors backend/internal/auth/scopes.go)', () => {
  it('builds the consent URL with the backend scope list, offline access and consent prompt', () => {
    const u = new URL(googleAuthUrl('client-1', 'http://localhost:3000/login', 'state-1'))
    expect(u.origin + u.pathname).toBe('https://accounts.google.com/o/oauth2/v2/auth')
    expect(u.searchParams.get('client_id')).toBe('client-1')
    expect(u.searchParams.get('redirect_uri')).toBe('http://localhost:3000/login')
    expect(u.searchParams.get('response_type')).toBe('code')
    expect(u.searchParams.get('access_type')).toBe('offline')
    expect(u.searchParams.get('prompt')).toBe('consent')
    expect(u.searchParams.get('state')).toBe('state-1')
    expect(u.searchParams.get('scope')).toBe(GOOGLE_SCOPES.join(' '))
  })

  it('requests calendar.events and tasks at first sign-in so the google slice never forces re-consent', () => {
    expect(GOOGLE_SCOPES).toEqual([
      'openid',
      'email',
      'profile',
      'https://www.googleapis.com/auth/calendar.events',
      'https://www.googleapis.com/auth/tasks',
    ])
  })

  it('randomState is 32 hex chars and not repeated', () => {
    const a = randomState()
    expect(a).toMatch(/^[0-9a-f]{32}$/)
    expect(randomState()).not.toBe(a)
  })
})
```

`tests/unit/authStore.test.ts`:
```ts
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { AUTH_STORAGE_KEY, useAuthStore, type SignInResponse } from '~/stores/auth'

const spec61: SignInResponse = {
  access_token: 'eyJ.test',
  token_type: 'Bearer',
  expires_in: 86400,
  user: { id: 'u1', email: 'user@example.com', full_name: 'Nguyen Hendrix', cefr_current: 'B1' },
}

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('signIn stores the §6.1 access_token and computes expiresAt from expires_in', () => {
    const auth = useAuthStore()
    auth.signIn(spec61, 1_000_000)
    expect(auth.accessToken).toBe('eyJ.test')
    expect(auth.expiresAt).toBe(1_000_000 + 86400 * 1000)
    expect(auth.user?.full_name).toBe('Nguyen Hendrix')
    expect(JSON.parse(localStorage.getItem(AUTH_STORAGE_KEY) ?? '{}')).toEqual({
      accessToken: 'eyJ.test',
      expiresAt: 1_000_000 + 86400 * 1000,
      user: spec61.user,
    })
  })

  it('does not accept the pre-spec {token} shape', () => {
    const auth = useAuthStore()
    // The merged handler's shape; the frontend targets the spec, not the handler.
    auth.signIn({ token: 'legacy', user: spec61.user } as unknown as SignInResponse, 0)
    expect(auth.accessToken).toBeNull()
    expect(auth.isAuthenticated).toBe(false)
  })

  it('hydrate restores a persisted session and isAuthenticated respects expiry', () => {
    localStorage.setItem(
      AUTH_STORAGE_KEY,
      JSON.stringify({ accessToken: 't', expiresAt: Date.now() + 60_000, user: spec61.user }),
    )
    const auth = useAuthStore()
    auth.hydrate()
    expect(auth.isAuthenticated).toBe(true)

    localStorage.setItem(
      AUTH_STORAGE_KEY,
      JSON.stringify({ accessToken: 't', expiresAt: Date.now() - 1, user: spec61.user }),
    )
    setActivePinia(createPinia())
    const expired = useAuthStore()
    expired.hydrate()
    expect(expired.isAuthenticated).toBe(false)
  })

  it('hydrate tolerates garbage in storage', () => {
    localStorage.setItem(AUTH_STORAGE_KEY, '{not json')
    const auth = useAuthStore()
    auth.hydrate()
    expect(auth.hydrated).toBe(true)
    expect(auth.isAuthenticated).toBe(false)
  })

  it('signOut clears state and storage (what the 401 hook calls)', () => {
    const auth = useAuthStore()
    auth.signIn(spec61, Date.now())
    auth.signOut()
    expect(auth.accessToken).toBeNull()
    expect(auth.user).toBeNull()
    expect(localStorage.getItem(AUTH_STORAGE_KEY)).toBeNull()
  })
})
```

- [ ] **Step 2: Run and confirm failure**

```sh
npx vitest run tests/unit/googleAuth.test.ts tests/unit/authStore.test.ts
```
Expected: FAIL — unresolved imports.

- [ ] **Step 3: Implement**

`utils/googleAuth.ts`:
```ts
/** Mirrors backend/internal/auth/scopes.go — keep the two lists identical. */
export const GOOGLE_AUTH_ENDPOINT = 'https://accounts.google.com/o/oauth2/v2/auth'

export const GOOGLE_SCOPES = [
  'openid',
  'email',
  'profile',
  'https://www.googleapis.com/auth/calendar.events',
  'https://www.googleapis.com/auth/tasks',
] as const

export function googleAuthUrl(clientId: string, redirectUri: string, state: string): string {
  const q = new URLSearchParams({
    client_id: clientId,
    redirect_uri: redirectUri,
    response_type: 'code',
    scope: GOOGLE_SCOPES.join(' '),
    access_type: 'offline',
    prompt: 'consent',
    state,
  })
  return `${GOOGLE_AUTH_ENDPOINT}?${q.toString()}`
}

export function randomState(): string {
  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  return Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('')
}
```

`utils/session.ts`:
```ts
/** Name of the service worker's NetworkFirst cache for /quests/daily and /pet/status (sw.ts). */
export const API_STATE_CACHE = 'api-state'

/** Drop cached per-user API responses; called on sign-out and on a 401. */
export async function clearApiCache(): Promise<void> {
  if (typeof caches === 'undefined') return
  await caches.delete(API_STATE_CACHE)
}
```

`stores/auth.ts`:
```ts
import { defineStore } from 'pinia'

/** Backend spec §6.1 `user`. */
export interface AuthUser {
  id: string
  email: string
  full_name: string
  cefr_current: string | null
}

/** Backend spec §6.1 `POST /api/v1/auth/google` 200 body — the contract, not the merged handler. */
export interface SignInResponse {
  access_token: string
  token_type: string
  expires_in: number
  user: AuthUser
}

interface Persisted {
  accessToken: string
  expiresAt: number
  user: AuthUser
}

export const AUTH_STORAGE_KEY = 'aelp.auth'

function storageOrNull(): Storage | null {
  return typeof localStorage === 'undefined' ? null : localStorage
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    accessToken: null as string | null,
    expiresAt: null as number | null,
    user: null as AuthUser | null,
    hydrated: false,
  }),
  getters: {
    isAuthenticated: s => s.accessToken !== null && s.expiresAt !== null && s.expiresAt > Date.now(),
    initial: s => (s.user?.full_name?.trim().charAt(0) || '?').toUpperCase(),
  },
  actions: {
    hydrate() {
      this.hydrated = true
      const raw = storageOrNull()?.getItem(AUTH_STORAGE_KEY)
      if (!raw) return
      try {
        const p = JSON.parse(raw) as Partial<Persisted>
        if (typeof p.accessToken === 'string' && typeof p.expiresAt === 'number' && p.user) {
          this.accessToken = p.accessToken
          this.expiresAt = p.expiresAt
          this.user = p.user
        }
      } catch {
        storageOrNull()?.removeItem(AUTH_STORAGE_KEY)
      }
    },
    signIn(res: SignInResponse, now: number = Date.now()) {
      if (typeof res.access_token !== 'string' || typeof res.expires_in !== 'number') return
      this.accessToken = res.access_token
      this.expiresAt = now + res.expires_in * 1000
      this.user = res.user
      this.hydrated = true
      const p: Persisted = { accessToken: this.accessToken, expiresAt: this.expiresAt, user: res.user }
      storageOrNull()?.setItem(AUTH_STORAGE_KEY, JSON.stringify(p))
    },
    signOut() {
      this.accessToken = null
      this.expiresAt = null
      this.user = null
      storageOrNull()?.removeItem(AUTH_STORAGE_KEY)
    },
  },
})
```

`composables/useApi.ts`:
```ts
import { navigateTo, useRuntimeConfig } from '#app'
import { useAuthStore } from '~/stores/auth'
import { createApiClient, type ApiClient } from '~/utils/apiClient'
import { clearApiCache } from '~/utils/session'

let client: ApiClient | null = null

/** The app's single API client: bearer from useAuthStore, sign-out + /login on 401. */
export function useApi(): ApiClient {
  if (client) return client
  const config = useRuntimeConfig()
  const auth = useAuthStore()
  client = createApiClient({
    baseURL: config.public.apiBase,
    getToken: () => auth.accessToken,
    onUnauthorized: () => {
      auth.signOut()
      void clearApiCache()
      void navigateTo('/login')
    },
  })
  return client
}
```

`middleware/auth.global.ts`:
```ts
import { useAuthStore } from '~/stores/auth'

export default defineNuxtRouteMiddleware((to) => {
  const auth = useAuthStore()
  if (!auth.hydrated) auth.hydrate()

  if (to.path === '/login') {
    // A signed-in user has no business on /login unless Google just sent them back.
    if (auth.isAuthenticated && !to.query.code) return navigateTo('/', { replace: true })
    return
  }
  if (!auth.isAuthenticated) {
    auth.signOut()
    return navigateTo('/login', { replace: true })
  }
})
```

- [ ] **Step 4: Run and confirm pass**

```sh
npx vitest run tests/unit/googleAuth.test.ts tests/unit/authStore.test.ts && npm run lint && npm run typecheck
```
Expected: 8 passed; lint and typecheck clean.

- [ ] **Step 5: Commit**

```sh
cd .. && git add frontend/utils/googleAuth.ts frontend/utils/session.ts frontend/stores/auth.ts frontend/composables/useApi.ts frontend/middleware/auth.global.ts frontend/tests/unit/googleAuth.test.ts frontend/tests/unit/authStore.test.ts && git commit -m "frontend: useAuthStore against the §6.1 shape, Google consent URL, useApi, route guard"
```

### Task 4: Pure helpers — the 30-minute target, the plant, the roadmap

**Files:**
- Create: `frontend/utils/progress.ts`, `frontend/utils/plant.ts`, `frontend/utils/roadmap.ts`
- Test: `frontend/tests/unit/progress.test.ts`, `frontend/tests/unit/plant.test.ts`, `frontend/tests/unit/roadmap.test.ts`

Design §1 (the three-segment day), §3 (plant states and speech), §2.5 (node states). Thresholds match the merged engine (`.worktrees/pet-.../backend/internal/pet/engine.go` `StageFor`) and `quests.MaxDurationSeconds` (3600).

- [ ] **Step 1: Write the failing tests**

`tests/unit/progress.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import {
  DAILY_TARGET_SECONDS,
  SEGMENT_COUNT,
  SEGMENT_SECONDS,
  clampDuration,
  formatCountdown,
  minutesOf,
  percentOf,
  segmentFills,
} from '~/utils/progress'

describe('the 30-minute target (design §1)', () => {
  it('is three segments of ten minutes', () => {
    expect(SEGMENT_COUNT).toBe(3)
    expect(SEGMENT_SECONDS).toBe(600)
    expect(DAILY_TARGET_SECONDS).toBe(1800)
  })

  it('fills segments left to right', () => {
    expect(segmentFills(0)).toEqual([0, 0, 0])
    expect(segmentFills(300)).toEqual([0.5, 0, 0])
    expect(segmentFills(1200)).toEqual([1, 1, 0])
    expect(segmentFills(1500)).toEqual([1, 1, 0.5])
    expect(segmentFills(4000)).toEqual([1, 1, 1])
  })

  it('supports the 15-minute single-segment revive bar', () => {
    expect(segmentFills(450, 900, 1)).toEqual([0.5])
  })

  it('labels minutes and percent the way wireframe 7.2 does (20 / 30, 66%)', () => {
    expect(minutesOf(1200)).toBe(20)
    expect(minutesOf(1259)).toBe(20)
    expect(percentOf(1200)).toBe(66)
    expect(percentOf(1800)).toBe(100)
    expect(percentOf(5000)).toBe(100)
  })

  it('clamps a posted duration to the backend range 1..3600', () => {
    expect(clampDuration(0)).toBe(1)
    expect(clampDuration(-5)).toBe(1)
    expect(clampDuration(599.7)).toBe(599)
    expect(clampDuration(99_999)).toBe(3600)
  })

  it('formats a countdown as mm:ss', () => {
    expect(formatCountdown(582)).toBe('09:42')
    expect(formatCountdown(0)).toBe('00:00')
    expect(formatCountdown(3600)).toBe('60:00')
  })
})
```

`tests/unit/plant.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { PLANT_STAGES, daysSince, healthTone, normalizeStage, speechLine } from '~/utils/plant'

describe('plant helpers (design §3)', () => {
  it('knows the six DDL stages', () => {
    expect(PLANT_STAGES).toEqual(['seed', 'sprout', 'sapling', 'flowering', 'fruitful', 'wilted'])
  })

  it('normalises unknown stages to sprout and flags them', () => {
    expect(normalizeStage('fruitful')).toEqual({ stage: 'fruitful', known: true })
    expect(normalizeStage('cactus')).toEqual({ stage: 'sprout', known: false })
    expect(normalizeStage(undefined)).toEqual({ stage: 'sprout', known: false })
  })

  it('tones health: growth ≥ 60, streak ≥ 30, alert below', () => {
    expect(healthTone(100)).toBe('growth')
    expect(healthTone(60)).toBe('growth')
    expect(healthTone(59)).toBe('streak')
    expect(healthTone(30)).toBe('streak')
    expect(healthTone(29)).toBe('alert')
    expect(healthTone(0)).toBe('alert')
  })

  it('speaks the wireframe 7.2 line when healthy and the target is not met', () => {
    expect(speechLine({ stage: 'sprout', health: 80, targetMet: false })).toBe('Tưới cho tớ 10 phút học đi!')
    expect(speechLine({ stage: 'sapling', health: 45, targetMet: false })).toBe('Tớ hơi khát rồi… 10 phút thôi?')
    expect(speechLine({ stage: 'sapling', health: 10, targetMet: false })).toBe('Tớ sắp héo mất! Học một chút nhé?')
    expect(speechLine({ stage: 'sapling', health: 10, targetMet: true })).toBe('Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿')
    expect(speechLine({ stage: 'wilted', health: 0, targetMet: false })).toBe('…')
  })

  it('counts whole days since last practice, or null when unknown', () => {
    const now = new Date('2026-09-23T10:00:00Z')
    expect(daysSince('2026-09-21T20:15:00Z', now)).toBe(1)
    expect(daysSince('2026-09-20T09:00:00Z', now)).toBe(3)
    expect(daysSince(null, now)).toBeNull()
    expect(daysSince('not a date', now)).toBeNull()
  })
})
```

`tests/unit/roadmap.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { ROADMAP_DAYS, roadmapNodes } from '~/utils/roadmap'

describe('roadmapNodes (design §2.5)', () => {
  it('derives 28 nodes from day_number: before = completed, equal = today, after = locked', () => {
    const nodes = roadmapNodes(3)
    expect(nodes).toHaveLength(ROADMAP_DAYS)
    expect(nodes[0]).toEqual({ day: 1, week: 1, state: 'completed' })
    expect(nodes[2]).toEqual({ day: 3, week: 1, state: 'today' })
    expect(nodes[3]).toEqual({ day: 4, week: 1, state: 'locked' })
    expect(nodes[27]).toEqual({ day: 28, week: 4, state: 'locked' })
  })

  it('groups seven days to a week (airouter RoadmapSchema: 4 modules × 7 days)', () => {
    const nodes = roadmapNodes(1)
    expect(nodes[6].week).toBe(1)
    expect(nodes[7].week).toBe(2)
    expect(nodes[21].week).toBe(4)
  })

  it('clamps day_number into 1..28', () => {
    expect(roadmapNodes(0)[0].state).toBe('today')
    expect(roadmapNodes(99)[27].state).toBe('today')
  })
})
```

- [ ] **Step 2: Run and confirm failure**

```sh
npx vitest run tests/unit/progress.test.ts tests/unit/plant.test.ts tests/unit/roadmap.test.ts
```
Expected: FAIL — three unresolved imports.

- [ ] **Step 3: Implement**

`utils/progress.ts`:
```ts
/** Backend spec §6.2: 3 × 10-minute tasks make the 30-minute day. */
export const SEGMENT_COUNT = 3
export const SEGMENT_SECONDS = 600
export const DAILY_TARGET_SECONDS = SEGMENT_COUNT * SEGMENT_SECONDS

/** quests.MaxDurationSeconds — a POST /quests/progress body outside 1..3600 is a 400. */
export const MAX_DURATION_SECONDS = 3600

/** Fill ratio (0..1) of each segment, left to right (design §1). */
export function segmentFills(valueSeconds: number, segmentSeconds = SEGMENT_SECONDS, segments = SEGMENT_COUNT): number[] {
  const fills: number[] = []
  let left = Math.max(0, valueSeconds)
  for (let i = 0; i < segments; i++) {
    fills.push(Math.min(1, left / segmentSeconds))
    left = Math.max(0, left - segmentSeconds)
  }
  return fills
}

export function minutesOf(seconds: number): number {
  return Math.floor(Math.max(0, seconds) / 60)
}

export function percentOf(seconds: number, target = DAILY_TARGET_SECONDS): number {
  return Math.min(100, Math.floor((Math.max(0, seconds) / target) * 100))
}

export function clampDuration(seconds: number): number {
  return Math.min(MAX_DURATION_SECONDS, Math.max(1, Math.floor(seconds)))
}

export function formatCountdown(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds))
  const mm = String(Math.floor(s / 60)).padStart(2, '0')
  const ss = String(s % 60).padStart(2, '0')
  return `${mm}:${ss}`
}
```

`utils/plant.ts`:
```ts
/** The DDL `pet_stage` enum (0001_init.up.sql). */
export const PLANT_STAGES = ['seed', 'sprout', 'sapling', 'flowering', 'fruitful', 'wilted'] as const
export type PlantStage = (typeof PLANT_STAGES)[number]

export type Tone = 'growth' | 'streak' | 'alert'

export function normalizeStage(stage: string | undefined | null): { stage: PlantStage, known: boolean } {
  if (stage && (PLANT_STAGES as readonly string[]).includes(stage)) return { stage: stage as PlantStage, known: true }
  return { stage: 'sprout', known: false }
}

/** design §3: 100–60 growth, 59–30 streak, 29–0 alert. */
export function healthTone(health: number): Tone {
  if (health >= 60) return 'growth'
  if (health >= 30) return 'streak'
  return 'alert'
}

export function speechLine(o: { stage: string, health: number, targetMet: boolean }): string {
  if (o.stage === 'wilted' || o.health <= 0) return '…'
  if (o.targetMet) return 'Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿'
  if (o.health >= 60) return 'Tưới cho tớ 10 phút học đi!'
  if (o.health >= 30) return 'Tớ hơi khát rồi… 10 phút thôi?'
  return 'Tớ sắp héo mất! Học một chút nhé?'
}

/** Whole days between an ISO timestamp and now; null when absent or unparsable. */
export function daysSince(iso: string | null | undefined, now: Date = new Date()): number | null {
  if (!iso) return null
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return null
  return Math.max(0, Math.floor((now.getTime() - t) / 86_400_000))
}
```

`utils/roadmap.ts`:
```ts
export const ROADMAP_DAYS = 28

export type NodeState = 'completed' | 'today' | 'locked'
export interface RoadmapNode {
  day: number
  week: number
  state: NodeState
}

/**
 * design §2.5. There is no per-day completion endpoint, so "completed" means
 * "before today" — derived from GET /quests/daily day_number (open question).
 */
export function roadmapNodes(dayNumber: number, total = ROADMAP_DAYS): RoadmapNode[] {
  const today = Math.min(total, Math.max(1, Math.floor(dayNumber)))
  return Array.from({ length: total }, (_, i) => {
    const day = i + 1
    return {
      day,
      week: Math.floor(i / 7) + 1,
      state: day < today ? 'completed' : day === today ? 'today' : 'locked',
    }
  })
}
```

- [ ] **Step 4: Run and confirm pass**

```sh
npx vitest run tests/unit/progress.test.ts tests/unit/plant.test.ts tests/unit/roadmap.test.ts
```
Expected: 14 passed.

- [ ] **Step 5: Commit**

```sh
cd .. && git add frontend/utils/progress.ts frontend/utils/plant.ts frontend/utils/roadmap.ts frontend/tests/unit/progress.test.ts frontend/tests/unit/plant.test.ts frontend/tests/unit/roadmap.test.ts && git commit -m "frontend: pure helpers — three-segment day, plant stage/tone/speech, roadmap nodes"
```

### Task 5: `useQuestStore` and `usePetStore`

**Files:**
- Create: `frontend/stores/quest.ts`, `frontend/stores/pet.ts`
- Test: `frontend/tests/unit/questStore.test.ts`, `frontend/tests/unit/petStore.test.ts`

Frontend spec §4: `useQuestStore` — "active 10-minute/30-minute learning timers, active quest payload, task completions"; `usePetStore` — "stage transitions, health bar, streak counters". Both call `useApi()`; the tests replace that module with a fake, so no Nuxt runtime is needed.

- [ ] **Step 1: Write the failing tests**

`tests/unit/questStore.test.ts`:
```ts
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '~/utils/apiClient'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const { useQuestStore } = await import('~/stores/quest')

const daily = {
  date: '2026-09-23',
  day_number: 3,
  total_minutes_required: 30,
  accumulated_seconds: 600,
  is_target_met: false,
  tasks: [
    { id: 'ex-3', task_type: 'practice', title: 'Viết phản hồi', duration_minutes: 10, is_completed: false, content_json: {} },
    { id: 'ex-1', task_type: 'vocabulary', title: 'Từ vựng', duration_minutes: 10, is_completed: true, content_json: {} },
    { id: 'ex-2', task_type: 'reading', title: 'Đọc hiểu', duration_minutes: 10, is_completed: false, content_json: {} },
  ],
}

describe('useQuestStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    api.get.mockReset()
    api.post.mockReset()
  })

  it('load fetches GET /api/v1/quests/daily and orders tasks vocabulary → reading → practice', async () => {
    api.get.mockResolvedValue(daily)
    const q = useQuestStore()
    await q.load()
    expect(api.get).toHaveBeenCalledWith('/api/v1/quests/daily')
    expect(q.sortedTasks.map(t => t.id)).toEqual(['ex-1', 'ex-2', 'ex-3'])
    expect(q.nextTaskId).toBe('ex-2')
    expect(q.accumulatedSeconds).toBe(600)
    expect(q.noRoadmap).toBe(false)
    expect(q.loading).toBe(false)
  })

  it('load maps 404 no_active_roadmap to the empty state, other errors to error', async () => {
    api.get.mockRejectedValue(new ApiError(404, 'no_active_roadmap'))
    const q = useQuestStore()
    await q.load()
    expect(q.noRoadmap).toBe(true)
    expect(q.error).toBeNull()

    api.get.mockRejectedValue(new TypeError('Failed to fetch'))
    await q.load()
    expect(q.error).toBe('network_error')
  })

  it('timers count down from duration_minutes and report elapsed seconds', () => {
    const q = useQuestStore()
    q.startTimer('ex-2', 10)
    expect(q.timers['ex-2']?.remainingSeconds).toBe(600)
    q.tick('ex-2', 18)
    expect(q.timers['ex-2']?.remainingSeconds).toBe(582)
    expect(q.elapsedSeconds('ex-2')).toBe(18)
    q.tick('ex-2', 10_000)
    expect(q.timers['ex-2']?.remainingSeconds).toBe(0)
    q.startTimer('ex-2', 10) // re-entry resumes, never resets
    expect(q.elapsedSeconds('ex-2')).toBe(600)
  })

  it('complete posts the §6.2 body with a clamped duration and applies the response', async () => {
    api.get.mockResolvedValue(daily)
    api.post.mockResolvedValue({ daily_seconds_spent: 1200, daily_minutes_spent: 20, is_target_met: false, pet_health: 100, streak_count: 5 })
    const q = useQuestStore()
    await q.load()
    q.startTimer('ex-2', 10)
    const res = await q.complete('ex-2', 0, { q1: 'A' })
    expect(api.post).toHaveBeenCalledWith('/api/v1/quests/progress', {
      exercise_id: 'ex-2',
      duration_seconds: 1,
      user_answers: { q1: 'A' },
    })
    expect(res.pet_health).toBe(100)
    expect(q.accumulatedSeconds).toBe(1200)
    expect(q.taskById('ex-2')?.is_completed).toBe(true)
    expect(q.nextTaskId).toBe('ex-3')
    expect(q.timers['ex-2']).toBeUndefined()
  })

  it('complete omits user_answers when none were collected', async () => {
    api.post.mockResolvedValue({ daily_seconds_spent: 60, daily_minutes_spent: 1, is_target_met: false, pet_health: 100, streak_count: 0 })
    const q = useQuestStore()
    await q.complete('ex-2', 60)
    expect(api.post).toHaveBeenCalledWith('/api/v1/quests/progress', { exercise_id: 'ex-2', duration_seconds: 60 })
  })
})
```

`tests/unit/petStore.test.ts`:
```ts
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '~/utils/apiClient'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const { REVIVE_SECONDS, REVIVE_STORAGE_KEY, usePetStore } = await import('~/stores/pet')

const status = { plant_name: 'My Green Buddy', health_points: 80, stage: 'sprout', current_streak: 5, last_practiced_at: '2026-09-21T20:15:00Z' }

describe('usePetStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    api.get.mockReset()
    api.post.mockReset()
  })

  it('load fetches GET /api/v1/pet/status', async () => {
    api.get.mockResolvedValue(status)
    const pet = usePetStore()
    await pet.load()
    expect(api.get).toHaveBeenCalledWith('/api/v1/pet/status')
    expect(pet.status?.health_points).toBe(80)
    expect(pet.isWilted).toBe(false)
  })

  it('isWilted when health is 0 or stage is wilted', async () => {
    api.get.mockResolvedValue({ ...status, health_points: 0, stage: 'wilted' })
    const pet = usePetStore()
    await pet.load()
    expect(pet.isWilted).toBe(true)
  })

  it('applyProgress updates health and streak from a §6.2 progress response', async () => {
    api.get.mockResolvedValue(status)
    const pet = usePetStore()
    await pet.load()
    pet.applyProgress({ pet_health: 100, streak_count: 6 })
    expect(pet.status).toMatchObject({ health_points: 100, current_streak: 6 })
  })

  it('revive: first call starts a challenge anchored to the current daily seconds', async () => {
    api.get.mockResolvedValue({ ...status, health_points: 0, stage: 'wilted' })
    api.post.mockResolvedValue({ revival_passed: false, pet_state: { health_points: 0, stage: 'wilted', current_streak: 0 } })
    const pet = usePetStore()
    await pet.load()
    const res = await pet.revive('2026-09-23', 120)
    expect(api.post).toHaveBeenCalledWith('/api/v1/pet/revive', { answers: {} })
    expect(res?.revival_passed).toBe(false)
    expect(pet.challenge).toEqual({ date: '2026-09-23', startSeconds: 120 })
    expect(JSON.parse(localStorage.getItem(REVIVE_STORAGE_KEY) ?? 'null')).toEqual({ date: '2026-09-23', startSeconds: 120 })
    expect(pet.challengeProgress(120 + 450)).toBe(450)
    expect(REVIVE_SECONDS).toBe(900)
  })

  it('revive: a pass applies pet_state and clears the challenge', async () => {
    localStorage.setItem(REVIVE_STORAGE_KEY, JSON.stringify({ date: '2026-09-23', startSeconds: 0 }))
    api.get.mockResolvedValue({ ...status, health_points: 0, stage: 'wilted' })
    api.post.mockResolvedValue({ revival_passed: true, pet_state: { health_points: 50, stage: 'sprout', current_streak: 0 } })
    const pet = usePetStore()
    pet.hydrateChallenge()
    await pet.load()
    const res = await pet.revive('2026-09-23', 1000)
    expect(res?.revival_passed).toBe(true)
    expect(pet.status).toMatchObject({ health_points: 50, stage: 'sprout', current_streak: 0 })
    expect(pet.challenge).toBeNull()
    expect(localStorage.getItem(REVIVE_STORAGE_KEY)).toBeNull()
  })

  it('revive: 409 pet_not_wilted sets notWilted and returns null', async () => {
    api.post.mockRejectedValue(new ApiError(409, 'pet_not_wilted'))
    const pet = usePetStore()
    await expect(pet.revive('2026-09-23', 0)).resolves.toBeNull()
    expect(pet.notWilted).toBe(true)
  })
})
```

- [ ] **Step 2: Run and confirm failure**

```sh
npx vitest run tests/unit/questStore.test.ts tests/unit/petStore.test.ts
```
Expected: FAIL — unresolved `~/stores/quest`, `~/stores/pet`.

- [ ] **Step 3: Implement**

`stores/quest.ts`:
```ts
import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { ApiError } from '~/utils/apiClient'
import { clampDuration } from '~/utils/progress'

export const TASK_ORDER = ['vocabulary', 'reading', 'practice'] as const
export type TaskType = (typeof TASK_ORDER)[number]

/** Backend spec §6.2 GET /quests/daily task. content_json is the roadmap task object (airouter). */
export interface QuestTask {
  id: string
  task_type: TaskType | string
  title: string
  duration_minutes: number
  is_completed: boolean
  content_json: unknown
}

export interface DailyQuests {
  date: string
  day_number: number
  total_minutes_required: number
  accumulated_seconds: number
  is_target_met: boolean
  tasks: QuestTask[]
}

/** Backend spec §6.2 POST /quests/progress 200 body. */
export interface ProgressResponse {
  daily_seconds_spent: number
  daily_minutes_spent: number
  is_target_met: boolean
  pet_health: number
  streak_count: number
}

interface Timer {
  totalSeconds: number
  remainingSeconds: number
}

function rank(type: string): number {
  const i = (TASK_ORDER as readonly string[]).indexOf(type)
  return i === -1 ? TASK_ORDER.length : i
}

export const useQuestStore = defineStore('quest', {
  state: () => ({
    daily: null as DailyQuests | null,
    noRoadmap: false,
    loading: false,
    error: null as string | null,
    timers: {} as Record<string, Timer>,
  }),
  getters: {
    sortedTasks: (s): QuestTask[] => (s.daily ? [...s.daily.tasks].sort((a, b) => rank(a.task_type) - rank(b.task_type)) : []),
    nextTaskId(): string | null {
      return this.sortedTasks.find(t => !t.is_completed)?.id ?? null
    },
    accumulatedSeconds: s => s.daily?.accumulated_seconds ?? 0,
    targetMet: s => s.daily?.is_target_met ?? false,
  },
  actions: {
    async load() {
      this.loading = true
      this.error = null
      try {
        this.daily = await useApi().get<DailyQuests>('/api/v1/quests/daily')
        this.noRoadmap = false
      } catch (e) {
        if (e instanceof ApiError && e.code === 'no_active_roadmap') {
          this.daily = null
          this.noRoadmap = true
        } else {
          this.error = e instanceof ApiError ? e.code : 'network_error'
        }
      } finally {
        this.loading = false
      }
    },
    taskById(id: string): QuestTask | null {
      return this.daily?.tasks.find(t => t.id === id) ?? null
    },
    /** Idempotent: re-entering a task resumes its timer (design §2.4). */
    startTimer(taskId: string, durationMinutes: number) {
      if (this.timers[taskId]) return
      const total = Math.max(60, Math.floor(durationMinutes * 60))
      this.timers[taskId] = { totalSeconds: total, remainingSeconds: total }
    },
    tick(taskId: string, seconds = 1) {
      const t = this.timers[taskId]
      if (t) t.remainingSeconds = Math.max(0, t.remainingSeconds - seconds)
    },
    elapsedSeconds(taskId: string): number {
      const t = this.timers[taskId]
      return t ? t.totalSeconds - t.remainingSeconds : 0
    },
    async complete(exerciseId: string, durationSeconds: number, userAnswers?: Record<string, string>): Promise<ProgressResponse> {
      const body: Record<string, unknown> = { exercise_id: exerciseId, duration_seconds: clampDuration(durationSeconds) }
      if (userAnswers && Object.keys(userAnswers).length > 0) body.user_answers = userAnswers
      const res = await useApi().post<ProgressResponse>('/api/v1/quests/progress', body)
      if (this.daily) {
        this.daily.accumulated_seconds = res.daily_seconds_spent
        this.daily.is_target_met = res.is_target_met
        const t = this.daily.tasks.find(x => x.id === exerciseId)
        if (t) t.is_completed = true
      }
      delete this.timers[exerciseId]
      return res
    },
  },
})
```

`stores/pet.ts`:
```ts
import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { ApiError } from '~/utils/apiClient'

/** Backend spec §6.3 GET /pet/status. */
export interface PetStatus {
  plant_name: string
  health_points: number
  stage: string
  current_streak: number
  last_practiced_at: string | null
}

/** Backend spec §6.3 POST /pet/revive 200 body. */
export interface ReviveResponse {
  revival_passed: boolean
  pet_state: { health_points: number, stage: string, current_streak: number }
}

/** The pet slice's pass condition: 15 minutes of study recorded after the challenge starts. */
export const REVIVE_SECONDS = 900
export const REVIVE_STORAGE_KEY = 'aelp.revive'

interface Challenge {
  date: string
  startSeconds: number
}

function storageOrNull(): Storage | null {
  return typeof localStorage === 'undefined' ? null : localStorage
}

export const usePetStore = defineStore('pet', {
  state: () => ({
    status: null as PetStatus | null,
    loading: false,
    error: null as string | null,
    challenge: null as Challenge | null,
    notWilted: false,
  }),
  getters: {
    isWilted: s => s.status !== null && (s.status.health_points <= 0 || s.status.stage === 'wilted'),
  },
  actions: {
    async load() {
      this.loading = true
      this.error = null
      try {
        this.status = await useApi().get<PetStatus>('/api/v1/pet/status')
      } catch (e) {
        this.error = e instanceof ApiError ? e.code : 'network_error'
      } finally {
        this.loading = false
      }
    },
    applyProgress(res: { pet_health: number, streak_count: number }) {
      if (!this.status) return
      this.status.health_points = res.pet_health
      this.status.current_streak = res.streak_count
    },
    hydrateChallenge() {
      const raw = storageOrNull()?.getItem(REVIVE_STORAGE_KEY)
      if (!raw) return
      try {
        const c = JSON.parse(raw) as Partial<Challenge>
        if (typeof c.date === 'string' && typeof c.startSeconds === 'number') this.challenge = { date: c.date, startSeconds: c.startSeconds }
      } catch {
        storageOrNull()?.removeItem(REVIVE_STORAGE_KEY)
      }
    },
    /**
     * §6.3 revive. `today` is GET /quests/daily `date`; `accumulatedSecondsNow`
     * its `accumulated_seconds` — the client-side anchor for the 15-minute bar
     * (the response carries no progress; pet plan Notes).
     */
    async revive(today: string, accumulatedSecondsNow: number): Promise<ReviveResponse | null> {
      this.error = null
      this.notWilted = false
      try {
        const res = await useApi().post<ReviveResponse>('/api/v1/pet/revive', { answers: {} })
        if (res.revival_passed) {
          if (this.status) Object.assign(this.status, res.pet_state)
          this.challenge = null
          storageOrNull()?.removeItem(REVIVE_STORAGE_KEY)
        } else if (!this.challenge || this.challenge.date !== today) {
          this.challenge = { date: today, startSeconds: accumulatedSecondsNow }
          storageOrNull()?.setItem(REVIVE_STORAGE_KEY, JSON.stringify(this.challenge))
        }
        return res
      } catch (e) {
        if (e instanceof ApiError && e.code === 'pet_not_wilted') {
          this.notWilted = true
          return null
        }
        this.error = e instanceof ApiError ? e.code : 'network_error'
        throw e
      }
    },
    challengeProgress(accumulatedSecondsNow: number): number {
      return this.challenge ? Math.max(0, accumulatedSecondsNow - this.challenge.startSeconds) : 0
    },
  },
})
```

- [ ] **Step 4: Run and confirm pass**

```sh
npx vitest run tests/unit/questStore.test.ts tests/unit/petStore.test.ts && npm run lint && npm run typecheck
```
Expected: 11 passed; lint and typecheck clean.

- [ ] **Step 5: Commit**

```sh
cd .. && git add frontend/stores/quest.ts frontend/stores/pet.ts frontend/tests/unit/questStore.test.ts frontend/tests/unit/petStore.test.ts && git commit -m "frontend: useQuestStore (daily, timers, progress) and usePetStore (status, revive challenge)"
```

### Task 6: Design-system components and the plant

**Files:**
- Create: `frontend/components/ui/AppButton.vue`, `frontend/components/ui/AppCard.vue`, `frontend/components/ui/StateBlock.vue`, `frontend/components/ui/SegmentedProgress.vue`, `frontend/components/ui/HealthBar.vue`, `frontend/components/plant/PlantSvg.vue`, `frontend/components/plant/SpeechBubble.vue`
- Test: `frontend/tests/unit/SegmentedProgress.test.ts`, `frontend/tests/unit/PlantSvg.test.ts`

Design §1 (shape, motion, signature bar), §3 (six drawings, tones), §4 (inventory), §6 (accessibility floor). Components import from `vue` explicitly so they mount under plain Vitest.

- [ ] **Step 1: Write the failing tests**

`tests/unit/SegmentedProgress.test.ts`:
```ts
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SegmentedProgress from '~/components/ui/SegmentedProgress.vue'

describe('SegmentedProgress (design §1)', () => {
  it('renders three segments and the wireframe 7.2 label', () => {
    const w = mount(SegmentedProgress, { props: { valueSeconds: 1200, label: 'Tiến độ hôm nay' } })
    const segments = w.findAll('[data-segment]')
    expect(segments).toHaveLength(3)
    expect(segments[0].attributes('style')).toContain('width: 100%')
    expect(segments[1].attributes('style')).toContain('width: 100%')
    expect(segments[2].attributes('style')).toContain('width: 0%')
    expect(w.text()).toContain('20 / 30 phút')
    expect(w.text()).toContain('66%')
    expect(w.find('[role="progressbar"]').attributes('aria-valuenow')).toBe('1200')
    expect(w.find('[role="progressbar"]').attributes('aria-valuemax')).toBe('1800')
  })

  it('marks the target met', () => {
    const w = mount(SegmentedProgress, { props: { valueSeconds: 1800, label: 'x', met: true } })
    expect(w.text()).toContain('Mục tiêu hôm nay đã đạt')
  })

  it('renders a single 15-minute segment for revival', () => {
    const w = mount(SegmentedProgress, { props: { valueSeconds: 450, label: 'Học 15 phút để hồi sinh', segments: 1, segmentSeconds: 900 } })
    expect(w.findAll('[data-segment]')).toHaveLength(1)
    expect(w.text()).toContain('7 / 15 phút')
  })
})
```

`tests/unit/PlantSvg.test.ts`:
```ts
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PlantSvg from '~/components/plant/PlantSvg.vue'

describe('PlantSvg (design §3)', () => {
  it.each(['seed', 'sprout', 'sapling', 'flowering', 'fruitful', 'wilted'])('draws stage %s', (stage) => {
    const w = mount(PlantSvg, { props: { stage, health: 80 } })
    expect(w.find(`[data-stage="${stage}"]`).exists()).toBe(true)
    expect(w.find('svg').attributes('aria-label')).toContain(stage)
    expect(w.find('svg').attributes('aria-label')).toContain('80')
  })

  it('falls back to the sprout silhouette for an unknown stage', () => {
    const w = mount(PlantSvg, { props: { stage: 'cactus', health: 50 } })
    expect(w.find('[data-stage="sprout"]').exists()).toBe(true)
    expect(w.find('svg').classes()).toContain('text-mute')
  })

  it('tints leaves by health tone and droops when wilted', () => {
    expect(mount(PlantSvg, { props: { stage: 'sapling', health: 20 } }).find('svg').classes()).toContain('text-alert')
    expect(mount(PlantSvg, { props: { stage: 'sapling', health: 45 } }).find('svg').classes()).toContain('text-streak')
    const wilted = mount(PlantSvg, { props: { stage: 'wilted', health: 0 } })
    expect(wilted.find('[data-stage="wilted"]').classes()).toContain('plant-droop')
  })
})
```

- [ ] **Step 2: Run and confirm failure**

```sh
npx vitest run tests/unit/SegmentedProgress.test.ts tests/unit/PlantSvg.test.ts
```
Expected: FAIL — unresolved `.vue` imports.

- [ ] **Step 3: Implement**

`components/ui/AppButton.vue`:
```vue
<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  variant?: 'primary' | 'danger' | 'ghost'
  loading?: boolean
  disabled?: boolean
  type?: 'button' | 'submit'
  block?: boolean
}>(), { variant: 'primary', loading: false, disabled: false, type: 'button', block: false })

const classes = computed(() => [
  'inline-flex h-12 items-center justify-center gap-2 rounded-btn px-5 font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-50',
  props.block ? 'w-full' : '',
  {
    primary: 'bg-growth text-white hover:bg-growth/90',
    danger: 'bg-alert text-white hover:bg-alert/90',
    ghost: 'bg-transparent text-ink hover:bg-ink/5 dark:text-paper dark:hover:bg-paper/10',
  }[props.variant],
])
</script>

<template>
  <button :type="type" :class="classes" :disabled="disabled || loading" :aria-busy="loading || undefined">
    <span v-if="loading" class="size-4 animate-spin rounded-full border-2 border-current border-t-transparent" aria-hidden="true" />
    <span :class="{ 'opacity-0': loading }"><slot /></span>
  </button>
</template>
```

`components/ui/AppCard.vue`:
```vue
<template>
  <section class="rounded-card border border-ink/10 bg-white p-4 dark:border-paper/10 dark:bg-ink">
    <h2 v-if="title" class="mb-3 text-xs font-semibold uppercase tracking-wider text-mute">
      {{ title }}
    </h2>
    <slot />
  </section>
</template>

<script setup lang="ts">
defineProps<{ title?: string }>()
</script>
```

`components/ui/StateBlock.vue` — the shared loading / empty / error block (design §4):
```vue
<script setup lang="ts">
defineProps<{
  state: 'loading' | 'empty' | 'error'
  message?: string
  action?: string
}>()
defineEmits<{ action: [] }>()
</script>

<template>
  <div v-if="state === 'loading'" class="animate-pulse space-y-3" aria-busy="true" aria-label="Đang tải">
    <div class="h-4 w-2/3 rounded bg-mute/20" />
    <div class="h-4 w-1/2 rounded bg-mute/20" />
    <div class="h-4 w-3/4 rounded bg-mute/20" />
  </div>
  <div v-else class="flex flex-col items-start gap-3" :class="state === 'error' ? 'text-alert' : 'text-mute'" role="status">
    <p class="text-base">
      {{ message }}
    </p>
    <AppButton v-if="action" :variant="state === 'error' ? 'danger' : 'primary'" @click="$emit('action')">
      {{ action }}
    </AppButton>
  </div>
</template>
```
(`AppButton` resolves through Nuxt's component auto-registration in the app; in the two component tests below nothing renders `StateBlock`, so no stub is needed.)

`components/ui/SegmentedProgress.vue` — the signature (design §1):
```vue
<script setup lang="ts">
import { computed } from 'vue'
import { DAILY_TARGET_SECONDS, SEGMENT_COUNT, SEGMENT_SECONDS, minutesOf, percentOf, segmentFills } from '~/utils/progress'

const props = withDefaults(defineProps<{
  valueSeconds: number
  label: string
  segments?: number
  segmentSeconds?: number
  met?: boolean
}>(), { segments: SEGMENT_COUNT, segmentSeconds: SEGMENT_SECONDS, met: false })

const target = computed(() => props.segments * props.segmentSeconds)
const fills = computed(() => segmentFills(props.valueSeconds, props.segmentSeconds, props.segments))
const minutes = computed(() => minutesOf(Math.min(props.valueSeconds, target.value)))
const pct = computed(() => percentOf(props.valueSeconds, target.value))
const totalMinutes = computed(() => minutesOf(target.value))
void DAILY_TARGET_SECONDS
</script>

<template>
  <div>
    <div class="flex items-baseline justify-between">
      <span class="text-xs font-semibold uppercase tracking-wider text-mute">{{ label }}:</span>
      <span class="font-display text-2xl" :class="met ? 'text-growth' : ''">
        {{ minutes }} / {{ totalMinutes }} phút
      </span>
    </div>
    <div
      class="mt-2 flex gap-1.5"
      role="progressbar"
      :aria-valuenow="Math.min(valueSeconds, target)"
      aria-valuemin="0"
      :aria-valuemax="target"
      :aria-label="label"
    >
      <div v-for="(fill, i) in fills" :key="i" class="h-3 flex-1 overflow-hidden rounded-full bg-mute/20">
        <div
          data-segment
          class="h-full rounded-full bg-growth transition-[width] duration-400 motion-reduce:transition-none"
          :style="{ width: `${Math.round(fill * 100)}%` }"
        />
      </div>
    </div>
    <p class="mt-1 text-right text-xs text-mute">
      <template v-if="met">Mục tiêu hôm nay đã đạt ✓</template>
      <template v-else>{{ pct }}%</template>
    </p>
  </div>
</template>
```
Remove the `void DAILY_TARGET_SECONDS` line and its import if ESLint does not flag the unused import; it is there only because the default target is derived from the two props.

`components/ui/HealthBar.vue`:
```vue
<script setup lang="ts">
import { computed } from 'vue'
import { healthTone } from '~/utils/plant'

const props = defineProps<{ health: number }>()
const clamped = computed(() => Math.min(100, Math.max(0, props.health)))
const barClass = computed(() => ({ growth: 'bg-growth', streak: 'bg-streak', alert: 'bg-alert' })[healthTone(clamped.value)])
</script>

<template>
  <div class="flex items-center gap-3">
    <span class="text-sm text-mute">Máu cây:</span>
    <div class="h-2 flex-1 overflow-hidden rounded-full bg-mute/20" role="meter" :aria-valuenow="clamped" aria-valuemin="0" aria-valuemax="100" aria-label="Máu cây">
      <div class="h-full rounded-full transition-[width] duration-400 motion-reduce:transition-none" :class="barClass" :style="{ width: `${clamped}%` }" />
    </div>
    <span class="w-10 text-right text-sm font-semibold tabular-nums">{{ clamped }}%</span>
  </div>
</template>
```

`components/plant/PlantSvg.vue` — six drawings on one silhouette (design §3). Leaves use `currentColor`; the `<svg>` carries the tone class:
```vue
<script setup lang="ts">
import { computed } from 'vue'
import { healthTone, normalizeStage } from '~/utils/plant'

const props = withDefaults(defineProps<{ stage: string, health: number, size?: number }>(), { size: 160 })

const norm = computed(() => normalizeStage(props.stage))
const toneClass = computed(() => {
  if (!norm.value.known) return 'text-mute'
  if (norm.value.stage === 'wilted') return 'text-alert'
  return { growth: 'text-growth', streak: 'text-streak', alert: 'text-alert' }[healthTone(props.health)]
})
const label = computed(() => `Cây đang ở giai đoạn ${norm.value.stage}, máu ${Math.max(0, props.health)}%`)
</script>

<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 160 160"
    role="img"
    :aria-label="label"
    :class="toneClass"
    class="mx-auto block"
  >
    <!-- pot and soil: identical in every stage so the plant reads as one plant growing -->
    <path d="M52 112h56l-8 32H60z" class="fill-ink dark:fill-paper/80" />
    <ellipse cx="80" cy="112" rx="30" ry="5" class="fill-ink/70 dark:fill-paper/50" />

    <g v-if="norm.stage === 'seed'" data-stage="seed">
      <path d="M62 112c0-8 8-12 18-12s18 4 18 12z" class="fill-ink/50" />
      <circle cx="80" cy="106" r="3" class="fill-ink" />
    </g>

    <g v-else-if="norm.stage === 'sprout'" data-stage="sprout" class="plant-sway">
      <path d="M80 110V78" stroke="currentColor" stroke-width="5" stroke-linecap="round" fill="none" />
      <path d="M80 86c-18 0-26-12-26-22 16 0 26 8 26 22z" fill="currentColor" />
      <path d="M80 80c18 0 26-12 26-22-16 0-26 8-26 22z" fill="currentColor" />
    </g>

    <g v-else-if="norm.stage === 'sapling'" data-stage="sapling" class="plant-sway">
      <path d="M80 110V50" stroke="currentColor" stroke-width="5" stroke-linecap="round" fill="none" />
      <path d="M80 96c-16 0-22-10-22-18 14 0 22 7 22 18z" fill="currentColor" />
      <path d="M80 84c16 0 22-10 22-18-14 0-22 7-22 18z" fill="currentColor" />
      <path d="M80 72c-16 0-22-10-22-18 14 0 22 7 22 18z" fill="currentColor" />
      <path d="M80 60c16 0 22-10 22-18-14 0-22 7-22 18z" fill="currentColor" />
      <path d="M80 50c-8-6-10-14-8-20 8 4 10 12 8 20z" fill="currentColor" />
    </g>

    <g v-else-if="norm.stage === 'flowering'" data-stage="flowering" class="plant-sway">
      <path d="M80 110V44" stroke="currentColor" stroke-width="5" stroke-linecap="round" fill="none" />
      <path d="M80 96c-16 0-22-10-22-18 14 0 22 7 22 18z" fill="currentColor" />
      <path d="M80 84c16 0 22-10 22-18-14 0-22 7-22 18z" fill="currentColor" />
      <path d="M80 72c-16 0-22-10-22-18 14 0 22 7 22 18z" fill="currentColor" />
      <path d="M80 60c16 0 22-10 22-18-14 0-22 7-22 18z" fill="currentColor" />
      <circle cx="58" cy="56" r="5" class="fill-streak" />
      <circle cx="102" cy="44" r="5" class="fill-streak" />
      <circle cx="80" cy="40" r="6" class="fill-streak" />
    </g>

    <g v-else-if="norm.stage === 'fruitful'" data-stage="fruitful" class="plant-sway">
      <path d="M80 110V40" stroke="currentColor" stroke-width="5" stroke-linecap="round" fill="none" />
      <path d="M80 98c-20 0-28-12-28-22 18 0 28 9 28 22z" fill="currentColor" />
      <path d="M80 86c20 0 28-12 28-22-18 0-28 9-28 22z" fill="currentColor" />
      <path d="M80 72c-20 0-28-12-28-22 18 0 28 9 28 22z" fill="currentColor" />
      <path d="M80 60c20 0 28-12 28-22-18 0-28 9-28 22z" fill="currentColor" />
      <circle cx="56" cy="60" r="5" class="fill-streak" />
      <circle cx="104" cy="48" r="5" class="fill-streak" />
      <circle cx="66" cy="84" r="7" class="fill-streak" />
      <circle cx="96" cy="70" r="7" class="fill-streak" />
    </g>

    <g v-else data-stage="wilted" class="plant-droop origin-[80px_110px]">
      <path d="M80 110c0-24 6-40 14-52" stroke="currentColor" stroke-width="5" stroke-linecap="round" fill="none" opacity="0.7" />
      <path d="M84 94c-14 4-22-4-24-12 12-2 20 4 24 12z" fill="currentColor" opacity="0.6" />
      <path d="M90 76c14 2 20-6 20-14-12 0-18 6-20 14z" fill="currentColor" opacity="0.6" />
      <path d="M94 62c-6 8-14 10-20 8 4-8 12-12 20-8z" fill="currentColor" opacity="0.5" />
      <path d="M46 116c8-6 16-6 22-2-6 4-14 6-22 2z" fill="currentColor" opacity="0.5" />
    </g>
  </svg>
</template>

<style scoped>
@media (prefers-reduced-motion: no-preference) {
  .plant-sway {
    transform-origin: 80px 110px;
    animation: sway 4s ease-in-out infinite;
  }
  .plant-droop {
    animation: droop 600ms ease-out forwards;
  }
}
@media (prefers-reduced-motion: reduce) {
  .plant-droop {
    transform: rotate(12deg);
  }
}
@keyframes sway {
  0%, 100% { transform: rotate(-2deg); }
  50% { transform: rotate(2deg); }
}
@keyframes droop {
  from { transform: rotate(0deg); }
  to { transform: rotate(12deg); }
}
</style>
```

`components/plant/SpeechBubble.vue`:
```vue
<template>
  <p class="relative mx-auto mt-3 max-w-xs rounded-card border border-ink/10 bg-white px-4 py-2 text-center font-display text-base dark:border-paper/10 dark:bg-ink" aria-live="polite">
    💬 "{{ line }}"
  </p>
</template>

<script setup lang="ts">
defineProps<{ line: string }>()
</script>
```

- [ ] **Step 4: Run and confirm pass**

```sh
npx vitest run tests/unit/SegmentedProgress.test.ts tests/unit/PlantSvg.test.ts && npm run lint && npm run typecheck
```
Expected: 10 passed (3 + 6 parametrised + 1 + 1 — Vitest reports 11 if `it.each` counts individually); lint/typecheck clean. If `duration-400` is not a Tailwind 3 utility in the resolved version, use `duration-300`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add frontend/components/ui frontend/components/plant frontend/tests/unit/SegmentedProgress.test.ts frontend/tests/unit/PlantSvg.test.ts && git commit -m "frontend: design-system components, three-segment progress, six-stage plant SVG"
```

### Task 7: `/login` (wireframe 7.1 upper; design §2.1)

**Files:**
- Create: `frontend/pages/login.vue`

No unit test — the page is exercised end to end by the Playwright smoke in Task 14; its logic lives in Tasks 2–3, which are unit-tested.

- [ ] **Step 1: Implement**

`pages/login.vue`:
```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useApi } from '~/composables/useApi'
import { useAuthStore, type SignInResponse } from '~/stores/auth'
import { ApiError } from '~/utils/apiClient'
import { googleAuthUrl, randomState } from '~/utils/googleAuth'

const STATE_KEY = 'aelp.oauth_state'

const config = useRuntimeConfig()
const route = useRoute()
const auth = useAuthStore()

const busy = ref(false)
const error = ref<string | null>(null)
const redirectUri = computed(() => `${window.location.origin}/login`)

function startSignIn() {
  error.value = null
  const state = randomState()
  sessionStorage.setItem(STATE_KEY, state)
  window.location.assign(googleAuthUrl(config.public.googleClientId, redirectUri.value, state))
}

async function finishSignIn(code: string, state: string) {
  const expected = sessionStorage.getItem(STATE_KEY)
  sessionStorage.removeItem(STATE_KEY)
  if (!expected || expected !== state) {
    error.value = 'Phiên đăng nhập không hợp lệ. Thử lại.'
    return
  }
  busy.value = true
  try {
    const res = await useApi().post<SignInResponse>('/api/v1/auth/google', { code, redirect_uri: redirectUri.value })
    auth.signIn(res)
    if (!auth.isAuthenticated) {
      // The backend answered 200 but not in the §6.1 shape (see the plan's merge blocker).
      error.value = 'Máy chủ trả về phiên đăng nhập không hợp lệ. Thử lại sau.'
      return
    }
    await navigateTo('/', { replace: true })
  } catch (e) {
    error.value = e instanceof ApiError && e.code === 'google_auth_failed'
      ? 'Google không xác nhận được tài khoản. Thử lại.'
      : 'Không đăng nhập được. Thử lại.'
  } finally {
    busy.value = false
  }
}

onMounted(() => {
  const { code, state } = route.query
  if (typeof code === 'string' && typeof state === 'string') {
    window.history.replaceState(null, '', '/login') // never keep the code in the address bar
    void finishSignIn(code, state)
  }
})
</script>

<template>
  <main class="mx-auto flex min-h-screen max-w-md flex-col items-center justify-center gap-6 px-6 text-center">
    <PlantSvg stage="sprout" :health="100" :size="96" />
    <h1 class="font-display text-3xl">
      Chào mừng bạn! 🌱
    </h1>
    <p class="text-mute">
      Học 30 phút mỗi ngày, nuôi một cái cây.
    </p>

    <div v-if="busy" class="flex items-center gap-2 text-mute" role="status">
      <span class="size-4 animate-spin rounded-full border-2 border-current border-t-transparent" aria-hidden="true" />
      Đang đăng nhập…
    </div>
    <AppButton v-else block @click="startSignIn">
      Đăng nhập bằng Google
    </AppButton>

    <p v-if="error" class="w-full rounded-card border border-alert/40 bg-alert/10 px-4 py-3 text-sm text-alert" role="alert">
      {{ error }}
    </p>

    <p class="text-xs text-mute">
      Bằng cách tiếp tục, bạn cho phép ứng dụng đọc lịch và nhiệm vụ Google của bạn để lên lịch học.
    </p>
  </main>
</template>
```

- [ ] **Step 2: Build**

```sh
npm run lint && npm run typecheck && npm run build
```
Expected: clean; the build lists `/login` among the routes.

- [ ] **Step 3: Commit**

```sh
cd .. && git add frontend/pages/login.vue && git commit -m "frontend: /login — Google consent redirect, state check, §6.1 code exchange"
```

### Task 8: `/` — dashboard and plant hub (wireframe 7.2; design §2.3)

**Files:**
- Create: `frontend/components/AppHeader.vue`, `frontend/components/quest/QuestRow.vue`, `frontend/pages/index.vue`

- [ ] **Step 1: Implement the header and the quest row**

`components/AppHeader.vue`:
```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '~/stores/auth'
import { clearApiCache } from '~/utils/session'

defineProps<{ streak?: number | null }>()

const auth = useAuthStore()
const menuOpen = ref(false)

async function signOut() {
  auth.signOut()
  await clearApiCache()
  await navigateTo('/login', { replace: true })
}
</script>

<template>
  <header class="flex items-center justify-between py-3">
    <div class="relative flex items-center gap-3">
      <button
        type="button"
        class="flex size-10 items-center justify-center rounded-full bg-growth font-semibold text-white"
        aria-haspopup="menu"
        :aria-expanded="menuOpen"
        aria-label="Tài khoản"
        @click="menuOpen = !menuOpen"
      >
        {{ auth.initial }}
      </button>
      <span class="font-semibold">{{ auth.user?.full_name }}</span>
      <div v-if="menuOpen" role="menu" class="absolute left-0 top-12 z-10 w-44 rounded-card border border-ink/10 bg-white p-1 shadow-sm dark:border-paper/10 dark:bg-ink">
        <NuxtLink to="/settings" role="menuitem" class="block rounded-btn px-3 py-2 hover:bg-ink/5 dark:hover:bg-paper/10" @click="menuOpen = false">
          Cài đặt
        </NuxtLink>
        <button type="button" role="menuitem" class="block w-full rounded-btn px-3 py-2 text-left hover:bg-ink/5 dark:hover:bg-paper/10" @click="signOut">
          Đăng xuất
        </button>
      </div>
    </div>
    <div class="flex items-center gap-3">
      <NuxtLink to="/roadmap" class="text-sm text-mute underline-offset-2 hover:underline">
        Lộ trình
      </NuxtLink>
      <span v-if="streak !== null && streak !== undefined" class="rounded-full bg-streak/15 px-3 py-1 text-sm font-semibold text-streak">
        🔥 Streak: {{ streak }} ngày
      </span>
    </div>
  </header>
</template>
```

`components/quest/QuestRow.vue`:
```vue
<script setup lang="ts">
import { computed } from 'vue'
import type { QuestTask } from '~/stores/quest'

const props = defineProps<{ task: QuestTask, index: number, state: 'done' | 'next' | 'locked' | 'open' }>()

const glyph = computed(() => ({ done: '[✓]', next: '[▶]', locked: '[ ]', open: '[ ]' })[props.state])
const action = computed(() => ({ done: 'Xong', next: 'Học', locked: 'Khóa', open: 'Học' })[props.state])
</script>

<template>
  <li class="flex items-center gap-3 py-2">
    <span class="w-7 font-mono text-sm" :class="state === 'done' ? 'text-growth' : 'text-mute'" aria-hidden="true">{{ glyph }}</span>
    <span class="flex-1" :class="{ 'text-mute': state === 'locked' }">
      {{ index }}. {{ task.title }}
      <span class="text-sm text-mute">({{ task.duration_minutes }}m)</span>
    </span>
    <NuxtLink v-if="state === 'next' || state === 'open'" :to="`/learn/${task.id}`" class="inline-flex h-9 items-center rounded-btn bg-growth px-4 text-sm font-semibold text-white" :aria-label="`Học: ${task.title}`">
      {{ action }}
    </NuxtLink>
    <span v-else class="inline-flex h-9 items-center px-4 text-sm font-semibold" :class="state === 'done' ? 'text-growth' : 'text-mute'">
      {{ action }}
    </span>
  </li>
</template>
```

- [ ] **Step 2: Implement the page**

`pages/index.vue`:
```vue
<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { speechLine } from '~/utils/plant'

const quest = useQuestStore()
const pet = usePetStore()

onMounted(() => {
  void Promise.all([pet.load(), quest.load()])
})

const bubble = computed(() => pet.status
  ? speechLine({ stage: pet.status.stage, health: pet.status.health_points, targetMet: quest.targetMet })
  : '')

function rowState(taskId: string, completed: boolean): 'done' | 'next' | 'locked' | 'open' {
  if (completed) return 'done'
  if (quest.targetMet) return 'open' // extra study is allowed once the day is met
  return taskId === quest.nextTaskId ? 'next' : 'locked'
}
</script>

<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <AppHeader :streak="pet.status?.current_streak ?? null" />

    <NuxtLink
      v-if="pet.isWilted"
      to="/revive"
      class="mb-4 flex items-center justify-between rounded-card bg-alert px-4 py-3 font-semibold text-white"
    >
      <span>⚠️ Cây xanh đang bị héo rũ!</span>
      <span class="text-sm underline">Cứu cây ngay</span>
    </NuxtLink>

    <AppCard class="mb-4">
      <StateBlock v-if="pet.loading && !pet.status" state="loading" />
      <StateBlock v-else-if="pet.error && !pet.status" state="error" message="Không tải được cây của bạn." action="Thử lại" @action="pet.load()" />
      <template v-else-if="pet.status">
        <PlantSvg :stage="pet.status.stage" :health="pet.status.health_points" />
        <HealthBar class="mt-3" :health="pet.status.health_points" />
        <SpeechBubble v-if="bubble !== '…'" :line="bubble" />
      </template>
    </AppCard>

    <template v-if="quest.noRoadmap">
      <AppCard>
        <StateBlock state="empty" message="Bạn chưa có lộ trình học." action="Tạo lộ trình 28 ngày" @action="navigateTo('/onboarding')" />
      </AppCard>
    </template>
    <template v-else>
      <AppCard class="mb-4">
        <StateBlock v-if="quest.loading && !quest.daily" state="loading" />
        <StateBlock v-else-if="quest.error && !quest.daily" state="error" message="Không tải được tiến độ." action="Thử lại" @action="quest.load()" />
        <SegmentedProgress v-else :value-seconds="quest.accumulatedSeconds" label="Tiến độ hôm nay" :met="quest.targetMet" />
      </AppCard>

      <AppCard title="Nhiệm vụ hôm nay (Quests)">
        <StateBlock v-if="quest.loading && !quest.daily" state="loading" />
        <StateBlock v-else-if="quest.error && !quest.daily" state="error" message="Không tải được nhiệm vụ." action="Thử lại" @action="quest.load()" />
        <ul v-else class="divide-y divide-ink/10 dark:divide-paper/10">
          <QuestRow
            v-for="(task, i) in quest.sortedTasks"
            :key="task.id"
            :task="task"
            :index="i + 1"
            :state="rowState(task.id, task.is_completed)"
          />
        </ul>
      </AppCard>
    </template>
  </main>
</template>
```

- [ ] **Step 3: Build**

```sh
npm run lint && npm run typecheck && npm run build
```
Expected: clean.

- [ ] **Step 4: Commit**

```sh
cd .. && git add frontend/components/AppHeader.vue frontend/components/quest/QuestRow.vue frontend/pages/index.vue && git commit -m "frontend: / dashboard — plant hub, three-segment day, quest list, wilted banner"
```

### Task 9: `/learn/:id` — distraction-free learning room (wireframe 7.3; design §2.4)

**Files:**
- Create: `frontend/components/learn/CountdownTimer.vue`, `frontend/components/learn/ContentViewer.vue`, `frontend/utils/content.ts`, `frontend/pages/learn/[id].vue`
- Test: `frontend/tests/unit/content.test.ts`

`content_json` is the roadmap task object (CODEMAP airouter: `{type, title, duration_minutes, content}`) — but the §6.2 example puts `words` at the top level and the demo seed (`store.SeedDemoRoadmap`) has neither. `utils/content.ts` classifies all three so the room never renders blank.

- [ ] **Step 1: Write the failing test**

`tests/unit/content.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { classifyContent } from '~/utils/content'

describe('classifyContent (design §2.4)', () => {
  it('finds a §6.2 vocabulary list at the top level or under content', () => {
    const words = [{ term: 'Inquire', definition: 'To ask for information' }]
    expect(classifyContent({ words })).toEqual({ kind: 'words', words })
    expect(classifyContent({ type: 'vocabulary', content: { words } })).toEqual({ kind: 'words', words })
  })

  it('finds a question list', () => {
    const questions = [{ id: 'q1', prompt: 'Choose…', options: { A: 'x', B: 'y' } }]
    expect(classifyContent({ content: { questions } })).toEqual({ kind: 'questions', questions })
  })

  it('falls back to raw JSON for anything else (the demo seed shape)', () => {
    const seed = { title: 'Day 1 vocabulary', duration_minutes: 10, day: 1, task: 'vocabulary' }
    expect(classifyContent(seed)).toEqual({ kind: 'raw', text: JSON.stringify(seed, null, 2) })
    expect(classifyContent(null)).toEqual({ kind: 'raw', text: 'null' })
  })
})
```

- [ ] **Step 2: Run and confirm failure**

```sh
npx vitest run tests/unit/content.test.ts
```
Expected: FAIL — unresolved import.

- [ ] **Step 3: Implement**

`utils/content.ts`:
```ts
export interface Word {
  term: string
  definition: string
}
export interface Question {
  id: string
  prompt: string
  options: Record<string, string>
}

export type Classified
  = { kind: 'words', words: Word[] }
    | { kind: 'questions', questions: Question[] }
    | { kind: 'raw', text: string }

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}

function pick(cj: Record<string, unknown>, key: string): unknown {
  if (Array.isArray(cj[key])) return cj[key]
  const inner = cj.content
  if (isRecord(inner) && Array.isArray(inner[key])) return inner[key]
  return undefined
}

/** Picks a renderer for a task's content_json without ever returning nothing. */
export function classifyContent(contentJson: unknown): Classified {
  if (isRecord(contentJson)) {
    const words = pick(contentJson, 'words')
    if (Array.isArray(words) && words.length > 0) return { kind: 'words', words: words as Word[] }
    const questions = pick(contentJson, 'questions')
    if (Array.isArray(questions) && questions.length > 0) return { kind: 'questions', questions: questions as Question[] }
  }
  return { kind: 'raw', text: JSON.stringify(contentJson, null, 2) }
}
```

`components/learn/CountdownTimer.vue`:
```vue
<script setup lang="ts">
import { computed } from 'vue'
import { formatCountdown } from '~/utils/progress'

const props = defineProps<{ remainingSeconds: number }>()
const text = computed(() => formatCountdown(props.remainingSeconds))
</script>

<template>
  <span class="tabular-nums" :class="remainingSeconds === 0 ? 'text-alert' : ''" aria-live="off">
    ⏱️ Thời gian: <span class="font-semibold">{{ text }}</span>
  </span>
</template>
```

`components/learn/ContentViewer.vue` — steps through words or questions; emits `answer` per question and `finished` when the last item is passed:
```vue
<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Classified } from '~/utils/content'

const props = defineProps<{ content: Classified, title: string }>()
const emit = defineEmits<{ answer: [questionId: string, option: string], finished: [] }>()

const index = ref(0)
const selected = ref<string | null>(null)

const items = computed(() => props.content.kind === 'words' ? props.content.words : props.content.kind === 'questions' ? props.content.questions : [])
const total = computed(() => items.value.length)
const isLast = computed(() => index.value >= total.value - 1)

const buttonLabel = computed(() => {
  if (props.content.kind === 'questions' && selected.value !== null) return 'Gửi đáp án'
  return isLast.value || props.content.kind === 'raw' ? 'Hoàn thành' : 'Tiếp tục'
})

function next() {
  if (props.content.kind === 'questions' && selected.value !== null) {
    emit('answer', props.content.questions[index.value]!.id, selected.value)
  }
  selected.value = null
  if (props.content.kind === 'raw' || isLast.value) {
    emit('finished')
    return
  }
  index.value += 1
}
</script>

<template>
  <div class="space-y-4">
    <p v-if="content.kind !== 'raw'" class="text-sm text-mute">
      Câu {{ index + 1 }} / {{ total }}
    </p>

    <template v-if="content.kind === 'words'">
      <p class="font-display text-3xl">
        {{ content.words[index]?.term }}
      </p>
      <p class="text-lg">
        {{ content.words[index]?.definition }}
      </p>
    </template>

    <template v-else-if="content.kind === 'questions'">
      <p class="text-lg">
        "{{ content.questions[index]?.prompt }}"
      </p>
      <div role="radiogroup" class="space-y-2">
        <button
          v-for="(text, key) in content.questions[index]?.options"
          :key="key"
          type="button"
          role="radio"
          :aria-checked="selected === key"
          class="block w-full rounded-btn border px-4 py-3 text-left"
          :class="selected === key ? 'border-growth bg-growth/10' : 'border-ink/15 dark:border-paper/15'"
          @click="selected = String(key)"
        >
          ({{ key }}) {{ text }}
        </button>
      </div>
    </template>

    <template v-else>
      <h2 class="font-display text-2xl">
        {{ title }}
      </h2>
      <pre class="overflow-x-auto rounded-card bg-ink/5 p-3 text-xs dark:bg-paper/10">{{ content.text }}</pre>
    </template>

    <AppButton block @click="next">
      {{ buttonLabel }}
    </AppButton>
  </div>
</template>
```
(An unanswered question may be skipped — the button then reads "Tiếp tục" instead of "Gửi đáp án", so the label tells the user which they are doing.)

`pages/learn/[id].vue`:
```vue
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { ApiError } from '~/utils/apiClient'
import { classifyContent } from '~/utils/content'

const route = useRoute()
const quest = useQuestStore()
const pet = usePetStore()

const id = computed(() => String(route.params.id))
const task = computed(() => quest.taskById(id.value))
const content = computed(() => classifyContent(task.value?.content_json))
const timer = computed(() => quest.timers[id.value])
const answers = ref<Record<string, string>>({})
const finished = ref(false)
const posting = ref(false)
const error = ref<string | null>(null)
const online = ref(typeof navigator === 'undefined' ? true : navigator.onLine)

let interval: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  if (!quest.daily && !quest.noRoadmap) await quest.load()
  if (task.value && !task.value.is_completed) {
    quest.startTimer(task.value.id, task.value.duration_minutes || 10)
    interval = setInterval(() => quest.tick(task.value!.id), 1000)
  }
  window.addEventListener('online', setOnline)
  window.addEventListener('offline', setOnline)
})

onBeforeUnmount(() => {
  if (interval) clearInterval(interval) // the store keeps remainingSeconds, so re-entry resumes
  window.removeEventListener('online', setOnline)
  window.removeEventListener('offline', setOnline)
})

function setOnline() {
  online.value = navigator.onLine
}

const buttonLabel = computed(() => {
  if (task.value?.is_completed) return 'Đã hoàn thành'
  if (timer.value?.remainingSeconds === 0) return 'Hết giờ — Hoàn thành'
  return 'Hoàn thành'
})

async function complete() {
  if (!task.value || posting.value) return
  posting.value = true
  error.value = null
  try {
    const res = await quest.complete(task.value.id, quest.elapsedSeconds(task.value.id), answers.value)
    pet.applyProgress(res)
    await navigateTo('/', { replace: true })
  } catch (e) {
    error.value = e instanceof ApiError && e.code === 'exercise_not_found'
      ? 'Nhiệm vụ này không còn trong hôm nay. Về trang chính để tải lại.'
      : 'Chưa ghi được tiến độ. Thử lại.'
  } finally {
    posting.value = false
  }
}
</script>

<template>
  <main class="mx-auto flex min-h-screen max-w-md flex-col px-4 pb-8">
    <div class="flex items-center justify-between py-3 text-sm">
      <NuxtLink to="/" class="text-mute hover:underline">‹ Quay lại</NuxtLink>
      <CountdownTimer v-if="timer" :remaining-seconds="timer.remainingSeconds" />
    </div>

    <AppCard class="flex-1">
      <StateBlock v-if="quest.loading && !quest.daily" state="loading" />
      <StateBlock v-else-if="!task" state="empty" message="Nhiệm vụ này không có trong hôm nay." action="Về trang chính" @action="navigateTo('/')" />
      <template v-else>
        <ContentViewer
          v-if="!finished && !task.is_completed"
          :content="content"
          :title="task.title"
          @answer="(qid, opt) => { answers[qid] = opt }"
          @finished="finished = true"
        />
        <div v-else class="space-y-4">
          <h2 class="font-display text-2xl">
            {{ task.title }}
          </h2>
          <p v-if="task.is_completed" class="text-growth">
            ✓ Đã hoàn thành
          </p>
          <p v-else class="text-mute">
            Bạn đã xem hết nội dung. Ghi lại thời gian học để tưới cây.
          </p>
        </div>
      </template>
    </AppCard>

    <p v-if="error" class="mt-3 rounded-card border border-alert/40 bg-alert/10 px-4 py-3 text-sm text-alert" role="alert">
      {{ error }}
    </p>
    <p v-if="!online" class="mt-3 text-center text-sm text-mute">
      Cần kết nối để ghi tiến độ
    </p>

    <AppButton
      v-if="task"
      class="mt-4"
      block
      :loading="posting"
      :disabled="task.is_completed || !online || (!finished && timer?.remainingSeconds !== 0)"
      @click="complete"
    >
      {{ buttonLabel }}
    </AppButton>
  </main>
</template>
```
The bottom button is enabled once the content has been stepped through **or** the timer has run out — either proves the ten minutes were spent or the material was covered. Elapsed seconds are what is posted; the backend clamps nothing, so `clampDuration` in the store guarantees `1..3600`.

- [ ] **Step 4: Run, build**

```sh
npx vitest run tests/unit/content.test.ts && npm run lint && npm run typecheck && npm run build
```
Expected: 3 passed; clean build with `/learn/:id` in the route list.

- [ ] **Step 5: Commit**

```sh
cd .. && git add frontend/utils/content.ts frontend/tests/unit/content.test.ts frontend/components/learn frontend/pages/learn && git commit -m "frontend: /learn/:id learning room — countdown, content viewer, §6.2 progress post"
```

### Task 10: `/roadmap` — 28-day curriculum tree (wireframe 7.4; design §2.5)

**Files:**
- Create: `frontend/components/roadmap/RoadmapNode.vue`, `frontend/pages/roadmap.vue`

- [ ] **Step 1: Implement**

`components/roadmap/RoadmapNode.vue`:
```vue
<script setup lang="ts">
import { computed } from 'vue'
import type { RoadmapNode } from '~/utils/roadmap'

const props = defineProps<{ node: RoadmapNode }>()

const glyph = computed(() => ({ completed: '⭐', today: '🌱', locked: '🔒' })[props.node.state])
const text = computed(() => ({ completed: 'Đã hoàn thành', today: 'HÔM NAY - Đang học', locked: 'Chưa mở khóa' })[props.node.state])
const pill = computed(() => ({
  completed: 'border-streak/40 bg-streak/10 text-ink dark:text-paper',
  today: 'border-growth bg-growth text-white scale-105',
  locked: 'border-mute/30 text-mute',
})[props.node.state])
</script>

<template>
  <component
    :is="node.state === 'today' ? 'NuxtLink' : 'div'"
    :to="node.state === 'today' ? '/' : undefined"
    :id="`day-${node.day}`"
    class="inline-flex items-center gap-2 rounded-full border px-4 py-2 text-sm font-semibold"
    :class="pill"
    :aria-current="node.state === 'today' ? 'step' : undefined"
  >
    <span aria-hidden="true">{{ glyph }}</span>
    <span>Ngày {{ node.day }}: {{ text }}</span>
  </component>
</template>
```
If `<component :is="'NuxtLink'">` does not resolve the auto-imported component in the resolved Nuxt version, import it explicitly: `import { NuxtLink } from '#components'` and bind `:is="node.state === 'today' ? NuxtLink : 'div'"`.

`pages/roadmap.vue`:
```vue
<script setup lang="ts">
import { computed, nextTick, onMounted } from 'vue'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { roadmapNodes } from '~/utils/roadmap'

const quest = useQuestStore()
const pet = usePetStore()

onMounted(async () => {
  if (!pet.status) void pet.load()
  if (!quest.daily && !quest.noRoadmap) await quest.load()
  await nextTick()
  document.getElementById(`day-${quest.daily?.day_number ?? 0}`)?.scrollIntoView({ block: 'center' })
})

const nodes = computed(() => (quest.daily ? roadmapNodes(quest.daily.day_number) : []))
</script>

<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <AppHeader :streak="pet.status?.current_streak ?? null" />
    <h1 class="mb-4 font-display text-2xl">
      Lộ trình học 28 ngày
    </h1>

    <AppCard>
      <StateBlock v-if="quest.loading && !quest.daily" state="loading" />
      <StateBlock v-else-if="quest.noRoadmap" state="empty" message="Bạn chưa có lộ trình học." action="Tạo lộ trình 28 ngày" @action="navigateTo('/onboarding')" />
      <StateBlock v-else-if="quest.error && !quest.daily" state="error" message="Không tải được lộ trình." action="Thử lại" @action="quest.load()" />
      <ol v-else class="relative space-y-3">
        <template v-for="(node, i) in nodes" :key="node.day">
          <li v-if="i % 7 === 0" class="pt-2 text-xs font-semibold uppercase tracking-wider text-mute" aria-hidden="true">
            Tuần {{ node.week }}
          </li>
          <li class="flex" :class="i % 2 === 0 ? 'justify-start pl-2' : 'justify-end pr-2'">
            <RoadmapNode :node="node" />
          </li>
          <li v-if="i < nodes.length - 1 && (i + 1) % 7 !== 0" class="h-4 border-mute/30" :class="i % 2 === 0 ? 'ml-[40%] border-l' : 'mr-[40%] border-r'" aria-hidden="true" />
        </template>
      </ol>
    </AppCard>
  </main>
</template>
```
The zig-zag: even indexes sit left, odd sit right, and a short vertical rule between them stands in for the wireframe's `\` and `/`.

- [ ] **Step 2: Build**

```sh
npm run lint && npm run typecheck && npm run build
```
Expected: clean.

- [ ] **Step 3: Commit**

```sh
cd .. && git add frontend/components/roadmap frontend/pages/roadmap.vue && git commit -m "frontend: /roadmap — 28-day zig-zag tree derived from day_number"
```

### Task 11: `/revive` — plant revival mode (wireframe 7.5; design §2.6)

**Files:**
- Create: `frontend/pages/revive.vue`

Walks the §6.3 contract as the pet slice implements it (pet plan Notes: "Revive `answers` are accepted and ignored"; the pass condition is 15 minutes of study on today's counter after the challenge starts).

- [ ] **Step 1: Implement**

`pages/revive.vue`:
```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { REVIVE_SECONDS, usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { daysSince } from '~/utils/plant'

const pet = usePetStore()
const quest = useQuestStore()

const busy = ref(false)
const error = ref<string | null>(null)
const passed = ref(false)

onMounted(async () => {
  pet.hydrateChallenge()
  await Promise.all([pet.status ? Promise.resolve() : pet.load(), quest.daily ? Promise.resolve() : quest.load()])
})

const missedDays = computed(() => daysSince(pet.status?.last_practiced_at ?? null))
const today = computed(() => quest.daily?.date ?? new Date().toISOString().slice(0, 10))
const challengeActive = computed(() => pet.challenge !== null && pet.challenge.date === today.value)
const progress = computed(() => pet.challengeProgress(quest.accumulatedSeconds))
const canCheck = computed(() => progress.value >= REVIVE_SECONDS)

async function revive() {
  busy.value = true
  error.value = null
  try {
    const res = await pet.revive(today.value, quest.accumulatedSeconds)
    if (res?.revival_passed) passed.value = true
  } catch {
    error.value = 'Không bắt đầu được thử thách. Thử lại.'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <template v-if="pet.loading && !pet.status">
      <AppCard class="mt-4">
        <StateBlock state="loading" />
      </AppCard>
    </template>

    <template v-else-if="passed">
      <AppCard class="mt-4 text-center">
        <PlantSvg :stage="pet.status?.stage ?? 'sprout'" :health="pet.status?.health_points ?? 50" />
        <p class="mt-3 font-display text-2xl text-growth">
          Cây đã hồi sinh!
        </p>
        <p class="text-mute">
          Máu cây: {{ pet.status?.health_points ?? 50 }}%
        </p>
        <AppButton class="mt-4" block @click="navigateTo('/')">
          Về trang chính
        </AppButton>
      </AppCard>
    </template>

    <template v-else-if="pet.notWilted || (pet.status && !pet.isWilted)">
      <AppCard class="mt-4 text-center">
        <PlantSvg :stage="pet.status?.stage ?? 'sprout'" :health="pet.status?.health_points ?? 100" />
        <p class="mt-3 font-display text-2xl">
          Cây của bạn vẫn khỏe 🌱
        </p>
        <AppButton class="mt-4" block @click="navigateTo('/')">
          Về trang chính
        </AppButton>
      </AppCard>
    </template>

    <template v-else>
      <div class="mt-4 rounded-card bg-alert px-4 py-3 text-center font-semibold uppercase tracking-wide text-white" role="alert">
        ⚠️ Cây xanh đang bị héo rũ!
      </div>

      <AppCard class="mt-4 text-center">
        <PlantSvg stage="wilted" :health="0" />
        <p class="text-sm text-mute">
          Cây héo - 0%
        </p>
      </AppCard>

      <AppCard v-if="!challengeActive" class="mt-4">
        <p class="font-display text-lg">
          "Bạn đã bỏ học<template v-if="missedDays !== null"> {{ missedDays }} ngày liên tiếp</template>. Hãy hoàn thành Bài kiểm tra Cứu Cây 15 phút để hồi sinh!"
        </p>
        <AppButton class="mt-4" variant="danger" block :loading="busy" @click="revive">
          🚨 Cứu cây ngay (Quiz 15 phút)
        </AppButton>
      </AppCard>

      <AppCard v-else class="mt-4">
        <SegmentedProgress :value-seconds="progress" label="Học 15 phút để hồi sinh" :segments="1" :segment-seconds="900" :met="canCheck" />
        <div class="mt-4 flex flex-col gap-2">
          <AppButton block @click="navigateTo('/')">
            Vào học ngay
          </AppButton>
          <AppButton variant="ghost" block :loading="busy" @click="revive">
            Kiểm tra hồi sinh
          </AppButton>
        </div>
      </AppCard>

      <p v-if="error" class="mt-3 rounded-card border border-alert/40 bg-alert/10 px-4 py-3 text-sm text-alert" role="alert">
        {{ error }}
      </p>
    </template>
  </main>
</template>
```
The 15-minute bar's `met` label will read "Mục tiêu hôm nay đã đạt ✓" — acceptable for the shell; if the design owner wants "Đủ 15 phút — kiểm tra hồi sinh", add a `metLabel` prop to `SegmentedProgress` (Task 6) then.

- [ ] **Step 2: Build**

```sh
npm run lint && npm run typecheck && npm run build
```
Expected: clean.

- [ ] **Step 3: Commit**

```sh
cd .. && git add frontend/pages/revive.vue && git commit -m "frontend: /revive — §6.3 revive challenge with 15-minute bar, not-wilted and passed states"
```

### Task 12: `/onboarding` against a documented stub, and the `/settings` placeholder (wireframe 7.1 lower; design §2.2, §2.7)

**Files:**
- Create: `frontend/stubs/onboarding.ts`, `frontend/composables/useOnboardingApi.ts`, `frontend/components/onboarding/GoalCard.vue`, `frontend/pages/onboarding.vue`, `frontend/pages/settings.vue`
- Test: `frontend/tests/unit/onboardingStub.test.ts`

The onboarding backend is planned, not executed (`harness/plans/2026-09-22-onboarding-placement-test-cefr-grading-and-roadmap-generatio.md`). Its `GET /api/v1/onboarding/quiz` body is `{questions[{id, prompt, options}]}` and `POST /api/v1/onboarding/assessment` is Backend spec §6.1. The stub mirrors both so flipping `NUXT_PUBLIC_STUB_ONBOARDING=false` is the only change when the slice merges.

- [ ] **Step 1: Write the failing test**

`tests/unit/onboardingStub.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { STUB_QUIZ, stubAssessment } from '~/stubs/onboarding'

describe('onboarding stub (mirrors the onboarding plan and Backend spec §6.1)', () => {
  it('serves questions shaped like GET /api/v1/onboarding/quiz', () => {
    expect(STUB_QUIZ.questions.length).toBeGreaterThanOrEqual(3)
    for (const q of STUB_QUIZ.questions) {
      expect(q).toMatchObject({ id: expect.any(String), prompt: expect.any(String) })
      expect(Object.keys(q.options)).toEqual(['A', 'B', 'C', 'D'])
    }
  })

  it('answers the §6.1 assessment shape', () => {
    const res = stubAssessment({
      target_goal: 'IELTS 7.0',
      notification_time: '20:00:00',
      timezone: 'Asia/Ho_Chi_Minh',
      answers: [{ question_id: 'q1', selected_option: 'B' }],
    })
    expect(res).toEqual({
      status: 'success',
      assessed_level: expect.stringMatching(/^[ABC][12]$/),
      roadmap_id: expect.any(String),
      pet_state: { plant_name: 'My Green Buddy', health_points: 100, stage: 'sprout' },
    })
  })
})
```

- [ ] **Step 2: Run and confirm failure**

```sh
npx vitest run tests/unit/onboardingStub.test.ts
```
Expected: FAIL — unresolved import.

- [ ] **Step 3: Implement**

`stubs/onboarding.ts`:
```ts
/**
 * STUB — replaces the onboarding backend until its slice merges. Shapes:
 * quiz = the onboarding plan's QuizResponse; assessment = Backend spec §6.1.
 * Selected by NUXT_PUBLIC_STUB_ONBOARDING=true (nuxt.config runtimeConfig).
 */
export interface QuizQuestion {
  id: string
  prompt: string
  options: Record<string, string>
}
export interface QuizResponse {
  questions: QuizQuestion[]
}
export interface AssessmentRequest {
  target_goal: string
  notification_time: string
  timezone: string
  answers: { question_id: string, selected_option: string }[]
}
export interface AssessmentResponse {
  status: 'success'
  assessed_level: string
  roadmap_id: string
  pet_state: { plant_name: string, health_points: number, stage: string }
}

export const STUB_QUIZ: QuizResponse = {
  questions: [
    { id: 'q1', prompt: 'She ___ to work every day.', options: { A: 'go', B: 'goes', C: 'going', D: 'gone' } },
    { id: 'q2', prompt: 'Choose the correct formal phrasing for requesting a price quotation:', options: { A: 'Give me the cost details right now.', B: 'Could you please provide a price quotation?', C: 'Send me how much this thing costs.', D: 'Price. Now.' } },
    { id: 'q3', prompt: 'If I ___ more time, I would travel.', options: { A: 'have', B: 'had', C: 'has', D: 'having' } },
    { id: 'q4', prompt: 'The report ___ by the team before the deadline.', options: { A: 'completed', B: 'was completed', C: 'has completing', D: 'complete' } },
    { id: 'q5', prompt: 'Which word is closest in meaning to "inquire"?', options: { A: 'ask', B: 'ignore', C: 'answer', D: 'inspire' } },
  ],
}

const STUB_KEY = { q1: 'B', q2: 'B', q3: 'B', q4: 'B', q5: 'A' } as const

export function stubAssessment(req: AssessmentRequest): AssessmentResponse {
  const correct = req.answers.filter(a => (STUB_KEY as Record<string, string>)[a.question_id] === a.selected_option).length
  const level = correct >= 5 ? 'B2' : correct >= 3 ? 'B1' : correct >= 1 ? 'A2' : 'A1'
  return {
    status: 'success',
    assessed_level: level,
    roadmap_id: 'stub-roadmap-0000-0000-000000000000',
    pet_state: { plant_name: 'My Green Buddy', health_points: 100, stage: 'sprout' },
  }
}
```

`composables/useOnboardingApi.ts`:
```ts
import { useRuntimeConfig } from '#app'
import { useApi } from '~/composables/useApi'
import { STUB_QUIZ, stubAssessment, type AssessmentRequest, type AssessmentResponse, type QuizResponse } from '~/stubs/onboarding'

export function useOnboardingApi() {
  const stub = useRuntimeConfig().public.stubOnboarding === 'true'
  return {
    isStub: stub,
    quiz: (): Promise<QuizResponse> => (stub ? Promise.resolve(STUB_QUIZ) : useApi().get<QuizResponse>('/api/v1/onboarding/quiz')),
    assess: (req: AssessmentRequest): Promise<AssessmentResponse> =>
      stub ? Promise.resolve(stubAssessment(req)) : useApi().post<AssessmentResponse>('/api/v1/onboarding/assessment', req),
  }
}
```

`components/onboarding/GoalCard.vue`:
```vue
<script setup lang="ts">
defineProps<{ label: string, emoji: string, selected: boolean }>()
defineEmits<{ select: [] }>()
</script>

<template>
  <button
    type="button"
    role="radio"
    :aria-checked="selected"
    class="flex h-24 flex-1 flex-col items-center justify-center gap-1 rounded-card border-2 text-center font-semibold"
    :class="selected ? 'border-growth bg-growth/10' : 'border-ink/15 dark:border-paper/15'"
    @click="$emit('select')"
  >
    <span class="text-2xl" aria-hidden="true">{{ emoji }}</span>
    <span>{{ label }}</span>
  </button>
</template>
```

`pages/onboarding.vue`:
```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useOnboardingApi } from '~/composables/useOnboardingApi'
import { useQuestStore } from '~/stores/quest'
import type { AssessmentResponse, QuizQuestion } from '~/stubs/onboarding'

const GOALS = [
  { label: 'IELTS 7.0', emoji: '🎓', value: 'IELTS 7.0 Preparation' },
  { label: 'Business English', emoji: '💼', value: 'Business English' },
]

const api = useOnboardingApi()
const quest = useQuestStore()

const step = ref<'goal' | 'quiz' | 'result'>('goal')
const goal = ref<string | null>(null)
const time = ref('20:00')
const questions = ref<QuizQuestion[]>([])
const index = ref(0)
const answers = ref<Record<string, string>>({})
const loading = ref(false)
const error = ref<string | null>(null)
const result = ref<AssessmentResponse | null>(null)

onMounted(async () => {
  if (!quest.daily && !quest.noRoadmap) await quest.load()
  if (quest.daily) await navigateTo('/', { replace: true }) // home is the dashboard once a roadmap exists
})

const current = computed(() => questions.value[index.value])
const isLast = computed(() => index.value >= questions.value.length - 1)

async function startQuiz() {
  if (!goal.value) return
  loading.value = true
  error.value = null
  try {
    questions.value = (await api.quiz()).questions
    step.value = 'quiz'
  } catch {
    error.value = 'Không tải được bài kiểm tra. Thử lại.'
  } finally {
    loading.value = false
  }
}

async function next() {
  if (!isLast.value) {
    index.value += 1
    return
  }
  loading.value = true
  error.value = null
  try {
    result.value = await api.assess({
      target_goal: goal.value!,
      notification_time: `${time.value}:00`,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      answers: Object.entries(answers.value).map(([question_id, selected_option]) => ({ question_id, selected_option })),
    })
    step.value = 'result'
  } catch {
    error.value = 'Không tạo được lộ trình. Thử lại.' // answers are kept
  } finally {
    loading.value = false
  }
}

async function finish() {
  await quest.load()
  await navigateTo('/', { replace: true })
}
</script>

<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <AppHeader />

    <AppCard v-if="step === 'goal'" class="mt-2">
      <h1 class="font-display text-2xl">
        Mục tiêu học của bạn là gì?
      </h1>
      <div class="mt-4 flex gap-3" role="radiogroup" aria-label="Mục tiêu">
        <GoalCard v-for="g in GOALS" :key="g.value" :label="g.label" :emoji="g.emoji" :selected="goal === g.value" @select="goal = g.value" />
      </div>
      <label class="mt-6 block">
        <span class="text-sm text-mute">Chọn giờ nhắc học hằng ngày</span>
        <input v-model="time" type="time" class="mt-1 block w-full rounded-btn border border-ink/15 bg-transparent px-3 py-2 dark:border-paper/15">
      </label>
      <AppButton class="mt-6" block :disabled="!goal" :loading="loading" @click="startQuiz">
        Bắt đầu bài kiểm tra đầu vào
      </AppButton>
      <p v-if="api.isStub" class="mt-3 text-center text-xs text-mute">
        Bản thử: bài kiểm tra và lộ trình là dữ liệu mẫu cho đến khi máy chủ onboarding sẵn sàng.
      </p>
    </AppCard>

    <AppCard v-else-if="step === 'quiz'" class="mt-2">
      <StateBlock v-if="questions.length === 0" state="empty" message="Chưa có bài kiểm tra. Quay lại sau." action="Về trang chính" @action="navigateTo('/')" />
      <template v-else-if="current">
        <p class="text-sm text-mute">
          Câu {{ index + 1 }} / {{ questions.length }}
        </p>
        <p class="mt-2 text-lg">
          "{{ current.prompt }}"
        </p>
        <div class="mt-4 space-y-2" role="radiogroup">
          <button
            v-for="(text, key) in current.options"
            :key="key"
            type="button"
            role="radio"
            :aria-checked="answers[current.id] === key"
            class="block w-full rounded-btn border px-4 py-3 text-left"
            :class="answers[current.id] === key ? 'border-growth bg-growth/10' : 'border-ink/15 dark:border-paper/15'"
            @click="answers[current.id] = String(key)"
          >
            ({{ key }}) {{ text }}
          </button>
        </div>
        <AppButton class="mt-6" block :disabled="!answers[current.id]" :loading="loading" @click="next">
          {{ isLast ? 'Hoàn thành' : 'Tiếp tục' }}
        </AppButton>
      </template>
    </AppCard>

    <AppCard v-else-if="result" class="mt-2 text-center">
      <PlantSvg :stage="result.pet_state.stage" :health="result.pet_state.health_points" />
      <p class="mt-3 font-display text-2xl">
        Trình độ của bạn: {{ result.assessed_level }}
      </p>
      <p class="text-mute">
        {{ result.pet_state.plant_name }} đã nảy mầm. Tưới cây bằng 30 phút học mỗi ngày.
      </p>
      <AppButton class="mt-4" block @click="finish">
        Xem nhiệm vụ hôm nay
      </AppButton>
    </AppCard>

    <p v-if="error" class="mt-3 rounded-card border border-alert/40 bg-alert/10 px-4 py-3 text-sm text-alert" role="alert">
      {{ error }}
    </p>
  </main>
</template>
```
With the stub on, `finish()` reloads `/quests/daily`, which still answers `404 no_active_roadmap` from the real backend (the stub created nothing), so the dashboard shows the empty state again. That loop is the documented limitation of the stub; the reviewer's click-through should use `store.SeedDemoRoadmap` (quests slice) for a live roadmap.

`pages/settings.vue`:
```vue
<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <AppHeader />
    <AppCard title="Sắp ra mắt" class="mt-2">
      <p class="text-mute">
        Nhắc học và đồng bộ Google sẽ bật ở đây khi máy chủ sẵn sàng.
      </p>
      <div class="mt-4 flex flex-col gap-2">
        <AppButton block disabled aria-describedby="settings-soon">Bật nhắc học</AppButton>
        <AppButton block disabled variant="ghost" aria-describedby="settings-soon">Đồng bộ Google</AppButton>
      </div>
      <p id="settings-soon" class="sr-only">Chưa khả dụng</p>
    </AppCard>
  </main>
</template>
```

- [ ] **Step 4: Run, build**

```sh
npx vitest run tests/unit/onboardingStub.test.ts && npm run lint && npm run typecheck && npm run build
```
Expected: 2 passed; clean build listing `/onboarding` and `/settings`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add frontend/stubs/onboarding.ts frontend/composables/useOnboardingApi.ts frontend/components/onboarding frontend/pages/onboarding.vue frontend/pages/settings.vue frontend/tests/unit/onboardingStub.test.ts && git commit -m "frontend: /onboarding against a documented stub, /settings placeholder"
```

### Task 13: The service worker — §3 caching, push, icons (design §5)

**Files:**
- Create: `frontend/service-worker/push.ts`, `frontend/pwa-assets.config.ts`, `frontend/public/pwa-64x64.png`, `frontend/public/pwa-192x192.png`, `frontend/public/pwa-512x512.png`, `frontend/public/maskable-icon-512x512.png`, `frontend/public/apple-touch-icon-180x180.png`, `frontend/public/favicon.ico`
- Modify: `frontend/service-worker/sw.ts` (replace the Task 1 placeholder)
- Test: `frontend/tests/unit/push.test.ts`

- [ ] **Step 1: Write the failing test**

`tests/unit/push.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { parsePushPayload } from '~/service-worker/push'

describe('parsePushPayload (idea: "shows the notify payload and opens its url")', () => {
  it('reads title, body and url from a JSON payload', () => {
    expect(parsePushPayload('{"title":"Đến giờ học","body":"Cây đang khát","url":"/"}')).toEqual({
      title: 'Đến giờ học',
      body: 'Cây đang khát',
      url: '/',
    })
  })

  it('falls back to a default notification for an empty or non-JSON payload', () => {
    expect(parsePushPayload(null)).toEqual({ title: 'Học 30 phút', body: 'Cây của bạn đang chờ bạn.', url: '/' })
    expect(parsePushPayload('plain text')).toEqual({ title: 'Học 30 phút', body: 'plain text', url: '/' })
  })

  it('only opens same-origin urls', () => {
    expect(parsePushPayload('{"url":"https://evil.example/x"}').url).toBe('/')
    expect(parsePushPayload('{"url":"/revive"}').url).toBe('/revive')
  })
})
```

- [ ] **Step 2: Run and confirm failure**

```sh
npx vitest run tests/unit/push.test.ts
```
Expected: FAIL — unresolved import.

- [ ] **Step 3: Implement**

`service-worker/push.ts` (pure — no worker globals, so it unit-tests in Node):
```ts
export interface PushPayload {
  title: string
  body: string
  url: string
}

const DEFAULT: PushPayload = { title: 'Học 30 phút', body: 'Cây của bạn đang chờ bạn.', url: '/' }

/** Notify slice payload → notification. Unknown or cross-origin urls open the dashboard. */
export function parsePushPayload(text: string | null): PushPayload {
  if (!text) return { ...DEFAULT }
  try {
    const p = JSON.parse(text) as Partial<PushPayload>
    const url = typeof p.url === 'string' && p.url.startsWith('/') ? p.url : '/'
    return {
      title: typeof p.title === 'string' && p.title ? p.title : DEFAULT.title,
      body: typeof p.body === 'string' && p.body ? p.body : DEFAULT.body,
      url,
    }
  } catch {
    return { ...DEFAULT, body: text }
  }
}
```

`service-worker/sw.ts` (replaces the placeholder):
```ts
/// <reference lib="webworker" />
import { cleanupOutdatedCaches, precacheAndRoute } from 'workbox-precaching'
import { registerRoute } from 'workbox-routing'
import { NetworkFirst, StaleWhileRevalidate } from 'workbox-strategies'
import { parsePushPayload } from './push'

declare let self: ServiceWorkerGlobalScope

// App shell: routes, fonts, plant SVGs, icons (Frontend spec §3 "Static Assets").
precacheAndRoute(self.__WB_MANIFEST)
cleanupOutdatedCaches()

// §3 "User Progress & Pet Status: NetworkFirst" — synchronised state when
// online, last-known state when not. Cache name must match utils/session.ts.
registerRoute(
  ({ url, request }) => request.method === 'GET' && /\/api\/v1\/(quests\/daily|pet\/status)$/.test(url.pathname),
  new NetworkFirst({ cacheName: 'api-state', networkTimeoutSeconds: 5 }),
)

// §3 "Static Assets & Exercises: StaleWhileRevalidate" for anything the
// precache manifest did not cover (hashed chunks loaded later, fonts).
registerRoute(
  ({ request }) => ['style', 'script', 'font', 'image'].includes(request.destination),
  new StaleWhileRevalidate({ cacheName: 'assets' }),
)

self.addEventListener('push', (event) => {
  const payload = parsePushPayload(event.data?.text() ?? null)
  event.waitUntil(
    self.registration.showNotification(payload.title, {
      body: payload.body,
      icon: '/pwa-192x192.png',
      data: { url: payload.url },
    }),
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const url = (event.notification.data as { url?: string } | undefined)?.url ?? '/'
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then(async (clients) => {
      const existing = clients.find(c => 'focus' in c)
      if (existing) {
        await existing.focus()
        await existing.navigate(url)
        return
      }
      await self.clients.openWindow(url)
    }),
  )
})
```

`pwa-assets.config.ts`:
```ts
import { defineConfig, minimal2023Preset } from '@vite-pwa/assets-generator/config'

export default defineConfig({
  preset: minimal2023Preset,
  images: ['public/logo.svg'],
})
```

Generate the icons (local only; the PNGs are committed):
```sh
npm run pwa:assets
```
Expected: `public/pwa-64x64.png`, `pwa-192x192.png`, `pwa-512x512.png`, `maskable-icon-512x512.png`, `apple-touch-icon-180x180.png`, `favicon.ico` created. If the generator's preset names differ in the resolved version, keep the manifest `icons` in `nuxt.config.ts` in step with the files it produced.

- [ ] **Step 4: Run, build, inspect**

```sh
npx vitest run tests/unit/push.test.ts && npm run lint && npm run typecheck && npm run build
ls .output/public/sw.js .output/public/manifest.webmanifest
grep -c 'api-state' .output/public/sw.js
```
Expected: 3 passed; build clean; both files exist; `grep` prints ≥ 1.

- [ ] **Step 5: Commit**

```sh
cd .. && git add frontend/service-worker frontend/pwa-assets.config.ts frontend/public frontend/tests/unit/push.test.ts && git commit -m "frontend: service worker — §3 NetworkFirst/SWR routes, push + notificationclick, PWA icons"
```

### Task 14: Playwright smoke — `/login` against a stubbed API

**Files:**
- Create: `frontend/playwright.config.ts`, `frontend/tests/e2e/login.spec.ts`

No backend: every `**/api/v1/**` request is fulfilled by `page.route`, and Google's consent page is fulfilled with a stub so the redirect URL can be asserted. Service workers are blocked in the test browser so `page.route` sees every request.

- [ ] **Step 1: Config**

`playwright.config.ts`:
```ts
import { defineConfig } from '@playwright/test'

const port = 3100

export default defineConfig({
  testDir: 'tests/e2e',
  timeout: 30_000,
  retries: 0,
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    serviceWorkers: 'block',
  },
  webServer: {
    command: 'node .output/server/index.mjs',
    port,
    reuseExistingServer: false,
    timeout: 60_000,
    env: {
      PORT: String(port),
      HOST: '127.0.0.1',
      NUXT_PUBLIC_API_BASE: 'http://127.0.0.1:3199',
      NUXT_PUBLIC_GOOGLE_CLIENT_ID: 'test-client-id',
      NUXT_PUBLIC_STUB_ONBOARDING: 'true',
    },
  },
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }],
})
```

- [ ] **Step 2: Write the smoke**

`tests/e2e/login.spec.ts`:
```ts
import { expect, test } from '@playwright/test'

const SIGN_IN = {
  access_token: 'jwt-test',
  token_type: 'Bearer',
  expires_in: 86400,
  user: { id: 'u1', email: 'user@example.com', full_name: 'Nguyen Hendrix', cefr_current: 'B1' },
}
const DAILY = {
  date: '2026-09-23',
  day_number: 3,
  total_minutes_required: 30,
  accumulated_seconds: 1200,
  is_target_met: false,
  tasks: [
    { id: 'ex-1', task_type: 'vocabulary', title: 'Từ vựng Email Công việc', duration_minutes: 10, is_completed: true, content_json: {} },
    { id: 'ex-2', task_type: 'reading', title: 'Đọc hiểu Mẫu Thư Thương mại', duration_minutes: 10, is_completed: false, content_json: {} },
    { id: 'ex-3', task_type: 'practice', title: 'Viết Phản hồi Khách hàng', duration_minutes: 10, is_completed: false, content_json: {} },
  ],
}
const PET = { plant_name: 'My Green Buddy', health_points: 80, stage: 'sprout', current_streak: 5, last_practiced_at: '2026-09-22T13:00:00Z' }

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    const hit = path.endsWith('/auth/google') ? SIGN_IN : path.endsWith('/quests/daily') ? DAILY : path.endsWith('/pet/status') ? PET : null
    await route.fulfill({
      status: hit ? 200 : 404,
      contentType: 'application/json',
      body: JSON.stringify(hit ?? { error: 'not_stubbed' }),
    })
  })
  await page.route('https://accounts.google.com/**', route =>
    route.fulfill({ status: 200, contentType: 'text/html', body: '<title>google-stub</title>' }))
})

test('/login renders and sends the user to Google with the backend scope list', async ({ page }) => {
  await page.goto('/login')
  await expect(page.getByRole('heading', { name: /Chào mừng bạn/ })).toBeVisible()
  await page.getByRole('button', { name: 'Đăng nhập bằng Google' }).click()
  await page.waitForURL(/accounts\.google\.com/)
  const u = new URL(page.url())
  expect(u.searchParams.get('access_type')).toBe('offline')
  expect(u.searchParams.get('prompt')).toBe('consent')
  expect(u.searchParams.get('scope')).toContain('https://www.googleapis.com/auth/calendar.events')
  expect(u.searchParams.get('scope')).toContain('https://www.googleapis.com/auth/tasks')
  expect(u.searchParams.get('redirect_uri')).toBe('http://127.0.0.1:3100/login')
})

test('a guarded route without a token redirects to /login', async ({ page }) => {
  await page.goto('/')
  await page.waitForURL(/\/login$/)
  await expect(page.getByRole('button', { name: 'Đăng nhập bằng Google' })).toBeVisible()
})

test('the Google callback exchanges the code (§6.1 shape) and lands on the dashboard', async ({ page }) => {
  await page.addInitScript(() => sessionStorage.setItem('aelp.oauth_state', 'state-1'))
  await page.goto('/login?code=code-1&state=state-1')
  await page.waitForURL(/127\.0\.0\.1:3100\/$/)
  await expect(page.getByText('20 / 30 phút')).toBeVisible()
  await expect(page.getByText(/Streak: 5 ngày/)).toBeVisible()
  await expect(page.getByText('Đọc hiểu Mẫu Thư Thương mại')).toBeVisible()
  await expect(page.getByRole('link', { name: /Học: Đọc hiểu/ })).toBeVisible()
})
```

- [ ] **Step 3: Run**

```sh
npx playwright install chromium
npm run build && npx playwright test --reporter=list
```
Expected: `3 passed`. (`npx playwright install` is a one-time local download; it is not in CI — see Notes.)

- [ ] **Step 4: Commit**

```sh
cd .. && git add frontend/playwright.config.ts frontend/tests/e2e/login.spec.ts && git commit -m "frontend: Playwright smoke — /login, guard redirect, §6.1 callback against a stubbed API"
```

### Task 15: CI `frontend` job and CODEMAP

**Files:**
- Modify: `.github/workflows/ci.yml` (append one job; the three existing jobs are untouched)
- Modify: `harness/CODEMAP.md` (replace *Planned frontend areas*; add the CI bullet)

`actions/setup-node`'s current major is **v7** (v7.0.0, 2026-07-14; the action's README uses `@v7`) — verified against the GitHub API at planning time, matching the `@v7` majors the file already uses for `checkout`, `setup-go` and `setup-python`.

- [ ] **Step 1: Append the job** (after `harness-tooling`, same indentation; do not edit anything above it)

```yaml
  frontend:
    runs-on: ubuntu-latest
    timeout-minutes: 10
    defaults:
      run:
        working-directory: frontend
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-node@v7
        with:
          node-version: lts/*
          cache: npm
          cache-dependency-path: frontend/package-lock.json
      - name: Install
        run: npm ci
      - name: Lint
        run: npm run lint
      - name: Typecheck
        run: npm run typecheck
      - name: Unit tests
        run: npm run test:unit
      - name: Build
        run: npm run build
```

- [ ] **Step 2: Verify the workflow is still well-formed and the other jobs untouched** (repo root)

```sh
cd .. && git diff --stat main -- .github/workflows/ci.yml
git diff main -- .github/workflows/ci.yml | grep -c '^-[^-]'
grep -c '^  [a-z-]*:$' .github/workflows/ci.yml
grep -n 'TestIntegration' .github/workflows/ci.yml
```
Expected: one file changed, insertions only (`grep -c '^-[^-]'` prints `0`); four job keys (`backend-unit`, `backend-integration`, `harness-tooling`, `frontend`); the `TestIntegration` counting lines are unchanged (same two hits as on `main`). If `actionlint` is installed, `actionlint .github/workflows/ci.yml` prints nothing.

- [ ] **Step 3: CODEMAP** — replace the whole *Planned frontend areas* section with:

```markdown
## Frontend (`frontend/`)

- **shell** — Nuxt 3 SPA/PWA (`ssr: false`; Vue 3, Pinia via `@pinia/nuxt`, Tailwind via `@nuxtjs/tailwindcss`, `@vite-pwa/nuxt` with an `injectManifest` worker in `service-worker/sw.ts`). Design: `harness/designs/frontend-shell.md`. One API client `utils/apiClient.ts` (`createApiClient` — bearer from `useAuthStore`, `{error}` envelope → `ApiError{status, code}`, 401 → sign out + `/login`) behind `composables/useApi.ts`; every screen goes through it. Stores per Frontend spec §4: `stores/auth.ts` (§6.1 `{access_token, expires_in, user}` — **the spec shape, not the merged handler's `{token}`**; persisted at `localStorage['aelp.auth']`), `stores/quest.ts` (`GET /quests/daily`, per-task countdown timers, `POST /quests/progress` with `clampDuration` 1..3600), `stores/pet.ts` (`GET /pet/status`, `POST /pet/revive` with a client-side 15-minute challenge anchor at `localStorage['aelp.revive']`). Pages = the §7 wireframes: `/login` (7.1, Google consent URL built in `utils/googleAuth.ts` — **keep its scope list identical to `backend/internal/auth/scopes.go`**), `/onboarding` (7.1, against `stubs/onboarding.ts` while `NUXT_PUBLIC_STUB_ONBOARDING=true`), `/` (7.2 plant hub + three-segment day + quests), `/learn/:id` (7.3), `/roadmap` (7.4 — derived from `day_number`; no per-day endpoint), `/revive` (7.5), `/settings` (placeholder for notify/google). Route guard `middleware/auth.global.ts`. Service worker: §3 NetworkFirst (`api-state` cache — cleared by `utils/session.ts` on sign-out) for `/quests/daily` + `/pet/status`, StaleWhileRevalidate for assets, `push`/`notificationclick` via `service-worker/push.ts`. Runtime config: `NUXT_PUBLIC_API_BASE`, `NUXT_PUBLIC_GOOGLE_CLIENT_ID`, `NUXT_PUBLIC_VAPID_PUBLIC_KEY`, `NUXT_PUBLIC_STUB_ONBOARDING` (`.env.example`). Tests: `npm run test:unit` (Vitest, `happy-dom`, no Nuxt runtime — stores/utils import explicitly and tests `vi.mock('~/composables/useApi')`); `npm run build && npm run test:e2e` (Playwright, `page.route` stubs, service workers blocked; needs `npx playwright install chromium` once). Resolved majors at execution: <fill in from package-lock.json>.
```

And add to the CI section, after the `harness-tooling` bullet:

```markdown
- **`frontend`** — from `frontend/`: `actions/setup-node@v7` (`lts/*`, npm cache keyed on `frontend/package-lock.json`), `npm ci`, `npm run lint`, `npm run typecheck`, `npm run test:unit`, `npm run build`. Playwright is local-only (browser download; see the frontend-shell plan's Notes).
```

- [ ] **Step 4: Validate and commit**

```sh
python3 tools/harness/cli.py validate; echo exit=$?
git add .github/workflows/ci.yml harness/CODEMAP.md && git commit -m "ci: frontend job (setup-node@v7, lint, typecheck, unit, build); codemap: frontend shell"
```
Expected: `exit=0`.

## Verification

Run from the worktree root (`cd frontend` where shown). Everything is local — no backend, no Docker, no Google.

```sh
cd frontend && npm ci
# expect: clean install from the committed package-lock.json; postinstall runs `nuxi prepare`

npm run lint
# expect: no output (ESLint 9 flat config from @nuxt/eslint)

npm run typecheck
# expect: vue-tsc finishes with no errors

npm run test:unit
# expect: 13 test files, all passing — tokens, apiClient, googleAuth, authStore, progress, plant,
#         roadmap, questStore, petStore, SegmentedProgress, PlantSvg, content, onboardingStub, push

npm run build
# expect: Nitro output summary; a vite-plugin-pwa line naming sw.js and manifest.webmanifest

ls .output/public/sw.js .output/public/manifest.webmanifest
# expect: both files

grep -c 'api-state' .output/public/sw.js
# expect: >= 1 — the §3 NetworkFirst route survived the build

npx playwright install chromium && npx playwright test --reporter=list
# expect: 3 passed (login renders + Google URL, guard redirect, §6.1 callback → dashboard "20 / 30 phút")

grep -rn '"token"' stores/auth.ts pages/login.vue
# expect: no hits — the frontend reads access_token, never the merged handler's token

grep -n 'calendar.events\|auth/tasks' utils/googleAuth.ts ../backend/internal/auth/scopes.go
# expect: two hits in each file — the scope lists match

grep -rn '#10B981\|#F59E0B\|#EF4444\|#1E293B' --include='*.vue' --include='*.ts' . | grep -v tailwind.config.ts | grep -v nuxt.config.ts | grep -v node_modules | grep -v '.nuxt/' | grep -v '.output/'
# expect: no hits — colours come from tokens (nuxt.config theme_color/background_color are the manifest's copies)

grep -rn 'onboarding/quiz\|onboarding/assessment' composables/useOnboardingApi.ts
# expect: 2 hits — the real endpoints are wired behind the stub flag

grep -n 'NetworkFirst\|StaleWhileRevalidate\|addEventListener(.push.\|notificationclick' service-worker/sw.ts
# expect: 4 hits

cd .. && git diff main -- .github/workflows/ci.yml | grep -c '^-[^-]'
# expect: 0 — insertions only; backend-unit, backend-integration, harness-tooling untouched

grep -n 'setup-node@v7' .github/workflows/ci.yml
# expect: 1 hit

python3 tools/harness/cli.py validate; echo exit=$?
# expect: exit=0

git log --oneline main..HEAD
# expect: 15 commits, one per task, each with the Co-Authored-By trailer

git status --short
# expect: clean
```

After pushing the branch: `gh run list --branch <branch>` must show **four** green jobs — `backend-unit`, `backend-integration`, `harness-tooling`, `frontend`. Manual, noted for the reviewer: serve `.output/public` (or `npm run preview`) and run Lighthouse's PWA "installable" audit in Chrome — it must pass (manifest with 192/512 icons, registered service worker, `start_url` served).

**Click-through against the real backend (reviewer):** `backend` on `main` with `store.SeedDemoRoadmap` applied to the signed-in user; `frontend` with `NUXT_PUBLIC_API_BASE` pointing at it and a real `NUXT_PUBLIC_GOOGLE_CLIENT_ID` whose authorised redirect URI is `http://localhost:3000/login`. Expected until the auth-response bug is fixed: sign-in shows "Máy chủ trả về phiên đăng nhập không hợp lệ" (the §6.1 mismatch, made visible on purpose). With the bug fixed: dashboard renders the demo roadmap's three tasks in the raw-content viewer, "Hoàn thành" advances the segments, and `/pet/status` (once the pet branch merges) fills the hub.

## Notes and open questions

- **Merge blocker, restated.** This slice cannot merge until `harness/ideas/_inbox/auth-google-response-returns-token-and-omits-token-type-and-.md` is fixed on `main`: the frontend deliberately targets the Backend spec §6.1 shape and does not read `token`. Not fixed here (AGENTS.md: one slice, one layer). The reviewer should file it as a blocker against this plan at review time if it is still open.
- **Live vs stubbed at execution.** Live on `main`: `/login` (wrong shape, above), `/` progress + quests, `/learn/:id`, `/roadmap` (via `day_number`). Contract-wired to a merged-but-unreviewed branch: `/pet/status` and `/pet/revive` — the plant hub and `/revive` render the error state against `main` until the pet branch merges. Stubbed: `/onboarding` (quiz + assessment) behind `NUXT_PUBLIC_STUB_ONBOARDING`. Unwired: `/settings` buttons (notify, google).
- **Frontend spec §5 is not a matrix.** Its body is the backend architecture diagram; the matrix above is this plan's. The spec owner should replace §5 (a documentation fix, not a slice).
- **Six screens, not three.** The idea listed `/login`, `/`, `/pet`; the Frontend spec §7 draws five screens with the plant on the dashboard. The design and this plan follow §7 (spec wins for its layer); the idea's `## Evaluation` records it.
- **Roadmap "completed" is derived.** No endpoint reports per-day completion, so days before `day_number` are drawn as completed. A `GET /api/v1/roadmap` (or `daily_progress` in `/quests/daily`) would make 7.4 truthful — a backend idea for the ideator, not this slice.
- **Revive progress is a client-side anchor.** §6.3's response carries no challenge progress (pet plan Notes), so `usePetStore` records `accumulated_seconds` at challenge start in `localStorage`. The server's own anchor is authoritative; "Kiểm tra hồi sinh" re-posts and shows the truth. Extending the DTO with `challenge{seconds_remaining}` is a pet follow-up.
- **`SegmentedProgress` `met` label** reads "Mục tiêu hôm nay đã đạt ✓" on the 15-minute revive bar too; a `metLabel` prop is a one-line follow-up if the design owner wants distinct copy.
- **`$fetch` vs `fetch`.** The idea said "`$fetch` wrapper"; the client uses the platform `fetch` with an injectable override so it unit-tests without ofetch's internals. Behaviour is the same (JSON in/out, bearer header, error envelope).
- **Env names.** The idea's `API_BASE`, `GOOGLE_CLIENT_ID`, `VAPID_PUBLIC_KEY` are exposed as `NUXT_PUBLIC_API_BASE`, `NUXT_PUBLIC_GOOGLE_CLIENT_ID`, `NUXT_PUBLIC_VAPID_PUBLIC_KEY` (Nuxt's runtime-config convention). `vapidPublicKey` is carried but unused until the notify slice subscribes.
- **Offline progress queueing is out.** Frontend spec §2/§4 mention "offline progress queueing"; run-01 dropped offline quest caching into this slice, and the design (§5) shows last-known cards offline but disables "Hoàn thành" offline. Queueing `POST /quests/progress` for replay changes the backend's once-per-day hook semantics (quests CODEMAP) and is its own idea.
- **Playwright is not in CI.** The requested job is `npm ci`, lint, unit tests, build (plus `typecheck`, which the idea asked for). Adding the smoke would need `npx playwright install --with-deps chromium` (≈1 min, ~150 MB); recommended as a follow-up once the job is green for a few runs.
- **Nuxt 3 vs 4.** Nuxt 4 is current at planning time; the spec names Nuxt 3, so the plan pins `^3` and the classic directory layout (no `app/`). Migrating is a one-line `compatibilityVersion: 4` plus a folder move — a later idea.
- **Version floors.** Ranges were written without `npm`; see *Version ranges are floors* at the top. The executor records resolved majors in CODEMAP.
- **Tailwind 3 via `@nuxtjs/tailwindcss`, not Tailwind 4.** Chosen so `tailwind.config.ts` can export the `tokens` object that the tokens test imports; Tailwind 4's CSS-first config would move the palette into CSS. Either satisfies §6.1.
- **UI language.** Vietnamese, as every §7 wireframe is drawn; strings live in the components (no i18n module). If English UI is wanted for the reviewer, that is a copy pass, not a structural change.
- **`seed` stage** is in the DDL enum but never produced by the merged pet engine (default `sprout`); `PlantSvg` draws it anyway so an unexpected value degrades gracefully (design §3).
- **`AppHeader` menu** has no outside-click dismissal (MVP); Escape/second tap closes it.
