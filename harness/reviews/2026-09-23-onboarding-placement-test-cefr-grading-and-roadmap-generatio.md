---
plan: harness/plans/2026-09-22-onboarding-placement-test-cefr-grading-and-roadmap-generatio.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/two-concurrent-assessment-submits-double-spend-the-ai-and-or.md, harness/ideas/_inbox/a-re-submitted-assessment-silently-discards-target-goal-noti.md, harness/ideas/_inbox/no-post-handler-bounds-the-request-body-so-one-jwt-can-decod.md, harness/ideas/_inbox/onboarding-service-stores-a-now-clock-it-never-uses-so-every.md, harness/ideas/_inbox/get-onboarding-quiz-is-shipped-but-absent-from-backend-spec-.md]
---
# Review — Onboarding placement test CEFR grading and roadmap generation

**Plan:** `harness/plans/2026-09-22-onboarding-placement-test-cefr-grading-and-roadmap-generatio.md`
**Branch/worktree:** `harness/2026-09-23-high-onboarding-placement-test-cefr-grading-and-roadmap-generatio` / `.worktrees/onboarding-placement-test-cefr-grading-and-roadmap-generatio`
**Diff:** `git diff main...harness/2026-09-23-high-onboarding-placement-test-cefr-grading-and-roadmap-generatio --stat`

## Plan vs idea
Delivered. The idea's *Expected output* — a placement quiz the client can fetch,
answers graded to a CEFR level through `airouter`, a validated 28-day roadmap and
its 84 exercises persisted, the §6.1 body on the wire, and the quests slice's
TEMPORARY `store.SeedDemoRoadmap` retired — all exist on the branch.

Spec conformance is exact. The §6.1 request (`target_goal`, `notification_time`,
`timezone`, `answers[{question_id, selected_option}]`) and the 201 body
(`status`, `assessed_level`, `roadmap_id`, `pet_state{plant_name, health_points,
stage}`) match the backend spec's JSON examples field for field
(`internal/onboarding/types.go:9-45` against
`project-base/Adaptive English Learning Platform - Backend Technical Specification.md:254-268`),
and `TestAssessmentReturns201WithTheSpec61Body` (`handler_test.go:53`) asserts
the serialised body as a literal string, so drift breaks the test. **No
contract drift of the `token`/`access_token` class exists in onboarding's
responses.** The one addition beyond the spec — `GET /api/v1/onboarding/quiz` —
is filed as a low spec bug, exactly as the plan asked.

The seed retirement is complete: `internal/store/seed.go` and `seed_test.go` are
deleted, `grep -rn SeedDemoRoadmap --include='*.go'` returns nothing, and
`internal/quests/integration_test.go` now seeds through
`onboarding.NewPgRepo(...).SaveAssessment`, which makes that test a live
contract check on the `content_json` shape quests reads. This also closes the
standing inbox bug
`seeddemoroadmap-is-not-transactional-and-can-leave-a-partial.md` — the
non-transactional helper it describes no longer exists.

## Code vs plan
All ten tasks followed, one commit each, file structure as the plan's table
specifies. The three logged deviations check out:

1. **Task 5 `go vet` split — intent preserved and slightly strengthened.** The
   plan's single assertion `h.quiz.lastTTL != store.PlacementQuizTTL ||
   h.quiz.lastTTL != 2*time.Hour` is `x != a || x != a` once you substitute, so
   it only ever tested the first clause and `go vet`'s suspect-or check is right
   to reject it. The split (`service_test.go:70-75`) asserts *both* facts
   separately — `lastTTL == store.PlacementQuizTTL`, and
   `store.PlacementQuizTTL == 2h` per §4 — which is strictly more coverage than
   the original expression could give. Justified.
2. **Task 8's second `SeedDemoRoadmap` call site.** Correct and necessary; the
   plan's own Step 3 requires zero remaining hits repo-wide, unachievable
   otherwise. The replacement reuses one `integrationRoadmap()` fixture for both
   users.
3. **Task 10 grep-count nit.** Cosmetic; the CODEMAP content is right.

### Verification re-run (from the worktree, this review)

```
$ cd backend && go build ./... && go vet ./...
(no output)

