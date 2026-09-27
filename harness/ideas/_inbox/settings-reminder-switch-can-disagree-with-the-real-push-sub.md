---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Settings reminder switch can disagree with the real push subscription after a failed time save or a cleared subscription

## Why
The switch position is derived from `reminders.state` (`on`/`requesting` → on), not from whether a subscription exists:
- With reminders **on**, a failed "Lưu giờ nhắc" sets `state = 'error'`, so the switch renders **off** while `store.remindersOn` is still `true` and the browser still holds the subscription. The design (§4.1 `error`: switch "unchanged … returns to its previous position") says it must stay on. Tapping the now-"off" switch calls `enable()` and re-subscribes on top of the existing subscription.
- `init()` sets `on` from `localStorage['aelp.settings'].remindersOn` alone. If the browser dropped the subscription (site data cleared, permission reset to `default`, push service expired it), the page says "Đang nhắc lúc 20:00 mỗi ngày." while nothing will ever arrive — the exact "never leave you guessing whether it worked" failure the design names.
- Same root, on save: `saveTime()` with `remindersOn` true but `currentSubscription()` null posts the time **without** `push_subscription` and still lands on `on` (`useReminders.ts` `saveTime`), so the page claims reminders are active when the server holds no subscription for this device.

## Expected output
After a failed save the switch keeps its prior position (on stays on) and the error line shows beneath it. `init()` confirms `getSubscription()` (async) before showing `on`; with no subscription it shows `off` and clears `remindersOn`. Tests: "a failed saveTime while on leaves the switch on", "init with remindersOn but no subscription lands on off".

## Evidence
- Plan `harness/plans/2026-09-24-settings-screen-wires-web-push-reminders-and-google-calendar.md`; design `harness/designs/settings.md` §4.1 (`error` row).
- `frontend/composables/useReminders.ts` `init()` (`state.value = store.remindersOn ? 'on' : 'off'`) and `saveTime()` failure branch; `frontend/pages/settings.vue` `switchOn` computed.
- Scratch Vitest (not committed): `setRemindersOn(true)`, `init()`, `api.post` rejects 500, `saveTime('08:00')` → `state = 'error'`, `store.remindersOn = true` (switch renders off).
