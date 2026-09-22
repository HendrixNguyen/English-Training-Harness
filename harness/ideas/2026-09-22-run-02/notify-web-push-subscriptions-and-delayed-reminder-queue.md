---
type: mvp-slice
status: proposed
source: ideator
run: 2026-09-22-run-02
order: 7
---
# Notify: Web Push subscriptions and delayed reminder queue

## Why
A daily reminder at the time the user chose is the cheapest lever on the ≥30 min/day metric, and Web Push is the only channel a PWA has when the tab is closed. The §4 `queue:webpush:delay` ZSET plus an in-process cron means reminders fire from the single Go binary the spec deploys (§2.1, §8) with no extra infrastructure. Skipping the reminder once today's target is already met keeps the nudge credible instead of noisy — which is what run-01's "adaptive reminder timing" idea later tunes.

## Expected output
Delivers (Go package `backend/internal/notify`, route behind `auth.Require()`):
- `POST /api/v1/settings/notifications` — body `{subscription:{endpoint, keys:{p256dh, auth}}, notification_time, timezone}`; inserts into `push_subscriptions` (dedupe on `endpoint`), updates `users.notification_time` and `users.timezone`, then `ZADD queue:webpush:delay <next_fire_unix> <user_id>` where the next fire is the next occurrence of `notification_time` in the user's timezone. Response `{next_reminder_at}`. Subscription is optional in the body so the same route can update only the time.
- Reminder worker on the in-process cron (ticker every 30s): `ZRANGEBYSCORE queue:webpush:delay -inf <now>`, for each user: skip if today's `daily_progress.is_target_met` is already true, else send a Web Push (VAPID from `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`, e.g. `webpush-go`) to every subscription; delete subscriptions that return 404/410; `ZADD` the user for tomorrow's slot. Payload `{title, body, url:"/"}` handled by the PWA service worker.
- `VAPID_PUBLIC_KEY` is exposed to the client via Nuxt runtime config (no new endpoint).
- Tests: next-fire computation across DST and a non-UTC timezone; worker pops only due members and reschedules them; target-met users are skipped; 410 removes the subscription; duplicate `endpoint` is not inserted twice.
- Tables: `push_subscriptions` (insert/delete), `users` (update `notification_time`, `timezone`), `daily_progress` (read). Redis keys: `queue:webpush:delay`. Screens: none (subscribe button sits in the frontend-shell settings placeholder).

Depends on: store (1), auth (2); quests (3) for the `daily_progress` skip check (soft — the worker sends unconditionally if the table is empty).

## Evidence
- Spec §7 (line 682) `POST /api/v1/settings/notifications`: "Registers VAPID push subscription and updates preferred practice time".
- Spec §4 (line 265) `queue:webpush:delay` Sorted Set, persistent, "UNIX timestamps as scores to trigger scheduled Web Push reminders".
- Spec §3.2 `push_subscriptions` (lines 176–190) `endpoint`, `p256dh`, `auth`; `users.notification_time`, `users.timezone` (lines 168–170).
- Spec §2.1 (lines 12, 14) "integrated background cron worker", "Web Push delay queues"; §8 (line 700) `VAPID_PUBLIC_KEY` & `VAPID_PRIVATE_KEY`.
- `harness/CODEMAP.md` → `notify`: "Web Push, queue:webpush:delay ZSET, in-process cron".
- Prior run `harness/ideas/2026-09-22-run-01/adaptive-reminder-timing-and-pre-decay-rescue-push.md` builds on this worker.
