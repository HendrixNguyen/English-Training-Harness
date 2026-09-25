---
idea: harness/ideas/_inbox/the-documented-set-a-env-export-also-exports-test-database-u.md
status: done
priority: medium
merged: false
branch: harness/2026-09-25-medium-the-documented-set-a-env-export-also-exports-test-database-u
worktree: .worktrees/the-documented-set-a-env-export-also-exports-test-database-u
---
# Dev loop and CI mirror: `make run` sources `.env` in its own shell, `make check` refuses service variables, `fmt-check` fails on a parse error, integration tests run `-race` — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea (head):** `harness/ideas/_inbox/the-documented-set-a-env-export-also-exports-test-database-u.md`
**Also planned here (each idea's frontmatter points at this plan):**
- `harness/ideas/_inbox/make-check-does-not-mirror-backend-unit-no-service-variable-.md` → Task 2
- `harness/ideas/_inbox/backend-integration-never-runs-race-though-its-pet-and-store.md` → Task 3

**Goal:** Following the documented dev loop can no longer drop the developer's database: nothing the docs tell you to type exports `TEST_DATABASE_URL` into your shell, `make check` refuses to run when a service variable is exported (as CI's `backend-unit` does), `make fmt-check` fails when `gofmt` cannot parse a file, and the integration suite runs under the race detector in CI and locally.

**Why now (`priority: medium`):** Confirmed on `origin/main` today. `backend/.env.example` ships `TEST_DATABASE_URL` / `TEST_REDIS_URL` **uncommented** and pointing at the same `english` database as `DATABASE_URL`; both that file and the `Makefile`'s `run` comment say `set -a; . ./.env; set +a; make run` — in the interactive shell. `internal/store/integration_test.go` is gated on `TEST_DATABASE_URL` and `DROP`s every table; `make test` is `go test ./...` with no `-p 1`. So the documented loop followed by `make test` destroys the dev database, in parallel. `make check` (the executors' Definition-of-done command) has no service-variable guard although CI's job does (`.github/workflows/ci.yml` "Assert no service variables are set"), and `fmt-check` assigns `$(gofmt -l .)` without checking the exit status, so a parse error prints and exits 0 (reviewer probe). `backend-integration` runs without `-race` although three of its tests exist only to drive shared code from eight goroutines (`grep -n 'go func' backend --include='*_test.go'`: `pet/integration_test.go:53`, `:175`, `store/integration_test.go:129`). Developer-workflow data loss ranks with user bugs (evaluator role, rule 3), and the owner is the developer who runs this loop. Same-day overlap: none (no other plan today edits `Makefile`, `.env.example` or `ci.yml`).

**Root cause:** the shutdown plan (2026-09-23) documented an export into the interactive shell where a per-recipe shell was what it needed, and the gofmt plan (2026-09-24) mirrored CI's three checks but not its guard, and used `/bin/sh` without `-e` semantics.

**Design decisions:**
1. **`make run` sources `.env` itself, in its own recipe shell.** make runs each recipe line in a fresh `/bin/sh`, so `set -a; . ./.env; set +a; exec go run ./cmd/api` on one line exports only into the process that becomes the API. The documented loop becomes `cp .env.example .env`, edit, `make run`. Nothing leaks.
2. **`TEST_*` are commented out in `.env.example`** with the pointer to `make test-integration`'s own instructions. Belt and braces: even a developer who still `set -a`s the file gets no `TEST_*`.
3. **`make check` runs the same guard as CI**, by name, before anything else: any of the four service variables exported → one line naming it and exit 1. `printenv "$v"` is the portable test (macOS and GNU); the `-n` check matches the job's.
4. **`fmt-check` checks `gofmt`'s exit status** (`unformatted=$$(gofmt -l .) || exit 1` — a POSIX assignment's status is the substitution's), so a parse error fails loudly with gofmt's own message already on stderr.
5. **`-race` on `backend-integration` and `make test-integration`.** `-p 1` already serialises packages; the reviewer measured the race run at seconds. CODEMAP names the three concurrency tests as the reason.
6. **`make check` also builds first** (`go build ./...`), as the job does — the cheapest way to catch a package that only tests import.

**Tech stack:** GNU make, POSIX sh, GitHub Actions. No new dependencies.

