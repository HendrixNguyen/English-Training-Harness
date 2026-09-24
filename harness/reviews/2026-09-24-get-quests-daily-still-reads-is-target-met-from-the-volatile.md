---
plan: harness/plans/2026-09-24-get-quests-daily-still-reads-is-target-met-from-the-volatile.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/revive-s-absolute-save-erases-a-concurrent-ontargetmet-s-20-.md, harness/ideas/_inbox/plan-branch-edits-harness-plans-and-misses-its-own-verificat.md, harness/ideas/_inbox/durable-flag-and-live-row-success-tests-leave-updated-at-and.md]
---
# Review — quests + pet: the daily screen reads the durable flag, and every verdict write is one conditional SQL statement

**Plan:** `harness/plans/2026-09-24-get-quests-daily-still-reads-is-target-met-from-the-volatile.md`
**Branch/worktree:** `harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile` / `.worktrees/get-quests-daily-still-reads-is-target-met-from-the-volatile`
**Diff:** `git diff origin/main...harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile --stat`

## Plan vs idea
The plan delivers all six ideas it folds in. Each idea's Expected output checks out against the branch:

- **Head (`get-quests-daily-…`).** `ProgressRepo.TargetMet` runs the exact `SELECT COALESCE(is_target_met, FALSE)`, and no row means false. `Daily` returns `total >= TargetSeconds || flagged`. `AccumulatedSeconds` stays live, as the idea asked, so the §6.2 shape is unchanged. The unit test and the integration `Daily`-after-`DEL` are both present. Reproduced live over HTTP (below).
- **`ontargetmet-s-read-then-write-…`.** `saveTargetMetSQL` now computes health, streak and stage on the live row, keeps the predicate unchanged, and holds to `ApplyTargetMet`. Section 4b pins miss-then-met at 90. The parent plan's residual-race note is annotated as resolved.
- **`marktargetmet-ignores-rowsaffected-…`**, **`fakeprogressrepo-upsert-…`**, **`pgrepo-save-resets-…`**: delivered as specified.
- **`testtwoconcurrentsweeps…`.** The barrier sits in `SweepCandidates` (after the read, mutex released) rather than the idea's suggested `beforeWrite` inside `PenaliseMiss`. The effect is the same: both sweeps hold identical stale lists before either writes. I measured the difference it makes (below).

## Code vs plan
Diff base `origin/main` (fetched; the branch is 2 harness-only commits behind and merges cleanly). There are 8 executor commits plus 1 reviewer CODEMAP commit (`8d43b33`). CI on the executor's HEAD `db1f5e8` is green: run 35960183886, with `backend-unit`, `backend-integration`, `harness-tooling` and `frontend` all passing. My push re-triggers CI for `8d43b33`, which is a CODEMAP-only change.

