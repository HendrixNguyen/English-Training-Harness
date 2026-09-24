# Agent Harness — Design

**Date:** 2026-09-22
**Status:** approved in brainstorm, pending user review of this document

## 1. Purpose

A tool-neutral, file-driven pipeline of four agent roles that continuously improves the Adaptive English Learning Platform: propose ideas → judge and plan them → implement → review. It runs manually stage-by-stage, or unattended via an orchestrator called by any scheduler. Its first workload is building the MVP from `project-base/1st-thinking-architecture-doc.md`.

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

Run folders are append-only. No cross-run deduplication; an idea may recur and the evaluator judges it fresh. `_run.md` summarises what the ideator read and proposed, and which inbox bugs it noticed and deliberately did not duplicate.

### 3.2 Plan — `harness/plans/<date>-<slug>.md`

```yaml
idea: harness/ideas/<run>/<slug>.md
status: draft | approved | executing | done | failed
branch: harness/<date>-<priority>-<slug>   # set by executor
worktree: .worktrees/<slug>                # set by executor
pr: https://github.com/<owner>/<repo>/pull/N   # set by executor, if remote exists
merged: false                              # set true by /harness merge
design: harness/designs/<slug>.md    # UI features only
```
Body is the `writing-plans` format: bite-sized tasks with verification steps. On completion the executor appends `## Execution summary` (built, deviations + why, verification evidence). On failure it appends `## Failure` (tried, blocker).

### 3.2b Blockers — the fast path for merge-blocking review findings

Most reviewer-filed bugs wait for the next ideation run, so bugs and features get ranked together (§4). A **blocker** is the exception: a finding that means the branch under review must not merge as it stands — data loss, a broken developer workflow, a security hole, a dishonest test.

A blocker is an ordinary bug idea with two extra constraints, enforced by `schema.py`:

```yaml
type: bug
priority: high                       # required for a blocker
blocks: harness/plans/<blocked>.md   # the plan whose branch it holds up
```

It changes three things:

1. **It skips the ideation queue.** The evaluator takes blockers first (`cli.py blockers`), because an unmerged branch is waiting on them.
2. **Its plan amends the branch, not `main`.** The evaluator sets `amends: <blocked plan>` on the fix plan; the executor then works in that plan's *existing* worktree and branch instead of creating new ones, so the fix lands in the history the reviewer blocked.
3. **It stops the merge mechanically.** `cli.py set <plan> merged=true` refuses while `blockers_for(plan)` is non-empty, and `/harness merge` checks the same thing. A blocker is resolved when its own fix plan reaches `done`.

`STATE.md` lists unresolved blockers in their own section above Inbox.

### 3.3 Review — `harness/reviews/<date>-<slug>.md`

```yaml
plan: harness/plans/....md
verdict: pass | pass-with-bugs | fail
bugs: [harness/ideas/_inbox/<slug>.md, ...]
```
Checks code against plan **and** plan against idea. Each bug becomes an idea file in `_inbox/` with `type: bug`, `source: reviewer`, referencing the plan. A `fail` verdict creates at least one `priority: high` bug.

**Who chooses the work.** The ideator only proposes features; the reviewer only files bugs; the **evaluator is the single point that ranks across both queues** (`next --stage evaluate` returns run ideas and `_inbox/` bugs together — blockers first, then priority, then inbox before runs). Bugs therefore never wait for an ideation run, and the ideator never moves or re-files them; it reads the inbox only to avoid proposing a known bug as a feature.

### 3.4 `STATE.md` and `CODEMAP.md`

`STATE.md` sections: Invalid, Inbox, Proposed, Selected, Planned (awaiting approval), Approved, Executing, Done (last 10), Failed. Regenerated at the end of every command.

`CODEMAP.md`: one paragraph per `backend/internal/<pkg>` and per Nuxt area — what it owns, public interface, key files. Read first by every role; updated by the executor when a package changes, corrected by the reviewer. This is the token-efficiency mechanism: CODEMAP → `rg` → read matched ranges only. Never `cat` a directory.

## 4. Roles

| Role | Skill | Reuses | Reads | Writes |
|---|---|---|---|---|
| Ideator | harness-ideate | remembering-conversations, web research | spec, CODEMAP, `_inbox/`, last 2 `_run.md` | run folder: ideas + `_run.md` |
| Evaluator | harness-evaluate | brainstorming (why), writing-plans, frontend-design (UI → designs/), code-review + systematic-debugging (bugs) | one idea or all `proposed` in a run | idea status/priority/reason, plan `draft`, optional design |
| Executor | harness-execute | executing-plans, test-driven-development, verification-before-completion, using-git-worktrees | one `approved` plan, CODEMAP | code in `.worktrees/<slug>`, pushed branch + Draft PR, plan status + summary, CODEMAP |
| Reviewer | harness-review | code-review, requesting-code-review, the-validator, typescript-review | plan, its idea, worktree + `main...harness/<slug>` diff | review file, `_inbox/` bugs, CODEMAP fixes, PR comment + ready state |

