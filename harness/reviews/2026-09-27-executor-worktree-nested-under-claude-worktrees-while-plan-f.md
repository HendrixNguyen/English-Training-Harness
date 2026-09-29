---
plan: harness/plans/2026-09-27-executor-worktree-nested-under-claude-worktrees-while-plan-f.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/stale-worktrees-has-no-test-for-the-legacy-basename-match-on.md]
---
# Review — `stale-worktrees` reads `git worktree list`, the executor records the real worktree path, and amend re-reviews count for the amended plan

**Plan:** `harness/plans/2026-09-27-executor-worktree-nested-under-claude-worktrees-while-plan-f.md`
**Branch/worktree:** `harness/2026-09-27-low-executor-worktree-nested-under-claude-worktrees-while-plan-f` / `.worktrees/executor-worktree-nested-under-claude-worktrees-while-plan-f`
**Diff:** `git diff main...harness/2026-09-27-low-executor-worktree-nested-under-claude-worktrees-while-plan-f --stat`

## Plan vs idea
Delivered for both ideas. (1) Primary: `stale-worktrees` now reads `git worktree list --porcelain`, so it finds merged-branch worktrees nested under `.claude/worktrees/*/.worktrees/` and in the main checkout's `.worktrees/`. It prints absolute paths, and the executor role and skill tell the executor to record the git-reported path. (2) Folded idea: `new-review --covers` plus `reviews_for` honouring `covers`, applied the same way in `next --stage review` and `STATE.md`, with unit tests for both the covered and uncovered amend.

## Code vs plan
Branch head: CI `36291400442` completed/success. `git merge-tree origin/main origin/<branch>` is clean.

- Task 1 (`_git_worktree_list`, `parse_worktree_list`, `stale_worktrees`, rewritten `cmd_stale_worktrees`, fallback): followed.
- Task 2 (executor role and skill recording sentence, orchestrate `prune` wording): followed. `.claude/skills` is a tracked symlink to `.agents/skills` (`git ls-tree origin/main .claude/` → `120000 … .claude/skills`), so the "edit both copies" step is satisfied by construction.
- Task 3 (`covers` in schema, scan, `new-review`; the review skill sentence): followed.
- The `_merged_plan(branch=None)` helper deviation is justified.

```
$ python3 -m unittest discover -s tools/harness/tests
Ran 50 tests in 1.866s — OK
$ python3 tools/harness/cli.py validate; echo validate=$?   → validate=0
$ python3 tools/harness/cli.py doctor                       → ok: 0 problem(s)
# real command, branch cli.py run against fresh-main plans (prints only, removes nothing):
$ python3 ../rv-stalewt/tools/harness/cli.py stale-worktrees
…/.claude/worktrees/daily-task-evaluation-planning-971a45/.worktrees/every-google-403-becomes-409-reauth-required-so-a-quota-erro
…/.claude/worktrees/daily-task-evaluation-planning-971a45/.worktrees/geminiprovider-drops-every-response-part-after-the-first-so-
…/.claude/worktrees/harness-daily-execute-154026/.worktrees/nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect
…/.claude/worktrees/harness-daily-execute-154026/.worktrees/providertimeout-of-30-s-makes-roadmap-generation-impossible-
…/.worktrees/containerised-deploy-dockerfiles-production-compose-runbook-
$ python3 tools/harness/cli.py stale-worktrees    (main's cli.py, same plans)   → (nothing)
```

## Quality
- Design: branch-first matching with a guarded legacy basename rule is the right call. Absolute output makes `prune` cwd-independent.
- Test honesty (low bug): the legacy-basename clause is never exercised on git's list, only in the no-git fallback, and the ancestor skip is never exercised with `cwd` below an entry. Both guard a command whose output is fed to `git worktree remove`.
- Operational note: `git worktree list` also reports worktrees inside other live sessions' sandboxes (`.claude/worktrees/<session>/…`). `prune` could remove one under a running session if its branch is merged. `git worktree remove` without `--force` refuses a dirty or locked tree, so the risk is small. The orchestrator should still not run `prune` while other sessions are active.
- Pre-existing limitation, not this plan's: the listing depends on `merged: true`. Squash-merged branches whose flag was never set (for example plan 1 of today's set) are not listed.
- CODEMAP Harness tooling paragraph: accurate.

## Bugs filed
- `harness/ideas/_inbox/stale-worktrees-has-no-test-for-the-legacy-basename-match-on.md` (low). Two untested branches of `stale_worktrees()`.

## Verdict
`pass-with-bugs`. It may go into today's daily PR; it merges cleanly onto main.
