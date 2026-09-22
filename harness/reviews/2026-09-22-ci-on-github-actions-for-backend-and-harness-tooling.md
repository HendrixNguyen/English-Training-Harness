---
plan: harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/ci-never-runs-on-harness-branches-so-it-gates-nothing-before.md, harness/ideas/_inbox/cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md, harness/ideas/_inbox/ci-jobs-have-no-timeout-minutes-and-the-harness-job-floats-p.md, harness/ideas/_inbox/codemap-ci-section-overstates-the-unit-job-and-prescribes-a-.md]
---
# Review — CI on GitHub Actions for backend and harness tooling

**Plan:** `harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md`
**Branch/worktree:** `harness/2026-09-22-high-ci-on-github-actions-for-backend-and-harness-tooling` / `.worktrees/ci-on-github-actions-for-backend-and-harness-tooling`
**Diff:** `.github/workflows/ci.yml` (+105), `AGENTS.md` (+1), `harness/CODEMAP.md` (+4) — 110 insertions, 0 deletions, no app code.
**PR:** none. `gh pr create` failed for the executor with `must be a collaborator`; the account here has no write access, so PR steps are skipped (see bug 1, which is about exactly that).

## Plan vs idea

The idea's *Expected output* is delivered item for item: one `.github/workflows/ci.yml` on `push`→`main` + `pull_request`, `permissions: contents: read`, `concurrency` with `cancel-in-progress`, and the three named jobs with the prescribed commands, service containers, health checks and the "must run, not skip" assertion. `golangci-lint` is correctly left out — the idea made it conditional on the existing code passing with no changes and no silenced rules, and it does not (`errcheck` on `internal/store/redis_test.go:20`); the plan says so in *Architecture* and CODEMAP records the condition for re-adding it. The docs half is delivered: `AGENTS.md` gains the "outer verification loop" rule, `harness/CODEMAP.md` gains a CI paragraph.

One idea-level deviation, and it is an improvement: the idea specified `actions/checkout@v6`, the plan used `@v7`. I verified independently rather than trusting the evaluator's note — `gh api repos/actions/checkout/releases/latest` → `v7.0.1 (2026-07-20)`, `actions/setup-go` → `v7.0.0`, `actions/setup-python` → `v7.0.0`, and `git/ref/tags/v7` resolves on all three, so the moving major tags the workflow references exist. This was worth checking directly because `actionlint` cannot: I ran it against a copy with `actions/chekout@v7` and it reported nothing, while catching a `ubunut-latest` typo on the same file. A wrong action tag is the one failure mode the plan's entire local evidence set is structurally blind to; it is not present.

## Code vs plan

Task 1 (workflow) and Task 2 (docs) are both transcribed verbatim from the plan — I diffed the plan's code block against the committed file and they match character for character, including the `concurrency` group, the four-variable guard, the `want`/`got` derivation and the `::error::` annotations. Two commits, one per task, with exactly the staged shapes the plan required (1 file, then 2).

One logged deviation, judged **correct**: the plan's *Verification* step 5 greps for `outer verification loop` in both docs and expects "one hit in each", but the CODEMAP text the plan prescribes verbatim in Task 2 Step 1 does not contain that phrase — only the `AGENTS.md` bullet does. The executor followed the prescribed content rather than inventing wording to satisfy the plan's own prose, and recorded the inconsistency. That is the right call: the idea only asks that `AGENTS.md` note CI is the outer loop, and the grep as written still exits 0. Both documents are accurate as written on this point. (I do have separate accuracy complaints about the CODEMAP paragraph — bug 4 — but not this one.)

Executor evidence re-run in the worktree, all reproducing:

```
actionlint exit=0
yaml checks ok
on: {'push': {'branches': ['main']}, 'pull_request': None}
3                                    # hyphenated job keys
22/23, 74/75, 98/99: uses: actions/{checkout,setup-go,setup-python}@v7   # 6 lines, all @v7
```
```
?   backend/cmd/api          [no test files]
ok  backend/internal/config  0.595s
ok  backend/internal/health  0.993s
ok  backend/internal/store   1.546s
unit exit=0
```
```
# POSTGRES_PORT=5433 REDIS_PORT=6380, scratch backend/.env, removed afterwards
Container backend-postgres-1  Healthy
Container backend-redis-1  Healthy
--- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (0.07s)
--- PASS: TestIntegrationPetStatesRejectsASecondRowForTheSameUser (0.04s)
--- PASS: TestIntegrationRedisRoundTrip (0.00s)
3/3 integration tests ran and passed
integration exit=0
negative: skip detected          # env -u TEST_DATABASE_URL -u TEST_REDIS_URL
negative exit=1
```
```
Ran 30 tests in 0.178s / OK
validate exit=0
docker ps after teardown: wizardly_franklin, scio3-redis-1, scio3-mongo-1   # 6379 untouched
git status --short (worktree): clean
```

No executor gate failure: every documented command behaves as the summary claims.

## Quality

**Service-container wiring is correct.** The job runs directly on the runner VM, not in a container, so `services.*.ports: 5432:5432` / `6379:6379` publish to the VM's loopback and `localhost:5432` / `localhost:6379` in the job-level `env` are right. (The `localhost` that would be wrong is the container-job case, where the service is reached by its label — not this workflow.) Ubuntu runner images ship PostgreSQL and Redis but leave them stopped, so the published ports are free. `go-version-file: backend/go.mod` and `cache-dependency-path: backend/go.sum` are workspace-relative and unaffected by `defaults.run.working-directory: backend`, which applies only to `run:` steps — those paths are right for a module in `backend/`, and setup-go@v7 still declares both inputs.

