---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# The hourly sweep loads every pet in a zone into memory and reparses tzdata per user

## Why
`Service.Sweep` is the only code in the backend that touches every user at once, and it is written
as if it touched a handful. Three costs compound at `00:00 UTC`, when the default `'UTC'` timezone
means *the whole user base* is one zone:

1. **Unbounded read.** `SweepCandidates` runs one query with no `LIMIT` and materialises every
   matching row into a single slice (`backend/internal/pet/repo.go:143-163`). The whole pet table
   is resident in the API process for the duration of the sweep.
2. **Non-sargable predicate.** `WHERE COALESCE(u.timezone, 'UTC') = ANY($1)`
   (`backend/internal/pet/repo.go:62-65`) cannot use an index on `users.timezone` even if one
   existed — and none does (`0001_init.up.sql` indexes nothing but the primary/unique keys). Both
   `timezonesSQL` and `candidatesSQL` are full scans of `users ⋈ pet_states`, every hour, whether
   or not any zone is at midnight.
3. **tzdata reparsed per user.** `quests.Location(c.Timezone)` is called *inside the candidate
   loop* (`backend/internal/pet/service.go:150`), and Go's `time.LoadLocation` is **not cached** —
   it opens and parses the zoneinfo file on every call. Measured on this machine (Go 1.25):

   ```
   50000 LoadLocation calls: 609.398ms (12.187µs each)
   same pointer across two calls (i.e. cached)? false
   ```

   At 100 000 users that is ~1.2 s of pure file parsing plus 100 000 `*time.Location`
   allocations, per sweep, for a value that takes one map lookup to reuse.
4. **Serial I/O per user.** One Redis `GET` (`study.Total`) and one `UPDATE` (`repo.Save`) per
   candidate, sequentially. The plan's *Notes* acknowledge the Redis half ("`MGET` when it is
   not"); the unbounded row load, the missing index and the tzdata cost are not recorded anywhere.

None of this is wrong today at MVP scale. It is filed because the sweep is the one background job
in the system, it is the piece a later slice (`notify`'s `queue:webpush:delay` cron) will copy, and
because the fixes are small and local while the code is 45 lines long.

## Expected output
The sweep's cost is bounded by the number of users actually at local midnight, not by the table:

- `Location` results are resolved once per sweep into a `map[string]*time.Location` (built from the
  `Timezones()` list) and looked up in the candidate loop, so `time.LoadLocation` is called once
  per zone instead of once per user;
- `SweepCandidates` takes a keyset/`LIMIT … OFFSET`-free page (e.g. `WHERE p.user_id > $2 ORDER BY
  p.user_id LIMIT $3`) and `Sweep` drains it page by page, so memory is O(page) not O(users);
- the counter reads for one page go out as a single `MGET` through a widened `StudyCounter`
  (`Totals(ctx, []userID, localDate)`), and the misses are written with one statement
  (`UPDATE … WHERE user_id = ANY($1) AND updated_at < $2`), which also fixes the read-then-write
  race filed separately;
- a migration adds `CREATE INDEX ON users (timezone)` and the two sweep queries drop the
  `COALESCE` from the predicate (keep it in the projection) so the index is usable; or the sweep
  selects on `pet_states.updated_at` and joins `users` afterwards;
- CODEMAP's pet bullet records what the sweep costs per hour.

## Evidence
- Plan: `harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md`
  (*Notes* — "`Sweep` reads the counter per user (N Redis `GET`s per midnight zone). Fine at MVP
  scale; `MGET` when it is not." — the other three costs are unrecorded).
- `backend/internal/pet/repo.go:58-65` — `timezonesSQL` / `candidatesSQL`, both full scans.
- `backend/internal/pet/repo.go:143-163` — `SweepCandidates`, no `LIMIT`, one slice.
- `backend/internal/pet/service.go:149-170` — the per-candidate `Location`, `Total` and `Save`.
- `backend/internal/quests/day.go:34-43` — `Location` calls `time.LoadLocation` with no memoisation.
- `backend/internal/store/migrations/0001_init.up.sql` — no index on `users.timezone`.
- Reviewer benchmark, 2026-09-23, Go 1.25 (output quoted above).
- Related: `harness/ideas/_inbox/no-index-supports-the-quests-lookups-on-roadmaps-and-exercis.md`
  (the same missing-index theme for quests).