$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1
?   	.../backend/cmd/api	[no test files]
ok  	.../backend/internal/airouter	0.876s
ok  	.../backend/internal/auth	1.328s
ok  	.../backend/internal/config	2.416s
ok  	.../backend/internal/health	1.924s
ok  	.../backend/internal/onboarding	3.256s
ok  	.../backend/internal/pet	4.030s
ok  	.../backend/internal/quests	5.653s
ok  	.../backend/internal/store	4.726s

$ git status --short
(clean)

$ gh run list --branch harness/2026-09-23-high-onboarding-placement-test-cefr-grading-and-roadmap-generatio
completed  success  CI  push  35811737991  44s  2026-09-23T02:47:06Z
```

CI green on the branch. No PR exists — `gh pr create` 403s because the `gh` CLI
is authenticated as an account without write access to this repo; the branch
push is what gates CI, and it ran. Not a finding.

Per the role's scope, the executor's runtime proof was not re-run as the main
activity; the build, the full suite and CI all reproduce, and the executor's
live-boot evidence is consistent with what the code does. **No executor gate
failure.**

## Quality
### Atomicity — correct

`PgRepo.SaveAssessment` (`repo.go:94-141`) is one transaction: `Begin`, a
`defer tx.Rollback`, `UPDATE users`, deactivate, `INSERT roadmaps ... RETURNING
id`, an 84-statement `pgx.Batch` whose every result is checked individually
(`repo.go:127-132`) rather than only the last, `results.Close()` checked, then
`Commit`. There is **no state where a user has a roadmap but the wrong CEFR, or
exercises orphaned from their roadmap** — a failure at any point, including
mid-batch, rolls the whole thing back. `tag.RowsAffected() == 0` on the user
update is turned into `ErrUnknownUser` rather than committing a roadmap for a
user row that does not exist. `TestAssessFailsWithoutWritingWhenTheRoadmapIsBadTwice`
(`service_test.go:138`) pins that the user update never runs alone.

Both AI calls complete before the transaction opens (`service.go:73-88` vs
`service.go:90`), so **no database transaction is ever held across a provider
call**. That was the specific risk to check and it is absent.

### The AI boundary — bounded per call, unbounded per request

Each provider call has a 30 s client timeout (`airouter/gemini.go:15,42`,
`openai.go:39`), so no single hang pins a request forever. The *request* is the
weak point: onboarding makes two `Route` calls, each retried once
(`service.go:114-132`), and each `Route` may itself fall back across three
providers. Worst case is four `Route` calls ≈ 12 provider attempts ≈ 6 minutes,
with no `context.WithTimeout` anywhere on the path and no `ReadTimeout`/
`WriteTimeout` on the server (`cmd/api/main.go:129`, `r.Run`). This is the
already-filed
`harness/ideas/_inbox/route-has-no-overall-deadline-so-one-call-can-take-90-second.md`
at four times the magnitude; I did not re-file it, but whoever plans that bug
should size the fix against onboarding, not against a single `Route`.

JSON is validated for **shape**, not just parseability. `airouter.ParseRoadmap`
(`airouter/roadmap.go:72-132`) rejects fences, preamble, trailing tokens, and
enforces 4 modules × 7 days × 3 tasks with the three task types each present
exactly once, non-empty titles and durations in 1..30. A well-formed-but-wrong-
shape roadmap **cannot** reach the database and cannot break
`/quests/daily` later. `TestAssessRecoversWhenTheSecondRoadmapAttemptIsValid`
and `TestAssessFailsWithoutWritingWhenTheRoadmapIsBadTwice` exercise both
branches with real malformed payloads.

### Grading honesty — correct

`ParsePlacement` (`grade.go:55-74`) checks the decoded `cefr_level` against the
§3.2 enum set before returning (`grade.go:70`), so an out-of-range or unknown
level is an `ErrBadAIOutput` → retry → 502, never a Postgres enum cast error →
500. A `cefr_level` of `"b1"`, `"Beginner"` or `""` is rejected at the parser.
`grade_test.go` covers it.

### Input validation — bounded, with one gap outside this package

`validate` (`service.go:134-165`) runs before the limiter, before Redis and
before any AI call, and `TestAssessValidatesTheRequestBeforeTouchingAnything`
asserts zero side effects across nine mutations. The `duration_seconds`-class
overflow risk is absent: `target_goal` is bounded 1..255 bytes against a
`VARCHAR(255)` column (bytes ≤ characters, so the check is the strict side),
`timezone` must be a real IANA zone via `time.LoadLocation` and IANA names fit
`VARCHAR(50)`, `notification_time` must parse as `15:04:05`, and `answers` is
implicitly bounded at 10 because every id must be in `Bank` and duplicates are
rejected. What is *not* bounded is the HTTP body itself before
`ShouldBindJSON` decodes `[]Answer` into memory — filed as a medium,
repo-wide bug, pre-existing across all four handlers.

### Idempotency — works, but on a check-then-act plus an accident

This was the sharpest thing I looked at. The guard is a pool-level `SELECT`
(`service.go:48`) with no constraint behind it: `roadmaps` has no unique index
on the active row (`0001_init.up.sql:53-59`), so "active" is a convention. Two
concurrent submits both pass the check and both pay for two Gemini calls.

The "one active roadmap" invariant nevertheless holds today — but only because
`SaveAssessment` happens to run `UPDATE users` first (`repo.go:106`), taking a
row lock that serialises the two transactions, so under READ COMMITTED the
loser's `deactivateSQL` takes a fresh snapshot that already contains the
winner's roadmap. That is load-bearing, undocumented and reversible by an
innocent statement reorder. The residual damage even when it works — a duplicate
roadmap generation billed, an inactive roadmap with 84 dead exercise rows, and a
`roadmap_id` in the winner's 201 that is already `is_active = FALSE` — is
filed as the first medium bug, with the partial unique index as the fix.

### Boundaries and conventions — clean

`onboarding` imports neither `pet` nor `quests` (verified, zero hits); the
`Pet`/`Generator` seams are interfaces and `cmd/api/main.go`'s
`petForOnboarding` adapter (`main.go:25-36`) keeps the cycle from forming, with
the reason written down. Redis keys and TTLs go through `store.PlacementQuizKey`
/ `store.PlacementQuizTTL` — no literal key strings. The package reads no other
package's tables: it owns `roadmaps`/`exercises` writes and the four `users`
columns §6.1 puts on this request. `StageAnswers` uses a `TxPipeline` with
`DEL`+`HSET`+`EXPIRE` so a re-take replaces rather than merges stale answers,
and a `Clear` failure is logged rather than failing a successful assessment
(`service.go:105-107`) — correct, since the key expires anyway. Error mapping
(`handler.go:43-60`) covers 400/429/502/503/500 and is pinned by a table test.
File layout, naming, doc comments and the spec-section references all match the
surrounding packages.

### Test honesty — the assertions are load-bearing

I checked the key ones for whether a regression would actually fail them. They
would: the 201 body is compared as an exact literal string, the happy path
counts AI calls per task type, limiter calls, pet ensures and saved
assessments, the failure paths assert `len(h.repo.saved) != 0` is a failure
(so "writes nothing on a 502" is genuinely pinned, not assumed), the retry test
scripts a fenced-then-wrong-key pair and asserts exactly 2 placement calls and
0 roadmap calls, and the rate-limit test asserts an empty side-effect set. The
scripted provider sits behind a **real** `airouter.Router`, so the production
`Generator` type is what is exercised. The integration test asserts
`active=1, total=2`, `84/28/28` exercises, and reads
`content_json->>'title'` / `duration_minutes` back — the exact keys quests'
`toTask` consumes. No assertion I found was vacuous.

The gaps are the ones the filed bugs name: nothing tests concurrent submits
(both the unit and the integration idempotency tests are strictly sequential),
and `TestAssessIsIdempotentWhileARoadmapIsActive` pins the silent discard of
`timezone`/`target_goal`/`notification_time` as intended behaviour rather than
questioning it.

### Migrations

None added, none needed — the slice writes only tables `0001_init` already
creates. The spec DDL ↔ `0001_init.up.sql` identity is therefore untouched.
(The first filed bug proposes a new migration for the partial unique index;
that change must update the backend spec's DDL in the same commit.)

### CODEMAP

Accurate. The `onboarding` bullet describes the real order of operations, the
real error mapping and the real transaction contents; the `SeedDemoRoadmap`
sentence is correctly removed from the `store` bullet. No correction needed.

### Nits (not filed)

- `PublicBank()` (`bank.go`) copies each `Question` into a `PublicQuestion` but
  shares the `Options` map by reference with the package-level `Bank`. Nothing
  mutates it, so it is not a bug today; a `maps.Clone` would make the "public"
  in the name true.
- `ErrUnknownUser` (`repo.go:16`) falls through to `500 internal_error`. Only
  reachable when a valid session outlives its `users` row, so it is a corner,
  but `404`/`401` would describe it better than "internal".
- `validate` accepts a single answer, so a learner can submit one C1 item and be
  graded high. Self-inflicted only, and `TestAssessAcceptsAPartialAnswerSet`
  shows the partial set is deliberate.

## Bugs filed
All five are non-blocking. Nothing here requires a fix before this branch merges.

- `harness/ideas/_inbox/two-concurrent-assessment-submits-double-spend-the-ai-and-or.md`
  — **medium.** Check-then-act idempotency with no unique constraint on the
  active roadmap; the invariant survives only because `UPDATE users` happens to
  run first in the transaction. Costs a duplicate roadmap generation and leaves
  84 orphan exercise rows; the winner's `roadmap_id` is already inactive.
- `harness/ideas/_inbox/a-re-submitted-assessment-silently-discards-target-goal-noti.md`
  — **medium.** A repeat submit skips all four `users` column writes and still
  answers `status: "success"`; `timezone` drives `quests`' `day_number` and
  `pet`'s sweep, and no other writer for it exists yet.
- `harness/ideas/_inbox/no-post-handler-bounds-the-request-body-so-one-jwt-can-decod.md`
  — **medium, pre-existing and repo-wide.** No `MaxBytesReader` and no server
  read/write timeouts; onboarding's `[]Answer` is the first unbounded array DTO.
- `harness/ideas/_inbox/onboarding-service-stores-a-now-clock-it-never-uses-so-every.md`
  — **low.** `Service.now` is assigned and never read; `main.go` and both test
  files pass a clock that controls nothing.
- `harness/ideas/_inbox/get-onboarding-quiz-is-shipped-but-absent-from-backend-spec-.md`
  — **low, spec bug.** The endpoint is necessary but missing from backend §6.1
  and 1st-thinking §7; the plan asked for this to be filed if the spec was not
  updated.

## Verdict
**pass-with-bugs.** The plan delivered the idea, the code delivered the plan,
and the two things this slice was riskiest on — transactional atomicity across
85 rows, and not holding a database transaction across an AI call — are both
right. Shape validation of the model's JSON and rejection of out-of-enum CEFR
levels before the write are right too. Five bugs filed: three medium, two low,
**none blocking**. `python3 tools/harness/cli.py blockers --plan <plan>` exits 0.

No PR to mark ready (`gh pr create` is 403 for this account, as expected).
Merge command for the human: `/harness merge harness/plans/2026-09-22-onboarding-placement-test-cefr-grading-and-roadmap-generatio.md`
