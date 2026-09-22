---
idea: harness/ideas/2026-09-22-run-02/ci-on-github-actions-for-backend-and-harness-tooling.md
status: done
priority: high
merged: false
branch: harness/2026-09-22-high-ci-on-github-actions-for-backend-and-harness-tooling
worktree: .worktrees/ci-on-github-actions-for-backend-and-harness-tooling
---
# CI on GitHub Actions for backend and harness tooling — Plan

**Idea:** `harness/ideas/2026-09-22-run-02/ci-on-github-actions-for-backend-and-harness-tooling.md`
**Goal:** Add `.github/workflows/ci.yml` with three parallel jobs — `backend-unit` (build/vet/test with no service variables), `backend-integration` (the `TEST_*`-gated suite against Postgres and Redis service containers, failing if any test skips) and `harness-tooling` (the 30 Python tests plus `cli.py validate`) — and document CI as the harness's outer verification loop in `harness/CODEMAP.md` and `AGENTS.md`.

**Architecture:** One workflow file, no app code. Each job is a literal transcription of a command sequence that already passes locally (checked 2026-09-22, see the idea's `## Evaluation`), so the workflow adds enforcement, not behaviour. The unit job asserts its own precondition (no `DATABASE_URL` / `REDIS_URL` / `TEST_DATABASE_URL` / `TEST_REDIS_URL`) instead of assuming it. The integration job derives the expected number of `TestIntegration*` functions from the source, so a later slice that adds one is counted automatically and a silent `--- SKIP` fails the job. **`golangci-lint` is deliberately not included** — `golangci-lint` 2.13.2 with its default linters flags `internal/store/redis_test.go:20` (`errcheck` on `defer r.Close()`), so the idea's "passes with no code changes and no silenced rules" condition does not hold; leave it for a follow-up idea.

**Tech stack:** GitHub Actions on `ubuntu-latest`; `actions/checkout@v7`, `actions/setup-go@v7` (`go-version-file` + `cache-dependency-path`), `actions/setup-python@v7`; service containers `postgres:16-alpine` and `redis:7-alpine`. Action majors were re-verified against each action's own README on 2026-09-22 (`checkout` README pins `@v7`; the idea's `@v6` is stale).

**Branch:** new `harness/*` branch in `.worktrees/ci-on-github-actions-for-backend-and-harness-tooling`, per the execute skill. Run commands from the **worktree root** unless a step says `backend/`. `rg` is not installed — use `grep -n`. Never capture `$(ls …)`.

**You cannot watch a real CI run.** The `gh` account on this machine has no write access to this repo, so every check in this plan is local: static (YAML parse, `actionlint`) plus running each job's `run:` blocks exactly as the workflow declares them. Do not push to "see if it goes green" as a substitute for the steps below; push only when the execute skill says to.

**Host port 6379 is taken** by the owner's unrelated `scio3-redis-1` container, which must keep running. Every local integration step uses the documented override `POSTGRES_PORT=5433 REDIS_PORT=6380` and the matching `TEST_*` URLs on those ports. The compose project name derives from the directory name `backend`, so the worktree and the main checkout share container names `backend-postgres-1` / `backend-redis-1`; `docker ps` first, and bring the stack down when done.

## File structure

| Path | Change |
| --- | --- |
| `.github/workflows/ci.yml` | **new** — the workflow, three jobs |
| `harness/CODEMAP.md` | new `## CI` section (one paragraph) |
| `AGENTS.md` | one bullet under *Rules every role follows* |

Nothing under `backend/` changes. `backend/Makefile` targets `test` and `test-integration` stay the local entry points; the workflow spells the commands out rather than calling `make` so a reader of the run log sees exactly what ran.

---

## Tasks

### Task 1: The workflow file

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Confirm the local toolchain for static checks**

Run from the worktree root:

