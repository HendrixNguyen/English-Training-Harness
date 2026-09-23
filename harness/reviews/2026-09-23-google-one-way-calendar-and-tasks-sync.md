---
plan: harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/every-google-403-becomes-409-reauth-required-so-a-quota-erro.md, harness/ideas/_inbox/a-failed-savesyncstate-orphans-the-google-object-just-create.md, harness/ideas/_inbox/a-user-deleted-tasks-list-is-never-rebuilt-and-sync-keeps-an.md, harness/ideas/_inbox/the-google-sync-route-logs-nothing-so-a-502-or-500-discards-.md, harness/ideas/_inbox/no-test-asserts-the-calendar-patch-body-so-an-empty-patch-pa.md, harness/ideas/_inbox/the-google-fakes-have-no-error-field-for-eleven-of-sync-s-er.md, harness/ideas/_inbox/synctimeout-gives-thirty-sequential-google-calls-a-two-secon.md, harness/ideas/_inbox/pgrefreshtokensource-has-no-sentinel-for-a-missing-user-so-a.md, harness/ideas/_inbox/store-reset-hard-codes-the-down-migration-list-so-migration-.md, harness/ideas/_inbox/schedule-go-boundaries-are-untested-at-notification-time-equ.md, harness/ideas/_inbox/a-roadmap-with-zero-exercise-rows-creates-an-empty-google-ta.md]
---
# Review — Google: one-way Calendar and Tasks sync

**Plan:** `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md`
**Branch/worktree:** `harness/2026-09-23-high-google-one-way-calendar-and-tasks-sync` / `.worktrees/google-one-way-calendar-and-tasks-sync`
**Diff:** `git diff main...harness/2026-09-23-high-google-one-way-calendar-and-tasks-sync --stat`

## Plan vs idea

Delivered. Every item of the idea's *Expected output* exists: `backend/internal/google`,
`POST /api/v1/integrations/google/sync` behind `auth.Require()` (`cmd/api/main.go:98,112` — the
route is in the `guarded` group, and `handler.go:23-27` re-checks `auth.UserID` as defence in
depth), a recurring 30-minute "English practice" event with `RRULE:FREQ=DAILY;COUNT=28`, an
"English daily quests" Tasks list with one task per roadmap day, idempotent re-sync, and
`invalid_grant` → `409 reauth_required`. One-way holds: nothing in the package reads Google state
back, and `grep -n 'roadmap_json' internal/google/*.go` confirms it never writes into onboarding's
document (the single hit is an `INSERT` in the integration-test fixture, required because the column
is `NOT NULL`).

Two idea-level items were superseded on the record, correctly:
- **Wire shape.** The idea's `{calendar_event_id, tasklist_id, tasks_created}` was replaced by
  backend spec §6.4's `{status, calendar_event_id, tasks_created_count}`. Per AGENTS.md the backend
  spec wins for its own layer; the idea's `## Evaluation` and the plan's *Spec precedence* both say
  so. Verified against the spec itself (line 344): the field names, the `200`, the path and the
  `Authorization: Bearer <JWT>` header all match. **No §6.4 drift found** — this slice does not
  repeat the `token` vs `access_token` class of bug filed against auth.
- **Storage.** `google_sync` via migration `0002` instead of JSONB on `roadmaps`, as the idea's
  `## Evaluation` directed.

One idea item quietly did not survive: "when no active roadmap exists, only the Calendar event,
**and the response says so**". It does not say so — `tasks_created_count: 0` is the only signal, and
§6.4 has no field for it. The plan records that decision in *Notes*. Acceptable, but it collides
with a real case (see the zero-exercise bug below) where the same body means something else.

## Code vs plan

All 8 tasks followed as written; 8 commits, one per task, `git status --short` clean. Verification
re-run in the worktree, every command reproduces:

```
$ go build ./... && go vet ./...
BUILD+VET OK

$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 180s
ok  .../internal/airouter 0.658s    ok  .../internal/auth 1.555s
ok  .../internal/config 0.892s      ok  .../internal/google 2.171s
ok  .../internal/health 3.405s      ok  .../internal/pet 2.808s
ok  .../internal/quests 4.604s      ok  .../internal/store 3.965s

$ go test ./internal/google/... -run 'Sync|Resync' -v -count=1 -timeout 60s
--- PASS x8: FirstTimeInsertsEventListAnd28Tasks, PersistsStateAfterTheEventAndAfterTheList...,
    FailureMidTasksKeepsTheListIDSoRetryDeletesIt, ResyncSameRoadmapPatchesEventAndCreatesNothing,
    ResyncNewRoadmapDeletesOldListAndBuildsANewOne, ResyncReinsertsTheEventWhenGoogleLostIt,
    SyncWithoutARoadmapPushesOnlyTheEvent, SyncNeedsReauthWithoutARefreshTokenOrOnInvalidGrant
    (+ 4 handler tests also matched the pattern; TestIntegrationSyncStateIsOneRowPerUser SKIP)

$ go test ./internal/google/... -run 'OAuthClient|Calendar|Tasks' ... -> 14 --- PASS
$ go test ./internal/google/... -run 'NextOccurrence|PracticeEvent|DayDue' ... -> 4 --- PASS
$ go test ./internal/store/... -run 'Migration0002|AppliesPendingVersions' ... -> 2 --- PASS
$ grep -rn 'googleapis.com' internal/google/*_test.go            -> no hits
$ grep -c '"status"|"calendar_event_id"|"tasks_created_count"' internal/google/service.go -> 3
$ grep -rn '"tasklist_id"|"tasks_created"' internal/google/      -> no hits
$ grep -c 'RRULE:FREQ=DAILY;COUNT=28' internal/google/schedule.go -> 1
$ grep -rn 'Getenv' internal/google/  -> 1 (TEST_DATABASE_URL only)
$ grep -c '^func TestIntegration' internal/google/integration_test.go -> 1
$ grep -c 'google.SyncHandler' cmd/api/main.go -> 1
$ python3 tools/harness/cli.py validate -> exit=0
$ git status --short -> clean
$ gh run list --branch harness/2026-09-23-high-google-one-way-... -> completed success (35813268849)
```

**The three grep "misses" the executor logged are genuine plan-authoring artifacts, not a real
miss.** Re-run and inspected hit by hit:
- `google_refresh_token` — 5 hits, not 1. Four are in `token.go` (three doc-comment lines, plus the
  one `refreshTokenSQL` constant at `token.go:34`) and one is the `INSERT` fixture at
  `integration_test.go:35`. The grep's *intent* — a single read seam for the §7 encryption fix —
  holds exactly: `token.go:34` is the only place the column is read.
- `aes|cipher|ENCRYPTION_SECRET_KEY` — 1 hit, `token.go:19`, a doc comment explaining that
  encryption is out of scope. No cipher code exists.
- `roadmap_json` — 1 hit, `integration_test.go:71`, the `NOT NULL` fixture column. Neither
  `service.go` nor `repo.go` mentions it.

**The spec edit is faithful and minimal.** `git diff main...HEAD -- "project-base/… Backend
Technical Specification.md"` is `+10 -0`, all of it appended after the `exercises` DDL: one comment
line naming migration `0002`, then the `CREATE TABLE google_sync` block **byte-identical** to
`backend/internal/store/migrations/0002_google_sync.up.sql:8-15`. Nothing else in the specification
changed — no prose, no other DDL, no §6.4 edit. This is the AGENTS.md requirement met, not a slice
rewriting its own contract.

