---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# cancel-in-progress cancels CI on main, the only ref CI actually runs on

## Why
`.github/workflows/ci.yml:11-13`:

```yaml
concurrency:
  group: ci-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

The group keys on `github.ref`, so every push to `main` shares one group with every other push to
`main`. Two merges landing within the ~3-5 minutes a run takes means the first run is **cancelled**,
and the commit it was checking never gets a verdict — no red, no green, just `cancelled`, which
GitHub does not surface as a failure anywhere the harness looks.

Cancelling superseded runs is right for a feature branch, where only the tip matters. It is wrong for
a trunk where every commit is a merge someone will later bisect against. And it bites harder here
than in a normal repo: per the companion bug
`ci-never-runs-on-harness-branches-so-it-gates-nothing-before.md`, `main` is currently the *only* ref
CI runs on at all, and `/harness merge` pushes merges to it back to back during an unattended
`/harness run`. A batch of three merged plans can leave two of the three merge commits unverified.

## Expected output
Superseded runs are still cancelled on branches, but never on the default branch: every commit that
lands on `main` completes its own run.

Technical — in `.github/workflows/ci.yml`:

```yaml
concurrency:
  group: ci-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: ${{ github.ref != 'refs/heads/main' }}
```

`cancel-in-progress` accepts an expression. Optionally drop the redundant `ci-` literal, since
`${{ github.workflow }}` is already `CI`.

## Evidence
- Plan under review: `harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md`
- `.github/workflows/ci.yml:11-13` — the concurrency block.
- `.agents/skills/harness-orchestrate/SKILL.md:29` — `merge` pushes `main`; step 7 of `run` merges
  several plans in one unattended pass.
- Related: `harness/ideas/_inbox/ci-never-runs-on-harness-branches-so-it-gates-nothing-before.md`.