**The skip gate is genuinely non-vacuous.** This was the thing most worth attacking, so I ran the exact `run:` block under `bash -e` (GitHub's default `bash -e {0}`) against a stubbed `go` in three failure shapes:

| shape | result |
| --- | --- |
| test binary fails to build (`go` exits 1 with compiler errors) | `exit 1` — `set -o pipefail` + errexit abort before any counting |
| `go` exits 0 printing only `[no tests to run]`, one `func TestIntegration*` in source | `::error::expected 1 passing TestIntegration* tests, saw 0`, `exit 1` |
| zero `TestIntegration*` in source and none run | `::error::expected 0 …`, `exit 1` — the `want -eq 0` arm fires |

Plus the real negative (`--- SKIP` with the `TEST_*` variables unset) reproduced above. There is no path I could find where the job goes green without three real `--- PASS` lines. Deriving `want` from source rather than hardcoding `3` is the right choice and makes the gate self-maintaining; `^--- PASS` correctly excludes indented subtests. This job also closes half of the open bug `integration-gate-tests-only-prove-the-skip-and-would-pass-if.md` — the positive direction now runs automatically on every run instead of needing a human to bring up the stack.

**The unit job's guard is near-tautological but harmless.** Nothing in the job sets those four variables, so the loop can only fire if someone later adds workflow- or job-level `env`. That is exactly the regression it is there to catch, and it costs one second. Fine.

**Triggers, permissions, concurrency.** `permissions: contents: read` is correct least privilege — nothing writes back, no packages, no OIDC. Concurrency and triggers are where the real weaknesses are, and they are the two medium bugs below: the push trigger excludes `harness/**`, and `pull_request` is dead on this repo because the harness account cannot open PRs, so CI only ever runs on `main` *after* `/harness merge` has already pushed it; and `cancel-in-progress: true` then cancels back-to-back merge runs on that same `main`. Neither makes CI green while the code is broken — the failure mode is an absent or cancelled check, which is visible — but together they mean the workflow does not yet do the job `AGENTS.md:23` claims for it.

**Readability, duplication, cost.** 105 lines for three jobs is proportionate; the "why each non-obvious line is there" reasoning lives in the plan rather than as comments in the YAML, which is a defensible split but means a maintainer reading only `ci.yml` will not know why `grep -c … || true` is there. The checkout+setup-go pair is duplicated between the two backend jobs — correct to keep them separate (the unit job's whole point is an environment with no service variables, which a matrix would muddy) and not worth a composite action at this size. Both backend jobs compile the module independently and both write the same setup-go cache key concurrently, which produces a harmless "unable to reserve cache" warning on one of them; total wall clock is a few minutes. `go build` before `go test` is redundant work but makes the run log readable, which the plan states as the intent. Pinning is floating majors on three first-party `actions/*` with a read-only token — acceptable; SHA pinning would be over-engineering at this permission level.

**Missing hardening:** no `timeout-minutes` on any job (default 360) and a floating `python-version: "3.x"` — bug 3.

**CODEMAP/AGENTS accuracy.** Both describe the triggers and the three jobs correctly. The CODEMAP paragraph has three smaller problems — it credits `backend-unit` with proving the suite "never drops tables" (the guard proves only that no service variable is exported), it points readers at `make up`, which lacks the `--wait` the verified procedure used, and it is a single ~210-word sentence in a file whose contract is scannable one-paragraph-per-module. Bug 4. `AGENTS.md:23`'s "A red check on a `harness/*` PR is a review blocker" is currently unenforceable for the reason in bug 1; I left both files as the executor wrote them rather than editing, because the fixes belong with the trigger fix in one amending plan.

## Bugs filed

1. `harness/ideas/_inbox/ci-never-runs-on-harness-branches-so-it-gates-nothing-before.md` — **medium.** `on.push.branches: [main]` plus a `pull_request` trigger that can never fire (no write access) plus `/harness merge` pushing `main` directly means CI first runs *after* the merge. Fix is one line: `branches: [main, 'harness/**']`.
2. `harness/ideas/_inbox/cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md` — **medium.** Unconditional `cancel-in-progress` cancels superseded runs on `main`, so back-to-back merges in one unattended `/harness run` can leave merge commits with no verdict.
3. `harness/ideas/_inbox/ci-jobs-have-no-timeout-minutes-and-the-harness-job-floats-p.md` — **low.** No `timeout-minutes` (360-minute default on a suite that finishes in seconds); `python-version: "3.x"` drifts from the 3.9.6 the tooling is developed against.
4. `harness/ideas/_inbox/codemap-ci-section-overstates-the-unit-job-and-prescribes-a-.md` — **low.** CODEMAP overstates what `backend-unit` proves, prescribes `make up` (no `--wait`) as the local repro, and is one run-on sentence.

None is a blocker. All four are visible-when-wrong or documentation-only; none can make CI report green on broken code, which is the failure mode that would have justified holding the branch.

## Verdict

**pass-with-bugs.** The workflow delivers the idea, the code matches the plan verbatim, and every piece of the executor's local evidence reproduced. The one risk local verification structurally could not cover — whether `@v7` is real for all three actions — I checked directly against the GitHub API and it holds; `actionlint` demonstrably cannot catch that class of error, so future CI plans should treat action-tag verification as a separate step rather than something the linter covers. What the workflow does *not* yet do is run before a merge, which is the whole premise of `AGENTS.md:23`; that is bug 1 and deserves to be the next thing fixed, but it is a widening of a trigger, not a reason to hold this branch.

Merge with `/harness merge harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md`.
