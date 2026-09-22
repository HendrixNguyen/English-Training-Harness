---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---

# No index supports the quests lookups on roadmaps and exercises

## Why
Both quests endpoints run, on every request, two lookups that have no supporting index:

```sql
SELECT id, created_at FROM roadmaps  WHERE user_id = $1 AND is_active = TRUE ORDER BY created_at DESC LIMIT 1
SELECT ...            FROM exercises WHERE roadmap_id = $1 AND day_number = $2 ORDER BY task_type
```

`0001_init.up.sql` declares no index on `roadmaps(user_id)`, `roadmaps(is_active)` or
`exercises(roadmap_id, day_number)` — a foreign key creates none on the referencing side in
Postgres, and the only indexes in the schema are the primary keys and the declared `UNIQUE`
constraints. `exercises` grows at 84 rows per user, so at any real user count the daily screen is
doing a sequential scan of the whole table per request, twice a session (once for `GET /daily`, once
per `POST /progress`). `MarkComplete`'s `WHERE id = $1 AND roadmap_id = $2` is fine — it is on the
primary key.

This is invisible in the current test data and in CI, and it will not be visible in the MVP either.
It is filed because the access pattern is now fixed by two shipped endpoints, and because the DDL is
the spec §3.2 text verbatim (AGENTS.md requires `0001_init` to match it), so the index has to arrive
as a **new migration** rather than an edit — which is exactly the kind of thing that is cheap now and
awkward once there is data.

## Expected output
A `0002_*.up.sql` adding, at minimum:

```sql
CREATE INDEX IF NOT EXISTS idx_exercises_roadmap_day ON exercises (roadmap_id, day_number);
CREATE INDEX IF NOT EXISTS idx_roadmaps_user_active  ON roadmaps (user_id) WHERE is_active;
```

`0001_init.up.sql` stays byte-identical to spec §3.2. CODEMAP's `store` paragraph notes that indexes
beyond the spec DDL live in later migrations, so the "must stay identical" rule and the index are not
read as contradicting each other.

## Evidence
- Plan: `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md` (Task 3 — the two queries).
- `backend/internal/quests/repo.go:63-74`.
- `backend/internal/store/migrations/0001_init.up.sql:53-68` — no `CREATE INDEX`.
- `AGENTS.md` -> *Reading the spec* — `0001_init` must stay identical to the backend spec DDL.
