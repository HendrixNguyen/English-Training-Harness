---
idea: harness/ideas/2026-09-25-run-01/roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md
status: approved
priority: medium
merged: false
design: harness/designs/roadmap-tree.md
---
# Roadmap tree shows the real plan: module and day titles with true per-day completion — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/2026-09-25-run-01/roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md`
**Design:** `harness/designs/roadmap-tree.md` (layout, the five day states, components, tokens, copy — follow it exactly).

**Goal:** `/roadmap` shows the learner's actual 28-day plan — roadmap title and level, four module headers, every day named — and tells the truth about each day: a star only where `daily_progress.is_target_met` is true, the minutes where the target was missed, a muted mark where nothing happened, today, and the locked days ahead.

**Why now (`priority: medium`):** Confirmed on `origin/main` today: `frontend/utils/roadmap.ts` derives all 28 nodes from `GET /quests/daily`'s `day_number` with `state: day < today ? 'completed' : …` under the comment "There is no per-day completion endpoint, so 'completed' means 'before today' … (open question)". A skipped day is shown as ⭐ "Đã hoàn thành" while the pet lost health for it, and no day has a name although `roadmaps.roadmap_json` (the `airouter.Roadmap` that `onboarding.PgRepo.SaveAssessment` marshals) carries a title, focus and three task titles for every day — Google Tasks users already see them ("Day N: title · title · title", `google.TaskTitle`). All data exists; this is a read path. Feature slot 4 of 5 under the two-cap rule (owner, 2026-09-25).

