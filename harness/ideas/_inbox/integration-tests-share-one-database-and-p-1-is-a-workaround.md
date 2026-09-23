---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: medium
rejected_reason: "Internal tidiness: -p 1 is documented in ci.yml, the Makefile and CODEMAP, the suite passes, and per-package schemas are a refactor of a working test harness with no user or developer-visible failure today."
---
# integration tests share one database and -p 1 is a workaround not isolation

## Why
CI's `backend-integration` job runs every package's integration tests against **one** Postgres service
container, and `internal/store`'s own tests destructively `DROP` and recreate the entire schema around
each of their cases. As soon as a second package (`auth`) grew a `TestIntegration*`, that collision
became real: the branch hit it twice in CI, first as a `pg_type_typname_nsp_index` duplicate-key race
inside `Migrate`, then as `first run applied [], want [0001_init]` when `store`'s `reset()` ran
concurrently with `auth`'s tests.

`-p 1` fixed it by making `go test` run one package's binary at a time, and that is a correct and
proportionate fix for today. But it is a scheduling flag, not isolation: the destructive `reset()`
still targets a database that every other package is also using, and nothing in the code prevents the
collision — only the command line does. That leaves several ways back into the same failure:

- a `t.Parallel()` added inside any package, which `-p 1` does not constrain;
- a developer running `go test ./internal/auth/...` and `go test ./internal/store/...` in two shells,
  or an IDE doing it for them — `make test-integration` carries the flag, a bare `go test` does not;
- a future CI matrix or a second job sharing the same service container;
- anyone invoking `go test ./... -run Integration` from memory, which is what the CODEMAP's own
  "reproduce from `backend/`" instructions used to say.

It also serialises CI for a suite that will only grow: seven more slices each add `TestIntegration*`,
and each one now costs its own connection setup and migration in sequence.

## Expected output
Integration tests are isolated by construction, so `-p 1` becomes a performance choice rather than a
correctness requirement:

- give each package its own Postgres schema — create `test_<package>` and set `search_path` on the
  test connection — so `reset()` drops only that package's tables, or
- give each package its own database (`CREATE DATABASE` per test binary, dropped at the end), or
- at minimum, replace `internal/store`'s schema-wide `reset()` with truncation of the tables that test
  actually touches, so it stops being destructive to anything but itself.

Keep `-p 1` if it is still wanted for determinism, but the suite should stay green without it. The
check: remove `-p 1` locally, run `go test ./... -run Integration -count=1` repeatedly, and see it
pass.

## Evidence
- Plan: `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md` (third deviation — `-p 1` added outside the plan's file list).
- `.github/workflows/ci.yml` — `go test ./... -count=1 -v -run Integration -p 1` in `backend-integration`.
- `backend/Makefile` — `test-integration` carries the same flag; plain `make test` and a bare `go test` do not.
- `backend/internal/store/integration_test.go` — `reset(t, pg)` drops and recreates the schema around each test.
- CI run 35740944612 (`pg_type_typname_nsp_index` duplicate key) and 35742036352
  (`first run applied [], want [0001_init]`) on branch
  `harness/2026-09-22-high-auth-google-oauth-code-exchange-and-jwt-sessions`.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Reject.** The workaround is correct, documented in three places, and the failure it prevents has not recurred. Isolation-by-construction would be nice; it is not a bug.
