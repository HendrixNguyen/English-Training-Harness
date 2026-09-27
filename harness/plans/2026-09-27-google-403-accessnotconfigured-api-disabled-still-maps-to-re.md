---
idea: harness/ideas/_inbox/google-403-accessnotconfigured-api-disabled-still-maps-to-re.md
status: approved
priority: low
merged: false
---
# Google 403 accessNotConfigured (API disabled) still maps to reauth_required, looping users through re-consent — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Bug team — ticket **B2** of 2026-09-27. **Estimate:** 3.5 h. **Branch:** `harness/2026-09-27-low-google-403-accessnotconfigured-api-disabled-still-maps-to-re`.

**Idea:** `harness/ideas/_inbox/google-403-accessnotconfigured-api-disabled-still-maps-to-re.md`. **Folds** (each closed as `rejected: Folded into …`): `harness/ideas/_inbox/fakecalendar-returns-the-same-nextid-for-every-google-assign.md` (Task 2), `harness/ideas/_inbox/no-test-pins-a-409-from-calendar-events-patch-to-502-google-.md` (Task 3), `harness/ideas/_inbox/practiceeventid-is-a-lossy-filter-that-can-return-a-4-charac.md` (Task 4). Backend only; **no design doc**.

**Goal:** A Google 403 sends the user back through consent only when re-consenting can fix it (a scope/permission reason, or a 403 with no parseable reason); an API-disabled / configuration 403 answers `502 google_unavailable` like a quota 403 does — and the three `google` test-double gaps the reviewer found on the merged branches are closed on the same branch.

**Root cause (idea, confirmed on `origin/main`):** `backend/internal/google/client.go:113` — `case resp.StatusCode == http.StatusForbidden && !throttleReasons[googleErrorReason(raw)]` → `ErrReauthRequired`. Reauth is the *default* branch of a four-entry deny-list (`client.go:27-32`), so `accessNotConfigured` (Calendar or Tasks API not enabled in the GCP project), `SERVICE_DISABLED`, `domainPolicy`, `forbiddenForServiceAccounts`, … all become `409 reauth_required` (`handler.go:39`) and the PWA loops consent → sync → consent with no exit. `googleErrorReason` (`client.go:36-48`) also reads only `error.errors[0].reason`, so Google's newer `status` + `details[]` (`google.rpc.ErrorInfo`) form is invisible.

**Decision — allow-list, not a longer deny-list.** A 403 is reauth iff (a) no reason can be parsed from the body (empty / HTML 403 — the legacy shape; kept as reauth, as the idea asks) or (b) any parsed reason is a scope/permission reason: `insufficientPermissions`, `forbidden` (`error.errors[].reason`) or `ACCESS_TOKEN_SCOPE_INSUFFICIENT` (`error.details[].reason`). Every other parsed reason → `*UpstreamError` → 502 with the body in the server log. Rationale: a wrong 409 is unrecoverable for the user (consent loop) while a wrong 502 is "try later" with the reason already in `logSyncFailure`; the scopes are fixed by `auth.Scopes` and Google documents exactly those reasons for scope problems on Calendar v3 / Tasks v1, so the allow-list is short and stable, whereas the set of non-auth 403 reasons is open-ended. `throttleReasons` is deleted (subsumed). The handler is **not** changed: `*UpstreamError` already maps to 502 (`handler.go:44`).

**Merge-safety (owner context, 2026-09-27):** 21 `done` plans are unmerged and their branches edit most of the tree. This plan touches **only** `backend/internal/google/{client,schedule}.go`, the `_test.go` files listed below and one sentence of the `google` bullet in `harness/CODEMAP.md`. It must not touch `backend/internal/google/token.go`, `token_test.go`, `integration_test.go` (edited by the unmerged refresh-token branch), `backend/cmd/api/`, `backend/internal/config/`, `backend/internal/secrets/`, or anything under `project-base/`. No migration. A spec sentence that turns out to need a change goes in the Execution summary's *Follow-ups*, not in a commit.

## Global Constraints
- Work in `.worktrees/<slug>` on the branch above, cut from fresh `origin/main`; run Go from `backend/`; `rg` / `timeout` are not installed — use `grep -n`, `go test -timeout`.
- Unit tests only: `go test ./internal/google -count=1`. The package's `TestIntegration*` is `TEST_DATABASE_URL`-gated and skips locally; nothing here needs Docker.
- **`go test ./...` never calls Google** — every new case runs against `fakeGoogleAPI` (httptest) or the `fakes_test.go` doubles.
- Keep existing test names and assertions unless a step says otherwise; the executor must not weaken `TestCalendarMapsStatusesToSentinelErrors` or `TestTasksMapsAQuota403ToUpstreamNotReauth`, they are extended.
- `gofmt -l internal/google` empty; `go vet ./internal/google` clean; one commit per task.
- If the handler, `service.go`'s error flow, or any file outside the safe set seems to need editing, stop and report instead of editing.

