# AGENTS.md

Canonical instructions for any coding agent (Claude Code, Codex, Gemini CLI, …) working in this repository. Tool-specific files (`CLAUDE.md`, `GEMINI.md`) import this file.

## What this repo is

An adaptive English-learning PWA (spec: `project-base/1st-thinking-architecture-doc.md`) built and evolved by an **agent harness** (design: `docs/superpowers/specs/2026-09-22-agent-harness-design.md`). App code lives in `frontend/` (Nuxt 3) and `backend/` (Go modular monolith) once the MVP slices land.

## The harness in one paragraph

Four roles in `.agents/roles/` — ideator, evaluator, executor, reviewer — pass markdown artifacts through `harness/`. The ideator proposes features, the reviewer files bugs, and the **evaluator alone chooses and ranks across both**; the executor builds what was approved. **Standing priority (owner, 2026-09-22): finish the MVP slices in `order` first; inbox bugs wait unless they are blockers.** Status lives in each file's frontmatter; `harness/STATE.md` is a generated dashboard. All state changes go through `python3 tools/harness/cli.py` — never hand-edit frontmatter. Read `harness/CODEMAP.md` before exploring code.

## Reading the spec

Three documents in `project-base/`, all canonical:

- `1st-thinking-architecture-doc.md` — the original whole-system spec: goals (§1), stack (§2), DDL (§3.2), Redis keys (§4), sequence flows (§5), AI router (§6), endpoint list (§7), deployment (§8).
- `Adaptive English Learning Platform - Backend Technical Specification.md` — extends it for the backend: **REST DTO contracts per endpoint (§6.1–6.4)**, security & token lifecycle (§7), **virtual-plant math and the hourly cron (§8)**, Railway checklist (§9). When planning or reviewing a backend slice, its §6 request/response shapes are the contract.
- `Adaptive English Learning Platform - Frontend Technical Specification.md` — extends it for the Nuxt PWA: service-worker caching (§3), Pinia stores (§4), API→UI mapping (§5), design system (§6), and five wireframes (§7). Frontend ideas, designs and plans start here.

Where the documents disagree, the backend/frontend specs win for their own layer; say so in the plan. The filenames contain spaces — quote them in shell.

All three were pasted from a rich-text editor: headings and symbols are backslash-escaped (`\#\# 7\.` is §7, `\+` is `+`), so search by content (`grep -n 'Core REST'`), not by `^## `. In the 1st-thinking doc the DDL and Go are escaped and de-indented too (treat its §6 Go as pseudocode); in the backend/frontend specs the fenced ```sql blocks are clean. The backend spec's DDL must stay identical to `backend/internal/store/migrations/0001_init.up.sql` (plus later migrations) — the executor's Definition of done for any store change includes that diff.

## Rules every role follows

- Never modify the main checkout's app code; executors work in `.worktrees/<slug>` on `harness/*` branches.
- Never merge to `main`. Only a human runs `/harness merge`.
- A review finding that must be fixed before its branch can merge is a **blocker**: `type: bug`, `priority: high`, `blocks: <plan>`. Blockers skip the ideation queue and go straight to the evaluator; their plan sets `amends: <plan>` and lands on the same branch. `cli.py` refuses `merged=true` while any are unresolved.
- Pushing `harness/*` branches and opening Draft PRs is allowed without asking. Pushing `main` is not.
- CI (`.github/workflows/ci.yml`) is the outer verification loop and must stay green: `backend-unit`, `backend-integration` and `harness-tooling` run on every push to `main` or a `harness/**` branch, and on pull requests when one exists. The push of a `harness/*` branch is what gates it: the reviewer reads that branch's run (`gh run list --branch <branch>`), a red or missing check on it is a review blocker, and `/harness merge` must not push `main` while it is red. Superseded runs are cancelled on branches, never on `main`. Never make a job pass by skipping, loosening or deleting a check — fix the code or the artifact it flagged.
- Token discipline: CODEMAP → `rg`/`grep -n` with tight patterns → read only matched ranges. Never `cat` a directory.
- Docker Compose in a worktree: every worktree's `backend/` shares the default project name `backend`, so `make up` in one worktree recreates another's containers. Always set a unique project — `COMPOSE_PROJECT_NAME=<slug>` in the scratch `backend/.env` (or `docker compose -p <slug>`) — alongside the port overrides, and `make down` the same project after.
- Tooling caveats on this machine: `rg` is **not installed** — use `grep -n` / `grep -c`. `timeout` / `gtimeout` are **not installed** either — bound long commands with the tool's own flag (`go test -timeout`, `curl --max-time`, `docker compose --wait-timeout`). An output-rewriting proxy (rtk) wraps shell commands and can mangle `ls` output — capture directory names with shell globs, `find`, or `python3`, never `$(ls …)`.
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
python3 tools/harness/cli.py blockers [--plan FILE]     # exit 1 if any unresolved
python3 tools/harness/cli.py lock PLAN | unlock
python3 tools/harness/cli.py slug "Title"
python3 tools/harness/cli.py stale-worktrees
```
