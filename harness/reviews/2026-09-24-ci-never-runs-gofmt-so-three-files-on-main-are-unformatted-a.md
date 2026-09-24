---
plan: harness/plans/2026-09-24-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/backend-integration-never-runs-race-though-its-pet-and-store.md, harness/ideas/_inbox/make-check-does-not-mirror-backend-unit-no-service-variable-.md]
---
# Review — CI: a `gofmt` gate and the race detector in `backend-unit`

**Plan:** `harness/plans/2026-09-24-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md`
**Branch/worktree:** `harness/2026-09-24-medium-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a` / `.worktrees/ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a`
**Diff:** `git diff origin/main...harness/2026-09-24-medium-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a --stat`

## Plan vs idea
Both ideas are delivered.

- **gofmt idea** (head): `backend-unit` has a `gofmt` step before `Vet` that prints the offending files under a
  `::error::` annotation; the drifted files are formatted in the same change; CODEMAP names the step. The idea
  listed three files (2026-09-23); `internal/google/fakes_test.go` had since been formatted, and `gofmt -l .` on
  `origin/main` today prints exactly the two this branch fixes. Planned "format what the command prints" was right.
- **race idea** (Also planned here): `Test without services` is `go test ./... -count=1 -race`, the full unit suite,
  not the scoped fallback. The idea asked for "the unit suite under the race detector", so this meets its letter.
  Its intent — "a data race introduced on a `harness/*` branch fails a check before merge" — is only partly met:
  the integration tests that concurrently drive `pet.Service.Ensure`, `pet.PgRepo.PenaliseMiss` and `store.Migrate`
  still run without `-race` (bug 1). The plan's stated reason for scoping it out ("its tests are sequential by
  design") is factually wrong.

## Code vs plan
Diff (`git diff origin/main...HEAD --stat`, `origin/main` freshly fetched): 5 files, +26/−5 — `ci.yml` +11/−1,
`Makefile` +11, `quests/handler_test.go` 1/1, `quests/repo.go` 1/1, `CODEMAP.md` 2/2. 4 commits, each with a
Co-Authored-By trailer. Branch is 2 commits behind `origin/main` (`8e69b3b`, `9517f25`, harness/docs only); it
merges cleanly.

- **Task 1 (format)** — followed. Go diff is whitespace only (normalising whitespace, the ± lines are identical:
  one struct-field comment and one table-test comment realigned).
- **Task 2 (gofmt gate)** — followed; step sits between `Build` and `Vet` (`ci.yml:41-50`). `actionlint` clean.
- **Task 3 (-race + Makefile)** — followed, Step 2a (`ci.yml:54`); `fmt-check`/`vet`/`check` targets exactly as
  specified (`Makefile:12-21`).
- **Task 4 (CODEMAP)** — followed; the `backend-unit` bullet is accurate against `ci.yml`. No CODEMAP fix needed.

Re-run evidence (reviewer, 2026-09-24, worktree at `76bb6ca`, no service variables exported, local go1.27.1):
```
cd backend && gofmt -l . ; echo exit=$?        → (no output) exit=0
make check                                      → vet silent; go test ./... -count=1 -race: ok × 10 packages
                                                  (airouter auth config google health notify onboarding pet
                                                  quests store; cmd/api has no tests), exit 0
grep -n gofmt .github/workflows/ci.yml          → 41 (step), 44 (gofmt -l .), 46, 50 — after Build, before Vet
grep -n '\-race' .github/workflows/ci.yml       → 54: run: go test ./... -count=1 -race
grep -c 'gofmt\|race' harness/CODEMAP.md        → 3
actionlint .github/workflows/ci.yml             → clean
cli.py validate                                 → exit 0
```
The execution summary says "9 tested packages"; there are 10 `ok` lines both locally and in CI. Cosmetic.

Negative proof — the CI step's script extracted verbatim from `ci.yml` and run under `bash -eo pipefail`, and
`make fmt-check`, each against a probe `func  F() {}` file, removed after each run (`git status --short` empty):
```
internal/zz_probe.go            CI gate: ::error:: + path, exit 1   make fmt-check: path, exit 2
internal/testdata/zz_probe.go   CI gate: exit 1 (named)             make fmt-check: exit 2
.hidden/zz_probe.go             CI gate: exit 1 (named)             make fmt-check: exit 2
_skip/zz_probe.go               CI gate: exit 1 (named)             make fmt-check: exit 2
parse error (func F( {})        CI gate: exit 2 (no annotation)     make fmt-check: exit 0  ← bug 2
```