## Review Focus
1. The 403 table in Task 1 Step 1 holds exactly: `accessNotConfigured`, `SERVICE_DISABLED`-only, `domainPolicy` → `*UpstreamError{Status: 403}` carrying Google's raw body; message-only 403, HTML 403, `insufficientPermissions`, `forbidden`, `ACCESS_TOKEN_SCOPE_INSUFFICIENT`-only → `ErrReauthRequired`; `rateLimitExceeded` / `dailyLimitExceeded` / `userRateLimitExceeded` still → upstream. Same on the Tasks client.
2. `fakeCalendar` never returns the same id for two Google-assigned inserts, and still 409s a repeated client-supplied id (including one Google assigned earlier). The `ev.ID = ""` mutation (Task 2 Step 3) now fails `TestAFailedSaveAfterTheEventInsertDoesNotCreateASecondEvent` on `retry inserted a second event`.
3. Insert 409 → patch 409 reaches the handler as 502 `google_unavailable`, with exactly one `calendar.InsertEvent` and one `calendar.PatchEvent` in the call log, no Tasks call and no `SaveSyncState`.
4. `PracticeEventID` of the pinned UUID is byte-for-byte unchanged (`aelpa0eebc999c0b4ef8bb6d6bb9bd380a11`) — live `google_sync.calendar_event_id` rows and the retry's 409-consumption depend on it — and every other input yields a 36-character base32hex id, distinct across the test table.
5. `git diff --stat origin/main` lists only the safe files (Verification).

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/google/client.go` | `reauthReasons` replaces `throttleReasons`; `googleErrorReasons(raw) []string` (errors[] + details[] reasons) replaces `googleErrorReason`; `isReauth403(raw) bool`; doc comments on `ErrReauthRequired` and `doJSON` |
| `backend/internal/google/calendar_test.go` | 403 rows (Task 1), PATCH 409 row (Task 3) |
| `backend/internal/google/tasks_test.go` | `TestTasksMapsAnAPIDisabled403ToUpstreamNotReauth` |
| `backend/internal/google/fakes_test.go` | `fakeCalendar.autoInserts` counter + `TestFakeCalendarAssignsAFreshIDPerGoogleAssignedInsert` |
| `backend/internal/google/handler_test.go` | `TestSyncHandlerMapsA409OnThePatchAfterA409InsertTo502` |
| `backend/internal/google/schedule.go`, `schedule_test.go` | `PracticeEventID` total for every input |
| `harness/CODEMAP.md` | the one "Errors:" sentence of the `google` bullet |

## Tasks

### Task 1: A 403 is reauth only for a scope/permission reason (allow-list)

**Files:** `backend/internal/google/client.go`, `calendar_test.go`, `tasks_test.go`, `harness/CODEMAP.md`.

- [ ] **Step 1 (tests first):** extend `TestCalendarMapsStatusesToSentinelErrors` (`calendar_test.go:148`) with these `PATCH /calendars/primary/events/<id>` rows and assertions:
  - `apioff` — 403, Google's real API-disabled body: `{"error":{"code":403,"message":"Google Calendar API has not been used in project 123 before or it is disabled. Enable it by visiting https://console.developers.google.com/apis/api/calendar-json.googleapis.com/overview?project=123 then retry.","errors":[{"message":"Google Calendar API has not been used in project 123 before or it is disabled.","domain":"usageLimits","reason":"accessNotConfigured","extendedHelp":"https://console.developers.google.com"}],"status":"PERMISSION_DENIED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"SERVICE_DISABLED","domain":"googleapis.com","metadata":{"consumer":"projects/123","service":"calendar-json.googleapis.com"}}]}}` → `errors.As(err, &up)`, `up.Service == "calendar"`, `up.Status == 403`, `strings.Contains(up.Body, "accessNotConfigured")` (so `logSyncFailure` shows Google's text), and **not** `errors.Is(err, ErrReauthRequired)`.
  - `svcdisabled` — 403 with **no** `errors[]`: `{"error":{"code":403,"message":"Google Tasks API has not been used in project 123 before or it is disabled.","status":"PERMISSION_DENIED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"SERVICE_DISABLED","domain":"googleapis.com"}]}}` → upstream, not reauth.
  - `unknownreason` — 403 `{"error":{"code":403,"errors":[{"reason":"domainPolicy"}]}}` → upstream, not reauth (pins the allow-list: an unlisted reason is never reauth).
  - `scopedetail` — 403 with no `errors[]`: `{"error":{"code":403,"message":"Request had insufficient authentication scopes.","status":"PERMISSION_DENIED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"ACCESS_TOKEN_SCOPE_INSUFFICIENT","domain":"googleapis.com"}]}}` → `ErrReauthRequired`.
  - `forbiddenreason` — 403 `{"error":{"code":403,"errors":[{"domain":"global","reason":"forbidden","message":"Forbidden"}]}}` → `ErrReauthRequired`.
  - Keep the existing `forbidden` (message only), `noscope`, `notjson403` → reauth and `throttled`, `daily`, `toomany`, `broken` → upstream rows exactly as they are.
  In `tasks_test.go` add `TestTasksMapsAnAPIDisabled403ToUpstreamNotReauth` mirroring `TestTasksMapsAQuota403ToUpstreamNotReauth` (`tasks_test.go:80`) on `POST /users/@me/lists` (`InsertTaskList`) with the `apioff` body (`tasks.googleapis.com` in `metadata.service`) → `*UpstreamError{tasks, 403}`, not reauth. Run `go test ./internal/google -run 'Calendar|Tasks' -count=1`: `apioff`, `svcdisabled`, `unknownreason` and the Tasks test fail (they are reauth today); the rest pass.
- [ ] **Step 2:** `client.go` — replace `throttleReasons` (lines 25-32) with
  `var reauthReasons = map[string]bool{"insufficientPermissions": true, "forbidden": true, "ACCESS_TOKEN_SCOPE_INSUFFICIENT": true}` (doc: the only reasons re-consent can fix — `error.errors[].reason` on Calendar v3 / Tasks v1, and the `google.rpc.ErrorInfo` detail reason). Replace `googleErrorReason` with `googleErrorReasons(raw []byte) []string` that decodes `{"error":{"errors":[{"reason"}],"details":[{"reason"}]}}` and returns every non-empty reason in order (errors first, then details); `nil` when the body is not that shape. Add `isReauth403(raw []byte) bool`: `len(reasons) == 0` → true; any reason in `reauthReasons` → true; otherwise false. Change line 113 to `case resp.StatusCode == http.StatusForbidden && isReauth403(raw):` and keep the reasons in the wrapped message (`strings.Join(reasons, ",")`). Update the comment on the final `case` ("Includes every 403 with a known non-auth reason — quota, API disabled, domain policy — and 429: the handler answers 502") and the doc comments on `ErrReauthRequired` (lines 14-22) and `doJSON` (lines 74-78) so they describe the allow-list. No other file changes; `handler.go` stays as is.
- [ ] **Step 3:** `harness/CODEMAP.md`, `google` bullet — replace **only** the sentence that begins `Errors: \`invalid_grant\`, 401, or a 403 whose \`error.errors[].reason\` is **not** a quota reason → 409 \`reauth_required\`; …` with: ``Errors: `invalid_grant`, 401, or a 403 whose reason (`error.errors[].reason` or an `ErrorInfo` `details[].reason`) is a scope/permission reason — `insufficientPermissions`, `forbidden`, `ACCESS_TOKEN_SCOPE_INSUFFICIENT` — or that carries no parseable reason at all → 409 `reauth_required`; a 403 with any other reason (the quota family, `accessNotConfigured` / `SERVICE_DISABLED` when the Calendar or Tasks API is off in the GCP project, `domainPolicy`, …), 429, any other non-2xx, a 409 the Calendar insert did not consume, or the 60 s `SyncTimeout` → 502 `google_unavailable`; anything else → 500 `internal_error`.`` Touch nothing else in that bullet (an unmerged branch edits the same bullet; a one-sentence change keeps the daily merge trivial).
- [ ] **Step 4:** `go test ./internal/google -count=1 -v -run 'Calendar|Tasks|Handler'` green; `gofmt -l internal/google` empty; `grep -n throttleReasons internal/google/*.go` empty. Commit: `google: a 403 is reauth only for a scope/permission reason; API-disabled and other config 403s answer 502`.

### Task 2: `fakeCalendar` assigns a fresh id per Google-assigned insert

**Files:** `backend/internal/google/fakes_test.go` (fake + its test). Folds `fakecalendar-returns-the-same-nextid-…`.

- [ ] **Step 1 (test first):** at the bottom of `fakes_test.go` add `TestFakeCalendarAssignsAFreshIDPerGoogleAssignedInsert`: `f := &fakeCalendar{log: &callLog{}, nextID: "evt_new"}`; two `InsertEvent(ctx, "t", Event{})` calls (empty `ID`) must both succeed and return `"evt_new"` then `"evt_new-2"`, `len(f.inserted) == 2`; a third insert with `Event{ID: "evt_new"}` must return `ErrAlreadyExists` (Google-assigned ids stay reserved, like the real API); a fourth with `Event{ID: "aelpx"}` succeeds and a repeat of it 409s. Run it: the second auto-id insert 409s today → red.
- [ ] **Step 2:** `fakeCalendar` gains `autoInserts int` (doc: "Google-assigned inserts so far; the real API never repeats an assigned id"). In `InsertEvent`, replace `if id == "" { id = f.nextID }` with: increment `autoInserts`; `id = f.nextID` for the first, `fmt.Sprintf("%s-%d", f.nextID, f.autoInserts)` for the n-th (n ≥ 2). Leave the `known` bookkeeping as it is. `TestAReservedButGoneIDFallsBackToAGoogleAssignedInsert` (`service_test.go:211`) keeps asserting `evt_fresh` because the first auto-id insert is unchanged.
- [ ] **Step 3 (mutation check, then restore):** `sed -i '' 's/ev.ID = PracticeEventID(userID)/ev.ID = ""/' internal/google/service.go`, `go test ./internal/google -run FailedSaveAfterTheEventInsert -count=1` must now fail with `retry inserted a second event` (not the patch-id message the earlier plan saw); `git checkout -- internal/google/service.go`; `git diff --quiet internal/google/service.go`.
- [ ] **Step 4:** `go test ./internal/google -count=1` green. Commit: `google: fakeCalendar assigns a fresh id per Google-assigned insert so two auto-id inserts never false-409`.

### Task 3: Pin insert-409 → patch-409 → `502 google_unavailable`

**Files:** `backend/internal/google/handler_test.go`, `calendar_test.go`. Folds `no-test-pins-a-409-from-calendar-events-patch-…`.

- [ ] **Step 1 (handler test):** add `TestSyncHandlerMapsA409OnThePatchAfterA409InsertTo502` after `TestSyncHandlerMapsAnUnconsumed409To502` (`handler_test.go:104`): `h := newHarness()`; `conflict := fmt.Errorf("%w: calendar returned 409", ErrAlreadyExists)`; `h.cal.errs = map[string]error{"InsertEvent": conflict, "PatchEvent": conflict}` (a concurrent sync owns the event); `w := post(t, router(h.svc, "u1"))`; assert `w.Code == http.StatusBadGateway` and body `{"error":"google_unavailable"}`; over `h.log.calls` count entries with prefix `calendar.InsertEvent(` == 1 and `calendar.PatchEvent(` == 1, the patch entry ends with `","+PracticeEventID("u1")+")"` (it patched *our* id), and no entry starts with `tasks.` or `repo.SaveSyncState(`. This passes today — it pins `service.go:91-103` + `handler.go:44` against a refactor; say so in the test comment.
- [ ] **Step 2 (client test):** in `TestCalendarMapsStatusesToSentinelErrors` add `"PATCH /calendars/primary/events/taken": {409, <the dup body from TestCalendarInsertSendsTheClientIDAndMaps409ToAlreadyExists>}` and assert `errors.Is(c.PatchEvent(ctx, "t", "taken", sampleEvent()), ErrAlreadyExists)`.
- [ ] **Step 3:** `go test ./internal/google -run 'Handler|Calendar' -count=1 -v` green. Commit: `google: pin a 409 from events.patch after a 409 insert to 502 google_unavailable`.

### Task 4: `PracticeEventID` keeps Calendar's 5–1024 base32hex bound for every input

**Files:** `backend/internal/google/schedule.go`, `schedule_test.go`. Folds `practiceeventid-is-a-lossy-filter-…`.

- [ ] **Step 1 (tests first):** in `schedule_test.go` add `TestPracticeEventIDHoldsItsBoundsForEveryInput`: table `"" , "wxyz", "a-b", "ab", "u1", "u2", "A0EEBC99-9C0B-4EF8-BB6D-6BB9BD380A11", "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"`; for each: `regexp ^[a-v0-9]+$`, `5 <= len <= 1024`, `PracticeEventID(in) == PracticeEventID(in)`; the two UUID spellings are equal to each other **and** to the literal `aelpa0eebc999c0b4ef8bb6d6bb9bd380a11`; every non-UUID result has `len == 36` and every result is distinct from every other (except the UUID pair). Leave `TestPracticeEventIDIsDeterministicBase32Hex` (`schedule_test.go:111`) untouched. Run: `""`/`"wxyz"` fail on length and `"a-b"`/`"ab"` on distinctness → red.
- [ ] **Step 2:** `schedule.go` — rewrite `PracticeEventID` (lines 95-111): lower-case the input; if it is a canonical UUID (36 chars, hyphens at offsets 8/13/18/23, hex everywhere else — a small unexported `isCanonicalUUID(s string) bool`), return `"aelp" + the 32 hex digits` exactly as today; otherwise return `"aelp" + hex.EncodeToString(sha256.Sum256([]byte(userID))[:])[:32]` (`crypto/sha256`, `encoding/hex`). Update the doc comment: the UUID path is unchanged for compatibility with stored `google_sync.calendar_event_id` values and the retry's 409-consumption; the hash path guarantees the 5–1024 base32hex postcondition (36 chars, `[0-9a-f] ⊂ [a-v0-9]`) for any other string (no caller passes one today — `auth.UserID` is the JWT subject issued from `users.id UUID`).
- [ ] **Step 3:** `go test ./internal/google -count=1` green (service/handler tests use `PracticeEventID("u1")` symbolically and keep passing). Commit: `google: PracticeEventID keeps the 5–1024 base32hex bound for every input; UUID ids unchanged`.

## Verification
```
cd backend && go build ./... && gofmt -l internal/google && go vet ./internal/google
go test -timeout 120s ./internal/google -count=1 -race -v 2>&1 | grep -E '^(--- (FAIL|SKIP)|ok|FAIL)'      # only TestIntegration* may SKIP
go test -timeout 300s ./... -count=1
grep -n 'reauthReasons\|isReauth403\|ACCESS_TOKEN_SCOPE_INSUFFICIENT' internal/google/client.go && ! grep -n throttleReasons internal/google/*.go
grep -n 'ACCESS_TOKEN_SCOPE_INSUFFICIENT' ../harness/CODEMAP.md
grep -n 'apioff\|svcdisabled\|unknownreason\|scopedetail\|events/taken' internal/google/calendar_test.go
grep -n 'func TestTasksMapsAnAPIDisabled403ToUpstreamNotReauth\|func TestFakeCalendarAssignsAFreshIDPerGoogleAssignedInsert\|func TestSyncHandlerMapsA409OnThePatchAfterA409InsertTo502\|func TestPracticeEventIDHoldsItsBoundsForEveryInput' internal/google/*_test.go   # four hits
sed -i '' 's/ev.ID = PracticeEventID(userID)/ev.ID = ""/' internal/google/service.go && (go test ./internal/google -run FailedSaveAfterTheEventInsert -count=1 2>&1 | grep -q 'retry inserted a second event') ; git checkout -- internal/google/service.go && git diff --quiet internal/google/service.go
git diff --stat origin/main -- . .. | grep -v 'internal/google/\(client\|schedule\)\.go\|internal/google/.*_test\.go\|harness/CODEMAP.md\|harness/plans/' | grep -v '^ *[0-9]* files\? changed'   # must print nothing
git diff --quiet origin/main -- internal/google/token.go internal/google/token_test.go internal/google/integration_test.go
git push -u origin harness/2026-09-27-low-google-403-accessnotconfigured-api-disabled-still-maps-to-re
```

## Notes
- **Follow-ups outside the safe set — list in the Execution summary, do not edit:** the backend spec's §6.4 error table, if it enumerates the 403 → `reauth_required` mapping (`grep -n 'reauth_required' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md"`), should say "scope/permission 403 only"; `deploy/README.md`'s Google console checklist could state that the Calendar API and Tasks API must be enabled in the project (an API-disabled deployment now answers 502 with the reason in the log instead of a consent loop).
- The unmerged `google-refresh-token-…` branch also edits the CODEMAP `google` bullet; Task 1 Step 3 changes one sentence only so the daily integration merge stays trivial.
- Not reproduced against Google: the owner's project has both APIs enabled. The mapping was confirmed by reading `client.go`; the `apioff` / `svcdisabled` bodies are Google's documented `accessNotConfigured` / `ErrorInfo SERVICE_DISABLED` shapes.