**Design decisions (taken by the evaluator; do not re-litigate):**
1. **One new read in `quests`:** `GET /api/v1/roadmap` behind `auth.Require()`, body `{roadmap_id, title, cefr_level, created_at, day_number, modules[4]{week, title, focus, days[7]{day_number, date, title, tasks[3]{task_type, title, duration_minutes}, minutes_spent, is_target_met}}}`; `404 {"error":"no_active_roadmap"}` exactly like `/quests/daily`; `500 internal_error` otherwise. Exercise `content_json` is **not** included — this is the outline, not the exercise payload. `quests` stays inside its boundary: `roadmaps`, `daily_progress` and `users.timezone` through its own SQL; no pet tables; no Redis for this read (the live counter is `/quests/daily`'s job).
2. **`date` for day N is the inverse of `DayNumber`.** New `DayDate(createdAt, n, loc)` in `day.go`, built on the same `startOfDay` that `DayNumber` uses: the local date of `roadmaps.created_at` plus N−1 calendar days in `users.timezone` — the rule `google.DayDue` already applies to Tasks `due` (`google` cannot be imported: package boundary, and it returns a UTC-midnight `time.Time`, not the string quests keys `daily_progress` by). A round-trip test on `day_test.go`'s DST fixtures pins `DayNumber(created, DayDate(created, n)) == n`, so the tree and the daily suite can never disagree about which date a day is.
3. **The join is one range query:** `daily_progress` rows for `DayDate(1) ≤ date ≤ DayDate(28)` keyed by `YYYY-MM-DD`; a day with no row is `minutes_spent: 0, is_target_met: false`. `minutes_spent` is what `RecordProgress` upserted (every call), `is_target_met` is the durable flag (set on the crossing call after the pet hook) — the tree reports what the pet was told, nothing more (design §4.1).
4. **Frontend: a new `stores/roadmap.ts`**, not a section of `quest.ts`. Different resource (an outline, read-only, refetched on page open) with a different lifecycle (`quest.daily` is mutated on every progress call), and today's approved bug plan `harness/plans/2026-09-25-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md` edits `stores/quest.ts` — a separate file keeps the daily merge clean. `utils/roadmap.ts` `roadmapNodes(outline)` takes the outline (which carries per-day progress) instead of an integer; `dayState` precedence is design §4.1: `today` → `completed` → `partial` → `missed` → `locked`, by `day_number` relative to the server's `day_number`, never by the client clock.
5. **Same-day overlap (must hold for the merge):** the pet bug plan above edits `quests/service.go` (`ProgressResult` fields, the tail of `RecordProgress`), lines in `service_test.go`, appends to `handler_test.go`, and edits `stores/quest.ts` + `stores/pet.ts` + `tests/unit/petStore.test.ts`. This plan therefore puts all new backend code in **new files** (`roadmap.go`, `roadmap_test.go`), appends only to `day.go`, `day_test.go`, `fakes_test.go`, `integration_test.go` and the two interfaces in `repo.go`, makes a **one-line insertion** inside `newQuestRouter` in `handler_test.go` (not an append at the end), and never reorders, reformats or re-indents an existing declaration. Nothing here touches `service.go`, `service_test.go`, `stores/quest.ts` or `stores/pet.ts` (the util imports `TASK_ORDER` from `stores/quest.ts` read-only).
6. **Specs:** backend spec §6.2 wins for the wire; both spec files gain the route in their own escaped style (Task 4). CODEMAP `quests` and `shell` paragraphs are updated (Task 7); no other `harness/` file changes on the branch.

**Tech stack:** Go 1.25, Gin, pgx/v5 (backend; `quests` gains a production import of `internal/airouter` for the `Roadmap` type — airouter has no routes and no tables, and quests' integration test already imports it). Nuxt 3, Pinia, Tailwind, Vitest + `@vue/test-utils` (frontend). No new dependencies.

**Run every command from the worktree root** unless a step says otherwise; `backend/` and `frontend/` commands say so. `rg` and `timeout` are not installed — use `grep -n`, bound long commands with the tool's own flag. Backend unit tests run with `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL` in front so nothing touches a live service. Integration tests use an isolated compose stack (Task 2, Step 8).

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/quests/day.go` | **append** `DayDate` (inverse of `DayNumber`, on `startOfDay`) |
| `backend/internal/quests/day_test.go` | **append** the round-trip test over the DST fixtures and a late-in-the-day creation |
| `backend/internal/quests/repo.go` | `QuestRepo` gains `ActiveRoadmapDoc`; `ProgressRepo` gains `ProgressBetween` (two interface lines each with a doc comment; nothing else moves) |
| `backend/internal/quests/roadmap.go` | **new** — storage types, the two SQL constants and `PgRepo` methods, the wire DTOs, `Service.Roadmap`, `RoadmapHandler` |
| `backend/internal/quests/roadmap_test.go` | **new** — service tests with fakes, handler tests (body keys, no `content`, 404) |
| `backend/internal/quests/fakes_test.go` | **append** `roadmapJSON` field on `fakeQuestRepo` (last field), `ActiveRoadmapDoc` and `ProgressBetween` fake methods at the end of the file |
| `backend/internal/quests/handler_test.go` | one line inside `newQuestRouter`: `g.GET("/roadmap", RoadmapHandler(svc))` |
| `backend/internal/quests/integration_test.go` | **append** `TestIntegrationRoadmapOutlineJoinsDailyProgress` |
| `backend/cmd/api/main.go` | one line after the `/quests/progress` route |
| `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` | §6.2: the `GET /api/v1/roadmap` block after `POST /api/v1/quests/progress` |
| `project-base/1st-thinking-architecture-doc.md` | §7: one escaped bullet after `POST /api/v1/quests/progress` |
| `frontend/stores/roadmap.ts` | **new** — `RoadmapOutline` types, `useRoadmapStore` (`load`, `noRoadmap`, `completedDays`) |
| `frontend/utils/roadmap.ts` | rewritten: `dayState`, `roadmapNodes(outline)`, `completedDays`, `statusText`, `formatDayDate`, `TASK_LABELS`; the "open question" comment is gone |
| `frontend/components/roadmap/RoadmapMarker.vue` | **new** — the 24 px marker for one state |
| `frontend/components/roadmap/RoadmapModuleHeader.vue` | **new** — week, title, focus, `{met}/7` |
| `frontend/components/roadmap/RoadmapNode.vue` | rewritten: the day row with `aria-expanded` toggle, task panel, today's "Học ngay →" |
| `frontend/pages/roadmap.vue` | uses `useRoadmapStore`; module headers; today expanded and scrolled into view |
| `frontend/service-worker/sw.ts` | `/api/v1/roadmap` joins the `api-state` NetworkFirst route |
| `frontend/tests/unit/roadmap.test.ts` | rewritten for the new util |
| `frontend/tests/unit/roadmapStore.test.ts` | **new** |
| `frontend/tests/unit/roadmapPage.test.ts` | **new** — module rendering, states, expand, empty state |
| `harness/CODEMAP.md` | `quests` and `shell` paragraphs |

---

## Tasks

### Task 1: `DayDate` — the inverse of `DayNumber`

**Files:**
- Modify (append only): `backend/internal/quests/day.go`, `backend/internal/quests/day_test.go`

- [ ] **Step 1: Write the failing tests** at the end of `day_test.go`:
  - `TestDayDateIsTheInverseOfDayNumberAcrossDST`: reuse the `at(loc, y, m, d)` helper shape and the same zone/created fixtures as `TestDayNumberCountsCalendarDaysAcrossDSTTransitions` (London 2026-03-25 and 2026-10-20, New York 2026-03-05 and 2026-10-28, Sydney 2026-09-29 and 2026-04-01; 09:00 local). For each `created` and every `n` in 1..28: `d := DayDate(created, n, loc)`; parse `d` back as `time.ParseInLocation("2006-01-02", d, loc)` plus 9 h; assert `DayNumber(created, that, loc) == n`. Also assert the specific values `DayDate(london 2026-03-25, 5, loc) == "2026-03-29"` (the spring-forward day itself) and `DayDate(london 2026-03-25, 28, loc) == "2026-04-21"`.
  - `TestDayDateStartsOnTheLocalCreationDate`: `created := time.Date(2026, 9, 1, 18, 30, 0, 0, time.UTC)` (01:30 on 2026-09-02 in Ho Chi Minh City): `DayDate(created, 1, Location("UTC")) == "2026-09-01"`, `DayDate(created, 1, Location("Asia/Ho_Chi_Minh")) == "2026-09-02"`, `DayDate(created, 2, Location("Asia/Ho_Chi_Minh")) == "2026-09-03"` — a roadmap created late in the day still counts that day as day 1, in the learner's zone.
- [ ] **Step 2: Run red:** from `backend/`: `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./internal/quests/ -run TestDayDate -count=1` → compile error (`DayDate` undefined).
- [ ] **Step 3: Append to `day.go`** (after `startOfDay`; touch nothing above it):

```go
// DayDate is the inverse of DayNumber: the YYYY-MM-DD of roadmap day n
// (1-based) in the user's timezone — the local date of created_at plus n-1
// calendar days. It is the key GET /roadmap joins daily_progress on, and the
// same rule google.DayDue applies to Google Tasks `due`. Calendar days again,
// not 24-hour multiples: AddDate re-resolves the wall-clock date, so a
// spring-forward or fall-back day is still one day.
func DayDate(createdAt time.Time, n int, loc *time.Location) string {
	return startOfDay(createdAt, loc).AddDate(0, 0, n-1).Format("2006-01-02")
}
```

- [ ] **Step 4: Run green:** the same command → `ok`; then `env -u … go test ./internal/quests/ -count=1` → `ok`.
- [ ] **Step 5: Commit:** `git add backend/internal/quests/day.go backend/internal/quests/day_test.go && git commit -m "quests: DayDate, the inverse of DayNumber, pinned to the DST fixtures"`.

### Task 2: Repo reads — the roadmap document and the progress range

**Files:**
- Modify: `backend/internal/quests/repo.go` (two interface methods), `backend/internal/quests/fakes_test.go` (append), `backend/internal/quests/integration_test.go` (append)
- Create: `backend/internal/quests/roadmap.go` (storage part), `backend/internal/quests/roadmap_test.go` (fake-backed tests come in Task 3; create the file here with the fixture helper)

- [ ] **Step 1: Write the failing integration test** at the end of `integration_test.go`, copying the setup of `TestIntegrationDailyAndProgressAgainstRealServices` (the gate, `store.Migrate`, the `gid`-based user insert and cleanup, the `onboarding.NewPgRepo(pg.Pool).SaveAssessment(… integrationRoadmap())` seed with `Timezone: "UTC"`), named `TestIntegrationRoadmapOutlineJoinsDailyProgress`, using a distinct `gid` (`google-roadmap-integration`) and email so it can run in the same database as its sibling:
  - `repo := NewPgRepo(pg.Pool)`; `doc, err := repo.ActiveRoadmapDoc(ctx, userID)` → `doc.ID != ""`, `doc.CreatedAt` within the last minute, `json.Valid(doc.JSON)` and `doc.JSON` contains `"modules"`.
  - `loc := Location("UTC")`; `d1 := DayDate(doc.CreatedAt, 1, loc)` (today), `d2 := DayDate(doc.CreatedAt, 2, loc)`. `repo.Upsert(ctx, userID, d1, 30)` then `repo.MarkTargetMet(ctx, userID, d1)`; `repo.Upsert(ctx, userID, d2, 12)`; and one row **outside** the range: `repo.Upsert(ctx, userID, "1999-01-01", 5)`.
  - `rows, err := repo.ProgressBetween(ctx, userID, d1, DayDate(doc.CreatedAt, 28, loc))` → `len(rows) == 2`, `rows[d1] == DayProgress{MinutesSpent: 30, IsTargetMet: true}`, `rows[d2] == DayProgress{MinutesSpent: 12, IsTargetMet: false}`, no `"1999-01-01"` key.
  - Then the service end to end (this part goes green only after Task 3; write it now): `svc := NewService(NewRedisCounter(rdb), repo, repo, NopPet{}, func() time.Time { return time.Now().UTC() })`; `out, err := svc.Roadmap(ctx, userID)` → `out.DayNumber == 1`, `len(out.Modules) == 4`, each `len(m.Days) == 7`, `out.Modules[0].Days[0]` has `Date == d1, MinutesSpent == 30, IsTargetMet == true, Title == "Day 1"`, `Days[1]` has `12 / false`, `Days[2]` has `0 / false`, every day has 3 tasks with `DurationMinutes == 10` and `Title` starting `"Day task: "`.
  - Cleanup deletes the user (cascades roadmaps/exercises/daily_progress) as the sibling does. This raises the package's `TestIntegration*` count from 1 to 2 and the repo's from 12 to 13 on this branch (CI's `backend-integration` job derives `want` from the tree, so the daily integration branch's total is whatever the day's branches add together — the auth plan adds one too).
- [ ] **Step 2: Write the failing fake-backed test scaffolding** in a new `roadmap_test.go`: a helper `func outlineJSON(t *testing.T) json.RawMessage` that returns `json.Marshal(integrationRoadmap())` (the 4×7×3 fixture already in `integration_test.go`, same package, no build tag) and a `newRoadmapHarness(t, now)` that calls `newHarness(t, now)` and sets `h.quests.roadmapJSON = outlineJSON(t)`. Then the first repo-level fake test: `TestFakeProgressBetweenFiltersByDate` — seed `h.progress.rows` for `u1|2026-09-01` (30, met), `u1|2026-09-03` (12), `u1|2026-10-30` (5), `u2|2026-09-02` (9); `ProgressBetween(ctx, "u1", "2026-09-01", "2026-09-28")` → exactly the first two keys. (This pins the fake so Task 3's tests mean something.)
- [ ] **Step 3: Run red:** `env -u … go test ./internal/quests/ -run 'TestFakeProgressBetween|TestIntegrationRoadmap' -count=1` → compile errors (`ActiveRoadmapDoc`, `ProgressBetween`, `DayProgress`, `roadmapJSON` undefined).
- [ ] **Step 4: Interfaces** in `repo.go`. Add as the **last** method of `QuestRepo` (after `MarkComplete`):

```go
	// ActiveRoadmapDoc is ActiveRoadmap plus roadmap_json — the outline
	// GET /api/v1/roadmap renders. Same row, same ErrNoActiveRoadmap.
	ActiveRoadmapDoc(ctx context.Context, userID string) (RoadmapDoc, error)
```

  and as the last method of `ProgressRepo` (after `TargetMet`):

```go
	// ProgressBetween returns the user's daily_progress rows with
	// fromDate <= date <= toDate (both YYYY-MM-DD, inclusive), keyed by
	// YYYY-MM-DD. Days without a row are simply absent.
	ProgressBetween(ctx context.Context, userID, fromDate, toDate string) (map[string]DayProgress, error)
```

  Do not touch any other line of `repo.go`.
- [ ] **Step 5: Create `roadmap.go`** (storage half; the service half is Task 3):

```go
package quests

// GET /api/v1/roadmap — the 28-day outline joined with per-day progress.
// Everything for that read lives in this file (design: harness/designs/
// roadmap-tree.md). It shares Service, QuestRepo and ProgressRepo with the
// daily loop and adds no write.

// RoadmapDoc is the §3.2 roadmaps row with its roadmap_json: the
// airouter.Roadmap onboarding validated and stored.
type RoadmapDoc struct {
	ID        string
	CreatedAt time.Time
	JSON      json.RawMessage
}

// DayProgress is one daily_progress row as GET /roadmap reports it.
type DayProgress struct {
	MinutesSpent int
	IsTargetMet  bool
}

const (
	activeRoadmapDocSQL = `
SELECT id, created_at, roadmap_json
FROM roadmaps
WHERE user_id = $1 AND is_active = TRUE
ORDER BY created_at DESC
LIMIT 1`

	// date::text is YYYY-MM-DD under the default ISO DateStyle — the same
	// string LocalDate/DayDate produce, so the map key needs no formatting.
	progressBetweenSQL = `
SELECT date::text, COALESCE(minutes_spent, 0), COALESCE(is_target_met, FALSE)
FROM daily_progress
WHERE user_id = $1 AND date BETWEEN $2::date AND $3::date`
)
```

  followed by `func (r *PgRepo) ActiveRoadmapDoc(...)` (scan `id, created_at, roadmap_json`; `pgx.ErrNoRows` → `ErrNoActiveRoadmap`; other errors wrapped `"quests: reading active roadmap document: %w"`) and `func (r *PgRepo) ProgressBetween(...)` (`Query`, scan into `map[string]DayProgress`, return `rows.Err()`; an empty result is an empty non-nil map). The existing `var _ QuestRepo = (*PgRepo)(nil)` / `_ ProgressRepo` assertions in `repo.go` make the compiler check the two new methods; add nothing there.
- [ ] **Step 6: Fakes** — append to `fakes_test.go`: the field `roadmapJSON json.RawMessage // nil → ActiveRoadmapDoc mirrors ActiveRoadmap's roadmap but with no JSON` as the last field of `fakeQuestRepo`; at the end of the file:

```go
func (f *fakeQuestRepo) ActiveRoadmapDoc(context.Context, string) (RoadmapDoc, error) {
	if f.roadmap == nil {
		return RoadmapDoc{}, ErrNoActiveRoadmap
	}
	return RoadmapDoc{ID: f.roadmap.ID, CreatedAt: f.roadmap.CreatedAt, JSON: f.roadmapJSON}, nil
}

// ProgressBetween mirrors progressBetweenSQL over the rows map: ISO dates
// compare lexically, so the string range is the date range.
func (f *fakeProgressRepo) ProgressBetween(_ context.Context, userID, fromDate, toDate string) (map[string]DayProgress, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := map[string]DayProgress{}
	for key, row := range f.rows {
		uid, date, ok := strings.Cut(key, "|")
		if !ok || uid != userID || date < fromDate || date > toDate {
			continue
		}
		out[date] = DayProgress{MinutesSpent: row.minutes, IsTargetMet: row.targetMet}
	}
	return out, nil
}
```

  (add `strings` to the file's imports).
- [ ] **Step 7: Run green (unit):** `env -u … go test ./internal/quests/ -run TestFakeProgressBetween -count=1` → `ok`; the whole package still compiles (`Service.Roadmap` is not referenced until Task 3 — if you wrote the service half of the integration test already, leave that test red until Task 3 and run only the repo assertions now, or temporarily guard with `t.Skip` and remove the guard in Task 3; do not commit a skip).
- [ ] **Step 8: Run the integration test against an isolated stack** (AGENTS.md worktree rule), from `backend/`:

```bash
COMPOSE_PROJECT_NAME=roadmaptree POSTGRES_PORT=55504 REDIS_PORT=56504 docker compose up -d --wait
TEST_DATABASE_URL='postgres://english:english@localhost:55504/english?sslmode=disable' TEST_REDIS_URL='redis://localhost:56504/0' go test ./internal/quests/ -run 'TestIntegration' -count=1 -v -p 1
COMPOSE_PROJECT_NAME=roadmaptree docker compose down
```

  Expected after Task 3: `--- PASS` for both `TestIntegration*` in `quests`, no `--- SKIP`. (Check `backend/docker-compose.yml` / `.env.example` for the credentials the URL needs; the auth plan uses the same shape with ports 55501/56501.)
- [ ] **Step 9: Commit:** `git add backend/internal/quests/repo.go backend/internal/quests/roadmap.go backend/internal/quests/roadmap_test.go backend/internal/quests/fakes_test.go backend/internal/quests/integration_test.go && git commit -m "quests: ActiveRoadmapDoc and ProgressBetween reads for the roadmap outline"`.

### Task 3: `Service.Roadmap` — the outline joined with progress

**Files:**
- Modify: `backend/internal/quests/roadmap.go`, `backend/internal/quests/roadmap_test.go`

- [ ] **Step 1: Write the failing tests** in `roadmap_test.go` (all through `newRoadmapHarness`; `h.quests.roadmap.CreatedAt` is `now − 24h` from `newHarness` — override it per test as below):
  - `TestRoadmapReturnsFourModulesOfSevenNamedDaysWithDates`: `now := 2026-09-10T10:00Z`, `h.quests.roadmap.CreatedAt = 2026-09-01T22:00Z` (late in the day). `out, err := h.svc.Roadmap(ctx, "u1")` → `out.RoadmapID == "rm-1"`, `out.Title == "Integration"`, `out.CEFRLevel == "B1"`, `out.CreatedAt.Equal(created)`, `out.DayNumber == 10`; `len(out.Modules) == 4`; `out.Modules[1].Week == 2`, `.Title == "Week 2"`, `.Focus == "integration"`; `out.Modules[0].Days[0]` has `DayNumber 1, Date "2026-09-01", Title "Day 1"`; `out.Modules[0].Days[1].Date == "2026-09-02"`; `out.Modules[3].Days[6]` has `DayNumber 28, Date "2026-09-28", Title "Day 28"`; every day has exactly 3 tasks and the first is `{TaskType: "vocabulary", Title: "Day task: vocabulary", DurationMinutes: 10}` (the fixture's order is `airouter.TaskTypes`; the service keeps stored order).
  - `TestRoadmapJoinsDailyProgressByLocalDate`: same clock; `h.progress.rows` for `u1|2026-09-01` (32, met), `u1|2026-09-03` (12, not met), `u1|2026-09-10` (10, not met — today), `u1|2026-08-30` (7, met — before the roadmap, must be ignored), `u2|2026-09-02` (30, met — another user). Assert days 1 → `32/true`, 2 → `0/false`, 3 → `12/false`, 10 → `10/false`, 11 → `0/false`, and that `out.Modules[0].Days[0].IsTargetMet` is the only `true` in the whole outline.
  - `TestRoadmapDatesFollowTheUsersTimezone`: `h.quests.timezone = "Asia/Ho_Chi_Minh"`, `CreatedAt = 2026-09-01T18:30Z` (= 09-02 01:30 local), `now = 2026-09-01T19:00Z` → `DayNumber == 1`, `Days[0].Date == "2026-09-02"`; and `h.progress.rows["u1|2026-09-02"] = (5, false)` shows up on day 1 — the row is keyed by the local date `RecordProgress` would have used.
  - `TestRoadmapDatesAgreeWithDayNumberAcrossDST`: `h.quests.timezone = "Europe/London"`, `CreatedAt = 2026-03-25 09:00 London`, `now = 2026-04-21 09:00 London` → `DayNumber == 28`, `Modules[3].Days[6].Date == "2026-04-21"`, `Modules[0].Days[4].Date == "2026-03-29"` (the spring-forward day is day 5, as `day_test.go` says). Skip with `t.Fatalf` if `Location("Europe/London") == time.UTC` (tzdata missing), as `day_test.go` does.
  - `TestRoadmapDefaultsAMissingDuration`: build the fixture, set `Modules[0].Days[0].Tasks[0].DurationMinutes = 0`, marshal → that task reports `DefaultTaskMinutes` (same rule as `toTask`).
  - `TestRoadmapWithoutAnActiveRoadmap`: `h.quests.roadmap = nil` → `errors.Is(err, ErrNoActiveRoadmap)`.
  - `TestRoadmapWithUnreadableJSONFails`: `h.quests.roadmapJSON = json.RawMessage("{not json")` → `err != nil`, no panic, and the error string contains `roadmap_json`.
  - `TestRoadmapWithTheWrongShapeFails`: `roadmapJSON = {"title":"x","cefr_level":"B1","modules":[]}` → `err != nil` mentioning `modules` (a stored document that is not 4×7 would render a broken tree; refuse it rather than pad it).
- [ ] **Step 2: Run red:** `env -u … go test ./internal/quests/ -run TestRoadmap -count=1` → compile errors (`Service.Roadmap`, `RoadmapOutline` undefined).
- [ ] **Step 3: Append to `roadmap.go`** the wire DTOs and the service method:

```go
// RoadmapTask is one entry of a day's tasks in the GET /api/v1/roadmap body
// (backend spec §6.2): the outline only — no content_json.
type RoadmapTask struct {
	TaskType        string `json:"task_type"`
	Title           string `json:"title"`
	DurationMinutes int    `json:"duration_minutes"`
}

// RoadmapDay is one of the 28 days: its plan and what daily_progress says
// happened on its date.
type RoadmapDay struct {
	DayNumber    int           `json:"day_number"`
	Date         string        `json:"date"` // YYYY-MM-DD in the user's timezone (DayDate)
	Title        string        `json:"title"`
	Tasks        []RoadmapTask `json:"tasks"`
	MinutesSpent int           `json:"minutes_spent"`
	IsTargetMet  bool          `json:"is_target_met"`
}

// RoadmapModule is one of the four weekly modules (§6.1).
type RoadmapModule struct {
	Week  int          `json:"week"`
	Title string       `json:"title"`
	Focus string       `json:"focus"`
	Days  []RoadmapDay `json:"days"`
}

// RoadmapOutline is the GET /api/v1/roadmap 200 body — backend spec §6.2,
// field for field.
type RoadmapOutline struct {
	RoadmapID string          `json:"roadmap_id"`
	Title     string          `json:"title"`
	CEFRLevel string          `json:"cefr_level"`
	CreatedAt time.Time       `json:"created_at"`
	DayNumber int             `json:"day_number"`
	Modules   []RoadmapModule `json:"modules"`
}
```

  and `func (s *Service) Roadmap(ctx context.Context, userID string) (RoadmapOutline, error)`: `Profile` → `ActiveRoadmapDoc` → `json.Unmarshal(doc.JSON, &airouter.Roadmap{})` (error → `fmt.Errorf("quests: roadmap_json for roadmap %s: %w", doc.ID, err)`); `len(Modules) != airouter.Modules` or any `len(Days) != airouter.DaysPerModule` → `fmt.Errorf("quests: roadmap %s has %d modules …", …)` naming `modules`/`days`; `loc := Location(profile.Timezone)`, `now := s.now()`, `day := DayNumber(doc.CreatedAt, now, loc)`; `rows, err := s.progress.ProgressBetween(ctx, userID, DayDate(doc.CreatedAt, 1, loc), DayDate(doc.CreatedAt, RoadmapDays, loc))`; then build the modules: `n := mi*airouter.DaysPerModule + di + 1`, `date := DayDate(doc.CreatedAt, n, loc)`, `p := rows[date]` (zero value when absent), tasks mapped with `DurationMinutes <= 0 → DefaultTaskMinutes`, `Tasks` and `Days`/`Modules` made with `make(…, 0, n)` so they never serialise as `null`. The doc comment on `Roadmap` states the three rules: dates are `DayDate` in the user's zone; a missing row is `0/false`; the flag is what the pet was told (set after the hook), so a day past 30 minutes whose hook failed reads `30/false` until the next progress call — the tree does not know better than the pet.
- [ ] **Step 4: Run green:** `env -u … go test ./internal/quests/ -count=1` → `ok`; re-run Task 2 Step 8 against the isolated stack → both `TestIntegration*` pass.
- [ ] **Step 5: Commit:** `git commit -am "quests: Service.Roadmap renders the 28-day outline joined with daily_progress"`.

### Task 4: Handler, route, and the two specs

**Files:**
- Modify: `backend/internal/quests/roadmap.go`, `backend/internal/quests/roadmap_test.go`, `backend/internal/quests/handler_test.go` (one line), `backend/cmd/api/main.go` (one line), the two spec files

- [ ] **Step 1: Write the failing handler tests** in `roadmap_test.go`, using `newQuestRouter` from `handler_test.go` after adding — **inside `newQuestRouter`, directly under the `/quests/progress` line, nothing else in that file** — `g.GET("/roadmap", RoadmapHandler(svc))`:
  - `TestRoadmapHandlerReturnsTheSpec62Body`: `GET /api/v1/roadmap` → `200`; decode into `map[string]any` and assert the top-level keys are exactly `roadmap_id, title, cefr_level, created_at, day_number, modules`; `modules` has 4 entries, each with keys `week, title, focus, days`, 7 days each with keys `day_number, date, title, tasks, minutes_spent, is_target_met`, 3 tasks each with keys `task_type, title, duration_minutes`; and `!strings.Contains(w.Body.String(), `"content`)` (neither `content` nor `content_json` leaks — the fixture's tasks carry `Content: {}`).
  - `TestRoadmapHandlerReturns404WithoutARoadmap`: `h.quests.roadmap = nil` → `404 {"error":"no_active_roadmap"}` (same body as `TestDailyHandlerReturns404WithoutARoadmap`).
  - `TestRoadmapHandlerReturns500OnAnUnreadableDocument`: `roadmapJSON = "{"` → `500 {"error":"internal_error"}`.
- [ ] **Step 2: Run red:** `env -u … go test ./internal/quests/ -run TestRoadmapHandler -count=1` → compile error (`RoadmapHandler` undefined).
- [ ] **Step 3: Append `RoadmapHandler` to `roadmap.go`** — the same shape as `DailyHandler` (`auth.UserID` guard → `401 unauthorized`; `ErrNoActiveRoadmap` → `404 no_active_roadmap`; other errors → `500 internal_error`, and `log.Printf("quests: roadmap outline for user %s: %v", userID, err)` before the 500 so a corrupt document is diagnosable). Doc comment: "serves GET /api/v1/roadmap (backend spec §6.2). Mount behind auth.Require()."
- [ ] **Step 4: Register the route** in `backend/cmd/api/main.go`, directly after `guarded.POST("/quests/progress", quests.ProgressHandler(questSvc))`:

```go
	guarded.GET("/roadmap", quests.RoadmapHandler(questSvc))
```

- [ ] **Step 5: Run green:** `env -u … go test ./internal/quests/ -count=1` → `ok`; `cd backend && make check` → fmt-check silent, vet silent, every package `ok` under `-race`; `go build ./...` clean.
- [ ] **Step 6: Backend spec §6.2.** In `project-base/Adaptive English Learning Platform - Backend Technical Specification.md`, after the `POST /api/v1/quests/progress` response block (the ```` ```json ```` block ending `"streak_count": 5}` and its closing fence, before the blank line and `## **6.3 Pet State & Revival Endpoints**`), insert a block in the exact style of the `GET /api/v1/quests/daily` entry — note the **two trailing spaces** on the `* GET`, `* Description`, `* Request Headers` lines and the two-space indent of the sub-bullets (check with `sed -n '286,289p' … | cat -e`):

```
* GET /api/v1/roadmap  
  * Description: Fetching the active 28-day roadmap outline (4 modules x 7 days, 3 task titles per day, no exercise content) joined with per-day progress from daily\_progress, for the roadmap tree screen. date is the user's local date of the roadmap's created\_at plus day\_number \- 1.  
  * Request Headers: Authorization: Bearer \<JWT\>  
  * Response (200 OK):

```json
{"roadmap_id": "b11c22d3-44e5-66f7-88a9-00bbccddeeff", "title": "Business English for meetings", "cefr_level": "B1", "created_at": "2026-09-22T13:05:00Z", "day_number": 3, "modules": [{"week": 1, "title": "Everyday small talk", "focus": "Greetings and introductions", "days": [{"day_number": 1, "date": "2026-09-22", "title": "Meeting a new colleague", "tasks": [{"task_type": "vocabulary", "title": "10 Key Business Email Phrasings", "duration_minutes": 10}, {"task_type": "reading", "title": "Two sample emails", "duration_minutes": 10}, {"task_type": "practice", "title": "Write a reply", "duration_minutes": 10}], "minutes_spent": 32, "is_target_met": true}]}]}
```

  * Response (404 Not Found): {"error": "no\_active\_roadmap"}  
```

  (the sample shows one day of one module; the §6.2 `quests/daily` sample is abbreviated the same way). Keep the file's line-ending convention (`file` on it before and after must agree).
- [ ] **Step 7: 1st-thinking §7.** In `project-base/1st-thinking-architecture-doc.md`, after the line `\* \*\*POST /api/v1/quests/progress\*\*: Records completed task minutes, updates Redis counter and daily progress.` and its following blank line, insert (with a blank line after it, matching the list's spacing):

```
\* \*\*GET /api/v1/roadmap\*\*: Fetches the active 28-day roadmap outline (module and day titles, task titles and durations) joined with per-day completion, for the roadmap tree.
```

- [ ] **Step 8: Verify the specs** from the worktree root: `grep -c 'api/v1/roadmap' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md" project-base/1st-thinking-architecture-doc.md` → `1` and `1` (both were `0`); `grep -n 'api/v1/roadmap' project-base/1st-thinking-architecture-doc.md` shows the line between `quests/progress` and `pet/status`.
- [ ] **Step 9: Commit:** `git add backend/internal/quests/roadmap.go backend/internal/quests/roadmap_test.go backend/internal/quests/handler_test.go backend/cmd/api/main.go project-base && git commit -m "quests: GET /api/v1/roadmap — the outline read, routed and specified"`.

### Task 5: Frontend store and the node derivation

**Files:**
- Create: `frontend/stores/roadmap.ts`, `frontend/tests/unit/roadmapStore.test.ts`
- Modify: `frontend/utils/roadmap.ts` (rewrite), `frontend/tests/unit/roadmap.test.ts` (rewrite)

- [ ] **Step 1: Write the failing tests.**
  - `tests/unit/roadmapStore.test.ts`, in the shape of `questStore.test.ts` (`vi.mock('~/composables/useApi')`, `await import('~/stores/roadmap')`): `load` calls `api.get` with `/api/v1/roadmap` and stores the outline; a rejected `new ApiError(404, 'no_active_roadmap')` (check `ApiError`'s constructor in `utils/apiClient.ts`) sets `noRoadmap = true` and `outline = null`, `error` stays null; any other `ApiError` sets `error` to its code; a non-`ApiError` rejection sets `'network_error'`; `completedDays` counts days with `is_target_met` across all modules.
  - `tests/unit/roadmap.test.ts` (replace the three old cases): build a fixture `outline(dayNumber, overrides)` with 4 modules × 7 days × 3 tasks (tasks deliberately in the order practice, vocabulary, reading; titles `Day n`, module titles `Week m`, dates `2026-09-${n}` for n ≤ 28 via a small helper) and test:
    - `dayState`: past + met → `completed`; past + minutes 12 → `partial`; past + 0 → `missed`; `day_number === today` → `today` even when met and even at 0 minutes; future → `locked` even if a row says met (the server never writes one; pin the precedence anyway); past + minutes 30 + not met → `partial`.
    - `roadmapNodes(outline)`: 28 nodes in order; `nodes[7].week === 2`; each node's `tasks` are sorted vocabulary → reading → practice; `nodes[0].title === 'Day 1'`; `nodes[0].date === '2026-09-01'`.
    - `completedDays`: 2 met days → `2`.
    - `statusText`: `completed` → `'Đã hoàn thành'`; `partial` 12 → `'12/30 phút'`; `missed` → `'Bỏ lỡ'`; `locked` → `'Chưa mở khóa'`; `today` 10 min → `'Đang học · 10/30 phút'`; `today` met → `'Đã đủ 30 phút'`.
    - `formatDayDate('2026-09-22') === 'T3 22/9'`, `formatDayDate('2026-09-27') === 'CN 27/9'`, `formatDayDate('2026-10-01') === 'T5 1/10'` (parse with `Date.UTC`, never `new Date(iso)` in local time — the test must pass in any `TZ`).
    - `TASK_LABELS` has exactly `vocabulary: 'Từ vựng', reading: 'Đọc hiểu', practice: 'Thực hành'`.
- [ ] **Step 2: Run red:** from `frontend/`: `npm run test:unit -- roadmap` → the store test fails to resolve `~/stores/roadmap`; the util test fails on the missing exports.
- [ ] **Step 3: Create `stores/roadmap.ts`** — types mirroring the §6.2 body field for field (`RoadmapTask`, `RoadmapDay`, `RoadmapModule`, `RoadmapOutline`, with a doc comment "Backend spec §6.2 GET /roadmap. Outline only — no exercise content."), and `useRoadmapStore` with state `{ outline: null as RoadmapOutline | null, noRoadmap: false, loading: false, error: null as string | null }`, getter `completedDays`, and `load()` copied from `useQuestStore.load` with the path and the field names changed (same `no_active_roadmap` branch, same `error` mapping).
- [ ] **Step 4: Rewrite `utils/roadmap.ts`.** Keep `ROADMAP_DAYS = 28`; add `TARGET_MINUTES = 30`; `NodeState = 'completed' | 'partial' | 'missed' | 'today' | 'locked'`; `RoadmapNode { day, week, date, title, state, minutesSpent, isTargetMet, tasks: RoadmapTask[] }`; the functions listed in Step 1, importing `TASK_ORDER` from `~/stores/quest` (a const; do not import the store function) and the types from `~/stores/roadmap`. The new header comment: "design harness/designs/roadmap-tree.md §4.1. States come from the server's per-day `daily_progress` join and its `day_number`; the client clock is never consulted." The old "open question" comment is removed.

```ts
export function dayState(day: RoadmapDay, todayNumber: number): NodeState {
  if (day.day_number === todayNumber) return 'today'
  if (day.day_number > todayNumber) return 'locked'
  if (day.is_target_met) return 'completed'
  return day.minutes_spent > 0 ? 'partial' : 'missed'
}
```

- [ ] **Step 5: Run green:** `npm run test:unit -- roadmap` → both files pass; `npm run lint` → clean. `npm run typecheck` is expected to fail here on `pages/roadmap.vue` (it still calls `roadmapNodes(number)`); Task 6 rewrites the page and runs `typecheck` there.
- [ ] **Step 6: Commit:** `git add frontend/stores/roadmap.ts frontend/utils/roadmap.ts frontend/tests/unit/roadmap.test.ts frontend/tests/unit/roadmapStore.test.ts && git commit -m "frontend: roadmap store and truthful per-day node states"`.

### Task 6: The page — spine, module headers, expandable days

**Files:**
- Create: `frontend/components/roadmap/RoadmapMarker.vue`, `frontend/components/roadmap/RoadmapModuleHeader.vue`, `frontend/tests/unit/roadmapPage.test.ts`
- Modify: `frontend/components/roadmap/RoadmapNode.vue`, `frontend/pages/roadmap.vue`, `frontend/service-worker/sw.ts`

- [ ] **Step 1: Write the failing page test** `tests/unit/roadmapPage.test.ts` in the shape of `revivePage.test.ts` / `onboardingPage.test.ts` (`vi.mock('~/composables/useApi')`, `await import('~/pages/roadmap.vue')`, `mount` with `components: { AppCard, AppButton, StateBlock, RoadmapNode, RoadmapMarker, RoadmapModuleHeader }`, `stubs: { AppHeader: true, NuxtLink: { template: '<a><slot /></a>' } }`, `mocks: { navigateTo }`; `Element.prototype.scrollIntoView = vi.fn()` in `beforeEach` — happy-dom does not implement it). Route `api.get` by path: `/pet/status` → a resolved status, `/api/v1/roadmap` → the fixture (reuse the `outline()` builder from `roadmap.test.ts` by exporting it from a small `tests/unit/fixtures/roadmap.ts`, or duplicate it — pick one and say so). Cases:
  - renders the roadmap title, `Trình độ B1` and `Đã hoàn thành 2/28 ngày` in the header block;
  - renders four `RoadmapModuleHeader`s with `Tuần 1`…`Tuần 4`, the module titles and focus lines, and the per-module `2/7` / `0/7` counts;
  - renders 28 rows; row 1's status is `Đã hoàn thành`, row 2's is `Bỏ lỡ`, row 3's `12/30 phút`, row 10's `Chưa mở khóa`; row 1's title text is `Day 1`;
  - today's row (`day_number` 9 in the fixture) has `aria-current="step"`, `aria-expanded="true"`, its panel lists the three task titles with `10 phút` and a `Học ngay` link to `/`; every other row has `aria-expanded="false"` and no visible panel;
  - clicking row 10 toggles its `aria-expanded` to `true` and shows its tasks; clicking again hides them (locked days expand too);
  - `scrollIntoView` was called once on the element with id `day-9`;
  - on `ApiError 404 no_active_roadmap` the page shows `Bạn chưa có lộ trình học.` and a `Tạo lộ trình 28 ngày` action that calls `navigateTo('/onboarding')`; on a network failure it shows `Không tải được lộ trình.` with `Thử lại`, and the retry calls `api.get` again.
- [ ] **Step 2: Run red:** `npm run test:unit -- roadmapPage` → fails (components missing / old page shape).
- [ ] **Step 3: Build the components** per the design:
  - `RoadmapMarker.vue`: prop `state: NodeState`; a 24 px round `span` (`aria-hidden="true"`) with the glyph/ring pair from design §4.1 (`today` 🌱 on `border-growth` ring; `completed` ★ `text-streak bg-streak/10 border-streak/40`; `partial` ◐ `text-streak border-streak/40`; `missed` ○ `text-mute border-mute/40`; `locked` 🔒 `text-mute border-mute/30`); `data-state="<state>"` for tests.
  - `RoadmapModuleHeader.vue`: props `module: RoadmapModule`, `met: number`; renders the `ink` dot on the rail, `Tuần {week} · {title}` (`text-xs font-semibold uppercase tracking-wider text-mute` for the "Tuần n" span, `text-base font-semibold` for the title), `{met}/7` right-aligned in `text-[13px] tabular-nums text-mute`, and `focus` as a caption below.
  - `RoadmapNode.vue` (rewrite): props `node: RoadmapNode`, `expanded: boolean`; emits `update:expanded`. A `<li :id="`day-${node.day}`">` containing a `<button type="button" :aria-expanded="expanded" :aria-current="node.state === 'today' ? 'step' : undefined">` whose accessible text is the three lines in order — `Ngày {n} · {formatDayDate(date)}` (caption, `text-mute tabular-nums`), the title (`text-base font-semibold`, `text-mute` for `missed`/`locked`), `statusText(node)` (caption; `text-growth` for today, `text-streak` for completed/partial, `text-mute` otherwise) — plus the `HÔM NAY` chip (`bg-growth text-white text-xs rounded-full px-2`) on today's row; `RoadmapMarker` on the rail; and a `v-show="expanded"` panel: `<ul :aria-label="`Nhiệm vụ ngày ${n}`">` inside an `AppCard`, one `li` per task — `TASK_LABELS[task_type] ?? task_type` · title · `{duration_minutes} phút` — and, for today only, `<NuxtLink to="/">` styled as the primary button with the text `Học ngay →`. Min row height 56 px; whole row is the target; the shell's focus ring on the button.
- [ ] **Step 4: Rewrite `pages/roadmap.vue`.** `const roadmap = useRoadmapStore()`, `const pet = usePetStore()`; `onMounted`: `if (!pet.status) void pet.load()`; `if (!roadmap.outline && !roadmap.noRoadmap) await roadmap.load()`; `expanded[roadmap.outline.day_number] = true`; `await nextTick()`; `document.getElementById(`day-${roadmap.outline?.day_number ?? 0}`)?.scrollIntoView({ block: 'center' })`. `const nodes = computed(() => roadmap.outline ? roadmapNodes(roadmap.outline) : [])`; `const expanded = reactive<Record<number, boolean>>({})`. Template: `AppHeader` → eyebrow `Lộ trình học 28 ngày` (`text-xs font-semibold uppercase tracking-wider text-mute`) → `<h1 class="font-display text-[28px] leading-8 line-clamp-2">{{ roadmap.outline.title }}</h1>` → `<p class="text-mute">Trình độ {{ cefr_level }} · Đã hoàn thành {{ roadmap.completedDays }}/28 ngày</p>` → the `AppCard` with the three `StateBlock` branches unchanged in copy (bind them to `roadmap.loading/noRoadmap/error` and `roadmap.outline`) → otherwise an `<ol class="relative border-l-2 border-mute/30 ml-3 pl-7 space-y-4">` iterating `roadmap.outline.modules`: a `RoadmapModuleHeader` (`met = module.days.filter(d => d.is_target_met).length`) then a `RoadmapNode` per node of that week (`nodes.filter(n => n.week === module.week)`) with `v-model:expanded="expanded[node.day]"`. The h1/eyebrow/line only render when `roadmap.outline` is present; the empty state keeps the old `Lộ trình học 28 ngày` h1 so the page never renders headless. `quest` store is no longer imported here.
- [ ] **Step 5: Service worker:** in `sw.ts` change the NetworkFirst matcher to `/\/api\/v1\/(quests\/daily|pet\/status|roadmap)$/` and extend the comment: "`/roadmap` is the same class of state (§3 'User Progress'): synchronised when online, last known when not; `utils/session.ts` drops the whole cache on sign-out, so it needs no separate clearing."
- [ ] **Step 6: Run green:** from `frontend/`: `npm run lint && npm run typecheck && npm run test:unit` → clean, `roadmapPage.test.ts` all cases pass; `npm run build` → succeeds (the `injectManifest` worker compiles). Then `npm run test:e2e` (`npx playwright install chromium` once if needed) → `login.spec.ts` still green (it does not visit `/roadmap`; no stub to add).
- [ ] **Step 7: Look at it.** Start the dev server against a stubbed or real API (or mount the page in the browser preview with the fixture) and check the design's §2 picture at 375 px: rail continuous through module milestones, today's ring `growth` and its panel open, stars only on met days, no horizontal scroll, dark mode readable. Fix spacing before committing; note anything you changed from the design in the plan summary.
- [ ] **Step 8: Commit:** `git add frontend && git commit -m "frontend: /roadmap renders the real plan on a calendar spine with truthful day states"`.

### Task 7: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (`quests` and `shell` paragraphs only)

- [ ] **Step 1: `quests` paragraph.** After the sentence ending "404 `no_active_roadmap` when there is none." insert: "`GET /api/v1/roadmap` (2026-09-25; `roadmap.go`) returns the outline `{roadmap_id, title, cefr_level, created_at, day_number, modules[4]{week, title, focus, days[7]{day_number, date, title, tasks[3]{task_type, title, duration_minutes}, minutes_spent, is_target_met}}}` — `roadmap_json` unmarshalled as `airouter.Roadmap` (a document that is not 4×7 is a 500, never padded), no `content_json`; `date` = `DayDate(created_at, n, tz)` in `day.go`, the inverse of `DayNumber` and the rule `google.DayDue` uses (round-trip pinned on the DST fixtures); `minutes_spent`/`is_target_met` from one `ProgressBetween` range read of `daily_progress` keyed by that date, absent row → `0/false` — the flag is what the pet was told, so it never runs ahead of `pet`; same `404 no_active_roadmap`." In the Tests sentence, change "`TestIntegrationDailyAndProgressAgainstRealServices` is gated" to "`TestIntegrationDailyAndProgressAgainstRealServices` and `TestIntegrationRoadmapOutlineJoinsDailyProgress` are gated".
- [ ] **Step 2: `shell` paragraph.** Add `stores/roadmap.ts` to the stores list: "`stores/roadmap.ts` (`GET /roadmap` outline, `noRoadmap`, `completedDays`)". Replace "`/roadmap` (7.4 — derived from `day_number`; no per-day endpoint)" with "`/roadmap` (7.4 as `harness/designs/roadmap-tree.md`: a calendar spine of four module headers and 28 named days from `GET /roadmap`; `utils/roadmap.ts` `dayState` = today → completed (`is_target_met`) → partial (minutes, "12/30 phút") → missed → locked, by the server's `day_number`, never the client clock; rows expand to their task titles; today scrolls into view)". In the service-worker clause change "for `/quests/daily` + `/pet/status`" to "for `/quests/daily` + `/pet/status` + `/roadmap`".
- [ ] **Step 3:** `python3 tools/harness/cli.py validate` → exit 0. Commit: `git commit -am "harness: CODEMAP — GET /roadmap and the roadmap tree"`.

---

## Verification

```bash
cd backend
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./internal/quests/ -count=1 -race -v 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
# expect: one `ok` line, no FAIL
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./internal/quests/ -count=1 -v -run 'TestRoadmap|TestDayDate|TestFakeProgressBetween' 2>&1 | grep -c '^--- PASS'
# expect: ≥ 14 (2 DayDate + 1 fake + 8 service + 3 handler)
grep -c '^func Test' internal/quests/roadmap_test.go
# expect: 12
grep -c '^func TestIntegration' internal/quests/integration_test.go
# expect: 2 (was 1)
grep -rho '^func TestIntegration[A-Za-z0-9_]*' --include='*_test.go' . | wc -l | tr -d ' '
# expect: 13 on this branch (12 on origin/main today + 1; CI derives its `want` from the tree)
grep -n 'ContentJSON\|json:"content' internal/quests/roadmap.go
# expect: no output (the outline never carries exercise content)
grep -n 'roadmap' cmd/api/main.go
# expect: exactly one route line: guarded.GET("/roadmap", quests.RoadmapHandler(questSvc))
git diff origin/main -- internal/quests/service.go internal/quests/service_test.go
# expect: no output (untouched — the pet bug plan owns them today)
git diff origin/main --stat -- internal/quests/handler_test.go
# expect: 1 file changed, 1 insertion(+)
make check
# expect: fmt-check silent, vet silent, ok for every package under -race
cd ..
grep -c 'api/v1/roadmap' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md" project-base/1st-thinking-architecture-doc.md
# expect: 1 and 1 (both 0 on origin/main)
cd frontend
npm run lint && npm run typecheck && npm run test:unit
# expect: all clean; roadmap.test.ts, roadmapStore.test.ts, roadmapPage.test.ts pass
grep -c 'registerRoute' service-worker/sw.ts
# expect: 3 (unchanged: the import plus two routes) — and:
grep -n 'quests\\/daily|pet\\/status|roadmap' service-worker/sw.ts
# expect: 1 line (the NetworkFirst matcher)
grep -n 'open question\|before today' utils/roadmap.ts
# expect: no output
grep -n "from '~/stores/quest'" utils/roadmap.ts pages/roadmap.vue
# expect: only utils/roadmap.ts (the TASK_ORDER import); the page no longer uses the quest store
git diff origin/main -- stores/quest.ts stores/pet.ts
# expect: no output
npm run build && npm run test:e2e
# expect: build ok; login.spec.ts green
cd ..
grep -n 'GET /api/v1/roadmap\|roadmap-tree' harness/CODEMAP.md
# expect: the quests sentence and the shell sentence
git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP
# expect: no output (the branch touches no other harness/ file)
python3 tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0

# The outer loop: after `git push -u origin <branch>`,
gh run list --branch "$(git branch --show-current)" --limit 1
# expect: backend-unit, backend-integration (13 TestIntegration* pass, 0 skip), harness-tooling, frontend all green
```

Integration proof (record it in the summary): Task 2 Step 8's isolated stack run with both `quests` `TestIntegration*` passing and no `--- SKIP`; `docker compose -p roadmaptree down` afterwards.

Mutation checks (record the results): (1) change `if (day.day_number === todayNumber) return 'today'` to run after the `is_target_met` check in `dayState` → the "today even when met" case must go red; revert. (2) In `Service.Roadmap`, replace `DayDate(doc.CreatedAt, n, loc)` with a `time.Duration`-based `createdAt.Add(24h × (n−1))` date → `TestRoadmapDatesAgreeWithDayNumberAcrossDST` must go red; revert.

## Notes and open questions

- **Today's flag can lag the live counter by one call.** `is_target_met` is set by `MarkTargetMet` after the pet hook on the crossing call; if the hook failed, the row says `30/false` until the next progress call retries it. The tree shows exactly that ("30/30 phút", partial) — it reports what the pet was told. Do not read the Redis counter here to "fix" it; `/quests/daily` is the live view.
- **Task order on the wire is stored order.** `airouter.ParseRoadmap` guarantees the three types, not their order; the client sorts by `TASK_ORDER` in `utils/roadmap.ts`, as the dashboard does with `sortedTasks`.
- **`airouter` is now a production import of `quests`.** It has no routes and no tables (CODEMAP), so the boundary rule is respected; the alternative — a private copy of the shape in `quests` — would drift from `RoadmapSchema` silently.
- **A stored document that is not 4×7 is a 500, logged with the roadmap id.** It cannot happen through onboarding (`ParseRoadmap` validated it), so padding or truncating would hide a real corruption.
- **Merge with the pet bug plan** (`2026-09-25-a-pet-state-failure…`): both branches land on `harness/daily-2026-09-25`. This plan's only edits to files that plan touches are the one-line route insertion inside `newQuestRouter` (`handler_test.go`; that plan appends a test at the end of the file). If the integration merge still conflicts there, keep both hunks — they are independent.
- **Not in scope:** roadmap history / completed roadmaps (the 2026-09-24 *day-28 checkpoint* idea will extend this read), persisting the expanded state, tapping a past day to replay its exercises (no route for a past day's exercises exists — `CheckExercise` rejects any exercise off today's `day_number`), and an offline caption on this page.
