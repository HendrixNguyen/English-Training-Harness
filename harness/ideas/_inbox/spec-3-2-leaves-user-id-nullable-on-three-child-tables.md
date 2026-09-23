---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# spec 3.2 leaves user_id nullable on three child tables

## Why
Migration 0001 transcribes §3.2 faithfully, which is correct for this slice — but the transcription
surfaces a defect in the spec itself. `push_subscriptions.user_id`, `daily_progress.user_id` and
`roadmaps.user_id` are declared `UUID REFERENCES users(id) ON DELETE CASCADE` with no `NOT NULL`,
while `pet_states.user_id` (corrected by the merged `add-unique-user-id-to-pet-states-ddl` plan) is
`UNIQUE NOT NULL`.

None of the three rows means anything without an owner: a push subscription nobody owns is never
sent to, a roadmap nobody owns is never served, and a day's progress nobody owns is uncountable. Two
concrete consequences:

- `daily_progress` carries `UNIQUE(user_id, date)`. In Postgres, NULLs are distinct in a unique
  index, so any number of rows with `user_id IS NULL` can exist for the same date. The quests slice
  intends to upsert on that constraint; the constraint silently stops protecting anything the moment
  a NULL slips in.
- `ON DELETE CASCADE` cannot clean up a NULL-owner row, so those rows outlive every user forever.

This is a spec correction, so it belongs in migration 0002, not in an edit to 0001 (0001 has been
applied). Same class as the two `pet_states` spec bugs already in the inbox.

## Expected output
Spec §3.2 marks `user_id` as `NOT NULL` on `push_subscriptions`, `daily_progress` and `roadmaps`
(matching the ERD, which shows each as a child of `users`), and a migration 0002 adds
`ALTER TABLE … ALTER COLUMN user_id SET NOT NULL` for the three tables. `exercises.roadmap_id`
deserves the same review. Migration 0001 is left untouched.

## Evidence
- Plan: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
  (*Notes and open questions* — spec corrections land as 0002, never as an edit to 0001).
- Spec §3.2, `1st-thinking-architecture-doc.md` lines 180, 216, 232 — the three nullable `user\_id`
  columns; line 196 is the `NOT NULL` counter-example.
- `backend/internal/store/migrations/0001_init.up.sql:26`, `:46`, `:50` (`UNIQUE(user_id, date)`), `:55`.
- `backend/internal/store/migrations_test.go:59-61` — the test asserts exactly three plain
  `user_id UUID REFERENCES …` columns, i.e. the nullability is currently pinned by a test.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — low.** Spec correction; `UNIQUE(user_id, date)` genuinely stops protecting `daily_progress` if a NULL ever lands. Ships in the `0003` migration plan with the indexes and the one-active-roadmap constraint (backend spec DDL updated in the same change, per AGENTS.md).
