---
idea: harness/ideas/_inbox/parseroadmap-day-sum-rejection-has-no-test-the-90-and-9-minu.md
status: approved
priority: high
merged: false
amends: harness/plans/2026-09-24-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md
---
# airouter amend: `TestParseRoadmapRejects` proves the day-sum rule — rows only the day budget can reject, keyed by reason — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Bug team — blocker of 2026-09-26. **Estimate:** 1 h. **Amends:** `harness/plans/2026-09-24-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md` — work in **its** worktree on **its** branch `harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut` (harness-execute skill, "Amending plan"); no new branch, no new worktree.

**Idea:** `harness/ideas/_inbox/parseroadmap-day-sum-rejection-has-no-test-the-90-and-9-minu.md` (blocker, from review `harness/reviews/2026-09-25-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md`). Also closed here: `harness/ideas/_inbox/testparseroadmaprejects-checks-only-that-an-error-occurred-s.md` (the reason column, Task 1) and `harness/ideas/_inbox/week-2-provenance-assertion-in-testexercisesflattens-repeats.md` (Task 1b) — both reviewer findings on this same branch.

**Goal:** Deleting the day-sum `if` in `ParseRoadmap` makes `go test ./internal/airouter` fail, and every row of `TestParseRoadmapRejects` proves the rule it is named after.

**Not on `main`:** `origin/main` still has the pre-plan parser (`maxTaskMinutes = 30`, no day sum); nothing here applies to `main` directly — the daily merge carries branch + amend together.

**Test-only change.** `backend/internal/airouter/roadmap.go` is not edited. If the executor finds the day-sum message differs from `adds up to %d minutes, want %d..%d` on the branch, use the branch's actual text — never change production text to fit the test.

## Global Constraints
- Amending plan: `git worktree list` shows the branch's worktree (frontmatter `worktree:` of the amended plan; if that path is missing, `git worktree add .worktrees/parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut` — the branch, never a new one). Do **not** merge `origin/main` into it unless it already contains the branch's own commits; the reviewer compares against `42f8fd6`.
- `rg`/`timeout` not installed: `grep -n`, `go test -timeout 60s`. Run from `backend/`.
- `gofmt -l internal/airouter` prints nothing (CI gate).

## Tasks

### Task 1: Key the rejection table by expected reason

**Files:** Modify `backend/internal/airouter/roadmap_test.go` — `TestParseRoadmapRejects` (map literal at ~line 93, loop at ~line 128).

