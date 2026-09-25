---
name: harness-review
description: Review a done harness plan — re-run verification in its worktree, compare code to plan and plan to idea, file bugs into the inbox, write a review file, comment on the PR. Use for /review and the review stage of /harness run.
---

# harness-review

Adopt `.agents/roles/reviewer.md`. Input: a plan path, or nothing (then `PLAN=$(python3 tools/harness/cli.py next --stage review)`; if empty, report "nothing to review" and stop).

## Procedure

1. `python3 tools/harness/cli.py validate` — stop on failure.
2. Read the plan (including its execution summary), its idea, its design if any, `harness/CODEMAP.md`. Note `branch`, `worktree`, `pr`.
3. `cd <worktree>`; `git diff main...<branch> --stat`; run the project's build/tests, the plan's *Verification* commands, and the executor's *Runtime proof* block from `## Execution summary`. Record real output. Anything that does not reproduce is an executor gate failure — file it as a blocker and say so in the review.
4. Walk the diff against the plan tasks (code-review skill; typescript-review for Nuxt code). Walk the idea's Expected output against the result. For a plan with a `design:`, walk the design doc's acceptance list and states against the running screen (browser or `npm run dev`) and check the UI against `harness/UI-KIT.md` — a kit violation or a missing state is a finding. Then review for **quality** — the role's scope list: design and boundaries, correctness on untried inputs, performance and resource use, conventions and idiom, error handling, test honesty, documentation accuracy, security.
5. For each bug found: `python3 tools/harness/cli.py new-idea --run harness/ideas/_inbox --title "<bug title>" --type bug --source reviewer --priority <high|medium|low>`, then fill the body (*Why* = impact, *Expected output* = correct behaviour, *Evidence* = plan path, file:line, failing command).
   - If the bug must be fixed before this branch can merge, make it a **blocker**: `--priority high`, then `python3 tools/harness/cli.py set <file> blocks=$PLAN`. Verify with `python3 tools/harness/cli.py blockers --plan $PLAN` (exit 1 while any are unresolved).
6. Decide the verdict per the role. `REV=$(python3 tools/harness/cli.py new-review --plan $PLAN --verdict <v> --bugs <bug paths…>)`; fill the review body sections, paste verification output under *Code vs plan*.
7. If CODEMAP needs correction: edit it in the worktree and commit on the branch.
8. **PR**: there is normally none — the owner takes one PR per day, opened by `/harness daily-pr` after every plan for the day is reviewed. The verdict lives in the review file, and the daily PR body quotes it. Only if the plan already carries a `pr` (an older per-plan PR, or the daily PR is already open): `gh pr comment <pr> --body "<verdict + 3-line summary + path to review file>"`; on `pass`/`pass-with-bugs` `gh pr ready <pr>`; on `fail` leave it Draft.
9. `python3 tools/harness/cli.py validate && python3 tools/harness/cli.py state`; commit `harness/` with `harness: review <slug> (<verdict>)`.
10. **Report:** verdict, bugs filed (paths), whether the PR is ready, and the merge command the human can run: `/harness merge $PLAN`.
