---
idea: harness/ideas/_inbox/the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn.md
status: done
priority: low
merged: false
branch: harness/2026-09-27-low-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn
worktree: .worktrees/2026-09-27-low-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn
---
# The 404-on-patch fallback re-opens the orphan window and Sync's new doc says the Calendar half is covered — Plan

**Idea:** `harness/ideas/_inbox/the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn.md`
**Goal:** The docs stop claiming the Calendar half is fully covered by the deterministic id, and a test pins what actually happens. The documentation option is chosen, not the idempotent-fallback option.

**Root cause:** In `backend/internal/google/service.go` `Sync`, the path runs: insert the deterministic id → 409 `ErrAlreadyExists` → `PatchEvent(ev.ID)` → `ErrNotFound`. At that point it inserts with a **Google-assigned** id (`ev.ID = ""`), and `state.CalendarEventID` is only persisted by the `SaveSyncState` that follows. If that save fails, the Google-assigned event is orphaned. The next sync repeats the same 409 → 404 → insert chain and orphans another event. The function doc ends with "The Calendar half is covered by the deterministic id", and the CODEMAP google bullet says "a failure never orphans a Google object". Both are false on this branch.

**Decision:** document, do not re-engineer. The branch needs a user to delete the event by hand **and** Google to report the reserved id as gone on PATCH, **and** a failed Postgres write in the same request. The idempotent alternative (a generation counter persisted write-ahead) needs a migration on `google_sync`, which is disproportionate for a low bug. The test pins the documented outcome so that a future fix has to change it deliberately.

**Scope / files:** `backend/internal/google/service.go` (doc comments only, no behaviour change), `backend/internal/google/service_test.go` (one new test), `harness/CODEMAP.md` (google bullet). The test reuses the existing fakes as-is (`fakeCalendar.known`, `patchErr`, `nextID`; `fakeRepo.failSaveAt`), so `fakes_test.go` is **not** edited; today's unmerged `…403-accessnotconfigured…` branch owns it. That branch does not touch `service.go` or `service_test.go`.

## Tasks

### Task 1: Pin the documented outcome (service_test.go)
Files: `backend/internal/google/service_test.go`. Add it after `TestAReservedButGoneIDFallsBackToAGoogleAssignedInsert`.

`TestAFailedSaveAfterTheGoneIDFallbackOrphansTheEventAndIsDocumented`:
1. `h := newHarness()`, `h.cal.known = map[string]bool{PracticeEventID("u1"): true}`, `h.cal.patchErr = ErrNotFound`, `h.cal.nextID = "evt_first"`, `h.repo.failSaveAt = 1`.
2. First `Sync` → `errors.Is(err, errSaveBoom)`, `len(h.cal.inserted) == 1` with `h.cal.inserted[0].ID == ""`, and `h.repo.state.CalendarEventID == ""`. `fakeCalendar.InsertEvent` returns before appending on a 409, so the deterministic-id attempt is not counted. The event exists at "Google" but nothing stored its id.
3. `h.repo.failSaveAt = 0`, `h.cal.nextID = "evt_second"`. Second `Sync` → no error, and `res.CalendarEventID == "evt_second"`.
4. Assert that two Google-assigned events now exist (`evt_first` orphaned, `evt_second` stored): count `h.cal.inserted` entries with `ID == ""` == 2.
5. Add a comment in the test: "the Tasks-half analogue is TestAFailedSaveAfterTheListInsertOrphansTheListAndIsDocumented; this branch is not covered by the deterministic id".
6. Run `cd backend && go test ./internal/google/ -run GoneIDFallback -count=1 -v`. Expect PASS on current code: this test pins existing behaviour, so it passes first. State that in the commit body.

### Task 2: Correct the docs (service.go)
Files: `backend/internal/google/service.go`.
1. `Sync` doc: replace "The Calendar half is covered by the deterministic id." with two sentences. The Calendar half is covered by the deterministic id except after the 410 fallback: when a user deleted the event and Google has released the id, the event is re-inserted with a Google-assigned id, and a `SaveSyncState` failure right after that orphans it, once per failed attempt.
2. The inline comment at the fallback (`// The id is reserved but the event is gone for good (410)…`) gains: "— a save failure after this insert orphans it; see Sync's doc and TestAFailedSaveAfterTheGoneIDFallbackOrphansTheEventAndIsDocumented."
3. Run `gofmt -l internal/google` (expect empty output) and `go test ./internal/google/ -count=1`.
4. Commit: `google: document that the 410 fallback insert can orphan an event on a failed save`.

### Task 3: CODEMAP
Files: `harness/CODEMAP.md`, google bullet. Replace "so a failure never orphans a Google object" with "so a failure never orphans a Google object, with two documented exceptions: a failed save right after `tasklists.insert` orphans an empty list, and one right after the 410 fallback insert (Google-assigned id) orphans that event". Commit: `harness: CODEMAP — google's two documented orphan windows`.

## Verification
```bash
cd backend
go test ./internal/google/ -count=1 -v -run 'GoneIDFallback|ReservedButGone|FailedSave'
go test ./... -count=1
gofmt -l . && go vet ./...
grep -n 'Google-assigned' internal/google/service.go ../harness/CODEMAP.md
```
- The new test passes and asserts two Google-assigned inserts and the error on the first sync.
- No sentence in `service.go` or CODEMAP still claims the Calendar half is unconditionally covered (`grep -n 'Calendar half is covered' internal/google/service.go` shows only the qualified sentence).
- `git diff origin/main -- internal/google/fakes_test.go` is empty.
- CI is green on the pushed branch.

## Execution summary

Branch `harness/2026-09-27-low-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn`, head `44df75a` (commits `347b6d0` test, `6658d6e` service.go docs, `44df75a` CODEMAP). All three tasks done as written; no behaviour change.

- **Task 1:** `TestAFailedSaveAfterTheGoneIDFallbackOrphansTheEventAndIsDocumented` added after `TestAReservedButGoneIDFallsBackToAGoogleAssignedInsert`. It passes on current code (it pins existing behaviour; stated in the commit body). Mutation check: changing the fallback's `ev.ID = ""` to a deterministic id makes it FAIL (`after first sync: inserted=[{… ID:aelpu1x}]`); reverted.
- **Task 2:** `Sync` doc now says the Calendar half is covered by the deterministic id *except after the 410 fallback*; the fallback inline comment points to Sync's doc and the new test.
- **Task 3:** CODEMAP google bullet names both documented orphan windows.

Verification (real output, abridged):
```
--- PASS: TestAFailedSaveAfterTheEventInsertDoesNotCreateASecondEvent
--- PASS: TestAReservedButGoneIDFallsBackToAGoogleAssignedInsert
--- PASS: TestAFailedSaveAfterTheGoneIDFallbackOrphansTheEventAndIsDocumented
--- PASS: TestAFailedSaveAfterTheListInsertOrphansTheListAndIsDocumented
go test ./... -count=1   -> ok for all 14 packages
gofmt -l .               -> (empty);  go vet ./... -> ok;  go build ./... -> ok
grep 'Google-assigned'   -> service.go:45, service.go:99, CODEMAP.md:18
grep 'Calendar half'     -> only service.go:42, the qualified sentence
git diff origin/main -- internal/google/fakes_test.go -> empty
```
CI: https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36296809833 — success (backend-unit, backend-integration, frontend, docker-images, harness-tooling).

Deviations: none in code. The app was not booted with compose: the change is comments plus one unit test, so no runtime path changed. The build and the CI docker-images/backend-integration jobs cover it. Nothing was left running. Only `origin/main` was used as base; `fakes_test.go` and `token*.go` untouched.
