---
idea: harness/ideas/2026-09-22-run-01/spaced-repetition-vocabulary-review-in-the-daily-quest.md
status: draft
priority: low
merged: false
---
# Spaced-repetition vocabulary review (backend): `vocab_reviews`, `reviews[]` on `GET /quests/daily`, `review_grades[]` on `POST /quests/progress` — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Feature team — ticket **F2** of 2026-09-27. **Estimate:** 5 h. **Branch:** `harness/2026-09-27-low-spaced-repetition-vocabulary-review-in-the-daily-quest`.

**Idea:** `harness/ideas/2026-09-22-run-01/spaced-repetition-vocabulary-review-in-the-daily-quest.md` — **backend half**. The review deck UI (up to 8 cards at the top of the Vocabulary task, the "N reviews due" line, the "words retained" count) is a separate plan written after a design doc from the designer role. Backend only; **no design doc**.

**Depends on (must be on `origin/main` before execution):** the 2026-09-26 daily code PR — specifically branches `harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a` (the `airouter.VocabularyContent` / `airouter.Word{Term, Definition, Example}` contract this plan seeds from), `harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile`, `harness/2026-09-25-medium-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-` and `harness/2026-09-26-medium-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-` (all edit `internal/quests`) — if any is missing when you start, stop and report. Check with `git log --oneline origin/main | grep -c -E 'typed task content|volatile|pet_health|roadmap tree'` or `git branch -r --merged origin/main | grep <branch>`.

**Goal:** Every vocabulary word a learner completes becomes a review card that comes back on an expanding SM-2 schedule: `GET /api/v1/quests/daily` lists up to 8 cards due today (in the learner's timezone) and `POST /api/v1/quests/progress` accepts their grades and reschedules them — nothing else on the two endpoints changes, no new endpoint, no AI call.

**Architecture:** `quests` owns the new table (a `ReviewRepo` interface beside `QuestRepo`/`ProgressRepo`, implemented by the same `PgRepo`) and the pure scheduler (`srs.go`: `NextReview(state, grade, today) ReviewState`). Seeding hangs off the existing `RecordProgress` write sequence: `CheckExercise` is widened to return the `Exercise` row (same query, same call-log entry), and after `MarkComplete` a `task_type = 'vocabulary'` exercise's `content_json.words[]` is decoded with `airouter.VocabularyContent` (packages share exported types and functions, never tables — `onboarding` and `quests`' integration test already import `airouter`) and inserted `ON CONFLICT DO NOTHING`. `Daily` gains one read (`DueReviews`), `RecordProgress` gains one validated, transactional write (`ApplyGrades`). `store` gains the migration and the spec §3.2 block.

**Wire contract (additive; backend spec §6.2 — added by Task 5):**
```
GET /api/v1/quests/daily   … + "reviews": [ {"item_key": "<exercise_id>:<term>", "term": "Inquire", "definition": "To ask for information", "example": "…"} ]   (≤ 8, due_on <= today in users.timezone, ordered due_on ASC then item_key ASC; [] when none — never null)
POST /api/v1/quests/progress   {"exercise_id", "duration_seconds", "user_answers"?, "review_grades"?: [ {"item_key": "…", "grade": "again"|"hard"|"good"|"easy"} ]}
   400 invalid_request  — any grade outside the four values, a blank/duplicate item_key, more than MaxReviewGrades (32) entries, or an item_key the caller does not own (checked before any write)
   200 body unchanged (§6.2)
```

**Scheduling arithmetic (`srs.go`; `TestNextReview*` pins every line):** a card is `ReviewState{IntervalDays int, Ease float64, Reps int, DueOn string}`; seeded as `{1, 2.50, 0, today+1}`. Grade `q` ∈ {again, hard, good, easy}; `today` = the learner's local date (`LocalDate(now, loc)`).
1. `ease' = max(1.30, round2(ease + Δ))` with Δ = again **−0.20**, hard **−0.15**, good **0.00**, easy **+0.15** (`round2` = 2 decimals, what `NUMERIC(4,2)` stores).
2. `again`: `reps' = 0`, `interval' = 1`.
3. otherwise `reps' = reps + 1`; `interval'` = **1** when `reps' == 1`, **6** when `reps' == 2`, else `round(interval × ease')` for good, `max(interval + 1, round(interval × 1.20))` for hard, `round(interval × ease' × 1.30)` for easy; always `≥ 1` and capped at `MaxIntervalDays = 365`.
4. `due_on' = today + interval'` calendar days (`time.Date` on the parsed local date, `AddDate(0,0,n)`, no timezone math — dates only); `last_grade` stores the SM-2 quality `again=0, hard=3, good=4, easy=5`.
Worked row (good, good, good from seed): reps 0→1 interval 1; →2 interval 6; →3 interval round(6×2.5)=15. `again` after that: reps 0, interval 1, ease 2.30.

