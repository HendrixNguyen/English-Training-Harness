# AGENTS.md

Canonical instructions for any coding agent (Claude Code, Codex, Gemini CLI, …) working in this repository. Tool-specific files (`CLAUDE.md`, `GEMINI.md`) import this file.

## What this repo is

An adaptive English-learning PWA (spec: `project-base/1st-thinking-architecture-doc.md`) built and evolved by an **agent harness** (design: `docs/superpowers/specs/2026-09-22-agent-harness-design.md`). App code lives in `frontend/` (Nuxt 3) and `backend/` (Go modular monolith) once the MVP slices land.

## The harness in one paragraph

Five roles in `.agents/roles/` — ideator, evaluator, designer, executor, reviewer — pass markdown artifacts through `harness/`. The **designer** (owner, 2026-09-25) is spawned by the evaluator for anything that touches `frontend/`: it researches the screen, clarifies expectations and writes `harness/designs/<slug>.md` inside `harness/UI-KIT.md` and the spec's §7 wireframes; executors build from that doc and reviewers check the screen against it. The ideator proposes features, the reviewer files bugs, and the **evaluator alone chooses and ranks across both**; the executor builds what was approved. **Standing priority (owner, 2026-09-23): the nine MVP slices are merged, so the MVP-first rule has expired. The evaluator now ranks the `_inbox` on user impact — data loss, security, and anything a real user hits on the happy path outrank internal tidiness. Blockers still jump the queue.** **Auto-approve (owner, 2026-09-24): the evaluator approves every bug plan, every `mvp-slice` plan and every `priority: high` feature plan as it writes them, so the executor picks them up without a human `/approve`; medium/low feature plans still wait for `/approve`. The owner's gate is the daily PR merge.** **Two queues, two caps (owner, 2026-09-25): the 10:00 bugfix run and the 14:00 feature run are separate capacity, so the decide run selects up to 5 bug plans *and* up to 5 feature plans per day, ranked as two independent lists — a bug never displaces a feature and a feature never displaces a bug. Feature slots go to the oldest `selected`-but-unplanned features first, so a feature selected on day N is planned by day N+1 instead of ageing behind the inbox; an empty feature slot is not given to a bug.** Status lives in each file's frontmatter; `harness/STATE.md` is a generated dashboard. All state changes go through `python3 tools/harness/cli.py` — never hand-edit frontmatter. Read `harness/CODEMAP.md` before exploring code.

**Session start:** every agent begins with `python3 tools/harness/cli.py context` — git, the CODEMAP index and the actionable harness state (read-only; it never rewrites `STATE.md`). Claude Code (`.claude/settings.json`), Gemini CLI (`.gemini/settings.json`) and Codex (`.codex/hooks.json`) inject it automatically through a `SessionStart` hook calling `context --hook`; any other agent runs it by hand before its first task.

**Daily cadence:** the harness runs unattended on five scheduled routines, defined tool-neutrally in `.agents/routines/` (README there has the table and the rules every run shares): 02:00 ideate → 06:00 decide (evaluator picks ≤5 bug plans and ≤5 feature plans) → 10:00 bugfix execute (`type: bug` only) → 14:00 feature execute (`type: feature` / `mvp-slice` only) → 20:00 review + the day's single code PR, all owner-local time (Asia/Saigon). A tool's scheduler is only an adapter whose prompt is "follow `.agents/routines/<name>.md` exactly"; change the routine file, not the adapter.

## Reading the spec

Three documents in `project-base/`, all canonical:

- `1st-thinking-architecture-doc.md` — the original whole-system spec: goals (§1), stack (§2), DDL (§3.2), Redis keys (§4), sequence flows (§5), AI router (§6), endpoint list (§7), deployment (§8).
- `Adaptive English Learning Platform - Backend Technical Specification.md` — extends it for the backend: **REST DTO contracts per endpoint (§6.1–6.4)**, security & token lifecycle (§7), **virtual-plant math and the hourly cron (§8)**, Railway checklist (§9). When planning or reviewing a backend slice, its §6 request/response shapes are the contract.
- `Adaptive English Learning Platform - Frontend Technical Specification.md` — extends it for the Nuxt PWA: service-worker caching (§3), Pinia stores (§4), API→UI mapping (§5), design system (§6), and five wireframes (§7). Frontend ideas, designs and plans start here.

Where the documents disagree, the backend/frontend specs win for their own layer; say so in the plan. The filenames contain spaces — quote them in shell.

