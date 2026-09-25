---
name: harness-orchestrate
description: Drive the harness end to end — status, one bounded autonomous run, the daily PR, human merge, worktree prune. Use for /harness and for scheduled unattended runs.
---

# harness-orchestrate

Subcommands: `status`, `run [--stages ideate,evaluate,execute,review]`, `daily-pr`, `merge <plan>`, `prune`.

## status
`python3 tools/harness/cli.py state`; then `python3 tools/harness/cli.py stale-worktrees` and append a "Stale worktrees" list. Print.

## run
Idempotent; safe to call on a schedule. `LOG=harness/runs/$(date +%Y%m%dT%H%M%S).log`. Every step appends one line to `$LOG`. Stages default to all four; `--stages` limits them.

1. `python3 tools/harness/cli.py validate` → if it fails, log the invalid files and **stop** (never build on broken state).
2. **ideate** if enabled and there are no `proposed` ideas in any run folder (inbox bugs do not count — they are the evaluator's, not the ideator's): spawn the ideator role (features mode, count 5). Log the run path.
3. **blockers** — run `python3 tools/harness/cli.py blockers`. Any listed bug is holding up an unmerged branch: spawn the evaluator on those first, ahead of the ordinary queue, and log them.
4. **evaluate** if enabled and `next --stage evaluate --all` is non-empty: spawn the evaluator role on all of them — features and inbox bugs together; the evaluator ranks them as two lists with two caps (≤5 bug plans and ≤5 feature plans per day, aged `selected` features before new ideas — owner, 2026-09-25). Log verdicts.
4. **auto-approve** (always on; owner, 2026-09-24): the evaluator approves eligible plans as it writes them; this step catches any eligible `draft` left over — every `type: bug` plan (any priority, blockers included), every `type: mvp-slice` plan, and every `type: feature` plan whose idea is `priority: high`: `cli.py set <plan> status=approved`. Log each. Medium/low feature plans stay `draft` for `/approve`. (`--auto-approve` is accepted and ignored for old schedules.)
5. **execute** if enabled and `next --stage execute` is non-empty: spawn the executor role on exactly that one plan. Log status, branch, PR.
6. **review** if enabled: for each path in `next --stage review --all`: spawn the reviewer role. Log verdicts and bugs filed.
7. `cli.py state`; append "Awaiting human: N draft plans, M passed reviews to merge" to `$LOG`; commit `harness/` with `harness: orchestrator run $(basename $LOG .log)`.
8. Print the log.

## daily-pr
The owner takes **one PR per day, not one per plan** (AGENTS.md). Executors only push branches; this is what turns a day's work into something to review.

Preconditions: every plan whose `branch` you are about to include is `status=done`, has a review whose verdict is `pass` or `pass-with-bugs`, has `merged=false`, and `cli.py blockers --plan <plan>` exits 0. A plan that is `done` but unreviewed, or whose review is `fail`, waits for tomorrow — say so rather than sweeping it in.

```
DATE=$(date +%Y-%m-%d); DAILY=harness/daily-$DATE
git fetch origin main && git checkout -b $DAILY origin/main
# then, oldest plan first:
git merge --no-ff <branch> -m "Merge <branch>: <idea title>"
```
Resolve conflicts by hand. **`harness/CODEMAP.md` conflicts on almost every branch** — every slice edits it and the bullets sit adjacent, so git picks a side and both sides look valid. Read both and keep what is true of the merged tree; never accept the automatic resolution there.

After each merge run the suite for the layers touched (`cd backend && go build ./... && go test ./...`, and/or `cd frontend && npm run lint && npm run typecheck && npm run test && npm run build`). A daily branch that does not build is worse than three branches that do — stop and report rather than pushing it.

Then `git push -u origin $DAILY` and wait for CI (`gh run list --branch $DAILY`, `gh run watch <id> --exit-status`). Open one PR:
```
gh pr create --base main --head $DAILY \
  --title "$DATE [<highest priority among included>] Daily: <n> fixes" \
  --body-file <tmpfile> --label harness
```
Body: one section per included plan — idea title, what changed, the review verdict and the path to the review file — then the CI run URL, then `🤖 Generated with [Claude Code](https://claude.com/claude-code)`. Record it on every included plan: `cli.py set <plan> pr=<url>`.

The owner merges that PR. Afterwards, for each included plan: `cli.py set <plan> merged=true`, then `/harness prune`.

## merge <plan>  (human-invoked only)
Preconditions: plan `status=done`, latest review verdict `pass` or `pass-with-bugs`, `merged=false`, and `python3 tools/harness/cli.py blockers --plan <plan>` exits 0. Refuse otherwise — `cli.py set … merged=true` enforces the blocker check independently, so a merge that skips it cannot be recorded.
```
git checkout main && git merge --no-ff <branch> -m "Merge <branch>: <idea title>" && git push origin main
git worktree remove <worktree> && git branch -d <branch> && git push origin --delete <branch>
python3 tools/harness/cli.py set <plan> merged=true
```
If `git remote get-url origin` fails, skip both `git push` commands and say so. If a `pr` exists, `gh pr merge <pr> --merge` may replace the local merge — pick one, never both. Commit `harness/`.

## prune
For each path from `cli.py stale-worktrees`: `git worktree remove <path>`. Print what was removed.
