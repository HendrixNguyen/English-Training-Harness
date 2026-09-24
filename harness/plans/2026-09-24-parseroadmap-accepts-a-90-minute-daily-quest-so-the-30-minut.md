---
idea: harness/ideas/_inbox/parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md
status: approved
priority: medium
merged: false
---
# ParseRoadmap enforces the 30-minute day, non-empty titles at every level, and `week` = position — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Bug team — ticket **B2** of 2026-09-24. **Estimate:** 3 h. **Branch:** `harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut`.

**Idea (head):** `harness/ideas/_inbox/parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md`
**Also planned here (its frontmatter points at this plan):**
- `harness/ideas/_inbox/module-week-is-never-validated-and-day-number-comes-from-arr.md` → Task 2

**Goal:** A roadmap the model returns is written to `roadmaps.roadmap_json` only if every day adds up to roughly 30 minutes (§6.1 constraint 4), every roadmap/module/day/task has a title, and module *i* declares `week == i`, so `GET /quests/daily`'s "30 minutes required" header and `exercises.day_number` are always true of the stored content.

**Architecture:** All changes are inside `airouter.ParseRoadmap` (`backend/internal/airouter/roadmap.go`) and its test table. Two named constants carry the day budget with the §6.1 quote above them. Validation stays "reject, never repair" — a non-conforming answer is the caller's cue to retry (onboarding already retries once → `ai_bad_output`). Nothing outside `airouter` changes.

**Tech stack:** Go stdlib only.

**Spec:** 1st-thinking §6.1 constraint 4 (verbatim in `RoadmapSystemPrompt`): *"Each Daily Quest MUST be calculated to take approximately 30 minutes to complete, split into 3 distinct tasks (10 mins each)"*; §3.2 `exercises.day_number`.

**Root cause (re-read on `main` @ 9517f25):** `roadmap.go:116-126` — per-task `1..maxTaskMinutes(30)` only, no per-day sum, so `3×30` passes. `roadmap.go:113` — only `task.Title` is checked. `roadmap.go:34` — `Module.Week` is decoded and never compared with `mi+1`; `Exercises()` (`roadmap.go:152`) computes `day_number` from position, so a mis-numbered module produces stored JSON and `exercises` rows that disagree.

**⚠ Conflict note:** Feature ticket F2 (`harness/plans/2026-09-24-typed-task-content-with-answer-keys-so-every-quest-renders-a.md`) also edits `ParseRoadmap`, on the same day. F2 is written to add **one line** to the task loop (`if err := validateContent(task, …)`) and to keep its logic in a new `content.go`. Keep this plan's edits inside the existing loop structure and append new test-table rows **at the end** of `TestParseRoadmapRejects` so the daily merge is a two-line conflict at most. If `origin/main` has F2 when you start, `git fetch origin main && git merge origin/main --no-edit` first and keep both sides.

## Global Constraints

- Never edit app code in the main checkout; work in `.worktrees/<slug>` (AGENTS.md).
- `rg`/`timeout` not installed: `grep -n`, `go test -timeout 60s`.
- `DefaultTaskMinutes = 10` stays the default for a missing `duration_minutes` — `quests.toTask` and `TestParseRoadmapDefaultsAMissingDurationToTen` rely on it; the default is applied **before** the day-sum check.
- Every rejection wraps `ErrInvalidRoadmap` through `invalid(...)` and names module/day/task by 1-based index like the existing messages.
- `gofmt -l internal/airouter` must print nothing.

## Review Focus

