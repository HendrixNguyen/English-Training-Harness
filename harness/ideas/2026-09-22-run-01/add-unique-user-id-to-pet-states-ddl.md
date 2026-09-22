---
type: bug
status: proposed
source: human
run: 2026-09-22-run-01
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
