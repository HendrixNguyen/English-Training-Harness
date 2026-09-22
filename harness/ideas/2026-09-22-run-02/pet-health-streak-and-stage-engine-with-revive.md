---
type: mvp-slice
status: planned
source: ideator
run: 2026-09-22-run-02
order: 4
priority: high
plan: harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md
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

## Evaluation

**Verdict: select — priority `high`.**

**Is the *Why* real?** Yes. The plant is the only feedback loop in the spec that makes a missed day *cost* something: §5.2 steps 4–5 reward the target immediately, and the backend spec §8 makes the miss visible the next morning (`Health = Max(0, Health - 30)`, `wilted` at 0). Without this package `POST /quests/progress` reports `pet_health: 100, streak_count: 0` forever (the quests plan's `NopPet` placeholder) and the frontend's pet view has nothing to draw. Run-01's `pet-streak-shield` and `pre-decay-rescue-push` both need the decay engine to exist first.

**Is the *Expected output* achievable in one plan?** Yes. Pure arithmetic (`+20` cap 100, `-30` floor 0, stage from streak), one 1:1 table, two routes, one Redis key, one hourly sweep. Every piece of logic is a function of `(state, clock)` so it tests without I/O; only row creation needs a live database.

**Dependencies:** `store` (1) and `auth` (2) are merged. `quests` (3) is executing now on `harness/2026-09-22-high-quests-daily-quest-suite-and-progress-recording`; its plan defines the interface this slice implements — `quests.Pet { OnTargetMet(ctx, userID, localDate string) error; State(ctx, userID string) (quests.PetState, error) }` with `quests.PetState{Health, Streak int}` — and `main.go` registers `quests.NopPet{}` where this slice's implementation goes. Pet imports `quests` (for the interface types); `quests` never imports `pet`, so there is no cycle. The plan is written against that interface, not a fresh design, and must land after quests merges.

**Spec reconciliation (backend spec wins for its layer, AGENTS.md → *Reading the spec*):**
- **Success (+20 / streak++) fires once, at progress time, from the quests hook.** Backend spec §6.2 says `POST /quests/progress` "increases plant health (+20%), and increments streak"; §8's "Success Logic" carries the same arithmetic on the hourly cron. Applying both would double-bump. Decision: the hook (quests fires `OnTargetMet` exactly once per user per local day) applies §8's success logic; the cron applies **only** §8's inactivity logic. Recorded in the plan and CODEMAP.
- **Miss penalty is −30, not −20.** The *Expected output* above proposed −20 "mirroring +20"; backend spec §8 fixes it at 30. 100 → 70 → 40 → 10 → 0 (`wilted`) after four missed days. §8 says nothing about the streak on a miss; a streak that survives a missed day is not a streak, so `current_streak = 0` on a miss (recorded as a decision).
- **Revive passes at health 50, not 20.** Backend spec §6.3: "Resets health to 50% upon passing", response `{"revival_passed": true, "pet_state": {"health_points": 50, "stage": "sprout", "current_streak": 0}}`. The request body is `{"answers": {...}}`; nothing in either spec says how answers are graded, so they are accepted (like quests' `user_answers`) and the pass condition is the one both §7 lines state — a 15-minute challenge: ≥ 900 s of `POST /quests/progress` study recorded since the challenge started, on the local day it started. Progress is read through a `StudyCounter` interface satisfied by `quests.RedisCounter.Total` (no table coupling).
- **`GET /pet/status`** returns the §6.3 body `{plant_name, stage, health_points, current_streak, last_practiced_at}` field for field; `last_practiced_at` is `null` until the first target is met.
- **Stage thresholds** (spec gives none): `wilted` iff `health_points = 0`; otherwise by streak — 0–2 `sprout`, 3–6 `sapling`, 7–13 `flowering`, 14+ `fruitful`. `seed` stays unreachable, matching the DDL default `'sprout'`; the `reconcile-pet-states-stage-…` bug remains the spec-side fix.
- **Revive Redis key** (not in §4): `pet:revive:{user_id}`, Hash `{started_at, local_date, start_seconds}`, TTL 24 h — long enough to span the local day the challenge is bound to, shorter than the 48 h `daily:accumulated` TTL the pass check reads. Builder `store.PetReviveKey` + `store.PetReviveTTL`; documented in CODEMAP.
- **Row creation:** `INSERT INTO pet_states (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING` on first `GET /pet/status` (and from onboarding via the same `Ensure`), so the `spec-never-states-when-the-pet-states-row-is-created` bug is answered in code; the spec sentence stays a separate low bug.
- **Cron:** in-process (§2.1), fires at `:00` UTC every hour (§8), selects users whose `users.timezone` is currently in local hour 0, and applies the miss to those whose *previous* local day has `< 1800` s on the daily counter. Idempotency guard: skip a pet whose `updated_at` is already past that local midnight (a re-run in the same hour, or a restart, cannot double-penalise).

**Priority rationale:** `high` — MVP `order` 4; the frontend-shell pet screen and two run-01 ideas depend on it.

**Test strategy:** pure unit tests with fakes and an injected clock for the arithmetic, the hook, revive start/pass/409, and the sweep (timezone selection, miss vs met, idempotency); one `TestIntegrationEnsureCreatesExactlyOnePetRow` (gated on `TEST_DATABASE_URL` only, counted by CI) proving concurrent `Ensure` calls create one row and no error.
