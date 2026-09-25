---
name: harness-design
description: Design a learner-facing screen or flow for a UI idea or frontend bug — research, clarify expectations, then write harness/designs/<slug>.md in the project's UI kit, following the spec wireframes. Use when the evaluator plans anything that touches frontend/, or on /design.
---

# harness-design

Adopt `.agents/roles/designer.md`. Input: an idea path (feature or inbox bug) whose expected output touches `frontend/`, and optionally an existing plan path to attach the design to.

## Procedure

1. `python3 tools/harness/cli.py validate` — stop on failure.
2. Read, in this order: the idea (and its `## Evaluation`), `harness/UI-KIT.md`, the frontend spec §6 and the §7 wireframe for this screen, `harness/designs/frontend-shell.md`, the existing design doc for the screen if one exists, and the current `frontend/pages` / `frontend/components` files the screen uses. Read nothing else unless the idea's evidence points there.
3. **Research** (template §0): write the learner's job, the moment that earns the next minute, what today does wrong (look at the live app or `npm run dev` in a worktree when the idea is about something visible), and the open questions. Answer them from spec and evidence. If a *human* idea is ambiguous in a way that changes the screen, ask the owner once, batched.
4. **Clarify expectations** by writing the acceptance list the reviewer will check: 5–10 observable statements ("after the last answer the button enables within 100 ms", "an offline learner sees the cached passage and a banner").
5. **Design** with the frontend-design skill's craft guidance, inside the kit: layout, states, interactions, components, copy, motion. Every token and component must exist in `harness/UI-KIT.md`; anything new goes under "Kit additions" with a reason. Keep the spec wireframe's structure unless the research says it fails the learner — then say why.
6. Optional, when a Figma connection is available: draw the main state with `figma-generate-design` and link the frame in the doc. The markdown doc stays the contract.
7. Write `harness/designs/<slug>.md` from `.agents/templates/design.md` (`slug` = the idea's slug). Under ~250 lines.
8. Self-critique (template §8): name the trade-offs and the places an executor could go wrong.
9. `python3 tools/harness/cli.py validate`; if a plan path was given, `python3 tools/harness/cli.py set <plan> design=harness/designs/<slug>.md`. Commit `harness/designs/<slug>.md` with `harness: design <slug>`.

## Report
The design path, the acceptance list, kit additions proposed (if any), and the open question you asked or the assumption you made instead.