CI on the pushed branch — run 35959338732, head `76bb6ca` (= local HEAD = remote branch), all four jobs `success`.
`backend-unit` log: setup-go `go1.25.0 linux/amd64`, shell `/usr/bin/bash -e {0}`; `gofmt` step printed
`gofmt: clean`; `Test without services` ran `go test ./... -count=1 -race` with 10 `ok` lines, ~50 s wall.
`backend-integration` log: `go test ./... -count=1 -v -run Integration -p 1` (no `-race`), `12/12 integration tests ran and passed`.

Integration suite under `-race`, isolated stack (`COMPOSE_PROJECT_NAME=rev-ci`, Postgres 55445, Redis 56395),
`go test ./... -count=1 -v -run Integration -p 1 -race -timeout 300s` → exit 0, 12/12 `--- PASS`, no `DATA RACE`,
every package ≤ 1.8 s. `docker compose down -v` afterwards; no `rev-ci` containers or volumes remain.

## Quality
- **Bypass.** `gofmt -l .` runs from `backend/` and walks every directory, including `testdata`, dot- and
  underscore-prefixed ones (probes above), so there is no vendored/generated escape hatch today — the repo has no
  `vendor/`, no `// Code generated` files, and **no `.go` file outside `backend/`**. A Go file added outside
  `backend/` (e.g. a future `tools/` Go helper) would bypass the gate; not a bug today, worth one line in whatever
  plan first adds one.
- **Race coverage.** Unit: full suite, including the goroutine tests in `notify/worker_test.go:14`,
  `pet/cron_test.go:29`, `pet/service_test.go:378`. Integration: not covered — `pet/integration_test.go:53,175`
  and `store/integration_test.go:129` are concurrency tests of real repositories, and they pass under `-race` in
  seconds, so there is no cost reason to leave them out (bug 1, medium).
- **Local mirror.** `make check` omits the job's service-variable guard (with `TEST_*` exported it runs the
  destructive integration tests without `-p 1`) and `fmt-check` swallows gofmt's parse-error exit (bug 2, low).
- **Toolchain skew (note, no bug).** CI's gofmt is go1.25.0 (`GOTOOLCHAIN: local`, `go-version-file`); this machine
  has go1.27.1, so `make check` formats with a newer gofmt than the gate. gofmt's output has been stable across
  these versions for this tree (both agree it is clean), but a divergence would show as "green locally, red in CI".
- **Nit.** `set -o pipefail` in the `gofmt` step does nothing (no pipeline); harmless, copied from the plan.
- **Conflict with the sibling quests branch.** `git merge-tree --write-tree` of this branch and
  `origin/harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile` (`db1f5e8`) →
  clean tree `813ab45`, exit 0; the one-line comment realignment in `quests/repo.go` (struct `Exercise`, line 33) does
  not touch the quests branch's hunks. The quests branch on its own still carries the unformatted `repo.go`
  (pre-existing drift); the merged tree is gofmt-clean and `make check` on it is green (10 × `ok` under `-race`).
  A gofmt scan of every other `harness/2026-09-24-*` branch's added/modified `.go` files found nothing else
  unformatted.
  **Integrator:** merge both into `harness/daily-2026-09-24` in either order — no conflict, no manual step. Do not
  merge the quests branch without this one expecting the gate to be green on it alone (it is not, until combined),
  and run `cd backend && gofmt -l .` plus `make check` on the daily branch before opening the PR, since from this
  merge on every sibling's code is gated by gofmt and `-race` there.
- Test-gap analysis via the-validator skipped: diff is 26 lines of YAML/Make/whitespace, well under the
  200-line threshold. Code walked by hand; no Go logic changed.

## Bugs filed
- `harness/ideas/_inbox/backend-integration-never-runs-race-though-its-pet-and-store.md` — medium, not a blocker.
- `harness/ideas/_inbox/make-check-does-not-mirror-backend-unit-no-service-variable-.md` — low, not a blocker.

## Verdict
**pass-with-bugs.** Every task followed, every Verification command and the executor's evidence reproduce, the
gate fails on a misformatted file locally and ran (`gofmt: clean`, `-race`) in the branch's green CI run. The
two bugs are follow-ups, neither holds the branch. Merge with `/harness merge harness/plans/2026-09-24-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md`
(via the daily PR).
