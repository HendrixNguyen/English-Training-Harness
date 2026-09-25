---
name: daily-bugfix-execute
schedule: "0 10 * * *"
schedule_utc: "0 3 * * *"
role: .agents/roles/executor.md
skills: [harness-execute, harness-evaluate, systematic-debugging, test-driven-development]
writes: harness/plans/ (status), harness/reviews/ (none), code on harness/* branches
pr_title: "harness: daily bugfix <date>"
budget: until 14:00 local
---

# Daily bugfix execute — 10:00 local

Unattended BUGFIX execute run for the English-Training-Harness repo (GitHub, `gh`). This run only touches bug plans — a plan's type is the `type:` of the idea its `idea:` line points at (plans carry no `type:` of their own). Feature and mvp-slice plans belong to `daily-feature-execute` — never execute, approve or reprioritise them here. Make routine choices yourself and report at the end.

1. **Sync, recover stranded work, merge green harness PRs** per `.agents/routines/README.md` — that pulls in the 06:00 `harness: evaluate <date>` PR carrying today's approved plans. If a merge is refused or CI is red, note it and continue with what `main` has.

2. **Select bug plans**, blockers first:
   ```
   for p in $(python3 tools/harness/cli.py next --stage execute --all); do i=$(sed -n 's/^idea: *//p' "$p" | head -1); grep -q '^type: bug' "$i" && echo "$p"; done
   ```
   Plans with `blocks:` or `amends:` are blockers and jump the queue. Leave every `type: feature` / `type: mvp-slice` plan untouched, even if approved.

3. **Evaluate only if needed.** If step 2 is empty and `python3 tools/harness/cli.py next --stage evaluate --all` contains bug items (anything under `harness/ideas/_inbox/`, or an idea with `type: bug`), spawn the evaluator role (load skill harness-evaluate) on at most 5 bug items that can each be finished today. Bug plans auto-approve; a bug whose fix touches `frontend/` still gets a designer-role design doc before its plan (states, copy and kit rules are where frontend fixes go wrong). Skip feature ideas entirely. Evaluation and execution share this checkout, so the executor reads the new plans directly — never merge another branch to get a plan. Re-run step 2.

4. **Execute.** For each bug plan in priority order: `python3 tools/harness/cli.py lock <plan>` (skip and report a plan already locked — another run owns it), then spawn the executor role (load skill harness-execute). Every worktree is created from freshly fetched `origin/main`. Plans touching different layers may run in parallel (at most 3 at once), each with its own `COMPOSE_PROJECT_NAME` and ports; the rest run one after another. Plans with `amends:` run on their parent plan's branch. A bug fix starts with a failing test that reproduces the bug (systematic-debugging, test-driven-development); a fix with no reproduction is not done.

5. **Test before claiming done.** A plan is `done` only when its Verification section, the execute skill's runtime proof (build, full test suite, boot the app, exercise the path the bug broke) and CI on its pushed `harness/*` branch are all green. Otherwise it stays `failed` with the reproduction — no hand-fixes outside the plan. Unlock every plan you locked when its executor finishes.

6. **Bookkeeping PR** titled `harness: daily bugfix <date>` per the README.

7. **Report.** PRs merged or refused; bug items evaluated and approved; per plan: status, branch, CI run, reproduction test and evidence. Everything waiting on the owner (failures, refused permissions, locked plans skipped) and any approved feature plans you left for the feature run.
