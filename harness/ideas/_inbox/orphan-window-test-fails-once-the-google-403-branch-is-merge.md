---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-27-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn.md
---
# Orphan-window test fails once the google 403 branch is merged: the fake now names the second Google-assigned insert evt_second-2

## Why
Two of today's `google` branches are each green alone and red together. The 403 branch (`harness/2026-09-27-low-google-403-accessnotconfigured-api-disabled-still-maps-to-re`, Task 2) changes `fakeCalendar.InsertEvent` so the n-th Google-assigned insert (n ≥ 2) gets id `"<nextID>-<n>"`. The orphan-window branch's new `TestAFailedSaveAfterTheGoneIDFallbackOrphansTheEventAndIsDocumented` makes two Google-assigned inserts on one fake and, for the second, sets `h.cal.nextID = "evt_second"` and asserts `res.CalendarEventID == "evt_second"`. With both merged the second id is `evt_second-2` and the test fails, so the daily integration branch (and `main`, whichever merges second) goes red on `backend-unit`. The plan asserted the two branches were independent because it did not edit `fakes_test.go`; it depends on that file's behaviour instead. The two branches also conflict textually in the `google` bullet of `harness/CODEMAP.md`.

## Expected output
The orphan-window test is independent of how the fake numbers auto-ids: assert the second sync's `CalendarEventID` is non-empty and differs from the first auto-assigned id (or equals `h.cal.inserted`'s last assigned id as reported by the fake), instead of the literal `"evt_second"`; keep the two-Google-assigned-inserts assertion. Land it on this branch after syncing with (or merging after) the 403 branch, and resolve the CODEMAP `google` bullet so it carries both sentences (the 403 allow-list "Errors:" sentence and the "two documented exceptions" orphan clause). `go test ./internal/google -count=1` green on `origin/main` + both branches.

## Evidence
- Blocked plan: `harness/plans/2026-09-27-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn.md`; interacting plan: `harness/plans/2026-09-27-google-403-accessnotconfigured-api-disabled-still-maps-to-re.md`.
- `backend/internal/google/service_test.go` (orphan branch) `TestAFailedSaveAfterTheGoneIDFallbackOrphansTheEventAndIsDocumented`: `h.cal.nextID = "evt_second"` … `if res.CalendarEventID != "evt_second"`; `backend/internal/google/fakes_test.go` (403 branch) `InsertEvent`: `id = fmt.Sprintf("%s-%d", f.nextID, f.autoInserts)`.
- Reproduction: scratch worktree at `origin/main`, `git merge --no-ff origin/harness/2026-09-27-low-google-403-…` then `git merge --no-ff origin/harness/2026-09-27-low-the-404-on-patch-…` → `CONFLICT (content): Merge conflict in harness/CODEMAP.md`; with CODEMAP set aside, `cd backend && go test ./internal/google -count=1` → `--- FAIL: TestAFailedSaveAfterTheGoneIDFallbackOrphansTheEventAndIsDocumented … service_test.go:255: second sync: res={Status:synced CalendarEventID:evt_second-2 TasksCreatedCount:28}`.