**Division of labour between executor and reviewer:**

The executor owns *does it work*; the reviewer owns *is it good*.

`done` is a strong claim: the executor must prove the branch builds, passes its whole suite from a clean shell, boots and serves one real request, and that **every command the plan, Makefile, README or CODEMAP tells a human to run actually works as documented**, including that destructive ones refuse when they should. A documented workflow that fails is a plan `failure` with a reproduction — never a workaround applied outside the repo, and never a note in the summary. The evidence goes in a *Runtime proof* subsection of `## Execution summary`.

The reviewer therefore does not spend its budget asking "does it run" — it re-runs the executor's evidence to confirm it, then reviews **quality**: design and boundaries, correctness on inputs nobody tried, performance and resource use, conventions and idiom, error handling, test honesty, documentation accuracy, security. If the executor's evidence does not reproduce, that is an **executor gate failure** — a blocker, plus an explicit statement in the review that the plan should not have been marked `done`.

**Boundaries:**
- Ideator never writes plans or code, and never moves, edits or ranks bugs.
- Evaluator never touches app code; rejection with a reason is a first-class outcome.
- Executor never changes a plan's intent. Small deviations are logged; an unfollowable plan goes to `failed`, not reinterpreted.
- Reviewer never fixes code. Bugs flow back through ideation.
- Nobody merges to `main`; only the human via `/harness merge`.

**Human ideas:** `/idea "<text>"` writes a `source: human` idea into the current run (or a new `manual` run folder) and immediately runs the evaluator on it.

**MVP bootstrap:** `/ideate --mvp` produces `type: mvp-slice` ideas with `order:` — store → auth → quests → pet → airouter → google → notify → frontend — instead of features.

## 5. Status transitions

| Transition | Owner |
|---|---|
| — → `proposed` | ideator, human (`/idea`), reviewer (into `_inbox/`) |
| `proposed` → `selected` / `rejected` (+ priority) | evaluator |
| `selected` → `planned`; plan created `draft` | evaluator |
| plan `draft` → `approved` | evaluator, automatically, for every bug, every `mvp-slice` and `priority: high` features (owner, 2026-09-24); human (`/approve`) for medium/low features |
| `approved` → `executing` → `done` / `failed` | executor |
| review written; bugs → `_inbox/`; blockers get `blocks:` | reviewer |
| blocker → fix plan with `amends:`, executed on the same branch | evaluator, then executor |
| `failed` → `approved` (after human edit) or idea `rejected` | human only |

## 6. Commands and orchestration

**Stage commands (manual mode):**

| Command | Effect |
|---|---|
| `/ideate [--mvp] [--count N]` | New run folder; default 5 ideas |
| `/idea "<text>"` | Human idea → evaluator |
| `/evaluate [<idea> \| --run <run>]` | One idea or all `proposed` in run |
| `/approve <plan>` | `draft → approved` for plans outside the auto-approve rule (medium/low features). |
| `/execute [<plan>]` | Default: highest-priority `approved` |
| `/review [<plan>]` | Default: oldest `done` without review |
| `/harness status` | Regenerate + print `STATE.md`, list stale worktrees |
| `/harness merge <plan>` | Human-only: merge branch into `main` (`--no-ff`), push `main`, remove worktree, delete branch, set `merged: true` |
| `/harness prune` | Remove worktrees of merged plans |
| `cli.py blockers [--plan P]` | List unresolved blockers; exit 1 if any (used by merge and by the orchestrator) |

**Orchestrator:** `/harness run [--stages a,b,c]`

```
1. validate harness/ → list malformed files, skip them
2. if _inbox non-empty OR no proposed ideas → ideate
3. if proposed ideas → evaluate all
4. approve any eligible draft the evaluator left (all bugs, mvp-slices, high features)
5. if approved plans → execute ONE (highest priority, lowest order)
6. if done plans lack review → review them
7. regenerate STATE.md; write harness/runs/<timestamp>.log
```

One execute per run bounds each scheduled tick to a reviewable diff. The orchestrator is idempotent: running with nothing to do exits cleanly and logs. Scheduling is external — a Claude Code routine or any cron calls `/harness run`.

## 7. Failure handling

