---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Settings sync status always says hom nay for a sync persisted from an earlier day

## Why
`pages/settings.vue` `syncLine` renders `Đã đồng bộ · N nhiệm vụ · hôm nay HH:MM` from `lastSync.at`, which is persisted in `localStorage['aelp.settings']` and survives reloads (design §4.2). A learner who synced last week and opens settings today reads "hôm nay" (today) with last week's clock time — a false statement in the one line the design says is "the only place state is written".

## Expected output
"hôm nay HH:MM" only when `lastSync.at` is the local calendar day of now; otherwise a date (`dd/MM HH:MM` or "hôm qua HH:MM"). A page test with a `lastSync.at` two days old asserts the line does not contain "hôm nay".

## Evidence
- Plan `harness/plans/2026-09-24-settings-screen-wires-web-push-reminders-and-google-calendar.md`; design `harness/designs/settings.md` §4.2 `synced` row.
- `frontend/pages/settings.vue` `syncLine` computed (hard-coded `hôm nay`).
