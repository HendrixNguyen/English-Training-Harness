---
type: mvp-slice
status: proposed
source: ideator
run: 2026-09-22-run-02
order: 9
---
# Frontend Shell: Nuxt 3 PWA with auth, daily quest and pet screens

## Why
None of slices 1–7 is visible to a learner until there is a client, and the spec's reach goal is a PWA — installable, service-worker cached, able to receive push (§2.1, §8). Standing up the Nuxt 3 shell with the three screens that make up the daily loop (log in → do today's quests → watch the plant grow) is the smallest thing a user can actually retain on. Placeholder screens wired to the real endpoints also give every backend slice an end-to-end check the reviewer can click through.

## Expected output
Delivers (`frontend/`):
- Nuxt 3 app initialised with Pinia, Tailwind CSS and `@vite-pwa/nuxt` (web manifest, service worker precaching the app shell, installable; a `push` event handler that shows the notify payload and opens its `url`). Runtime config: `API_BASE`, `GOOGLE_CLIENT_ID`, `VAPID_PUBLIC_KEY`.
- `composables/useApi.ts` — `$fetch` wrapper adding `Authorization: Bearer <jwt>` from the Pinia `auth` store (persisted to `localStorage`), redirecting to `/login` on 401. Route middleware guards every page except `/login`.
- Screen `/login` (auth flow): Google sign-in button using the same scope list as the auth slice with `access_type=offline`; on callback posts `{code, redirect_uri}` to `POST /api/v1/auth/google`, stores `{token, user}`, routes to `/`.
- Screen `/` (daily quest): calls `GET /api/v1/quests/daily`; renders three task cards (vocabulary, reading, practice) with a 10-minute timer each; "Done" posts `{exercise_id, seconds}` to `POST /api/v1/quests/progress`; shows a 0–30 min progress bar from `total_seconds`; renders the 404 `no_active_roadmap` state as a placeholder card.
- Screen `/pet` (pet view): calls `GET /api/v1/pet/status`; shows plant name, a stage placeholder graphic per `pet_stage` value (including `wilted`), a health bar, and the streak; a "Revive" button visible only at health 0 posts to `POST /api/v1/pet/revive`.
- Placeholder routes `/onboarding` and `/settings` ("coming soon"); the settings placeholder holds an unwired "Enable reminders" and "Sync to Google" button so notify/google can attach later without a new screen.
- Verification: `npm run build` and `npx nuxi typecheck` pass; Vitest unit tests for the `auth` store (token persistence, 401 clears) and the quest progress bar; the built app passes the PWA "installable" check in Lighthouse (manual, noted in the plan).
- Screens: auth flow, daily quest screen, pet view, PWA shell (from CODEMAP "Planned frontend areas"); onboarding wizard and settings as placeholders. Endpoints consumed: `POST /api/v1/auth/google`, `GET /api/v1/quests/daily`, `POST /api/v1/quests/progress`, `GET /api/v1/pet/status`, `POST /api/v1/pet/revive`. Tables/Redis: none directly.

Depends on: auth (2), quests (3), pet (4) for API contracts; notify (7) and google (6) are optional attach points.

## Evidence
- Spec §2.1 (line 11) "Nuxt 3 (Vue 3, Pinia state management, Tailwind CSS, @vite-pwa/nuxt PWA module, Service Workers)".
- Spec §2.2 architecture diagram: Nuxt 3 PWA Client ↔ Go API Server over REST.
- Spec §5.1 step 8 "Init Dashboard" and §5.2 step 5 "Render Growth Animation & Status" — the two client renders this shell must provide.
- Spec §7 (lines 670–684) endpoint list consumed above.
- Spec §8 (line 704) "Deploy Nuxt 3 PWA with @vite-pwa/nuxt configured for service worker caching".
- `harness/CODEMAP.md` "Planned frontend areas": auth flow, onboarding wizard, daily quest screen, pet view, settings, PWA shell.
- Prior run `harness/ideas/2026-09-22-run-01/_run.md` Notes: offline quest caching dropped as "better as part of the frontend-shell MVP slice".
