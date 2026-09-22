# Agent Harness — Design

**Date:** 2026-09-22
**Status:** approved in brainstorm, pending user review of this document

## 1. Purpose

A tool-neutral, file-driven pipeline of four agent roles that continuously improves the Adaptive English Learning Platform: propose ideas → judge and plan them → implement → review. It runs manually stage-by-stage, or unattended via an orchestrator called by any scheduler. Its first workload is building the MVP from `1st-thinking-architecture-doc.md`.

Non-goals: merging code to `main` (always human), vector/semantic code search (deferred; see §7), adapters for tools other than Claude Code (follow-ups).

## 2. Repository layout

```
Learning-English-Project/
├── AGENTS.md                        # canonical project instructions, tool-neutral
├── CLAUDE.md / GEMINI.md            # one-line imports of AGENTS.md + tool-specific notes
├── .agents/                         # the portable harness
│   ├── roles/                       # ideator.md evaluator.md executor.md reviewer.md
│   └── skills/
│       ├── harness-ideate/SKILL.md
│       ├── harness-evaluate/SKILL.md
│       ├── harness-execute/SKILL.md
│       ├── harness-review/SKILL.md
│       └── harness-orchestrate/SKILL.md
├── .claude/                         # thin Claude Code adapter
│   ├── skills -> ../.agents/skills
│   ├── agents/<role>.md             # frontmatter + "you are .agents/roles/<role>.md"
│   └── commands/*.md                # /ideate /idea /evaluate /approve /execute /review /harness
├── harness/                         # runtime data, all markdown
│   ├── ideas/<YYYY-MM-DD>-run-<NN>/  <slug>.md, _run.md
│   ├── ideas/_inbox/                # reviewer-created bugs awaiting next ideation
│   ├── plans/<YYYY-MM-DD>-<slug>.md
│   ├── designs/<slug>.md            # UI features only
│   ├── reviews/<YYYY-MM-DD>-<slug>.md
│   ├── runs/<timestamp>.log         # orchestrator logs
│   ├── CODEMAP.md                   # one paragraph per package/module
│   ├── STATE.md                     # derived dashboard, never hand-edited
│   └── .lock                        # executor mutex
├── docs/                            # spec + design docs
├── frontend/                        # Nuxt 3 PWA (created by MVP slices)
└── backend/                         # Go modular monolith (created by MVP slices)
```

**Portability rule:** files under `.agents/` and `harness/` never name a tool-specific mechanism (`Agent`, `Skill`, `AskUserQuestion`, …). They say "spawn the evaluator role", "load skill X", "ask the user". Each adapter maps those phrases to its own tools.

## 3. Artifacts

All artifacts are markdown with YAML frontmatter. `status` in frontmatter is the source of truth; `STATE.md` is regenerated from a scan.

### 3.1 Idea — `harness/ideas/<run>/<slug>.md`

```yaml
type: feature | bug | mvp-slice
status: proposed | selected | rejected | planned
source: ideator | human | reviewer
run: 2026-09-22-run-01
priority: high | medium | low        # set by evaluator
order: 3                             # mvp-slice only: dependency order
plan: harness/plans/....md           # set when planned
rejected_reason: "..."               # set when rejected
```
Body: `# Title`, `## Why` (tied to spec goals — retention, 30 min/day, CEFR progression), `## Expected output` (user-visible + technical), `## Evidence` (spec sections, research links, prior runs).

Run folders are append-only. No cross-run deduplication; an idea may recur and the evaluator judges it fresh. `_run.md` summarises what the ideator read, proposed, and which inbox bugs it swept in.

### 3.2 Plan — `harness/plans/<date>-<slug>.md`

```yaml
idea: harness/ideas/<run>/<slug>.md
status: draft | approved | executing | done | failed
branch: harness/<slug>               # set by executor
design: harness/designs/<slug>.md    # UI features only
```
Body is the `writing-plans` format: bite-sized tasks with verification steps. On completion the executor appends `## Execution summary` (built, deviations + why, verification evidence). On failure it appends `## Failure` (tried, blocker).

### 3.3 Review — `harness/reviews/<date>-<slug>.md`

```yaml
plan: harness/plans/....md
verdict: pass | pass-with-bugs | fail
bugs: [harness/ideas/_inbox/<slug>.md, ...]
```
Checks code against plan **and** plan against idea. Each bug becomes an idea file in `_inbox/` with `type: bug`, `source: reviewer`, referencing the plan. A `fail` verdict creates at least one `priority: high` bug.

### 3.4 `STATE.md` and `CODEMAP.md`

`STATE.md` sections: Invalid, Inbox, Proposed, Selected, Planned (awaiting approval), Approved, Executing, Done (last 10), Failed. Regenerated at the end of every command.

`CODEMAP.md`: one paragraph per `backend/internal/<pkg>` and per Nuxt area — what it owns, public interface, key files. Read first by every role; updated by the executor when a package changes, corrected by the reviewer. This is the token-efficiency mechanism: CODEMAP → `rg` → read matched ranges only. Never `cat` a directory.

## 4. Roles

