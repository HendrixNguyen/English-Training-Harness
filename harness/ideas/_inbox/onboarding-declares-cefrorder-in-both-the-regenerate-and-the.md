---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-26-a-session-a-learner-wants-to-finish-level-true-content-do-to.md
---
# onboarding declares cefrOrder in both the regenerate and the level-floor branches so the daily merge does not compile

## Why
**Blocker for the level-true-content branch in today's daily PR.** Both branches add a package-level `var cefrOrder = []string{"A1","A2","B1","B2","C1","C2"}` to package `onboarding`: the regenerate branch (B2) in `service.go`, and the level-floor branch (F3) in `grade.go`. They are in different files, so git merges them without a textual conflict (`git merge-tree` reports only `harness/CODEMAP.md`). The merged tree then fails to compile, and CI would go red on the daily integration branch. F3's plan anticipated this ("if B2's `cefrOrder` is on the branch, reuse — one definition"), but the two branches were cut the same day from the same base, so neither could see the other. The two branches also carry near-duplicate index helpers (`indexOf` in service.go, `cefrIndex` in grade.go).

## Expected output
One `cefrOrder` and one index helper in package `onboarding`, used by `stepAllowed`, `GradeFloor` and `maxLevel`. Fix it on the level-true-content branch (this `blocks` it): merge the regenerate branch into it once that branch is on `main`, or rename F3's copy and have `stepAllowed` reuse `cefrIndex` after the merge. The acceptance check is `git merge` of both branches onto `origin/main` followed by `go build ./... && go test ./internal/onboarding` green.

## Evidence
- Plans: `harness/plans/2026-09-26-a-session-a-learner-wants-to-finish-level-true-content-do-to.md` (grade.go) and `harness/plans/2026-09-26-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a.md` (service.go).
- Reproduced in a scratch detached worktree at `origin/main`: merged `origin/harness/2026-09-26-high-59-of-84-…` then `origin/harness/2026-09-26-high-a-session-a-learner-wants-…` (only CODEMAP conflicted). Then `cd backend && go build ./...` gave:
```
internal/onboarding/service.go:30:5: cefrOrder redeclared in this block
	internal/onboarding/grade.go:56:5: other declaration of cefrOrder
```
