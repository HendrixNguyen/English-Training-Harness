---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Settings reminders: requesting renders the switch on, a stale saveError can mislabel a failed enable, and several guards are mutation-proof in tests

## Why
Smaller defects and test gaps found in the settings reminder code (test-gap pass, confirmed against the code):
- `pages/settings.vue` `switchOn` includes `'requesting'`, so the switch shows **on** while permission and subscribe are in flight. Design §4.1 `requesting` says "off, disabled". A dismissed prompt then visibly flips on → off.
- `composables/useReminders.ts` `enable()`: `if (!flat || !(await store.saveReminder(...)))` short-circuits when `flattenSubscription` returns null. `saveReminder` is never called, so the next line reads the `saveError` left over from an earlier action and may show "Giờ nhắc không hợp lệ." for an enable that never reached the server.
- Mutation-surviving tests:
  - `tests/unit/settingsPage.test.ts`: the sync-button lookup `button[type="button"]:not([role])` matches any role-less button, and `unsupported` hiding the switch is never asserted.
  - `tests/unit/pushClient.test.ts`: `auth`-missing alone and `Notification`-missing alone are never tested, so dropping either guard survives.
  - `tests/unit/useReminders.test.ts`: a dismissed prompt (`requestPermission` → `'default'` → `off`) is never exercised.

## Expected output
- `requesting` renders the switch off and disabled with the busy style.
- `enable()` clears `saveError` itself and maps an incomplete subscription to `error`, then unsubscribes it.
- Tests for each gap above: an `unsupported` page test, pushClient cases for `auth` alone and `Notification` alone, an enable test with `'default'`, a flat-null enable after a prior `invalid`, and a sync-button lookup by its label.

## Evidence
- Plan `harness/plans/2026-09-24-settings-screen-wires-web-push-reminders-and-google-calendar.md`; design `harness/designs/settings.md` §4.1.
- `frontend/pages/settings.vue` `switchOn`; `frontend/composables/useReminders.ts` `enable()`; `frontend/tests/unit/settingsPage.test.ts`, `pushClient.test.ts`, `useReminders.test.ts` (branch head `1f1922a`).
