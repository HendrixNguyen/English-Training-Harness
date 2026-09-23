---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Due plus re-slot is not atomic so two API instances double-send the same reminder

## Why
The "exactly once" property of the reminder queue rests on `Tick` re-slotting a member before
it sends. That is crash-safe within one process, and the executor proved it live. It is not
safe across processes: `Due` is a plain `ZRANGEBYSCORE` read and the re-slot is a separate
`ZADD`, with a `repo.Preferences` round trip in between. Two processes that call `Due` in that
window both see the member and both send.

This is not hypothetical on Railway. Even with a replica count of 1, a rolling deploy runs the
old and new container concurrently for the overlap window, and any due reminder in that window
fires twice. If the service is ever scaled to 2 replicas, every reminder fires twice, every
day.

The limitation is honestly documented (plan *Notes* → "Exactly one worker"; `worker.go:13-14`
"Run exactly one worker per deployment"), so this is a hardening item rather than a
misrepresentation — but the documentation lives in a Go doc comment and a plan file, and
nothing in the deployment configuration enforces it or fails loudly if it is violated.

## Expected output
The pop is atomic, so a member can be claimed by exactly one process:

- either `ZPOPMIN`-style claiming behind a Lua script that pops only members with
  `score <= now` and immediately re-adds them at the next occurrence in the same script, or
- a `SET notify:lock:{user_id} <instance> NX EX <PollInterval*2>` guard taken before the
  send, or
- a single process-wide leader lock (`SET notify:worker:leader NX EX 60` refreshed each tick)
  so only one replica ticks at all — the smallest change and a good match for the spec's
  single-binary deployment.

A test proves it: two `Service` values over the same real Redis both `Tick` at the same `now`
against one due member, and the fake sender records exactly one send.

## Evidence
- Plan under review: `harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md`; the limitation is recorded in its *Notes and open questions* → "**Exactly one worker.**"
- `backend/internal/notify/queue.go:43-53` — `Due` is `ZRangeByScore(-inf, now, Count: limit)`, a read with no claim.
- `backend/internal/notify/service.go:102` (`s.queue.Due`) then `service.go:110` (`repo.Preferences`) then `service.go:127` (`s.queue.Schedule`) — the claim window spans a Postgres round trip.
- `backend/internal/notify/worker.go:13-14` — the doc comment that carries the whole constraint.
- Spec §2.1 / §8 describe a single binary, but nothing in the repo pins the replica count, and no deployment manifest exists (`find . -iname 'Dockerfile*' -o -iname 'railway*' -o -iname 'nixpacks*'` returns nothing).