| Role | Skill | Reuses | Reads | Writes |
|---|---|---|---|---|
| Ideator | harness-ideate | remembering-conversations, web research | spec, CODEMAP, `_inbox/`, last 2 `_run.md` | run folder: ideas + `_run.md` |
| Evaluator | harness-evaluate | brainstorming (why), writing-plans, frontend-design (UI → designs/), code-review + systematic-debugging (bugs) | one idea or all `proposed` in a run | idea status/priority/reason, plan `draft`, optional design |
| Executor | harness-execute | executing-plans, test-driven-development, verification-before-completion, using-git-worktrees | one `approved` plan, CODEMAP | code on branch, plan status + summary, CODEMAP |
| Reviewer | harness-review | code-review, requesting-code-review, the-validator, typescript-review | plan, its idea, branch diff | review file, `_inbox/` bugs, CODEMAP fixes |

**Boundaries:**
- Ideator never writes plans or code.
- Evaluator never touches app code; rejection with a reason is a first-class outcome.
- Executor never changes a plan's intent. Small deviations are logged; an unfollowable plan goes to `failed`, not reinterpreted.
- Reviewer never fixes code. Bugs flow back through ideation.
- Nobody merges to `main`.

**Human ideas:** `/idea "<text>"` writes a `source: human` idea into the current run (or a new `manual` run folder) and immediately runs the evaluator on it.

**MVP bootstrap:** `/ideate --mvp` produces `type: mvp-slice` ideas with `order:` — store → auth → quests → pet → airouter → google → notify → frontend — instead of features.

## 5. Status transitions

| Transition | Owner |
|---|---|
| — → `proposed` | ideator, human (`/idea`), reviewer (into `_inbox/`) |
| `proposed` → `selected` / `rejected` (+ priority) | evaluator |
| `selected` → `planned`; plan created `draft` | evaluator |
| plan `draft` → `approved` | human (`/approve`); orchestrator `--auto-approve` for `priority: high` bugs and `mvp-slice` only |
| `approved` → `executing` → `done` / `failed` | executor |
| review written; bugs → `_inbox/` | reviewer |
| `failed` → `approved` (after human edit) or idea `rejected` | human only |

## 6. Commands and orchestration

**Stage commands (manual mode):**

| Command | Effect |
|---|---|
| `/ideate [--mvp] [--count N]` | New run folder; default 5 ideas |
| `/idea "<text>"` | Human idea → evaluator |
| `/evaluate [<idea> \| --run <run>]` | One idea or all `proposed` in run |
| `/approve <plan>` | `draft → approved`. The safety boundary. |
| `/execute [<plan>]` | Default: highest-priority `approved` |
| `/review [<plan>]` | Default: oldest `done` without review |
| `/harness status` | Regenerate + print `STATE.md` |

**Orchestrator:** `/harness run [--auto-approve] [--stages a,b,c]`

```
1. validate harness/ → list malformed files, skip them
2. if _inbox non-empty OR no proposed ideas → ideate
3. if proposed ideas → evaluate all
4. if --auto-approve → approve eligible drafts (high bugs, mvp-slices)
5. if approved plans → execute ONE (highest priority, lowest order)
6. if done plans lack review → review them
7. regenerate STATE.md; write harness/runs/<timestamp>.log
```

One execute per run bounds each scheduled tick to a reviewable diff. The orchestrator is idempotent: running with nothing to do exits cleanly and logs. Scheduling is external — a Claude Code routine or any cron calls `/harness run`.

## 7. Failure handling

| Failure | Handling |
|---|---|
| Executor blocked | Plan `failed` + `## Failure`. Branch kept. Never auto-retried; surfaces in STATE.md until human acts. |
| Review `fail` | Plan stays `done`; high-priority bug in `_inbox/` referencing the plan. |
| Malformed frontmatter | Listed under Invalid in STATE.md, skipped. Never consumed silently. |
| Concurrent executors | `harness/.lock` holds plan path; second executor exits. Lock older than 2h is stale. |
| Empty/useless ideation | Evaluator bulk-rejects; run is still recorded as history. |

Deferred: vector/semantic search. Revisit when CODEMAP + `rg` stop being enough; the ideator may propose it.

## 8. Git

The harness initialises the repo. Executor works on `harness/<slug>` branches and never merges. Reviews link the branch. Merging is a human action after review, in every mode.

## 9. Acceptance tests for the harness

1. `/ideate` against the spec alone → well-formed idea files, `## Why` tied to spec goals, `STATE.md` shows them Proposed.
2. `/idea "add UNIQUE(user_id) to pet_states in the DDL doc"` → evaluate → `/approve` → `/execute` → `/review`: every transition fires, artifacts link correctly, review verdict `pass`.
3. `/harness run` with nothing to do → exits cleanly, writes a log, changes nothing.
4. `/ideate --mvp` → 8 ordered `mvp-slice` ideas; `/harness run --auto-approve` executes exactly the `order: 1` slice.

## 10. Build order

1. `AGENTS.md`, `CLAUDE.md` import, folder skeleton, `git init`
2. Artifact schemas + `harness-orchestrate` validation/STATE.md generation (shared by every command)
3. Ideator role + skill + `/ideate`, `/idea`
4. Evaluator role + skill + `/evaluate`, `/approve`
5. Executor role + skill + `/execute`
6. Reviewer role + skill + `/review`
7. `/harness run` orchestrator
8. Acceptance tests 1–3, then `/ideate --mvp`
