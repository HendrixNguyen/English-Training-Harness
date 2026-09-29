---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Folded into harness/plans/2026-09-27-executor-worktree-nested-under-claude-worktrees-while-plan-f.md: Task 3 adds an explicit covers: list on reviews so reviews_for, next --stage review and STATE.md count a parent re-review for its amend plans"
---
# next --stage review re-queues merged amend plans whose review was filed under the parent plan

## Why
The review stage and every unattended `/harness run` start from `cli.py next --stage review`. Four 2026-09-22 amend plans
come back from it, and all four are `done` and `merged: true` with reviews on file. That queue head is noise, so a
scheduled reviewer either burns a run re-reviewing merged code or learns to ignore `next`. Once it ignores `next`,
it will also miss a plan that really is unreviewed.

## Expected output
- `next --stage review` returns nothing when every done plan has a review, whether the review is filed under the plan
  or under the plan it `amends:` (a focused re-review of the parent after the amend lands, as on 2026-09-22).
- Pick one rule and apply it everywhere: either `reviews_for` counts a parent-plan review dated at or after the
  amend's execution, or `new-review` gets a way to record the amend plans a re-review covers (e.g. `covers: [...]`).
  `STATE.md` applies the same rule, so the Done list stops showing no review for these plans.
- A unit test in `tools/harness/tests` for an amend plan whose re-review sits on the parent.

## Evidence
- `python3 tools/harness/cli.py next --stage review --all` on 2026-09-23 returns:
  `2026-09-22-a-rejected-post-quests-progress-still-writes-redis-and-daily.md`,
  `2026-09-22-cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md`,
  `2026-09-22-docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md`,
  `2026-09-22-go-test-drops-every-table-in-whatever-database-url-points-at.md`. All four are `merged: true`.
- Their reviews are `harness/reviews/2026-09-22-quests-daily-quest-suite-and-progress-recording-2.md`,
  `...-ci-on-github-actions-for-backend-and-harness-tooling.md` and
  `...-store-go-module-postgres-and-redis-clients-migration-0001-rereview.md`, each with `plan:` set to the parent.
- `tools/harness/scan.py:27` `reviews_for` matches only `fm["plan"] == plan_rel`; `tools/harness/cli.py:192` uses it.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low. Not planned today.**

*Confirmed from the idea's evidence.* `tools/harness/scan.py` `reviews_for` matches only `fm["plan"] == plan_rel`, so an amend plan whose re-review was filed under its parent is re-queued forever by `next --stage review`, and STATE.md shows it without a review.

*Fix, when planned.* One rule applied in `reviews_for` and STATE.md: a review on the parent plan dated at or after the amend plan's `done` transition covers the amend (or `new-review --covers` records it explicitly); a unit test in `tools/harness/tests` for an amend whose re-review sits on the parent. Low: noise for the unattended reviewer, no user impact — but worth doing before the next scheduled `/harness run` burns a review slot on merged code.

_Evaluator, 2026-09-27 — daily decide (bug queue). **Folded into `harness/plans/2026-09-27-executor-worktree-nested-under-claude-worktrees-while-plan-f.md`, Task 3** (ticket B5); this file is closed as `rejected` only because the CLI has no "folded" status — the work is planned and approved._

*Re-checked today.* The four 2026-09-22 amend plans no longer appear in `next --stage review --all` because the 2026-09-25 daily run filed a post-hoc stub review under each amend plan's own path (`harness/reviews/2026-09-25-<amend-slug>.md`). The root cause is untouched: `tools/harness/scan.py:27` `reviews_for` still matches only `fm["plan"] == plan_rel`, so the next parent re-review that re-verifies an amend (the 2026-09-22 pattern in `…-quests-…-2.md` and `…-store-…-rereview.md`) re-queues it again, and `state.py:41` marks it `(unreviewed)`.

*Rule chosen.* The explicit one: a review may carry `covers: [<amend plan>, …]`, written by `new-review --covers`; `reviews_for(p)` counts a review when `plan == p` or `p in covers`, so `next --stage review` and `STATE.md` agree by construction; `schema.validate` requires `covers` to be a list; the reviewer skill's step 6 says when to pass it. The date rule was rejected on the data: plans record no `done` date, and `harness/reviews/2026-09-25-parseroadmap-accepts-…` predates the 2026-09-26 amend `…-parseroadmap-day-sum-rejection-…`, which is legitimately unreviewed and must stay queued — a "parent review dated after the amend" heuristic would need a date the artifacts do not hold. The two 2026-09-22 re-reviews are not back-filled on the code branch (review files must not change there); their amends are already covered by the 09-25 stubs. Unit tests: `covers` clears the queue and the `(unreviewed)` marker; a parent review without `covers` still queues the amend.
