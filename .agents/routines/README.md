# Routines — the unattended daily cadence

Each file here is one scheduled, unattended run of the harness. The files are the **source of truth** for the schedule: a tool's own scheduler (Claude Code scheduled tasks, a cloud routine, a Codex or Gemini cron, plain `cron` + CLI) is only an adapter whose prompt says "follow `.agents/routines/<name>.md` exactly". Never keep a longer prompt in the adapter than "read and follow this file" — the prompt drifts otherwise.

Frontmatter per routine: `name`, `schedule` (5-field cron, owner's local time, Asia/Saigon) and `schedule_utc`, `role` (the `.agents/roles/` file to adopt), `skills` (the `.agents/skills/` to load), `writes` (what the run may change), `pr_title` (the one bookkeeping PR it opens). The body is the prompt, written for any agent — "spawn the evaluator role", never a tool-specific agent name.

## The day

| Local | UTC | Routine | Produces |
|---|---|---|---|
| 02:00 | 19:00 (prev day) | [daily-ideate](daily-ideate.md) | a new run folder of `proposed` ideas |
| 06:00 | 23:00 (prev day) | [daily-decide](daily-decide.md) | ≤5 plans for the day, bugs / mvp-slices / high features already `approved` |
| 10:00 | 03:00 | [daily-bugfix-execute](daily-bugfix-execute.md) | `type: bug` plans → `done` on pushed `harness/*` branches |
| 14:00 | 07:00 | [daily-feature-execute](daily-feature-execute.md) | `type: feature` / `mvp-slice` plans → `done` on pushed `harness/*` branches |
| 20:00 | 13:00 | [daily-review](daily-review.md) | reviews, inbox bugs for tomorrow, the day's single code PR |

## Rules shared by every routine

- **Sync first.** Abort if on `main`. If the only dirty file is a regenerated `harness/STATE.md`, restore it. `git fetch origin main && git merge origin/main --no-edit`. Only `main` / `origin/main` is ever merged into a routine's branch — never another feature or `harness/*` branch, not even to pick up a plan.
- **Merge green harness PRs before working.** `gh pr list --state open`; a PR whose every changed file is under `harness/` (or `.agents/`) is a harness-artifact PR. If its checks are all green and it is mergeable, `gh pr merge <n> --merge`, then sync again. Red or refused: report it, never work around it.
- **The only channel between runs is `origin/main` plus open harness PRs.** A routine starts a fresh session and sees nothing from the previous one except what reached GitHub. Plans committed on a local branch, sitting in a worktree, or uncommitted are invisible to the next run — that is how the 2026-09-24 decide run approved six plans and the 10:00 execute run still reported "nothing decided". Every run therefore ends with its PR open, and a report that says "nothing is pushed" is a failed run.
- **Recover stranded work before calling the queue empty.** After the sync, check for a decide branch that was committed but never pushed: `git for-each-ref --format='%(refname:short)' 'refs/heads/harness/plan-*'`. For each one not on origin (`git ls-remote --heads origin <b>` prints nothing) whose commits touch only `harness/` (`git diff --quiet $(git merge-base origin/main <b>) <b> -- . ':!harness'`), push it (`git push -u origin <b>`), open its PR (`gh pr create --base main --head <b> --title "harness: recovered <b>"`) and treat it like any other harness PR in the next rule. Only `harness/plan-*` branches are recovered automatically — they are the decide routine's own convention. Any other local branch that is ahead of `origin/main` in `harness/` may belong to a live session: list it in the report (`git for-each-ref --format='%(refname:short) %(committerdate:relative)' 'refs/heads/claude/*'`, then `git log --oneline origin/main..<b> -- harness/`) and never push or commit it. Uncommitted changes in another session's worktree are that session's too — never touch them, list the worktree instead.
- **One bookkeeping PR per run.** `python3 tools/harness/cli.py validate && python3 tools/harness/cli.py state`, commit `harness/` on the run's own branch, push, open one PR with the routine's `pr_title`, then prove it: `gh pr view --json url -q .url` must print the PR URL, and that URL goes in the report. Never push `main`, never merge code branches, never open per-plan PRs — the owner merges.
- **Locks keep runs apart.** The two execute routines overlap in scope only by type; each takes `python3 tools/harness/cli.py lock <plan>` before spawning an executor, skips (and reports) a plan another run holds, and unlocks when its executor finishes.
- **Time-boxed.** Each routine states its budget; when it runs out, record what is unfinished (`failed` with the reproduction, or `draft`) and report — never leave a plan `executing`.
- **Report at the end**: what was merged or refused, what changed state, and everything waiting on the owner (medium/low feature drafts needing `/approve`, failures, refused permissions, locked plans skipped).

## Adapters

- Claude Code: `~/.claude/scheduled-tasks/<id>/SKILL.md`, cron in local time; prompt = "Follow `.agents/routines/<name>.md` in this repository exactly."
- Cloud routine / any other scheduler: same prompt, cron in `schedule_utc`.
