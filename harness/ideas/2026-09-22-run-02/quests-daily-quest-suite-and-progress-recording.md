---
type: mvp-slice
status: planned
source: ideator
run: 2026-09-22-run-02
order: 3
priority: high
plan: harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md
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

## Evaluation

**Verdict: select — priority `high`.**

**Is the *Why* real?** Yes. §1 names "at least 30 minutes/day" as the retention objective and §5.2 steps 1–3 are the only write path that produces it. `daily_progress.is_target_met` has no other writer in the whole spec. The second claim in the *Why* — that Redis-first ordering is what keeps the threshold computed from one source under bursty clients — is exactly what §5.2 draws (INCRBY, read the total back, *then* mark progress) and what CODEMAP already records for this package, so the ordering is a contract, not a preference. Worth pinning with a test that fails if someone reorders it.

**Is the *Expected output* achievable in one plan?** Yes. Two endpoints, one counter, one upsert, one listener interface and a seed fixture. The riskiest parts are small and self-contained: `day_number` arithmetic across a timezone boundary, and firing `OnTargetMet` exactly once. Both are pure functions given a clock and a timezone, so both are cheap to test exhaustively.

**Dependencies:** `store` (1) for the module, `roadmaps`/`exercises`/`daily_progress` and `DailyAccumulatedKey`; `auth` (2) for `Require()` and `UserID(c)`. Both are queued ahead at `order` 1 and 2. The one genuine gap is that **no slice in this run creates `roadmaps` rows** — `_run.md` records that `onboarding` was deliberately omitted. The idea handles this itself with a seed fixture and a 404 `no_active_roadmap`, which is the right call: a fixture is throwaway, whereas inventing an onboarding endpoint here would be scope creep and a new API.

**Priority rationale:** `high` — it blocks MVP order. Slice 4 (pet) consumes `TargetMetListener`, and slice 7 (notify) suppresses reminders based on `is_target_met`.

**Test strategy (no live services).** Same shape as slices 1 and 2: `Counter`, `ProgressRepo` and `QuestRepo` are interfaces with in-memory fakes, so the whole package tests pure. The ordering assertion is made real by giving the fakes a **shared call log**: the test asserts the recorded sequence is `INCRBY`, `EXPIRE`, `upsert daily_progress`, `mark exercise complete`. The seed fixture gets a `DATABASE_URL`-gated integration test that skips.

**Design decisions taken in the plan that the idea left open:**
- `newly_met` is derived from the counter, not from a read of `daily_progress`: the INCRBY return value crossing 1800 for the first time is the single source of truth (`total-seconds >= 1800 && total-seconds-delta < 1800`). This is what makes "fires exactly once" hold without a second round-trip, and it is testable with arithmetic alone.
- `minutes_spent = total / 60` (integer division, as the idea states). 1799s therefore stores 29 minutes with `is_target_met = false`; 1800s stores 30 with `true`.
- `OnTargetMet` is called **after** the Postgres upsert commits and is best-effort: a listener error is logged, not returned. The pet's health update must not be able to roll back a recorded study session. Recorded as an open question since the spec does not say.
- `day_number` uses `users.timezone` (§3.2, default `'UTC'`) via `time.LoadLocation`, falling back to UTC on an unknown name rather than erroring — a bad timezone string must not lock a learner out of their quests.
- A `POST /quests/progress` for an exercise that does not belong to the caller's active roadmap is a 404, not a 403: it keeps other users' exercise ids unprobeable.

**Open questions recorded for the human (not blocking):**
1. **The seed fixture is a stopgap.** It exists only because `onboarding` has no slice in this run (`_run.md` first Note). When onboarding lands, the fixture should be deleted, not extended. The plan puts it behind an explicit `SeedDemoRoadmap` helper with a doc comment saying so, rather than scattering INSERTs through tests.
2. **`day_number` clamps at 28 rather than ending the roadmap.** §6.1 fixes the roadmap at 28 days but the spec never says what day 29 looks like. Clamping keeps a returning learner on day 28's quests forever, which is wrong long-term but harmless for the MVP. The real answer (generate a new roadmap, or surface a "course complete" state) needs a product decision.
3. **Nothing resets `exercises.is_completed`,** so a learner who finishes day 5 sees it permanently complete. That matches the §3.2 column (it is per-exercise, not per-day), but it means the daily screen shows day N's three tasks already ticked if the learner revisits. Flagged, not fixed.
4. **A crash between INCRBY and the Postgres upsert loses the Postgres row but keeps the Redis count.** The next progress call re-derives the total from Redis and re-upserts, so the system self-heals within the 48h TTL. Accepted rather than solved; recorded in CODEMAP so a reviewer does not file it as a bug.

**Reconciliation note (2026-09-22).** The *Expected output* above was written against the 1st-thinking doc, before the *Backend Technical Specification* existed. Its §6.2 JSON blocks are now the wire contract, and the plan's `## Reconciliation` section carries the field-for-field mapping: `GET /quests/daily` → `{date, day_number, total_minutes_required, accumulated_seconds, is_target_met, tasks[{id, task_type, title, duration_minutes, is_completed, content_json}]}`; `POST /quests/progress` takes `{exercise_id, duration_seconds, user_answers?}` and returns `{daily_seconds_spent, daily_minutes_spent, is_target_met, pet_health, streak_count}`. `newly_met` is internal only; `TargetMetListener` became `quests.Pet` (`OnTargetMet` + `State`). Where this note and the bullets above disagree, the plan wins.
