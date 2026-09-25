---
name: harness-evaluate
description: Evaluate one idea or every proposed idea in a run — reject with reason or select with priority and write a plan (and design doc for UI), auto-approving bugs, mvp-slices and high features. Use for /evaluate, /idea, and the evaluate stage of /harness run.
---

# harness-evaluate

Adopt `.agents/roles/evaluator.md`. Input: one idea path, or `--run <dir>` for all `proposed` ideas in it, or nothing (then use `python3 tools/harness/cli.py next --stage evaluate --all` — this includes the reviewer's `_inbox/` bugs; you rank features and bugs together, per the role's *Choosing the work*).

## Two queues, two caps (owner, 2026-09-25)

Bugs (`_inbox/`) and features (run folders) are ranked as **two separate lists**, each capped at **5 plans per day** — the 10:00 bugfix run and the 14:00 feature run are separate capacity, so a bug never costs a feature its slot and an empty feature slot is never given to a bug. Within the feature list, an idea that is already `selected` but has no plan outranks any new `proposed` idea: `next --stage evaluate --all` lists only `proposed` items, so find the aged ones with `grep -l '^status: selected' harness/ideas/*/*.md | xargs grep -l '^type: feature'` (also under *Selected* in `harness/STATE.md`) and plan them first, oldest run folder first. A feature selected on day N therefore gets its plan by day N+1.

## Two queues, two caps (owner, 2026-09-25)

Bugs (`_inbox/`) and features (run folders) are ranked as **two separate lists**, each capped at **5 plans per day** — the 10:00 bugfix run and the 14:00 feature run are separate capacity, so a bug never costs a feature its slot and an empty feature slot is never given to a bug. Within the feature list, an idea that is already `selected` but has no plan outranks any new `proposed` idea: `next --stage evaluate --all` lists only `proposed` items, so find the aged ones with `grep -l '^status: selected' harness/ideas/*/*.md | xargs grep -l '^type: feature'` (also under *Selected* in `harness/STATE.md`) and plan them first, oldest run folder first. A feature selected on day N therefore gets its plan by day N+1.

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
10. **Auto-approve** (owner, 2026-09-24 — always on): if the idea is `type: bug` (any priority), `type: mvp-slice`, or `type: feature` with `priority: high`, run `python3 tools/harness/cli.py set $PLAN status=approved` so the executor can pick it up. Leave medium/low feature plans `draft` for the owner's `/approve`. Only approve a plan whose body is complete — tasks, paths, tests and `## Verification` filled in; an unfinished plan stays `draft`.
11. `python3 tools/harness/cli.py validate && python3 tools/harness/cli.py state`; commit `harness/` with message `harness: evaluate <slug>`.

## Report
List each idea → verdict (+ priority / plan path and `approved` or `draft`, or rejected reason). Remind the user that the remaining drafts (medium/low features) need `/approve <plan>`.