1. A day of `10 + 10 + 0` (missing third duration → defaults to 10) sums to 30 and must pass — Task 1 test "missing duration counts as ten".
2. A day of `15 + 15 + 5` sums to 35 → passes the day band but each task must also be within `5..15` — both bands apply; Task 1 test "task band still applies".
3. A whitespace-only roadmap title (`"  "`) must be rejected like an empty one — Task 3 uses `strings.TrimSpace`.
4. `week: 0` on module 1 (a model that counts from zero) is rejected with a message that says `want 1` — Task 2.
5. The existing `absurd duration` (120) case still fails, now by the task band — no test loses its reason to exist; Task 1 keeps it.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/airouter/roadmap.go` | Constants `minTaskMinutes`, `maxTaskMinutes` (now 15), `minDayMinutes`, `maxDayMinutes`; day-sum, title and week checks inside the existing loops; doc comment updated |
| `backend/internal/airouter/roadmap_test.go` | New rows in `TestParseRoadmapRejects`; new `TestParseRoadmapAcceptsTheDayBudgetEdges`; extended `TestExercisesFlattensTo84RowsCarryingTitleAndDuration` |
| `harness/CODEMAP.md` | `airouter` bullet: "empty **task** titles" → the new list |

Run every command from `backend/` in the worktree.

---

## Tasks

### Task 1: The day budget — tasks in `5..15`, day in `20..40`

**Files:**
- Modify: `backend/internal/airouter/roadmap.go:10-17` (constants), `:115-126` (task loop), `:96-100` (day loop)
- Test: `backend/internal/airouter/roadmap_test.go`

**Interfaces:**
- Produces: exported constant `DefaultTaskMinutes` unchanged; unexported `minTaskMinutes = 5`, `maxTaskMinutes = 15`, `minDayMinutes = 20`, `maxDayMinutes = 40`.

- [ ] **Step 1: Add the failing rows** to the `cases` map in `TestParseRoadmapRejects` (append at the end of the map literal):

```go
		"three thirty-minute tasks (90-minute day)": validRoadmapJSON(t, func(r *Roadmap) {
			for i := range r.Modules[0].Days[0].Tasks {
				r.Modules[0].Days[0].Tasks[i].DurationMinutes = 30
			}
		}),
		"three three-minute tasks (9-minute day)": validRoadmapJSON(t, func(r *Roadmap) {
			for i := range r.Modules[1].Days[2].Tasks {
				r.Modules[1].Days[2].Tasks[i].DurationMinutes = 3
			}
		}),
		"one 20-minute task in an otherwise normal day": validRoadmapJSON(t, func(r *Roadmap) { r.Modules[2].Days[4].Tasks[1].DurationMinutes = 20 }),
```

and add a new acceptance test after `TestParseRoadmapDefaultsAMissingDurationToTen`:

```go
func TestParseRoadmapAcceptsTheDayBudgetEdges(t *testing.T) {
	cases := map[string]func(r *Roadmap){
		"exactly 30":                       nil,
		"missing duration counts as ten":   func(r *Roadmap) { r.Modules[0].Days[0].Tasks[2].DurationMinutes = 0 },
		"task band still applies at 15/15/5": func(r *Roadmap) {
			d := &r.Modules[3].Days[6]
			d.Tasks[0].DurationMinutes, d.Tasks[1].DurationMinutes, d.Tasks[2].DurationMinutes = 15, 15, 5
		},
		"lower day edge 5+5+10 = 20": func(r *Roadmap) {
			d := &r.Modules[1].Days[1]
			d.Tasks[0].DurationMinutes, d.Tasks[1].DurationMinutes, d.Tasks[2].DurationMinutes = 5, 5, 10
		},
		"upper day edge 15+15+10 = 40": func(r *Roadmap) {
			d := &r.Modules[2].Days[3]
			d.Tasks[0].DurationMinutes, d.Tasks[1].DurationMinutes, d.Tasks[2].DurationMinutes = 15, 15, 10
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseRoadmap(validRoadmapJSON(t, mutate)); err != nil {
				t.Fatalf("ParseRoadmap rejected a day inside the §6.1 budget: %v", err)
			}
		})
	}
}
```

- [ ] **Step 2: Run to see the new rows fail**

Run: `go test -timeout 60s ./internal/airouter -run 'TestParseRoadmapRejects|TestParseRoadmapAcceptsTheDayBudgetEdges' -v`
Expected: the three new `Rejects` sub-tests FAIL with `ParseRoadmap accepted it`; the edge cases PASS already (they are inside today's `1..30`).

- [ ] **Step 3: Implement** — constants block becomes:

```go
// The §6.1 shape.
const (
	Modules            = 4
	DaysPerModule      = 7
	TasksPerDay        = 3
	DefaultTaskMinutes = 10
)

