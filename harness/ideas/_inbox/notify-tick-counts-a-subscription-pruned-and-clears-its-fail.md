---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# notify Tick counts a subscription pruned and clears its failure counter even when DeleteSubscription fails

## Why
When a subscription reaches `MaxConsecutiveFailures`, `Tick` increments `stats.Pruned`, calls `DeleteSubscription` and then `ClearFailures` unconditionally. If the delete fails (a Postgres blip), the row survives, the stats say it was pruned, and the failure counter is reset — the dead endpoint gets three more free days of failed sends before pruning is retried. The 410/403 branches likewise count `Pruned` before knowing the delete worked and leave `push:fail:{id}` orphaned until its TTL. The fakes cannot inject a `DeleteSubscription` error, and `fakeQueue.failErr` fails `RecordFailure` and `ClearFailures` together, so none of this is testable today.

## Expected output
- `stats.Pruned` counts only subscriptions whose delete succeeded; a failed delete is `Failed` and keeps its failure counter.
- Every prune path (410, 403, three failures) clears `push:fail:{id}` only after a successful delete.
- `fakeRepo` gains a `DeleteSubscription` error hook and `fakeQueue` separate hooks for `RecordFailure` / `ClearFailures`; tests pin the three cases.

## Evidence
- Plan `harness/plans/2026-09-27-no-per-user-subscription-cap-and-no-length-bound-on-endpoint.md`, branch `harness/2026-09-27-medium-no-per-user-subscription-cap-and-no-length-bound-on-endpoint`.
- `backend/internal/notify/service.go:171-192` (on that branch): `stats.Pruned++` precedes `DeleteSubscription`; `ClearFailures` runs regardless of its result.
- `backend/internal/notify/fakes_test.go:75-83` (no delete error hook), `:129,138` (one `failErr` for both queue calls).
- Found by the test-gap pass of the 2026-09-27 daily review (orchestrator-verified by reading the branch source); not a blocker — failure-path accounting only.
