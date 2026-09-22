---
type: feature
status: planned
source: human
run: 2026-09-22-run-02
priority: high
plan: harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md
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

## Evaluation

**Verdict: select, priority high.** The empty *Why* is answered by the reviewer's inbox bug
`harness/ideas/_inbox/no-ci-runs-the-integration-suite-so-pgmigrator-is-never-veri.md`: the store slice
merged with `PgMigrator`, `Postgres.Ping`, `Redis.Ping` and `cmd/api` at 0 % executed coverage because
nothing ever runs the `TEST_*`-gated suite. Every later slice builds on `store`, so this is the outer
verification loop the harness itself lacks — `high` because it gates the MVP order rather than a user
feature. It is not a product-retention feature and should not be dressed as one.

**Achievable in one plan:** yes — one workflow file, two doc paragraphs, no app code. Depends only on
MVP slice 1 (merged, `8d97139`).

**Checked on 2026-09-22 in the main checkout (read-only):**
- Unit job commands pass as declared: `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u
  TEST_REDIS_URL sh -c 'go build ./... && go vet ./... && go test ./... -count=1'` → all `ok`, exit 0.
- Integration job commands pass against the compose stack on `POSTGRES_PORT=5433 REDIS_PORT=6380`
  (host 6379 is the owner's unrelated `scio3-redis-1`): `go test ./... -count=1 -v -run Integration`
  prints exactly three `--- PASS: TestIntegration…` lines and no `--- SKIP`. Stack brought down again.
- Action majors re-verified against each action's own README via `gh api repos/<owner>/<repo>/contents/README.md`
  (public reads work with the read-only work account): `actions/checkout` README pins **`@v7`** (v7.0.1,
  2026-07-20) — the idea's `@v6` is one major stale; `actions/setup-go@v7` (v7.0.0) and
  `actions/setup-python@v7` (v7.0.0) as cited. The plan uses `@v7` for all three.
- **golangci-lint decision: leave it out.** `golangci-lint` 2.13.2 with its default linter set reports
  one finding on the existing code — `internal/store/redis_test.go:20:15` `Error return value of
  r.Close is not checked (errcheck)`. The idea's condition (existing code passes with no code changes
  and no silenced rules) therefore does not hold; excluding `errcheck` for `_test.go` would be exactly
  the silencing it forbids, and fixing the test is app code outside a CI plan. Recommend a follow-up
  idea: change `defer r.Close()` to `t.Cleanup(func() { _ = r.Close() })`, then add
  `golangci/golangci-lint-action@v9` (latest v9.3.0) with a committed `.golangci.yml` (`version: "2"`).
- `actionlint` 1.7.12 and `golangci-lint` are Homebrew-installable on this machine (both installed
  during evaluation). `python3` here is 3.9.6 with PyYAML 6.0.3 importable, and `ruby -ryaml` also
  works, so the workflow can be parsed locally without pushing.

**Design choices recorded for the executor:**
- Three parallel jobs on `ubuntu-latest`, `permissions: contents: read`, `concurrency` keyed on
  `github.workflow` + `github.ref` with `cancel-in-progress`. Triggers: `push` to `main` and
  `pull_request` (Draft PRs from `harness/*` branches are covered by the latter).
- The unit job starts with an explicit guard step that fails if any of the four service variables is
  set — the "standing proof" must be asserted, not assumed.
- The integration job counts `func TestIntegration…` definitions in `*_test.go` and requires the same
  number of top-level `--- PASS: TestIntegration` lines and zero `--- SKIP` lines, so adding a
  fourth integration test later is picked up without editing the workflow, and a silent skip fails.
- The harness job uses `actions/setup-python@v7` `python-version: "3.x"`; the tooling has no
  dependencies and runs on 3.9 locally.
- Verification is local-only (static YAML parse, `actionlint`, and running each job's commands exactly
  as declared) because the executor's `gh` account cannot observe a CI run on this repo.
- Supersedes the inbox bug above, which is rejected with `rejected_reason` pointing here.
