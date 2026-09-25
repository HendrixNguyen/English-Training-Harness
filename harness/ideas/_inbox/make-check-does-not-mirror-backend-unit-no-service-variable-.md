---
type: bug
status: planned
source: reviewer
run: _inbox
priority: low
plan: harness/plans/2026-09-25-the-documented-set-a-env-export-also-exports-test-database-u.md
---
# make check does not mirror backend-unit: no service-variable guard and fmt-check passes on a parse error

## Why
`make check` is documented (Makefile comment, CODEMAP CI section) as "what CI's backend-unit job runs, so a
branch can be checked before it is pushed", and executors now use it as their Definition-of-done command.
It diverges from the job in two ways that make its result mean something different:

1. **No service-variable guard.** `backend-unit` first fails if `DATABASE_URL`, `REDIS_URL`,
   `TEST_DATABASE_URL` or `TEST_REDIS_URL` is set. `make check` has no such step. A developer who followed
   the Makefile's own `test-integration` instructions (`export TEST_DATABASE_URL=… TEST_REDIS_URL=…`) and then
   runs `make check` gets the destructive integration tests too, **without `-p 1`** — the parallel
   run CODEMAP and the Makefile both say races on the shared database — so `make check` goes flaky-red for a
   reason CI can never reproduce, or proves the suite green while it silently depended on live services.
2. **`fmt-check` passes on a parse error.** `unformatted=$$(gofmt -l .)` under `/bin/sh` without `-e`
   discards gofmt's exit 2; stdout is empty, so the target prints the parse error and exits 0. The CI step
   fails correctly (Actions runs `bash -e`). `make check` as a whole is still saved by `go vet`, but
   `make fmt-check` on its own reports green on a file gofmt could not read.

Low: nothing reaches `main` wrongly (CI is the gate); this is the local mirror being less trustworthy than
its comment says.

## Expected output
- `make check` refuses to run (or unsets them for the test step, as the job's contract requires) when any
  of the four service variables is exported, with a message naming the variable.
- `fmt-check` fails when `gofmt -l` exits non-zero (e.g. `unformatted=$$(gofmt -l .) || exit 1; …`).
- Optionally `make check` runs `go build ./...` first, as the job does.

## Evidence
- Plan under review: `harness/plans/2026-09-24-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md` (Task 3 Step 3).
- `backend/Makefile:12-21` on branch `harness/2026-09-24-medium-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a` (`76bb6ca`);
  compare `.github/workflows/ci.yml:31-38` (the guard) and `backend/Makefile:38-44` (`-p 1` rationale).
- Reviewer probe, 2026-09-24: `printf 'package probe\n\nfunc F( {}\n' > internal/zz_syntax.go` →
  CI gate script under `bash -eo pipefail`: exit 2; `make fmt-check`: prints `expected ')', found '{'`, **exit 0**. Probe removed.

## Evaluation
_Evaluator, 2026-09-25 — daily decide (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low, planned today, folded into `the-documented-set-a-env-export-also-exports-test-database-u.md`.** Confirmed on `main`: `backend/Makefile` `check` has no service-variable guard (the CI job has one at `.github/workflows/ci.yml` "Assert no service variables are set"), and `fmt-check` assigns `$$(gofmt -l .)` without checking the exit status, so a parse error prints and exits 0. Both are Makefile lines next to the ones the head plan rewrites, and the guard is the other half of the same hazard (a `TEST_*` export reaching a parallel `go test ./...`).
