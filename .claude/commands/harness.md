---
description: Harness orchestrator — status | run [--auto-approve] [--stages …] | merge <plan> | prune
argument-hint: status | run [--auto-approve] | merge <plan-file> | prune
---

Load `.agents/skills/harness-orchestrate/SKILL.md` and perform the subcommand in `$ARGUMENTS`.

Role mapping for `run`: ideator → spawn agent `harness-ideator`; evaluator → `harness-evaluator`; executor → `harness-executor`; reviewer → `harness-reviewer`. Run stages sequentially — each depends on the previous one's files.

`merge` is human-only: if this command is being executed by a scheduler or another agent rather than the user, refuse and explain.
