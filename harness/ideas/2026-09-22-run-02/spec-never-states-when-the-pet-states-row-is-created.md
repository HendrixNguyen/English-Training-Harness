---
type: bug
status: proposed
source: reviewer
run: 2026-09-22-run-02
priority: low
---
# Spec never states when the pet_states row is created

## Why
With `pet\_states.user\_id` now `UNIQUE NOT NULL` (plan below), exactly one pet row per user is enforced by the schema, but the spec never says who inserts it or when. §5.1 ends with "8. Init Dashboard" (line 302) without naming a pet-row insert, and §7 `GET /api/v1/pet/status` (line 678) simply "retrieves current plant stage", assuming the row exists. The `store`/`pet` MVP slices are generated from this spec; an executor who lazily inserts on first `GET /pet/status` will hit `unique_violation` on a concurrent first request or a retried onboarding step, and one who never inserts returns 404 for every new user. Low priority: no app code exists yet, but this should be pinned down before the `pet` slice is planned.

## Expected output
- One sentence in §5.1 (onboarding flow) states that the `pet\_states` row is created once per user at onboarding completion (step 8 "Init Dashboard" or the assessment handler), idempotently (`INSERT ... ON CONFLICT (user\_id) DO NOTHING` or equivalent).
- §7 `GET /api/v1/pet/status` states the behaviour when no row exists (either impossible after onboarding, or a documented 404/creation rule).
- `harness/CODEMAP.md` `pet` bullet notes where the row is created.

## Evidence
- Plan under review: `harness/plans/2026-09-22-add-unique-user-id-to-pet-states-ddl.md` (review 2026-09-22) — introduces the UNIQUE NOT NULL constraint.
- `1st-thinking-architecture-doc.md:196` — `user\_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,`
- `1st-thinking-architecture-doc.md:302` — `8. Init Dashboard` is the only onboarding step that could create the pet row; no insert is named.
- `1st-thinking-architecture-doc.md:678,680` — `GET /pet/status`, `POST /pet/revive` read/write the row without stating creation.
- `grep -n -i 'INSERT\|ON CONFLICT\|upsert' 1st-thinking-architecture-doc.md` → no hits.
