---
name: harness-execute
description: Execute one approved harness plan in a dedicated git worktree, push the branch, and record the outcome. Never opens a PR — the owner takes one PR per day. Use for /execute and the execute stage of /harness run.
---

# harness-execute

Adopt `.agents/roles/executor.md`. Input: a plan path, or nothing (then `PLAN=$(python3 tools/harness/cli.py next --stage execute)`; if empty, report "nothing approved" and stop).

Let `ROOT` = main checkout (where you start). All `cli.py` calls run from `ROOT`.

## Procedure

1. `python3 tools/harness/cli.py validate` — stop on failure.
2. `python3 tools/harness/cli.py lock $PLAN` — locks this plan only; if it fails, another executor is already running *this same plan*; report and stop.
3. Read the plan, its idea, its design (if any), `harness/UI-KIT.md` when the plan touches `frontend/`, and `harness/CODEMAP.md`. The design doc is binding for layout, states, copy and components; a deviation from it is logged like any other deviation. Do not explore beyond what the plan names.
4. Derive names: `SLUG=$(basename $PLAN .md | sed 's/^[0-9-]*-//')`; `PRIO` = plan frontmatter `priority`; `DATE=$(date +%F)`; `BRANCH=harness/$DATE-$PRIO-$SLUG`; `WT=.worktrees/$SLUG`.
5. `git fetch origin main && git worktree add $WT -b $BRANCH origin/main` (using-git-worktrees skill). Always base on the freshly fetched `origin/main`, never local `main` — local `main` in the main checkout is not kept up to date and can be many commits behind. Never merge another session's or plan's branch to get a plan: plans are read from `ROOT`'s `harness/plans/`, so run the evaluator and executor in the same session (`/harness run --stages evaluate,execute`) or wait for the PR that carries the plan to reach `main`. Then `python3 tools/harness/cli.py set $PLAN status=executing branch=$BRANCH worktree=$WT`.
   - **Amending plan** (frontmatter has `amends: <other plan>`): do **not** create a branch or worktree. Read `branch` and `worktree` from that other plan and work in its existing worktree, on its existing branch — the fix has to land in the same history that the reviewer blocked. Set `status=executing branch=<inherited> worktree=<inherited>` on your own plan too, so `STATE.md` and the reviewer can find it. It lands on the same branch, so nothing extra is pushed or opened.
6. `cd $WT`. Execute the plan with the executing-plans skill: for each task — write the failing test, run it, implement, run, commit with the plan's message. Use `rg`/`grep -n` to find code; read only matched ranges.
7. After the last task, run the plan's *Verification* section. Update `harness/CODEMAP.md` for touched packages and commit it.
8. **Prove it runs** (still in `$WT`) — the role's *Definition of done*. In order, capturing real output:
   a. Build the project.
   b. Run the entire test suite from a clean shell (`env -u <every service/env var the project reads>` where that is meaningful), not only the tests this plan added.
   c. Boot the application and exercise one real path end to end (e.g. start the server on a spare port and `curl` an endpoint; for a library, run a real call). Shut it down afterwards.
   d. Run **every command the plan, the Makefile, the README or `harness/CODEMAP.md` tells a human to run**, exactly as documented, in a clean environment. Include the destructive-sounding ones and check they refuse when they should.
   e. If any of these fails: do not work around it and do not hand-fix outside the plan. Record the reproduction and go to `failed` in step 9.
   f. **Bound blocking commands** with each tool's own flag — `go test -timeout 120s`, `curl --max-time 60`, `docker compose up -d --wait --wait-timeout 120`. There is no `timeout`/`gtimeout` binary here.
   g. **Clean up before moving on.** Kill any app process you started, `make down` your compose project, delete the scratch `backend/.env`. Verify with `pgrep -fl exe/api` and `docker ps`; both must be free of anything you created. Use a unique `COMPOSE_PROJECT_NAME` and non-default host ports so you never touch another worktree's or the owner's containers.

9. **Record outcome** (back in `ROOT` — `cd $ROOT` first). The plan file under `harness/plans/` is **ROOT bookkeeping only**: append the summary to ROOT's copy and never edit or commit `harness/plans/*` inside the worktree, or the merge will conflict on it. The only `harness/` file the branch may change is `harness/CODEMAP.md`.
   - Success (every check in step 8 passed): append `## Execution summary` — built / deviations + why / the plan's verification output / a **Runtime proof** subsection with the step 8 output — then `python3 tools/harness/cli.py set $PLAN status=done`.
   - Blocked, or any step 8 check failed: append `## Failure` (what you ran, what happened, suggested plan change), then `python3 tools/harness/cli.py set $PLAN status=failed`. Skip steps 10–11.
10. **Push the branch** (skip with a note in the summary if `git remote get-url origin` fails):
   `cd $WT && git push -u origin $BRANCH`
   **Do not open a PR.** The owner takes one PR per day, not one per plan (AGENTS.md); `/harness daily-pr` opens it once the day's plans are all done and reviewed. Opening a per-plan PR is a defect, not initiative.

   **Then wait for CI on your branch.** The push triggers `.github/workflows/ci.yml`. Watch it: `gh run list --branch $BRANCH --limit 1`, then `gh run watch <id> --exit-status`. CI is the only check that runs somewhere other than this machine, so it is the one that can catch an environment-specific pass.
   - Green: record the run URL in `## Execution summary`.
   - Red: read the failing job's log (`gh run view <id> --log-failed`), fix the cause **within the plan's intent**, and push again. Never make CI pass by skipping, loosening or deleting a check.
   - Red for a reason the plan does not cover, or still red after a genuine attempt: set `status=failed` with the run URL and the failing output in `## Failure`. A branch whose CI is red is not `done`.
   - If `gh` cannot read runs (no access, no remote), say so in the summary and leave the plan `done` on your local evidence alone.
11. `python3 tools/harness/cli.py unlock $PLAN && python3 tools/harness/cli.py state`; in `ROOT`: `git add harness && git commit -m "harness: execute $SLUG ($STATUS)"`.
12. **Report:** status, branch, worktree, PR URL, verification result, deviations. Stop — do not review.
