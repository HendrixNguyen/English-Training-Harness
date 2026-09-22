---
type: mvp-slice
status: proposed
source: ideator
run: 2026-09-22-run-02
order: 3
---
# Quests: daily quest suite and progress recording

## Why
"≥30 minutes a day" is the retention metric the whole product is built around, and this slice is the write path that turns study time into `daily_progress.is_target_met` (§5.2 steps 1–3). Without it the pet has nothing to react to and the reminder worker has nothing to suppress. It also fixes the one ordering rule the spec insists on — Redis counter first, then Postgres — so that the 30-minute threshold is computed from a single source even when the client sends progress in bursts.

## Expected output
Delivers (Go package `backend/internal/quests`, routes behind `auth.Require()`):
- `GET /api/v1/quests/daily` — resolves the user's active roadmap (`roadmaps.is_active`), computes `day_number` = days since `roadmaps.created_at` in `users.timezone` + 1 (clamped to 1..28), and returns that day's three `exercises` rows (`vocabulary`, `reading`, `practice`) with `content_json`, `is_completed`, plus today's `total_seconds` and `target_met`. With no active roadmap returns 404 `{error:"no_active_roadmap"}` (onboarding is not sliced in this run; see `_run.md` Notes).
- `POST /api/v1/quests/progress` — body `{exercise_id, seconds}`; `INCRBY daily:accumulated:{user_id}:{YYYY-MM-DD}` (date in the user's timezone, `EXPIRE` 48h) **first**, then upsert `daily_progress` on `UNIQUE(user_id, date)` with `minutes_spent = total/60` and `is_target_met = total >= 1800`, then set `exercises.is_completed = true`. Response `{total_seconds, target_met, newly_met}`.
- An in-process hook interface `quests.TargetMetListener` (`OnTargetMet(ctx, userID, date)`) called exactly once when `newly_met` flips to true; the pet slice registers the implementation.
- A dev/test fixture (`store` seed helper or SQL) that inserts a 28-day roadmap with 3 exercises per day so `GET /quests/daily` returns data before onboarding exists.
- Tests: INCRBY precedes the Postgres upsert; crossing exactly 1800s sets `is_target_met` and fires the listener once; a second progress call the same day does not re-fire; day_number clamps at 28; timezone boundary picks the right `date`.
- Tables: `roadmaps`, `exercises` (read; `is_completed` write), `daily_progress` (upsert). Redis keys: `daily:accumulated:{user_id}:{YYYY-MM-DD}`. Screens: none (daily quest screen is in frontend-shell).

Depends on: store (1), auth (2).

## Evidence
- Spec §5.2 (lines 304–332) steps 1–3: complete task → `INCRBY 600sec` → return total → mark progress `TargetMet = True`.
- Spec §7 (lines 674, 676) `GET /api/v1/quests/daily` (3x 10-min tasks) and `POST /api/v1/quests/progress` (updates Redis counter and daily progress).
- Spec §6.1 (lines 336–350) 4 modules × 7 daily quests × 3 tasks of 10 min — defines `day_number` range 1..28 and the three `task_category` values.
- Spec §3.2 `daily_progress` (lines 212–226) `UNIQUE(user_id, date)`; `exercises` (lines 242–256).
- Spec §4 (line 264) `daily:accumulated:{user_id}:{YYYY-MM-DD}` String(Int), 48h.
- `harness/CODEMAP.md` → `quests`: "Redis INCRBY first, then Postgres daily_progress".
