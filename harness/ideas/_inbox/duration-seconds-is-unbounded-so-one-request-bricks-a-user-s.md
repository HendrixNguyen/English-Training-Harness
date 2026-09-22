---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md
plan: harness/plans/2026-09-22-a-rejected-post-quests-progress-still-writes-redis-and-daily.md
---

# duration_seconds is unbounded so one request bricks a user's day for 48h

## Why
`duration_seconds` is validated only as `binding:"required,gt=0"` (`backend/internal/quests/handler.go:39`)
and `seconds <= 0` in the service. There is no upper bound, per call or per day. The plan records
the *trust* decision deliberately ("§5.2 shows the client reporting duration, so this matches the
spec") — this bug is not about that. It is about what happens to the unbounded value downstream.

`minutes_spent` is `INT` (spec §3.2 / `0001_init.up.sql:48`), so once the Redis total exceeds
~1.29e11 seconds the upsert fails with an integer-out-of-range error. The INCRBY has already
committed, and it is never rolled back or compensated, so the counter stays poisoned for the whole
48h TTL and **every subsequent progress call for that user fails with 500**. Reproduced against the
real binary and real services:

```
POST /quests/progress {"duration_seconds":1000000000}      -> 200 {"daily_minutes_spent":16666713,...}
POST /quests/progress {"duration_seconds":100000000000000} -> 500 {"error":"internal_error"}
POST /quests/progress {"duration_seconds":60}              -> 500 {"error":"internal_error"}   <-- day bricked
redis GET daily:accumulated:<uid>:2026-09-22 -> 100001000002859
psql  SELECT minutes_spent, is_target_met    -> 16666713 | t
```

`GET /quests/daily` keeps answering 200 with `accumulated_seconds: 100001000002859`, so the client
sees a plausible screen over a write path that can no longer record anything. The user cannot
recover for 48 hours; there is no endpoint that resets the counter. It is self-inflicted rather than
cross-tenant, but it is reachable by any authenticated client (including a buggy one sending
milliseconds, or a `Number` that overflowed in JS), it permanently corrupts that day's
`is_target_met`, and it takes the slice's only write path offline for the user.

## Expected output
A per-call bound and a per-day bound, enforced before the INCRBY:
- reject `duration_seconds` above a plausible per-call maximum (a task is 10 minutes; a generous cap
  such as 3600s is a full hour of continuous study in one report) with `400 {"error":"invalid_request"}`;
- clamp or reject once the day's total would exceed a plausible maximum (86400s is a hard ceiling; the
  §6.2 target is 1800s), so `minutes_spent` can never approach the `INT` range;
- on the write path, a failure after the INCRBY does not leave an unusable counter — either the cap
  makes the failure unreachable, or the increment is compensated (`DECRBY`) before returning 500.

Add table-driven handler tests for the rejected magnitudes and a service test that a post-INCRBY
`Upsert` failure (the already-present, currently unused `fakeProgressRepo.err`) leaves the counter
usable.

## Evidence
- Plan: `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md` (*Notes and open questions*, "trusts the client's `duration_seconds`" — the trust
  is accepted; the overflow and the 48h brick are not covered there).
- `backend/internal/quests/handler.go:39` — `binding:"required,gt=0"`, no maximum.
- `backend/internal/quests/service.go:59-61,85` — `seconds <= 0` guard only; `int(total/60)` handed to the upsert.
- `backend/internal/store/migrations/0001_init.up.sql:48` — `minutes_spent INT`.
- Runtime proof above, branch `harness/2026-09-22-high-quests-daily-quest-suite-and-progress-recording` at `ee99b1a`.

## Evaluation

**Verdict: select, `high` (blocker).** Confirmed by the reviewer against real services (review, *Runtime proof
re-run*: `1e14` → 500, then an ordinary `60` → 500). Not re-litigated. The *Why* is real: the slice's only write path
goes offline for the user for the 48h TTL, `is_target_met` for that day is corrupted, and a client bug (milliseconds,
an overflowed JS `Number`) is enough to trigger it.

**Root cause:** `handler.go:39` binds `duration_seconds` as `required,gt=0` with no maximum; `service.go:59-61` guards
`seconds <= 0` only; `service.go:85` hands `int(total/60)` to an upsert into `minutes_spent INT`
(`0001_init.up.sql:48`). Past ~1.29e11 seconds the upsert fails with integer-out-of-range, the INCRBY has already
committed and is never compensated, and every later call re-derives the same overflowing value.

**Decision — cap per call, reject at the daily ceiling, no DECRBY:**
- `MaxDurationSeconds = 3600` per call (a task is 10 minutes; §6.2's whole day is 30; one hour is a learner who
  left a task open, not a plausible single report). Above it → `ErrInvalidDuration` → `400 {"error":"invalid_request"}`.
- `MaxDailySeconds = 86400`: before the INCRBY the service reads the counter (`Counter.Total`, a read) and rejects
  with the same 400 when `total + seconds` would exceed it. **Reject rather than clamp** because clamping would make
  `daily_seconds_spent` disagree with what the client sent, needs a compensating `SET`/`DECRBY` write on the rejection
  path (the very thing blocker 1 removes), and a human cannot legitimately reach 24h of study in one local day — the
  ceiling is an abuse guard, not a user state. The pre-check is racy under concurrency, but the bound it gives is what
  matters: the counter can exceed the ceiling by at most `MaxDurationSeconds` per request in flight, so
  `minutes_spent` overflowing `INT` would need ~3.6e7 max-size requests all in flight at once. Unreachable, so the
  post-INCRBY failure the idea worried about is unreachable too and no compensation path is needed (compensation would
  also interact with the once-only hook, which is the separate `ontargetmet-is-lost-forever…` bug, not folded in).
- The §6.2 request/response shapes are untouched; the only wire change is that two more inputs answer the existing 400.

**Dependencies:** none. **Plan:** shared amending plan hung off blocker 1 (`plan:` below points at it); both blockers
clear when it is `done`.