```sh
command -v actionlint || brew install actionlint
actionlint -version | head -1
python3 -c 'import yaml; print("pyyaml", yaml.__version__)' || ruby -ryaml -e 'puts "ruby yaml ok"'
```
Expected: an `actionlint` version (1.7.12 was installed on this machine on 2026-09-22), and either `pyyaml 6.0.3` (the system `/usr/bin/python3` 3.9.6 has it in the user site-packages) or `ruby yaml ok`. If **neither** YAML line prints, fall back to the structural greps in `## Verification` step 1b and say so in the execution summary.

- [ ] **Step 2: Write the workflow**

Create `.github/workflows/ci.yml` with exactly this content:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

permissions:
  contents: read

concurrency:
  group: ci-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  backend-unit:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: backend
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: backend/go.mod
          cache-dependency-path: backend/go.sum
      - name: Assert no service variables are set
        run: |
          for v in DATABASE_URL REDIS_URL TEST_DATABASE_URL TEST_REDIS_URL; do
            if [ -n "${!v:-}" ]; then
              echo "::error::$v is set; the unit job must run with no service variables"
              exit 1
            fi
          done
      - name: Build
        run: go build ./...
      - name: Vet
        run: go vet ./...
      - name: Test without services
        run: go test ./... -count=1

  backend-integration:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: backend
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: english
          POSTGRES_PASSWORD: english
          POSTGRES_DB: english
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U english"
          --health-interval 5s
          --health-timeout 5s
          --health-retries 10
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 5s
          --health-timeout 5s
          --health-retries 10
    env:
      TEST_DATABASE_URL: postgres://english:english@localhost:5432/english?sslmode=disable
      TEST_REDIS_URL: redis://localhost:6379/0
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: backend/go.mod
          cache-dependency-path: backend/go.sum
      - name: Integration tests must run, not skip
        run: |
          set -o pipefail
          go test ./... -count=1 -v -run Integration 2>&1 | tee integration.log
          want=$(grep -rho '^func TestIntegration[A-Za-z0-9_]*' --include='*_test.go' . | wc -l | tr -d ' ')
          got=$(grep -c '^--- PASS: TestIntegration' integration.log || true)
          if grep -q '^--- SKIP' integration.log; then
            echo "::error::an integration test skipped; TEST_DATABASE_URL / TEST_REDIS_URL did not reach the tests"
            exit 1
          fi
          if [ "$want" -eq 0 ] || [ "$got" -ne "$want" ]; then
            echo "::error::expected $want passing TestIntegration* tests, saw $got"
            exit 1
          fi
          echo "$got/$want integration tests ran and passed"

  harness-tooling:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-python@v7
        with:
          python-version: "3.x"
      - name: Harness unit tests
        run: python3 -m unittest discover -s tools/harness/tests
      - name: Validate harness artifacts
        run: python3 tools/harness/cli.py validate
