---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---

# quests reads the users table directly for the timezone

## Why
`internal/quests` runs `SELECT COALESCE(timezone, 'UTC') FROM users WHERE id = $1`
(`backend/internal/quests/repo.go:61`, `PgRepo.Profile`). `users` is the `auth` slice's table —
CODEMAP records `auth` as its only writer ("upserts `users` on `google_id`") — and AGENTS.md /
`.agents/roles/reviewer.md` state the rule as "packages talk through interfaces; no cross-package
table access". The idea's *Expected output* lists this slice's tables explicitly as `roadmaps`,
`exercises`, `daily_progress`; `users` is not among them.

The abstraction is half-built: `QuestRepo.Profile` is an interface and the tests are pure, so the
*Go* dependency is clean — but the concrete `PgRepo` still reaches into another package's table, so
the boundary the interface was meant to create does not exist at the SQL layer. Nothing today
detects it, and the next slice that needs a `users` column will copy the pattern. The practical cost
arrives when `users.timezone` changes shape (it is `VARCHAR(50) DEFAULT 'UTC'`, nullable, and
onboarding will start writing it): the change has to be found in two packages instead of one, with
no compiler or test to point at the second.

This is a design finding, not a runtime defect — the query is correct and the `COALESCE` handles the
nullable column properly.

## Expected output
`quests` obtains the timezone through the owning package rather than the table: `auth` (or a small
shared profile package) exports something like `Profiles.Timezone(ctx, userID) (string, error)`, and
`quests.QuestRepo` drops `Profile` in favour of that collaborator — kept as an interface so the
existing pure tests are unchanged. Whichever package ends up owning `users`, exactly one package
contains SQL against it, and CODEMAP says which.

If the owner would rather accept the direct read for the MVP, CODEMAP's `quests` paragraph should
say so explicitly ("reads `users.timezone` directly — accepted") so it is a recorded exception
rather than an undetected drift.

## Evidence
- Plan: `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md` (Task 3).
- `backend/internal/quests/repo.go:61,97-103`.
- `harness/CODEMAP.md` — `auth` paragraph (sole writer of `users`); intro rule "packages talk through interfaces".
- `harness/ideas/2026-09-22-run-02/quests-daily-quest-suite-and-progress-recording.md` — *Expected output*, "Tables: `roadmaps`, `exercises` …, `daily_progress`".
