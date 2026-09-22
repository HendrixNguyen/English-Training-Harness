---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md
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
