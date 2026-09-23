---
type: mvp-slice
status: planned
source: ideator
run: 2026-09-22-run-02
order: 8
priority: high
plan: harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md
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

## Evaluation
**Verdict: select, `priority: high`** — MVP slice, `order: 8`, the last backend slice; store (1), auth (2), quests (3) and airouter (5) are merged on `main`, pet (4) is executing and onboarding (6) is approved, so with google (7) planned this is next in `order`.

**Is the *Why* real?** Yes. §7 names the endpoint, §4 names the `queue:webpush:delay` ZSET (already `store.WebPushDelayQueueKey`), §2.1 names the in-process cron worker and §9 injects `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`. A reminder at the learner's chosen time is the cheapest lever on the ≥ 30 min/day metric and Web Push is the only channel a closed PWA has; skipping users who already met the target keeps it credible.

**Achievable in one plan?** Yes: one package (`backend/internal/notify`), one route, one Redis ZSET, one Postgres table written (`push_subscriptions`) and one column pair updated (`users.notification_time`/`timezone`), one worker goroutine following pet's `RunHourly` pattern. ≤ 1 day.

**Contract corrections against the backend spec §6.4 (which wins for the wire shape):** the request is `{"notification_time":"20:00:00","push_subscription":{"endpoint","p256dh","auth"}}` — flat `p256dh`/`auth`, not the idea's `subscription.keys.{p256dh,auth}` — and the response is `{"status":"updated","notification_time":"20:00:00"}`, not `{next_reminder_at}`. Two **additive** fields the plan keeps because the feature needs them and the spec has the columns: optional request `timezone` (§3.2 `users.timezone`; without it the "next local send time" is meaningless for a user who has not onboarded) and response `next_reminder_at` (RFC3339, so the client can show when the reminder fires). Both recorded as open questions.

**Target-met check:** the idea proposed reading `daily_progress.is_target_met`; the plan instead reads the §4 `daily:accumulated` counter through a `StudyCounter` interface satisfied by `quests.RedisCounter.Total` — the same arrangement the pet plan uses — because the counter is written first and is the source of truth for the day (quests CODEMAP), and because reading quests' table from notify would cross the package boundary.

**Library:** `github.com/SherClockHolmes/webpush-go` v1.4.0 — the only maintained pure-Go RFC 8291/8292 implementation; its `Options.HTTPClient` interface lets tests point the push at an `httptest.Server`; its `GenerateVAPIDKeys` makes tests self-contained; it depends only on `golang-jwt/jwt/v5` and `golang.org/x/crypto`, both already in `go.mod`.

**Dependencies:** all merged. **Worker wiring depends on pet (4) landing first** — pet's plan Task 9 introduces the `studyCounter := quests.NewRedisCounter(rdb)` local and the `go pet.RunHourly(ctx, petSvc)` pattern in `cmd/api/main.go` that notify reuses; the plan says to stop if pet is not merged when it executes. `VAPID_*` are optional at boot (worker starts only when both are set), so the existing dev/CI environments keep booting.