## Global Constraints
- Work in `.worktrees/<slug>`; Go from `backend/`; `rg`/`timeout` not installed (`grep -n`, `go test -timeout`); integration tests via `COMPOSE_PROJECT_NAME=<slug> make up` … `make down` with exported `TEST_DATABASE_URL`/`TEST_REDIS_URL`.
- **Validate before write** (quests' standing rule): grade shape and item ownership are checked with reads before the INCRBY; grades are applied in **one transaction** after `MarkComplete`, and a failure there returns 500 like a `MarkComplete` failure does (the client may retry; re-grading is not idempotent — document in CODEMAP).
- Seeding is **best-effort and idempotent**: a failed seed is logged, never returned (the study session is already recorded); `ON CONFLICT (user_id, item_key) DO NOTHING`. Malformed `content_json` (pre-typed-content roadmaps) seeds nothing and is not an error.
- Migration number = **the next free number when you start** (expected `0005` or `0006` — `0004_pet_shields` and `0004_rls` are already taken by unmerged branches); `ls backend/internal/store/migrations/` first. The up file ends with `ALTER TABLE vocab_reviews ENABLE ROW LEVEL SECURITY;` (the `0004_rls` convention, pinned by `TestEveryTableCreatedByAMigrationHasRLS` once merged). The backend spec §3.2 block gets the **identical** DDL appended (AGENTS.md rule), and `internal/store/integration_test.go`'s table list gains `vocab_reviews`.
- `NewService` gains a `reviews ReviewRepo` parameter; update every caller (`grep -n 'NewService(' internal/quests/*_test.go cmd/api/main.go`). No `nil` fallback — `PgRepo` implements it.
- `gofmt -l internal/quests internal/store cmd/api` empty; `go vet ./...` clean.

## Review Focus
1. `item_key` is built by one exported function `ReviewItemKey(exerciseID, term)` and the same normalisation is what grades are matched on; a term with leading spaces / capitals / double spaces yields the same key (table test).
2. Day boundary: a learner in `Asia/Saigon` at 23:30 local on D does **not** see a card due D+1; at 00:30 on D+1 they do; a learner in `America/Los_Angeles` with the same UTC instant sees the opposite (fixed `now`, `timezone` on the fake). `date` in the response and the due cut-off use the same `LocalDate`.
3. Cap: 9 due cards → 8 returned, oldest `due_on` first, ties by `item_key`; the 9th appears the next day only if still due (it is — nothing is dropped).
4. Grades for an `item_key` the user does not own (another user's, or unknown) → 400 before INCRBY (call log shows no `INCRBY`).
5. Seed happens only for `task_type = 'vocabulary'`, only after `MarkComplete`, and a second progress call for the same exercise inserts 0 rows (integration test counts rows).
6. `RecordProgress`'s existing pinned order (INCRBY → EXPIRE → UPSERT → ON TARGET MET → MARK TARGET MET → MARK COMPLETE) is unchanged; the new entries are `APPLY GRADES` then `SEED REVIEWS` after it.

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/store/migrations/000N_vocab_reviews.{up,down}.sql` | **New** table + index + RLS |
| `backend/internal/store/integration_test.go` | table list gains `vocab_reviews` |
| `backend/internal/quests/srs.go` + `srs_test.go` | **New**: `Grade`, `ReviewState`, `NextReview`, `ReviewItemKey`, constants |
| `backend/internal/quests/review_repo.go` | **New**: `ReviewRepo`, `ReviewItem`, `ReviewSeed`, SQL, `PgRepo` methods |
| `backend/internal/quests/repo.go` | `CheckExercise` returns `(Exercise, error)` |
| `backend/internal/quests/service.go` + `service_test.go` | `reviews` collaborator; `Daily.Reviews`; `RecordProgress(ctx, userID, exerciseID, seconds, grades)` |
| `backend/internal/quests/handler.go` + `handler_test.go` | `review_grades` binding + 400 mapping |
| `backend/internal/quests/fakes_test.go` | `fakeReviewRepo`; `CheckExercise` fake returns the row |
| `backend/internal/quests/integration_test.go` | seed / due / grade round trip against Postgres |
| `backend/cmd/api/main.go` | `NewService(…, questRepo, …)` |
| Backend spec §3.2 + §6.2, `harness/CODEMAP.md` (`quests`, `store`) | contract |

## Tasks

### Task 1: Migration and the spec DDL

**Files:** `backend/internal/store/migrations/000N_vocab_reviews.up.sql`, `…down.sql`, `backend/internal/store/integration_test.go`, backend spec §3.2 (`grep -n 'Added by migration' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md"` — append after the last block, inside the same ```sql fence).

- [ ] **Step 1 (tests first):** `internal/store/integration_test.go` — add `"vocab_reviews"` to the table list at the `for _, table := range []string{…}` loop (line ~94) so the existence check fails until the migration exists; if `migrations_test.go` pins down-file ordering, add the new file's `DROP TABLE IF EXISTS vocab_reviews;` expectation there too.
- [ ] **Step 2:** Up migration (comment header in the `0003` style, then exactly):
```sql
CREATE TABLE vocab_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exercise_id UUID NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    item_key TEXT NOT NULL,
    term TEXT NOT NULL,
    definition TEXT NOT NULL,
    example TEXT NOT NULL DEFAULT '',
    interval_days INT NOT NULL DEFAULT 1 CHECK (interval_days >= 1),
    ease NUMERIC(4,2) NOT NULL DEFAULT 2.50 CHECK (ease >= 1.30),
    due_on DATE NOT NULL,
    reps INT NOT NULL DEFAULT 0 CHECK (reps >= 0),
    last_grade SMALLINT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, item_key)
);
CREATE INDEX vocab_reviews_due_idx ON vocab_reviews (user_id, due_on);
ALTER TABLE vocab_reviews ENABLE ROW LEVEL SECURITY;
```
Down: `DROP TABLE IF EXISTS vocab_reviews;`. Spec §3.2: `-- Added by migration 000N (spaced-repetition review): one card per (user, vocabulary word), SM-2 state.` + the same DDL byte for byte.
- [ ] **Step 3:** `go test -timeout 60s ./internal/store -count=1` (pure tests) and, with the stack up, `go test -timeout 120s ./internal/store -run Integration -count=1`. `diff <(sed -n '/CREATE TABLE vocab_reviews/,/ENABLE ROW LEVEL SECURITY;/p' internal/store/migrations/000N_vocab_reviews.up.sql) <(sed -n '/CREATE TABLE vocab_reviews/,/ENABLE ROW LEVEL SECURITY;/p' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md")` prints nothing. Commit: `store: vocab_reviews table (migration 000N) and the spec DDL`.

### Task 2: The scheduler and the item key (pure)

**Files:** `backend/internal/quests/srs.go`, `backend/internal/quests/srs_test.go`.

**Interfaces (produces):**
```go
type Grade string
const ( GradeAgain Grade = "again"; GradeHard Grade = "hard"; GradeGood Grade = "good"; GradeEasy Grade = "easy" )
func ParseGrade(s string) (Grade, bool)          // exact, lower-case only
func (g Grade) Quality() int16                    // 0, 3, 4, 5
const ( InitialEase = 2.50; MinEase = 1.30; MaxIntervalDays = 365; MaxReviewsPerDay = 8; MaxReviewGrades = 32 )
type ReviewState struct { IntervalDays int; Ease float64; Reps int; DueOn string; LastGrade *int16 }
func SeedState(today string) ReviewState          // {1, 2.50, 0, today+1, nil}
func NextReview(cur ReviewState, g Grade, today string) ReviewState
func ReviewItemKey(exerciseID, term string) string // exerciseID + ":" + strings.Join(strings.Fields(strings.ToLower(term)), " ")
func addDays(date string, n int) string           // "2006-01-02" + n calendar days, no zone
```

- [ ] **Step 1 (tests first):** `srs_test.go` — `TestNextReviewFollowsTheTable`: a table of (state, grade, today) → want, covering at least: seed+good → `{1, 2.50, 1, today+1}`; reps1+good → `{6, 2.50, 2, today+6}`; reps2 interval6+good → `{15, 2.50, 3, today+15}`; reps2 interval6+easy → `{round(6×2.65×1.3)=21, 2.65, 3}`; reps2 interval6+hard → `{max(7, round(6×1.2)=7)=7, 2.35, 3}`; reps5 interval60+again → `{1, 2.30, 0, today+1}`; ease at 1.30+again stays `1.30`; interval 300 ease 2.50+good → capped `365`; `TestNextReviewEaseIsTwoDecimals`; `TestAddDaysCrossesMonthAndYear` (`2026-12-30`+3 → `2027-01-02`); `TestReviewItemKeyNormalises` (`"  Inquire "`, `"inquire"`, `"IN QUIRE"` vs `"in  quire"` → first three equal? no: `"IN QUIRE"` == `"in  quire"` == `"in quire"`, and both ≠ `"inquire"` — pin exactly); `TestParseGradeRejects` (`"Good"`, `""`, `"ok"`).
- [ ] **Step 2:** Implement `srs.go` per the arithmetic block above, `math.Round` for every rounding, `round2 := func(x float64) float64 { return math.Round(x*100) / 100 }`.
- [ ] **Step 3:** `go test -timeout 60s ./internal/quests -run 'NextReview|AddDays|ReviewItemKey|ParseGrade' -v`. Commit: `quests: SM-2 review scheduler and item key`.

### Task 3: `ReviewRepo` and the Postgres implementation

**Files:** `backend/internal/quests/review_repo.go`, `backend/internal/quests/repo.go` (`CheckExercise` signature), `backend/internal/quests/fakes_test.go`, `backend/internal/quests/integration_test.go`.

**Interfaces (produces):**
```go
type ReviewItem struct { ItemKey string `json:"item_key"`; Term string `json:"term"`; Definition string `json:"definition"`; Example string `json:"example"` }
type ReviewSeed struct { ExerciseID, Term, Definition, Example string }   // ItemKey derived by the repo via ReviewItemKey
type ReviewRepo interface {
    SeedReviews(ctx, userID string, seeds []ReviewSeed, st ReviewState) (inserted int, err error)          // INSERT … ON CONFLICT (user_id, item_key) DO NOTHING, one pgx.Batch
    DueReviews(ctx, userID, localDate string, limit int) ([]ReviewItem, error)                            // due_on <= $2::date ORDER BY due_on, item_key LIMIT $3; never nil
    ReviewStates(ctx, userID string, itemKeys []string) (map[string]ReviewState, error)                    // WHERE user_id=$1 AND item_key = ANY($2); missing keys absent
    ApplyGrades(ctx, userID string, updates map[string]ReviewState) error                                   // one tx; UPDATE … SET interval_days, ease, due_on, reps, last_grade, updated_at = now() WHERE user_id AND item_key; rows affected must equal len(updates) else rollback + error
}
// repo.go: CheckExercise(ctx, roadmapID, exerciseID string, day int) (Exercise, error)   // same query, now SELECT id, day_number, task_type, content_json, is_completed
```

- [ ] **Step 1 (tests first):** `fakes_test.go` — `fakeReviewRepo{log *callLog; cards map[string]ReviewState; items map[string]ReviewItem; seedErr, applyErr error}` logging `SEED REVIEWS`, `DUE REVIEWS`, `REVIEW STATES`, `APPLY GRADES`; `fakeQuestRepo.CheckExercise` returns the matching `Exercise` from `f.exercises[day]` (or `ErrExerciseNotFound`), still logging `CHECK EXERCISE`. `integration_test.go` — extend `TestIntegrationDailyAndProgressAgainstRealServices` (or add `TestIntegrationVocabReviewsRoundTrip` reusing its setup: `store.Migrate`, `onboarding.NewPgRepo(pg.Pool).SaveAssessment` with a roadmap whose vocabulary tasks carry `airouter.SampleContent("vocabulary")` — replace the `json.RawMessage(`{}`)` content for `tt == "vocabulary"`): seed 5 words for the user → `SeedReviews` returns 5, again → 0; `DueReviews(today)` empty, `DueReviews(today+1)` 5 in `item_key` order; `ApplyGrades` on 2 keys → `ReviewStates` shows the new interval/ease/due_on; `ApplyGrades` with a key of another user → error and the tx leaves the first user's rows unchanged; `DueReviews` with 9 due rows and limit 8 → 8. Gate on `TEST_DATABASE_URL`+`TEST_REDIS_URL` like the existing test.
- [ ] **Step 2:** `review_repo.go` with the SQL as constants beside the existing ones; `ReviewStates` scans `ease` into `float64` via `pgtype.Numeric` → `Float64Value()` (or `SELECT ease::float8`); `due_on` scanned as `time.Time` and formatted `2006-01-02`. `PgRepo` gets the four methods and `var _ ReviewRepo = (*PgRepo)(nil)`.
- [ ] **Step 3:** `go build ./... && go test -timeout 60s ./internal/quests -count=1` (pure) then with the stack up `go test -timeout 300s ./internal/quests -run Integration -p 1 -count=1 -v`. Commit: `quests: ReviewRepo over vocab_reviews; CheckExercise returns the row`.

### Task 4: `Daily.reviews[]`, seeding and grading in the service and handler

**Files:** `backend/internal/quests/service.go`, `service_test.go`, `handler.go`, `handler_test.go`, `backend/cmd/api/main.go`.

- [ ] **Step 1 (tests first, service):** `service_test.go` —
  - `TestDailyListsDueReviewsCappedAndOrdered`: fake has 9 cards due today (mixed `due_on` ≤ today and one due tomorrow) → `Reviews` has 8, sorted by `due_on` then `item_key`, tomorrow's absent; none due → `Reviews` is `[]` not nil (JSON `[]`).
  - `TestDailyReviewDayBoundaryFollowsTheTimezone` (Review Focus 2): `now = 2026-10-01T16:30:00Z`; `Asia/Saigon` → local 23:30 on 10-01, a card `due_on = 2026-10-02` absent; `now = 2026-10-01T17:30:00Z` → local 00:30 on 10-02, present; `America/Los_Angeles` at the second instant (10-01 10:30) → absent.
  - `TestRecordProgressSeedsVocabularyOnceAfterMarkComplete` (Review Focus 5/6): vocabulary exercise with `airouter.SampleContent("vocabulary")` → call log ends `… MARK COMPLETE, SEED REVIEWS`, 5 seeds with `ItemKey`-able terms and `SeedState(today)`; a `reading` exercise → no `SEED REVIEWS`; malformed content → no seeds, no error; `seedErr` set → result still returned, error logged.
  - `TestRecordProgressAppliesGradesInOrder`: two owned keys with grades good/again → `REVIEW STATES` before `INCRBY`, `APPLY GRADES` right after `MARK COMPLETE` and before `SEED REVIEWS`; the fake's cards now hold `NextReview(...)` values.
  - `TestRecordProgressRejectsBadGradesBeforeWriting` (Review Focus 4): unknown key / unowned key / duplicate key / `"Good"` / 33 grades → `ErrInvalidReviewGrades`, and the call log contains no `INCRBY`.
  - Update the existing pinned-order test to the new `RecordProgress(ctx, userID, exerciseID, seconds, nil)` signature and assert the order is unchanged when `grades == nil` and the exercise is not vocabulary.
- [ ] **Step 2 (tests first, handler):** `handler_test.go` — `review_grades` decoded and passed through (fake service is the real `Service` over fakes as today); `{"grade":"meh"}` → `400 {"error":"invalid_request"}`; body without `review_grades` still 200; `GET /quests/daily` body has `"reviews":[]` when none.
- [ ] **Step 3:** `service.go`: `var ErrInvalidReviewGrades = errors.New("quests: invalid review_grades")`; `type ReviewGrade struct{ ItemKey string; Grade Grade }`; `Service.reviews ReviewRepo` (constructor param after `progress`); `DailySuite.Reviews []ReviewItem \`json:"reviews"\``; `Daily` calls `DueReviews(ctx, userID, date, MaxReviewsPerDay)`. `RecordProgress(ctx, userID, exerciseID string, seconds int64, grades []ReviewGrade)`: after `CheckExercise` (now `exercise, err :=`) and before the counter read, `validateGrades` (len ≤ `MaxReviewGrades`, non-blank unique keys, parsed grades) then `ReviewStates` and reject when any key is missing; after `MarkComplete`: `if len(grades) > 0 { ApplyGrades(ctx, userID, next) }` (error → return), then `if exercise.TaskType == "vocabulary" { seedVocabulary(ctx, userID, exercise, date) }` best-effort. `seedVocabulary` decodes `airouter.VocabularyContent`, skips words with a blank term, builds `[]ReviewSeed`, calls `SeedReviews(…, SeedState(date))`, logs `quests: seeded %d review cards for user %s exercise %s` at info and the error at warn.
- [ ] **Step 4:** `handler.go`: `progressRequest.ReviewGrades []struct{ ItemKey string \`json:"item_key"\`; Grade string \`json:"grade"\` } \`json:"review_grades"\`` → `[]ReviewGrade` via `ParseGrade` (a parse failure is `400 invalid_request` in the handler, before the service); `ErrInvalidReviewGrades` → `400 invalid_request` in the `switch`. `main.go`: `quests.NewService(studyCounter, questRepo, questRepo, questRepo, pet.NewQuestHook(petSvc), time.Now)`.
- [ ] **Step 5:** `go build ./... && go test -timeout 120s ./internal/quests ./cmd/api -count=1 -race`. Commit: `quests: reviews[] on GET /quests/daily, review_grades[] on POST /quests/progress, vocabulary seeding`.

### Task 5: Contract and CODEMAP

**Files:** backend spec §6.2, `harness/CODEMAP.md` (`quests`, `store`).

- [ ] **Step 1:** §6.2 — after the `GET /api/v1/quests/daily` response JSON add a bullet `reviews[] (additive, spaced-repetition review 2026-09-27): up to 8 vocabulary cards due on the learner's local date, ordered oldest due first; {item_key, term, definition, example}` and extend the example JSON with one `reviews` entry; after the `POST /api/v1/quests/progress` request body add `review_grades[] (additive, optional): [{item_key, grade: again|hard|good|easy}] — graded cards are rescheduled (SM-2: ease 2.50 start, 1.30 floor, intervals 1 → 6 → ×ease, again resets); an unknown item_key or bad grade is 400 invalid_request before any write` and extend the example. Backslash-escape the way neighbouring lines do (`\_`, `\-`).
- [ ] **Step 2:** CODEMAP `quests` bullet: `vocab_reviews` ownership, `ReviewItemKey`, seeding rule (every accepted progress call on a vocabulary exercise, after MARK COMPLETE, idempotent, best-effort), the grade path (validated before INCRBY, applied in one tx after MARK COMPLETE, not idempotent on client retry), cap 8, day boundary via `LocalDate`, the new pinned order; `store` bullet: migration `000N_vocab_reviews` and the RLS line. No new endpoint, so §7 and 1st-thinking §7 are untouched.
- [ ] **Step 3:** `grep -n 'review_grades\|"reviews"' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md" ../harness/CODEMAP.md` shows both. Commit: `docs: §6.2 reviews[]/review_grades[], CODEMAP quests+store`.

## Verification
```
cd backend && ls internal/store/migrations/ | grep vocab_reviews
go build ./... && gofmt -l . && go vet ./... && go test -timeout 120s ./... -count=1 -race
COMPOSE_PROJECT_NAME=<slug> make up && go test -timeout 300s ./... -run Integration -p 1 -count=1 && make down
diff <(sed -n '/CREATE TABLE vocab_reviews/,/ENABLE ROW LEVEL SECURITY;/p' internal/store/migrations/000N_vocab_reviews.up.sql) <(sed -n '/CREATE TABLE vocab_reviews/,/ENABLE ROW LEVEL SECURITY;/p' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md")
grep -n 'reviews\|review_grades' internal/quests/service.go internal/quests/handler.go "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md" ../harness/CODEMAP.md
grep -n 'NewService(' cmd/api/main.go   # four repos: counter, questRepo ×3, hook, clock
git push -u origin harness/2026-09-27-low-spaced-repetition-vocabulary-review-in-the-daily-quest
```
