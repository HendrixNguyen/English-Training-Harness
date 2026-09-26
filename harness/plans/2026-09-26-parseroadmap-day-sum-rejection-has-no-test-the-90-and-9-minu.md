---
idea: harness/ideas/_inbox/parseroadmap-day-sum-rejection-has-no-test-the-90-and-9-minu.md
status: done
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

- [x] **Step 1:** Change the table type from `map[string]string` to `map[string]struct{ raw, want string }` where `want` is a substring of the `invalid(...)` message. Fill `want` for every existing row from the branch's `roadmap.go` messages, e.g. `"three modules"` → `"want 4 modules"` (use the real text: `grep -n 'invalid("' internal/airouter/roadmap.go`), `"duplicate task type"` → `"repeats task type"`, `"empty task title"` → `"has no title"`, `"absurd duration"` → `"outside 5..15"`, `"weeks reversed"` / `"duplicate week"` / `"week counts from 0"` → `"declares week"` (or the branch's wording, which must contain `want 1` for the counts-from-0 row), `"empty roadmap title"` / `"blank roadmap title"` / `"empty module title"` / `"empty day title"` → `"has no title"`, `"bad cefr"` → its message, and `"preamble"` / `"trailing garbage"` / `"empty"` / `"not an object"` → the decode-error wording the branch uses (read it; if those four wrap a `json` error with no stable text, `want` may be `""`, meaning "any reason", and only for those four — say so in a comment).
- [x] **Step 2:** Rename the two misnamed rows to what they test: `"three thirty-minute tasks (90-minute day)"` → `"thirty-minute task (task band)"` with `want: "outside 5..15"`; `"three three-minute tasks (9-minute day)"` → `"three-minute task (task band)"` with `want: "outside 5..15"`. Keep their mutations.
- [x] **Step 3:** Append the two rows only the day sum can reject, each task inside `5..15`:

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

- [x] **Step 4:** In the loop, after the `errors.Is` check add: `if tc.want != "" && !strings.Contains(err.Error(), tc.want) { t.Errorf("err = %q, want it to mention %q", err, tc.want) }`. (`strings` is already imported on the branch.)
- [x] **Step 5:** `go test -timeout 60s ./internal/airouter -run TestParseRoadmapRejects -v` → every row PASS. `gofmt -l internal/airouter` → empty.
- [x] **Step 6: Mutation check (the review's own evidence).** Temporarily edit `roadmap.go`: `if false && (dayMinutes < minDayMinutes || dayMinutes > maxDayMinutes) {`. Run the test: the two new rows **must FAIL** with "ParseRoadmap accepted it". Revert the edit (`git checkout -- internal/airouter/roadmap.go`; `git diff --stat` shows only `roadmap_test.go`). Paste the failing output into this plan's Execution summary.
- [x] **Step 7:** Commit: `airouter: TestParseRoadmapRejects keys every row by its rejection reason; day-sum rows at 15 and 45 minutes`.

### Task 1b: The week-2 provenance assertion proves provenance (folded: `week-2-provenance-assertion-in-testexercisesflattens-repeats.md`)

**Files:** `backend/internal/airouter/roadmap_test.go` — `validRoadmapJSON` fixture (~line 20) and `TestExercisesFlattensTo84RowsCarryingTitleAndDuration` (~lines 150–157).

- [x] **Step 1:** In `validRoadmapJSON`, make every task title unique per module/day/type: `Title: fmt.Sprintf("w%d-d%d-%s", m, d, tt)` (the `"<type> task"` string is asserted nowhere else — check with `grep -n '" task"' internal/airouter/*_test.go internal/onboarding/*_test.go`; if a test does depend on it, update that assertion in the same commit).
- [x] **Step 2:** Replace the branch's week-2 assertion (`r.Modules[1].Week != 2 || ex[21].DayNumber != …`) with: `if want := r.Modules[1].Days[0].Tasks[0].Title; ex[21].Title != want || ex[21].DayNumber != 8 { t.Errorf(...) }` — the row at day 8 must carry module 2 / day 1 / task 1's title, which only holds if `Exercises()` read the module that declares `week: 2`.
- [x] **Step 3:** Mutation check: temporarily make `Exercises()` iterate modules in reverse (or swap `r.Modules[0]` and `r.Modules[1]` inside the test before calling `Exercises()`): the assertion must FAIL; revert. Record in the Execution summary. `go test ./internal/airouter -run TestExercisesFlattens -v` green after revert.
- [x] **Step 4:** Commit: `airouter: TestExercisesFlattens proves day 8 comes from the module declaring week 2`.

### Task 2: CODEMAP sentence

**Files:** Modify `harness/CODEMAP.md` — `airouter` bullet.

- [x] **Step 1:** Where the bullet describes `ParseRoadmap`'s rules (the branch's 42f8fd6 wording), add one clause: "…`TestParseRoadmapRejects` asserts the reason of each rejection, not just that one happened."
- [x] **Step 2:** Commit: `codemap: ParseRoadmap rejection table asserts reasons`.

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

## Execution summary

**Environment deviation (setup, not the plan's own instructions):** the amended plan's normal worktree (`.claude/worktrees/daily-task-evaluation-planning-971a45/.worktrees/parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut`) was checked out by another live session and was not touched. Per the run's setup instructions a fresh **detached** worktree was created instead: `git worktree add --detach .worktrees/parseroadmap-day-sum-amend origin/harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut` (HEAD landed on `42f8fd6`, matching the plan's stated tip). The run's setup instructions also said to `git merge origin/main --no-edit` into that worktree; this was **skipped** because `origin/main` (`67ad0c0`) is not an ancestor of `42f8fd6` — it carries unrelated commits from other daily branches — and this plan's own Global Constraints explicitly say "Do not merge `origin/main` into it unless it already contains the branch's own commits; the reviewer compares against `42f8fd6`." Merging would have pulled unrelated history into a branch the reviewer diffs against that specific commit, and the plan is a test-only change with no dependency on anything newer on `main`. This is a deviation from the run's generic setup instructions, not from the plan.

**Task 1 (message text used, not the plan's illustrative wording):** read `internal/airouter/roadmap.go`'s actual `invalid(...)` call sites and keyed every row to the real substring, e.g. "three modules"/"five modules" → `"modules, want 4"`, "duplicate task type" → `"repeats task type"`, week rows → `"declares week N, want M"` (the counts-from-0 row is `"declares week 0, want 1"`, containing "want 1" as required). Determined that all four of "preamble"/"trailing garbage"/"empty"/"not an object" hit the pre-decode guards with **stable** fixed text (empty check, `{`-prefix check, trailing-token check) and never the generic `"decoding: %v"` wrapper — so none needed `want: ""`; each got a real substring, documented in a code comment. Renamed the two mislabeled rows to `"thirty-minute task (task band)"` / `"three-minute task (task band)"` (they were only ever hitting the per-task 5..15 check, never the day sum, since a task-level rejection short-circuits before the day total is summed) and appended the two rows that only the day-sum check can reject: `"three five-minute tasks (15-minute day)"` (want `"adds up to 15 minutes"`) and `"three fifteen-minute tasks (45-minute day)"` (want `"adds up to 45 minutes"`).

Mutation check (Task 1 Step 6): with `if false && (dayMinutes < minDayMinutes || dayMinutes > maxDayMinutes) {` in `roadmap.go`, `go test ./internal/airouter -run TestParseRoadmapRejects -v` gave:
```
--- FAIL: TestParseRoadmapRejects/three_five-minute_tasks_(15-minute_day): roadmap_test.go:154: ParseRoadmap accepted it
--- FAIL: TestParseRoadmapRejects/three_fifteen-minute_tasks_(45-minute_day): roadmap_test.go:154: ParseRoadmap accepted it
```
All 24 other rows still PASS (task-band and every other rule are unaffected). After reverting the mutation (`git diff --stat` showed only `roadmap_test.go` — production code back to the committed 42f8fd6 state), all 26 rows PASS and `gofmt -l internal/airouter` is empty.

**Task 1b (adapted the plan's illustrative code to the real `Exercise` type):** made every fixture task title unique (`w%d-d%d-%s`, module/day/type) in `validRoadmapJSON`. `grep -n '" task"' internal/airouter/*_test.go internal/onboarding/*_test.go` found a second, unrelated fixture (`internal/onboarding/fakes_test.go`'s `fixtureRoadmap`) using the same `tt + " task"` pattern — confirmed it is a separate function in a separate package, unaffected by this change; `internal/onboarding/integration_test.go`'s `"reading task"` assertion reads from that fixture, not from `airouter`'s. `Exercise` has no `Title` field (only `DayNumber`, `TaskType`, `ContentJSON`), so the plan's suggested `ex[21].Title` was adapted to unmarshal `ex[21].ContentJSON` and compare its `title` to `r.Modules[1].Days[0].Tasks[0].Title`.

Mutation check (Task 1b Step 3): the plan's first suggested mutation — swapping `r.Modules[0]`/`r.Modules[1]` inside the test before calling `Exercises()` — was tried first and did **not** fail, because for an already-validated `Roadmap` every module's `Week` equals its index+1, so swapping the slice moves the "week 2" module to `r.Modules[1]` right along with the assertion's own `want` (both sides read `r.Modules[1]` after the swap); the check is tautological against that particular mutation. Used the plan's other offered mutation instead — temporarily reading module content out of order inside `Exercises()` itself (`m := r.Modules[len(r.Modules)-1-mi]` while `DayNumber` stayed keyed off position `mi`) — which produced:
```
--- FAIL: TestExercisesFlattensTo84RowsCarryingTitleAndDuration
    roadmap_test.go:192: exercise[21] = day 8 title "w3-d1-vocabulary", want day 8 title "w2-d1-vocabulary..."
```
Reverted; `git diff --stat` showed only `roadmap_test.go` changed. `go test ./internal/airouter -run TestExercisesFlattens -v` → PASS after revert.

**Verification (all run from `backend/` in `.worktrees/parseroadmap-day-sum-amend`):**
```
$ git log --oneline -3
91f4a58 codemap: ParseRoadmap rejection table asserts reasons
24a3f82 airouter: TestExercisesFlattens proves day 8 comes from the module declaring week 2
6e378a3 airouter: TestParseRoadmapRejects keys every row by its rejection reason; day-sum rows at 15 and 45 minutes
$ go test -timeout 60s ./internal/airouter -count=1 -v -run 'TestParseRoadmap' | grep -c '^    --- PASS'
32
$ gofmt -l internal/airouter ; echo clean
clean
$ go vet ./internal/airouter/...
(no output)
$ make test
ok  backend/internal/airouter, auth, config, google, health, notify, onboarding, pet, quests, store   (all ok, no FAIL)
$ git push origin HEAD:harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut
   42f8fd6..91f4a58  HEAD -> harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut
```
CI on the pushed branch: run [36216052193](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36216052193) — `frontend`, `backend-unit`, `backend-integration`, `harness-tooling` all green (`success`).

Note: repo-wide `gofmt -l .` (not part of this plan's scope) flags two pre-existing, untouched files (`internal/quests/handler_test.go`, `internal/quests/repo.go`) already dirty at `42f8fd6`; left alone as out of scope.

No process or container was started for this plan (test-only Go changes); nothing to tear down.
