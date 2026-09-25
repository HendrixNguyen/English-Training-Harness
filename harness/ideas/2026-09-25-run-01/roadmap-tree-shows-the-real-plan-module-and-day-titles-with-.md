---
type: feature
status: planned
source: ideator
run: 2026-09-25-run-01
priority: medium
plan: harness/plans/2026-09-25-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md
---
# Roadmap tree shows the real plan: module and day titles with true per-day completion

## Why
Frontend spec §7.4 promises a 28-day curriculum tree where each node says what the day is and whether it was completed ("⭐ Ngày 1: Đã hoàn thành", "🌱 Ngày 3: HÔM NAY", "🔒 Ngày 4: Chưa mở khóa"). What ships is a tree of 28 numbered circles: `utils/roadmap.ts` derives every node from a single integer (`day_number` of `GET /quests/daily`) and defines "completed" as "before today". The consequences a learner notices on day 4:

- **Days they skipped are shown as completed with a star.** The pet lost 30 health for day 2, yet the roadmap awards it. The two surfaces contradict each other, and the tree stops being a truthful record of the streak.
- **No day has a name.** The AI generated a title, focus and three task titles for every one of the 28 days and four weekly modules (`roadmaps.roadmap_json`, `RoadmapSchema`), and Google Tasks users already see "Day N: title · title · title" — but inside the app the plan is invisible. The learner cannot see that week 2 is "Business email writing" or what tomorrow holds, which is the whole point of an AI-personalised roadmap (1st-thinking §1) and the main reason to open `/roadmap` at all.

All the data exists: `roadmaps.roadmap_json` (modules → days → tasks) and `daily_progress` (one row per `user_id, date` with `minutes_spent`, `is_target_met`). This is a read path, no new writes, no AI call.