| Failure | Handling |
|---|---|
| Executor blocked, or any *Definition of done* check fails (build, suite, boot, documented commands) | Plan `failed` + `## Failure` with a reproduction. Branch and worktree kept. Never auto-retried; surfaces in STATE.md until human acts. |
| Review `fail` | Plan stays `done`, unmerged; worktree kept for inspection; at least one **blocker** (§3.2b) filed against the plan, which mechanically prevents the merge. |
| Malformed frontmatter | Listed under Invalid in STATE.md, skipped. Never consumed silently. |
| No GitHub remote / `gh` not authenticated | Push and PR steps skipped, noted in execution summary; pipeline continues locally. |
| Concurrent executors | `harness/.lock` holds plan path; second executor exits. Lock older than 2h is stale. Separate worktrees mean a stale lock never corrupts another plan's files. |
| Empty/useless ideation | Evaluator bulk-rejects; run is still recorded as history. |

Deferred: vector/semantic search. Revisit when CODEMAP + `rg` stop being enough; the ideator may propose it.

## 8. Git and worktrees

The harness initialises the repo. `main` is the integration branch and the main checkout is never modified by any role.

**Executor:** for each plan, `git worktree add .worktrees/<slug> -b harness/<date>-<priority>-<slug> main`. All code changes, builds, and tests happen inside that worktree. Commits land on `harness/<slug>`. The plan's frontmatter records `branch:` and `worktree: .worktrees/<slug>`. The executor never merges and never deletes the worktree.

**Reviewer:** works inside the same worktree — runs the build and tests there, diffs `harness/<slug>` against `main`, and checks the result against plan and idea. The review file links branch, worktree, and the `git diff main...harness/<slug> --stat` summary.

**Pull request (GitHub):** when a plan reaches `done`, the executor pushes the branch and opens a **Draft** PR with `gh pr create --draft`, targeting `main`. Naming is fixed so PRs sort and filter by eye:

- Branch: `harness/<YYYY-MM-DD>-<priority>-<slug>` — e.g. `harness/2026-09-22-high-pet-unique-user`
- PR title: `[<YYYY-MM-DD>][<P1|P2|P3>] <Idea title>` — `P1` = high, `P2` = medium, `P3` = low; mvp-slices use `[MVP-<order>]` in place of the priority tag
- PR body: idea *Why* + *Expected output*, links to plan and idea files, execution summary; ends with the project's attribution line
- Labels: `harness`, `type: <feature|bug|mvp-slice>`, `priority: <high|medium|low>` (created on first use with `gh label create` if missing)

The plan's frontmatter records `pr:` (URL). If no GitHub remote is configured or `gh` is not authenticated, the executor skips push/PR and notes it in the execution summary; everything else works locally.

**Reviewer** posts its verdict as a PR comment (summary + link to the review file) and, on `pass`, marks the PR ready (`gh pr ready`). On `fail` the PR stays Draft.

**Human merge:** after a `pass` or `pass-with-bugs` review, the human merges — either in GitHub or via `/harness merge <plan>`, which merges `--no-ff` into `main`, pushes `main`, removes the worktree, deletes the branch, and sets `merged: true`. A `fail` review leaves worktree and PR in place. Merging is human-only in every mode, including auto-approved plans.

**Push permission:** the harness may push `harness/*` branches and open/update Draft PRs without asking. It never pushes `main` except inside `/harness merge`, which a human invoked.

**Cleanup:** `/harness status` lists worktrees whose plan is `done` + reviewed + merged as stale; `/harness prune` removes them. Worktrees for `failed` plans are kept until the plan is re-approved or its idea rejected.

`.worktrees/` is git-ignored.

## 9. Acceptance tests for the harness

1. `/ideate` against the spec alone → well-formed idea files, `## Why` tied to spec goals, `STATE.md` shows them Proposed.
2. `/idea "add UNIQUE(user_id) to pet_states in the DDL doc"` → evaluate → `/approve` → `/execute` → `/review` → `/harness merge`: every transition fires, a worktree is created and removed, a Draft PR titled `[2026-09-22][P<n>] …` is opened then marked ready by the reviewer, the change lands on `main` only via the merge command, review verdict `pass`.
3. `/harness run` with nothing to do → exits cleanly, writes a log, changes nothing.
4. `/ideate --mvp` → 8 ordered `mvp-slice` ideas; `/harness run` executes exactly the `order: 1` slice.

## 10. Build order

1. `AGENTS.md`, `CLAUDE.md` import, folder skeleton, `git init`
2. Artifact schemas + `harness-orchestrate` validation/STATE.md generation (shared by every command)
3. Ideator role + skill + `/ideate`, `/idea`
4. Evaluator role + skill + `/evaluate`, `/approve`
5. Executor role + skill + `/execute`
6. Reviewer role + skill + `/review`
7. `/harness run` orchestrator
8. Acceptance tests 1–3, then `/ideate --mvp`
