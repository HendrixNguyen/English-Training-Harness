---
plan: harness/plans/2026-09-23-a-failed-savesyncstate-orphans-the-google-object-just-create.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn.md, harness/ideas/_inbox/every-google-409-now-becomes-500-internal-error-instead-of-5.md, harness/ideas/_inbox/practiceeventid-is-a-lossy-filter-that-can-return-a-4-charac.md, harness/ideas/_inbox/fakecalendar-returns-the-same-nextid-for-every-google-assign.md]
---
# Review — google sync: idempotent Calendar insert via a client-supplied event id, and honest docs about what a failed save can orphan

**Plan:** `harness/plans/2026-09-23-a-failed-savesyncstate-orphans-the-google-object-just-create.md`
**Branch/worktree:** `harness/2026-09-23-high-a-failed-savesyncstate-orphans-the-google-object-just-create` / `.worktrees/a-failed-savesyncstate-orphans-the-google-object-just-create`
**Diff:** `git diff main...harness/2026-09-23-high-a-failed-savesyncstate-orphans-the-google-object-just-create --stat`
## Plan vs idea

The idea's *Expected output* was: `InsertEvent` sends a caller-supplied event id derived
deterministically from the user id; a repeat insert becomes a 409 the service treats as "already
ours" and patches; the Tasks half keeps its ordering but the plan's and CODEMAP's "a failure never
orphans a Google object" is corrected to the truth; and a test injects a `SaveSyncState` error
immediately after `InsertEvent` and asserts the following sync does not produce a second event.

All four delivered.

- `PracticeEventID(userID)` (`backend/internal/google/schedule.go:102`) is deterministic, and
  `Event.payload()` (`schedule.go:78-93`) sends `id` when set plus `status: "confirmed"`.
- `doJSON` maps 409 to `ErrAlreadyExists` (`client.go:82-83`) and `Sync` patches on it
  (`service.go:88-100`).
- `grep -n "never orphans" internal/google/service.go ../harness/CODEMAP.md` -> exit 1, no matches.
  Repo-wide the phrase survives only in harness bookkeeping (the merged plan, which already carries
  the evaluator's correction blockquote at
  `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md:21`, the parent idea, and the
  previous review) — all correctly left alone.
- `TestAFailedSaveAfterTheEventInsertDoesNotCreateASecondEvent` (`service_test.go:182`) is exactly the
  test the idea asked for.

## Code vs plan

Four tasks, four commits, all followed. Three deviations, all assessed below; nothing missing.

**Verification re-run in the worktree (real output):**

```
go build ./... && go vet ./... && gofmt -l ./internal/google
VET OK / GOFMT CLEAN

env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 300s
?   .../backend/cmd/api      [no test files]
ok  .../internal/airouter    0.586s
ok  .../internal/auth        1.233s
ok  .../internal/config      0.687s
ok  .../internal/google      1.730s
ok  .../internal/health      2.154s
ok  .../internal/onboarding  2.683s
ok  .../internal/pet         3.203s
ok  .../internal/quests      4.201s
ok  .../internal/store       3.645s
```

Runtime proof reproduced on the reviewer's own ports (`COMPOSE_PROJECT_NAME=orphrev`, Postgres 5451,
Redis 6399, API 8105):

```
migrations applied: [0001_init 0002_google_sync]
listening on :8105
GET  /healthz -> 200 {"postgres":"ok","redis":"ok","status":"ok"}
POST /api/v1/integrations/google/sync (valid session, google_refresh_token NULL)
     -> 409 {"error":"reauth_required"}
```

Integration test against that stack:
`TEST_DATABASE_URL=... go test ./internal/google/... -run Integration -count=1 -p 1` ->
`--- PASS: TestIntegrationSyncStateIsOneRowPerUser (0.04s)`.

CI on the branch: run `35843798665` (pull_request), all four jobs `success` —
`harness-tooling`, `backend-unit`, `frontend`, `backend-integration`. The push run `35843708403` is
also green. Nothing red or missing.

Stack torn down: `docker compose -p orphrev down -v` removed both containers, the volume and the
network; `docker ps -a`/`docker volume ls` filtered on `orphrev` are empty; the API process is killed;
the scratch `.env` and helper package were deleted and `git status --short` in the worktree is empty.

### Mutation checks