- [ ] **Step 1:** Change the table type from `map[string]string` to `map[string]struct{ raw, want string }` where `want` is a substring of the `invalid(...)` message. Fill `want` for every existing row from the branch's `roadmap.go` messages, e.g. `"three modules"` → `"want 4 modules"` (use the real text: `grep -n 'invalid("' internal/airouter/roadmap.go`), `"duplicate task type"` → `"repeats task type"`, `"empty task title"` → `"has no title"`, `"absurd duration"` → `"outside 5..15"`, `"weeks reversed"` / `"duplicate week"` / `"week counts from 0"` → `"declares week"` (or the branch's wording, which must contain `want 1` for the counts-from-0 row), `"empty roadmap title"` / `"blank roadmap title"` / `"empty module title"` / `"empty day title"` → `"has no title"`, `"bad cefr"` → its message, and `"preamble"` / `"trailing garbage"` / `"empty"` / `"not an object"` → the decode-error wording the branch uses (read it; if those four wrap a `json` error with no stable text, `want` may be `""`, meaning "any reason", and only for those four — say so in a comment).
- [ ] **Step 2:** Rename the two misnamed rows to what they test: `"three thirty-minute tasks (90-minute day)"` → `"thirty-minute task (task band)"` with `want: "outside 5..15"`; `"three three-minute tasks (9-minute day)"` → `"three-minute task (task band)"` with `want: "outside 5..15"`. Keep their mutations.
- [ ] **Step 3:** Append the two rows only the day sum can reject, each task inside `5..15`:

```go
		"three five-minute tasks (15-minute day)": {validRoadmapJSON(t, func(r *Roadmap) {
			for i := range r.Modules[0].Days[0].Tasks {
				r.Modules[0].Days[0].Tasks[i].DurationMinutes = 5
			}
		}), "adds up to 15 minutes"},
		"three fifteen-minute tasks (45-minute day)": {validRoadmapJSON(t, func(r *Roadmap) {
			for i := range r.Modules[1].Days[2].Tasks {
				r.Modules[1].Days[2].Tasks[i].DurationMinutes = 15
			}
		}), "adds up to 45 minutes"},
```

- [ ] **Step 4:** In the loop, after the `errors.Is` check add: `if tc.want != "" && !strings.Contains(err.Error(), tc.want) { t.Errorf("err = %q, want it to mention %q", err, tc.want) }`. (`strings` is already imported on the branch.)
- [ ] **Step 5:** `go test -timeout 60s ./internal/airouter -run TestParseRoadmapRejects -v` → every row PASS. `gofmt -l internal/airouter` → empty.
- [ ] **Step 6: Mutation check (the review's own evidence).** Temporarily edit `roadmap.go`: `if false && (dayMinutes < minDayMinutes || dayMinutes > maxDayMinutes) {`. Run the test: the two new rows **must FAIL** with "ParseRoadmap accepted it". Revert the edit (`git checkout -- internal/airouter/roadmap.go`; `git diff --stat` shows only `roadmap_test.go`). Paste the failing output into this plan's Execution summary.
- [ ] **Step 7:** Commit: `airouter: TestParseRoadmapRejects keys every row by its rejection reason; day-sum rows at 15 and 45 minutes`.

### Task 1b: The week-2 provenance assertion proves provenance (folded: `week-2-provenance-assertion-in-testexercisesflattens-repeats.md`)

**Files:** `backend/internal/airouter/roadmap_test.go` — `validRoadmapJSON` fixture (~line 20) and `TestExercisesFlattensTo84RowsCarryingTitleAndDuration` (~lines 150–157).

- [ ] **Step 1:** In `validRoadmapJSON`, make every task title unique per module/day/type: `Title: fmt.Sprintf("w%d-d%d-%s", m, d, tt)` (the `"<type> task"` string is asserted nowhere else — check with `grep -n '" task"' internal/airouter/*_test.go internal/onboarding/*_test.go`; if a test does depend on it, update that assertion in the same commit).
- [ ] **Step 2:** Replace the branch's week-2 assertion (`r.Modules[1].Week != 2 || ex[21].DayNumber != …`) with: `if want := r.Modules[1].Days[0].Tasks[0].Title; ex[21].Title != want || ex[21].DayNumber != 8 { t.Errorf(...) }` — the row at day 8 must carry module 2 / day 1 / task 1's title, which only holds if `Exercises()` read the module that declares `week: 2`.
- [ ] **Step 3:** Mutation check: temporarily make `Exercises()` iterate modules in reverse (or swap `r.Modules[0]` and `r.Modules[1]` inside the test before calling `Exercises()`): the assertion must FAIL; revert. Record in the Execution summary. `go test ./internal/airouter -run TestExercisesFlattens -v` green after revert.
- [ ] **Step 4:** Commit: `airouter: TestExercisesFlattens proves day 8 comes from the module declaring week 2`.

### Task 2: CODEMAP sentence

**Files:** Modify `harness/CODEMAP.md` — `airouter` bullet.

- [ ] **Step 1:** Where the bullet describes `ParseRoadmap`'s rules (the branch's 42f8fd6 wording), add one clause: "…`TestParseRoadmapRejects` asserts the reason of each rejection, not just that one happened."
- [ ] **Step 2:** Commit: `codemap: ParseRoadmap rejection table asserts reasons`.

## Verification
From `backend/` in the amended plan's worktree, on branch `harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut`:

```
git log --oneline -3                                   # 42f8fd6 is an ancestor; the two new commits on top
go test -timeout 60s ./internal/airouter -count=1 -v -run 'TestParseRoadmap' | grep -c '^    --- PASS'   # every row
gofmt -l . ; go vet ./...
# Mutation proof (Task 1 Step 6), recorded in the Execution summary:
#   with `if false && (…)` on the day-sum check -> FAIL: TestParseRoadmapRejects/three_five-minute_tasks_(15-minute_day) and …/(45-minute_day)
#   after revert -> PASS
make test                                              # whole backend unit suite green
git push origin harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut   # the reviewer re-reads this branch's CI run
```
