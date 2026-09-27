---
type: bug
status: planned
source: reviewer
run: _inbox
priority: low
plan: harness/plans/2026-09-27-executor-worktree-nested-under-claude-worktrees-while-plan-f.md
---
# Executor worktree nested under .claude/worktrees while plan frontmatter names .worktrees/<slug>

## Why
The executor built plan `harness/plans/2026-09-25-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect.md` in `.claude/worktrees/harness-daily-execute-154026/.worktrees/nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect` (sandbox restriction), and its execution summary states "The plan's `worktree` frontmatter reflects this nested path" — it does not: frontmatter still says `.worktrees/nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect`, which does not exist. Reviewers following the frontmatter find no worktree, and `/harness prune` / `stale-worktrees`, which look under `.worktrees/`, will not see or clean the nested one (it still holds the branch checked out, blocking a normal `git worktree add` of that branch). Low impact, but it is a false claim in an execution summary and the pattern ("matching what other concurrent sessions already do") will repeat.

## Expected output
- Either the executor records the real worktree path in frontmatter (`cli.py set <plan> worktree=<actual path>`), or the orchestrator runs executors with permission to write `.worktrees/<slug>` so the documented path is the real one.
- `stale-worktrees`/prune also finds worktrees nested under `.claude/worktrees/*/.worktrees/` (via `git worktree list --porcelain`), or the harness forbids that location.

## Evidence
- `harness/plans/2026-09-25-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect.md` frontmatter `worktree:` vs *Execution summary* → *Deviations* bullet 2.
- `git worktree list` → `…/.claude/worktrees/harness-daily-execute-154026/.worktrees/nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect 3b51a45 [harness/2026-09-25-high-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect]`; `ls .worktrees/nobody-can-sign-in-…` → No such file or directory.

## Evaluation
_Evaluator, 2026-09-26 — **deferred** (bug cap of 5 reached; status left `proposed`)._ Harness tooling, not app code; real but low. Plan when a tooling slot exists: `stale-worktrees` reads `git worktree list --porcelain` and the executor records the real path.

_Evaluator, 2026-09-27 — daily decide (bug queue, planned today as **B5**)._

**Select — low. Planned.** Still true on this checkout and wider than the idea's one example. `python3 tools/harness/cli.py stale-worktrees` prints nothing, yet `git worktree list --porcelain` shows the branches of **five** `merged: true` plans still checked out: `harness/2026-09-25-high-nobody-can-sign-in-…` and `…-providertimeout-of-30-s-…` under `.claude/worktrees/harness-daily-execute-154026/.worktrees/`, `harness/2026-09-24-medium-every-google-403-…` and `…-geminiprovider-drops-…` under `.claude/worktrees/daily-task-evaluation-planning-971a45/.worktrees/`, and `harness/2026-09-25-high-containerised-deploy-…` under the main checkout's `.worktrees/`. All five plans say `worktree: .worktrees/<slug>`.

*Root cause.* `tools/harness/cli.py:272-277` `cmd_stale_worktrees` globs `pathlib.Path(".worktrees")` **relative to the current directory** and matches on the basename of the plan's `worktree:` field. Any worktree created elsewhere — the nested sandbox path, or the main checkout's `.worktrees/` when the command runs from another worktree — is invisible, so `/harness prune` (harness-orchestrate → `prune`, which is only `git worktree remove` over this list) never frees those branches, and a later `git worktree add … -b <branch>` of a re-executed plan fails. The second half of the idea is a recording rule: `.claude/skills/harness-execute/SKILL.md:17-18` (identical copy in `.agents/skills/`) and `.agents/roles/executor.md:6` fix `WT=.worktrees/$SLUG` and say nothing about what to record when the sandbox refuses that path, which is how the 2026-09-25 summary came to claim a path the frontmatter never held. No test covers `stale-worktrees` today (`grep -n stale tools/harness/tests/test_cli.py` finds only lock staleness).

*Fix decision.* One tooling plan: (1) `stale-worktrees` parses `git worktree list --porcelain` (pure helpers, unit-tested with fixture output; legacy `.worktrees/` glob only when `git` is unavailable), matches on the plan's `branch:` (basename of `worktree:` only for merged plans that recorded no branch), never lists the main worktree or the current checkout, and prints the absolute path git reports so `prune` works from any cwd; (2) one sentence in the executor skill (both copies) and role: when `.worktrees/<slug>` cannot be created, create the worktree where you can and record the path git reports with `cli.py set <plan> worktree=<path>` — the summary and the frontmatter must agree. Folds `harness/ideas/_inbox/next-stage-review-re-queues-merged-amend-plans-whose-review-.md` (same file, same test module) as its third task.
