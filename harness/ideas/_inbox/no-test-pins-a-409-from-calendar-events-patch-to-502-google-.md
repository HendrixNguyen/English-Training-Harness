---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# No test pins a 409 from Calendar events.patch to 502 google_unavailable

## Why
The second folded idea (`every-google-409-now-becomes-500-internal-error-instead-of-5.md`) required "a client test pins a 409 from `events.patch` and a 409 from `tasks` to the chosen status". The branch adds only `TestSyncHandlerMapsAnUnconsumed409To502`, which injects a fake `ErrAlreadyExists` on `InsertTaskList`. Nothing drives the real path the idea named: `Service` inserts the event, gets 409, patches, and the **patch** itself returns 409 (a concurrent sync). That path goes through `service.go:92-100` and reaches the handler's new `errors.Is(err, ErrAlreadyExists)` case only by construction; no test proves it answers 502 and that the insert→patch consumption still works when the patch fails. A future refactor of the service's 409 handling could silently regress it to 500 or loop.

## Expected output
A service- or handler-level test where `fakeCalendar` returns `ErrAlreadyExists` on `InsertEvent` **and** on `PatchEvent`, asserting the route answers `502 {"error":"google_unavailable"}` and the call log shows exactly one insert and one patch. Optionally a `calendar_test.go` case pinning `PATCH …/events/x` 409 to `ErrAlreadyExists` at the client layer.

## Evidence
- Plan: `harness/plans/2026-09-24-every-google-403-becomes-409-reauth-required-so-a-quota-erro.md` (Task 3)
- `backend/internal/google/handler_test.go` on the plan branch: `TestSyncHandlerMapsAnUnconsumed409To502` (tasks fake only)
- `backend/internal/google/service.go:92-100` (insert 409 → patch)
- Idea: `harness/ideas/_inbox/every-google-409-now-becomes-500-internal-error-instead-of-5.md` *Expected output*