All three were pasted from a rich-text editor: headings and symbols are backslash-escaped (`\#\# 7\.` is §7, `\+` is `+`), so search by content (`grep -n 'Core REST'`), not by `^## `. In the 1st-thinking doc the DDL and Go are escaped and de-indented too (treat its §6 Go as pseudocode); in the backend/frontend specs the fenced ```sql blocks are clean. The backend spec's DDL must stay identical to `backend/internal/store/migrations/0001_init.up.sql` (plus later migrations) — the executor's Definition of done for any store change includes that diff.

## Deployment

`deploy/README.md` is the runbook. Two images (`backend/Dockerfile`, `frontend/Dockerfile`) and one env contract (`deploy/.env.example`) deploy to either target: the free managed split today (Railway / Supabase / Upstash / Cloudflare Pages) or the owner's Dokploy server later (`deploy/compose.yml`). Agents own everything in the repo; the owner completes the runbook's checklist — accounts, secrets, the Google OAuth console, DNS — and agents never hold production secrets. `deploy/smoke-api.sh` / `deploy/smoke-web.sh` are the post-deploy checks. CD (registry push + deploy webhook) is parked in `harness/BACKLOG.md`.

## Rules every role follows

- Never modify the main checkout's app code; executors work in `.worktrees/<slug>` on `harness/*` branches.
- Never merge to `main`. Only a human runs `/harness merge`.
- A review finding that must be fixed before its branch can merge is a **blocker**: `type: bug`, `priority: high`, `blocks: <plan>`. Blockers skip the ideation queue and go straight to the evaluator; their plan sets `amends: <plan>` and lands on the same branch. `cli.py` refuses `merged=true` while any are unresolved.
- Pushing `harness/*` branches is allowed without asking. Pushing `main` is not. `production` is the only branch that ships and it moves **only by pull request** (owner, 2026-09-25): a `release/v<x.y.z>` PR (cut from `production`, `main` merged in, a `CHANGELOG.md` section) that the nightly ship routine opens and merges once its CI is green, or an owner `hotfix/*` PR. Never push `production` directly, never force it; back-merges `production → main` are PRs the owner merges.
- **One PR per day, not per plan (owner, 2026-09-23).** Executors push their branch and stop; they never open a PR. When the day's approved plans are all `done` and reviewed, the orchestrator cuts one integration branch `harness/daily-<date>`, merges each finished plan branch into it, and opens a single PR titled `<date> [<highest priority>] Daily: <n> fixes`. Per-plan branches still exist — they are what CI and the reviewer work on — but the owner sees and merges one PR a day.
- CI (`.github/workflows/ci.yml`) is the outer verification loop and must stay green: `backend-unit`, `backend-integration`, `frontend`, `docker-images` and `harness-tooling` run on every push to `main`, `production` or a `harness/**` / `release/**` / `hotfix/**` branch, and on pull requests when one exists. The push of a `harness/*` branch is what gates it: the reviewer reads that branch's run (`gh run list --branch <branch>`), a red or missing check on it is a review blocker, and `/harness merge` must not push `main` while it is red. Superseded runs are cancelled on branches, never on `main`. Never make a job pass by skipping, loosening or deleting a check — fix the code or the artifact it flagged. `.github/workflows/deploy.yml` runs only after CI passes on a push to `production` (the nightly ship routine `.agents/routines/daily-ship.md` releases into it). It ships the PWA (Cloudflare Pages) from that commit, smoke-checks both public URLs and tags it `v<x.y.z>` from `CHANGELOG.md` with a GitHub Release; the API ships from the same `production` push through Railway's own GitHub connection with Wait for CI (owner, 2026-09-25) — nothing deploys on a merge to `main`. Rollback is redeploying an older tag (`deploy/README.md` "Versions and rollback"). Its variables and secrets are the owner's (`deploy/README.md` "Ship from `production`").
- Token discipline: CODEMAP → `rg`/`grep -n` with tight patterns → read only matched ranges. Never `cat` a directory.
- Docker Compose in a worktree: every worktree's `backend/` shares the default project name `backend`, so `make up` in one worktree recreates another's containers. Always set a unique project — `COMPOSE_PROJECT_NAME=<slug>` in the scratch `backend/.env` (or `docker compose -p <slug>`) — alongside the port overrides, and `make down` the same project after.
- Tooling caveats on this machine: `rg` is **not installed** — use `grep -n` / `grep -c`. `timeout` / `gtimeout` are **not installed** either — bound long commands with the tool's own flag (`go test -timeout`, `curl --max-time`, `docker compose --wait-timeout`). An output-rewriting proxy (rtk) wraps shell commands and can mangle `ls` output — capture directory names with shell globs, `find`, or `python3`, never `$(ls …)`.
- Files under `.agents/` and `harness/` are tool-neutral: say "spawn the evaluator role" or "load skill harness-evaluate", never name a specific tool.
- This project's remote is **GitHub** (`gh`, pull requests).

## Commands (Claude Code adapter)

`/ideate [--mvp]`, `/idea "<text>"`, `/evaluate [<idea>|--run <run>]`, `/design <idea> [<plan>]`, `/approve <plan>`, `/execute [<plan>]`, `/review [<plan>]`, `/harness status|run|merge|prune`.

## Harness CLI quick reference

```
python3 tools/harness/cli.py validate                 # exit 1 on malformed artifacts
python3 tools/harness/cli.py state                    # regenerate + print STATE.md
python3 tools/harness/cli.py context [--hook]          # read-only session briefing (--hook: SessionStart JSON)
python3 tools/harness/cli.py new-run [--mvp]          # -> harness/ideas/<date>-run-NN/
python3 tools/harness/cli.py new-idea --run DIR --title T --type feature|bug|mvp-slice --source ideator|human|reviewer [--order N]
python3 tools/harness/cli.py new-plan --idea FILE     # -> harness/plans/<date>-<slug>.md (status draft)
python3 tools/harness/cli.py new-review --plan FILE --verdict pass|pass-with-bugs|fail
python3 tools/harness/cli.py set FILE key=value ...   # frontmatter update with validation
python3 tools/harness/cli.py next --stage evaluate|execute|review
python3 tools/harness/cli.py blockers [--plan FILE]     # exit 1 if any unresolved
python3 tools/harness/cli.py lock PLAN | unlock [PLAN]   # per-plan; different plans never collide
python3 tools/harness/cli.py slug "Title"
python3 tools/harness/cli.py stale-worktrees
```
