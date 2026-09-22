---
name: harness-orchestrate
description: Drive the harness end to end — status, one bounded autonomous run, human merge, worktree prune. Use for /harness and for scheduled unattended runs.
---

# harness-orchestrate

Subcommands: `status`, `run [--auto-approve] [--stages ideate,evaluate,execute,review]`, `merge <plan>`, `prune`.

## status
`python3 tools/harness/cli.py state`; then `python3 tools/harness/cli.py stale-worktrees` and append a "Stale worktrees" list. Print.

## run
Idempotent; safe to call on a schedule. `LOG=harness/runs/$(date +%Y%m%dT%H%M%S).log`. Every step appends one line to `$LOG`. Stages default to all four; `--stages` limits them.

1. `python3 tools/harness/cli.py validate` → if it fails, log the invalid files and **stop** (never build on broken state).
2. **ideate** if enabled and (`ls harness/ideas/_inbox/*.md` is non-empty **or** `next --stage evaluate` is empty): spawn the ideator role (features mode, count 5). Log the run path.
3. **evaluate** if enabled and `next --stage evaluate --all` is non-empty: spawn the evaluator role on all of them. Log verdicts.
4. **auto-approve** if `--auto-approve`: for each `draft` plan whose idea is `type: mvp-slice` **or** (`type: bug` and `priority: high`): `cli.py set <plan> status=approved`. Log each. Never auto-approve features.
5. **execute** if enabled and `next --stage execute` is non-empty: spawn the executor role on exactly that one plan. Log status, branch, PR.
6. **review** if enabled: for each path in `next --stage review --all`: spawn the reviewer role. Log verdicts and bugs filed.
7. `cli.py state`; append "Awaiting human: N draft plans, M passed reviews to merge" to `$LOG`; commit `harness/` with `harness: orchestrator run $(basename $LOG .log)`.
8. Print the log.

## merge <plan>  (human-invoked only)
Preconditions: plan `status=done`, latest review verdict `pass` or `pass-with-bugs`, `merged=false`. Refuse otherwise.
```
git checkout main && git merge --no-ff <branch> -m "Merge <branch>: <idea title>" && git push origin main
git worktree remove <worktree> && git branch -d <branch> && git push origin --delete <branch>
python3 tools/harness/cli.py set <plan> merged=true
```
If a `pr` exists, `gh pr merge <pr> --merge` may replace the local merge — pick one, never both. Commit `harness/`.

## prune
For each path from `cli.py stale-worktrees`: `git worktree remove <path>`. Print what was removed.
