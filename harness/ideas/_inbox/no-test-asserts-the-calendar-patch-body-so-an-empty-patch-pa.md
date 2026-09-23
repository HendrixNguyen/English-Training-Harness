---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# No test asserts the Calendar PATCH body, so an empty patch passes the whole suite

## Why
Patching the recurring Calendar event is *the entire work* of the common re-sync: when the roadmap
has not changed, `Sync` makes exactly one outbound call, `PatchEvent`, and returns
(`service.go:68-99`). Nothing in the suite asserts what that PATCH carries.

Proven by mutation. Change `calendar.go:47` to send no body —

```go
return doJSON(ctx, c.HTTPClient, "calendar", http.MethodPatch,
    c.BaseURL+"/calendars/primary/events/"+url.PathEscape(eventID), accessToken, nil, nil)
```

— so that a re-sync silently stops updating the event's start time, timezone and recurrence, and:

```
$ go test ./internal/google/... -count=1 -timeout 60s
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/google	0.598s
```

The whole package still passes. (Mutation applied and reverted during review; `git status --short`
clean.)

Two independent holes let it through:
- `TestCalendarPatchEventUsesTheStoredID` (`calendar_test.go:78-92`) asserts only `Method` and
  `Path`. The `fakeGoogleAPI` helper *does* capture the decoded body (`calendar_test.go:19-43`) and
  the insert test uses it well (`calendar_test.go:69-76` checks `recurrence`, `start.timeZone`,
  `start.dateTime`) — the patch test just never looks.
- `fakeCalendar.PatchEvent` throws the `Event` away: its signature is
  `func (f *fakeCalendar) PatchEvent(_ context.Context, tok, id string, _ Event) error`
  (`fakes_test.go:52`), so no service-level test can check that the *right* event — built from the
  current `notification_time` and `timezone` — is the one being patched either. `InsertEvent` by
  contrast records the full `Event` (`fakes_test.go:46-50`) and `service_test.go:23` uses it.

So the branch's headline idempotency claim ("a re-sync **patches** the stored event") is verified
only as far as "a PATCH was issued to that id".

## Expected output
`TestCalendarPatchEventUsesTheStoredID` asserts the PATCH body the same way the insert test does —
`summary`, `start.dateTime`, `start.timeZone`, `end`, `recurrence` — so dropping the payload fails
the suite. `fakeCalendar` records the patched `Event` (e.g. `patched []struct{ID string; Ev Event}`)
and `TestResyncSameRoadmapPatchesEventAndCreatesNothing` asserts the patched event's `Start` matches
the profile's notification time, so a regression that patches with a stale or default event is
caught at the service layer too.

## Evidence
- Plan: `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md` (reviewed 2026-09-23); the plan's *Verification* claims the client tests cover the wire shape.
- `backend/internal/google/calendar_test.go:78-92` — patch test asserts `Method`/`Path` only.
- `backend/internal/google/calendar_test.go:69-76` — the insert test that shows the assertion style already exists.
- `backend/internal/google/fakes_test.go:52` — `_ Event`, the discarded patch body.
- `backend/internal/google/service_test.go:96-118` — the same-roadmap re-sync test, which checks `h.cal.patched[0] == "evt_old"` and nothing about content.
- Mutation run recorded above: `calendar.go:47` payload dropped → full package still `ok`.
