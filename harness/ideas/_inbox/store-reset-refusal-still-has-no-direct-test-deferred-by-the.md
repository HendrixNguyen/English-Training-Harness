---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# store reset refusal still has no direct test (deferred by the integration-gate plan until integration_test.go is free)

## Why
The integration-gate idea's Expected output had four parts; the plan delivered three and explicitly deferred the fourth — a direct test that `reset` refuses to drop tables when `TEST_DATABASE_URL` is unset (`integration_test.go:46-48`) — because `reset(t *testing.T, …)` calls `t.Fatal` and needs a `testing.TB`-shaped seam, a signature change in `integration_test.go`, which unmerged branches hold. Nothing else records the deferral, so the second line of defence against a destructive `go test` stays unproven once those branches merge.

## Expected output
After the RLS / pet-streak-shield branches that edit `backend/internal/store/integration_test.go` have merged: `reset` takes a `testing.TB` (or a small `fataler` interface), and `integration_gate_test.go` gains `TestResetRefusesWithoutTestDatabaseURL`, which passes a recording fake, asserts it was failed with the refusal message and that no SQL ran. Mutation: deleting the guard fails the test.

## Evidence
- Plan `harness/plans/2026-09-27-integration-gate-tests-only-prove-the-skip-and-would-pass-if.md` ("Deferred (recorded, not done)"); idea `harness/ideas/_inbox/integration-gate-tests-only-prove-the-skip-and-would-pass-if.md` Expected output, last bullet.
- `backend/internal/store/integration_test.go:42-48` (`reset` guard).