| Task | Result |
| --- | --- |
| 1 `Daily` reads the durable flag | followed verbatim |
| 2 `MarkTargetMet` → `ErrNoProgressRow` | followed. The caller in `RecordProgress` only logs it, which is correct because it's unreachable after `Upsert` |
| 3 monotonic-minutes unit test | followed. The 8th commit `db1f5e8` is a gofmt-only fix to the plan's own snippet. Justified |
| 4 `SaveTargetMet(ctx, user, now, D)` in SQL | followed verbatim (SQL, method, fake, `OnTargetMet`, comments) |
| 5 `Save` `GREATEST` on the marker | followed |
| 6 barrier | followed |
| 7 CODEMAP + parent-plan note | followed, but the parent-plan edit breaks execute-skill step 9 (see Bugs, #2) |

**Re-run evidence (worktree, isolated stack `rev-quests`, PG 55446 / Redis 56396 / API 18095):**
```
gofmt -l ./internal/quests ./internal/pet   -> handler_test.go, repo.go (both pre-existing; origin/main's repo.go is gofmt-dirty on the same Exercise.TaskType comment)
go build ./... && go vet ./... && go test ./... -count=1   -> ok x10
go test ./internal/quests/ -run 'DailyReportsTheDurableFlag|ALostCounterNeverLowers|FakeMarkTargetMetRefuses' -v -> PASS x3
go test ./internal/pet/ -run 'OnTargetMet|FakeSaveKeepsAMarker|TwoConcurrentSweeps' -v -> PASS x7
TestTwoConcurrentSweepsPenaliseOnce  -count=300 -> ok; -count=20 -race -> ok; -count=100 -race -> ok
go test ./internal/pet/ ./internal/quests/ -race -count=1 -> ok, ok
go test ./internal/quests/ ./internal/pet/ -run Integration -p 1 -v -> PASS DailyAndProgress, EnsureCreatesExactlyOne, VerdictWritesAreConditional
make test-integration (TEST_* exported) -> ok x10, no SKIP
cli.py validate -> exit 0
```

**Runtime proof (the real API binary over HTTP; the user, roadmap and exercises were seeded by SQL and the HS256 JWT was minted by hand, so nothing landed in the worktree):**
```
GET  /quests/daily (fresh)          -> day_number 1, accumulated_seconds 0, is_target_met false, 3 tasks
POST /quests/progress 1800s         -> {"daily_seconds_spent":1800,"daily_minutes_spent":30,"is_target_met":true,"pet_health":100,"streak_count":1}
DEL daily:accumulated:<uid>:2026-09-24 -> 1
GET  /quests/daily                  -> accumulated_seconds 0, is_target_met TRUE      <- head defect fixed
GET  /pet/status                    -> sprout, health 100, streak 1
daily_progress                      -> 2026-09-24 | 30 | t
POST /quests/progress 60s (lost ctr) -> is_target_met true, streak 1 (no re-bump); pet row 100|1|sprout|2026-09-24
```

**Verification greps: the executor's two "cosmetic" mismatches are confirmed.** `IsTargetMet:` gives 2 lines. Only `Daily`'s (`service.go:255`) contains `||` literally. `RecordProgress:154` uses `targetMet` from `service.go:121` (`total >= TargetSeconds || alreadyMet`), which was already on main. Same rule, different textual form. `pet_states` in `internal/quests/` gives 6 hits, all comments in `pet.go` plus one test error string, and `git grep` on `origin/main` also gives 6. No SQL crosses the boundary. `daily_progress` in `internal/pet/` gives 0. Both plan expectations were written wrong; the code is right. The rest match: `health_points = $2` gives 1 hit (saveSQL), `GREATEST(last_target_met_date` gives 1, and `ApplyTargetMet(` in `pet/service.go` gives none. Commit count is 8 (7 plus the justified gofmt commit), but **none carries the Co-Authored-By trailer** the plan's Verification requires, and the summary didn't mention it (Bug #2).

**Mutation checks (run on a scratch copy of `backend/`, never the worktree; each mutation must compile):**
| Mutation | Result |
| --- | --- |
| `Daily`: `\|\| (flagged && false)` | unit `DailyReportsTheDurableFlag…` + integration `:244` FAIL |
| fake `Upsert` without `max` | `ALostCounterNeverLowers…` FAIL (`minutes:1`) |
| `MarkTargetMet` RowsAffected check disabled | integration `:76` FAIL |
| `saveTargetMetSQL` absolute health | integration 4b `:240` FAIL |
| `saveTargetMetSQL` predicate dropped | integration `:159` FAIL |
| `saveTargetMetSQL` sapling threshold 3→4 | mirror loop `:258` FAIL |
| `saveSQL` marker absolute | integration section 5 `:271` FAIL |
| fake `Save` marker absolute | `FakeSaveKeepsAMarker…` FAIL |
| `Sweep` unconditional `penalised++` | 30/30 FAIL |
| fake `PenaliseMiss` without the judged guard, **with** barrier | **30/30 and 200/200 FAIL** |
| same, **barrier removed** (i.e. before this plan) | 22/30 and 152/200 FAIL |
| `saveTargetMetSQL` `updated_at = updated_at` | **passes the whole suite** (Bug #3) |
| `saveTargetMetSQL` `last_practiced_at = COALESCE(last_practiced_at, $5)` | **passes the whole suite** (Bug #3) |

The barrier does what it claims. Before, the guard mutation escaped about a quarter of runs; now it is caught every time. With the mutation absent, the test passes 300/300, plus 120 runs under `-race`.

## Quality
- **SQL arithmetic in `SaveTargetMet` is correct:**
  - *Boundaries.* `LEAST($4, COALESCE(health_points,$4)+$3)` matches `min(100, h+20)`, and a NULL health reads as 100, like `stateColumns`. The streak `COALESCE(...,0)+1` thresholds 3/7/14 are `>=` on the new streak, matching `StageFor`. The wilted branch is correctly omitted: the DDL `CHECK (health_points BETWEEN 0 AND 100)` guarantees health' ≥ 20. The four mirror cases (95/2, 40/6, 0/0 wilted, 100/13) cover the cap, sapling, flowering, revive-from-0 and fruitful edges.
  - *Concurrency (live Postgres, two sessions, one holding the row lock for 2 s).* Held miss then blocked met gives 90/1. Held met then blocked miss gives 70/0. Two mets for the same date give `UPDATE 1` then `UPDATE 0`, ending at 100/6. Under READ COMMITTED the blocked UPDATE re-evaluates both its WHERE and its SET against the committed row, so every order is serialisable and nothing is lost. Param types infer cleanly: `$3`/`$4` are int via `health_points`, `$2` is date, `$5` is timestamptz.
- **Boundaries hold.** quests never touches `pet_states` and pet never touches `daily_progress`. `cmd/api/main.go` is untouched.
- **Residual lost update (Bug #1).** Revive's `Save` still writes health, streak and stage absolutely from an `Ensure` pre-image. A concurrent `OnTargetMet` inside that window loses its +20 and streak day for good, because the marker survives and quests has already flagged the day.
- **Performance.** `Daily` now does one more indexed point read on the `(user_id, date)` unique key. Fine.
- **Parent-plan edit (known issue 1): it does not need reverting before merge.** Filed as low, not a blocker. It is body-only. ROOT's copy is identical to `origin/main`. `git merge-tree` is clean against both `origin/main` and ROOT `HEAD`. `/harness merge` only rewrites that file's frontmatter, about 1980 lines away. The root cause is the evaluator's Task 7 Step 3, which contradicts execute-skill step 9. The risk is only that ROOT edits the same Notes bullet before the daily PR lands.
- **Docs.** The `Repo` type comment's new clause reads awkwardly ("never an error — and compute on the live row"), and it describes the `Save` exception without saying it is still unsafe. Bug #1 covers the fix. I corrected CODEMAP on the branch: the quests `is_target_met` note had a nested backtick that broke the code span, and the inserted `Save` sentence garbled the revive clause. It now also says revive's health and streak are still pre-image writes. Commit `8d43b33`, pushed.
- **Not filed (deliberate per design decision 1).** After a counter loss, `POST /quests/progress` answers `daily_minutes_spent: 1` while the durable row holds 30, and `GET /quests/daily` shows `accumulated_seconds: 0` with `is_target_met: true`. The progress bar resets while the day stays met. The idea explicitly chose this.

## Bugs filed
1. `harness/ideas/_inbox/revive-s-absolute-save-erases-a-concurrent-ontargetmet-s-20-.md` (low): Revive's pre-image `Save` erases a concurrent success's +20 and streak.
2. `harness/ideas/_inbox/plan-branch-edits-harness-plans-and-misses-its-own-verificat.md` (low): the branch edits `harness/plans/*` against execute step 9, which the plan ordered. Two Verification greps were authored wrong. The trailer expectation went unmet and unreported.
3. `harness/ideas/_inbox/durable-flag-and-live-row-success-tests-leave-updated-at-and.md` (low): `updated_at` and `last_practiced_at` are unasserted in the mirror loop (two mutations survive), the `Daily` flag-read error path is untested, and one test discards an error.

No blockers.

## Verdict
**pass-with-bugs.** The plan and all six ideas are delivered, and every piece of executor evidence reproduces: build, full suite, integration, runtime proof and CI. The new SQL is correct at the boundaries and under real row-lock contention, and the barrier makes the sweep test deterministic. All three bugs are low and none blocks the merge. Merge with `/harness merge harness/plans/2026-09-24-get-quests-daily-still-reads-is-target-met-from-the-volatile.md`, via the daily PR.
