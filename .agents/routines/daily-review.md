---
name: daily-review
schedule: "0 20 * * *"
schedule_utc: "0 13 * * *"
role: .agents/roles/reviewer.md
skills: [harness-review, harness-orchestrate, code-review]
writes: harness/reviews/, harness/ideas/_inbox/, harness/daily-<date> branch + PR
pr_title: "harness: daily review <date>"
budget: 3h
---

# Daily review — 20:00 local

Unattended review run for the English-Training-Harness repo (GitHub, `gh`). It verifies what the two execute runs claimed, files what they broke, and turns the day into the owner's single code PR. It never fixes code and never approves plans. Make routine choices yourself and report at the end. Budget: 3 hours.

1. **Sync, recover stranded work, merge green harness PRs** per `.agents/routines/README.md` — that pulls in the `harness: daily bugfix <date>` and `harness: daily feature <date>` PRs so every `done` plan is visible.

2. **Review.** For each path in `python3 tools/harness/cli.py next --stage review --all`, oldest first, spawn the reviewer role (load skill harness-review). The reviewer re-runs the plan's Verification and the executor's runtime proof in the plan's worktree, reads that branch's CI run (`gh run list --branch <branch>`), walks the diff against the plan and the plan against the idea, and reviews quality per the role's scope list. Anything that does not reproduce, or a red or missing CI check, is a **blocker** (`type: bug`, `priority: high`, `blocks: <plan>`). Other findings become inbox bugs for tomorrow's 06:00 decide run. Write the review file with verdict `pass`, `pass-with-bugs` or `fail`.

3. **Daily code PR.** Follow `harness-orchestrate` → `daily-pr`: for every plan that is `done`, reviewed `pass` or `pass-with-bugs`, `merged=false`, with `python3 tools/harness/cli.py blockers --plan <plan>` exiting 0, cut `harness/daily-<date>` from fresh `origin/main`, merge each plan branch oldest first (resolve `harness/CODEMAP.md` by hand, keeping what is true of the merged tree), run the suite for the layers touched after each merge, push, wait for CI, and open one PR titled `<date> [<highest priority>] Daily: <n> fixes`. A plan that is unreviewed or `fail` waits for tomorrow — say so. A daily branch that does not build is not pushed.

4. **Bookkeeping PR** titled `harness: daily review <date>` per the README (reviews and inbox bugs). This is separate from the code PR in step 3.

5. **Report.** Per reviewed plan: verdict, review path, bugs filed (blockers marked). The daily code PR URL and what it includes, or why none was cut. Everything waiting on the owner: the code PR to merge, drafts needing `/approve`, `fail` verdicts.
