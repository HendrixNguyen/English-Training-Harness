---
type: feature
status: planned
source: ideator
run: 2026-09-24-run-01
priority: medium
plan: harness/plans/2026-09-24-settings-screen-wires-web-push-reminders-and-google-calendar.md
---
# Settings screen wires Web Push reminders and Google Calendar sync to the shipped backend

## Why
Two of the spec's four retention levers are finished on the server and unreachable by any user. `POST /api/v1/settings/notifications` (notify slice) and `POST /api/v1/integrations/google/sync` (google slice) are merged, tested and covered by CI, but `frontend/pages/settings.vue` is still the frontend-shell placeholder: a "Sắp ra mắt" card with both buttons `disabled`. No browser ever calls `pushManager.subscribe`, so `push_subscriptions` stays empty, the notify worker has nobody to remind, and the 30-minute daily target (1st-thinking §1) relies entirely on the learner remembering. Onboarding does not trigger the Google push either, so the recurring 30-minute calendar block and daily checklist of §5.1 steps 6–7 never appear in anyone's Google account.

This is the cheapest retention win left: no new endpoints, no schema change, only the UI the Frontend spec §5 already maps ("Settings & Integration Modal" → both endpoints). Daily reminders at a fixed time are the backbone of Duolingo's habit loop, which it credits with a large share of its drop in daily churn among its best users. Each day the placeholder stays shipped is a day of backend work producing zero user value.

## Expected output
User-visible:
- `/settings` shows a **daily reminder** block: time picker (prefilled from the user's `notification_time`), an "Enable reminders" toggle that asks for notification permission, subscribes via the service worker's `pushManager.subscribe({ applicationServerKey: NUXT_PUBLIC_VAPID_PUBLIC_KEY })` and posts `{notification_time, timezone, push_subscription{endpoint, p256dh, auth}}` — **flat keys**, flattened from `PushSubscription.toJSON().keys` (this closes the inbox bug *nothing-tells-the-pwa-to-flatten-pushsubscription*). It shows the returned `next_reminder_at` in local time.
- Clear states for: permission denied (explain how to re-enable in the browser), push unsupported (iOS Safari outside an installed PWA — suggest "Add to Home Screen"), VAPID key missing in config (block hidden, not broken).
- A **Google sync** block: "Sync to Google Calendar & Tasks" button that calls the sync route and shows "Synced — N daily tasks" from `tasks_created_count`; `409 reauth_required` sends the user through Google consent again; `502 google_unavailable` shows a retry.
- Onboarding's success step offers both actions once (turn on reminders, add to Google Calendar) so a new learner does not have to find the settings page.

Technical:
- A `useNotificationSettings` (or store) module and a `useGoogleSync` module going through `useApi`; no backend change.
- Vitest covers the flattening, each error code → UI state, and the permission-denied path; a Playwright test stubs both routes.
- CODEMAP's frontend paragraph drops "`/settings` (placeholder …)".

## Evidence
- Frontend spec §5 API→UI table, row "Settings & Integration Modal" (`POST /api/v1/settings/notifications`, `POST /api/v1/integrations/google/sync`); §4 `useAuthStore` "notification settings".
- 1st-thinking §1 (30 min/day goal), §5.1 steps 6–7 (Google push at onboarding).
- CODEMAP **notify**, **google** (both shipped), **shell** ("`/settings` (placeholder for notify/google)"); `frontend/pages/settings.vue` on `main` 2026-09-24 has both buttons `disabled`.
- Inbox: `nothing-tells-the-pwa-to-flatten-pushsubscription-tojson-so-.md` (the payload-shape trap this feature must avoid).
- Duolingo reminder/streak retention analyses: https://www.digia.tech/post/duolingo-habit-forming-reminders-retention-architecture/ , https://medium.com/@siddhartha-arora102/product-stories-how-duolingo-reignited-growth-by-mastering-retention-gamification-15b6d190b840
- Prior run `2026-09-22-run-01` proposed *adaptive reminder timing* — that idea tunes reminders; this one makes any reminder possible at all.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — medium. Not planned today; the next feature to plan.**

*Is the Why real?* Yes: `POST /settings/notifications` and `POST /integrations/google/sync` are merged, tested and unreachable — `frontend/pages/settings.vue` is the shell's placeholder with both buttons disabled, `push_subscriptions` stays empty, and §5.1 steps 6–7 never happen for anyone.

*Achievable in one plan?* Yes, about a day, and it needs no backend change. It requires `harness/designs/settings.md` first (frontend-design skill: reminder block states — permission denied, push unsupported, VAPID key absent; Google block states — synced N, `409 reauth_required`, `502 google_unavailable`) and must destructure `PushSubscription.toJSON().keys` into the flat `{endpoint, p256dh, auth}` body (the rejected `nothing-tells-the-pwa-to-flatten-…` bug is folded here) and add that note to CODEMAP's frontend paragraph. The onboarding success-step prompt is in scope only if it stays a link to `/settings`.

*Dependencies.* Inert in production until the CORS plan (approved today) is merged and `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY` are set on Railway. *Priority.* Medium — clear retention value, nothing blocked; as a medium feature its plan will stay `draft` for the owner's `/approve`.

## Evaluation — 2026-09-24 daily planning (evaluator)
_Owner instruction 2026-09-24: pick ≤ 5 one-day tickets from the `selected` backlog, split Bug team / Feature team, write and approve the plans, one planning PR._

**Planned today — Feature team ticket F1. Estimate 6 h. Approved by owner override (see below).**
*Still true on `main`.* `frontend/pages/settings.vue` is the disabled placeholder; `POST /settings/notifications` and `POST /integrations/google/sync` are merged and tested on `main` (`backend/internal/notify`, `backend/internal/google`); `service-worker/sw.ts` already handles `push`/`notificationclick`; `runtimeConfig.public.vapidPublicKey` already exists. Nothing on the server needs to change.
*Design.* `harness/designs/settings.md` (frontend-design skill) — two blocks, every state enumerated, copy in Vietnamese per the shell design.
*Decisions.* (1) No `GET` for settings exists, so the reminder time is pre-filled from the client's own last submission (`localStorage['aelp.settings']`, written by onboarding and by this screen), default `20:00`; a read endpoint is a later backend idea, not this ticket. (2) "Turn off" = `PushSubscription.unsubscribe()` client-side; the server prunes the dead endpoint on its next 404/410 — no unsubscribe endpoint is added. (3) `409 reauth_required` re-uses `googleAuthUrl` (already `prompt=consent`). (4) Onboarding's result step gets one ghost link to `/settings`; no second copy of the blocks.
*Approval.* Medium feature → would stay `draft` under the auto-approve rule; the owner's 2026-09-24 planning instruction approves today's five plans explicitly. Recorded here and in the commit message.
*One-day check.* Frontend only: one page, one store, one composable, one util, tests; CI `frontend` job. Inert in production until `VAPID_*` are set on Railway (documented, not blocking).