**Mutation 1 (the plan's own, `ev.ID = PracticeEventID(userID)` -> `ev.ID = ""`):** reproduces.

```
--- FAIL: TestAFailedSaveAfterTheEventInsertDoesNotCreateASecondEvent (0.00s)
    service_test.go:202: retry should patch aelpu1 once, got [{ID: Ev:{... ID:}}]
FAIL	github.com/HendrixNguyen/English-Training-Harness/backend/internal/google	0.670s
```

Note the message is the patch-id assertion, **not** the `"retry inserted a second event"` the plan's
Verification step 3 predicted — `len(h.cal.inserted)` stays 1 because the fake 409s on a repeated
`nextID`. The executor reported the real message rather than the predicted one. Filed as a
low-severity fake-fidelity bug; the test itself is not hollow.

**Mutation 2 (reviewer's own, on the least-exercised branch — delete `ev.ID = ""` inside the
404 fallback so the fallback insert keeps the deterministic id):** kills the fallback test.

```
--- FAIL: TestAReservedButGoneIDFallsBackToAGoogleAssignedInsert (0.00s)
    service_test.go:219: google: resource already exists: calendar returned 409
FAIL	github.com/HendrixNguyen/English-Training-Harness/backend/internal/google	0.534s
```

Both mutations reverted; `git diff --quiet internal/google/service.go` clean afterwards.

### The three executor deviations

**1. Regex split — correct, and the plan was wrong.** Go's RE2 caps a repeat count at 1000, so the
plan's `^[a-v0-9]{5,1024}$` panics at `MustCompile`. The replacement — `^[a-v0-9]+$` plus
`if len(id) < 5 || len(id) > 1024` (`schedule_test.go:117-124`) — enforces the identical constraint:
`+` gives at least one character and the explicit test gives both boundaries. Accepted.

**2. Tasks assertion — the conclusion is right, the stated reason is backwards.** I measured it with a
temporary probe rather than accepting either account. `fakeTasks.InsertTaskList`
(`fakes_test.go:109-118`) returns `f.nextList` and never writes a key; only `InsertTask` does
(`fakes_test.go:129-138`). With the orphan present the probe printed:

```
PLAN ASSERTION: len(h.tasks.tasks)=1 (plan wanted 2), keys=[list_second], deleted=[]
```

So the plan's `len(h.tasks.tasks) != 2` would have **failed**, not "spuriously passed" as the execution
summary says. Either way the plan's assertion did not measure lists created at Google, so the
executor's charge that it could not detect the orphan is correct. The replacement — counting
`tasks.InsertTaskList(` in the shared call log (`service_test.go:244-250`) — does measure it:
created=2, deleted=0 with the gap present, and it breaks if any future fix either deletes the orphan
or avoids creating the second list. Genuine, not hollow. Accepted with the correction noted.

**3. `handler_test.go` spec-body update — necessary, not scope creep.**
`TestSyncHandlerAnswersTheSpec64Body` drives the first-time insert path, which now returns
`PracticeEventID("u1")` instead of the fake's `nextID`. The one-line change is the same substitution
the plan enumerates for the 8 occurrences in `service_test.go`; without it the package does not
compile green. Accepted.

### Scope

`backend/cmd/api/main.go` is untouched — the diff is 9 files
(`internal/google/{calendar_test,client,fakes_test,handler_test,schedule,schedule_test,service,service_test}.go`
and `harness/CODEMAP.md`). No idea file is in the diff, so the other google inbox findings are
untouched; the two folded ones are `status: rejected` with `rejected_reason` pointing at this plan.

## Quality

**Is `PracticeEventID` collision-free?** In production, yes. `users.id` is `UUID`
(`0001_init.up.sql:12`) and `auth.UserID(c)` is the JWT subject issued from that column, so every real
id is a canonical 36-character UUID; all 32 hex digits are inside `[a-v0-9]` and the hyphens sit at
fixed positions, so the map is injective and the output is 36 characters — comfortably inside
Google's 5-1024. Confirmed end to end on the live stack: user `b855d068-5368-4881-ad7c-5d5e895663da`
-> `aelpb855d06853684881ad7c5d5e895663da`.

Outside canonical UUIDs the function is lossy and unguarded. Measured: `PracticeEventID("")` ==
`PracticeEventID("wxyz")` == `"aelp"` — four characters, below the minimum the function's own doc and
test assert — and `PracticeEventID("a-b") == PracticeEventID("ab")`. Not reachable today, and a
collision would not cross users anyway (Calendar event ids are per-calendar and each user's event is
on their own account's `primary`), so this is a low-severity robustness gap, filed.

**Does the 409 path converge?** Yes. insert -> 409 -> patch; a patch failure that is not `ErrNotFound`
returns the error (`service.go:101-103`), so there is no loop and no path that returns success without
a write. Because `state.CalendarEventID` stays empty on every failure, the next sync re-enters the same
branch and converges as soon as Google is healthy. A patch that succeeds followed by a failed
`SaveSyncState` also converges — the retry 409s and patches again, idempotently.

**Is the 404 fallback orphan-free?** **No — merely narrower.** `service.go:98-99` inserts with
`ev.ID = ""`, and the id is only persisted at `service.go:105-108`. A `SaveSyncState` failure there
orphans the Google-assigned event exactly as before the fix, and it repeats without bound: the next
sync inserts the deterministic id, gets a 409 (the id is still reserved), patches, 404s again, and
inserts another Google-assigned event. The branch needs a manual delete plus Google releasing the id,
so it is narrow, and the comment at the site is honest ("today's non-idempotent path"). But `Sync`'s
new function doc ends "The Calendar half is covered by the deterministic id"
(`service.go:40-43`), which is false in that branch, and the CODEMAP replacement describes the fallback
without naming the residual gap. Replacing a broad false invariant with a narrow one is the failure
mode this plan was filed to end, so it is filed as a bug — medium rather than a blocker, because no
code is wrong and a maintainer who reads past the doc comment finds the truth twenty lines down.

**The documented Tasks gap is honest, and the test genuinely pins it.** `service.go:40-43` states it in
the second paragraph of the `Sync` doc, not in a footnote; CODEMAP marks it **Known gap** and names the
test. The test asserts created=2, deleted=0 from the call log, which is the real measure — see
deviation 2 above. It is not overstated.

**The false invariant is gone.** Both `service.go` and `harness/CODEMAP.md` no longer claim a failure
never orphans a Google object; the scoped grep the plan specifies returns nothing. What replaced them
is true for the main path; the one over-broad clause is the fallback sentence above.

**Test honesty.** 34 tests in the package, all passing. The two new handler tests close the previously
unreachable 500 and 502 branches. The PATCH-body assertions (`calendar_test.go:103-111` and
`service_test.go:114-121`) close the folded "an empty patch passes the whole suite" finding — an empty
patch body now fails. Both mutations above kill a test, including on the 404 fallback, the least
exercised branch. Two fidelity gaps, both filed low: `fakeCalendar` reuses `nextID` for every
Google-assigned insert, and `TestPracticeEventIDIsDeterministicBase32Hex` pins only one UUID.

**Error mapping regression.** `doJSON` now maps 409 for **every** Google service
(`client.go:82-83`), but `SyncHandler` has no `ErrAlreadyExists` case (`handler.go:34-42`), so any 409
the insert path does not consume falls to the catch-all and is answered `500 internal_error` where it
used to be `502 google_unavailable`. `events.patch` on a stored id can return 409 on a concurrent
modification and `service.go:73-80` returns it straight out. This contradicts the CODEMAP contract
("other Google failures ... -> 502") and is the same class as the already-selected
`every-google-403-becomes-409` finding. Filed medium.

**Boundaries and conventions.** No package boundary is crossed: everything stays inside
`internal/google`, dependencies are still the five interfaces, no new table access, no new dependency.
Style matches the package — sentinel errors beside `ErrNotFound`, a pure helper in `schedule.go`, fakes
in `fakes_test.go`. `gofmt` clean, `go vet` clean.

## Bugs filed

- `harness/ideas/_inbox/the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn.md` — **medium.** The 404 fallback inserts a Google-assigned id and can orphan it on a failed save, repeatedly; `Sync`'s doc says the Calendar half is covered.
- `harness/ideas/_inbox/every-google-409-now-becomes-500-internal-error-instead-of-5.md` — **medium.** A 409 from any service other than the Calendar insert now surfaces as 500 rather than 502.
- `harness/ideas/_inbox/practiceeventid-is-a-lossy-filter-that-can-return-a-4-charac.md` — **low.** Lossy filter, no guard on the 5-character minimum, test pins only one UUID. Not reachable while ids are UUIDs.
- `harness/ideas/_inbox/fakecalendar-returns-the-same-nextid-for-every-google-assign.md` — **low.** Two auto-id inserts collide in the fake, so the plan's mutation reports the wrong symptom.

## Verdict

**pass-with-bugs.** The plan delivers the idea: the Calendar insert is idempotent under a client
disconnect, the 409 path converges, the false invariant is gone from code and CODEMAP, and the Tasks
gap is stated plainly and pinned by a test that genuinely measures it. Build, vet, gofmt, the full
suite, the integration test, the runtime proof and all four CI jobs reproduce. Both mutations —
the plan's and mine on the 404 fallback — kill a test.

Four bugs filed, none a blocker. **Nothing blocks the merge.** The two medium findings (the fallback's
residual orphan window plus its over-broad doc sentence, and the 409 -> 500 mapping) are follow-up
work for the evaluator; the 409 one pairs naturally with the already-selected
`every-google-403-becomes-409` finding, which touches the same switch.
