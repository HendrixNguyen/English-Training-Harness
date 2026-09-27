---
plan: harness/plans/2026-09-27-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/orphan-window-test-fails-once-the-google-403-branch-is-merge.md]
---
# Review — The 404-on-patch fallback re-opens the orphan window and Sync's new doc says the Calendar half is covered

**Plan:** `harness/plans/2026-09-27-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn.md`
**Branch/worktree:** `harness/2026-09-27-low-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn` / `.worktrees/2026-09-27-low-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn`
**Diff:** `git diff main...harness/2026-09-27-low-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn --stat`

## Plan vs idea
Delivered, taking the idea's "the docs say so" option. `Sync`'s doc now says the Calendar half is not covered after the 404/410 fallback. So does the inline comment at the fallback, and so does the CODEMAP `google` bullet ("two documented exceptions"). A test injects `failSaveAt` on the sync that goes through the fallback and asserts the documented outcome: two Google-assigned inserts, error on the first sync. Choosing documentation over a migration-backed generation counter is proportionate for a low bug.

## Code vs plan
Reviewed at head `44df75a` (detached scratch worktree; 23 behind main). CI run 36296809833 `success`.
- Task 1 (test): followed, placed after `TestAReservedButGoneIDFallsBackToAGoogleAssignedInsert`. It pins existing behaviour. The executor's mutation (a deterministic id in the fallback) makes it fail.
- Task 2 (service.go docs): followed. Comments only, no behaviour change.
- Task 3 (CODEMAP): followed.
- `fakes_test.go` untouched, as scoped.

Verification (reviewer, `backend/`):
```
go build ./... ; gofmt -l . ; go vet ./...      -> clean
env -u TEST_DATABASE_URL go test ./... -count=1 -> all packages ok
go test ./internal/google/ -count=1 -v -run 'GoneIDFallback|ReservedButGone|FailedSave'
  --- PASS: TestAFailedSaveAfterTheEventInsertDoesNotCreateASecondEvent
  --- PASS: TestAReservedButGoneIDFallsBackToAGoogleAssignedInsert
  --- PASS: TestAFailedSaveAfterTheGoneIDFallbackOrphansTheEventAndIsDocumented
  --- PASS: TestAFailedSaveAfterTheListInsertOrphansTheListAndIsDocumented
  ok
```

**Merge interaction (blocker):** the plan claims the branch is independent of today's google-403 branch because it doesn't edit `fakes_test.go`. That is wrong: the test depends on how the fake assigns ids, and the 403 branch changes exactly that. Scratch worktree at `origin/main`, merge the 403 branch, then merge this one:
```
CONFLICT (content): Merge conflict in harness/CODEMAP.md          (google bullet)
go test ./internal/google -count=1
--- FAIL: TestAFailedSaveAfterTheGoneIDFallbackOrphansTheEventAndIsDocumented
    service_test.go:255: second sync: res={Status:synced CalendarEventID:evt_second-2 TasksCreatedCount:28}
```
Whichever of the two lands second turns `backend-unit` red on the daily branch and on `main`.

## Quality
- Docs: accurate. "Once per failed attempt" matches the code path (the state stays empty, so every retry goes 409 → 404 → a new insert).
- Test honesty: the asserted outcome is right, but the literal `"evt_second"` couples the test to the fake's id numbering. That coupling is what breaks under the 403 branch; see the blocker.
- Nit: the docs and comments say "410 fallback", but the fallback triggers on `ErrNotFound` (404 *or* 410), and the idea calls it the 404-on-patch fallback. Not filed separately; the blocker's fix can reword it.

## Bugs filed
- `harness/ideas/_inbox/orphan-window-test-fails-once-the-google-403-branch-is-merge.md` — **high, blocker** (`blocks` this plan): the test fails with the google-403 branch merged, and CODEMAP conflicts.

## Verdict
pass-with-bugs — the idea is delivered and the branch is green alone. The branch is **held by a blocker**: do not put it in today's daily PR alongside the google-403 branch. The amend makes the test independent of the fake's id numbering and resolves the CODEMAP bullet.
