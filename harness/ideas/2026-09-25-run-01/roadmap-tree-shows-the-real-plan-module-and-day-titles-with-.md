---
type: feature
status: proposed
source: ideator
run: 2026-09-25-run-01
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