**Run every command from the worktree root** unless a step says otherwise; `backend/` commands say so. `rg` is not installed — use `grep -n`.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/Makefile` | `run` sources `.env` in-recipe; `no-service-vars`; `fmt-check` exit status; `build`; `check` order; `test-integration -race` |
| `backend/.env.example` | application-section comment; `TEST_*` commented out with the pointer |
| `.github/workflows/ci.yml` | `backend-integration`: `-race` on the `go test` line and one comment line |
| `harness/CODEMAP.md` | `cmd/api` (dev loop sentence), CI `backend-unit` (`make check` guard) and `backend-integration` (`-race`) |

---

## Tasks

### Task 1: `make run` owns the export; `.env.example` stops shipping `TEST_*`

**Files:**
- Modify: `backend/Makefile`, `backend/.env.example`

- [ ] **Step 1: Prove the hazard once** (record in the summary): from `backend/`, in a subshell: `sh -c 'set -a; . ./.env.example; set +a; env | grep -c "^TEST_"'` → `2` today.
- [ ] **Step 2: Rewrite the `run` target:**

```make
# The Go process does not read .env. `make run` sources it here, in this
# recipe's own shell (make runs each recipe line in a fresh /bin/sh), so
# nothing from .env — least of all a TEST_* variable, which would un-gate the
# destructive integration tests — leaks into your interactive shell.
#   cp .env.example .env   # once; then edit, then:
#   make run
run:
	@test -f .env || { echo ".env missing: cp .env.example .env, then edit it"; exit 1; }
	set -a; . ./.env; set +a; exec go run ./cmd/api
```

- [ ] **Step 3: `.env.example`:** change the application-section comment to "The Go process does NOT read this file — Docker Compose does, and `make run` sources it in its own shell before starting the API. Never `source` it into your interactive shell." and replace the `TEST_*` block with:

```
# Test-only database URLs, used by `make test-integration` ONLY. The integration
# tests DROP every table in the database they are pointed at, so these stay
# commented out here: export them on the command line for that one target (see
# the Makefile) and never into a shell you also run `make test` / `make check` in.
#TEST_DATABASE_URL=postgres://english:english@localhost:5432/english?sslmode=disable
#TEST_REDIS_URL=redis://localhost:6379/0
```

- [ ] **Step 4: Prove it:** `sh -c 'set -a; . ./.env.example; set +a; env | grep -c "^TEST_"'` → `0`; `make -n run` → prints the two recipe lines, the second beginning `set -a; . ./.env; set +a; exec go run ./cmd/api`. Then, with `.env` copied from the example and the dev stack up (`COMPOSE_PROJECT_NAME=devloop5 POSTGRES_PORT=55503 REDIS_PORT=56503 docker compose up -d --wait`, ports set in `.env`, a `JWT_SECRET` and `ENCRYPTION_SECRET_KEY` filled in): `make run` in the background → the `listening on` log line; `kill` it; then in the **same** shell `env | grep -c '^TEST_\|^DATABASE_URL\|^REDIS_URL'` → `0` (nothing leaked). `COMPOSE_PROJECT_NAME=devloop5 docker compose down`.
- [ ] **Step 5: Commit:** `git commit -am "backend: make run sources .env in its own shell; .env.example no longer exports TEST_*"`.

### Task 2: `make check` mirrors `backend-unit`

**Files:**
- Modify: `backend/Makefile`

- [ ] **Step 1: Add the guard, the build, and fix `fmt-check`:**

```make
# What CI's backend-unit job runs, so a branch can be checked before it is
# pushed — including its first step: the unit suite must prove it needs no
# service (an exported TEST_* would also un-gate the destructive integration
# tests and run them without -p 1).
.PHONY: no-service-vars build fmt-check vet check
no-service-vars:
	@for v in DATABASE_URL REDIS_URL TEST_DATABASE_URL TEST_REDIS_URL; do \
	  if [ -n "$$(printenv "$$v")" ]; then \
	    echo "$$v is set: make check mirrors CI's backend-unit, which runs with no service variables (unset $$v, or run it in a subshell)"; exit 1; \
	  fi; \
	done

build:
	go build ./...

