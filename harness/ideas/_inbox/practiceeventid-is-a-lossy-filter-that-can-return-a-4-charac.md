---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# PracticeEventID is a lossy filter that can return a 4-character id and collide, with no guard and no test

## Why
`PracticeEventID` drops every character outside `[a-v0-9]` from the lower-cased user id and prefixes
`"aelp"`. Its own doc and test assert Google's 5-1024 character bound, but the function neither
enforces nor documents the precondition that makes that bound hold.

Measured in the worktree (temporary probe, removed):

```
in=""                                     out="aelp"                                 len=4
in="wxyz"                                 out="aelp"                                 len=4
in="a-b"                                  out="aelpab"                               len=6
in="ab"                                   out="aelpab"                               len=6
in="a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11" out="aelpa0eebc999c0b4ef8bb6d6bb9bd380a11" len=36
```

So `PracticeEventID("")` == `PracticeEventID("wxyz")` == `"aelp"`, four characters — below the minimum
the function's own doc claims, which Google answers with a 400 that becomes `502 google_unavailable`
and never recovers for that user. And `PracticeEventID("a-b") == PracticeEventID("ab")`.

**Not reachable in production today**: `users.id` is `UUID` (`0001_init.up.sql:12`) and `auth.UserID(c)`
is the JWT subject issued from that column, so every real id is a canonical 36-character UUID, which
maps to a 36-character base32hex id, injective across distinct UUIDs. The exposure is that the
function is exported, takes a bare `string`, and states no precondition, so the first caller who
passes anything but a canonical UUID gets a silently invalid or colliding id. (Even a collision would
not cross users: Calendar event ids are scoped per calendar and each user's event lives on their own
account's `primary`.)

## Expected output
`PracticeEventID` either documents and enforces its precondition (a canonical UUID) or guarantees its
own postcondition regardless of input — e.g. hash the user id and encode the digest in base32hex, so
the result is a fixed length inside 5-1024 and injective for any string. Its test covers the
degenerate inputs, not only one UUID: empty, all-non-base32hex, and a pair that differs only in
dropped characters.

## Evidence
- Plan under review: `harness/plans/2026-09-23-a-failed-savesyncstate-orphans-the-google-object-just-create.md`
- `backend/internal/google/schedule.go:95-112` — `PracticeEventID`; the filter loop has no length guard.
- `backend/internal/google/schedule_test.go:111-131` — `TestPracticeEventIDIsDeterministicBase32Hex` asserts `len(id)` 5-1024 for one UUID and `PracticeEventID("u1") != PracticeEventID("u2")`; no degenerate input is covered.
- `backend/internal/store/migrations/0001_init.up.sql:12` — `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`.
- `backend/internal/google/handler.go:23` — `userID := auth.UserID(c)` (JWT subject).
- Reproduced in `.worktrees/a-failed-savesyncstate-orphans-the-google-object-just-create` with a temporary probe test (deleted; worktree left clean).
