---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Folded into harness/plans/2026-09-27-google-403-accessnotconfigured-api-disabled-still-maps-to-re.md: closed by its Task 2/3/4 on the same google branch"
---
# fakeCalendar returns the same nextID for every Google-assigned insert so a second auto-id insert is a false 409

## Why
`fakeCalendar.InsertEvent` returns `f.nextID` whenever the event carries no client id, and now also
records every returned id in `f.known` and 409s on a repeat. Those two behaviours contradict each
other: the real Calendar API assigns a *fresh* id to every auto-id insert, so two consecutive
Google-assigned inserts can never 409. In the fake they always do.

This makes the plan's own mutation check report the wrong symptom. Setting `ev.ID = ""` in
`service.go` (the plan's Verification step 3) should reproduce the original bug — a second event on
the calendar. Re-run in review, the fake instead 409s on the repeated `nextID`, so `len(h.cal.inserted)`
stays 1 and the assertion that fires is the patch-id one:

```
--- FAIL: TestAFailedSaveAfterTheEventInsertDoesNotCreateASecondEvent (0.00s)
    service_test.go:202: retry should patch aelpu1 once, got [{ID: Ev:{...}}]
```

not the `"retry inserted a second event"` message the plan predicted. The test is not hollow — it does
fail, and the executor reported the real message rather than the predicted one — but the fake's id
model is wrong in a way that will mislead the next person who writes a two-insert test against it.

## Expected output
`fakeCalendar` assigns a distinct id per Google-assigned insert (e.g. `nextID` as a prefix plus a
counter, or a `nextIDs []string` queue), so `known` only 409s on a genuinely repeated *client-supplied*
id, as the real API does. With that, dropping the deterministic id makes
`TestAFailedSaveAfterTheEventInsertDoesNotCreateASecondEvent` fail on `len(h.cal.inserted) != 1`
— the actual user-visible symptom the test claims to guard.

## Evidence
- Plan under review: `harness/plans/2026-09-23-a-failed-savesyncstate-orphans-the-google-object-just-create.md`
- `backend/internal/google/fakes_test.go:65-82` — `InsertEvent` falls back to `f.nextID` and then checks `f.known[id]`.
- `backend/internal/google/service_test.go:182-204` — `TestAFailedSaveAfterTheEventInsertDoesNotCreateASecondEvent`; the `len(h.cal.inserted) != 1` guard cannot fire under the plan's mutation.
- Plan Verification step 3 predicts a failure message the mutation does not produce.
- Reproduced in the worktree: `sed` `ev.ID = PracticeEventID(userID)` -> `ev.ID = ""`, `go test ./internal/google/... -run 'FailedSaveAfterTheEventInsert' -count=1`. Restored; `git diff --quiet internal/google/service.go` clean.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low. Not planned today; plan with the other google test-double findings (`practiceeventid-is-a-lossy-filter-…`, `the-404-on-patch-fallback-…`) and the 409 mapping — one google branch.**

*Confirmed from the idea's evidence (mutation re-run by the reviewer; not re-run here).* `backend/internal/google/fakes_test.go:65-82` returns one `nextID` for every Google-assigned insert and 409s on a repeat, which the real API never does. *Fix.* A per-insert counter (`nextID` + `-N`) or a `nextIDs` queue, so `known` 409s only on a repeated client-supplied id, and the plan's mutation check then fails on `len(h.cal.inserted) != 1` as predicted. Low: test-double fidelity; the guarded behaviour is correct.

_Evaluator, 2026-09-27 — daily decide (bug queue)._ **Folded into `harness/plans/2026-09-27-google-403-accessnotconfigured-api-disabled-still-maps-to-re.md` as Task 2.** Confirmed on `origin/main` `backend/internal/google/fakes_test.go:63-81`: `InsertEvent` returns `f.nextID` for every Google-assigned insert and then 409s on `f.known[id]`, so a second auto-id insert is a false 409 (the real API never repeats an assigned id). Task 2 gives the fake a per-insert counter — the first auto-id insert still returns `nextID`, so `TestAReservedButGoneIDFallsBackToAGoogleAssignedInsert` keeps its `evt_fresh` assertion; the n-th returns `<nextID>-<n>` — pins it with a fake-level test, and re-runs the earlier plan's `ev.ID = ""` mutation so `TestAFailedSaveAfterTheEventInsertDoesNotCreateASecondEvent` fails on `len(h.cal.inserted) != 1`, the symptom it guards. Status becomes `rejected` only as the harness's "closed by another plan" marker.
