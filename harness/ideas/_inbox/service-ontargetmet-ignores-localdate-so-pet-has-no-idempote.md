---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Service.OnTargetMet ignores localDate so pet has no idempotency of its own

## Why
The pet slice's central invariant — §8's `+20 / streak+1` runs **exactly once per user per local
day** — is enforced entirely outside the pet package. `Service.OnTargetMet` takes the local date
and throws it away:

```go
// backend/internal/pet/service.go:49-58
// localDate is informational — the row is not keyed by day.
func (s *Service) OnTargetMet(ctx context.Context, userID, localDate string) error {
        st, err := s.Ensure(ctx, userID)
        ...
        return s.repo.Save(ctx, userID, ApplyTargetMet(st, s.now()))
}
```

Every call bumps. The once-per-day property comes only from quests inferring a rising edge on a
Redis counter (`newlyMet := total >= 1800 && total-seconds < 1800`,
`backend/internal/quests/service.go:110-111`). That inference is correct while the counter is
intact, but the counter is volatile state with a 48 h TTL (`store.DailyAccumulatedTTL`) in a Redis
that spec §9 provisions as a Railway plugin with no stated persistence. If
`daily:accumulated:{user}:{date}` is lost mid-day — eviction under `maxmemory`, a Redis restart
without AOF/RDB, a `FLUSHDB`, a failover to an empty replica — the counter restarts at 0 and the
*next* 1800 s of study crosses the edge a second time. The plant then gains `+40` and **two streak
days for one calendar day**, which is precisely the double-bump the plan's "§6.2-vs-§8 split,
decided" section exists to prevent. The streak is what drives `StageFor`, so the user also reaches
`fruitful` early and permanently.

The same shape means a future second caller of `Pet.OnTargetMet` (an onboarding backfill, an admin
tool, a replayed webhook) silently double-bumps with no defence in the package that owns the
invariant.

The fix is nearly free because the state already carries the evidence: `ApplyTargetMet` stamps
`LastPracticedAt`, and the caller already passes `localDate`.

## Expected output
`pet` enforces its own once-per-local-day rule, so the invariant survives a lost Redis counter and
any second caller:

- `Service.OnTargetMet` resolves the user's timezone (it already does this in `Revive`) and returns
  early, without saving, when `st.LastPracticedAt` is non-nil and its local date equals
  `localDate` — the parameter stops being "informational";
- the skip is not an error: quests logs hook errors, and a no-op must not look like a failure;
- `service_test.go` gains a test that calls `OnTargetMet(ctx, "u1", "2026-09-22")` twice and
  asserts health went `80 → 100` once, streak `4 → 5` once, and `repo.saved == 1`;
- a second test calls it for `"2026-09-23"` after `"2026-09-22"` and asserts it *does* bump, so the
  guard cannot be satisfied by simply never bumping;
- CODEMAP's "applied exactly once per local day" claim becomes true of the pet package rather than
  of quests' counter.

## Evidence
- Plan: `harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md`
  (**The §6.2-vs-§8 split, decided** — "Doing both would double-bump"; *Notes* — "Success once,
  from the hook").
- `backend/internal/pet/service.go:49-58` — `localDate` unused, unconditional `Save`.
- `backend/internal/pet/engine.go:60-68` — `ApplyTargetMet` already writes `LastPracticedAt`.
- `backend/internal/quests/service.go:110-111` — the counter-edge inference that is the only guard.
- `backend/internal/store/keys.go:14` — `DailyAccumulatedTTL = 48 * time.Hour`, i.e. the guard
  lives in expiring, non-durable state.
- Backend spec §8 Success Logic; §6.2 `pet_health` / `streak_count`.
- Related: `harness/ideas/_inbox/ontargetmet-is-lost-forever-if-a-write-after-the-incrby-fail.md`
  — the opposite failure of the same inference (the hook never firing).
