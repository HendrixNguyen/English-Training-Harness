---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md
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
