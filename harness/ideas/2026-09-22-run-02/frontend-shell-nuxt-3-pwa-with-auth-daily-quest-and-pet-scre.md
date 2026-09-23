---
type: mvp-slice
status: planned
source: ideator
run: 2026-09-22-run-02
order: 9
priority: high
plan: harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md
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

## Evaluation

**Verdict: select, `priority: high`.** This is the last `mvp-slice` in `order` (9); every backend slice before it is merged or on a branch, and none of them is visible to a learner until this client exists. Standing priority (AGENTS.md, 2026-09-22) puts the next unmerged MVP slice ahead of every inbox bug.

**Is the *Why* real?** Yes. Spec §2.1/§8 name a Nuxt 3 PWA as the delivery vehicle; §5.1 step 8 and §5.2 step 5 are client renders nothing else provides. The reviewer's end-to-end click-through argument holds: today the only way to exercise `auth → quests → pet` is `curl`.

**Achievable in one plan?** Yes, as a shell: scaffold + one API client + three Pinia stores (Frontend spec §4 names exactly `useAuthStore`, `useQuestStore`, `usePetStore`) + six thin pages against the Backend spec §6 DTOs. The Frontend spec's five wireframes (§7.1–7.5) are the screen list; the design doc `harness/designs/frontend-shell.md` derives layout, states and tokens from §6.1 and those wireframes. No polish beyond that doc.

**Where the plan departs from the idea's *Expected output*, and why (Frontend spec wins for its layer):**
- The idea lists three screens (`/login`, `/` daily quest, `/pet`). The Frontend spec §7 draws five: onboarding (7.1), dashboard + plant hub (7.2 — plant and quests on one screen, so there is no separate `/pet`), learning room (7.3), roadmap tree (7.4), revival mode (7.5). The plan builds all five plus `/login`; `/settings` stays a placeholder holding the two unwired buttons the idea asked for.
- The idea stores `{token, user}` — the shape the merged handler emits. The Backend spec §6.1 contract is `{access_token, token_type, expires_in, user}` and the frontend targets **the spec**. The inbox bug `harness/ideas/_inbox/auth-google-response-returns-token-and-omits-token-type-and-.md` is the backend fix; this slice cannot merge until it lands, and the plan says so.
- `POST /quests/progress` body is `{exercise_id, duration_seconds}` (§6.2), not `{exercise_id, seconds}`.
- Frontend spec §5 is titled "API Data to UI Mapping Matrix" but its body is a pasted copy of the backend architecture diagram; the plan supplies the matrix itself (endpoint → store → screen) from Backend spec §6 and the wireframes, and records the spec gap.

**Dependencies.** Live on `main`: `POST /api/v1/auth/google` (wrong response shape — see above), `GET /api/v1/quests/daily`, `POST /api/v1/quests/progress`. Also live on `main` since `c33fb73` (merged while this evaluation was written): `GET /api/v1/pet/status`, `POST /api/v1/pet/revive` (§6.3 shapes confirmed in `backend/internal/pet/handler.go`). Planned, not executed: `GET /api/v1/onboarding/quiz`, `POST /api/v1/onboarding/assessment` — the onboarding screen renders against a documented stub until that slice lands. Google sync and notify are unbuilt; their attach points are the two placeholder buttons. There is no endpoint for per-day roadmap completion, so the 7.4 tree is derived from `day_number`.

**Priority rationale.** `high`: MVP order, and it is the gate on the whole daily loop being usable by a human.
