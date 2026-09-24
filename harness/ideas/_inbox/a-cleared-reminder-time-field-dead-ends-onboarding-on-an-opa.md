---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# A cleared reminder-time field dead-ends onboarding on an opaque error after ten answered questions

## Why
Onboarding is the one screen every user must pass to get a roadmap (§5.1). `pages/onboarding.vue:64` builds the §6.1 `notification_time` as `` `${time.value}:00` `` from a bare `<input type="time">` (`pages/onboarding.vue:24`, `:95`) that has no `required`, no validation and no gate — the "start the quiz" button is disabled only on `!goal` (`pages/onboarding.vue:97`). Clearing the field (a single click on the input's clear affordance, or any locale where the user wipes it) makes `time.value === ''`, so the page posts `notification_time: ":00"`.

I drove that exact body at the merged Go backend: `service.go validate()` parses it with `time.Parse("15:04:05", …)` and answers `400 {"error":"invalid_request"}`. `assessErrorMessage` (`pages/onboarding.vue:13-17`) has no `invalid_request` branch, so the learner — who has just answered all ten placement questions — sees only "Không tạo được lộ trình. Thử lại.", with nothing pointing at the time field. Every retry posts the same invalid body and fails identically; the only escape is a reload, which loses the answers. This is the last step before the user ever sees a quest, so the drop-off cost is the whole funnel.

The plan asserted the opposite — "the page already satisfies all of these: it sends `` `${time}:00` ``" — which holds only while the input is non-empty.

## Expected output
`/onboarding` cannot post a malformed `notification_time`: either the submit/start control is disabled while `time` is empty (matching how `goal` already gates it), or `next()` falls back to a default before building the body. Additionally `assessErrorMessage` names `invalid_request` distinctly, so a rejected body never renders as a generic "try again" that can only loop. A component test clears the time field, completes the quiz and asserts the learner is told what to fix.

## Evidence
- Plan: `harness/plans/2026-09-23-nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment.md` (review: `harness/reviews/2026-09-23-nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment.md`).
- `frontend/pages/onboarding.vue:64` — `notification_time: \`${time.value}:00\``; `:24` `const time = ref('20:00')`; `:95` the unguarded `<input type="time">`; `:97` `:disabled="!goal"`.
- `frontend/pages/onboarding.vue:13-17` — `assessErrorMessage` maps `rate_limited` and `ai_*` only; everything else, including `invalid_request`, falls through to the generic string.
- `backend/internal/onboarding/service.go` `validate()` — `time.Parse("15:04:05", req.NotificationTime)`.
- Reproduced against the real backend (Postgres 5453 / Redis 6401 / API 8107, `COMPOSE_PROJECT_NAME=onbrev`): body with `notification_time: ":00"` → `{"error":"invalid_request"} HTTP 400`; the same body with `"20:00"` → also `400`. The normal `"20:00:00"` path is fine (verified end to end: `users.notification_time` = `20:00:00`).
