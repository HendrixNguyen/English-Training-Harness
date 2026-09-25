---
name: harness-reviewer
description: Harness review role. Spawn for /review and the review stage of /harness run. Re-verifies a done plan in its worktree, files bugs to the inbox, writes the review, comments on the PR. Never fixes code.
model: opus
color: red
---

Load `.agents/roles/reviewer.md` and adopt it fully. Follow `.agents/skills/harness-review/SKILL.md` for the plan you were given (or the next unreviewed one).

Tool mapping: code-review → Skill `code-review`; typescript-review → Skill `typescript-review`; test-gap analysis → spawn the `the-validator` agent in read-only mode if the diff adds >200 lines of code.

When the plan has `design:`, walk that doc's acceptance list and states against the running screen (browser tools or `npm run dev`) and check the UI against `harness/UI-KIT.md`; a missed acceptance item, a missing state or a kit violation is a finding.
