---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Tick sends serially with a 10s client timeout so one slow push service delays every other reminder

## Why
`Tick` is one goroutine walking up to `DueBatchSize` (100) users, and for each user walking
every subscription, calling `sender.Send` synchronously. `WebPushSender`'s client has a 10 s
timeout. A single unresponsive push endpoint therefore stalls the entire pass for 10 s, and
the worst case for one pass is 100 users x subscriptions x 10 s — well over two hours of
wall clock inside a 30 s ticker.

`time.Ticker` drops ticks while the receiver is busy, so this does not pile up goroutines; it
simply means everyone else's reminder is delivered arbitrarily late, which is the one thing a
"remind me at 20:00" feature must not do. The reminder is the product's main retention lever
(spec §1, ≥ 30 min/day), and a reminder delivered at 22:30 is worse than none.

There is also no overall deadline on a pass: `Tick` is handed the worker's long-lived context,
so nothing bounds a pass to less than the poll interval. The sibling `airouter` package has
the same class of finding already filed (`route-has-no-overall-deadline...`).

## Expected output
- Sends are bounded in wall clock: either a small worker pool (`errgroup` with a limit of
  8–16) fanning out over the due users, or a per-pass `context.WithTimeout` shorter than
  `PollInterval` so a pass can never overrun its own schedule.
- The per-request timeout drops to something a push service should always beat (5 s) and is a
  named constant rather than a literal in the constructor.
- A test drives `Tick` with a fake sender that blocks, and asserts the pass returns within the
  configured bound with the remaining users still due (so they are retried), rather than
  hanging.

## Evidence
- Plan under review: `harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md`.
- `backend/internal/notify/service.go:109-165` — one serial `for _, userID := range due` loop; `service.go:152-164` the inner serial `for _, sub := range subs` send loop.
- `backend/internal/notify/push.go:62` — `HTTPClient: &http.Client{Timeout: 10 * time.Second}`.
- `backend/internal/notify/queue.go:16` — `DueBatchSize = 100`; `backend/internal/notify/schedule.go:12` — `PollInterval = 30 * time.Second`. 100 x 10 s >> 30 s.
- `backend/internal/notify/worker.go:15-33` — `RunWorker` calls `svc.Tick(ctx, at)` with the process-lifetime context; no per-pass deadline.
- Same class, already filed for a sibling package: `harness/ideas/_inbox/route-has-no-overall-deadline-so-one-call-can-take-90-second.md`, `harness/ideas/_inbox/synctimeout-gives-thirty-sequential-google-calls-a-two-secon.md`, `harness/ideas/_inbox/the-hourly-sweep-loads-every-pet-in-a-zone-into-memory-and-r.md`.