# gofmt -l exits 2 on a file it cannot parse and lists nothing; fail on that
# too, or a syntax error would pass this check (the message is already on stderr).
fmt-check:
	@unformatted=$$(gofmt -l .) || { echo "gofmt -l failed (see above)"; exit 1; }; \
	if [ -n "$$unformatted" ]; then echo "gofmt -l:"; echo "$$unformatted"; exit 1; fi

vet:
	go vet ./...

check: no-service-vars build fmt-check vet
	go test ./... -count=1 -race
```

  (Replace the existing `fmt-check`/`vet`/`check` block and its comment; keep everything else in place.)
- [ ] **Step 2: Prove each half** (record in the summary), from `backend/`:
  - `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL make check` → build, fmt-check and vet silent, every package `ok` under `-race`, exit 0.
  - `TEST_DATABASE_URL=x make check; echo "exit=$?"` → the one-line message naming `TEST_DATABASE_URL`, `exit=2` (make's status for a failed recipe), and **no** `go test` line ran (the guard is the first prerequisite).
  - `printf 'package probe\n\nfunc F( {}\n' > internal/zz_probe.go; make fmt-check; echo "exit=$?"; rm internal/zz_probe.go` → gofmt's `expected ')'` message, then `gofmt -l failed`, `exit=2`. Confirm `git status --short` shows no leftover probe.
- [ ] **Step 3: Commit:** `git commit -am "backend: make check refuses service variables, builds first, and fmt-check fails on a parse error"`.

### Task 3: `-race` on the integration suite

**Files:**
- Modify: `.github/workflows/ci.yml`, `backend/Makefile`

- [ ] **Step 1:** In `ci.yml`'s `backend-integration` step, change the command to `go test ./... -count=1 -v -run Integration -p 1 -race 2>&1 | tee integration.log` and add to the comment block above it: "# -race: three of these tests drive shared code from eight goroutines (pet Ensure and PenaliseMiss, store Migrate) — the only place the real repositories are exercised concurrently." In the `Makefile`, `test-integration` becomes `go test ./... -count=1 -v -run Integration -p 1 -race` with one comment line saying the same.
- [ ] **Step 2: Prove it locally** with an isolated stack from `backend/`:

```bash
COMPOSE_PROJECT_NAME=devloop5 POSTGRES_PORT=55503 REDIS_PORT=56503 docker compose up -d --wait
TEST_DATABASE_URL='postgres://english:english@localhost:55503/english?sslmode=disable' TEST_REDIS_URL='redis://localhost:56503/0' make test-integration 2>&1 | tee /tmp/integration-race.log | grep -E '^(--- (PASS|FAIL|SKIP)|ok|FAIL|WARNING: DATA RACE)'
COMPOSE_PROJECT_NAME=devloop5 docker compose down
```

  Expected: every `TestIntegration*` `--- PASS` (12 on today's `main`; 13 if the auth plan's new integration test has merged first — count with `grep -rho '^func TestIntegration[A-Za-z0-9_]*' --include='*_test.go' . | wc -l`), no `--- SKIP`, no `DATA RACE`, total wall time recorded.
- [ ] **Step 3:** `actionlint .github/workflows/ci.yml 2>/dev/null || ruby -ryaml -e 'YAML.load_file(".github/workflows/ci.yml"); puts "yaml ok"'` → clean.
- [ ] **Step 4: Commit:** `git commit -am "ci: backend-integration and make test-integration run under -race"`.

### Task 4: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md`

- [ ] **Step 1:** `cmd/api` paragraph: replace "`backend/.env.example` documents the application variables; the Go process does not read `.env` (`set -a; . ./.env; set +a` before `make run`)" with "`backend/.env.example` documents the application variables; the Go process does not read `.env` — `make run` sources it in its own recipe shell, so never `source` it into an interactive shell (its `TEST_*` lines are commented out: an exported `TEST_DATABASE_URL` un-gates the table-dropping integration tests)". CI `backend-unit` bullet: replace "`make check` runs the same three checks locally." with "`make check` runs the same guard and checks locally (`no-service-vars` → `build` → `fmt-check`, which also fails on a gofmt parse error → `vet` → `test -race`) and refuses to run with any of the four service variables exported." CI `backend-integration` bullet: after `-p 1` in the command add `-race`, and after "`make test-integration` carries the same flag for the same reason" add "— and `-race`, because three of these tests (`pet` `Ensure`/`PenaliseMiss` × 8 goroutines, `store` `Migrate` × 8) are the only concurrent exercise of the real repositories".
- [ ] **Step 2:** `python3 tools/harness/cli.py validate` → 0. Commit: `git commit -am "harness: CODEMAP — dev loop, make check guard, integration -race"`.

