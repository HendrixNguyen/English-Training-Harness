---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# CODEMAP CI section overstates the unit job and prescribes a racy local repro

## Why
`harness/CODEMAP.md`'s new `## CI` section is the first thing every role reads before touching CI.
Three problems with it as written.

**It credits `backend-unit` with a property that job does not test.** The text calls the guard step
"the standing proof that the default suite needs no services and never drops tables". The guard
asserts that `DATABASE_URL` / `REDIS_URL` / `TEST_DATABASE_URL` / `TEST_REDIS_URL` are unset — nothing
more. "Never drops tables" is a property of `requirePostgres` / `reset` in
`backend/internal/store/integration_test.go`, whose only automated proof is
`integration_gate_test.go`, and the open bug
`harness/ideas/_inbox/integration-gate-tests-only-prove-the-skip-and-would-pass-if.md` shows that
proof is one-directional. A reader who trusts CODEMAP will believe the destructive-test hazard is now
covered by CI when it is not.

**Its local-reproduction recipe races.** CODEMAP says to reproduce the integration job with
"`make up` with `POSTGRES_PORT`/`REDIS_PORT` overrides and the matching `TEST_*` URLs". `backend/Makefile:15-16`
defines `up` as `docker compose up -d` with **no** `--wait`, so `go test` can start before Postgres is
healthy and die in `NewPostgres` with a connection error. The plan's own verified procedure — and the
reviewer's re-run — used `docker compose up -d --wait`, not `make up`. The documented path is not the
path anyone actually exercised.

**It is one ~210-word sentence-chain.** `harness/CODEMAP.md`'s own contract is "One paragraph per
package/module. Read this before exploring code." — the file is meant to be scannable under token
discipline. Three short bullets, one per job, carry the same information at a fraction of the reading
cost.

(Separately, the plan's *Verification* step 5 expects `grep -n 'outer verification loop'` to hit both
docs, but the CODEMAP text the plan prescribes verbatim contains no such phrase. The executor logged
this and followed the prescribed text rather than inventing wording — the right call, and the idea
only asks `AGENTS.md` to carry that phrase. No fix needed; noted so the next reader does not
re-discover it.)

## Expected output
The `## CI` section in `harness/CODEMAP.md`:

- describes `backend-unit`'s guard as what it is — "fails if any service variable is exported, so the
  default suite is proved to need no services" — and drops the "never drops tables" claim, or points
  at `integration_gate_test.go` as where that property actually lives;
- gives the reproduction command that was actually run:
  `POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose up -d --wait` from `backend/`, then the `TEST_*`
  URLs on those ports — or `make up` gains `--wait` so the shorthand becomes true;
- is restructured as a short lead-in plus one bullet per job.

## Evidence
- Plan under review: `harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md`
  (Task 2 Step 1 prescribes the paragraph verbatim).
- `harness/CODEMAP.md`, `## CI (.github/workflows/ci.yml)` — the paragraph.
- `backend/Makefile:15-16` — `up: docker compose up -d`, no `--wait`.
- `harness/ideas/_inbox/integration-gate-tests-only-prove-the-skip-and-would-pass-if.md` — the
  one-directional gate proof.
