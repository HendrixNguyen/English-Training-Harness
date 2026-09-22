# Harness acceptance tests — 2026-09-22

All three tests from the design spec §9 passed on the first live run. Roles were driven via the Claude adapter prompts (slash commands were not yet registered in the session that built them).

| # | Test | Result | Notes |
|---|------|--------|-------|
| 3 | `/harness run --stages execute,review` with empty queues | pass | Log `20260922T152444.log`; only `harness/` changed. |
| 1 | `/ideate --count 3` | pass | Run `2026-09-22-run-01`: 3 feature ideas, every `## Why` argues from §5.2 / 30-min target / CEFR; valid frontmatter; STATE.md Proposed. |
| 2 | `/idea` → evaluate → approve → execute → review → merge | pass | Idea `add-unique-user-id-to-pet-states-ddl` (bug, human). Evaluator: selected/high, 2-task plan dry-run on scratch. Executor: worktree + branch `harness/2026-09-22-high-…`, 2 commits, push/PR skipped (no remote). Reviewer: `pass-with-bugs`, re-ran all verification, filed 2 pre-existing spec defects to `_inbox/`. Merge: `--no-ff` into `main`, worktree/branch removed, `merged: true`. |

## Findings fixed during the run
- Ideator reported "spec has no §7" — headings are backslash-escaped. Added *Reading the spec* to `AGENTS.md`.
- `/idea` used `$(ls -d …)`; rtk rewrote the output and broke the path. Command now uses a shell glob; caveat added to `AGENTS.md`.
- `rg` is not installed on this machine; roles/skills now say `rg`/`grep -n`.
- `/harness merge` now skips `git push` when no remote is configured.

## Open
- No GitHub remote yet, so push / Draft PR / `gh pr ready` paths are untested.
- The two reviewer-filed bugs (`_inbox/`) will be swept into the next ideation run.
