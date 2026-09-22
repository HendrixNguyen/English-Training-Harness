---
idea: harness/ideas/_inbox/cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md
status: done
priority: high
merged: true
amends: harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md
branch: harness/2026-09-22-high-ci-on-github-actions-for-backend-and-harness-tooling
worktree: .worktrees/ci-on-github-actions-for-backend-and-harness-tooling
---
# cancel-in-progress cancels CI on main, the only ref CI actually runs on — Plan

**Idea:** `harness/ideas/_inbox/cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md`
**Also resolves:** `harness/ideas/_inbox/ci-never-runs-on-harness-branches-so-it-gates-nothing-before.md` (second blocker on the same branch; its `plan:` points here) and the doc findings in `harness/ideas/_inbox/codemap-ci-section-overstates-the-unit-job-and-prescribes-a-.md` (rejected in favour of this plan).
**Amends:** `harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md`
**Goal:** Make CI run on pushed `harness/**` branches so a branch fails visibly *before* `/harness merge`, stop cancelling runs on `main` so every merge commit gets its own verdict, cap each job at 10 minutes, and make `AGENTS.md` / `harness/CODEMAP.md` describe the workflow as it will now actually behave.

**Why this is a blocker (do not re-litigate):** the CI idea's *Expected output* requires a branch to fail visibly "before a human is asked to merge it". With `on.push.branches: [main]`, a `pull_request` trigger that cannot fire (this account cannot open PRs — the CI plan's execution summary records `must be a collaborator`), and `/harness merge` pushing `main` directly, CI first runs *after* the merge has landed. The idea is not delivered until the trigger changes.

**Architecture:** Five changed lines in `.github/workflows/ci.yml` — the push trigger gains `'harness/**'`; the concurrency group keys on the commit SHA when the ref is `main` and on the ref otherwise; `cancel-in-progress` becomes the expression `github.ref != 'refs/heads/main'`; each of the three jobs gains `timeout-minutes: 10`. No `run:` block, service, or `env` changes, so the job-level evidence in the review (`harness/reviews/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md`) still stands and is proved to stand by a structural diff against the reviewed commit rather than by re-running Docker. The docs are rewritten to match. One small harness-tooling change makes `cli.py blockers` recognise that the second blocker is fixed by this plan (see Task 3 — without it the blocker would never clear and `merged=true` on the CI plan would be refused forever).

**Concurrency decision.** The idea proposed keeping one group per ref and making only `cancel-in-progress` conditional. That is not enough on `main`: a concurrency group holds at most one running and one *pending* run, and a third push cancels the pending one, so three back-to-back `/harness merge` pushes in one unattended run would still leave the middle commit with no verdict. Keying the group on `github.sha` when the ref is `main` gives every merge commit its own singleton group, so runs on `main` are never cancelled and never queue behind each other. Branches keep one group per ref with `cancel-in-progress: true` (only the tip matters). `cancel-in-progress` is also written as the conditional expression, which is redundant with a singleton group but states the intent in the file and survives someone later collapsing the group back to `github.ref`. Both `group` and `cancel-in-progress` accept expressions; `actionlint` 1.7.12 accepts this exact text (checked on a copy on 2026-09-22).

