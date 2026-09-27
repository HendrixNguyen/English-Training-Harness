---
plan: harness/plans/2026-09-22-cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md
verdict: pass
bugs: []
---
# Review — cancel-in-progress cancels CI on main, the only ref CI actually runs on

**Plan:** `harness/plans/2026-09-22-cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md`
**Branch/worktree:** `harness/2026-09-22-high-ci-on-github-actions-for-backend-and-harness-tooling` / `.worktrees/ci-on-github-actions-for-backend-and-harness-tooling` — both gone; the plan is already merged.
**Reviewed on:** a detached worktree of `origin/main` at `3f4242d` (post-hoc review, 2026-09-25 daily run). The plan commits are on main: `61cce5c` (workflow), `87dcfb6` (docs), `9376d2b` (scan.py).

## Plan vs idea
The idea asked that runs on `main` never be cancelled. The plan went further than the idea's proposal: a per-SHA group on `main` also covers the pending-run cancellation that a third push would cause. It also folded in the second blocker, running CI on `harness/**` pushes, and delivered both. Live evidence on GitHub Actions: four main pushes within about three minutes on 2026-09-24 (`17ad3af` 16:27, `82c8852` 16:28, `da79c2d` 16:29, `0c36e3f` 16:30) all finished `success` with none cancelled. Across the last 200 runs, the only cancelled runs are on `harness/*` branches (the superseded-branch behaviour the plan intended). None are on `main`.

## Code vs plan
- **Task 1 (workflow): followed, and still true.** `push.branches: [main, 'harness/**']`. `group` is keyed on `github.sha` on main and `github.ref` elsewhere, and `cancel-in-progress: ${{ github.ref != 'refs/heads/main' }}`. `timeout-minutes: 10` is on every job, including the `frontend` job a later plan added (`26cbd7e`), which kept the convention. `actionlint .github/workflows/ci.yml` exits 0 on main.
- **Task 2 (docs): followed.** `AGENTS.md` and `harness/CODEMAP.md` describe the `harness/**` gating and the no-cancel rule on `main`. The stale-wording grep has no hits. CODEMAP's "Three parallel GitHub Actions jobs" went stale when the frontend job arrived. That drift came after this plan and is already tracked: `codemap-ci-section-still-says-three-parallel-jobs-after-the-.md` was folded into `codemap-does-not-document-the-config-and-health-packages-and.md` (selected). `AGENTS.md:36` names only three jobs too, and the same fix should cover it. I did not file a duplicate.
- **Task 3 (scan.py back-link): followed.** `ScanResult.plan_by_rel` and the fallback are in `blockers_for`. `test_two_blockers_can_share_one_fix_plan_via_the_plan_backlink` passes. `cli.py blockers --plan …ci-on-github-actions…` exits 0 on main.

Verification re-run:
```
actionlint .github/workflows/ci.yml            # exit 0
python3 -m unittest discover -s tools/harness/tests   # Ran 38 tests … OK
cli.py blockers --plan harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md   # exit 0
grep 'harness/\*\*' AGENTS.md harness/CODEMAP.md      # AGENTS.md:36, CODEMAP.md:33
gh run list --branch main --limit 12            # 12 × push/completed/success, none cancelled
```

## Quality
- The conditional `cancel-in-progress` is redundant with the singleton group. The plan says it is intentional, as a guard in case someone later collapses the group. That is reasonable.
- A branch that has both a push and an open PR (for example `harness/daily-*`) runs twice, once for each ref's group. The plan's Notes accept this, and it costs minutes, not correctness.
- The `python-version` pin stays out of scope, as the plan said. It is a separate inbox idea.

## Bugs filed
None. The one drift I found (the CODEMAP/AGENTS job count) is already tracked by a selected idea.

## Verdict
**pass.** Delivered and still true on current main, with live Actions history confirming it.
