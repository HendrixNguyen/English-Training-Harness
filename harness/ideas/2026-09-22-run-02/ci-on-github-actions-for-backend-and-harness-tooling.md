---
type: feature
status: proposed
source: human
run: 2026-09-22-run-02
priority: high
---
# CI on GitHub Actions for backend and harness tooling

## Why
<!-- Business rationale. Tie to spec goals: retention (30 min/day), CEFR progression, gamified pet engine. -->

## Expected output
Visible behaviour: every push and every pull request to `main` gets a status check. A branch that builds only on the author's machine fails visibly, before a human is asked to merge it.

Technical — `.github/workflows/ci.yml`, triggered on `push` to `main` and on `pull_request`, with `permissions: contents: read` and concurrency cancelling superseded runs on the same ref. Three jobs, able to run in parallel:

1. **`backend-unit`** — `actions/checkout@v6`, `actions/setup-go@v7` with `go-version-file: backend/go.mod` and `cache-dependency-path: backend/go.sum`. Runs `go build ./...`, `go vet ./...`, and `go test ./... -count=1` from `backend/`, in a shell with **no** `DATABASE_URL`, `REDIS_URL`, `TEST_DATABASE_URL` or `TEST_REDIS_URL` set. This job is the standing proof that the default test run needs no services and is never destructive.
2. **`backend-integration`** — same setup plus `services:` for `postgres:16-alpine` (user/password/db `english`, `--health-cmd "pg_isready -U english"`) and `redis:7-alpine` (`--health-cmd "redis-cli ping"`), both with health interval/timeout/retries. Exports `TEST_DATABASE_URL` and `TEST_REDIS_URL` pointing at the mapped ports, then runs the integration suite. Must assert the tests actually **ran** rather than skipped — a silent skip here would recreate the exact hazard this job exists to close, so grep the verbose output for the expected `--- PASS` lines and fail if any `--- SKIP` appears.
3. **`harness-tooling`** — Python 3, no dependencies: `python3 -m unittest discover -s tools/harness/tests` and `python3 tools/harness/cli.py validate` (which exits 1 on malformed artifacts, so a bad frontmatter commit fails CI).

Linting is in scope only if it can be added without churn: if `golangci-lint` is included, it needs a committed `.golangci.yml` and the existing code must pass it with no changes to that code — otherwise leave it out and say so, rather than merging a red job or silencing rules.

Also: `harness/CODEMAP.md` gains a short CI paragraph, and `AGENTS.md` notes that CI is the outer verification loop and must stay green.

Depends on: MVP slice 1 (merged). Blocks nothing, but every later slice benefits.

## Evidence
<!-- Spec sections, research links, prior runs or reviews this idea rests on. -->