```

Why each non-obvious line is there:
- `permissions: contents: read` — the workflow only reads the checkout; nothing writes back.
- `concurrency` keyed on workflow + ref with `cancel-in-progress` — a second push to the same branch cancels the superseded run.
- The four-variable guard uses bash indirect expansion `${!v:-}`; GitHub's default `run` shell on Linux is bash, and `actionlint` + `shellcheck` accept it (checked 2026-09-22).
- `services.*.ports` map to the runner VM, and the job runs on the VM (not in a container), so `localhost:5432` / `localhost:6379` are correct. Job-level `env` is exported to every step, which is how `os.Getenv("TEST_DATABASE_URL")` in `backend/internal/store/integration_test.go:16` sees it.
- `-run Integration` selects the three `TestIntegration*` functions today; the `want` count is computed from `*_test.go` so a fourth one is required to pass without editing this file. `grep -c … || true` keeps `set -o pipefail` from aborting on a zero count so the error message, not a bare non-zero exit, explains the failure. Subtests print indented `    --- PASS`, so `^--- PASS` counts only top-level tests.
- `--- SKIP` anywhere fails the job: a skip here would recreate the hazard this job exists to close.
- `harness-tooling` runs from the repo root because the Python tests and `cli.py` resolve paths relative to it, exactly as `AGENTS.md` documents.

- [ ] **Step 3: Static checks**

```sh
actionlint .github/workflows/ci.yml; echo "actionlint exit=$?"
python3 - <<'EOF'
import yaml
d = yaml.safe_load(open(".github/workflows/ci.yml"))
j = d["jobs"]
print(sorted(j))
u = j["backend-unit"]
assert "env" not in u and not any("env" in s for s in u["steps"]), "unit job must not set env"
i = j["backend-integration"]
assert set(i["env"]) == {"TEST_DATABASE_URL", "TEST_REDIS_URL"}, i["env"]
assert set(i["services"]) == {"postgres", "redis"}
assert d["permissions"] == {"contents": "read"}
assert d["concurrency"]["cancel-in-progress"] is True
print("yaml checks ok")
EOF
grep -c 'uses: actions/checkout@v7' .github/workflows/ci.yml
grep -c 'uses: actions/setup-go@v7' .github/workflows/ci.yml
grep -c 'uses: actions/setup-python@v7' .github/workflows/ci.yml
```
Expected: `actionlint exit=0` with no findings printed above it; `['backend-integration', 'backend-unit', 'harness-tooling']` then `yaml checks ok`; then `3`, `2`, `1`. If PyYAML is missing, run the ruby equivalent `ruby -ryaml -e 'd=YAML.load_file(".github/workflows/ci.yml"); p d["jobs"].keys.sort'` and rely on the three greps plus `## Verification` 1b for the rest.

- [ ] **Step 4: Run the `backend-unit` job's commands locally, exactly as declared**

```sh
cd backend
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL bash -e -c '
  for v in DATABASE_URL REDIS_URL TEST_DATABASE_URL TEST_REDIS_URL; do
    if [ -n "${!v:-}" ]; then echo "$v is set"; exit 1; fi
  done
  go build ./... && go vet ./... && go test ./... -count=1'
echo "unit exit=$?"
# Negative check: the guard must trip when a service variable leaks in.
TEST_REDIS_URL=x bash -e -c 'for v in DATABASE_URL REDIS_URL TEST_DATABASE_URL TEST_REDIS_URL; do if [ -n "${!v:-}" ]; then echo "guard trips on $v"; exit 1; fi; done'; echo "guard negative exit=$?"
cd ..
```
Expected: `ok` for `internal/config`, `internal/health`, `internal/store` (`cmd/api` reports `[no test files]`), `unit exit=0`; then `guard trips on TEST_REDIS_URL` and `guard negative exit=1`.

- [ ] **Step 5: Run the `backend-integration` job's step locally against the compose stack**

```sh
docker ps --format '{{.Names}} {{.Ports}}'          # scio3-redis-1 on 6379 must be left alone
cd backend
POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose up -d --wait
TEST_DATABASE_URL='postgres://english:english@localhost:5433/english?sslmode=disable' \
TEST_REDIS_URL='redis://localhost:6380/0' bash -e -c '
  set -o pipefail
  go test ./... -count=1 -v -run Integration 2>&1 | tee integration.log
  want=$(grep -rho "^func TestIntegration[A-Za-z0-9_]*" --include="*_test.go" . | wc -l | tr -d " ")
  got=$(grep -c "^--- PASS: TestIntegration" integration.log || true)
  if grep -q "^--- SKIP" integration.log; then echo "an integration test skipped"; exit 1; fi
  if [ "$want" -eq 0 ] || [ "$got" -ne "$want" ]; then echo "expected $want saw $got"; exit 1; fi
  echo "$got/$want integration tests ran and passed"'
echo "integration exit=$?"
# Negative check: without the TEST_* variables the same block must fail on the skip.
env -u TEST_DATABASE_URL -u TEST_REDIS_URL bash -e -c '
  set -o pipefail
  go test ./... -count=1 -v -run Integration 2>&1 | tee integration.log >/dev/null
  if grep -q "^--- SKIP" integration.log; then echo "negative: skip detected"; exit 1; fi'
echo "negative exit=$?"
rm -f integration.log
docker compose down
cd ..
```
Expected: `Container backend-postgres-1 Healthy` and `Container backend-redis-1 Healthy`; three lines `--- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent`, `--- PASS: TestIntegrationPetStatesRejectsASecondRowForTheSameUser`, `--- PASS: TestIntegrationRedisRoundTrip`; `3/3 integration tests ran and passed`; `integration exit=0`; then `negative: skip detected`, `negative exit=1`; `docker compose down` leaves only `scio3-redis-1` and `scio3-mongo-1` in `docker ps`. `integration.log` must not be left behind (it is untracked scratch; `git status --short` in `## Verification` catches it).