## Expected output
User-visible:
- `/roadmap` shows the roadmap title and CEFR level at the top, then four module headers (week, title, focus) each followed by its seven days.
- Every day node shows "Ngày N · <day title>" and, expanded on tap, its three task titles with durations.
- Node state is truthful: `completed` (that local date has `daily_progress.is_target_met = true`), `partial` (minutes recorded but target not met — shows "12/30 phút"), `missed` (a past day with no target met — the wireframe's star is replaced by a muted mark), `today`, `locked`. Today's node scrolls into view as it does now.
- A progress line "Đã hoàn thành 9/28 ngày" under the title.
- A learner with no active roadmap still sees the existing empty state and the "Tạo lộ trình 28 ngày" action.

Technical:
- Backend `quests` package: new read `GET /api/v1/roadmap` (behind `auth.Require()`) returning `{roadmap_id, title, cefr_level, created_at, day_number, modules[4]{week, title, focus, days[7]{day_number, date, title, tasks[3]{task_type, title, duration_minutes}, minutes_spent, is_target_met}}}`. `title/cefr_level/modules` come straight from `roadmap_json` (already validated by `airouter.ParseRoadmap`); `date` = local date of `roadmaps.created_at` + N−1 in `users.timezone`, the same rule `google` uses for task `due`; `minutes_spent/is_target_met` are a left join on `daily_progress` by that date. Exercise `content_json` is **not** included — this is an outline, not the exercise payload. 404 `no_active_roadmap` as `/quests/daily`. Add it to 1st-thinking §7 and backend spec §6.2 in the escaped style, as previous additive routes did.
- Frontend: `stores/roadmap.ts` (or extend `quest.ts`), `utils/roadmap.ts` `roadmapNodes` takes the outline and per-day progress instead of a bare integer; `RoadmapNode.vue` gains title, state variants and the expandable task list; `/roadmap` renders module headers. Service worker: `/api/v1/roadmap` joins the NetworkFirst `api-state` cache (same rules as `/quests/daily`).
- Tests: Go unit tests with fakes for outline + join (missed / partial / met / today / future; a roadmap created late in the day; DST-crossing dates reuse `day_test.go` fixtures); integration test gated on `TEST_DATABASE_URL`; Vitest for `roadmapNodes` state derivation and the page's module rendering.
- CODEMAP `quests` and `shell` paragraphs updated; the `utils/roadmap.ts` "open question" comment about per-day completion is resolved.

## Evidence
- Frontend spec §7.4 wireframe (named per-day nodes with completed / today / locked states); frontend spec §5 API→UI mapping (roadmap screen).
- 1st-thinking §1 ("AI-personalized roadmaps"), §3.2 (`roadmaps.roadmap_json`, `daily_progress` unique on `user_id, date` with `minutes_spent`, `is_target_met`), §7 (endpoint list to keep in sync).
- Code: `frontend/utils/roadmap.ts` (`state: day < today ? 'completed' : …` and the comment "There is no per-day completion endpoint, so 'completed' means 'before today' — open question"), `frontend/pages/roadmap.vue`, `frontend/components/roadmap/RoadmapNode.vue`; backend `airouter.RoadmapSchema` (`{title, cefr_level, modules[4]{week,title,focus,days[7]{title,tasks[3]{…}}}}`), `google` package's `Day N: title · title · title` and `due` date rule (CODEMAP), `quests` `day_number` calendar-day rule (`day.go`).
- Prior run: `harness/ideas/2026-09-24-run-01/day-28-checkpoint-cefr-re-assessment-and-the-next-roadmap.md` wants `/roadmap` to list completed roadmaps as history; this idea gives it the outline read it can extend.
- Progress visibility as a retention lever in learning apps: https://userpilot.com/blog/app-retention-strategies/

## Evaluation
_Evaluator, 2026-09-25 — daily decide (feature slot 4 of 5; two-cap rule, owner 2026-09-25)._

**Verdict: select, `priority: medium`.** The Why is real and confirmed against the current tree, not inferred:

- `frontend/utils/roadmap.ts` derives all 28 nodes from one integer and documents the lie itself: `state: day < today ? 'completed' : day === today ? 'today' : 'locked'` under the comment *"There is no per-day completion endpoint, so 'completed' means 'before today' — derived from GET /quests/daily day_number (open question)."* `harness/designs/frontend-shell.md` §2.5 and the CODEMAP `shell` paragraph (`/roadmap` "derived from `day_number`; no per-day endpoint") record the same open question. So a skipped day 2 is a ⭐ "Đã hoàn thành" while the pet lost 30 health for it — the two retention surfaces contradict each other on the happy path.
- No day has a name: `RoadmapNode.vue` renders glyph + "Ngày n" + one of three fixed strings. The names exist — `roadmaps.roadmap_json` is the `airouter.Roadmap` that `onboarding.PgRepo.SaveAssessment` marshals (`{title, cefr_level, modules[4]{week,title,focus,days[7]{title,tasks[3]{type,title,duration_minutes,content}}}}`), and `google` already renders "Day N: title · title · title" from it into Google Tasks, so the plan is visible in Google and invisible in the app.
- The join exists: `daily_progress` is unique on `(user_id, date)` with `minutes_spent`, `is_target_met` (`0001_init.up.sql`), written by `quests.RecordProgress` for the user's local date. Read-only, no AI call, no new write, no migration.

**Achievable in one plan:** yes — one additive backend read in the package that already owns `roadmaps`/`exercises`/`daily_progress` reads and the day arithmetic, plus a frontend page/store/util rewrite of ~200 lines. Medium, not high: nothing is blocked and no data is at risk; it is the first thing a returning learner sees that is wrong, which is clear retention value (role: `medium` = clear retention/learning value).

**Decisions taken (the plan carries them; not to be re-litigated):**
1. `GET /api/v1/roadmap` lives in `quests` behind `auth.Require()`, body `{roadmap_id, title, cefr_level, created_at, day_number, modules[4]{week, title, focus, days[7]{day_number, date, title, tasks[3]{task_type, title, duration_minutes}, minutes_spent, is_target_met}}}`; exercise `content_json` is not included; `404 no_active_roadmap` like `/quests/daily`. Backend spec §6.2 wins for the wire; both spec files gain the route in their escaped style. `quests` stays inside its boundary: `roadmaps`, `daily_progress`, `users.timezone`; no pet tables, no Redis for this read.
2. `date` for day N = local date of `roadmaps.created_at` + N−1 in `users.timezone` — the inverse of `DayNumber` (`day.go`) and the rule `google.DayDue` already applies to Tasks `due`. One new helper `DayDate` in `day.go` built on the same `startOfDay`; the plan pins the round-trip `DayNumber(created, at(DayDate(created, n)), loc) == n` on `day_test.go`'s DST fixtures so the tree and the daily suite can never disagree about which date a day is.
3. Frontend: a new `stores/roadmap.ts` (not a section of `quest.ts` — different resource and lifecycle, and today's bug plan `2026-09-25-a-pet-state-failure…` edits `stores/quest.ts`); `utils/roadmap.ts` `roadmapNodes(outline)` takes the outline (which carries per-day progress) instead of an integer; states `completed` / `partial` ("12/30 phút") / `missed` / `today` / `locked`; module headers; tap-to-expand task titles with durations; "Đã hoàn thành 9/28 ngày"; today scrolls into view; empty state and "Tạo lộ trình 28 ngày" unchanged; `/api/v1/roadmap` joins the NetworkFirst `api-state` cache. Design: `harness/designs/roadmap-tree.md`.
4. Same-day overlap: the approved bug plan above edits `quests/service.go` (`ProgressResult`, the tail of `RecordProgress`), `service_test.go`, appends to `handler_test.go`, and edits `stores/quest.ts` + `stores/pet.ts`. This plan adds new files and new functions only, appends nothing to `handler_test.go`/`service_test.go`, and never reorders or reformats an existing declaration, so the daily integration merge is clean.

**Dependencies:** none — everything it reads is on `origin/main`. The 2026-09-24 idea *day-28 checkpoint* (selected, unplanned) may later extend this read with roadmap history; nothing here waits on it.

**Status:** planned today — `harness/plans/2026-09-25-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md` (`draft`; medium feature → owner's `/approve`).
