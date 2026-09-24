---
type: feature
status: selected
source: ideator
run: 2026-09-22-run-01
priority: low
---
# Adaptive Reminder Timing and Pre-Decay Rescue Push

## Why
The spec sends one Web Push at a fixed `users.notification_time` chosen during onboarding, before the user knows when they actually study. A randomised trial on a mobile language-learning programme found reminded learners hit 82% daily-practice adherence versus 67% self-scheduled, and industry data show notifications timed to a user's own historical activity window and referencing their current state outperform generic ones by roughly half. We already hold the two signals needed: `daily_progress`/`daily:accumulated` timestamps tell us when this user really practises, and the live `daily:accumulated` counter tells us, hours before midnight, whether the pet is about to decay. Turning those into (a) a reminder at the user's observed practice time and (b) a single "your plant needs 12 more minutes" rescue push directly attacks the two failure modes behind the 30-minute target: forgetting to start, and stopping short.

## Expected output
User-visible:
- Settings shows the reminder time as "Auto (learned from your practice, currently 19:40)" with an override to a fixed time; the default for new users after 7 practice days is Auto.
- If, at 2 hours before local midnight, the day's accumulated seconds are below 1800 and no shield/revive is pending, the user gets one rescue push stating the exact minutes remaining and the pet's current health. Never more than one rescue push per day; none on days already met.

Technical:
- `users` gains `reminder_mode ENUM('fixed','auto') DEFAULT 'fixed'` and `learned_notification_time TIME NULL`; `daily_progress` gains `first_activity_at TIMESTAMPTZ NULL` (set on first `POST /quests/progress` of the day).
- `notify` package nightly cron computes the median `first_activity_at` local time over the last 14 target-met days and writes `learned_notification_time`; ZSET scheduling uses it when `reminder_mode='auto'`.
- `notify` rescue job: hourly scan of users whose local time is within the T-2h window, reads `daily:accumulated:{user}:{date}`, enqueues one push with a `rescue:{user}:{date}` Redis flag (TTL 48h) to guarantee idempotence.
- `POST /settings/notifications` accepts `reminder_mode`; `GET` returns learned time. Unit tests for median computation, timezone handling, and once-per-day guard.
- CODEMAP `notify` and `quests` paragraphs updated.

## Evidence
- Spec §3.1 `users.notification_time`, `users.timezone`, `daily_progress`; §4 `daily:accumulated:*` (48h TTL) and `queue:webpush:delay` ZSET; §5.2 daily loop; §1 Web Push as a named retention lever.
- CODEMAP: `notify` package (Web Push, ZSET, in-process cron, `POST /settings/notifications`), `quests` (`INCRBY daily:accumulated:*`).
- RCT on push reminders and daily language-practice adherence: https://journals.kmanpub.com/index.php/aitechbesosci/article/view/4724
- Push notifications and learner engagement in a mobile learning app: https://www.researchgate.net/publication/311317244_Effects_of_Push_Notifications_on_Learner_Engagement_in_a_Mobile_Learning_App
- Duolingo reminder architecture breakdown: https://www.digia.tech/post/duolingo-habit-forming-reminders-retention-architecture/

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low.**

*Is the Why real?* Yes — a reminder at the learner's observed practice time and a single "12 more minutes" rescue push attack the two failure modes of the 30-minute target, and the signals exist (`daily:accumulated`, `daily_progress`, the notify queue).

*Achievable in one plan?* No. The *Expected output* is two features (a learned reminder time with a nightly median job and a new `reminder_mode`, and an hourly rescue-push job with its own idempotence key), a `users` + `daily_progress` migration, and settings-endpoint changes — at least two plans. *Dependencies not yet built:* no browser has ever subscribed (`push_subscriptions` is empty until the settings screen — `2026-09-24-run-01/settings-screen-…`, selected medium — ships) and the worker only starts with VAPID keys set on Railway, so no user can receive a rescue push today.

*Priority.* Low until the settings screen lands; then plan the rescue push first (it needs only the counter and the existing queue) and the learned time later, as separate plans.
