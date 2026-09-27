---
plan: harness/plans/2026-09-27-google-403-accessnotconfigured-api-disabled-still-maps-to-re.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/google-403-allow-list-refactor-dropped-the-pins-for-userrate.md]
---
# Review — Google 403 accessNotConfigured (API disabled) still maps to reauth_required, looping users through re-consent

**Plan:** `harness/plans/2026-09-27-google-403-accessnotconfigured-api-disabled-still-maps-to-re.md`
**Branch/worktree:** `harness/2026-09-27-low-google-403-accessnotconfigured-api-disabled-still-maps-to-re` / `.worktrees/google-403-accessnotconfigured-api-disabled-still-maps-to-re`
**Diff:** `git diff main...harness/2026-09-27-low-google-403-accessnotconfigured-api-disabled-still-maps-to-re --stat`

## Plan vs idea
Delivered. The idea's Expected output — `accessNotConfigured` and the `PERMISSION_DENIED` + `SERVICE_DISABLED` details form → `*UpstreamError` → 502; empty/non-JSON 403, `insufficientPermissions`, `forbidden` still reauth; a `calendar_test.go` row pinning `accessNotConfigured`; the CODEMAP error sentence — all exist. The plan's allow-list (instead of a longer deny-list) is the better design and is justified in the plan. The three folded bugs (fakeCalendar fresh ids, patch-409 pin, `PracticeEventID` bounds) are delivered too.

## Code vs plan
Reviewed at `origin/harness/2026-09-27-low-google-403-…` head `1256341` in a detached scratch worktree; base `git merge-base origin/main` (23 commits behind main). CI on head: run 36292025148 `success`.
- Task 1 (allow-list): followed. `reauthReasons` / `googleErrorReasons` (errors[] then details[]) / `isReauth403`; `throttleReasons` gone. Red check: with main's `client.go` restored, the new rows fail — `apioff`, `svcdisabled`, `unknownreason` (`want *UpstreamError{calendar, 403}`) and `tasks_test.go:107` — then pass on the branch.
- Task 2 (fake fresh ids): followed. Mutation `ev.ID = ""` in `service.go` → `TestAFailedSaveAfterTheEventInsertDoesNotCreateASecondEvent` fails with `retry inserted a second event` (reproduced); restored clean.
- Task 3 (insert 409 → patch 409 → 502): followed; regression pin, passes on first run as the plan said.
- Task 4 (`PracticeEventID`): followed. UUID path byte-identical (`aelpa0eebc999c0b4ef8bb6d6bb9bd380a11` pinned); non-UUID input hashed to 36 chars.
- Safe-file set respected: diff touches only `client.go`, `schedule.go`, five `_test.go` files and one CODEMAP sentence.

Verification (reviewer, `backend/`):
```
go build ./... ; gofmt -l internal/google ; go vet ./internal/google      -> clean
env -u TEST_DATABASE_URL -u TEST_REDIS_URL go test -timeout 300s ./... -count=1
  ok for all 13 packages (cmd/api airouter auth config google health middleware notify onboarding pet quests secrets store)
go test -timeout 120s ./internal/google -count=1 -race -v | grep -E '^(--- (FAIL|SKIP)|ok|FAIL)'
  --- SKIP: TestIntegrationSyncStateIsOneRowPerUser (0.00s)
  ok  .../internal/google 4.521s
```

## Quality
- Correctness: a real insufficient-scope response carries both `insufficientPermissions` and `ACCESS_TOKEN_SCOPE_INSUFFICIENT` → reauth; `RESOURCE_EXHAUSTED`/`RATE_LIMIT_EXCEEDED` details → upstream. The "any parsed reason is reauth → reauth" rule is the right bias given scopes are fixed.
- Test gap (filed, low): `userRateLimitExceeded` / `quotaExceeded` were explicit in `throttleReasons` and are now unpinned; the no-reason reauth message carries a trailing space.
- **Cross-branch interaction (orchestrator):** this branch's Task 2 fake change breaks the orphan-window branch's new test when both are merged (`evt_second-2`), and the two conflict in CODEMAP's `google` bullet. Filed as a blocker on the *orphan-window* plan (`harness/ideas/_inbox/orphan-window-test-fails-once-the-google-403-branch-is-merge.md`); this branch is fine to merge on its own. Verified: `origin/main` + this branch + the integration-gate branch merge clean and `go test ./internal/google ./internal/store` is ok.
- CODEMAP: the "Errors:" sentence matches the code.

## Bugs filed
- `harness/ideas/_inbox/google-403-allow-list-refactor-dropped-the-pins-for-userrate.md` — low: two quota reasons unpinned; trailing space in the no-reason reauth message.

## Verdict
pass-with-bugs — delivered and verified; one low test-gap bug. Mergeable; do not merge together with the orphan-window branch until its blocker is fixed.
