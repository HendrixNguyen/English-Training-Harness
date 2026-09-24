---
idea: harness/ideas/_inbox/ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md
status: done
priority: medium
merged: false
branch: harness/2026-09-24-medium-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a
worktree: .worktrees/ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a
---
# CI: a `gofmt` gate and the race detector in `backend-unit` — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea (head):** `harness/ideas/_inbox/ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md`
**Also planned here (each idea's frontmatter points at this plan):**
- `harness/ideas/_inbox/ci-never-runs-go-test-race-so-background-goroutine-races-go-.md` → Task 3

The duplicate inbox filing `gofmt-l-has-been-failing-on-two-internal-quests-files-since-.md` is rejected pointing here.

**Goal:** An unformatted Go file, or a data race between the API's long-lived goroutines (`pet.RunHourly`, `notify.RunWorker`) and anything else, turns the `harness/*` branch that introduces it red before the owner sees the daily PR — and the two files that have been drifting on `main` are formatted so the gate passes on day one.

**Why now (`priority: medium`):** `.github/workflows/ci.yml`'s `backend-unit` runs `go build`, `go vet`, `go test ./... -count=1` and nothing else: `grep -n 'gofmt\|race' .github/workflows/ci.yml` is empty. `gofmt -l .` in `backend/` on this branch prints `internal/quests/handler_test.go` and `internal/quests/repo.go`, and every executor's *Definition of done* has been re-verifying that those two are not theirs. The notify executor found a real read/write race with `-race` locally and noted CI would have stayed green either way. Developer-workflow bugs rank with user bugs in the standing priority (AGENTS.md: "bugs that affect users, data or the developer workflow"); this one costs nothing in user-facing risk and touches only CI, one Makefile and two files' whitespace, so it cannot conflict with today's other four plans.

**Root cause:** the CI slice (`harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md`) was written before any background goroutine existed and chose the minimum build/vet/test trio; formatting was left to each executor's manual check, which does not scale to a pipeline of agents.

**Design decisions:**
1. **The gate prints the offenders.** `test -z "$(gofmt -l .)"` alone fails silently; the step echoes the list under a `::error::` annotation so the fix is one `gofmt -w` away.
2. **`-race` on the whole unit suite, in the existing step.** The suite is pure (integration tests skip in `backend-unit`), so the race build's 2–10× slowdown is paid on seconds, not minutes; one command keeps one signal. Task 3 measures the wall time and records it; the documented fallback (a second step scoped to `./internal/notify/... ./internal/pet/...`) is used only if the full run threatens the job's `timeout-minutes: 10`.
3. **`make check` mirrors the job** (`fmt-check` + `vet` + `test -race`) so an executor can run exactly what CI will, without slowing the plain `make test` loop.
4. **Format only what `gofmt -l .` prints at execution time** — the idea says three files (2026-09-23), this branch shows two; the command is the truth.

**Tech stack:** GitHub Actions (`actions/setup-go@v7`, unchanged), `gofmt`, `go test -race` (needs cgo; present on `ubuntu-latest` and macOS). No new dependencies.

**Run every command from the worktree root** unless a step says otherwise; `backend/` commands say so. `rg` is not installed — use `grep -n`.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/quests/handler_test.go`, `backend/internal/quests/repo.go` (whatever `gofmt -l .` lists) | `gofmt -w` — whitespace only |
| `.github/workflows/ci.yml` | `backend-unit`: new `gofmt` step before Vet; test step gains `-race` |
| `backend/Makefile` | `fmt-check`, `vet`, `check` targets |
| `harness/CODEMAP.md` | CI section: `backend-unit` names the two new checks |

---

## Tasks

### Task 1: Format the drifted files

**Files:**
- Modify: whatever `gofmt -l .` prints from `backend/` (today: `internal/quests/handler_test.go`, `internal/quests/repo.go`)

- [ ] **Step 1: See the drift**

Run (from `backend/`): `gofmt -l .`
Expected: the two quests files (record the exact list in the execution summary).

- [ ] **Step 2: Format them and prove it is whitespace only**

```bash
gofmt -d $(gofmt -l .)                 # read the hunks: comment alignment, nothing semantic
gofmt -w $(gofmt -l .)
gofmt -l .                             # expect: no output
git diff --stat                        # expect: only the listed files, small +/− counts
go build ./... && go vet ./... && go test ./internal/quests/ -count=1   # expect: ok
```

- [ ] **Step 3: Commit**

```bash
git add -u internal/
git commit -m "backend: gofmt the quests files CI never checked"
```

### Task 2: `backend-unit` gains a `gofmt` gate that names the offenders

**Files:**
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Write the step's shell as a local script first, and prove it red and green**

```bash
cat > /tmp/gofmt-gate.sh <<'EOF'
set -o pipefail
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
  echo "::error::gofmt -l reports unformatted Go files — run: gofmt -w <file>"
  echo "$unformatted"
  exit 1
fi
echo "gofmt: clean"
EOF
cd backend && sh /tmp/gofmt-gate.sh; echo "exit=$?"
# expect: gofmt: clean / exit=0   (after Task 1)
printf 'package probe\n\nfunc  F() {}\n' > internal/zz_probe.go
sh /tmp/gofmt-gate.sh; echo "exit=$?"
# expect: the ::error:: line, then internal/zz_probe.go, then exit=1
rm internal/zz_probe.go && cd ..
```

- [ ] **Step 2: Add the step to `ci.yml`** — in `backend-unit`, between `Build` and `Vet`:

```yaml
      - name: gofmt
        run: |
          set -o pipefail
          unformatted=$(gofmt -l .)
          if [ -n "$unformatted" ]; then
            echo "::error::gofmt -l reports unformatted Go files — run: gofmt -w <file>"
            echo "$unformatted"
            exit 1
          fi
          echo "gofmt: clean"
```

- [ ] **Step 3: Validate the workflow file**

Run: `actionlint .github/workflows/ci.yml` if installed (`which actionlint`); otherwise `ruby -ryaml -e 'YAML.load_file(".github/workflows/ci.yml"); puts "yaml ok"'`.
Expected: no findings / `yaml ok`.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: backend-unit fails on unformatted Go files, naming them"
```

### Task 3: `backend-unit` runs the unit suite under the race detector

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `backend/Makefile`

- [ ] **Step 1: Measure locally** (from `backend/`; the unit job's contract is "no service variables", so unset them)

```bash
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL bash -c 'time go test ./... -count=1 -race'
```

Expected: `ok` for every package; record the `real` time in the execution summary. Decision rule: under 4 minutes → Step 2a; otherwise → Step 2b.

- [ ] **Step 2a (default): one command, race on** — change the `Test without services` step's `run:` to:

```yaml
        run: go test ./... -count=1 -race
```

- [ ] **Step 2b (fallback, only if Step 1 exceeded 4 minutes):** keep the plain step and add after it:

```yaml
      - name: Race detector on the packages with goroutines
        run: go test ./internal/notify/... ./internal/pet/... -count=1 -race
```

and say so in the CODEMAP sentence in Task 4.

- [ ] **Step 3: Makefile** — add after `tidy:`:

```make
# What CI's backend-unit job runs, so a branch can be checked before it is pushed.
.PHONY: fmt-check vet check
fmt-check:
	@unformatted=$$(gofmt -l .); if [ -n "$$unformatted" ]; then echo "gofmt -l:"; echo "$$unformatted"; exit 1; fi

vet:
	go vet ./...

check: fmt-check vet
	go test ./... -count=1 -race
```

Run: `cd backend && make check` → expected: `ok` for every package, exit 0.

- [ ] **Step 4: Validate and commit**

```bash
actionlint .github/workflows/ci.yml 2>/dev/null || ruby -ryaml -e 'YAML.load_file(".github/workflows/ci.yml"); puts "yaml ok"'
git add .github/workflows/ci.yml backend/Makefile
git commit -m "ci: run the unit suite under -race; make check mirrors backend-unit"
```

### Task 4: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (CI section)

- [ ] **Step 1: Edit** — the `**backend-unit**` bullet: "from `backend/`: `go build ./...`, a `gofmt -l .` gate that fails naming every unformatted file, `go vet ./...`, `go test ./... -count=1 -race` (the race detector runs on every push because the production binary carries two long-lived goroutines, `pet.RunHourly` and `notify.RunWorker`), after a guard that fails if …". Replace the closing "No linter yet: `golangci-lint` defaults …" paragraph's first words with "No linter beyond `gofmt` yet: …". Add to the `**store**` bullet's test sentence or the CI section: "`make check` runs the same three checks locally."

If Step 2b was taken, say "…`go test ./... -count=1`, then `-race` on `./internal/notify/... ./internal/pet/...` (full-suite race run measured at N min locally, too slow for the 10-minute cap)".

- [ ] **Step 2: Commit**

```bash
git add harness/CODEMAP.md
git commit -m "harness: CODEMAP names the gofmt gate and the race run"
```

---

## Verification

```bash
cd backend && gofmt -l . ; echo "exit=$?"
# expect: no output, exit=0
make check
# expect: fmt-check silent, go vet silent, `ok` for every package under -race
cd ..
grep -n 'gofmt' .github/workflows/ci.yml
# expect: the step name and the `gofmt -l .` line, positioned after Build and before Vet
grep -n '\-race' .github/workflows/ci.yml
# expect: 1 line (Step 2a) or 1 line in a separate step (Step 2b)
grep -n 'gofmt\|race' harness/CODEMAP.md
# expect: the backend-unit sentence names both
actionlint .github/workflows/ci.yml 2>/dev/null || ruby -ryaml -e 'YAML.load_file(".github/workflows/ci.yml"); puts "yaml ok"'
# expect: clean / yaml ok
git log --oneline origin/main..HEAD | wc -l
# expect: 4 commits, one per task, each with the Co-Authored-By trailer
python3 tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0

# The outer loop is the proof: after `git push -u origin <branch>`,
gh run list --branch "$(git branch --show-current)" --limit 1
# expect: the run completes; then
gh run view "$(gh run list --branch "$(git branch --show-current)" --limit 1 --json databaseId --jq '.[0].databaseId')" --log 2>/dev/null | grep -E 'gofmt: clean|-race|ok  ' | head
# expect: "gofmt: clean" in backend-unit's log and the -race test step green
```

Negative proof (local, Task 2 Step 1): the gate script exits 1 and names `internal/zz_probe.go` when an unformatted file exists; recorded in the execution summary. If you want the same proof on GitHub, push a throwaway commit with the probe file on this branch, watch `backend-unit` fail on the `gofmt` step, then `git revert` it — optional, and say so in the summary.

## Notes and open questions

- **Why replace rather than add a second test step (Step 2a)?** A race report is a failure like any other; two full runs double the job's time for one extra signal. The fallback exists for the case the measurement says the full race run is too slow.
- **`-race` and integration tests:** unchanged — `backend-integration` stays a plain run. Racing against a shared database with `-p 1` would only slow it, and its tests are sequential by design.
- **Formatting the two files** touches `internal/quests/repo.go`, which today's quests/pet plan also edits (different regions: a struct comment alignment vs. new SQL and a method). If both land the same day, the merge is trivial; whichever executor goes second runs `gofmt -l .` after merging `origin/main`.
- **Out of scope:** `golangci-lint` (CODEMAP already records what adopting it would need), `staticcheck`, a frontend `-race` analogue (none exists).

## Execution summary

**Status: done.** Branch `harness/2026-09-24-medium-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a`, worktree `.worktrees/ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a`, based on freshly fetched `origin/main` (`fe2c29a`).

**Tasks, exactly as planned, no deviations:**
1. `gofmt -l .` from `backend/` printed exactly the two files the plan named (`internal/quests/handler_test.go`, `internal/quests/repo.go`); `gofmt -d` confirmed whitespace-only (comment alignment); `gofmt -w` + `gofmt -l .` (empty) + `go build ./... && go vet ./... && go test ./internal/quests/ -count=1` (ok) → commit `76c5717`.
2. Wrote the gate script to a scratch file, proved it green (`gofmt: clean`, exit=0) after Task 1 and red (`::error::...`, then `internal/zz_probe.go`, exit=1) with a probe file, removed the probe, then added the `gofmt` step to `.github/workflows/ci.yml` between `Build` and `Vet`. `actionlint .github/workflows/ci.yml` — clean. Commit `cafab95`.
3. Measured `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL bash -c 'time go test ./... -count=1 -race'` from `backend/`: all 9 tested packages `ok`, `real 0m8.199s` — well under the 4-minute threshold, so **Step 2a** (single command, race on) applied, not the fallback. Changed the `Test without services` step to `go test ./... -count=1 -race`. Added `fmt-check`, `vet`, `check` targets to `backend/Makefile` after `tidy:`, exactly as specified. `make check` → fmt-check silent, vet silent, all 9 packages `ok` under `-race`, exit 0. `actionlint` clean. Commit `ebd170a`.
4. Updated the CODEMAP `backend-unit` bullet to name the `gofmt` gate and the race run (with the `pet.RunHourly`/`notify.RunWorker` justification) and appended "`make check` runs the same three checks locally."; changed "No linter yet: `golangci-lint`…" to "No linter beyond `gofmt` yet: `golangci-lint`…". Commit `76bb6ca`.

**Plan verification (all as expected):**
```
cd backend && gofmt -l . ; echo "exit=$?"        → (no output) exit=0
make check                                        → fmt-check silent, go vet silent, ok × 9 packages under -race, exit 0
grep -n 'gofmt' .github/workflows/ci.yml          → the gofmt step, positioned after Build and before Vet
grep -n '\-race' .github/workflows/ci.yml         → 1 line: `run: go test ./... -count=1 -race` (Step 2a)
grep -n 'gofmt\|race' harness/CODEMAP.md          → backend-unit sentence names both; make check sentence present
actionlint .github/workflows/ci.yml               → clean (exit 0)
git log --oneline origin/main..HEAD | wc -l       → 4 commits, each with the Co-Authored-By trailer (verified via git log --format)
python3 tools/harness/cli.py validate             → exit=0
```
Negative proof (Task 2 Step 1, local only): gate script exited 1 and named `internal/zz_probe.go` when an unformatted probe file was present; not repeated as a throwaway push to GitHub (optional per the plan).

**Runtime proof (Definition of done, step 8):**
- **Build:** `go build -o /tmp/aelp-api-check-exec-ci ./cmd/api` — succeeded, no errors/warnings.
- **Full suite, clean shell:** `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL` + `make check` (fmt-check, vet, `go test ./... -count=1 -race`) — all 9 tested packages `ok`, exit 0. Run twice (measurement + final verification), identical result both times.
- **Boots and answers, end to end:** started an isolated stack (`COMPOSE_PROJECT_NAME=exec-ci`, `POSTGRES_PORT=55437`, `REDIS_PORT=56384`) with `docker compose up -d --wait --wait-timeout 120`, ran the built binary on `PORT=18086` with dummy `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`/`JWT_SECRET`. Log showed migrations applied (`0001_init 0002_google_sync 0003_pet_verdict_dates`) and `listening on :18086`. `curl http://localhost:18086/healthz` → `{"postgres":"ok","redis":"ok","status":"ok"}`, HTTP 200.
- **Documented commands:** `gofmt -l .`, `make check`, `actionlint` all run as documented above, matching their documented behavior.
- **Cleanup:** the app process ignored SIGTERM (pre-existing bug on this `origin/main` base — no server-shutdown wiring on signal, being fixed in a separate in-flight plan/branch not yet merged; out of scope here) and needed `SIGKILL`; confirmed gone via `pgrep`. `docker compose down` (exec-ci project) removed both containers; the scratch `exec-ci_postgres_data` volume was removed explicitly since it was created only for this check. Deleted the scratch `backend/.env`, the built binary, and the log file. Final `pgrep -fl "exe/api|aelp-api"` → none; `docker ps -a --filter name=exec-ci` → none; `docker volume ls | grep exec-ci` → none; `git status --short` in the worktree → clean.

**CI on the pushed branch:** run [35959338732](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35959338732) — all 4 jobs green (`frontend`, `harness-tooling`, `backend-integration`, `backend-unit`). Confirmed in `backend-unit`'s job log: the `gofmt` step printed `gofmt: clean`, and `Test without services` ran `go test ./... -count=1 -race` with all 9 packages `ok`.
