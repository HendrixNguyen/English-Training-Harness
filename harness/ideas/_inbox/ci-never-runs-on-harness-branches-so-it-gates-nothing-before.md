---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md
plan: harness/plans/2026-09-22-cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md
---
# CI never runs on harness/* branches so it gates nothing before merge

## Why
`.github/workflows/ci.yml:3-6` triggers on `push` to `main` and on `pull_request`. Neither trigger
fires for the workflow the harness actually produces:

- **No PR can exist.** The `gh` account on this machine has no write access to this repo. The
  executor of the plan under review tried and got
  `GraphQL: must be a collaborator (createPullRequest)`. Every future `/execute` hits the same wall,
  so the `pull_request` trigger is dead in practice.
- **Pushing the branch fires nothing.** `on.push.branches: [main]` excludes `harness/**`, which is
  the only branch namespace the executor ever pushes
  (`.agents/skills/harness-execute/SKILL.md`, and `AGENTS.md` "Never merge to `main`").
- **The merge is not gated either.** `/harness merge` is
  `git checkout main && git merge --no-ff <branch> && git push origin main`
  (`.agents/skills/harness-orchestrate/SKILL.md:29`). CI therefore runs for the first time on the
  merge commit, *after* the human has already merged.

So the workflow that `AGENTS.md:23` calls "the outer verification loop" currently reports after the
fact on `main` and never before a merge. The same line asserts "A red check on a `harness/*` PR is a
review blocker" — a rule no role can apply, because no such check is ever produced.

This is a gap, not a false green: an absent check is visible, and the `main` run still catches a
broken merge. It is filed medium rather than as a blocker for that reason — but it is the difference
between CI being the harness's outer check and CI being a post-mortem.

## Expected output
Pushing a `harness/*` branch produces the three status checks, so a reviewer (and `/harness merge`)
can see them before `main` moves.

Technical — widen the push trigger in `.github/workflows/ci.yml`:

```yaml
on:
  push:
    branches: [main, 'harness/**']
  pull_request:
```

Keep `pull_request` for the day the account gains write access; `concurrency` already keys on
`github.ref`, so a branch that later gets a PR does not run twice for the same ref. Then correct
`AGENTS.md`'s "red check on a `harness/*` PR" wording to name the branch check, and have the review
and merge steps read it (`gh run list --branch <branch>` works read-only on a public repo).

## Evidence
- Plan under review: `harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md`
- `.github/workflows/ci.yml:3-6` — the triggers.
- `AGENTS.md:23` — the rule the triggers cannot support.
- That plan's `## Execution summary`, *Push / PR* — the recorded `must be a collaborator` failure.
- `.agents/skills/harness-orchestrate/SKILL.md:29` — merge pushes `main` directly.

## Evaluation
**Verdict: select, priority high** — planned together with `cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md`; this idea's `plan:` points at that plan, `harness/plans/2026-09-22-cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md`, which amends the CI plan.

**Why it is a blocker (recorded, not re-litigated).** The CI idea's *Expected output* requires a branch that builds only on the author's machine to fail visibly "before a human is asked to merge it". With `on.push.branches: [main]`, a `pull_request` trigger that cannot fire (this account cannot open PRs — `must be a collaborator` in the CI plan's execution summary) and `/harness merge` pushing `main` directly, CI first runs *after* the merge has landed, so that requirement is not delivered. The project owner's controller escalated the review's medium to a blocker on that basis.

**Root cause.** `.github/workflows/ci.yml:4-5` — `push.branches: [main]` excludes the only namespace the executor pushes. Fix: `branches: [main, 'harness/**']`, keeping `pull_request` (free now, useful the moment a PR can exist). The `AGENTS.md:23` rule is rewritten so the pushed branch's run, not a PR check, is what the reviewer reads; `harness/CODEMAP.md`'s CI section is rewritten to match (folding in the reviewer's separate doc findings).

**Wiring note.** One plan covers both blockers. `scan.blockers_for` today resolves a fix plan only through the plan's `idea:` field, so this idea's `plan:` back-link would be ignored and the blocker would never clear; the shared plan's Task 3 fixes that in `tools/harness/scan.py` with a regression test.
