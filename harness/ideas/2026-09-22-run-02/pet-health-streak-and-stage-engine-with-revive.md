---
type: mvp-slice
status: proposed
source: ideator
run: 2026-09-22-run-02
order: 4
---
# Pet: health, streak and stage engine with revive

## Why
The plant is the loss-aversion loop that makes the 30-minute target sticky: hitting the target is rewarded immediately (§5.2 steps 4–5, "Health += 20%, Streak++") and missing it is visible the next morning as a wilting plant. Without decay the pet is a static badge; without revive a lapsed user has no way back and churns. This slice is the first one whose output a user actually feels, and it is what run-01's feature ideas (streak shield, pre-decay rescue push) build on.

## Expected output
Delivers (Go package `backend/internal/pet`, routes behind `auth.Require()`):
- `GET /api/v1/pet/status` — returns `{plant_name, stage, health_points, current_streak, last_practiced_at}` from `pet_states`. Calls `pet.Ensure(ctx, userID)` first: `INSERT INTO pet_states (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, so the 1:1 row exists idempotently (resolves the open question in `spec-never-states-when-the-pet-states-row-is-created.md` for the code; the spec fix stays a separate bug).
- `POST /api/v1/pet/revive` — allowed only when `health_points = 0` (stage `wilted`), otherwise 409. Starts a 15-minute revival challenge: the user must accumulate ≥900s that day via `POST /quests/progress`; when they do, health → 20 and stage → `sprout`, streak stays 0. Tracking the active challenge needs one Redis key not in §4 (e.g. `pet:revive:{user_id}`, 24h TTL) — executor documents it in CODEMAP.
- `quests.TargetMetListener` implementation: `health = min(100, health + 20)`, `current_streak++`, `last_practiced_at = now`, stage recomputed from streak thresholds. The spec gives no thresholds; the executor picks a monotone mapping streak → `seed/sprout/sapling/flowering/fruitful` and records it in CODEMAP.
- Decay job on the in-process cron (§2.1 "integrated background cron worker"): once per user-local day, for users whose previous day has no `daily_progress.is_target_met = true`: `health -= 20`, `current_streak = 0`; at 0 set stage `wilted`. Decay amount mirrors the spec's +20 and is recorded in CODEMAP.
- Tests: +20 caps at 100; five missed days go 100 → 0 and `wilted`; revive rejected at health > 0; revive completes at 900s; concurrent `Ensure` calls do not error (UNIQUE + ON CONFLICT).
- Tables: `pet_states` (read/write), `daily_progress` (read). Redis keys: none from §4; one new `pet:revive:{user_id}` (documented). Screens: none (pet view is in frontend-shell).

Depends on: store (1), auth (2), quests (3) for `TargetMetListener` and `daily_progress`.

## Evidence
- Spec §5.2 (lines 304–332) steps 4–5: "Execute Pet Health State Change (Health += 20%, Streak++)" → "Render Growth Animation & Status".
- Spec §7 (lines 678, 680) `GET /api/v1/pet/status` (stage, health %, streak); `POST /api/v1/pet/revive` ("15-minute revival challenge when plant health hits 0%").
- Spec §3.2 `pet_states` (lines 192–210): `health_points` 0–100 CHECK, `stage pet_stage DEFAULT 'sprout'`, `current_streak`, `last_practiced_at`; `pet_stage` includes `wilted` (line 148).
- Spec §2.1 (line 12) Go binary "with integrated background cron worker".
- `harness/CODEMAP.md` → `pet`: "decay on missed target, revive challenge".
- Bugs swept into this run: `spec-never-states-when-the-pet-states-row-is-created.md`, `reconcile-pet-states-stage-between-erd-and-ddl-wilted-defaul.md`.
- Prior run `harness/ideas/2026-09-22-run-01/pet-streak-shield-earned-by-target-days.md` depends on this engine.
