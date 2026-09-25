---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Process finding for the owner: every remedy is an edit to .agents/skills or .agents/roles, not an executor plan on an app branch; the three edits are listed in the 2026-09-25 decide report."
---
# Plan branch edits harness/plans and misses its own Verification expectations

## Why
Three hygiene slips in one plan. None of them blocks this merge, and all of them will recur
unless the skills change.

1. **The branch edits `harness/plans/*`.** Commit `eb693c0` appends a "_Resolved 2026-09-24 …_"
   line to `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md`
   inside the worktree. Step 9 of `.agents/skills/harness-execute/SKILL.md` forbids this: "never
   edit or commit `harness/plans/*` inside the worktree, or the merge will conflict on it. The only
   `harness/` file the branch may change is `harness/CODEMAP.md`." The executor was following
   the plan, not the skill: plan Task 7 Step 3 and its File-structure table both order the edit.
   Nothing in `.agents/skills/harness-evaluate/SKILL.md` or `.agents/roles/evaluator.md` warns
   plan authors against it.
   *Why it is not a blocker today:* the edit touches the body only (no frontmatter). ROOT's copy
   of that file is identical to `origin/main`'s. `git merge-tree --write-tree` is clean against
   both `origin/main` and ROOT `HEAD`, and `/harness merge` only touches frontmatter lines 1-8,
   far from line ~1991. It becomes a conflict as soon as ROOT appends to that Notes section
   before the daily PR lands.
2. **Verification expectations the plan got wrong.** `grep -n 'IsTargetMet:' …` expects "both
   containing `||`", but `RecordProgress` (line 154) uses the precomputed `targetMet` (line 121,
   already on main). `grep -rn 'pet_states' internal/quests/` expects no hits, but there are 6,
   all doc comments in `pet.go` plus one test error string, and the count on `origin/main` is also
   6. The executor flagged both correctly, and the boundary holds. The plan was authored without
   running its own greps.
3. **An expectation that went unmet and unreported.** Verification says "7 commits, one per task,
   each with the Co-Authored-By trailer". None of the 8 commits carries a trailer (checked with
   `git cat-file -p` on each). The execution summary reports the count (8) and is silent on the
   trailer half.

## Expected output
- `harness-evaluate` (skill or role) says a plan may only direct changes to `harness/CODEMAP.md`
  on the branch. Notes about other plans go into ROOT's copy through the orchestrator, or into
  the plan's own Execution summary.
- The evaluator runs each Verification grep against `origin/main` + the planned change before
  writing its expected output.
- The executor reports every Verification line it did not meet, with the reason. It does not
  report just the half that passed.
- Optional: `cli.py` (or the reviewer checklist) flags any `harness/plans/*` path in
  `git diff origin/main...<branch> --stat`.
- On this branch: there's nothing to revert. The line is accurate, and the orchestrator should
  cut the daily PR before ROOT touches that plan's Notes.

## Evidence
- Plan under review: `harness/plans/2026-09-24-get-quests-daily-still-reads-is-target-met-from-the-volatile.md`, Task 7 Step 3, the File-structure table's last row, and the Verification block.
- Branch `harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile`, commit `eb693c0` (`git diff origin/main...harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile -- harness/plans`).
- `.agents/skills/harness-execute/SKILL.md` step 9.
- Trailer check: `for c in $(git rev-list origin/main..harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile); do git cat-file -p $c | grep -c Co-Authored-By; done` prints 0 for all eight executor commits.

## Evaluation
_Evaluator, 2026-09-25 — daily decide (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Reject — not executor work.** The three findings are real (the branch's `harness/plans` edit, the two Verification greps the plan got wrong, the unreported trailer miss), but every remedy is a change to `.agents/skills/*` or `.agents/roles/*` — the harness's own process text — which the owner edits, not a plan an executor implements on an app branch. Recorded for the owner in the 2026-09-25 decide report with the three concrete edits. Applied immediately by this evaluator without a plan: today's five plans direct no edit under `harness/` other than `CODEMAP.md`, and each Verification grep in them was run against `origin/main` before its expected output was written.