// §6.1 constraint 4: "Each Daily Quest MUST be calculated to take
// approximately 30 minutes to complete, split into 3 distinct tasks (10 mins
// each)". "Approximately" is read as ±10 minutes on the day and ±5 on a task;
// quests hard-codes total_minutes_required: 30 and the pet needs 1800 s, so
// a day outside this band is a promise the learner cannot keep.
const (
	minTaskMinutes = 5
	maxTaskMinutes = 15
	minDayMinutes  = 20
	maxDayMinutes  = 40
)
```

the task-duration check becomes:

```go
				if task.DurationMinutes == 0 {
					task.DurationMinutes = DefaultTaskMinutes
				}
				if task.DurationMinutes < minTaskMinutes || task.DurationMinutes > maxTaskMinutes {
					return Roadmap{}, invalid("module %d day %d task %d duration %d is outside %d..%d", mi+1, di+1, ti+1, task.DurationMinutes, minTaskMinutes, maxTaskMinutes)
				}
				dayMinutes += task.DurationMinutes
```

with `dayMinutes := 0` declared next to `seen := map[string]bool{}` and, after the task loop closes (still inside the day loop):

```go
			if dayMinutes < minDayMinutes || dayMinutes > maxDayMinutes {
				return Roadmap{}, invalid("module %d day %d adds up to %d minutes, want %d..%d (§6.1: approximately 30)", mi+1, di+1, dayMinutes, minDayMinutes, maxDayMinutes)
			}
```

- [ ] **Step 4: Run the package** — `go test -timeout 60s ./internal/airouter` → PASS (the old `absurd duration` row still fails validation, now by the task band).

- [ ] **Step 5: Commit**

```bash
gofmt -l internal/airouter && go vet ./internal/airouter
git add internal/airouter/roadmap.go internal/airouter/roadmap_test.go
git commit -m "airouter: ParseRoadmap enforces the §6.1 day budget (tasks 5..15, day 20..40)"
```

### Task 2: `week` must equal the module's position

**Files:**
- Modify: `backend/internal/airouter/roadmap.go` (module loop, just after `m := &r.Modules[mi]`)
- Test: `backend/internal/airouter/roadmap_test.go`

- [ ] **Step 1: Add failing rows** to `TestParseRoadmapRejects` (append):

```go
		"weeks reversed":      validRoadmapJSON(t, func(r *Roadmap) { r.Modules[0].Week, r.Modules[3].Week = 4, 1 }),
		"duplicate week":      validRoadmapJSON(t, func(r *Roadmap) { r.Modules[1].Week = 1 }),
		"week counts from 0":  validRoadmapJSON(t, func(r *Roadmap) { r.Modules[0].Week = 0 }),
```

and extend `TestExercisesFlattensTo84RowsCarryingTitleAndDuration`: after the `day numbers` check add

```go
	// day 8 is the first day of module 2 — and module 2 must be the one that
	// declared week 2, otherwise stored roadmap_json and exercises disagree.
	if r.Modules[1].Week != 2 || ex[21].DayNumber != (r.Modules[1].Week-1)*DaysPerModule+1 {
		t.Errorf("exercise[21] day %d should come from the module declaring week 2 (got week %d)", ex[21].DayNumber, r.Modules[1].Week)
	}
