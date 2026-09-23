---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
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
