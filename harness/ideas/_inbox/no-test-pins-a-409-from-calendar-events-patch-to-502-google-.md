---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Folded into harness/plans/2026-09-27-google-403-accessnotconfigured-api-disabled-still-maps-to-re.md: closed by its Task 2/3/4 on the same google branch"
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

## Evaluation
_Evaluator, 2026-09-26 — **deferred** (bug cap of 5 reached; status left `proposed`)._ Test-only gap on a merged branch; fold into the `google` plan above.

_Evaluator, 2026-09-27 — daily decide (bug queue)._ **Folded into `harness/plans/2026-09-27-google-403-accessnotconfigured-api-disabled-still-maps-to-re.md` as Task 3.** Confirmed on `origin/main`: `backend/internal/google/handler_test.go:104-114` `TestSyncHandlerMapsAnUnconsumed409To502` drives only `fakeTasks.errs["InsertTaskList"]`; no test makes `fakeCalendar` 409 on `InsertEvent` and again on `PatchEvent` (`service.go:91-103`), and `calendar_test.go` has no PATCH 409 row. Task 3 adds the handler-level test (call log shows exactly one `calendar.InsertEvent` and one `calendar.PatchEvent`, no Tasks call, no `SaveSyncState`; 502 `google_unavailable`) and the client-level `PATCH …/events/taken` 409 → `ErrAlreadyExists` row. Status becomes `rejected` only as the harness's "closed by another plan" marker.