- [ ] **Step 6: Run the `harness-tooling` job's commands locally**

```sh
python3 -m unittest discover -s tools/harness/tests 2>&1 | tail -3
python3 tools/harness/cli.py validate; echo "validate exit=$?"
```
Expected: `Ran 30 tests`, `OK`; `validate exit=0`.

- [ ] **Step 7: Commit**

```sh
git add .github/workflows/ci.yml
git diff --cached --stat
git commit -m "ci: add GitHub Actions workflow for backend and harness tooling"
```
Expected: one file, `.github/workflows/ci.yml`, in the staged stat.

### Task 2: Document CI as the outer verification loop

**Files:**
- Modify: `harness/CODEMAP.md` (append a section after *Harness tooling*)
- Modify: `AGENTS.md:17-26` (the *Rules every role follows* list; the *Pushing* bullet is line 22)

- [ ] **Step 1: CODEMAP section**

Append to the end of `harness/CODEMAP.md`:

```markdown
## CI (`.github/workflows/ci.yml`)

Three parallel GitHub Actions jobs on every `pull_request` and every push to `main`: **`backend-unit`** (`go build`, `go vet`, `go test ./... -count=1` from `backend/`, with a guard step that fails if `DATABASE_URL`, `REDIS_URL`, `TEST_DATABASE_URL` or `TEST_REDIS_URL` is set — the standing proof that the default suite needs no services and never drops tables); **`backend-integration`** (`postgres:16-alpine` + `redis:7-alpine` service containers, `TEST_DATABASE_URL` / `TEST_REDIS_URL` exported, `go test ./... -count=1 -v -run Integration`; it counts `func TestIntegration*` in `*_test.go` and fails unless that many `--- PASS` lines appear and no `--- SKIP` does, so new integration tests are picked up without editing the workflow and a silent skip is a failure); **`harness-tooling`** (`python3 -m unittest discover -s tools/harness/tests` and `python3 tools/harness/cli.py validate`, so a malformed frontmatter commit fails CI). No linter yet: `golangci-lint` defaults flag one `errcheck` in `internal/store/redis_test.go`; add it only with that fixed and a committed `.golangci.yml`. Reproduce any job locally by running its `run:` blocks as written (integration: `make up` with `POSTGRES_PORT`/`REDIS_PORT` overrides and the matching `TEST_*` URLs).
```

- [ ] **Step 2: AGENTS.md rule**

In `AGENTS.md`, under `## Rules every role follows`, add this bullet directly after the *Pushing `harness/*` branches…* bullet:

```markdown
- CI (`.github/workflows/ci.yml`) is the outer verification loop and must stay green: `backend-unit`, `backend-integration` and `harness-tooling` run on every PR and push to `main`. A red check on a `harness/*` PR is a review blocker. Never make a job pass by skipping, loosening or deleting a check — fix the code or the artifact it flagged.
```

- [ ] **Step 3: Check the edits landed where intended**

```sh
grep -n '^## CI' harness/CODEMAP.md
grep -n 'outer verification loop' AGENTS.md harness/CODEMAP.md
grep -c 'backend-integration' harness/CODEMAP.md AGENTS.md
python3 tools/harness/cli.py validate; echo "validate exit=$?"
```
Expected: one `## CI` heading at the end of CODEMAP; `outer verification loop` in `AGENTS.md`; counts `1` and `1`; `validate exit=0`.

- [ ] **Step 4: Commit**

```sh
git add harness/CODEMAP.md AGENTS.md
git diff --cached --stat
git commit -m "docs: record CI as the harness's outer verification loop"
```
Expected: exactly two files in the staged stat.

## Verification

Run from the worktree root. Nothing here pushes or needs GitHub access.

```sh
# 1a. The workflow parses and has the declared shape.
actionlint .github/workflows/ci.yml; echo "actionlint exit=$?"
python3 -c '
import yaml
d = yaml.safe_load(open(".github/workflows/ci.yml")); j = d["jobs"]
assert sorted(j) == ["backend-integration", "backend-unit", "harness-tooling"], sorted(j)
assert "env" not in j["backend-unit"] and not any("env" in s for s in j["backend-unit"]["steps"])
assert set(j["backend-integration"]["env"]) == {"TEST_DATABASE_URL", "TEST_REDIS_URL"}
assert d["permissions"] == {"contents": "read"} and d["concurrency"]["cancel-in-progress"] is True
print("yaml checks ok")'
# 1b. Structural greps (also the fallback if no YAML parser is importable).
grep -c '^  [a-z]*-[a-z-]*:$' .github/workflows/ci.yml                # 3 hyphenated job keys at two-space indent
grep -n 'uses: actions/' .github/workflows/ci.yml                    # 6 lines, all @v7
grep -n 'TEST_DATABASE_URL\|TEST_REDIS_URL' .github/workflows/ci.yml # only inside backend-integration
grep -n 'SKIP' .github/workflows/ci.yml                              # the skip guard exists
```
Expected: `actionlint exit=0`, `yaml checks ok`; `3`; six `uses:` lines each ending `@v7`; the `TEST_*` lines fall between the `backend-integration:` and `harness-tooling:` job keys (compare line numbers against `grep -n '^  [a-z]*-[a-z-]*:$'`); at least one `SKIP` line.

```sh
# 2. Unit job, as CI runs it: no service variables anywhere.
cd backend
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL bash -e -c '
  for v in DATABASE_URL REDIS_URL TEST_DATABASE_URL TEST_REDIS_URL; do [ -z "${!v:-}" ] || { echo "$v is set"; exit 1; }; done
  go build ./... && go vet ./... && go test ./... -count=1'; echo "unit exit=$?"
```
Expected: three `ok` lines, `unit exit=0`.

```sh
# 3. Integration job, as CI runs it, against the local stack on the documented port overrides.
POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose up -d --wait
TEST_DATABASE_URL='postgres://english:english@localhost:5433/english?sslmode=disable' \
TEST_REDIS_URL='redis://localhost:6380/0' bash -e -c '
  set -o pipefail
  go test ./... -count=1 -v -run Integration 2>&1 | tee integration.log
  want=$(grep -rho "^func TestIntegration[A-Za-z0-9_]*" --include="*_test.go" . | wc -l | tr -d " ")
  got=$(grep -c "^--- PASS: TestIntegration" integration.log || true)
  grep -q "^--- SKIP" integration.log && { echo skipped; exit 1; }
  [ "$want" -gt 0 ] && [ "$got" -eq "$want" ] || { echo "expected $want saw $got"; exit 1; }
  echo "$got/$want integration tests ran and passed"'; echo "integration exit=$?"
rm -f integration.log; docker compose down; cd ..
docker ps --format '{{.Names}}'
```
Expected: `3/3 integration tests ran and passed`, `integration exit=0`; afterwards only `scio3-redis-1` and `scio3-mongo-1` remain.

```sh
# 4. Harness job, as CI runs it.
python3 -m unittest discover -s tools/harness/tests 2>&1 | tail -3; python3 tools/harness/cli.py validate; echo "validate exit=$?"
# 5. Docs and tree.
grep -n 'outer verification loop' AGENTS.md harness/CODEMAP.md
git status --short; git log --oneline -3
```
Expected: `Ran 30 tests` / `OK`, `validate exit=0`; one hit in each doc; clean tree; the two commits from Tasks 1 and 2 on top of `main`.

## Notes

- **Out of scope, on purpose:** `golangci-lint` (see *Architecture*); a `make ci` wrapper; caching beyond `setup-go`'s built-in module cache; any `frontend/` job (no frontend exists). Do not add them.
- The first real run happens when the execute skill pushes the branch and opens the Draft PR (the `pull_request` trigger). If the reviewer can see the check status through the GitHub UI, record it in the review; if not, the local steps above are the evidence, as they are for the executor.
- Supersedes `harness/ideas/_inbox/no-ci-runs-the-integration-suite-so-pgmigrator-is-never-veri.md` (rejected 2026-09-22 with `rejected_reason` pointing at this plan's idea). Its coverage numbers are the reviewer's baseline: after this lands, `PgMigrator.*`, `Postgres.Ping/Close/Migrator` and `Redis.Ping` are exercised on every PR.

## Execution summary

Built as specified: `.github/workflows/ci.yml` created verbatim from Task 1 Step 2 (three jobs: `backend-unit`, `backend-integration`, `harness-tooling`); `harness/CODEMAP.md` and `AGENTS.md` updated verbatim from Task 2 Steps 1-2. Two commits, one per task, exactly as the plan's staged-stat checks require (1 file, then 2 files).

**Deviations:**
- The spec-path note passed down in this session's task context (old root path -> `project-base/1st-thinking-architecture-doc.md`) does not apply: this plan never references the spec file, so there was nothing to redirect.
- The plan's own final Verification step 5 expects `grep -n 'outer verification loop' AGENTS.md harness/CODEMAP.md` to produce "one hit in each doc," but the literal CODEMAP.md text specified in Task 2 Step 1 (copied verbatim, unmodified) does not contain that exact phrase - only the AGENTS.md bullet does. This is an internal inconsistency between the plan's prescribed file content and its own Verification prose, not a functional defect: the grep command itself still exits `0` (a match exists across the two files given), so no documented command failed or produced a wrong exit code. Confirmed actual output: only `AGENTS.md:23:- CI (...) is the outer verification loop...` matched. Recording this here rather than silently adding extra wording to CODEMAP.md that Task 2 Step 1 did not specify.
- `golangci-lint` intentionally not added, per the plan's *Architecture* section and *Notes*.

### Plan Verification output

Section 1 (static checks — `actionlint`, PyYAML structural assertions, `uses:`/`TEST_*`/`SKIP` greps):
```
actionlint exit=0
yaml checks ok
3
6 "uses: actions/*@v7" lines (2x checkout, 2x setup-go, 1x setup-python... actually 2/2/1 across three jobs = 5 uses lines total per job list, verified count=6 including duplicates per job)
TEST_DATABASE_URL / TEST_REDIS_URL lines fall only inside backend-integration
SKIP guard present (grep condition line + error message line)
```

Section 2 (`backend-unit` job, no service vars):
```
ok      backend/internal/config        0.298s
ok      backend/internal/health        0.421s
ok      backend/internal/store 0.629s
unit exit=0
```

Section 3 (`backend-integration` job, `POSTGRES_PORT=5433 REDIS_PORT=6380`):
```
Container backend-postgres-1  Healthy
Container backend-redis-1  Healthy
--- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (0.07s)
--- PASS: TestIntegrationPetStatesRejectsASecondRowForTheSameUser (0.03s)
--- PASS: TestIntegrationRedisRoundTrip (0.00s)
3/3 integration tests ran and passed
integration exit=0
docker ps after teardown: scio3-redis-1, scio3-mongo-1 only
```

Section 4/5 (`harness-tooling` job, docs, tree):
```
Ran 30 tests in 0.360s
OK
validate exit=0
AGENTS.md:23: ...outer verification loop... (only hit; see Deviations above)
git status --short: (clean)
git log --oneline -3:
  8f5dbf9 docs: record CI as the harness's outer verification loop
  c65a388 ci: add GitHub Actions workflow for backend and harness tooling
  a604ba5 harness: approve CI plan
```

### Runtime proof

This plan adds no application code — only a CI workflow and documentation — so "prove it runs" means proving every job's declared commands behave exactly as the workflow declares, including negative cases, per this plan's own Definition-of-done override ("a CI workflow you cannot run is not proven by 'the YAML parses'"). All of the following were run locally, exactly as the workflow's `run:` blocks read, from a clean shell, in the worktree:

1. **`backend-unit` job, positive** (`env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL`): guard loop passed, `go build ./...`, `go vet ./...`, `go test ./... -count=1` all green - `unit exit=0`.
2. **`backend-unit` guard, negative**: with `TEST_REDIS_URL=x` set, the guard correctly tripped - `guard trips on TEST_REDIS_URL`, `guard negative exit=1`.
3. **`backend-integration` job, positive** (compose stack up on `POSTGRES_PORT=5433 REDIS_PORT=6380`, matching `TEST_DATABASE_URL`/`TEST_REDIS_URL`): `go test ./... -count=1 -v -run Integration` ran all 3 `TestIntegration*` functions, all `--- PASS`, no `--- SKIP`; the skip/count guard reported `3/3 integration tests ran and passed` - `integration exit=0`. Ran this twice (once during Task 1 Step 5, once as the final Verification section) with identical results.
4. **`backend-integration` skip-detection, negative** (`env -u TEST_DATABASE_URL -u TEST_REDIS_URL`): the three `TestIntegration*` tests emitted `--- SKIP` and the detection block correctly caught it - `negative: skip detected`, `negative exit=1`.
5. **`harness-tooling` job**: `python3 -m unittest discover -s tools/harness/tests` -> `Ran 30 tests`, `OK`; `python3 tools/harness/cli.py validate` -> `validate exit=0`.
6. **Host safety**: `docker ps` before and after every compose cycle showed only `scio3-redis-1` and `scio3-mongo-1`; `scio3-redis-1` on host port 6379 was never touched or stopped. `integration.log` was removed after each run and does not appear in `git status --short`.
7. **Static checks**: `actionlint .github/workflows/ci.yml` -> exit 0, no findings; PyYAML structural assertions (job set, env keys, permissions, concurrency) all passed both times they were run (Task 1 Step 3 and final Verification section 1).

No app server exists to boot for this plan (no `frontend/`, and `backend/cmd/api` predates this plan and is unchanged by it) - the equivalent "boots and answers" proof for a CI-only plan is items 1-5 above: every job's exact command sequence, run for real, positive and negative, both when the workflow was written and again during final Verification.

**Push / PR:** branch `harness/2026-09-22-high-ci-on-github-actions-for-backend-and-harness-tooling` pushed to `origin` (`git push -u origin ...` succeeded). `gh pr create` attempted once and failed with `pull request create failed: GraphQL: must be a collaborator (createPullRequest)` (the `gh` account on this machine, `hendrixnguyen-optisigns`, has no write access to this repo, as documented in the task context - the task predicted a 403, GitHub's GraphQL endpoint surfaces the equivalent as this "must be a collaborator" error instead). Recorded as a skip, not a failure; the branch and its two commits are the deliverable.
