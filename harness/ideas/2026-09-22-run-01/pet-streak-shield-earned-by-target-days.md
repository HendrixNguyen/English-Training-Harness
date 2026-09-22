---
type: feature
status: proposed
source: ideator
run: 2026-09-22-run-01
---
# Pet Streak Shield Earned by Target Days

## Why
The pet engine as specified (§5.2) is pure loss aversion: miss the 30-minute target once and health decays, the streak resets, and the user faces a revive challenge. That is exactly the moment learners churn — Duolingo's data shows the churn risk cluster is users about to break a streak, and adding a Streak Freeze cut churn in that cohort by 21% while keeping the streak's pull intact (7-day-streak users are 2.4x more likely to return next day). We want the same asymmetry: keep the daily ≥30-minute pressure, but make one bad day survivable. Earning the shield only by hitting the target on 7 consecutive days ties the safety net to the exact behaviour we are trying to build, rather than giving it away, and the "silent apply, discover next morning" pattern turns a would-be quit moment into a relief moment that reinforces the pet bond.

## Expected output
User-visible:
- After 7 consecutive `is_target_met` days the pet view shows one shield icon (max 2 held); the daily-quest completion animation announces "Shield earned".
- On a missed day, if a shield is held, it is consumed automatically: no health loss, streak preserved, no revive challenge. Next open shows the shield spent on that calendar day in the pet view, with a one-line explanation.
- With no shield, behaviour is unchanged from §5.2 (decay, streak reset, revive).

Technical:
- `pet_states` gains `shields INT NOT NULL DEFAULT 0 CHECK (shields BETWEEN 0 AND 2)` and `last_shield_used_on DATE NULL` via a new store migration.
- `pet` package: streak++ path awards a shield every 7th consecutive target-met day; decay path consumes a shield before applying health/streak penalties. Unit tests for award, cap, consume, and no-shield fallthrough.
- `GET /pet/status` response includes `shields` and `last_shield_used_on`.
- Frontend pet view renders shield count and spent-shield marker; CODEMAP `pet` paragraph updated.

## Evidence
- Spec §3.1 `pet_states` (health_points, current_streak, last_practiced_at); §5.2 step 4 "Health += 20%, Streak++"; §1 retention goal.
- CODEMAP: `pet` package ("decay on missed target, revive challenge"), `store` migrations.
- Duolingo streak/freeze data: https://duolingo.deconstructoroffun.com/mechanics/streaks and https://www.strivecloud.io/blog/gamification-examples-boost-user-retention-duolingo
- Implementation breakdown of freeze caps and silent application: https://engagefabric.com/blog/building-duolingo-style-streak-system