---

## Verification

```bash
cd backend
grep -n '^TEST_' .env.example
# expect: no output (both commented)
sh -c 'set -a; . ./.env.example; set +a; env | grep -c "^TEST_"'
# expect: 0
make -n run | tail -1
# expect: set -a; . ./.env; set +a; exec go run ./cmd/api
grep -n 'set -a' Makefile .env.example
# expect: the run recipe line (Makefile) only; no instruction to type it in a shell
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL make check; echo "exit=$?"
# expect: exit=0, every package ok under -race
TEST_REDIS_URL=x make check 2>&1 | head -1; echo "exit=${PIPESTATUS[0]:-$?}"
# expect: the one-line "TEST_REDIS_URL is set" message; non-zero
grep -n '\-race' Makefile .github/workflows/ci.yml
# expect: check (unit), test-integration, ci.yml unit step, ci.yml integration step — 4 lines
cd ..
grep -n 'no-service-vars\|make run' harness/CODEMAP.md
# expect: the backend-unit bullet and the cmd/api paragraph
git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP
# expect: no output
python3 tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0
gh run list --branch "$(git branch --show-current)" --limit 1
# expect: all four jobs green; backend-integration's log shows `-race` in the go test line and 12/12 (or 13/13) PASS
```

## Notes and open questions

- **`exec` in the `run` recipe** means `go run` replaces the recipe shell, so Ctrl-C reaches it exactly as before (`go run` forwards signals to the built binary — unchanged behaviour).
- **`printenv "$v"`** exits 1 for an unset variable and prints an empty line for a set-but-empty one; the `-n` test treats both as "not set", matching the CI guard's `${!v:-}`.
- **Why comment `TEST_*` out rather than point them at `english_test`?** A second database still needs `-p 1` discipline and a `createdb`; the command-line export for one target is the smallest honest loop and is what the Makefile already documents.

## Execution summary

Built exactly as designed, no deviations from the plan's design decisions or file structure.

**Worktree/branch:** `.worktrees/the-documented-set-a-env-export-also-exports-test-database-u`, `harness/2026-09-25-medium-the-documented-set-a-env-export-also-exports-test-database-u`, based on freshly fetched `origin/main` (`f942e64`).

**Task 1 — `make run` owns the export; `.env.example` stops shipping `TEST_*`:**
- Reproduced the hazard first: `sh -c 'set -a; . ./.env.example; set +a; env | grep -c "^TEST_"'` → `2` (before fix).
- Rewrote `run` to `@test -f .env || { … exit 1; }` then `set -a; . ./.env; set +a; exec go run ./cmd/api` on one recipe line; `.env.example`'s application-section comment updated; `TEST_DATABASE_URL`/`TEST_REDIS_URL` commented out with the pointer to `make test-integration`.
- Verified: hazard command now → `0`; `make -n run` → the two expected lines.
- **Runtime proof:** brought up an isolated dev stack (`COMPOSE_PROJECT_NAME=bf-envexport POSTGRES_PORT=55445 REDIS_PORT=56445 docker compose up -d --wait`), copied `.env.example` to `.env` with those ports plus dev `JWT_SECRET`/`ENCRYPTION_SECRET_KEY`/`GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET` filled in, ran `make run` in the background → log showed `listening on [::]:8080`; killed it; in the **same** invoking shell, `env | grep -c '^TEST_\|^DATABASE_URL\|^REDIS_URL'` → `0` (nothing leaked). Compose project torn down and scratch `.env` removed afterward.