**Timeouts decision.** `timeout-minutes: 10` on each job is three one-line insertions in the same file, verified by the same YAML parse, so it is folded in (the suite finishes in seconds locally; GitHub's default is 360 minutes). The other half of `harness/ideas/_inbox/ci-jobs-have-no-timeout-minutes-and-the-harness-job-floats-p.md` — pinning `python-version` or running a matrix — is a separate judgement about which interpreters the tooling supports; it stays in the inbox and is **not** part of this plan. Do not touch `python-version`.

**Branch:** work on the existing `harness/2026-09-22-high-ci-on-github-actions-for-backend-and-harness-tooling` in `.worktrees/ci-on-github-actions-for-backend-and-harness-tooling`. Do **not** create a new branch or worktree — this plan amends the branch under review. Every file below already exists on it. Its head at planning time is `8f5dbf9` (the reviewed commit); Task 1 Step 4 diffs against that hash.

**Run every command from the worktree root** unless a step says otherwise. `rg` is not installed — use `grep -n`. Never capture `$(ls …)`. **Do not push** to see a run: the harness cannot rely on reading Actions results, and nothing below needs Docker or the network — host port 6379 belongs to the owner's unrelated `scio3-redis-1` container and must not be disturbed. `actionlint` validates workflow syntax and expressions but **does not check action tags or repo names** (the review proved this with `actions/chekout@v7`); this plan changes no `uses:` lines, and Task 1 Step 3 asserts that.

## File structure

| Path | Change |
| --- | --- |
| `.github/workflows/ci.yml:3-13` | push trigger adds `'harness/**'`; concurrency group per-SHA on `main`, `cancel-in-progress` conditional |
| `.github/workflows/ci.yml` jobs at lines 16, 42, 95 | `timeout-minutes: 10` under each `runs-on:` |
| `harness/CODEMAP.md` `## CI` section (line 24 to end) | rewritten: short lead-in + one bullet per job, accurate guard description, `--wait` repro |
| `AGENTS.md:23` | the CI bullet rewritten for `harness/**` push gating and no-cancel on `main` |
| `tools/harness/scan.py:30-42` | `blockers_for` also resolves the fix plan through the idea's own `plan:` back-link |
| `tools/harness/tests/test_cli.py` | regression test: two blockers, one shared amending plan |

Nothing under `backend/` changes.

---

## Tasks

### Task 1: Trigger, concurrency and timeouts in the workflow

**Files:**
- Modify: `.github/workflows/ci.yml:3-13` and the three `runs-on:` lines

- [ ] **Step 1: Confirm the local toolchain**

```sh
actionlint -version | head -1
python3 -c 'import yaml; print("pyyaml", yaml.__version__)'
git log --oneline -1
```
Expected: `1.7.12` (or newer), `pyyaml 6.0.3`, and `8f5dbf9 docs: record CI as the harness's outer verification loop`. If the head is not `8f5dbf9`, stop and check nothing else has landed on the branch before continuing.

- [ ] **Step 2: Edit the workflow header**

Replace lines 3-13 of `.github/workflows/ci.yml` (from `on:` through `cancel-in-progress: true`) with exactly:

```yaml
on:
  push:
    branches: [main, 'harness/**']
  pull_request:

permissions:
  contents: read

concurrency:
  # Branches: one group per ref, superseded runs cancelled. main: one group per
  # commit, so back-to-back merges each finish their own run (a shared group
  # would cancel the *pending* run when a third push arrives).
  group: ci-${{ github.workflow }}-${{ github.ref == 'refs/heads/main' && github.sha || github.ref }}
  cancel-in-progress: ${{ github.ref != 'refs/heads/main' }}
```

`'harness/**'` is quoted because `*` would otherwise start a YAML alias. `pull_request` stays: it costs nothing and gates automatically the day this account can open PRs; the PR merge ref is `refs/pull/N/merge`, a different group from the branch, so a branch that later gets a PR runs once per ref, not twice for the same one.

- [ ] **Step 3: Add the job timeouts**

Under each of the three `runs-on: ubuntu-latest` lines (jobs `backend-unit`, `backend-integration`, `harness-tooling`) insert one line at the same indentation as `runs-on`:

```yaml
    timeout-minutes: 10
```

The result is `runs-on:` immediately followed by `timeout-minutes:` in every job. Touch nothing else in the file — no `uses:`, `run:`, `services:`, `env:` or `with:` line changes.

- [ ] **Step 4: Static checks — the evidence the review used, plus the new structure**

```sh
actionlint .github/workflows/ci.yml; echo "actionlint exit=$?"
python3 - <<'EOF'
import yaml
d = yaml.safe_load(open(".github/workflows/ci.yml"))
on = d.get("on", d.get(True))          # PyYAML reads the bare key `on` as boolean True
assert on["push"]["branches"] == ["main", "harness/**"], on
assert "pull_request" in on, on
c = d["concurrency"]
assert "github.ref == 'refs/heads/main' && github.sha || github.ref" in c["group"], c
assert c["cancel-in-progress"] == "${{ github.ref != 'refs/heads/main' }}", c
j = d["jobs"]
assert sorted(j) == ["backend-integration", "backend-unit", "harness-tooling"], sorted(j)
assert all(job.get("timeout-minutes") == 10 for job in j.values()), {k: v.get("timeout-minutes") for k, v in j.items()}
assert d["permissions"] == {"contents": "read"}
print("on:", on); print("concurrency:", c); print("yaml checks ok")
EOF
# Job bodies are byte-for-byte what the reviewer verified: only the header and three timeout lines moved.
python3 - <<'EOF'
import subprocess, yaml
old = yaml.safe_load(subprocess.check_output(["git", "show", "8f5dbf9:.github/workflows/ci.yml"]))
new = yaml.safe_load(open(".github/workflows/ci.yml"))
for k in old["jobs"]:
    assert old["jobs"][k]["steps"] == new["jobs"][k]["steps"], k
    assert old["jobs"][k].get("services") == new["jobs"][k].get("services"), k
    assert old["jobs"][k].get("env") == new["jobs"][k].get("env"), k
print("job bodies unchanged since 8f5dbf9")
EOF
grep -n "^    branches:\|^  group:\|^  cancel-in-progress:\|^    timeout-minutes:" .github/workflows/ci.yml
grep -c 'uses: actions/checkout@v7' .github/workflows/ci.yml; grep -c 'uses: actions/setup-go@v7' .github/workflows/ci.yml; grep -c 'uses: actions/setup-python@v7' .github/workflows/ci.yml
git diff --stat 8f5dbf9 -- .github/workflows/ci.yml
```
Expected: `actionlint exit=0` with nothing printed above it; `on: {'push': {'branches': ['main', 'harness/**']}, 'pull_request': None}`; the concurrency dict with the two expressions; `yaml checks ok`; `job bodies unchanged since 8f5dbf9`; six grep lines — `branches:` at line 5, `group:` and `cancel-in-progress:` around lines 15-16, three `timeout-minutes: 10` lines; `3`, `2`, `1` (the `uses:` lines are untouched, which matters because `actionlint` cannot validate them); a diff stat of one file, roughly `+9 -3`.

- [ ] **Step 5: Commit**

```sh
git add .github/workflows/ci.yml
git diff --cached --stat
git commit -m "ci: run on harness/** pushes, never cancel runs on main, cap jobs at 10 min"
```
Expected: one file in the staged stat.

### Task 2: Make the docs describe how CI now runs

**Files:**
- Modify: `harness/CODEMAP.md` — the `## CI (.github/workflows/ci.yml)` section (line 24 to end of file)
- Modify: `AGENTS.md:23` — the CI bullet under *Rules every role follows*

- [ ] **Step 1: Rewrite the CODEMAP CI section**

Replace everything from the line `## CI (\`.github/workflows/ci.yml\`)` to the end of `harness/CODEMAP.md` with exactly:

```markdown
## CI (`.github/workflows/ci.yml`)

Three parallel GitHub Actions jobs, each capped at `timeout-minutes: 10`, on every push to `main` or a `harness/**` branch and on every pull request. A pushed `harness/*` branch is therefore checked before `/harness merge`; superseded runs are cancelled per branch but never on `main`, where each commit gets its own concurrency group. `actionlint` validates the file but not action tags or repo names — check those by hand.

- **`backend-unit`** — from `backend/`: `go build ./...`, `go vet ./...`, `go test ./... -count=1`, after a guard that fails if `DATABASE_URL`, `REDIS_URL`, `TEST_DATABASE_URL` or `TEST_REDIS_URL` is exported, so the default suite is proved to need no services. (Whether the destructive suite stays gated is `internal/store/integration_gate_test.go`'s job, not this one's.)
- **`backend-integration`** — `postgres:16-alpine` + `redis:7-alpine` service containers, `TEST_DATABASE_URL` / `TEST_REDIS_URL` exported, `go test ./... -count=1 -v -run Integration`; counts `func TestIntegration*` in `*_test.go` and fails unless that many `--- PASS` lines appear and no `--- SKIP` does, so new integration tests are picked up without editing the workflow and a silent skip is a failure. Reproduce from `backend/` with `POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose up -d --wait` (`make up` has no `--wait`), then the `TEST_*` URLs on those ports.
- **`harness-tooling`** — `python3 -m unittest discover -s tools/harness/tests` and `python3 tools/harness/cli.py validate`, so a malformed frontmatter commit fails CI.

No linter yet: `golangci-lint` defaults flag one `errcheck` in `internal/store/redis_test.go`; add it only with that fixed and a committed `.golangci.yml`.
```

This drops the "never drops tables" claim (the guard proves only that no service variable is exported), gives the reproduction command that was actually run, and is one bullet per job as `CODEMAP.md`'s own contract asks.

- [ ] **Step 2: Rewrite the AGENTS.md CI rule**

Replace line 23 of `AGENTS.md` (the bullet beginning `- CI (\`.github/workflows/ci.yml\`) is the outer verification loop`) with exactly:

```markdown
- CI (`.github/workflows/ci.yml`) is the outer verification loop and must stay green: `backend-unit`, `backend-integration` and `harness-tooling` run on every push to `main` or a `harness/**` branch, and on pull requests when one exists. The push of a `harness/*` branch is what gates it: the reviewer reads that branch's run (`gh run list --branch <branch>`), a red or missing check on it is a review blocker, and `/harness merge` must not push `main` while it is red. Superseded runs are cancelled on branches, never on `main`. Never make a job pass by skipping, loosening or deleting a check — fix the code or the artifact it flagged.
```

- [ ] **Step 3: Check the edits landed where intended**

```sh
grep -n '^## CI' harness/CODEMAP.md
grep -c '^- \*\*`backend-' harness/CODEMAP.md
grep -n 'never drops tables\|make up` with' harness/CODEMAP.md; echo "stale phrases: $?"
grep -n 'docker compose up -d --wait' harness/CODEMAP.md
grep -n 'outer verification loop' AGENTS.md
grep -n "harness/\*\*" AGENTS.md harness/CODEMAP.md
grep -c 'run on every PR and push to `main`' AGENTS.md; echo "old wording gone: $?"
python3 tools/harness/cli.py validate; echo "validate exit=$?"
```
Expected: one `## CI` heading at line 24; `3` job bullets; no matches for the stale phrases (`stale phrases: 1`); one `--wait` hit in CODEMAP; one `outer verification loop` hit at `AGENTS.md:23`; `harness/**` in both docs; `0` then `old wording gone: 1`; `validate exit=0`.

- [ ] **Step 4: Commit**

```sh
git add harness/CODEMAP.md AGENTS.md
git diff --cached --stat
git commit -m "docs: describe CI as gating harness/** pushes, never cancelling on main"
```
Expected: exactly two files in the staged stat.

### Task 3: Let `cli.py blockers` see that one plan fixes two blockers

**Files:**
- Modify: `tools/harness/scan.py:30-42`
- Test: `tools/harness/tests/test_cli.py` (add one method after `test_blocker_refuses_merge_until_fixed`)

Why: `ScanResult.blockers_for` finds a blocker's fix plan with `plan_for_idea`, which matches only a plan whose `idea:` is that blocker. This plan's `idea:` is the concurrency blocker; the trigger blocker was pointed here with `status: planned` / `plan: <this plan>`, and today that back-link is ignored, so `cli.py blockers --plan <CI plan>` keeps listing it after this plan is `done` and `cli.py set <CI plan> merged=true` is refused. (Reproduced 2026-09-22 on a scratch copy: exit 1 before the change below, exit 0 after, 30/30 tests still passing.) The fix is three lines and its own regression test.

- [ ] **Step 1: Write the failing test**

Add to `tools/harness/tests/test_cli.py`, directly after `test_blocker_refuses_merge_until_fixed`:

```python
    def test_two_blockers_can_share_one_fix_plan_via_the_plan_backlink(self):
        _, run = self.run_cli("new-run")
        _, idea = self.run_cli("new-idea", "--run", run, "--title", "Slice", "--type", "mvp-slice", "--source", "ideator", "--order", "1")
        self.run_cli("set", idea, "status=selected", "priority=high")
        _, plan = self.run_cli("new-plan", "--idea", idea)
        for st in ["approved", "executing", "done"]:
            self.run_cli("set", plan, f"status={st}")
        # the reviewer files two blockers against the same branch
        _, bug1 = self.run_cli("new-idea", "--run", run, "--title", "Trigger", "--type", "bug", "--source", "reviewer", "--priority", "high")
        _, bug2 = self.run_cli("new-idea", "--run", run, "--title", "Concurrency", "--type", "bug", "--source", "reviewer", "--priority", "high")
        for b in (bug1, bug2):
            self.assertEqual(self.run_cli("set", b, f"blocks={plan}")[0], 0)
        # one amending plan hangs off bug1; bug2 is pointed at it through its own plan: field
        self.run_cli("set", bug1, "status=selected")
        _, fix = self.run_cli("new-plan", "--idea", bug1)
        self.assertEqual(self.run_cli("set", fix, f"amends={plan}")[0], 0)
        self.run_cli("set", bug2, "status=selected")
        self.assertEqual(self.run_cli("set", bug2, "status=planned", f"plan={fix}")[0], 0)
        # both still block while the shared fix is in flight
        code, out = self.run_cli("blockers", "--plan", plan)
        self.assertEqual(code, 1)
        self.assertIn(bug1, out); self.assertIn(bug2, out)
        for st in ["approved", "executing", "done"]:
            self.run_cli("set", fix, f"status={st}")
        # and both clear once it is done
        self.assertEqual(self.run_cli("blockers", "--plan", plan), (0, ""))
        self.assertEqual(self.run_cli("set", plan, "merged=true")[0], 0)
```

- [ ] **Step 2: Run it and watch it fail**

```sh
python3 -m unittest tools.harness.tests.test_cli.CliTests.test_two_blockers_can_share_one_fix_plan_via_the_plan_backlink 2>&1 | tail -4
```
Expected: `FAIL` — the `assertEqual(self.run_cli("blockers", ...), (0, ""))` line, because the output still names `bug2`.

- [ ] **Step 3: Resolve the fix plan through the idea's `plan:` as well**

In `tools/harness/scan.py`, replace lines 30-42 (`plan_for_idea` and `blockers_for`) with:

```python
    def plan_for_idea(self, idea_rel):
        return next((p for p in self.plans if p.fm.get("idea") == idea_rel), None)

    def plan_by_rel(self, plan_rel):
        return next((p for p in self.plans if p.rel == plan_rel), None)

    def blockers_for(self, plan_rel):
        """Blocker ideas for plan_rel whose own fix plan is not yet done.

        The fix plan is the one whose `idea:` is the blocker, or — when several
        blockers share one amending plan — the plan the blocker's own `plan:`
        points at.
        """
        out = []
        for i in self.ideas:
            if i.fm.get("blocks") != plan_rel:
                continue
            fix = self.plan_for_idea(i.rel) or self.plan_by_rel(i.fm.get("plan"))
            if fix is None or fix.fm.get("status") != "done":
                out.append(i)
        return out
```

- [ ] **Step 4: Run the whole harness suite**

```sh
python3 -m unittest discover -s tools/harness/tests 2>&1 | tail -3
python3 tools/harness/cli.py validate; echo "validate exit=$?"
```
Expected: `Ran 31 tests`, `OK`; `validate exit=0`.

- [ ] **Step 5: Commit**

```sh
git add tools/harness/scan.py tools/harness/tests/test_cli.py
git diff --cached --stat
git commit -m "harness: let blockers resolve a shared fix plan through the idea's plan back-link"
```
Expected: exactly two files in the staged stat.

## Verification

Run from the worktree root. Nothing here pushes, needs GitHub access, or starts a container.

```sh
# 1. The workflow: the review's static evidence, re-run, plus the new shape.
actionlint .github/workflows/ci.yml; echo "actionlint exit=$?"
python3 - <<'EOF'
import yaml, subprocess
d = yaml.safe_load(open(".github/workflows/ci.yml")); on = d.get("on", d.get(True))
assert on["push"]["branches"] == ["main", "harness/**"] and "pull_request" in on, on
c = d["concurrency"]
assert "github.ref == 'refs/heads/main' && github.sha || github.ref" in c["group"], c
assert c["cancel-in-progress"] == "${{ github.ref != 'refs/heads/main' }}", c
j = d["jobs"]
assert sorted(j) == ["backend-integration", "backend-unit", "harness-tooling"]
assert all(job.get("timeout-minutes") == 10 for job in j.values())
assert d["permissions"] == {"contents": "read"}
old = yaml.safe_load(subprocess.check_output(["git", "show", "8f5dbf9:.github/workflows/ci.yml"]))
assert all(old["jobs"][k]["steps"] == j[k]["steps"] and old["jobs"][k].get("services") == j[k].get("services") and old["jobs"][k].get("env") == j[k].get("env") for k in old["jobs"])
print("on:", on); print("yaml checks ok; job bodies unchanged since 8f5dbf9")
EOF
grep -n "^    branches:\|^  group:\|^  cancel-in-progress:\|^    timeout-minutes:" .github/workflows/ci.yml
grep -n 'uses: actions/' .github/workflows/ci.yml        # still 6 lines, all @v7 — actionlint cannot check these
```
Expected: `actionlint exit=0`; `on: {'push': {'branches': ['main', 'harness/**']}, 'pull_request': None}`; `yaml checks ok; job bodies unchanged since 8f5dbf9`; six structural grep lines (one `branches:`, one `group:`, one `cancel-in-progress:`, three `timeout-minutes: 10`); six `uses:` lines each ending `@v7`.

```sh
# 2. Docs say what the file does.
grep -n 'harness/\*\*' AGENTS.md harness/CODEMAP.md
grep -c '^- \*\*`backend-\|^- \*\*`harness-' harness/CODEMAP.md
grep -n 'never drops tables\|make up` with\|run on every PR and push to `main`' AGENTS.md harness/CODEMAP.md; echo "stale wording: $?"
grep -n 'outer verification loop' AGENTS.md
```
Expected: `harness/**` in both files; `3`; `stale wording: 1` (no matches); one hit at `AGENTS.md:23`.

```sh
# 3. Harness job, as CI runs it, including the new regression test.
python3 -m unittest discover -s tools/harness/tests 2>&1 | tail -3
python3 tools/harness/cli.py validate; echo "validate exit=$?"
```
Expected: `Ran 31 tests`, `OK`; `validate exit=0`.

```sh
# 4. The blocker wiring: on a scratch copy of the main checkout's harness state, marking THIS plan
#    done clears both blockers on the CI plan (uses the scan.py from this worktree).
ROOT=$(git worktree list --porcelain | awk '/^worktree /{print $2; exit}')
TMP=$(mktemp -d); cp -R "$ROOT/harness" "$ROOT/.agents" "$TMP"/; cp -R tools "$TMP"/
CI=harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md
THIS=harness/plans/2026-09-22-cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md
( cd "$TMP" && for st in approved executing done; do python3 tools/harness/cli.py set $THIS status=$st >/dev/null || true; done
  python3 tools/harness/cli.py blockers --plan $CI; echo "blockers exit=$?" )
rm -rf "$TMP"
```
Expected: `blockers exit=0` with no blocker lines above it. (If this plan is already `executing` in the main checkout, the `approved` transition prints an error that `|| true` swallows; only the final `done` matters.) Before Task 3, the same commands print the `ci-never-runs-on-harness-branches…` line and `blockers exit=1`.

```sh
# 5. Tree and history.
git status --short; git log --oneline -4
```
Expected: clean tree; three new commits (Tasks 1-3) on top of `8f5dbf9`.

## Notes

- **In scope from the inbox:** the two blockers; the three doc findings from `codemap-ci-section-overstates-the-unit-job-and-prescribes-a-.md` (rejected with `rejected_reason` pointing here); the `timeout-minutes` half of `ci-jobs-have-no-timeout-minutes-and-the-harness-job-floats-p.md`. **Not in scope:** that idea's `python-version` pin/matrix (left open in the inbox), `golangci-lint`, any change to `uses:` tags, any change under `backend/`.
- The trigger blocker's *Expected output* also asked that "the review and merge steps read" the branch check. `AGENTS.md:23` now names `gh run list --branch <branch>`; the skill files under `.agents/skills/` are not edited by this plan — if the orchestrator wants a hard precondition on a green branch run, that is a follow-up idea, because reading Actions results has not been part of any role's verified evidence so far.
- The reviewer of this plan should re-run section 1 and 4 above; sections 2-3 are cheap enough to always run. Re-running the Docker-backed integration job is unnecessary because section 1 proves the job bodies are unchanged from the reviewed commit.

## Execution summary

Executed in the existing worktree `.worktrees/ci-on-github-actions-for-backend-and-harness-tooling` on branch `harness/2026-09-22-high-ci-on-github-actions-for-backend-and-harness-tooling` (head at start `8f5dbf9`), per the `amends:` instruction — no new branch or worktree was created. No deviations from the plan; all three tasks landed exactly as specified.

**Commits (on top of `8f5dbf9`):**
- `61cce5c` ci: run on harness/** pushes, never cancel runs on main, cap jobs at 10 min
- `87dcfb6` docs: describe CI as gating harness/** pushes, never cancelling on main
- `9376d2b` harness: let blockers resolve a shared fix plan through the idea's plan back-link

**Verification (run from the worktree root):**

Section 1 (workflow): `actionlint exit=0`; the YAML structural asserts all passed (`on: {'push': {'branches': ['main', 'harness/**']}, 'pull_request': None}`); the diff-vs-`8f5dbf9` job-body check passed (`job bodies unchanged since 8f5dbf9` — steps, services, env identical for all three jobs); the six structural greps matched exactly one `branches:`, one `group:`, one `cancel-in-progress:`, three `timeout-minutes: 10`; all six `uses:` lines are still `@v7`, untouched.

Section 2 (docs): `harness/**` present in both `AGENTS.md` and `harness/CODEMAP.md`; `outer verification loop` still at `AGENTS.md:23`; stale wording (`never drops tables`, `` make up` with ``, `run on every PR and push to `main` ``) absent from both files. Note: the plan's own Task 2 Step 3 mid-task grep (`^- \*\*`backend-`) counts only 2 lines by construction (it excludes the `harness-tooling` bullet) even though its prose says "3 job bullets" — that is a mismatch in the plan's own mid-task check text, not a defect in the docs; the final Verification section's grep (`backend-\|harness-`) correctly counts all 3 and returned `3`.

Section 3 (harness suite): `python3 -m unittest discover -s tools/harness/tests` → `Ran 31 tests ... OK` (30 before this plan, +1 regression test as specified); `python3 tools/harness/cli.py validate` → `validate exit=0`.

Section 4 (blocker wiring, scratch copy): copied `harness/`, `.agents/`, and this worktree's `tools/` (i.e. with the Task 3 fix) into a scratch dir, drove this plan through `approved`→`executing`→`done` there, then `python3 tools/harness/cli.py blockers --plan harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md` → `blockers exit=0` with no blocker lines printed — confirms `scan.py`'s fix correctly resolves both blockers (including the trigger blocker reached only via its `plan:` back-link) once this plan is done.

Section 5 (tree/history): `git status --short` clean; `git log --oneline -4` shows the three new commits above `8f5dbf9`.

**Runtime proof / scope note:** this plan touches only `.github/workflows/ci.yml`, two docs, and `tools/harness/scan.py` + its test — "Nothing under `backend/` changes" per the plan's own File structure table, and the plan explicitly states nothing in it needs Docker, the network, or a push to verify. No backend build/boot/integration run was performed, consistent with the plan's stated scope and verification (which itself substitutes the diff-vs-`8f5dbf9` job-body check for re-running the Docker-backed integration job). The harness tooling itself was proven to run via section 3 above (its own test suite, exercising the real `cli.py`/`scan.py` code paths end to end, is the "boot and answer" proof for this plan's actual deliverable).

**Post-hoc check on the real ROOT state (not a prediction):** after this plan's own status was set to `done` in the main checkout via `cli.py`, `python3 tools/harness/cli.py blockers --plan harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md` run from `ROOT` still reports the second blocker unresolved (exit 1), because the `scan.py` fix (Task 3) lives only on this plan's branch/worktree and has not been merged to `main` — `ROOT`'s `tools/harness/scan.py` is still the pre-fix version. This is expected: the scratch-copy test in Section 4 proves the *logic* is correct once the branch's `scan.py` is in effect; the *actual* CI-plan blocker list on `main` only clears after a human merges this branch (`/harness merge`), which is out of scope for an executor. See the literal command output quoted in this session's final report.
