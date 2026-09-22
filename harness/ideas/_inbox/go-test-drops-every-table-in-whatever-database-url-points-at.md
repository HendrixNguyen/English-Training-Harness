---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md
plan: harness/plans/2026-09-22-go-test-drops-every-table-in-whatever-database-url-points-at.md
---
# go test drops every table in whatever DATABASE_URL points at

## Why
`backend/internal/store/integration_test.go` gates its live-database tests only on the presence of
`DATABASE_URL` — the exact variable name spec §8 uses for the Railway production database. The first
thing each of those tests does is `reset(t, pg)` (integration_test.go:27-40), which executes
`0001_init.down.sql` (`DROP TABLE users; …`) and then `DROP TABLE IF EXISTS schema_migrations`
against that connection. There is no build tag, no `-short` guard, no separate `TEST_DATABASE_URL`,
and no "is this a throwaway database?" check.

That makes the project's own documented workflow destructive. `harness/CODEMAP.md` tells the reader
to `make up`, export `DATABASE_URL`/`REDIS_URL`, then `make test-integration`. Those exports stay in
the shell. The very next `make test` — described in the same CODEMAP sentence as needing "no live
services" — silently wipes the schema. Point the same shell at a real database (to inspect it, to run
a one-off query, to reproduce a bug) and `go test ./...` destroys it.

`internal/health` is unaffected; this is purely the store integration file, but it runs as part of
the default `./...` package set.

## Expected output
The destructive tests cannot run against a database the developer did not explicitly nominate as
disposable. Any of these is acceptable, cheapest first:
- Gate on a dedicated variable (`TEST_DATABASE_URL`), so `DATABASE_URL` alone never triggers a drop.
- Or put the file behind `//go:build integration` so `go test ./...` cannot compile it in.
- Plus a belt-and-braces refusal in `reset()` when the target database already contains rows in
  `users`, or when the host is not localhost.

`make test` must stay non-destructive no matter what is exported. The CODEMAP sentence claiming
`make test` "needs no live services" must then be true unconditionally.

## Evidence
- Plan: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
  (Task 9 Step 2 — the file is the plan's verbatim text, so this is a plan defect, not an execution one).
- `backend/internal/store/integration_test.go:11-23` (`requirePostgres`, skips only on empty `DATABASE_URL`),
  `:27-40` (`reset`, the drops), `:44`, `:80` (called before any other assertion).
- `harness/CODEMAP.md` store bullet — instructs exporting `DATABASE_URL` and calls `make test`
  "no live services needed" in the same sentence.
- Reproduced by the reviewer without a live database (the tests run instead of skipping; only the
  dial fails):
```
$ DATABASE_URL='postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2' \
    env -u REDIS_URL go test ./internal/store/ -count=1 -run TestIntegration -v
=== RUN   TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent
    integration_test.go:44: down migration: failed to connect to `user=u database=db`: …
--- FAIL: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (0.00s)
```
  With a reachable database that `down migration` succeeds and the schema is gone.

## Evaluation

**Verdict: select, `priority: high`.** This is a merge blocker on
`harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md` and it is
confirmed real — the project owner reproduced it independently. It is not re-litigated here.

**Is the *Why* real?** Yes, and it is worse than a test-hygiene complaint. `DATABASE_URL` and
`REDIS_URL` are the *production* variable names: spec §8 injects them as Railway plugin variables,
`internal/config` requires exactly those two, and `harness/CODEMAP.md` tells a reader to export them.
`integration_test.go:13-16` gates only on `DATABASE_URL` being non-empty, and the first statement of
both Postgres tests is `reset(t, pg)` (`:44`, `:80`), which executes `0001_init.down.sql` — six
`DROP TABLE` — and then `DROP TABLE IF EXISTS schema_migrations`. So the destructive action and the
production credential share one trigger. Any shell with a real `DATABASE_URL` exported turns a plain
`go test ./...` into a schema wipe, with no confirmation and no build tag in the way. The review
reproduced the gating without a live database: the tests *ran* instead of skipping and only failed
because the dial failed.

**Achievable in one plan?** Yes — a single file's gating plus a Makefile comment and one CODEMAP
sentence. Well under a day.

**Dependencies:** none. Everything it touches already exists on the branch under review.

**Root cause (systematic-debugging, read-only).** Not an execution defect. The plan's Task 9 Step 2
specifies `integration_test.go` verbatim, and the executor transcribed it character for character
(the reviewer diffed it). The defect is in the plan's own text: it chose "skip when the connection
variable is unset" as the gate without noticing that the variable it picked is the one production
uses, and that the skip guards a `DROP`, not a read. There is exactly one gate and it is the wrong
predicate. Fix the predicate and everything downstream follows.

**Chosen fix (owner's call among the idea's three options).** Gate on dedicated, unambiguous
test-only variables `TEST_DATABASE_URL` / `TEST_REDIS_URL`, and additionally make `reset()` refuse to
run unless `TEST_DATABASE_URL` is set, so no future caller can reach the drops by another path. A
build tag was rejected: it moves the hazard rather than removing it (a developer who builds with
`-tags integration` and a production `DATABASE_URL` is back where they started), and it hides the
tests from `go vet ./...`. The "refuse if `users` has rows / host is not localhost" heuristic was
also rejected: a nominated `TEST_DATABASE_URL` is an explicit act of consent, and a heuristic that is
right 95% of the time invites people to trust it. Requiring an unambiguous variable is a complete
answer, so the belt-and-braces check would only add surface.

The CODEMAP sentence that calls `make test` "no live services needed" one clause after telling the
reader to export `DATABASE_URL` becomes true unconditionally once the gate changes, and must be
rewritten in the same commit — the reviewer deliberately left it alone for this fix.

**Priority rationale:** `high`. It blocks an unmerged branch, and its failure mode is silent,
irreversible data loss triggered by the most ordinary command in the repo.
