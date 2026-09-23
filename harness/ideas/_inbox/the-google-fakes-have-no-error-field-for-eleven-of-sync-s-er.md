---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# The google fakes have no error field for eleven of Sync's error branches

## Why
`Service.Sync` has fourteen `if err != nil` returns. Eleven of them cannot be reached by any test,
because the fake that would have to fail has no error field at all. Deleting those lines from
`service.go` leaves the entire suite green — they are, as far as verification goes, untyped
comments.

Unreachable today (`service.go` line → the fake that would need an error hook):
- `:51-54` `repo.Profile` — `fakes_test.go:104-107` never errors.
- `:58-60` `repo.SyncState` non-`ErrNoSyncState` error — `fakes_test.go:122-128` returns only the sentinel or nil.
- `:65-67` `PracticeEvent` error (a malformed `notification_time`) — no test sets an invalid `Profile.NotificationTime` through the service.
- `:74-76` `PatchEvent` non-`ErrNotFound` error — `fakeCalendar.patchErr` is only ever `ErrNotFound`.
- `:79-82` `InsertEvent` error — `fakeCalendar` (`fakes_test.go:38-50`) has **no** error field.
- `:85-87`, `:112-114`, `:134-136` all three `SaveSyncState` errors — `fakes_test.go:130-135` always returns nil.
- `:94-96` `ActiveRoadmap` non-sentinel error — `fakes_test.go:109-115` has no generic error field.
- `:102-104` `DeleteTaskList` error — `fakes_test.go:78-82` always returns nil.
- `:107-110` `InsertTaskList` error — `fakeTasks` has `insertErr`/`failAfter` for *tasks* but nothing for the *list*.
- `:116-119` `DayTitles` error — `fakes_test.go:117-120` always returns nil.

This is not an abstract coverage number. The three `SaveSyncState` branches are precisely the
partial-failure window that orphans Google objects (see the sibling bug *A failed SaveSyncState
orphans the Google object just created*) — the behaviour the plan's *Architecture* paragraph makes
its strongest claim about is the behaviour no test can reach. `handler.go:41`'s
`500 internal_error` branch and the `errors.Is(err, context.DeadlineExceeded)` half of
`handler.go:38` are likewise never exercised (`handler_test.go` covers 200, 409, 502 and 401 only).
`repo.go` and `token.go` error branches are reachable only through the `TEST_DATABASE_URL`-gated
integration test, which asserts only the happy-path upsert.

## Expected output
`fakeCalendar`, `fakeTasks` and `fakeRepo` each gain an error hook for every method (the existing
`patchErr` / `insertErr` / `failAfter` pattern generalised — e.g. a `errs map[string]error` keyed by
call name), and tests cover at minimum: `SaveSyncState` failing after the event insert, after the
list insert, and after the task loop; `InsertEvent` failing; `InsertTaskList` failing;
`DeleteTaskList` failing with a non-404; `tokens.RefreshToken` returning a *generic* error and being
propagated as-is rather than reclassified as reauth. `handler_test.go` gains the plain-error → 500
case and a `context.DeadlineExceeded` → 502 case.

## Evidence
- Plan: `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md` (reviewed 2026-09-23), *Architecture*: "State is persisted after the event step … so a crash mid-sync never orphans a Google object."
- `backend/internal/google/service.go:51-136` — the error returns enumerated above.
- `backend/internal/google/fakes_test.go:38-50, 62-90, 100-135` — the fakes and their (absent) error fields.
- `backend/internal/google/handler_test.go` — four cases; no 500, no deadline.
- Same class as the merged-auth finding `service-and-require-failure-paths-are-untested-the-fakes-err.md`, filed against a different package.
