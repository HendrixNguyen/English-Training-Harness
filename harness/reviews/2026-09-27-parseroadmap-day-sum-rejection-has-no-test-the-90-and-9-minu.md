---
plan: harness/plans/2026-09-26-parseroadmap-day-sum-rejection-has-no-test-the-90-and-9-minu.md
verdict: pass
bugs: []
---
# Review — airouter amend: `TestParseRoadmapRejects` proves the day-sum rule — rows only the day budget can reject, keyed by reason

**Plan:** `harness/plans/2026-09-26-parseroadmap-day-sum-rejection-has-no-test-the-90-and-9-minu.md`
**Branch/worktree:** amend on `harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut`, commits `42f8fd6..91f4a58` (`6e378a3`, `24a3f82`, `91f4a58`). Reviewed in a fresh detached worktree (`.worktrees/rv-ai-2`, removed afterwards).
**Diff:** `git diff 42f8fd6 91f4a58 --stat`: 2 files, +71/−35 (`backend/internal/airouter/roadmap_test.go`, `harness/CODEMAP.md`). Test-only, as planned; `roadmap.go` is untouched.

_Reviewer, 2026-09-27 (unattended daily-review, id rv-ai)._

## Plan vs idea
The blocker idea asked that deleting the day-sum `if` must fail `go test ./internal/airouter`. It now does. The two folded low ideas are closed as well: every row asserts its reason, and the week-2 provenance check proves provenance through unique per-module titles.

## Code vs plan
- Task 1 was followed. Every row has a `want` taken from the branch's real `invalid(...)` text; the four pre-decode rows got stable substrings, which is the justified choice. The two misnamed rows were renamed to "(task band)", and the 15- and 45-minute day rows were added.
- Task 1b was followed with a justified adaptation. `Exercise` has no `Title`, so the test unmarshals `ContentJSON`. The executor correctly reported that the plan's "swap modules in the test" mutation is tautological and used the other offered mutation instead.
- Task 2 (CODEMAP clause) is present.

Re-run in the review worktree (`backend/`):
```
go test -timeout 60s ./internal/airouter -count=1 -v -run TestParseRoadmap | grep -c -- '--- PASS'  -> 37 (0 FAIL; includes parent tests)
gofmt -l internal/airouter -> (empty);  go vet ./... -> clean
Mutation 1 (roadmap.go:160 `if false && (dayMinutes < minDayMinutes || ...)`):
  --- FAIL: TestParseRoadmapRejects/three_five-minute_tasks_(15-minute_day)
  --- FAIL: TestParseRoadmapRejects/three_fifteen-minute_tasks_(45-minute_day)
Mutation 2 (Exercises() reads r.Modules[len-1-mi]):
  roadmap_test.go:192: exercise[21] = day 8 title "w3-d1-vocabulary", want day 8 title "w2-d1-vocabulary" (the module declaring week 2)
Both reverted; git status clean.
go test -timeout 300s ./... -count=1 -> all ok
gh run list --branch <branch> --limit 1 -> 91f4a58 completed success
```
Everything reproduced.

## Quality
- The table is now honest. The `"unknown task type" → "has type"` substring is loose but unambiguous in `roadmap.go`. The comments explain the pre-decode rows accurately.
- `onboarding/fakes_test.go` still uses `tt + " task"` titles. That is a separate fixture, as the executor noted, and correct to leave alone.
- **Merge note.** This branch conflicts textually with the typed-content branch in `roadmap.go`/`roadmap_test.go` (the `validRoadmapJSON` task line: this branch changes `Title`, typed-content changes `Content`; both rejects tables grew). The integration merge must keep both. It conflicts with `origin/main` and every other branch only in `harness/CODEMAP.md`.
- CODEMAP is accurate.

## Bugs filed
None.

## Verdict
**pass.** The amend delivers the blocker's fix and both folded findings. Mutation-proved and CI green. The amended plan's branch has no remaining unresolved blockers (`cli.py blockers --plan harness/plans/2026-09-24-parseroadmap-…` exit 0).
