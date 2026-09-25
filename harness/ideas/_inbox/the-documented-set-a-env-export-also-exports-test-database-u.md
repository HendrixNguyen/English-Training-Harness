---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-25-the-documented-set-a-env-export-also-exports-test-database-u.md
---
# The documented set -a env export also exports TEST_DATABASE_URL so a later make test drops the dev database

## Why
This branch adds the documented dev loop `set -a; . ./.env; set +a; make run` in both
`backend/.env.example` and the `backend/Makefile` `run` comment. `.env` is copied from
`.env.example`, and that file has `TEST_DATABASE_URL` / `TEST_REDIS_URL` **uncommented**, pointing
at the *same* dev database as `DATABASE_URL` (`english` on localhost:5432).

`set -a` exports every assignment, so after following the documented loop the developer's shell
holds `TEST_DATABASE_URL`. The next `make test` (`go test ./...`) or `go test ./...` in that shell
stops skipping the integration tests. `internal/store`'s integration test then **drops every
table** in the dev database the developer was just using. Because `make test` has no `-p 1`, the
packages also race each other on that one database, which is the exact hazard the
`test-integration` target's comment warns about.

Before this branch nothing told anyone to source `.env` into the shell. The TEST_* vars reached
the Go process only when the developer typed `export TEST_DATABASE_URL=…` on purpose, as the
Makefile instructs. The destructive gate was opt-in, and the new documentation makes it implicit.
This is dev-only data loss, which is why it is medium rather than a blocker.

## Expected output
- The documented export does not leak into the interactive shell and does not carry TEST_*. For
  example, run it in a subshell, `(set -a; . ./.env; set +a; exec go run ./cmd/api)`, as a `run`
  recipe. Alternatively, keep TEST_* commented out in `.env.example`, or point them at a separate
  `english_test` database.
- `.env.example`'s TEST_* comment and the Makefile's `test-integration` comment stay consistent
  with whichever option is chosen.

## Evidence
- Plan: `harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` (Task 4 Steps 1-2).
- `backend/.env.example` (new application section plus the existing TEST_* block) and
  `backend/Makefile` (`run` comment; `test:` = `go test ./...` without `-p 1`).
- `backend/internal/store/integration_test.go:19` (gate on `TEST_DATABASE_URL`) and `:66`
  (`DROP TABLE …`).
- The plan's own Verification commands prefix every `go test` with
  `env -u … -u TEST_DATABASE_URL -u TEST_REDIS_URL`, which works around this same hazard.

## Evaluation
_Evaluator, 2026-09-25 — daily decide (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — medium, planned today (head of the dev-loop group).** Confirmed on `main`: `backend/.env.example` ships `TEST_DATABASE_URL`/`TEST_REDIS_URL` uncommented and pointing at the same `english` database as `DATABASE_URL`, and both the file and the `Makefile` `run` comment tell the developer to `set -a; . ./.env; set +a` in their interactive shell; `internal/store/integration_test.go` is gated on `TEST_DATABASE_URL` and drops every table. Following the documented loop and then typing `make test` destroys the dev database, in parallel. Developer-workflow data loss ranks with user bugs (evaluator role, rule 3), and the owner is the developer who runs this loop. Root cause: the shutdown plan documented an export into the *interactive* shell where a per-recipe subshell was what it needed. Decision: `make run` sources `.env` inside its own recipe line (make runs each line in a fresh `/bin/sh`, so nothing leaks), `TEST_*` are commented out in `.env.example`, and the documented loop becomes plain `make run`. Two sibling Makefile/CI findings fold in: `make check` lacking the service-variable guard and `fmt-check` swallowing gofmt's exit 2 (`make-check-does-not-mirror-backend-unit-no-service-variable-.md`) and `-race` on `backend-integration` (`backend-integration-never-runs-race-though-its-pet-and-store.md`). Plan: `harness/plans/2026-09-25-the-documented-set-a-env-export-also-exports-test-database-u.md`.