**Migration quality is good.** `0002_google_sync.down.sql` is a true inverse
(`DROP TABLE IF EXISTS google_sync;`). The up-migration is safe against a populated `users` table:
a new table, every column nullable or defaulted, no backfill, no rewrite. And the "one sync row per
user" invariant is *real*, not conventional — `user_id UUID PRIMARY KEY REFERENCES users(id) ON
DELETE CASCADE` is an index-backed constraint, which is exactly the gap onboarding's review found
elsewhere; `TestIntegrationSyncStateIsOneRowPerUser` and the `ON CONFLICT (user_id) DO UPDATE` in
`repo.go:80-88` rest on it legitimately. `roadmap_id … ON DELETE SET NULL` is handled by the
`COALESCE`/`NULLIF` convention in `repo.go:74,82`. One cosmetic inconsistency: `0001_init` uses
`CREATE TABLE IF NOT EXISTS` and `0002` does not — harmless given `schema_migrations`, worth
matching next time.

**Deviation 1 (commit trailer)** is cosmetic and self-reported; the trailer says
`Claude Sonnet 5` where the plan text says `Claude Opus 5 (1M context)`. Noted, not a finding.
**Deviation 3** (a temporary `livecheck_test.go`, written for live proof and deleted) is confirmed
absent from the branch — the diff is the plan's files only.

## Quality

**Security — refresh-token handling: clean.** I traced every path the token value can reach and
found no leak.
- It is read in exactly one place, `token.go:34-44`, through the `RefreshTokenSource` seam.
- It never enters an error value: `token.go:39` wraps the pgx error, not the token; `oauth.go`
  places it only in the POST form body and its five error returns (`:47,:57,:62,:71,:80,:83`) carry
  the URL, the status and Google's `error` field, never the credential.
- It never reaches a response: `UpstreamError.Body` holds Google's raw error text but the handler
  answers a fixed `{"error":"google_unavailable"}` (`handler.go:39`), so the slice does not repeat
  `route-returns-upstream-provider-error-bodies-verbatim-to-its.md`.
- It is never logged, because the package logs nothing at all — which is itself a bug, filed
  separately, and whose fix must preserve this property.
- Access tokens travel only in the `Authorization` header (`client.go:55`), not in URLs.
- Committed fixtures are obvious placeholders (`"1//refresh"`, `client_secret: "secret"` at
  `oauth_test.go:36`; a fake column value at `integration_test.go:35`). No real credential is in the
  repo.
- **Plaintext at rest is real but pre-existing and correctly deferred.** `PgRefreshTokenSource`
  reads the column verbatim, which is what the merged `auth` slice writes; spec §7 wants AES-256-GCM
  via `ENCRYPTION_SECRET_KEY`. That is `harness/ideas/_inbox/google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md`,
  already filed — **not re-filed here**. The single-seam design means its fix replaces one struct
  and nothing else, which is the right call for this slice.

**Partial-failure semantics — mostly convergent, with one real hole.** I walked every interruption
point. The good news: the ordering genuinely works for the Google-call failures the plan designed
for. `state.RoadmapID` is written only *after* the task loop completes (`service.go:133`), so a
sync that dies mid-tasks leaves a stored `tasklist_id` with an empty `roadmap_id`, and the retry
takes the delete-and-rebuild branch. `DeleteTaskList` swallows a 404 (`tasks.go:62-68`), so a list
already gone does not wedge the retry — the obvious "500 forever" trap is avoided. A deleted
Calendar event 404s on PATCH and is re-inserted (`service.go:70-73`). All of that converges.

The hole is the **local** write, not the remote one: a failed `SaveSyncState` immediately after a
successful `InsertEvent`/`InsertTaskList` orphans the object just created, because the insert
carries no idempotency key. The plausible trigger is not a crash but a client disconnect — Gin
cancels `c.Request.Context()`, which `handler.go:28` wraps — and the result is a *second* 28-day
recurring event on the next sync, unreachable forever. The plan's *Architecture* and CODEMAP both
assert this cannot happen ("a failure never orphans a Google object"); that sentence is false and is
the part most worth correcting. Filed.

**Idempotency beyond the proven case.** The executor's live proof covered "sync twice". The harder
cases: a deleted *event* recovers (covered, tested); a deleted *tasklist* does **not** — the
same-roadmap branch (`service.go:97-99`) short-circuits before any Tasks call and answers
`{"status":"synced","tasks_created_count":28}` to a user with no list, permanently. Filed.

**Timeouts and blocking.** Every outbound call is bounded twice: a 15 s `http.Client.Timeout`
(`client.go:37`, `oauth.go:32`) and the request context, which `handler.go:28` gives a 60 s
`SyncTimeout` and which reaches every call via `http.NewRequestWithContext` and the pgx `ctx`
arguments. So the worst-case duration of this route is **60 s**, hard — it is in fact the only route
in the backend that sets its own deadline, so this slice does not worsen
`harness/ideas/_inbox/route-has-no-overall-deadline-so-one-call-can-take-90-second.md`
(no server `ReadTimeout`/`WriteTimeout` in `cmd/api`; that bug stands as filed and is **not**
duplicated). What the 60 s does not survive is the arithmetic: ~31 sequential Google calls share it,
about 1.9 s each, and a timeout mid-loop makes the retry delete its own partial list and start from
task 1 — no forward progress. Filed as its own bug.

**Boundaries and conventions: good.** The package talks to Postgres only through its own read-only
SQL plus one upsert into its own table — no cross-package table writes, no import of `quests`
(`Location`/`RoadmapDays` are deliberately duplicated, eight lines, and the plan says to hoist on a
third copy). Five injected interfaces, three HTTP clients with base-URL fields following the
`auth.GoogleClient` pattern, so `go test ./...` never touches Google — verified. No new
dependencies. Error sentinels and `UpstreamError` are idiomatic. The one convention slip is
`token.go` not mapping `pgx.ErrNoRows` to a sentinel the way its siblings `repo.go:108-110,145-147`
do (filed, low). The endpoint's SQL leans on the same missing indexes as quests — already filed as
`no-index-supports-the-quests-lookups-on-roadmaps-and-exercis.md`, **not** duplicated.

**CODEMAP: accurate but for one sentence.** The `google` and `store` paragraphs are substantive and
match the code, including the error mapping, the refresh-token seam and the migration note. The
claim "state is saved after the event and after the list is created, before the task inserts, so a
failure never orphans a Google object" is wrong for the reason above; correcting it is part of that
bug rather than a unilateral edit here, since the fix and the wording should land together. I left
CODEMAP unchanged.

**Test honesty: the weakest part of the slice.** The httptest-based client tests are genuinely
load-bearing — `fakeGoogleAPI` records method, path, auth header and the decoded JSON body, and the
tests assert on real fields (`calendar_test.go:69-76` checks `recurrence`, `start.timeZone`,
`start.dateTime`; `tasks_test.go` checks `title`, `notes`, `due`; `oauth_test.go:36` checks the
urlencoded form). But:
- **No test asserts the Calendar PATCH body.** Proven by mutation: dropping `ev.payload()` from
  `calendar.go:47` — so a re-sync silently stops updating anything — leaves the whole package `ok`.
  The patch test checks only method and path (`calendar_test.go:78-92`), and `fakeCalendar.PatchEvent`
  discards its `Event` argument outright (`fakes_test.go:52`, `_ Event`). The branch's headline
  idempotency claim is verified only as "a PATCH was issued to that id". (Mutation reverted;
  `git status --short` clean.)
- **Eleven of `Sync`'s error branches are unreachable by any test**, because the fakes have no
  error field for them — including all three `SaveSyncState` calls, i.e. exactly the window the
  plan makes its strongest claim about. `handler.go:41`'s 500 branch and the
  `context.DeadlineExceeded` half of `handler.go:38` are untested too.
- `schedule.go`'s boundaries (equality at `notification_time`, month/year rollover, the
  spring-forward gap) are unpinned.
- `TestMigration0002CreatesGoogleSync` asserts on SQL *text*, not on applied DDL; the real DDL is
  covered by the `TEST_DATABASE_URL`-gated integration test, which is the only exercise of
  `repo.go` and `token.go` at all and skips locally (standing bug
  `integration-gate-tests-only-prove-the-skip-and-would-pass-if.md`, not duplicated). CI's
  `backend-integration` job does run it, and it is green.

None of these is a *dishonest* test — no test claims more than it checks — so none is an executor
gate failure. They are gaps, and they are the gaps that matter for this particular slice.

## Bugs filed

Eleven, all in `harness/ideas/_inbox/`. **None is a blocker; nothing holds up the merge.**

Medium:
1. `every-google-403-becomes-409-reauth-required-so-a-quota-erro.md` — `client.go:74-75` maps every 403 to `ErrReauthRequired`, but Google returns 403 for `rateLimitExceeded`/`quotaExceeded`; a throttled user is sent through a re-consent that cannot help.
2. `a-failed-savesyncstate-orphans-the-google-object-just-create.md` — insert-then-save with no idempotency key; a cancelled request leaves a stray 28-day event and the next sync creates a second. Plan and CODEMAP claim the opposite.
3. `a-user-deleted-tasks-list-is-never-rebuilt-and-sync-keeps-an.md` — `service.go:97-99` answers `"synced"` with a stale count for a list that no longer exists, forever.
4. `the-google-sync-route-logs-nothing-so-a-502-or-500-discards-.md` — zero log statements; `UpstreamError`'s service/status/body is dropped on every failure.
5. `no-test-asserts-the-calendar-patch-body-so-an-empty-patch-pa.md` — mutation-proven: an empty PATCH passes the suite.
6. `the-google-fakes-have-no-error-field-for-eleven-of-sync-s-er.md` — eleven error branches unreachable, including all three `SaveSyncState` saves.

Low:
7. `synctimeout-gives-thirty-sequential-google-calls-a-two-secon.md` — ~31 calls in 60 s, and a timeout mid-loop restarts from task 1 instead of resuming.
8. `pgrefreshtokensource-has-no-sentinel-for-a-missing-user-so-a.md` — missing `users` row → 500 instead of 409, against the package's own convention.
9. `store-reset-hard-codes-the-down-migration-list-so-migration-.md` — `integration_test.go:51` hard-codes the down files; migration `0003` will silently not roll back between tests.
10. `schedule-go-boundaries-are-untested-at-notification-time-equ.md` — equality, month/year rollover and spring-forward all unpinned.
11. `a-roadmap-with-zero-exercise-rows-creates-an-empty-google-ta.md` — an exercise-less roadmap gets a permanently empty Google list and is marked fully synced.

Referenced, deliberately **not** duplicated: `google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md`,
`route-has-no-overall-deadline-so-one-call-can-take-90-second.md`,
`no-index-supports-the-quests-lookups-on-roadmaps-and-exercis.md`,
`integration-gate-tests-only-prove-the-skip-and-would-pass-if.md`.

## Verdict

**pass-with-bugs.** The slice delivers the idea and conforms to backend spec §6.4 field for field;
the spec edit is faithful and minimal; the migration is a genuine inverse with a real,
index-backed one-row-per-user invariant; credential handling is careful, with the refresh token
confined to one seam and absent from every error, response and log; CI is green on the pushed
branch; and every verification command reproduces, including the three greps the executor flagged,
which are confirmed plan-authoring artifacts. What holds it back from `pass` is quality, not
delivery: an error-mapping choice that will send throttled users to a pointless consent screen, a
local-write window that can orphan Google objects while the documentation promises it cannot, a
deleted Tasks list that the endpoint reports as synced forever, no logging at all on the backend's
only third-party integration, and a test suite whose fakes cannot reach eleven error branches and
never look at the PATCH body that is the whole of the common re-sync. All eleven are ordinary inbox
bugs; none blocks this branch.

No PR exists (`gh pr create` 403s — the `gh` CLI is authenticated as the owner's work account), so
step 8's PR comment and `gh pr ready` are skipped as instructed. Merge with
`/harness merge harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md`.
