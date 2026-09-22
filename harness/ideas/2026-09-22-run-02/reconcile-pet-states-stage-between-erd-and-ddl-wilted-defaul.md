---
type: bug
status: proposed
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
