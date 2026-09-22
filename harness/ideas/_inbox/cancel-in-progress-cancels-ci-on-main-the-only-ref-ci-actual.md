---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md
plan: harness/plans/2026-09-22-cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md
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

## Evaluation
**Verdict: select, priority high** (blocker on `harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md`, escalated from the review's medium by the project owner's controller — not re-litigated here).

**Is the Why real?** Yes. The concurrency group is `ci-${{ github.workflow }}-${{ github.ref }}` with `cancel-in-progress: true` (`.github/workflows/ci.yml:11-13` on the branch), so every push to `main` shares one group and a run for a merge commit is cancelled by the next merge. `/harness merge` pushes `main` directly and step 7 of `/harness run` merges several plans per pass, so this is the normal path, not an edge case.

**Root cause.** Unconditional `cancel-in-progress` on a group keyed only by ref. The idea's proposed fix — keep the group, make `cancel-in-progress` conditional on the ref — is necessary but not sufficient: a group holds one running plus one *pending* run, and a third push cancels the pending one, so three back-to-back merges would still drop a verdict. The plan therefore also keys the group on `github.sha` when the ref is `main`, giving every merge commit a singleton group, and keeps the conditional `cancel-in-progress` to state the intent. Checked with `actionlint` 1.7.12 and a PyYAML structural assert on a copy of the file (2026-09-22).

**Achievable in one plan?** Yes — a few lines of one file, plus the docs that describe it. It is planned together with the companion blocker `ci-never-runs-on-harness-branches-so-it-gates-nothing-before.md` (same file, same review, same branch); that idea's `plan:` points at this plan. The reviewer's CODEMAP doc findings and the `timeout-minutes` half of the timeout idea ride along because they touch the same lines and the same verification.

**Dependencies:** none beyond the branch under review. The plan also makes `cli.py blockers` honour an idea's `plan:` back-link, because without that the companion blocker can never clear (verified on a scratch copy: exit 1 before, exit 0 after).

**Plan:** `harness/plans/2026-09-22-cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md` (amends the CI plan).
