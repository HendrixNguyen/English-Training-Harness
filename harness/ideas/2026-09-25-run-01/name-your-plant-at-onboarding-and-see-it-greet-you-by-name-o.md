---
type: feature
status: proposed
source: ideator
run: 2026-09-25-run-01
---
# Name your plant at onboarding and see it greet you by name on the hub

## Why
The plant is the retention mechanism (1st-thinking §1), and the schema already gives it a name: `pet_states.plant_name VARCHAR(100) DEFAULT 'My Green Buddy'`, returned by `GET /pet/status` and by the onboarding result. Nothing lets the learner set it, so every plant in the product is called "My Green Buddy" — in English, in a Vietnamese-language app — and the onboarding success screen says "My Green Buddy đã nảy mầm". A pet the learner did not name is a widget; a pet they named is theirs. Naming is the cheapest ownership mechanic there is (Tamagotchi, Duolingo's Duo, every habit-pet app), it costs one text field, and it gives the speech bubble and the wilted alarm a name to use ("Mầm Non đang héo!" hits harder than "Cây xanh đang bị héo rũ!"). It also fixes a visible localisation wart on the very first screen a paying learner sees after the quiz.

This idea rides the existing onboarding request (an optional additive field on `POST /onboarding/assessment`) rather than adding a new endpoint; a rename later can be a follow-up once the settings screen (approved, 2026-09-24) exists.

## Expected output
User-visible:
- The onboarding goal step (frontend spec §7.1) gains one optional field "Đặt tên cho cây của bạn" with a Vietnamese placeholder name suggested (e.g. "Mầm Non"); leaving it blank keeps a Vietnamese default rather than "My Green Buddy".
- The result step says "<name> đã nảy mầm…"; the hub shows the name above or beside the plant; the speech bubble and the wilted banner/`/revive` heading use the name ("<name> đang bị héo rũ!").
- Names are trimmed, 1–30 characters, any Unicode letters/digits/spaces; longer or empty-after-trim input is rejected inline before submit.

Technical:
- Backend `onboarding`: `AssessmentRequest` gains optional `plant_name` (validated: trimmed length 1..30 — `400 invalid_request` otherwise; absent/blank → default). `onboarding.Pet` interface gains the name: `pet.Service.Ensure(ctx, user, opts)` or a sibling `EnsureNamed` that does `INSERT … ON CONFLICT (user_id) DO UPDATE SET plant_name = EXCLUDED.plant_name WHERE pet_states.plant_name IS DISTINCT FROM …` — still idempotent and still 1:1. The re-submit path (active roadmap exists → `200`, no write) stays a no-op. The DDL default `'My Green Buddy'` is left alone (spec §3.2 must match `0001_init`); the Vietnamese default lives in the frontend placeholder and in `onboarding` when the field is blank.
- Backend spec §6.1 request example and 1st-thinking §7 description get the optional field in the escaped style, as `timezone` did for §6.4.
- Frontend: `composables/useOnboardingApi.ts` request type, `pages/onboarding.vue` field + validation, `pages/index.vue` name display, `utils/plant.ts` `speechLine`/wilted copy take a `name`, `pages/revive.vue` heading.
- Tests: Go — validation bounds, default when blank, name persisted on first assessment and unchanged on re-submit (fake pet records the call), integration test extends `TestIntegrationSaveAssessmentPersists84Exercises…` or adds a sibling gated on `TEST_DATABASE_URL`; Vitest — field validation, request body contains `plant_name` only when set, result copy uses the returned name.
- CODEMAP `onboarding`, `pet`, `shell` updated.

## Evidence
- 1st-thinking §1 (pet/plant engine as the retention driver), §3.2 DDL `pet_states.plant_name VARCHAR(100) DEFAULT 'My Green Buddy'`; backend spec §6.1 (`pet_state.plant_name` in the 201 body), §6.3 (`plant_name` in `GET /pet/status`).
- Frontend spec §7.1 (goal + reminder step — the natural home for one more field), §7.2 (hub with speech bubble), §7.5 (wilted alarm copy).
- Code: `backend/internal/pet/repo.go` `stateColumns` (`COALESCE(p.plant_name, 'My Green Buddy')`), `backend/internal/pet` `Service.Ensure` (`INSERT … ON CONFLICT (user_id) DO NOTHING`), `backend/internal/onboarding` `AssessmentRequest` and the `onboarding.Pet` adapter in `cmd/api/main.go`; `frontend/pages/onboarding.vue:143` ("{{ result.pet_state.plant_name }} đã nảy mầm"), `frontend/utils/plant.ts` `speechLine`, `frontend/pages/index.vue` wilted banner.
- Approved sibling that would host a later rename: `harness/plans/2026-09-24-settings-screen-wires-web-push-reminders-and-google-calendar.md`.
- Gamified pet ownership and personalisation as an engagement mechanic in language apps: https://blakecrosley.com/guides/design/duolingo ; https://darewell.co/en/duolingo-streaks-retention-secret/
