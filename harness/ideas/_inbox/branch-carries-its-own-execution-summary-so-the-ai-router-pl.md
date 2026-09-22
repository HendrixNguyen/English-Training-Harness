---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
---
# Branch carries its own execution summary so the ai-router plan file conflicts on merge

## Why
Commit `0966115` on `harness/2026-09-22-high-ai-router-multi-llm-providers-task-strategies-and-rate-limit`
adds `## Execution summary` to `harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md`.
By convention (and per ROOT commit `6d55426`, "harness: execution summary is ROOT-only bookkeeping,
never a branch commit") that section is written on `main`, not on the branch. `main` already carries
a *different*, fuller version of it (ROOT commit `29c4355`), plus the `status: done` /
`branch:` / `worktree:` frontmatter the executor set through `cli.py`.

Both sides therefore edited the same file relative to the merge base `1db7d60`, and the merge does
not auto-resolve:

```
$ git merge-tree --write-tree --name-only main HEAD
harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md
Auto-merging harness/plans/…-ai-router-….md
CONFLICT (content): Merge conflict in harness/plans/…-ai-router-….md
```

The hazard is not the conflict itself — it is the shape of it. The branch's copy is *older*: its
frontmatter says `status: approved` and has no `branch:` or `worktree:` keys, and its summary is
missing the `### CI` section. A merge resolved "take theirs" (the usual instinct when the branch is
the thing being merged) silently rolls the plan back to `approved` and drops the branch/worktree
bookkeeping, after which `cli.py state` regenerates `harness/STATE.md` from wrong frontmatter and
`/harness merge`'s own `set … merged=true` would be operating on a plan that claims it was never
executed. No Go file conflicts — `merge-tree` reports this plan file and nothing else.

## Expected output
`/harness merge` for this plan resolves the conflict in favour of **`main`'s** copy of the plan file
(`git checkout --ours <plan>` during the merge, or `git merge -X ours` scoped to that path), keeps
`status: done` + `branch:` + `worktree:` + the `### CI` section, and takes every other path from the
branch unchanged. Afterwards `python3 tools/harness/cli.py validate` exits 0 and `STATE.md`
regenerates with the plan still `done`.

Longer term the convention is worth enforcing rather than remembering: either `.agents/roles/executor.md`'s
Definition of done gains an explicit "never commit the plan file on the branch" line with a
pre-push check, or `cli.py` grows a `stale-worktrees`-style lint that fails when a `harness/*` branch
touches `harness/plans/`.

## Evidence
- Plan under review: `harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md`
- Offending commit: `0966115` "harness: execution summary and runtime proof for ai-router plan" (90 added lines, the only non-`backend/`, non-CODEMAP change on the branch).
- ROOT commit `6d55426` states the convention; ROOT commit `29c4355` wrote the canonical summary on `main`.
- Reproduce: `cd .worktrees/ai-router-multi-llm-providers-task-strategies-and-rate-limit && git merge-tree --write-tree --name-only main HEAD` → `CONFLICT (content)` on that one path.
- Diff of the two copies: `git diff main HEAD -- harness/plans/2026-09-22-ai-router-….md` → branch has `status: approved`, no `branch:`/`worktree:`, no `### CI`.
