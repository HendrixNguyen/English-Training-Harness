---
name: harness-execute
description: Execute one approved harness plan in a dedicated git worktree, push the branch, open a Draft PR, and record the outcome. Use for /execute and the execute stage of /harness run.
---

# harness-execute

Adopt `.agents/roles/executor.md`. Input: a plan path, or nothing (then `PLAN=$(python3 tools/harness/cli.py next --stage execute)`; if empty, report "nothing approved" and stop).

Let `ROOT` = main checkout (where you start). All `cli.py` calls run from `ROOT`.

## Procedure

1. `python3 tools/harness/cli.py validate` — stop on failure.
2. `python3 tools/harness/cli.py lock $PLAN` — if it fails, another executor is running; report and stop.
3. Read the plan, its idea, its design (if any), and `harness/CODEMAP.md`. Do not explore beyond what the plan names.
4. Derive names: `SLUG=$(basename $PLAN .md | sed 's/^[0-9-]*-//')`; `PRIO` = plan frontmatter `priority`; `DATE=$(date +%F)`; `BRANCH=harness/$DATE-$PRIO-$SLUG`; `WT=.worktrees/$SLUG`.
5. `git worktree add $WT -b $BRANCH main` (using-git-worktrees skill). Then `python3 tools/harness/cli.py set $PLAN status=executing branch=$BRANCH worktree=$WT`.
6. `cd $WT`. Execute the plan with the executing-plans skill: for each task — write the failing test, run it, implement, run, commit with the plan's message. Use `rg` to find code; read only matched ranges.
7. After the last task, run the plan's *Verification* section. Update `harness/CODEMAP.md` for touched packages and commit it.
8. **Record outcome** (back in `ROOT`):
   - Success: append `## Execution summary` to the plan (built / deviations + why / verification output), then `python3 tools/harness/cli.py set $PLAN status=done`.
   - Blocked: append `## Failure` (tried / blocker / suggested plan change), then `python3 tools/harness/cli.py set $PLAN status=failed`. Skip steps 9–10.
9. **Push + Draft PR** (skip with a note in the summary if `git remote get-url origin` fails or `gh auth status` fails):
   `cd $WT && git push -u origin $BRANCH`, then
   `gh pr create --draft --base main --head $BRANCH --title "[$DATE][P<n>] <Idea title>" --body-file <tmpfile> --label harness --label "type: <type>" --label "priority: $PRIO"`
   where `P1/P2/P3` = high/medium/low, and for `mvp-slice` use `[MVP-<order>]` instead of `[P<n>]`. Create missing labels with `gh label create`. Body = idea *Why* + *Expected output*, links to plan and idea paths, the execution summary, then the attribution line `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
   Then `python3 tools/harness/cli.py set $PLAN pr=<url>`.
10. `python3 tools/harness/cli.py unlock && python3 tools/harness/cli.py state`; in `ROOT`: `git add harness && git commit -m "harness: execute $SLUG ($STATUS)"`.
11. **Report:** status, branch, worktree, PR URL, verification result, deviations. Stop — do not review.