```

- [ ] **Step 2: Run to see them fail**

Run: `go test -timeout 60s ./internal/airouter -run TestParseRoadmapRejects -v` → the three new rows FAIL with `ParseRoadmap accepted it`.

- [ ] **Step 3: Implement** — first statement inside `for mi := range r.Modules {` after `m := &r.Modules[mi]`:

```go
		if m.Week != mi+1 {
			// Exercises() derives day_number from position; a module that says
			// otherwise would store JSON the frontend renders as one week while
			// exercises serve another. Reject — the retry gives the model a
			// checkable instruction; sorting would hide the disagreement.
			return Roadmap{}, invalid("module %d declares week %d, want %d", mi+1, m.Week, mi+1)
		}
```

- [ ] **Step 4: Run the package** → PASS. Also run `go test -timeout 120s ./internal/onboarding` — its `fixtureRoadmap()` sets `Week: m` and 10-minute tasks, so it must still pass.

- [ ] **Step 5: Commit**

```bash
git add internal/airouter/roadmap.go internal/airouter/roadmap_test.go
git commit -m "airouter: ParseRoadmap rejects a module whose week disagrees with its position"
```

### Task 3: Titles are required at every level, not just on tasks

**Files:**
- Modify: `backend/internal/airouter/roadmap.go` (after the `cefr_level` check; module loop; day loop)
- Test: `backend/internal/airouter/roadmap_test.go`

- [ ] **Step 1: Add failing rows** (append to `TestParseRoadmapRejects`):

```go
		"empty roadmap title":     validRoadmapJSON(t, func(r *Roadmap) { r.Title = "" }),
		"blank roadmap title":     validRoadmapJSON(t, func(r *Roadmap) { r.Title = "   " }),
		"empty module title":      validRoadmapJSON(t, func(r *Roadmap) { r.Modules[2].Title = "" }),
		"empty day title":         validRoadmapJSON(t, func(r *Roadmap) { r.Modules[0].Days[5].Title = "" }),
```

- [ ] **Step 2: Run** → the four rows FAIL with `ParseRoadmap accepted it`.

- [ ] **Step 3: Implement** — after the `cefrLevels` check:

```go
	if strings.TrimSpace(r.Title) == "" {
		return Roadmap{}, invalid("roadmap has no title")
	}
```

inside the module loop after the week check: `if strings.TrimSpace(m.Title) == "" { return Roadmap{}, invalid("module %d has no title", mi+1) }`; inside the day loop after `d := &m.Days[di]`: `if strings.TrimSpace(d.Title) == "" { return Roadmap{}, invalid("module %d day %d has no title", mi+1, di+1) }`. `Module.Focus` stays unchecked (it is descriptive; the frontend does not render it — YAGNI).

- [ ] **Step 4: Update the `ParseRoadmap` doc comment** to read: "… exactly 4 modules × 7 days × 3 tasks with the three task types each present once, module i declaring week i, non-empty roadmap/module/day/task titles, task durations within 5..15 and each day summing to 20..40 minutes (a missing duration becomes 10 before the sum), and a valid cefr_level. …"

- [ ] **Step 5: Run the package and commit**

```bash
go test -timeout 60s ./internal/airouter && gofmt -l internal/airouter
git add internal/airouter/roadmap.go internal/airouter/roadmap_test.go
git commit -m "airouter: ParseRoadmap requires roadmap, module and day titles"
```

### Task 4: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (`airouter` bullet)

- [ ] **Step 1:** Replace `ParseRoadmap rejects fences, preamble, trailing tokens, wrong counts, duplicate/unknown task types, empty titles, durations outside 1..30 and bad CEFR (missing duration → 10)` with `ParseRoadmap rejects fences, preamble, trailing tokens, wrong counts, a module whose \`week\` ≠ its position, duplicate/unknown task types, empty roadmap/module/day/task titles, task durations outside 5..15, a day summing outside 20..40 minutes (§6.1 "approximately 30"; a missing duration becomes 10 before the sum) and bad CEFR`.

- [ ] **Step 2: Commit** — `git add harness/CODEMAP.md && git commit -m "codemap: ParseRoadmap day budget, week and title rules"`.

## Verification

From `backend/`:

```bash
gofmt -l . ; go vet ./... && go test -timeout 120s ./internal/airouter ./internal/onboarding -count=1 -v 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
go test -timeout 300s ./...
```

Expected: no gofmt output, `ok …/internal/airouter`, `ok …/internal/onboarding`, whole backend PASS. Push the branch; `gh run list --branch <branch>` green.

## Notes and open questions

- The bands (`5..15`, `20..40`) are the evaluator's reading of "approximately"; the owner can tighten them by editing two constants — no other code depends on the numbers.
- A model that consistently overshoots will now hit `ai_bad_output` after one retry instead of storing an unachievable day; if that happens in production the answer is a stronger prompt line, not a wider band.
