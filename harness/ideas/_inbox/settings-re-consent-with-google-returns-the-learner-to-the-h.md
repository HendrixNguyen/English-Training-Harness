---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Settings re-consent with Google returns the learner to the hub instead of settings, so the sync has to be found and tapped again

## Why
On `409 reauth_required`, "Cho phép lại với Google" stores the OAuth state and sends the learner to Google with `redirect_uri = <origin>/login`. `pages/login.vue` `finishSignIn` always ends with `navigateTo('/', { replace: true })`, so after re-consenting the learner lands on the hub, not on `/settings`, and nothing tells them the sync still has not run. They must remember to open settings and tap "Đồng bộ với Google" again; many won't, and the calendar never gets the event.

## Expected output
Re-consent started from `/settings` returns to `/settings` (e.g. a `sessionStorage` return-path key set next to `OAUTH_STATE_KEY`, read and cleared by `finishSignIn`; only same-origin relative paths accepted), and the page either re-runs the sync once or shows the idle "Đồng bộ với Google" with a line that consent succeeded. Tests: login honours a stored `/settings` return path and ignores an absolute URL.

## Evidence
- Plan `harness/plans/2026-09-24-settings-screen-wires-web-push-reminders-and-google-calendar.md` (Task 4 `onSync`); design `harness/designs/settings.md` §4.2 `reauth_required` row.
- `frontend/pages/settings.vue` `onSync()`; `frontend/pages/login.vue` `finishSignIn` → `navigateTo('/', { replace: true })`.
