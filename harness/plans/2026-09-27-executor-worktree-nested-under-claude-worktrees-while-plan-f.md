---
idea: harness/ideas/_inbox/executor-worktree-nested-under-claude-worktrees-while-plan-f.md
status: done
priority: low
merged: true
branch: harness/2026-09-27-low-executor-worktree-nested-under-claude-worktrees-while-plan-f
worktree: .worktrees/executor-worktree-nested-under-claude-worktrees-while-plan-f
pr: "https://github.com/HendrixNguyen/English-Training-Harness/pull/53"
---
# `stale-worktrees` reads `git worktree list`, the executor records the real worktree path, and amend re-reviews count for the amended plan — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Bug team — ticket **B5** of 2026-09-27. **Estimate:** 2.5 h. **Branch:** `harness/2026-09-27-low-executor-worktree-nested-under-claude-worktrees-while-plan-f`.

**Idea:** `harness/ideas/_inbox/executor-worktree-nested-under-claude-worktrees-while-plan-f.md` (primary). **Folds:** `harness/ideas/_inbox/next-stage-review-re-queues-merged-amend-plans-whose-review-.md` — Task 3. Harness tooling only; **no app code, no design doc**.

**Goal:** `python3 tools/harness/cli.py stale-worktrees` lists every worktree of this repo whose branch belongs to a `merged: true` plan, wherever git put it (today it prints nothing while five merged branches sit in `.claude/worktrees/*/.worktrees/` and the main checkout's `.worktrees/`); the executor's rules say the recorded `worktree:` is always the path git reports; and `next --stage review` / `STATE.md` stop re-queuing an amend plan whose re-review was filed under its parent, through an explicit `covers:` list on the review.

**Root causes (evaluator, 2026-09-27):**
- `tools/harness/cli.py:272-277` `cmd_stale_worktrees` globs `pathlib.Path(".worktrees")` relative to the cwd and matches the basename of each merged plan's `worktree:`. A worktree created anywhere else — the sandbox's nested `.claude/worktrees/<session>/.worktrees/<slug>`, or the main checkout's `.worktrees/` when the command runs from another worktree — is never listed, so `/harness prune` (harness-orchestrate → `prune` = `git worktree remove` over this list) never frees the branch. No test covers the command.
- `.claude/skills/harness-execute/SKILL.md:17-18` (byte-identical copy at `.agents/skills/harness-execute/SKILL.md`) and `.agents/roles/executor.md:6` fix `WT=.worktrees/$SLUG` and say nothing about recording a different real path, which is how a 2026-09-25 execution summary claimed a path its frontmatter never held.
- `tools/harness/scan.py:27` `ScanResult.reviews_for` matches only `fm["plan"] == plan_rel`; `cli.py:211` (`next --stage review`) and `state.py:41` (the `(unreviewed)` marker) both use it. A re-review of a parent plan after its amend plans land (`harness/reviews/2026-09-22-quests-…-2.md`, `…-store-…-rereview.md`) therefore never covers the amend plans. The four 2026-09-22 cases were silenced by hand-filed 2026-09-25 stub reviews; the rule is still wrong and the next amend re-review re-queues again.

**Rules chosen:**
1. *Stale worktree* = an entry of `git worktree list --porcelain` (other than the first, the main worktree, and never the current checkout or one of its ancestors) whose `branch refs/heads/<b>` equals the `branch:` of a `merged: true` plan. Basename-of-`worktree:` matching survives **only** for merged plans that recorded no `branch:` (older plans). Output is the absolute path git reports, one per line, sorted, so `prune` works from any cwd. When `git` is missing or fails (the unit tests run in a tempdir with no repo), fall back to today's `.worktrees/` glob.
2. *Recording rule* (one sentence, executor skill + role): if `.worktrees/<slug>` cannot be created, create the worktree where you can and record the path git reports — `worktree=` in frontmatter and the execution summary must agree with `git worktree list`.
3. *Amend coverage* = explicit. A review may carry `covers: [<plan>, …]`; `reviews_for(p)` returns reviews with `plan == p` **or** `p in covers`. `new-review --covers <plan…>` writes it; the reviewer passes it when a re-review of a parent re-verifies amend plans. No date heuristic: plans record no `done` date, and the existing data shows a parent review filed *before* an amend (`2026-09-25-parseroadmap-…` review vs the 2026-09-26 amend plan `…-parseroadmap-day-sum-rejection-…`) that must **not** be swallowed — that plan stays in the review queue, correctly. The two 2026-09-22 re-reviews are not back-filled on this branch (editing `harness/reviews/*` on a code branch conflicts with the daily review PR); the 2026-09-25 stubs already satisfy the queue.

## Global Constraints
- Work in `.worktrees/<slug>` (if the sandbox refuses it, apply rule 2 to yourself). Python stdlib only; run tests from the repo root: `python3 -m unittest discover -s tools/harness/tests -v`. `rg`/`timeout` not installed.
- Never edit `harness/plans/*`, `harness/reviews/*` or `harness/ideas/*` on the branch; the only `harness/` file this branch touches is `harness/CODEMAP.md`. Never hand-edit frontmatter — the tests drive `cli.py`.
- `.agents/skills/harness-execute/SKILL.md` and `.claude/skills/harness-execute/SKILL.md` are copies (likewise `harness-review`): edit both, `diff -q` must be silent after each.
- `stale-worktrees` never removes anything; it only prints. Keep `prune` in harness-orchestrate as the sole remover.
- No change to `.agents/toolchain.json` or any adapter; `cli.py doctor` must still pass.

## Review Focus
1. Nested worktree found: fixture porcelain with `/repo/.claude/worktrees/s/.worktrees/x` on a merged plan's branch → printed; same slug on a **non-merged** branch → not printed; entry whose basename matches a merged plan's `worktree:` but whose branch is a different, unmerged branch → not printed (rule 1 is branch-first).
2. Main worktree and the current checkout are never listed even when their branch is a merged plan's branch.
3. Fallback: with `_git_worktree_list()` returning `None`, `.worktrees/<name>` on disk with `name` = a merged plan's `worktree:` basename is printed, exactly as before.
4. `reviews_for`: a parent review with `covers: [amend]` makes `next --stage review` skip the amend and `STATE.md` drop `(unreviewed)` for it; a parent review **without** `covers` still queues the amend; `covers` on a review is validated as a list (`validate` exit 1 on a scalar).
5. Skill/role wording: one sentence each, no new step numbers; both copies identical.

## File structure

| Path | Change |
| --- | --- |
| `tools/harness/cli.py` | `_git_worktree_list()`, `parse_worktree_list(text)`, `stale_worktrees(plans, entries, cwd)`, `cmd_stale_worktrees` rewritten; `new-review --covers` |
| `tools/harness/scan.py` | `reviews_for` honours `covers` |
| `tools/harness/schema.py` | review `covers` must be a list when present |
| `tools/harness/tests/test_cli.py` | stale-worktree tests (3), `covers` tests (2) |
| `tools/harness/tests/test_scan_state.py` | `covers` clears `(unreviewed)` in STATE |
| `tools/harness/tests/test_schema.py` | `covers` type check |
| `.claude/skills/harness-execute/SKILL.md`, `.agents/skills/harness-execute/SKILL.md`, `.agents/roles/executor.md` | recording rule sentence |
| `.claude/skills/harness-review/SKILL.md`, `.agents/skills/harness-review/SKILL.md` | `--covers` sentence in step 6 |
| `.claude/skills/harness-orchestrate/SKILL.md`, `.agents/skills/harness-orchestrate/SKILL.md` | `prune`: paths are absolute, from `git worktree list` |
| `harness/CODEMAP.md` | Harness tooling paragraph: `stale-worktrees` source of truth, `covers` |

## Tasks

### Task 1: `stale-worktrees` discovers worktrees through `git worktree list --porcelain`

**Files:** `tools/harness/cli.py`, `tools/harness/tests/test_cli.py`.

- [ ] **Step 1 (tests first):** in `test_cli.py`, a helper `_merged_plan(self, title, branch, worktree)` that runs `new-run` → `new-idea` → `set status=selected priority=high` → `new-plan` → `set status=approved` → `executing` → `done` → `set branch=… worktree=… merged=true` and returns the plan path. Then, with `unittest.mock.patch.object(cli, "_git_worktree_list", return_value=FIXTURE)`:
  - `test_stale_worktrees_finds_merged_branches_anywhere_git_lists_them` — FIXTURE lists (in order) the main worktree `/repo` on `main`; `/repo/.claude/worktrees/s1/.worktrees/x` on `harness/2026-09-20-high-x` (merged plan, `worktree: .worktrees/x`); `/repo/.worktrees/y` on `harness/2026-09-21-high-y` (plan not merged); `/repo/.worktrees/x` on `harness/2026-09-26-high-x-again` (same basename as the merged plan's `worktree:`, unmerged branch); `/repo/.worktrees/z` **detached** (`detached` line, no `branch`). Assert output is exactly the nested `…/s1/.worktrees/x` line.
  - `test_stale_worktrees_skips_the_main_worktree_and_the_current_checkout` — FIXTURE's first entry and an entry equal to `pathlib.Path.cwd().resolve().as_posix()` both sit on a merged plan's branch → output empty.
  - `test_stale_worktrees_falls_back_to_the_worktrees_dir_without_git` — `_git_worktree_list` returns `None`; `pathlib.Path(".worktrees/x").mkdir(parents=True)`; a merged plan with `worktree: .worktrees/x` and **no** `branch:` → output `.worktrees/x`. Run `python3 -m unittest tools.harness.tests.test_cli -k stale -v`; all three fail on `AttributeError`/wrong output.
- [ ] **Step 2:** in `cli.py`, above `cmd_stale_worktrees`:
  - `_git_worktree_list()` → `subprocess.run(["git", "worktree", "list", "--porcelain"], capture_output=True, text=True)`; return `stdout` on returncode 0, else `None` (catch `OSError` too).
  - `parse_worktree_list(text)` → `[(path, branch_or_None), …]` in file order: a block starts at `worktree <path>`, `branch refs/heads/<b>` sets `<b>`, `detached` leaves `None`; blocks separated by blank lines.
  - `stale_worktrees(plans, entries, cwd)` → sorted list of paths per rule 1: `merged_branches = {p.fm["branch"] for merged plans with a branch}`; `legacy_names = {basename(p.fm["worktree"]) for merged plans with a worktree and no branch}`; skip `entries[0]`; skip an entry when `cwd == path or cwd.startswith(path + "/")`; include when `branch in merged_branches` or (`branch is None or branch not in any plan's branch`) **and** `basename(path) in legacy_names`.
  - `cmd_stale_worktrees`: `text = _git_worktree_list()`; if `text is None`, keep today's `.worktrees/` glob unchanged; else print each of `stale_worktrees(res.plans, parse_worktree_list(text), pathlib.Path.cwd().resolve().as_posix())`. Keep the docstring one line: "Worktrees whose branch belongs to a merged plan — git's list, not a directory scan".
- [ ] **Step 3:** `python3 -m unittest tools.harness.tests.test_cli -v` green; from this repo root `python3 tools/harness/cli.py stale-worktrees` prints the five paths listed in the idea's 2026-09-27 evaluation (paste into the execution summary; do **not** remove them — prune is the orchestrator's). Commit: `harness-cli: stale-worktrees reads git worktree list --porcelain`.

### Task 2: The executor records the worktree path git reports

**Files:** `.claude/skills/harness-execute/SKILL.md`, `.agents/skills/harness-execute/SKILL.md`, `.agents/roles/executor.md`, `.claude/skills/harness-orchestrate/SKILL.md`, `.agents/skills/harness-orchestrate/SKILL.md`.

- [ ] **Step 1:** harness-execute step 5, after `… worktree=$WT`, append one sentence: "If the sandbox refuses `.worktrees/<slug>`, create the worktree where you can and set `worktree=` to the path `git worktree list --porcelain` reports — frontmatter and the execution summary must agree with git, never with the plan's default." Apply to both copies.
- [ ] **Step 2:** `.agents/roles/executor.md` line 6 becomes: "Work only inside your plan's worktree (`.worktrees/<slug>` by default; whatever `git worktree list` reports if the sandbox put it elsewhere — record that path with `cli.py set <plan> worktree=<path>`) on the plan's branch. Never edit files in the main checkout."
- [ ] **Step 3:** harness-orchestrate `## prune`: "For each path from `cli.py stale-worktrees` (absolute, from `git worktree list`, wherever the worktree lives): `git worktree remove <path>`. Print what was removed." Both copies.
- [ ] **Step 4:** `diff -q .agents/skills/harness-execute/SKILL.md .claude/skills/harness-execute/SKILL.md && diff -q .agents/skills/harness-orchestrate/SKILL.md .claude/skills/harness-orchestrate/SKILL.md` silent; `python3 tools/harness/cli.py doctor` exit 0. Commit: `harness: executor records the worktree path git reports; prune takes absolute paths`.

### Task 3: A review's `covers:` list counts for the amend plans it re-verified (folded idea)

**Files:** `tools/harness/scan.py`, `tools/harness/schema.py`, `tools/harness/cli.py`, `tools/harness/tests/test_cli.py`, `test_scan_state.py`, `test_schema.py`, `.claude/skills/harness-review/SKILL.md`, `.agents/skills/harness-review/SKILL.md`, `harness/CODEMAP.md`.

- [ ] **Step 1 (tests first):**
  - `test_cli.py::test_next_review_honours_a_parent_review_that_covers_the_amend` — parent plan → `done`; amend plan (`new-plan` from a second idea, `set amends=<parent>`) → `done`; `next --stage review --all` lists both; `new-review --plan <parent> --verdict pass --covers <amend>` → review frontmatter `covers == [amend]`; `next --stage review --all` now empty.
  - `test_cli.py::test_next_review_still_queues_an_amend_the_parent_review_does_not_cover` — same setup, `new-review` on the parent **without** `--covers` → `next --stage review` returns the amend plan.
  - `test_scan_state.py::test_covered_amend_plan_is_not_flagged_unreviewed` — write a done amend plan and a parent review with `covers: [<amend>]` (fixture files, as the existing tests do); `render_state` output has no `(unreviewed)` on the amend's line.
  - `test_schema.py` — `validate("review", {..., "covers": "x"})` reports `covers must be a list`; a list passes.
  Run the four; they fail.
- [ ] **Step 2:** `schema.py` `validate`: under `kind == "review"`, `if "covers" in fm and not isinstance(fm["covers"], list): errs.append("covers must be a list")`. `scan.py` `reviews_for`: `[r for r in self.reviews if r.fm.get("plan") == plan_rel or plan_rel in (r.fm.get("covers") or [])]`. `cli.py`: `new-review` gains `p.add_argument("--covers", nargs="*")`; in `cmd_new_review` add `covers` to `fm` only when given (`if a.covers: fm["covers"] = a.covers`) so ordinary reviews are unchanged.
- [ ] **Step 3:** harness-review step 6 (both copies), after the `new-review` command: "When the review re-verifies amend plans that landed on this branch (`amends:` pointing at `$PLAN`), add `--covers <amend plan…>` so `next --stage review` and `STATE.md` count it for them." `harness/CODEMAP.md` Harness tooling paragraph: after "`cli.py` all mutations;" add "`stale-worktrees` lists worktrees from `git worktree list --porcelain` whose branch belongs to a merged plan (absolute paths; `.worktrees/` glob only without git); a review's `covers:` list makes `reviews_for` count it for the amend plans it re-verified;".
- [ ] **Step 4:** `python3 -m unittest discover -s tools/harness/tests -v` green; `python3 tools/harness/cli.py validate` exit 0 (existing reviews have no `covers`, so nothing changes); `diff -q` the two harness-review copies silent. Commit: `harness-cli: review covers: counts a parent re-review for its amend plans`.

## Verification
```
python3 -m unittest discover -s tools/harness/tests -v
python3 tools/harness/cli.py validate
python3 tools/harness/cli.py doctor
python3 tools/harness/cli.py stale-worktrees          # from the repo root: the five merged-branch paths from the idea's 2026-09-27 evaluation, absolute
python3 tools/harness/cli.py next --stage review --all # unchanged list (no review has covers: yet)
diff -q .agents/skills/harness-execute/SKILL.md .claude/skills/harness-execute/SKILL.md
diff -q .agents/skills/harness-review/SKILL.md .claude/skills/harness-review/SKILL.md
diff -q .agents/skills/harness-orchestrate/SKILL.md .claude/skills/harness-orchestrate/SKILL.md
grep -n 'git worktree list' tools/harness/cli.py .agents/roles/executor.md .claude/skills/harness-execute/SKILL.md harness/CODEMAP.md
grep -n 'covers' tools/harness/scan.py tools/harness/schema.py tools/harness/cli.py .claude/skills/harness-review/SKILL.md harness/CODEMAP.md
git diff --stat origin/main -- backend frontend deploy   # must be empty: tooling-only branch
git push -u origin harness/2026-09-27-low-executor-worktree-nested-under-claude-worktrees-while-plan-f
```

## Execution summary

Built all three tasks task-by-task, TDD (failing test committed conceptually verified red before implementation, then green):

- **Task 1** — `_git_worktree_list()`, `parse_worktree_list()`, `stale_worktrees()` added to `tools/harness/cli.py`; `cmd_stale_worktrees` now reads `git worktree list --porcelain` and falls back to the old `.worktrees/` glob only when git is unavailable. Reproduction: `test_stale_worktrees_finds_merged_branches_anywhere_git_lists_them`, `test_stale_worktrees_skips_the_main_worktree_and_the_current_checkout`, `test_stale_worktrees_falls_back_to_the_worktrees_dir_without_git` — all three failed with `AttributeError: ... does not have the attribute '_git_worktree_list'` before the change, all pass after. Ran the real command from the repo root against the actual worktree list; it printed the five nested/merged paths the idea described:
  ```
  /Users/hendrixnguyen/Workspaces/self/Learning-English-Project/.claude/worktrees/daily-task-evaluation-planning-971a45/.worktrees/every-google-403-becomes-409-reauth-required-so-a-quota-erro
  /Users/hendrixnguyen/Workspaces/self/Learning-English-Project/.claude/worktrees/daily-task-evaluation-planning-971a45/.worktrees/geminiprovider-drops-every-response-part-after-the-first-so-
  /Users/hendrixnguyen/Workspaces/self/Learning-English-Project/.claude/worktrees/harness-daily-execute-154026/.worktrees/nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect
  /Users/hendrixnguyen/Workspaces/self/Learning-English-Project/.claude/worktrees/harness-daily-execute-154026/.worktrees/providertimeout-of-30-s-makes-roadmap-generation-impossible-
  /Users/hendrixnguyen/Workspaces/self/Learning-English-Project/.worktrees/containerised-deploy-dockerfiles-production-compose-runbook-
  ```
  (printed, not removed — `prune` stays the orchestrator's job.)
- **Task 2** — executor role/skill wording updated to record the git-reported worktree path when the sandbox can't create `.worktrees/<slug>`; harness-orchestrate's `prune` section now says paths are absolute, from `git worktree list`. `.claude/skills` is a tracked symlink to `.agents/skills` on this checkout, so both copies are always byte-identical by construction; `diff -q` confirmed silent for all three skill pairs (harness-execute, harness-review, harness-orchestrate).
- **Task 3** — `schema.py` validates `covers` as a list when present; `scan.py` `reviews_for` now also matches `plan_rel in covers`; `cli.py new-review` gained `--covers`. Reproduction: `test_next_review_honours_a_parent_review_that_covers_the_amend`, `test_next_review_still_queues_an_amend_the_parent_review_does_not_cover`, `test_covered_amend_plan_is_not_flagged_unreviewed`, `test_covers_must_be_a_list` — all four failed before (missing `--covers` flag / `covers` not applied / assertion failed) and pass after.

**Deviation:** the `_merged_plan` test helper takes `branch=None` to omit the `branch:` field for the one fallback-path test that needs a legacy (no-branch) merged plan; the plan's helper signature didn't spell out that case but the fallback test (Task 1 Step 1, third bullet) requires it.

**Deviation (process, not code):** this plan's frontmatter/status bookkeeping could not be applied to the main checkout as harness-execute normally prescribes — the main checkout (`ROOT`) was on `harness/review-auto-merge` with unrelated uncommitted work and did not have this plan file at all (it only exists on `origin/main`), and the run's instructions explicitly forbade editing or staging anything there. This `## Execution summary` and the `status=done` below are therefore recorded only in this worktree's own (uncommitted, unpushed) copy of the plan file, per the branch's own rule that `harness/plans/*` is never committed on a plan branch. The orchestrator should apply `python3 tools/harness/cli.py set harness/plans/2026-09-27-executor-worktree-nested-under-claude-worktrees-while-plan-f.md status=done branch=harness/2026-09-27-low-executor-worktree-nested-under-claude-worktrees-while-plan-f worktree=.worktrees/executor-worktree-nested-under-claude-worktrees-while-plan-f` on whatever checkout is canonical.

### Runtime proof
- Build: N/A (pure Python stdlib tooling, no compiled artifact).
- Full suite from a clean shell: `python3 -m unittest discover -s tools/harness/tests -v` → 50 tests, OK.
- Boots and answers: `python3 tools/harness/cli.py context` and `python3 tools/harness/cli.py state` run clean; `stale-worktrees` and `next --stage review --all` exercised against the real repo state (see above).
- Every documented command: `validate` (exit 0), `doctor` (0 problems), `stale-worktrees`, `next --stage review --all` (unchanged — no existing review carries `covers:` yet).
- No process/container was started by this plan; nothing to clean up.
- CI on `harness/2026-09-27-low-executor-worktree-nested-under-claude-worktrees-while-plan-f`: https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36291400442 — conclusion `success` (frontend, backend-unit, harness-tooling, backend-integration, docker-images all green).
