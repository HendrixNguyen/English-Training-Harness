---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: medium
rejected_reason: "The frontend already maps 409 pet_not_wilted on the revive check to the 'your plant is healthy' screen (stores/pet.ts notWilted → pages/revive.vue), so the user sees the recovery, and the leftover pet:revive key expires within 24h and is replaced next day; no user-visible defect remains."
---
# Meeting the daily target during a revive challenge answers 409 and leaks the Redis key for 24h

## Why
`Service.Revive` rejects on health before it looks at the challenge:

```go
// backend/internal/pet/service.go:69-75
st, err := s.Ensure(ctx, userID)
...
if st.HealthPoints > 0 {
        return ReviveResult{}, ErrNotWilted
}
```

So a user who starts the 15-minute revival challenge and then simply keeps studying past the
30-minute daily target is answered `409 pet_not_wilted` on the follow-up call — even though they
recorded twice the study the challenge asked for — and the `pet:revive:{user_id}` Hash is never
reached by `Clear`, so it sits in Redis for its full 24 h TTL.

Reproduced on the live stack (scratch `petrev` compose project, 2026-09-23):

```
revive #1                       -> 200 {"revival_passed":false,"pet_state":{"health_points":0,"stage":"wilted","current_streak":0}}
redis hash after start           = started_at 2026-09-23T02:17:07Z / local_date 2026-09-23 / start_seconds 0, ttl 86400
POST /quests/progress 900s       -> {"is_target_met":false,"pet_health":0,"streak_count":0}
POST /quests/progress 1800s      -> {"is_target_met":true,"pet_health":20,"streak_count":1}   (OnTargetMet fired)
revive #2 (1800s recorded >= 900) -> 409 {"error":"pet_not_wilted"}
revive key still present?        -> 1, ttl 86400
db                               -> 20|sprout|1
```

Two problems, one user-visible and one operational:

- **The client cannot tell success from rejection.** A frontend that opened a countdown on
  `revival_passed: false` and polls until it flips gets a 409 forever. §6.3 gives it no other
  signal, and the plan's *Notes* explicitly leave progress off the DTO ("a client wanting a
  countdown can read `accumulated_seconds` from `GET /quests/daily`") — which does not tell it the
  challenge is over either.
- **The key is orphaned.** The only `Clear` is on the pass path (`service.go:111`). Every voided
  challenge leaves a Hash behind. It is harmless in itself — the next day's call sees
  `c.LocalDate != today` and replaces it — but it means `pet:revive:*` is not a reliable answer to
  "who has a challenge open", which is the first thing an operator or a later notification slice
  would ask it.

The health outcome itself is defensible: the plant recovered by the normal route, and 20 is what
§8's `+20` gives a wilted plant. This is about the response and the leftover state, not the
arithmetic.

## Expected output
Finishing the day's target while a challenge is open is a success, not a conflict:

- `Revive` reads the challenge **before** the health check. If a challenge is open for today and
  `total - start_seconds >= ReviveSeconds`, it clears the key and returns
  `{revival_passed: true, pet_state: …}` with whatever health the pet now has — no 409 — so a
  polling client terminates;
- if no challenge is open and health is above 0, the 409 `pet_not_wilted` stays exactly as it is
  (this is the §6.3 path, covered by `TestReviveReturns409WhileNotWilted`);
- the key is cleared on every terminal outcome, including the voided one, so `pet:revive:*` holds
  only genuinely open challenges;
- `service_test.go` gains the sequence above as a test: wilted → `Revive` (starts) → health raised
  to 20 by `OnTargetMet` → `Revive` returns passed with the key cleared, asserting
  `challenges.cleared == 1` and `len(challenges.items) == 0`.

## Evidence
- Plan: `harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md` (Task 5,
  `Service.Revive`; *Notes* — "Revive response carries no progress").
- `backend/internal/pet/service.go:69-75` — the health check before the challenge read.
- `backend/internal/pet/service.go:111-114` — `Clear` only on the pass path.
- Reviewer runtime probe, 2026-09-23, live Postgres + Redis (`petrev` compose project), transcript
  quoted above.
- Backend spec §6.3 — `POST /api/v1/pet/revive`, "Resets health to 50% upon passing".

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Reject.** Reproduced logic is right, but the user impact the finding assumed ("a polling client gets a 409 forever") does not occur: `pet.revive()` turns `pet_not_wilted` into `notWilted = true` and `/revive` renders "Cây của bạn vẫn khỏe". The orphaned Redis key is harmless (24 h TTL, replaced by the next day's challenge). Not worth a branch.
