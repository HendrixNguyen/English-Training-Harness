---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Concurrent subscription saves for one user overshoot the 10-row cap

## Why
`PgRepo.SaveSubscription` inserts and then runs `trimSubscriptionsSQL` in one transaction, but nothing serialises two transactions for the same user. Under READ COMMITTED each trim's `NOT IN (… ORDER BY created_at DESC LIMIT 10)` sees only its own uncommitted insert plus committed rows, so N parallel saves each delete the same oldest rows and leave up to 10 + N − 1. The next sequential save trims back to 10, so this is bounded and self-healing — but the cap's stated purpose is bounding `Tick`'s daily fan-out per user, and an authenticated client firing parallel `POST /settings/notifications` can exceed it until the next single save.

## Expected output
The cap holds under concurrency: serialise saves per user inside the transaction — e.g. `SELECT 1 FROM users WHERE id = $1 FOR UPDATE` (or `pg_advisory_xact_lock(hashtext($1))`) before the insert — and an integration test that fires ~20 concurrent `SaveSubscription` calls for one user after 10 sequential ones and asserts exactly 10 rows.

## Evidence
- Plan: `harness/plans/2026-09-27-no-per-user-subscription-cap-and-no-length-bound-on-endpoint.md` (Task 2; Review Focus 2 "exactly 10 rows").
- Branch `harness/2026-09-27-medium-no-per-user-subscription-cap-and-no-length-bound-on-endpoint` @ `273be93`: `backend/internal/notify/repo.go` `SaveSubscription` (`pgx.BeginFunc` insert + trim), `trimSubscriptionsSQL`.
- Reviewer probe 2026-09-27 (temporary `_test.go`, deleted after) against scratch Postgres: 10 sequential + 20 concurrent `SaveSubscription` for one user → `13`, `15`, `11` rows on three runs; one more sequential save → `10`.
