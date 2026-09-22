---
name: harness-evaluate
description: Evaluate one idea or every proposed idea in a run — reject with reason or select with priority and write a draft plan (and design doc for UI). Use for /evaluate, /idea, and the evaluate stage of /harness run.
---

# harness-evaluate

Adopt `.agents/roles/evaluator.md`. Input: one idea path, or `--run <dir>` for all `proposed` ideas in it, or nothing (then use `python3 tools/harness/cli.py next --stage evaluate --all` — this includes the reviewer's `_inbox/` bugs; you rank features and bugs together, per the role's *Choosing the work*).

## Blockers first

`python3 tools/harness/cli.py blockers` lists bugs with a `blocks:` field whose fix is not yet done — review findings holding up an unmerged branch. They jump the queue: evaluate them before anything else, and never reject one without saying why the branch is safe to merge without it. A blocker's plan **amends the branch under review rather than starting a new one**: after `new-plan`, set `amends=<the blocked plan>` on it, and write its tasks as edits to files that already exist on that branch. Its `## Verification` must re-run the evidence from the review that found it, so the fix is proven, not asserted.

## Per idea

1. `python3 tools/harness/cli.py validate` — stop on failure.
2. Read the idea file, `harness/CODEMAP.md`, and the spec sections it cites. Read nothing else unless the idea's evidence points there.
3. Apply the brainstorming skill's questioning to the *Why* — but answer the questions yourself from spec and evidence; do not ask the user unless the idea is a `source: human` idea and genuinely ambiguous.
4. Append `## Evaluation` to the idea body (verdict, reasoning, dependencies, priority rationale).
5. **Reject:** `python3 tools/harness/cli.py set <idea> status=rejected rejected_reason="<one sentence>"`. Done with this idea.
6. **Select:** `python3 tools/harness/cli.py set <idea> status=selected priority=<high|medium|low>`.
7. If the idea involves UI: write `harness/designs/<slug>.md` following the frontend-design skill. Keep it to layout, states, component list, and token choices — no code.
8. If `type: bug`: apply systematic-debugging to locate root cause in the worktree-free main checkout (read-only). Record it in `## Evaluation`.
9. `PLAN=$(python3 tools/harness/cli.py new-plan --idea <idea>)`. Fill the plan body using the writing-plans skill. If a design exists: `python3 tools/harness/cli.py set $PLAN design=harness/designs/<slug>.md`.
10. `python3 tools/harness/cli.py validate && python3 tools/harness/cli.py state`; commit `harness/` with message `harness: evaluate <slug>`.

## Report
List each idea → verdict (+ priority / plan path or rejected reason). Remind the user that drafts need `/approve <plan>`.
