---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# stale-worktrees has no test for the legacy basename match on git's list or for skipping an ancestor of the cwd

## Why
`stale-worktrees` feeds `/harness prune`, which runs `git worktree remove` on every path it prints. So a wrong line removes someone's worktree. Two branches of the new `stale_worktrees()` (`tools/harness/cli.py`) have no test. (1) The legacy rule on git's list: a merged plan with no `branch:` matched by basename, restricted to entries whose branch is `None` or unknown to any plan. It is only exercised through the no-git fallback, never through `stale_worktrees(plans, entries, cwd)`. (2) The ancestor skip `cwd.startswith(norm + "/")`: the current-checkout test passes `cwd` equal to the entry, never a subdirectory of it. A regression in either would print (and prune) the wrong path, and CI would stay green.

## Expected output
Two more cases in `tools/harness/tests/test_cli.py`, driven through `stale_worktrees()` or `cmd_stale_worktrees` with a porcelain fixture:
- a merged no-`branch` plan whose `worktree:` basename matches (a) a detached entry → printed, and (b) an entry on a branch another plan records → not printed;
- `cwd` = `<entry>/backend` for a merged-branch entry → not printed.

## Evidence
- Plan: `harness/plans/2026-09-27-executor-worktree-nested-under-claude-worktrees-while-plan-f.md` (Review Focus 1-3), branch `harness/2026-09-27-low-executor-worktree-nested-under-claude-worktrees-while-plan-f`.
- `tools/harness/cli.py` `stale_worktrees`: the `legacy_names`/`known_branches` clause and the `cwd.startswith(norm + "/")` guard. Tests: `test_stale_worktrees_finds_merged_branches_anywhere_git_lists_them` (the merged plan has a branch, so `legacy_names` is empty), `test_stale_worktrees_skips_the_main_worktree_and_the_current_checkout` (`cwd` == entry), `test_stale_worktrees_falls_back_to_the_worktrees_dir_without_git` (`_git_worktree_list` → `None`, so the `stale_worktrees` function is not called).
