---
type: bug
status: planned
source: human
run: 2026-09-22-run-01
priority: high
plan: harness/plans/2026-09-22-add-unique-user-id-to-pet-states-ddl.md
---
# Add UNIQUE user_id to pet_states DDL

## Why
<!-- Business rationale. Tie to spec goals: retention (30 min/day), CEFR progression, gamified pet engine. -->

## Expected output
- In `1st-thinking-architecture-doc.md` §3.2, the `CREATE TABLE pet_states` statement's `user_id` column reads `user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE` (keeping the document's existing backslash-escaping style).
- No other content in the document changes.
- The `store` entry in `harness/CODEMAP.md` notes that `pet_states.user_id` is unique (1:1).

## Evidence
- Spec §3.1 ERD: `FK user_id: UUID (1:1)` under `pet_states`.
- Spec §3.2 DDL: `user_id UUID REFERENCES users(id) ON DELETE CASCADE` — no UNIQUE.
- Human proposal (project owner), raised during the 2026-09-22 CLAUDE.md init review.

## Evaluation
**Verdict: select, priority high.**

- *Why is real:* the pet engine is the retention core of the product (spec §1 gamified pet, §5 decay/streak). §3.1 ERD declares `pet_states.user_id` as `(1:1)`, but the §3.2 DDL has no `UNIQUE`, so nothing stops a second `pet_states` row per user. `GET /pet/status` and `POST /pet/revive` would then read/write an ambiguous row — a correctness bug baked into the schema before any code exists. The `store` package generates its migrations from §3.2, so the spec is the single source of truth to fix.
- *Root cause (read-only, main checkout):* `1st-thinking-architecture-doc.md` line 196, inside `CREATE TABLE pet\_states (` (line 192): `    user\_id UUID REFERENCES users(id) ON DELETE CASCADE,` — missing `UNIQUE NOT NULL`. Contrast line 78 (ERD: `FK  user\_id: UUID (1:1)`) and the document's own style for unique columns on lines 156/160 (`email VARCHAR(255) UNIQUE NOT NULL`). The other three `user\_id ... REFERENCES users(id)` lines (180, 216, 232) are legitimately 1:N and must not change.
- *Achievable in one plan:* yes — a one-line spec edit plus one CODEMAP sentence; well under an hour.
- *Dependencies:* none. No app code or migrations exist yet, so no data migration is needed; landing this before the `store` MVP slice avoids a later `ALTER TABLE`.
- *Priority rationale:* `high` — it blocks the MVP `store` slice from generating a correct schema, and it is a confirmed spec defect raised by the project owner.
