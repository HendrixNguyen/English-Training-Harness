---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-25-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md
---

# A Pet.State failure reports pet_health 0 which means a dead plant

## Why
When `Pet.State` returns an error the service logs it and substitutes the zero value
(`backend/internal/quests/service.go:102-106`), so the §6.2 response carries
`"pet_health": 0, "streak_count": 0`. Zero is not a neutral placeholder in this domain: backend spec
§8 and the 1st-thinking doc define health as a 0-100 percentage where 0 is a dead plant and the
revive challenge (§6.3 `POST /pet/revive`) is the way back. A transient `pet_states` read failure —
a dropped connection, a slow query, a restarting pet slice — therefore tells the client, with a 200,
that the learner's plant just died and their streak reset, on the very call that recorded a
successful study session.

The frontend has no way to distinguish it: the field is a plain integer and the status is 200. The
Frontend spec's §5 mapping drives the pet animation from exactly these numbers, so the user sees a
death animation for a backend hiccup. The choice is recorded in the plan as an open question
("Confirm"), so it has not actually been decided — this bug is the reviewer's answer: the 200 is
right, the zeros are not.

The decision *not* to fail the request is correct and should be kept: the progress write has already
committed and a 500 would invite a double-counting retry (see the unbounded/ordering bugs).

## Expected output
A `Pet.State` failure never fabricates a plant state. Either:
- omit both fields (`omitempty` on pointer fields, or a `*int`) so the client keeps whatever it last
  read from `GET /pet/status` and renders no transition; or
- carry the last known value, if a later slice caches it.

`NopPet`'s (100, 0) placeholder is fine and unrelated — it is a real "freshly onboarded" state, not
an error path. `TestAPetStateFailureDoesNotFailTheRequest` (`service_test.go:187-201`) currently
asserts `PetHealth == 0` as the desired outcome and is updated with the chosen shape; the
interface doc comment in `pet.go:23-26` should also say what `State` must return for a user with no
`pet_states` row yet (the reviewer's reading: a not-found is not an error — it is the §3.2 defaults),
because the pet slice will otherwise have to guess.

## Evidence
- Plan: `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md` (*Reconciliation* -> open questions, "`Pet.State` failure -> log and report (0, 0) … Confirm").
- `backend/internal/quests/service.go:100-106`.
- `backend/internal/quests/service_test.go:187-201`.
- Backend spec §8 (plant math, 0 = dead, revive challenge) and §6.3.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — medium.** Still true on `main`: `quests/service.go:133-137` substitutes `PetState{}` on error, and the frontend's `pet.applyProgress` writes `pet_health: 0` straight into the plant hub — a transient `pet_states` read failure shows a dead plant right after a successful session. Fix: `pet_health`/`streak_count` become `*int` with `omitempty` (frontend `applyProgress` already tolerates missing fields by guarding on `this.status`; verify and adjust the store to skip `undefined`). Pet is the retention mechanism; a false death is the worst message we can show. Small, self-contained plan in `quests` + one store guard.

_Evaluator, 2026-09-25 — daily decide (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Planned today (still medium).** Re-confirmed on `main` (2026-09-25): `quests/service.go` still substitutes `PetState{}` when `s.pet.State` fails, `ProgressResult.PetHealth`/`StreakCount` are plain `int`, and `frontend/stores/pet.ts` `applyProgress` writes `res.pet_health` straight into `status.health_points` — a transient `pet_states` read shows a dead plant on the hub right after a successful session. Smallest self-contained fix in the queue with direct retention impact: the two fields become `*int` with `omitempty`, the store skips an absent field, and the `Pet` interface documents that a missing row is not an error (`pet.QuestHook.State` already goes through `Ensure`). Plan: `harness/plans/2026-09-25-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md`.
