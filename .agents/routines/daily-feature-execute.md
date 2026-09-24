---
name: daily-feature-execute
schedule: "0 14 * * *"
schedule_utc: "0 7 * * *"
role: .agents/roles/executor.md
skills: [harness-execute, harness-evaluate, frontend-design, test-driven-development]
writes: harness/plans/ (status), harness/designs/, code on harness/* branches
pr_title: "harness: daily feature <date>"
budget: until 20:00 local
---

# Daily feature execute — 14:00 local

Unattended FEATURE execute run for the English-Training-Harness repo (GitHub, `gh`). This run only touches plans whose frontmatter is `type: feature` or `type: mvp-slice`. Bug plans belong to `daily-bugfix-execute` (10:00) — never execute them here, except a blocker (`blocks:` / `amends:`) that targets one of this run's own feature branches. Make routine choices yourself and report at the end.

1. **Sync and merge green harness PRs** per `.agents/routines/README.md` — that pulls in the 06:00 evaluate PR and the 10:00 `harness: daily bugfix <date>` PR so the two runs' `STATE.md` regenerations land in sequence.

2. **Select feature plans:**
   ```
   for p in $(python3 tools/harness/cli.py next --stage execute --all); do grep -Eq '^type: (feature|mvp-slice)' "$p" && echo "$p"; done
   ```
   Leave every `type: bug` plan untouched, even if approved. `status: approved` is set by the evaluator for `priority: high` features and by the owner's `/approve` for medium/low — never set it yourself.

3. **Evaluate only if needed.** If step 2 is empty and `python3 tools/harness/cli.py next --stage evaluate --all` contains feature ideas (`type: feature` in a run folder, not `_inbox`), spawn the evaluator role (load skill harness-evaluate) on at most 3 feature ideas that can each be finished today, ranked on user impact per AGENTS.md. UI ideas get a design doc from the frontend spec §6–7 via the frontend-design skill; backend plans follow YAGNI, `harness/CODEMAP.md` and the backend spec's §6 DTO contracts. Only `priority: high` plans auto-approve; medium/low stay `draft` for the owner. Skip bug ideas entirely. Evaluation and execution share this checkout — never merge another branch to get a plan. Re-run step 2.

4. **Execute.** For each feature plan in priority order: `python3 tools/harness/cli.py lock <plan>` (skip and report a plan already locked), then spawn the executor role (load skill harness-execute). Every worktree is created from freshly fetched `origin/main`. Plans touching different layers may run in parallel (at most 3 at once), each with its own `COMPOSE_PROJECT_NAME` and ports; the rest run one after another. Plans with `amends:` run on their parent plan's branch. Frontend plans follow their design doc exactly; a plan that adds an endpoint keeps the 1st-thinking doc's §7 endpoint list and the backend spec's DTOs in sync; a store change keeps the spec DDL identical to the migrations.

5. **Test before claiming done.** A plan is `done` only when its Verification section, the execute skill's runtime proof (build, full test suite, boot the app, exercise the new feature end to end through one real user path) and CI on its pushed `harness/*` branch are all green. Otherwise it stays `failed` with the reproduction — no hand-fixes outside the plan. Unlock every plan you locked when its executor finishes.

6. **Bookkeeping PR** titled `harness: daily feature <date>` per the README.

7. **Report.** PRs merged or refused; feature ideas evaluated, approved or left draft; per plan: status, branch, CI run, test evidence and the user path exercised. Everything waiting on the owner (medium/low drafts needing `/approve`, failures, refused permissions, locked plans skipped) and any approved bug plans you left for the bugfix run.
