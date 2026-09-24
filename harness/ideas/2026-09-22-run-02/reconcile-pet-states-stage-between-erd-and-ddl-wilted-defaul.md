---
type: bug
status: selected
source: reviewer
run: 2026-09-22-run-02
priority: low
---
# Reconcile pet_states stage between ERD and DDL (wilted, default)

## Why
Same class of §3.1-vs-§3.2 drift that `add-unique-user-id-to-pet-states-ddl` fixed for `user_id`, in the same table. The ERD range `('seed'..'fruitful')` omits `'wilted'`, which is the state the §5 decay/revive engine (`POST /pet/revive` at 0% health) needs; and the DDL default is `'sprout'`, so the enum's first value `'seed'` is never reachable by any flow described in the doc. Anyone modelling the pet engine from §3.1 gets 5 stages and no wilted state; anyone reading §3.2 gets 6 and a start state that is not the first one. Found while reviewing the plan below; explicitly out of that idea's scope ("No other content in the document changes"), so filed separately.

## Expected output
- §3.1 ERD `pet\_states` block and §3.2 `CREATE TYPE pet\_stage` agree on the stage set, including `wilted` (e.g. ERD `stage: ENUM('seed'..'wilted')` or an explicit list), keeping the document's backslash-escaping style.
- The `pet\_states.stage` DEFAULT is either `'seed'` (matching the enum's first value) or the doc says in one sentence why a new pet starts at `'sprout'` and what `'seed'` is for.
- No other DDL or ERD content changes.

## Evidence
- Plan under review: `harness/plans/2026-09-22-add-unique-user-id-to-pet-states-ddl.md` (review 2026-09-22).
- `1st-thinking-architecture-doc.md:84` — `stage: ENUM('seed'..'fruitful')` (ERD).
- `1st-thinking-architecture-doc.md:148` — `CREATE TYPE pet\_stage AS ENUM ('seed', 'sprout', 'sapling', 'flowering', 'fruitful', 'wilted');`
- `1st-thinking-architecture-doc.md:202` — `stage pet\_stage DEFAULT 'sprout',`
- `grep -n -i wilted 1st-thinking-architecture-doc.md` → only line 148; the state is never mentioned in the ERD or any flow.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low. Not planned today.**

*Is the Why real?* The drift is real and cheap to fix, and the code has since settled the answer the ERD lacks: `backend/internal/pet/engine.go` `StageFor` yields `wilted` iff health is 0 and never produces `seed`; the DDL default is `sprout`; CODEMAP's pet bullet records both. The 1st-thinking doc's §3.1 ERD still lists `('seed'..'fruitful')` and never mentions `wilted`.

*Scope when planned (spec-only, one small edit set).* §3.1 ERD lists the six stages including `wilted`; a one-sentence note next to the §3.2 default says a new pet starts at `sprout` and `seed` is reserved; and — folded in from `spec-never-states-when-the-pet-states-row-is-created.md` (rejected today as moot) — one sentence in §5.1 and at §7 `GET /api/v1/pet/status` stating that the `pet_states` row is created idempotently (`INSERT … ON CONFLICT (user_id) DO NOTHING`) at onboarding completion and on first `GET /pet/status`, which is what `pet.Service.Ensure` does. Keep the document's backslash-escaping. Low: no runtime effect; it keeps the canonical spec honest for the next reader.