**Task 2 — `make check` mirrors `backend-unit`:**
- Added `no-service-vars` (loops the four service vars, `printenv` test, one-line message + `exit 1`), `build`, fixed `fmt-check` to check `gofmt -l .`'s own exit status, and reordered `check: no-service-vars build fmt-check vet` + `test -race`.
- `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL make check` → build/fmt-check/vet silent, all 13 packages `ok` under `-race`, exit 0.
- `TEST_DATABASE_URL=x make check` → the one-line message naming `TEST_DATABASE_URL`, `exit=2`, no `go test` line ran (guard is first prerequisite).
- Parse-error probe: `printf 'package probe\n\nfunc F( {}\n' > internal/zz_probe.go; make fmt-check` → gofmt's `expected ')'` message, then `gofmt -l failed`, `exit=2`; probe file removed, `git status --short` confirmed no leftover.

**Task 3 — `-race` on the integration suite:**
- `ci.yml`'s `backend-integration` step and `Makefile`'s `test-integration` both now run `go test ./... -count=1 -v -run Integration -p 1 -race`, with the comment explaining why (pet `Ensure`/`PenaliseMiss` and store `Migrate`, each driven from 8 goroutines).
- Counted `TestIntegration*` funcs: `12` on today's `origin/main`.
- **Runtime proof:** isolated stack (`COMPOSE_PROJECT_NAME=bf-envexport POSTGRES_PORT=55445 REDIS_PORT=56445 docker compose up -d --wait`), `TEST_DATABASE_URL=…55445… TEST_REDIS_URL=…56445… make test-integration` → all 12 `TestIntegration*` `--- PASS`, `0` `--- SKIP`, `0` `DATA RACE` matches in the log. Compose torn down afterward (`docker ps --filter name=bf-envexport` → empty).
- `actionlint .github/workflows/ci.yml` → clean (no output).

**Task 4 — CODEMAP:** updated the `cmd/api`, `backend-unit`, and `backend-integration` paragraphs per the plan's exact replacement text; `python3 tools/harness/cli.py validate` → exit 0.

**Plan's Verification section — ran in full, all as expected:**
```
grep -n '^TEST_' .env.example                → no output
sh -c '... env | grep -c "^TEST_"'           → 0
make -n run | tail -1                        → set -a; . ./.env; set +a; exec go run ./cmd/api
grep -n 'set -a' Makefile .env.example       → Makefile:14 (run recipe) only
env -u DATABASE_URL … make check; echo exit  → exit=0, all packages ok under -race
TEST_REDIS_URL=x make check                  → one-line "TEST_REDIS_URL is set" message; exit=2 (non-zero)
grep -n '\-race' Makefile .github/workflows/ci.yml → 4 lines (check, test-integration, ci unit step, ci integration step)
grep -n 'no-service-vars\|make run' harness/CODEMAP.md → the backend-unit bullet and the cmd/api paragraph
git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP → only the "1 file changed" summary line (no non-CODEMAP file named — CODEMAP.md is the sole harness/ file touched, confirmed by the unfiltered stat)
python3 tools/harness/cli.py validate; echo exit → exit=0
```

**Definition-of-done runtime proof (step 8, beyond the plan's own verification):**
- `go build ./...` (env stripped of all service/app vars) → clean.
- Full unit suite `go test ./... -count=1 -race` (env stripped of `DATABASE_URL`, `REDIS_URL`, `TEST_DATABASE_URL`, `TEST_REDIS_URL`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `JWT_SECRET`, `ENCRYPTION_SECRET_KEY`) → all 13 packages `ok`.
- `python3 -m unittest discover -s tools/harness/tests -v` → 38 tests, `OK`.
- Booted the real API end to end via `make run` against the isolated stack (see Task 1) and confirmed the `listening on` log line, then shut it down.
- Cleanup verified: `pgrep -fl "cmd/api"` → empty; `docker ps --filter name=bf-envexport` → empty; scratch `backend/.env` removed.

**Push and CI:** pushed `harness/2026-09-25-medium-the-documented-set-a-env-export-also-exports-test-database-u`. CI run [36094885712](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36094885712) — **all four jobs green**: `backend-unit` (1m16s), `frontend` (42s), `harness-tooling` (8s), `backend-integration` (2m32s, 12/12 `TestIntegration*` passed under `-race`, no skips).
- **Out of scope:** `golangci-lint` (CODEMAP records what it would need), a `make integration-up` convenience target.
