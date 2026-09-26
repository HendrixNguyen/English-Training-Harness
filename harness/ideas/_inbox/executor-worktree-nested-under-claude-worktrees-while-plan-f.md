---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
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
