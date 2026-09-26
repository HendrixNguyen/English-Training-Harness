---
type: feature
status: proposed
source: ideator
run: 2026-09-26-run-01
order: 5
---

## Why
Every reminder path in the product — the 30 s `notify` worker, `queue:webpush:delay`, the settings screen that saves a `push_subscription` — is inert until two things the app never asks for happen: the PWA is installed and notification permission is granted. On iPhone that is not a nicety but a hard requirement: Safari only delivers Web Push to web apps added to the Home Screen, and only after a permission request made from a user tap inside that installed app. Today nothing in `frontend/` handles `beforeinstallprompt`, detects `display-mode: standalone`, or asks for permission outside the settings page, and the settings design's "unsupported" row tells an iPhone user the recipe only if they wander there. A learner who finishes day 1 in the browser tab will get no reminder on day 2, and the plant will lose 30 health for a miss the app engineered. The moment to ask is the one retention research and the spec's own loop agree on — right after the first target-met celebration, when the learner has just seen the plant grow (growth-moment plan) and has a reason to want it kept alive. Asking then, once, with the reminder time they chose at onboarding already filled in, is the difference between a reminder system that exists and one that fires.

## Expected output
User-visible:
- After the first target-met moment (and, if dismissed, again on the first hub open of the next met day, max twice), the hub shows one card under the quests: "Cài Học 30 phút vào màn hình chính để {plant_name} nhắc cậu lúc 20:00." with the primary action **Cài đặt** and a quiet "Để sau".
  - Android/Chromium: **Cài đặt** fires the captured `beforeinstallprompt`; on `appinstalled` the card switches to step 2.
  - iOS Safari (not standalone): the card shows the two-step recipe (Chia sẻ → Thêm vào Màn hình chính) with the share glyph; when the app is next opened standalone, step 2 appears.
  - Already installed (`display-mode: standalone` or `navigator.standalone`): step 1 is skipped.
- Step 2 — "Bật nhắc học lúc 20:00" — requests notification permission on tap, subscribes with the VAPID key, and posts the existing settings body (`notification_time` from onboarding, `timezone`, `push_subscription`) through the settings store; success → "Tớ sẽ nhắc cậu lúc 20:00." and the card goes away for good. `denied` → the settings design's denied copy, card gone. Unsupported → card never shows.
- The card never appears on `/learn`, `/revive` or during the growth timeline; it waits for the timeline's `done`.

Technical (frontend only):
- `composables/useInstallPrompt.ts` (`beforeinstallprompt` capture on app boot in `app.vue`, `prompt()`, `installed` from `appinstalled` + the two standalone checks, `platform: android|ios|other`); `components/hub/InstallNudge.vue`; dismissal/state in `localStorage[aelp.nudge]` through the `storageOrNull()` guard (`{shownCount, done}`); permission + subscribe reuse `stores/settings.ts`/`utils/push.ts` from the settings plan (done, awaiting merge). No backend change: `POST /api/v1/settings/notifications` already accepts the flat body.
- Design from the designer role inside the retro-hub design (a `RetroPanel` with the companion as speaker; Vietnamese copy above is a draft for them). Reduced motion: none needed.
- Unit tests: platform detection matrix (standalone, iOS Safari, Chromium with/without the event), the twice-max rule, dismissal persistence, step 2 wiring against a mocked settings store, and that the card is absent while a growth delta is pending.
- Estimate: half a day.

## Evidence
- Spec: 1st-thinking §1 (Web Push as a retention driver; PWA reach), §4 (`queue:webpush:delay`); backend spec §6.4 (`POST /settings/notifications` body — unchanged); frontend spec §2 (`@vite-pwa/nuxt`, installable), §7.1 (reminder time chosen at onboarding — the value the nudge reuses).
- Code: `grep -rn beforeinstallprompt\|standalone\|requestPermission frontend` → only `nuxt.config.ts:52` (`display: standalone` in the manifest); `harness/designs/settings.md:71-72` (denied/unsupported rows with the iOS recipe — reused, not duplicated); CODEMAP `notify` (the worker only sends to stored subscriptions; nothing subscribes users today outside `/settings`).
- Harness context: `harness/plans/2026-09-24-settings-screen-…` (done; provides `stores/settings.ts`, `utils/push.ts`); `harness/plans/2026-09-25-growth-moment-…` (planned; the `done` moment this card waits for); `harness/ideas/2026-09-22-run-01/adaptive-reminder-timing-and-pre-decay-rescue-push.md` (selected; every push it schedules needs this subscription to exist).
- Research: iOS/iPadOS 16.4+ deliver Web Push only to Home Screen web apps and only after a permission request from direct user interaction — https://webkit.org/blog/13878/web-push-for-web-apps-on-ios-and-ipados/ and https://webkit.org/blog/13966/webkit-features-in-safari-16-4/ ; reminder architecture as the retention backbone (Duolingo) — https://www.digia.tech/post/duolingo-habit-forming-reminders-retention-architecture/
