# AGENTS.md

Canonical instructions for any coding agent (Claude Code, Codex, Gemini CLI, …) working in this repository. Tool-specific files (`CLAUDE.md`, `GEMINI.md`) import this file.

## What this repo is

An adaptive English-learning PWA (spec: `1st-thinking-architecture-doc.md`) built and evolved by an **agent harness** (design: `docs/superpowers/specs/2026-09-22-agent-harness-design.md`). App code lives in `frontend/` (Nuxt 3) and `backend/` (Go modular monolith) once the MVP slices land.

## The harness in one paragraph

Four roles in `.agents/roles/` — ideator, evaluator, executor, reviewer — pass markdown artifacts through `harness/`. Status lives in each file's frontmatter; `harness/STATE.md` is a generated dashboard. All state changes go through `python3 tools/harness/cli.py` — never hand-edit frontmatter. Read `harness/CODEMAP.md` before exploring code.

## Reading the spec

`1st-thinking-architecture-doc.md` was pasted from a rich-text editor: headings and symbols are backslash-escaped (`\#\# 7\.` is §7, `\+` is `+`) and Go code lost its indentation. §7 (REST endpoints) and §8 (deployment) exist — search with `rg -n '7\\\. Core REST'` or by content, not by `^## `. Treat §6.2 Go as pseudocode.

## Rules every role follows

- Never modify the main checkout's app code; executors work in `.worktrees/<slug>` on `harness/*` branches.
- Never merge to `main`. Only a human runs `/harness merge`.
- Pushing `harness/*` branches and opening Draft PRs is allowed without asking. Pushing `main` is not.
- Token discipline: CODEMAP → `rg` with tight patterns → read only matched ranges. Never `cat` a directory.
- Files under `.agents/` and `harness/` are tool-neutral: say "spawn the evaluator role" or "load skill harness-evaluate", never name a specific tool.
- This project's remote is **GitHub** (`gh`, pull requests).

## Commands (Claude Code adapter)

`/ideate [--mvp]`, `/idea "<text>"`, `/evaluate [<idea>|--run <run>]`, `/approve <plan>`, `/execute [<plan>]`, `/review [<plan>]`, `/harness status|run|merge|prune`.

## Harness CLI quick reference

```
python3 tools/harness/cli.py validate                 # exit 1 on malformed artifacts
python3 tools/harness/cli.py state                    # regenerate + print STATE.md
python3 tools/harness/cli.py new-run [--mvp]          # -> harness/ideas/<date>-run-NN/
python3 tools/harness/cli.py new-idea --run DIR --title T --type feature|bug|mvp-slice --source ideator|human|reviewer [--order N]
python3 tools/harness/cli.py new-plan --idea FILE     # -> harness/plans/<date>-<slug>.md (status draft)
python3 tools/harness/cli.py new-review --plan FILE --verdict pass|pass-with-bugs|fail
python3 tools/harness/cli.py set FILE key=value ...   # frontmatter update with validation
python3 tools/harness/cli.py next --stage evaluate|execute|review
python3 tools/harness/cli.py lock PLAN | unlock
python3 tools/harness/cli.py slug "Title"
python3 tools/harness/cli.py stale-worktrees
```
