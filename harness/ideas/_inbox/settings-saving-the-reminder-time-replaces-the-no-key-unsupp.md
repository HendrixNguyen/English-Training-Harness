---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-24-settings-screen-wires-web-push-reminders-and-google-calendar.md
---
# Settings: saving the reminder time replaces the no-key, unsupported and denied states with off, exposing a switch that cannot work

## Why
`useReminders.saveTime()` ends with `state.value = store.remindersOn ? 'on' : 'off'` (and `'error'`/`'invalid'` on failure) regardless of the state it started in, and `pages/settings.vue` shows the switch for every state except `unsupported`/`no-key`. So "Lưu giờ nhắc" — the one control the design keeps in those states — breaks them:

- **no-key** (production today: the plan's own Notes say the switch stays hidden until `NUXT_PUBLIC_VAPID_PUBLIC_KEY` is set): after a save the "Bật nhắc học" switch appears. Tapping it fires the browser's notification-permission prompt, then `pushManager.subscribe` with an empty key fails → "Không lưu được. Thử lại." The learner granted a permission for nothing.
- **unsupported** (iPhone Safari not installed as a PWA — a large share of a mobile Vietnamese audience): after a save the switch appears and the "Add to Home Screen" explanation disappears. Tapping it throws `TypeError: Cannot read properties of undefined (reading 'requestPermission')` inside `enable()` (unhandled rejection) and the switch is stuck busy in `requesting` until reload.
- **denied**: after a save the "browser is blocking notifications" explanation is replaced by "Đang tắt" and the switch is re-enabled.

Happy-path defect on the feature this branch delivers; one-line class of fix.

## Expected output
Saving the time never changes the reminder card's availability state: from `no-key`, `unsupported` or `denied`, a successful save leaves that state (and its line / hidden switch) in place; a failed save shows the error line without revealing the switch. `enable()` refuses to run unless `pushSupport(...) === 'ok'` and permission is not `denied`. `useReminders.test.ts` pins all three (save from no-key / unsupported / denied keeps the state; enable on unsupported does not throw).

## Evidence
- Plan: `harness/plans/2026-09-24-settings-screen-wires-web-push-reminders-and-google-calendar.md` (design `harness/designs/settings.md` §4.1: "no-key … the switch and its line are not rendered; the time field and Lưu giờ nhắc stay").
- `frontend/composables/useReminders.ts` `saveTime()` (last lines: `state.value = store.remindersOn ? 'on' : 'off'`), `enable()` first lines (unguarded `win.Notification.requestPermission()`); `frontend/pages/settings.vue` `showSwitch` computed.
- Browser, branch merged into origin/main, `npx nuxi dev` with empty `NUXT_PUBLIC_VAPID_PUBLIC_KEY`: /settings shows no switch; set time 07:30, tap "Lưu giờ nhắc" → page text becomes "Bật nhắc học / Không lưu được. Thử lại." and `document.querySelector('[role=switch]')` is present.
- Scratch Vitest (not committed) on the fakes from `tests/unit/useReminders.test.ts`: after `saveTime('07:30')` states were `no-key→off`, `unsupported→off`, `denied→off`; `enable()` on the unsupported window rejected with `TypeError: Cannot read properties of undefined (reading 'requestPermission')` and left state `requesting`.
