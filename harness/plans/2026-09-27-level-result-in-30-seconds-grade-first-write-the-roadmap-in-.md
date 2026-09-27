---
idea: harness/ideas/2026-09-26-run-01/level-result-in-30-seconds-grade-first-write-the-roadmap-in-.md
status: approved
priority: medium
merged: false
order: 1
design: harness/designs/level-result-in-30-seconds-grade-first-write-the-roadmap-in-.md
---
# Level result in 30 seconds: grade first, write the roadmap in the background (`202 generating`, `roadmap:gen:{user_id}`, `404 roadmap_generating`, the onboarding result step and the hub drawing state) — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Feature team — ticket **F4** of 2026-09-27. **Estimate:** 8 h. **Branch:** `harness/2026-09-27-medium-level-result-in-30-seconds-grade-first-write-the-roadmap-in-`.

**Design:** `harness/designs/level-result-in-30-seconds-grade-first-write-the-roadmap-in-.md` inside `harness/UI-KIT.md` v2. Frontend tasks (8–11) cite its sections; `## Verification` repeats its acceptance list.
**Idea:** `harness/ideas/2026-09-26-run-01/level-result-in-30-seconds-grade-first-write-the-roadmap-in-.md`

**Depends on (must be on origin/main before execution):** the 2026-09-26 daily code PR — specifically `harness/2026-09-26-high-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a` (`onboarding` `Regenerate`, `Repo.ReplaceRoadmap`, `insertActiveRoadmap` — the job's save), `harness/2026-09-26-high-a-session-a-learner-wants-to-finish-level-true-content-do-to` (`GradeFloor` inside the grading branch of `Assess`), `harness/2026-09-26-medium-name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o` (`Pet.Ensure(ctx, userID, plantName)`, `AssessmentRequest.PlantName`, `plantNameOrDefault`, the plant-name field on `/onboarding`), `harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a` (`airouter/content.go`, onboarding `fakes_test.go`), `harness/2026-09-25-medium-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-` (`quests/service.go`, `stores/quest.ts`), `harness/2026-09-26-medium-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-` (`quests/{repo,day,roadmap}.go`, `GET /roadmap` in `main.go`), `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` (`components/retro/*`, `useRetroToast`), `harness/2026-09-26-medium-growth-moment-after-every-task-health-gain-streak-and-target` and `harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days` (both rewrite `pages/index.vue`), `harness/2026-09-26-high-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve` (`utils/apiClient.ts`, `stores/auth.ts`), `harness/2026-09-25-high-task-timer-keeps-counting-through-reloads-and-background-tab` (`stores/quest.ts`) and `harness/2026-09-25-medium-settings-screen-wires-web-push-reminders-and-google-calendar` (`pages/onboarding.vue`) — if any is missing when you start, stop and report. Some older branches were squash-merged, so check **content**, not ancestry, from the repo root of a fresh `origin/main` worktree:
```
grep -q 'func (r \*PgRepo) ReplaceRoadmap' backend/internal/onboarding/repo.go && \
grep -q 'GradeFloor(req.Answers)' backend/internal/onboarding/service.go && \
grep -q 'plantNameOrDefault' backend/internal/onboarding/service.go && \
test -f backend/internal/airouter/content.go && \
grep -q 'pet_health?: number' frontend/stores/quest.ts && \
test -f backend/internal/quests/roadmap.go && \
test -f frontend/components/retro/RetroPanel.vue && \
test -f frontend/composables/useGrowthMoment.ts && \
test -f frontend/components/plant/ShieldRow.vue && \
grep -q 'onRenew' frontend/utils/apiClient.ts && \
grep -q 'TIMER_STORAGE_KEY' frontend/stores/quest.ts && \
test -f frontend/stores/settings.ts && echo GATE-OK
```
(`harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile` was checked on 2026-09-27 and its content is already on `main` — no gate needed.)

**Goal:** `POST /api/v1/onboarding/assessment` answers as soon as the placement is graded — `202 {status:"generating", assessed_level, pet_state}` with the level, goal, reminder and pet already saved — and writes the 28-day roadmap in an in-process background job guarded by `roadmap:gen:{user_id}`, while `GET /quests/daily` answers `404 roadmap_generating` until it lands; the PWA shows the level and the sprout at ~30 s and a hub "map being drawn" state that survives reloads, polls, reveals day 1, and retries a failed job without re-grading.

**Architecture:** `onboarding.Service.Assess` keeps its first half (validate → active-roadmap 200 → limiter → staged-level-or-grade with `GradeFloor`) and replaces the synchronous roadmap step with: acquire the job key → `Repo.SaveProfile` (the `users` columns) → `Pet.Ensure` → `Runner.Go(job)` → 202. The job (`Service.generate`) runs under its own context derived from the server's root context (not the request's) with an overall cap `RoadmapJobTimeout = 2 × airouter.TaskTimeout(TaskRoadmapGen)` = 360 s, under which `routeJSON`'s per-call 180 s budgets nest; it saves through the existing `Repo.ReplaceRoadmap` (one transaction: `cefr_current`, deactivate, insert roadmap + 84 exercises), clears the quiz hash and deletes the key — or sets it `failed`. The key's Redis implementation lives in `store` (`store.RoadmapJobs`, next to the key builder) so `onboarding` (writer) and `quests` (reader) each declare their own one-method interface over it and never import each other. `quests.Service.Daily` consults the job state only on the no-roadmap branch. The frontend adds `ApiError.body`, `quest.roadmapJob`, a persisted `aelp.assessment` record, `usePollRoadmap`, the `DrawingBar` and `RankPlate` kit additions, and the new onboarding and hub states.

**Tech stack:** Go 1.25, Gin, pgx/v5, go-redis/v9 (`redis.NewScript` for the atomic acquire); Nuxt 3, Pinia, Vitest (`vi.useFakeTimers`, happy-dom), `@vue/test-utils`.

**Wire contract (additive; backend spec §6.1/§6.2, 1st-thinking §5.1 note):**
```
POST /api/v1/onboarding/assessment      Authorization: Bearer <JWT>
{ "target_goal", "notification_time": "HH:MM:SS", "timezone", "plant_name"?, "answers": [{question_id, selected_option}] }   (unchanged)

202  { "status": "generating", "assessed_level": "B1",
       "pet_state": { "plant_name": "Mầm Non", "health_points": 100, "stage": "sprout" } }      // no roadmap_id
     — a new grade (or a staged level reused) and the job started, OR a job for this user is already
       running (no limiter slot, no AI call, no write; assessed_level = users.cefr_current)
200  { "status": "success", "assessed_level", "roadmap_id", "pet_state" }                    // unchanged: an active roadmap exists
     (201 is no longer emitted by this route; clients keep accepting it)
400 invalid_request · 429 rate_limited · 503 ai_unavailable · 502 ai_bad_output · 502 ai_upstream_failed · 504 ai_timeout   // grading step only
500 internal_error

GET /api/v1/quests/daily
200  unchanged
404  { "error": "roadmap_generating", "status": "generating" }   // no active roadmap, roadmap:gen:{user_id} = generating
404  { "error": "roadmap_generating", "status": "failed" }       // no active roadmap, roadmap:gen:{user_id} = failed
404  { "error": "no_active_roadmap" }                            // no active roadmap, no job key (unchanged)
```
An active roadmap always wins: a stale key next to an active roadmap answers 200. `POST /quests/progress`, `GET /roadmap` and `POST /roadmaps/regenerate` are unchanged (they answer `no_active_roadmap` while a job runs).

**Job key (`store/keys.go` + `store/roadmapgen.go`; not in spec §4 — documented in CODEMAP like `pet:revive`):**
```
roadmap:gen:{user_id}   String   "generating" | "failed"   TTL RoadmapGenTTL = 6 min (= 2 × 180 s = RoadmapJobTimeout)
Acquire  (Lua, atomic): if the key is absent or "failed" → SET "generating" EX 360, return 1; else return 0
Fail:    SET "failed" EX 360        Done: DEL        State: GET ("" when absent)
```
A crashed process leaves `generating` until the TTL; the client's 420 s stale rule (design §0.4) then offers "Thử lại", which re-acquires once the key has expired or is `failed`.

## Global Constraints
- Work in `.worktrees/<slug>`; Go from `backend/`, npm from `frontend/`. `rg`/`timeout` are not installed (`grep -n`; `go test -timeout`). Integration tests: `COMPOSE_PROJECT_NAME=<slug>` in the scratch `backend/.env` with port overrides, `make up` … `make down`.
- **Backend first.** Tasks 1–7 are the backend and must be complete, committed and green (`go build ./... && gofmt -l . && go vet ./... && go test -race ./...`) before Task 8 starts. If the day runs out after Task 7, push and set the plan's outcome accordingly — the backend is shippable on its own only together with Task 9 (the old onboarding page would show the 202 as a result and then an error on the hub), so **never push a branch with Tasks 1–7 and without 8–11 as `done`**.
- `Assess` order is a contract (Review Focus 1). Nothing is written before the grade has parsed; the limiter slot is taken only when a grading or roadmap call can follow; a submit while a job runs costs nothing.
- The job never uses the request context (`c.Request.Context()` is cancelled when the 202 is written). It uses the runner's root context (the `signal.NotifyContext` context in `main.go`) plus `RoadmapJobTimeout`.
- `RoadmapSystemPrompt`, `RoadmapUserPrompt`, `ParseRoadmap`, `routeJSON`, `route` and `Regenerate` are untouched. `airouter/router.go` and `timeouts.go` are untouched.
- Error bodies: only the codes in the wire contract; the `status` field on the 404 is the only addition to any error body.
- Frontend: no `rounded-*` above `rounded-sm`, no easing, no blur in new retro markup (UI-KIT v2, design §8); every `setTimeout` and listener in `usePollRoadmap` is removed on unmount; `aelp.assessment` is read/written only through `utils/assessmentRecord.ts` with try/catch around storage.
- `gofmt -l internal cmd` empty; `go vet ./...` clean; `-race` on. `npm run lint`, `npm run typecheck`, `npm run test:unit`, `npm run build` green after every frontend task.

## Review Focus
1. `Assess` order: validate (400) → `ActiveRoadmapID` hit → 200 unchanged → `jobs.State == "generating"` → 202 from `Profile.CEFRCurrent` + `Pet.Ensure(ctx, userID, "")`, **no** `ALLOW`, no AI, no write → `limiter.Allow` (429) → `StagedLevel` hit (log "reusing") or `StageAnswers` → `TaskPlacementTest` → `GradeFloor` → `StageLevel` → `jobs.Acquire` (lost → the same 202, no job) → `SaveProfile` → `Pet.Ensure(ctx, userID, plantNameOrDefault(req.PlantName))` → `run.Go(job)` → 202 with the graded level. A `SaveProfile`/`Pet.Ensure` failure after the acquire calls `jobs.Fail` and returns the error (500). `service_test.go`'s call log pins it.
2. The job: `generate(root, userID, level, goal)` derives `context.WithTimeout(root, s.jobBudget)` (`jobBudget` defaults to `RoadmapJobTimeout`), calls `routeJSON(TaskRoadmapGen, …)` exactly as `Assess` did, then `ReplaceRoadmap(ctx, userID, level, roadmap)` → `quiz.Clear` (log on error) → `jobs.Done`; any routing/parse/save error → `jobs.Fail` under a fresh 5 s `context.Background()` deadline (the job context may be the one that expired). It logs start, elapsed and the outcome kind, never a body. Nothing is written on a failed job.
3. Retry: a submit when the key is `failed` takes one limiter slot, reuses the staged level (no `TaskPlacementTest` call), re-acquires, and runs one job. A submit while `generating` takes nothing. Two simultaneous first submits may each grade once (inside the double-tap window) but only one acquires and only one roadmap is generated — the concurrent-submit inbox bug's "at most one extra AI call".
4. Overall cap: `RoadmapJobTimeout == 2 * airouter.TaskTimeout(airouter.TaskRoadmapGen)` and `store.RoadmapGenTTL == RoadmapJobTimeout` (pinned by a test in `onboarding`, because `store` must not import `airouter`); a malformed first roadmap followed by a hanging second one ends the job at the cap with the key `failed`. The request itself is bounded by the grading step (2 × 30 s).
5. `quests.Daily`: the job state is read only after `ActiveRoadmap` returned `ErrNoActiveRoadmap`; `generating` → `ErrRoadmapGenerating`, `failed` → `ErrRoadmapFailed`, `""` → `ErrNoActiveRoadmap`, a Redis error → that error (500). With no `RoadmapJobs` wired (`WithRoadmapJobs` not called) behaviour is byte-for-byte today's — every existing `quests` test passes unchanged.
6. Frontend: `quest.roadmapJob` is set only from a 404 whose code is `roadmap_generating` (any other 404 never means "failed" — design §8); the reveal keys off the `generating → 200` transition inside one mounted hub; `submitted_at` comes from the stored record; polling never runs while hidden and never calls `POST /onboarding/assessment`.

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/store/keys.go` + `keys_test.go` | `RoadmapGenKey`, `RoadmapGenTTL`, `RoadmapGenGenerating`, `RoadmapGenFailed` |
| `backend/internal/store/roadmapgen.go` (new) + `roadmapgen_integration_test.go` (new) | `RoadmapJobs` (`State`, `Acquire`, `Fail`, `Done`) over go-redis |
| `backend/internal/onboarding/jobs.go` (new) + `jobs_test.go` (new) | `Jobs` + `Runner` interfaces, `BackgroundRunner`, `RoadmapJobTimeout` |
| `backend/internal/onboarding/types.go` | `AssessmentResult`: `Generating bool` (`json:"-"`) replaces `Created`; `RoadmapID` `omitempty` |
| `backend/internal/onboarding/repo.go` | `ProfileUpdate`, `Repo.SaveProfile`; `SaveAssessment` leaves the interface (kept on `PgRepo` for the quests integration seed) |
| `backend/internal/onboarding/service.go` + `service_test.go`, `fakes_test.go` | `NewService(…, jobs, run, now)`, the new `Assess`, `generate` |
| `backend/internal/onboarding/handler.go` + `handler_test.go` | 202 / 200 |
| `backend/internal/onboarding/integration_test.go` | `TestIntegrationAssessAnswers202ThenTheJobSavesTheRoadmap` |
| `backend/internal/quests/service.go`, `handler.go`, `service_test.go`, `handler_test.go`, `fakes_test.go` | `RoadmapJobs` interface, `WithRoadmapJobs`, `ErrRoadmapGenerating`/`ErrRoadmapFailed`, the 404 bodies |
| `backend/cmd/api/main.go`, `server.go` | wiring, `runner.Wait` after `serve`, the `newServer` comment |
| Backend spec §6.1 + §6.2, 1st-thinking §5.1, `harness/CODEMAP.md` (`store`, `onboarding`, `quests`, `cmd/api`, `shell`), `CLAUDE.md` (AI router paragraph) | docs |
| `frontend/utils/apiClient.ts` + `tests/unit/apiClient.test.ts` | `ApiError.body` |
| `frontend/utils/assessmentRecord.ts` (new) + test | `aelp.assessment` read/write/clear |
| `frontend/stores/quest.ts`, `stores/auth.ts` + tests | `roadmapJob`; `auth.setLevel`; `signOut` clears the record |
| `frontend/composables/useOnboardingApi.ts` | 202 response type |
| `frontend/components/retro/DrawingBar.vue`, `RankPlate.vue` (new) + tests | kit additions |
| `frontend/composables/usePollRoadmap.ts` (new) + test | backoff poll |
| `frontend/pages/onboarding.vue` + `tests/unit/onboardingPage.test.ts` | grading + result steps |
| `frontend/pages/index.vue` + `tests/unit/indexPage.test.ts` | drawing / failed / reveal states |

## Tasks

### Task 1: The job key in `store`

**Files:** `backend/internal/store/keys.go`, `keys_test.go`, `roadmapgen.go` (new), `roadmapgen_integration_test.go` (new).

- [ ] **Step 1 (tests first):** `keys_test.go` — `TestRoadmapGenKey`: `RoadmapGenKey("u1") == "roadmap:gen:u1"`, `RoadmapGenTTL == 6*time.Minute`, `RoadmapGenGenerating == "generating"`, `RoadmapGenFailed == "failed"`. `roadmapgen_integration_test.go`, gated on `TEST_REDIS_URL` exactly like `onboarding/quiz_integration_test.go`: `TestIntegrationRoadmapJobsLifecycle` on a random user id (cleanup `Done`) — `State` → `""`; `Acquire` → true; `State` → `generating`, `TTL` in (0, 360 s]; second `Acquire` → false; `Fail` → `State` `failed`, TTL in (0, 360 s]; `Acquire` → true (failed is re-acquirable); `Done` → `State` `""`; and `TestIntegrationRoadmapJobsAcquireIsAtomic`: 20 goroutines `Acquire` at once → exactly one true.
- [ ] **Step 2:** `go test ./internal/store -run 'RoadmapGen|RoadmapJobs' -v` — FAIL (undefined).
- [ ] **Step 3:** `keys.go`: add to the TTL block `RoadmapGenTTL = 6 * time.Minute` with the comment "= 2 × airouter.TaskTimeout(TaskRoadmapGen); onboarding pins the equality — store must not import airouter"; add the two value constants and
  ```go
  // RoadmapGenKey is roadmap:gen:{user_id} — the background roadmap job for a
  // learner who has been graded but has no roadmap yet (String: "generating" |
  // "failed"; TTL RoadmapGenTTL). Not in spec §4; added by the level-result
  // plan and documented in CODEMAP.
  func RoadmapGenKey(userID string) string { return fmt.Sprintf("roadmap:gen:%s", userID) }
  ```
  `roadmapgen.go`: `type RoadmapJobs struct{ Client *redis.Client }`, `NewRoadmapJobs(r *Redis) *RoadmapJobs`, and the four methods; `Acquire` runs
  ```lua
  local v = redis.call('GET', KEYS[1])
  if v == false or v == ARGV[2] then
    redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[3])
    return 1
  end
  return 0
  ```
  via a package-level `redis.NewScript` with `ARGV = {RoadmapGenGenerating, RoadmapGenFailed, int(RoadmapGenTTL.Seconds())}`; `State` maps `redis.Nil` to `""`; errors wrap with `store: roadmap job …`.
- [ ] **Step 4:** `go test ./internal/store -count=1 -race` PASS (integration tests skip without `TEST_REDIS_URL`; run them once under `make up`).
- [ ] **Step 5:** Commit: `store: roadmap:gen:{user_id} job key — atomic acquire, fail, done`.

### Task 2: `onboarding` job seams — `Jobs`, `Runner`, `BackgroundRunner`, `RoadmapJobTimeout`

**Files:** `backend/internal/onboarding/jobs.go` (new), `jobs_test.go` (new).

- [ ] **Step 1 (tests first):** `jobs_test.go` — `TestRoadmapJobTimeoutIsTwiceTheRoadmapBudgetAndTheKeyTTL` (`RoadmapJobTimeout == 2*airouter.TaskTimeout(airouter.TaskRoadmapGen)` and `== store.RoadmapGenTTL`); `TestBackgroundRunnerCancelsJobsWithTheRoot`: root ctx cancelled → a job blocked on `<-ctx.Done()` returns, `Wait(time.Second)` returns true; `TestBackgroundRunnerWaitIsBounded`: a job that ignores ctx → `Wait(20*time.Millisecond)` returns false within ~50 ms; `TestBackgroundRunnerRecoversAPanic`: a panicking job is logged, `Wait` returns true, the process survives.
- [ ] **Step 2:** `go test ./internal/onboarding -run 'RoadmapJobTimeout|BackgroundRunner' -v` — FAIL.
- [ ] **Step 3:** `jobs.go`:
  ```go
  // RoadmapJobTimeout caps one background roadmap job: two TaskRoadmapGen
  // attempts (routeJSON's malformed-body retry), each under its own 180 s
  // budget, nest inside it. It equals store.RoadmapGenTTL.
  var RoadmapJobTimeout = 2 * airouter.TaskTimeout(airouter.TaskRoadmapGen)

  // Jobs is onboarding's view of the roadmap:gen:{user_id} key;
  // *store.RoadmapJobs satisfies it.
  type Jobs interface {
  	State(ctx context.Context, userID string) (string, error)
  	Acquire(ctx context.Context, userID string) (bool, error)
  	Fail(ctx context.Context, userID string) error
  	Done(ctx context.Context, userID string) error
  }

  // Runner starts a job outside the request. The ctx a job receives is the
  // runner's root, never the request's.
  type Runner interface{ Go(job func(ctx context.Context)) }

  // BackgroundRunner runs jobs as goroutines under root (main's
  // signal.NotifyContext context) and lets main wait for them on shutdown.
  type BackgroundRunner struct{ root context.Context; wg sync.WaitGroup }
  func NewBackgroundRunner(root context.Context) *BackgroundRunner
  func (r *BackgroundRunner) Go(job func(ctx context.Context))   // wg.Add, go, defer wg.Done + recover/log
  func (r *BackgroundRunner) Wait(d time.Duration) bool           // true when every job returned inside d
  ```
- [ ] **Step 4:** `go test ./internal/onboarding -run 'RoadmapJobTimeout|BackgroundRunner' -race -v` PASS.
- [ ] **Step 5:** Commit: `onboarding: job seams and a background runner under the server's root context`.

### Task 3: `Repo.SaveProfile` and the 202 result type

**Files:** `backend/internal/onboarding/repo.go`, `types.go`, `fakes_test.go`, `integration_test.go`.

- [ ] **Step 1 (tests first):** `integration_test.go` (gated on `TEST_DATABASE_URL` like its neighbours) — `TestIntegrationSaveProfileWritesTheUserColumnsOnly`: seed a user, `SaveProfile(ctx, id, ProfileUpdate{CEFRLevel:"B1", TargetGoal:"Business English", Timezone:"Asia/Saigon", NotificationTime:"20:00:00"})` → the four columns set, zero `roadmaps` rows; unknown user → `ErrUnknownUser`. `fakes_test.go`: `fakeRepo.SaveProfile` records `SAVE PROFILE <level>` in the shared call log (and supports `failSaveProfile`); `fakeRepo.ReplaceRoadmap` records `REPLACE ROADMAP <level>` (add it if the 59-of-84 branch's fake does not already log it).
- [ ] **Step 2:** `go test ./internal/onboarding -run SaveProfile -v` — FAIL (undefined).
- [ ] **Step 3:** `repo.go`: `type ProfileUpdate struct{ CEFRLevel, TargetGoal, Timezone, NotificationTime string }`; `Repo` gains `SaveProfile(ctx, userID string, p ProfileUpdate) error` (the existing `updateUserSQL`, `RowsAffected == 0` → `ErrUnknownUser`) and **drops** `SaveAssessment` from the interface (the method stays on `PgRepo` — `quests`' integration test seeds through it; update its doc comment to say so). `types.go`: `AssessmentResult.RoadmapID` gets `json:"roadmap_id,omitempty"`; replace `Created bool` with `Generating bool \`json:"-"\`` (202 vs 200); update the type's comment.
- [ ] **Step 4:** `go build ./... && go vet ./internal/onboarding` (service/handler will not compile until Task 4 if they reference `Created` — do Steps 3 of Tasks 3 and 4 before running the package's tests, and commit them together if needed).
- [ ] **Step 5:** Commit: `onboarding: SaveProfile and the 202 result type`.

### Task 4: `Assess` answers after grading; `generate` runs the roadmap in the background

**Files:** `backend/internal/onboarding/service.go`, `service_test.go`, `fakes_test.go`.

- [ ] **Step 1 (tests first):** `fakes_test.go` — `fakeJobs` (in-memory state per user, `failAcquire`/`lose` switches, logs `STATE`, `ACQUIRE`, `FAIL`, `DONE`) and `manualRunner` (appends jobs; `runAll(ctx)` runs them synchronously) so a test controls when the job runs; existing fakes log into one `[]string`. `service_test.go` (update the existing tests that asserted 201/`Created` to the new contract; add):
  - `TestAssessAnswers202AfterGradingAndStartsOneJob`: scripted provider (placement OK, roadmap OK). Result `Generating: true`, `Status "generating"`, graded level, pet state, empty `RoadmapID`. Log **before** `runAll`: `ACTIVE ROADMAP, STATE, ALLOW, STAGED LEVEL, STAGE ANSWERS, ROUTE placement_test, STAGE LEVEL, ACQUIRE, SAVE PROFILE B1, PET ENSURE Mầm Non, GO` — no roadmap route yet. After `runAll`: `ROUTE roadmap_generation, REPLACE ROADMAP B1, CLEAR, DONE`.
  - `TestAssessWhileGeneratingAnswers202WithoutAI`: jobs state `generating`, profile `B2` → 202 with `B2`, pet via `Ensure(…, "")`; log has no `ALLOW`, no `ROUTE`, no `SAVE PROFILE`, no `GO`; provider called 0 times.
  - `TestAssessLostAcquireStartsNoSecondJob`: `lose` set → 202 with the graded level; no `SAVE PROFILE`, no `GO`.
  - `TestAssessActiveRoadmapStill200`: unchanged behaviour (`Generating: false`, `RoadmapID` set, no `STATE`).
  - `TestRetryAfterFailedJobReusesTheStagedLevel`: quiz fake holds a staged `B1` for these answers, jobs state `failed` → one `ALLOW`, no `ROUTE placement_test`, `ACQUIRE` true, job runs → `REPLACE ROADMAP B1`, `DONE`.
  - `TestJobMalformedTwiceMarksFailedAndWritesNothing`: roadmap scripted `"nonsense"`, `"nonsense"` → after `runAll`: `FAIL`, no `REPLACE ROADMAP`, no `DONE`, no `CLEAR` (the staged level must survive for the retry).
  - `TestJobEndsAtTheOverallCap`: `svc.jobBudget = 30 * time.Millisecond`; roadmap scripted malformed then a provider call that blocks until ctx is done → `runAll` returns within ~200 ms, log ends `FAIL`.
  - `TestJobDoesNotUseTheRequestContext`: call `Assess` with a ctx cancelled right after it returns, then `runAll(context.Background())` → `REPLACE ROADMAP`, `DONE`.
  - `TestAssessProfileSaveFailureMarksTheJobFailed`: `failSaveProfile` → error returned, log `ACQUIRE, SAVE PROFILE, FAIL`, no `GO`.
  - `TestAssessGradingFailureLeavesTheKeyAlone`: placement malformed twice → `ErrBadAIOutput`, no `ACQUIRE`, key state unchanged (a `failed` key stays `failed`).
- [ ] **Step 2:** `go test ./internal/onboarding -run 'Assess|Job|Retry' -v` — FAIL.
- [ ] **Step 3:** `service.go`: `Service` gains `jobs Jobs`, `run Runner`, `jobBudget time.Duration`; `NewService(repo, quiz, limiter, ai, pet, jobs, run, now)` sets `jobBudget = RoadmapJobTimeout`. Rewrite `Assess` in the Review Focus 1 order; the "already generating" and "lost acquire" answers share one helper `generatingResult(ctx, userID, level)` (level `""` → `Profile.CEFRCurrent`; `Pet.Ensure(ctx, userID, "")`). After a successful `Acquire`, a `SaveProfile`/`Pet.Ensure` error calls `s.failJob(userID)` and returns. Then
  ```go
  goal := strings.TrimSpace(req.TargetGoal)
  s.run.Go(func(root context.Context) { s.generate(root, userID, level, goal) })
  return AssessmentResult{Status: "generating", AssessedLevel: level, PetState: pet, Generating: true}, nil
  ```
  `generate` per Review Focus 2 (`log.Printf("onboarding: roadmap job for %s started at %s", …)` / `"… finished in %s"` / `"… failed after %s: %v"` — the error is already cut by the drivers). `failJob` uses `context.WithTimeout(context.Background(), 5*time.Second)` and logs its own error. Update the `Assess` doc comment: "answers after grading; the roadmap is a background job (see generate)". Remove the now-unused synchronous save.
- [ ] **Step 4:** `go test ./internal/onboarding -count=1 -race -v` PASS (every test, old and new).
- [ ] **Step 5:** Commit: `onboarding: answer 202 after grading and write the roadmap in a background job`.

### Task 5: Handler 202 and the onboarding integration test

**Files:** `backend/internal/onboarding/handler.go`, `handler_test.go`, `integration_test.go`.

- [ ] **Step 1 (tests first):** `handler_test.go` — a fresh graded submit → `202` and the body is exactly `{"status":"generating","assessed_level":"B1","pet_state":{…}}` (decode into `map[string]any`, assert `roadmap_id` **absent**); a submit while generating → `202`; an active roadmap → `200` with `roadmap_id`; the existing error rows unchanged. `integration_test.go` — `TestIntegrationAssessAnswers202ThenTheJobSavesTheRoadmap`, gated on `TEST_DATABASE_URL` **and** `TEST_REDIS_URL`: real `PgRepo`, `RedisQuizStore`, `store.RoadmapJobs`, a scripted provider whose roadmap call waits on a `release` channel, `NewBackgroundRunner(ctx)`. After `Assess`: `users.cefr_current` = the level, no active roadmap, `State` = `generating`; a second `Assess` → 202 and the provider saw no second call; `close(release)`; poll ≤ 5 s until `State == ""` → one active roadmap with 84 exercises, `quiz:placement:{user}` gone.
- [ ] **Step 2:** `go test ./internal/onboarding -run 'Handler|Integration' -v` — FAIL on the status code.
- [ ] **Step 3:** `AssessmentHandler`: `case out.Generating: c.JSON(http.StatusAccepted, out)`, `default: c.JSON(http.StatusOK, out)`; the error switch is unchanged. Update the doc comment (202 / 200).
- [ ] **Step 4:** `go test ./internal/onboarding -count=1 -race` PASS; under `make up`: `go test -timeout 300s ./internal/onboarding ./internal/store -run Integration -p 1 -count=1` PASS.
- [ ] **Step 5:** Commit: `onboarding: POST /onboarding/assessment answers 202 generating`.

### Task 6: `GET /quests/daily` → `404 roadmap_generating`

**Files:** `backend/internal/quests/service.go`, `handler.go`, `service_test.go`, `handler_test.go`, `fakes_test.go`.

- [ ] **Step 1 (tests first):** `fakes_test.go` — `fakeRoadmapJobs{state string; err error}`. `service_test.go`: `TestDailyWhileGeneratingIsErrRoadmapGenerating`, `TestDailyAfterAFailedJobIsErrRoadmapFailed`, `TestDailyWithNoJobKeyIsStillNoActiveRoadmap`, `TestDailyWithoutJobsWiredIsUnchanged` (no `WithRoadmapJobs` → `ErrNoActiveRoadmap`), `TestDailyActiveRoadmapWinsOverAStaleKey` (state `generating` + an active roadmap → the suite; the fake records that `State` was never called), `TestDailyJobStateErrorIsReturned`. `handler_test.go`: the three 404 bodies exactly (`{"error":"roadmap_generating","status":"generating"}`, `…"failed"}`, `{"error":"no_active_roadmap"}`), a job-state error → 500 `internal_error`.
- [ ] **Step 2:** `go test ./internal/quests -run 'Daily' -v` — FAIL.
- [ ] **Step 3:** `service.go`:
  ```go
  // ErrRoadmapGenerating / ErrRoadmapFailed: no active roadmap yet, and the
  // onboarding background job is running / has failed (roadmap:gen:{user_id}).
  var (
  	ErrRoadmapGenerating = errors.New("quests: roadmap is being generated")
  	ErrRoadmapFailed     = errors.New("quests: roadmap generation failed")
  )

  // RoadmapJobs reads roadmap:gen:{user_id}; *store.RoadmapJobs satisfies it.
  type RoadmapJobs interface {
  	State(ctx context.Context, userID string) (string, error)
  }

  // WithRoadmapJobs lets Daily tell "being generated" from "never onboarded".
  // Without it Daily answers ErrNoActiveRoadmap as before.
  func (s *Service) WithRoadmapJobs(j RoadmapJobs) *Service { s.jobs = j; return s }
  ```
  In `Daily`, only when `ActiveRoadmap` returns `ErrNoActiveRoadmap` and `s.jobs != nil`: map `store.RoadmapGenGenerating` / `store.RoadmapGenFailed` / `""` / error per Review Focus 5. `RecordProgress` untouched. `handler.go` `DailyHandler`: `ErrRoadmapGenerating` → `404 {"error":"roadmap_generating","status":"generating"}`, `ErrRoadmapFailed` → `404 {"error":"roadmap_generating","status":"failed"}`, before the `ErrNoActiveRoadmap` case.
- [ ] **Step 4:** `go test ./internal/quests -count=1 -race` PASS — the existing call-log tests unchanged.
- [ ] **Step 5:** Commit: `quests: GET /quests/daily answers 404 roadmap_generating while the job runs`.

### Task 7: Wiring, shutdown, specs, CODEMAP

**Files:** `backend/cmd/api/main.go`, `backend/cmd/api/server.go`, backend spec, 1st-thinking doc, `harness/CODEMAP.md`, `CLAUDE.md`.

- [ ] **Step 1:** `main.go`: `roadmapJobs := store.NewRoadmapJobs(rdb)`; `jobRunner := onboarding.NewBackgroundRunner(ctx)` (the `signal.NotifyContext` ctx); `onboarding.NewService(…, petForOnboarding{petSvc}, roadmapJobs, jobRunner, nil)` (keep the argument order of the post-merge call and insert the two before `now`); `questSvc.WithRoadmapJobs(roadmapJobs)` right after `quests.NewService(…)`; after `serve(…)` returns and before `log.Printf("shutdown complete")`: `if !jobRunner.Wait(3 * time.Second) { log.Printf("shutdown: roadmap jobs still running; their keys expire in %s", store.RoadmapGenTTL) }`. `server.go`: rewrite the `newServer` comment's onboarding clause — "an onboarding assessment now answers after grading (≤ 2 × 30 s); the roadmap runs in a background job capped at onboarding.RoadmapJobTimeout (360 s), outside any request".
- [ ] **Step 2:** Backend spec §6.1 (`grep -n 'onboarding/assessment' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md"`): after the existing 201 response add "**202 Accepted** (added by the level-result plan, 2026-09-27)" with the 202 block and one paragraph: the roadmap is written by a background job; `roadmap:gen:{user_id}` (String `generating`|`failed`, TTL 6 min) guards it; a submit while it runs answers 202 without an AI call; a retry after `failed` reuses the staged level; 201 is no longer emitted. §6.2 under `GET /api/v1/quests/daily`: the two `404 roadmap_generating` bodies. 1st-thinking §5.1, after the sequence diagram: an escaped-style note "\*\*Note (2026-09-27):\*\* step 8 (init dashboard) now precedes the roadmap — the assessment answers 202 after grading and steps 5–6 run as an in-process background job; the dashboard shows the map being drawn until `GET /quests/daily` answers 200."
- [ ] **Step 3:** `harness/CODEMAP.md`: `store` bullet — `RoadmapGenKey`/`RoadmapGenTTL` and `store.RoadmapJobs` (the second key not in spec §4, beside `PetReviveKey`); `onboarding` bullet — rewrite the order sentence (Review Focus 1 + the job, `RoadmapJobTimeout`, `BackgroundRunner`, 202, "bounded by 2 × 30 s per request and 360 s per job" replacing "2 × 30 s + 2 × 180 s"); `quests` bullet — the `roadmap_generating` 404 and `WithRoadmapJobs`; `cmd/api` bullet — `jobRunner.Wait(3 s)` after `serve`. `CLAUDE.md` AI-router paragraph: one sentence — "Onboarding answers `202` after grading; the roadmap is generated in an in-process background job capped at `onboarding.RoadmapJobTimeout` (2 × 180 s), guarded by `roadmap:gen:{user_id}`."
- [ ] **Step 4:** `go build ./... && gofmt -l . && go vet ./... && go test -timeout 180s ./... -count=1 -race` PASS; `grep -n 'roadmap_generating' internal/quests/handler.go "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md" ../harness/CODEMAP.md` — ≥ 3 hits.
- [ ] **Step 5:** Commit: `api: wire the roadmap job; specs, CODEMAP and CLAUDE.md for the 202 onboarding`. **Backend complete — it must be green here before Task 8.**

### Task 8: `ApiError.body`, the assessment record, `quest.roadmapJob`, `auth.setLevel` (design §0.2, §0.3, §5 "Store", §8)

**Files:** `frontend/utils/apiClient.ts`, `utils/assessmentRecord.ts` (new), `stores/quest.ts`, `stores/auth.ts`, `composables/useOnboardingApi.ts`, `tests/unit/{apiClient,assessmentRecord,questStore,authStore}.test.ts`.

- [ ] **Step 1 (tests first):** `apiClient.test.ts`: a 404 `{"error":"roadmap_generating","status":"failed"}` → `ApiError{status:404, code:'roadmap_generating', body:{error:…, status:'failed'}}`; a non-JSON error → `code 'unknown_error'`, `body` undefined; the 401 path unchanged. `assessmentRecord.test.ts`: `saveAssessment({request, assessed_level, submitted_at})` / `readAssessment()` round-trip under `aelp.assessment`; malformed JSON → `null` and the key removed; storage throwing → `null`, no throw; `clearAssessment()`. `questStore.test.ts`: `load()` on 404 `roadmap_generating` + `status:'generating'` → `roadmapJob 'generating'`, `noRoadmap false`, `error null`, `daily null`; `status:'failed'` → `'failed'`; `roadmap_generating` with no/unknown `status` → `'generating'`; `no_active_roadmap` → `noRoadmap true`, `roadmapJob 'none'`; any other 404 code → `error` set, `roadmapJob` unchanged (design §8: never "failed" for an arbitrary 404); 200 → `'none'`. `authStore.test.ts`: `setLevel('B1')` updates `user.cefr_current` and the persisted `aelp.auth`; no user → no-op; `signOut()` removes `aelp.assessment`.
- [ ] **Step 2:** `npm run test:unit -- apiClient assessmentRecord questStore authStore` — FAIL.
- [ ] **Step 3:** `apiClient.ts`: `ApiError` gains an optional third constructor arg `public readonly body?: Record<string, unknown>`; `errorCode` becomes `errorInfo(res)` returning `{code, body}` (body only when the JSON is an object). `assessmentRecord.ts`: `ASSESSMENT_STORAGE_KEY = 'aelp.assessment'`, `AssessmentRecord {request: AssessmentRequest, assessed_level: string, submitted_at: number}`, `saveAssessment`, `readAssessment`, `clearAssessment` (try/catch around every storage access). `quest.ts`: state `roadmapJob: 'none' as 'none' | 'generating' | 'failed'`; `load()` maps per the tests. `auth.ts`: `setLevel(level)` writes through the same `Persisted` shape as `signIn`; `signOut()` calls `clearAssessment()`. `useOnboardingApi.ts`: `AssessmentResponse = { status: 'success', assessed_level, roadmap_id, pet_state } | { status: 'generating', assessed_level, pet_state }` (discriminated on `status`; 201 and 200 are both `'success'`).
- [ ] **Step 4:** `npm run lint && npm run typecheck && npm run test:unit` PASS.
- [ ] **Step 5:** Commit: `pwa: ApiError body, the assessment record and quest.roadmapJob`.

### Task 9: `DrawingBar`, `RankPlate` and `usePollRoadmap` (design §0.4, §5, §6; acceptance 3, 9)

**Files:** `frontend/components/retro/DrawingBar.vue`, `RankPlate.vue` (new), `composables/usePollRoadmap.ts` (new), `tests/unit/{DrawingBar,RankPlate,usePollRoadmap}.test.ts`.

- [ ] **Step 1 (tests first):** `DrawingBar.test.ts`: `role="progressbar"`, `aria-label="Đang vẽ bản đồ"`, **no** `aria-valuenow`; 40 cells / 164 px track (reuse `HpBar`'s cell geometry); default → the sweep class; `reduced` → no animation class and the block at cells 6–13. `RankPlate.test.ts`: renders `Cấp {level}` in the display face with the `torch` outer line; no `rounded-*` beyond `rounded-sm`. Add both to `retroRadius.test.ts`'s file list if it enumerates files. `usePollRoadmap.test.ts` (fake timers, a stub `load` that records `Date.now()`): `start()` with `submittedAt = now` → polls at 5-s intervals until 60 s, 10-s until 240 s, 30-s after; `stale` becomes true at 420 s and polling stops; `document.visibilityState = 'hidden'` → no poll fires; `visibilitychange` → visible fires one immediately; `online` fires one immediately; `submittedAt` 100 s in the past → the next poll is 10 s away (backoff resumes, acceptance 4); `stop()` and unmount clear every timer and listener (`vi.getTimerCount() === 0`).
- [ ] **Step 2:** `npm run test:unit -- DrawingBar RankPlate usePollRoadmap` — FAIL.
- [ ] **Step 3:** `DrawingBar.vue` per design §6 (8-cell `growth` block, `steps(8)`, 1.6 s, loop; `@media (prefers-reduced-motion: reduce)` and `reduced` → static). `RankPlate.vue` per `retro-onboarding.md` §6 (`level` prop; VT323 34 on a `torch` outer line). `usePollRoadmap({ load, submittedAt: () => number | null, active: () => boolean })` → `{ start, stop, elapsedSeconds, stale }`: a `setTimeout` chain whose next delay is `elapsed < 60 ? 5 : elapsed < 240 ? 10 : 30` s; stops when `active()` is false after a load or at 420 s (`stale = true`); `submittedAt() ?? mountTime`; `onBeforeUnmount(stop)`.
- [ ] **Step 4:** `npm run lint && npm run typecheck && npm run test:unit` PASS.
- [ ] **Step 5:** Commit: `pwa: DrawingBar and RankPlate kit additions, usePollRoadmap backoff`.

### Task 10: Onboarding — grading and result steps (design §2, §3 "Onboarding — grading/result", §4, §5, §7; acceptance 1, 8)

**Files:** `frontend/pages/onboarding.vue`, `tests/unit/onboardingPage.test.ts`.

- [ ] **Step 1 (tests first):** update `onboardingPage.test.ts` — the pins of the old wait copy (`thường mất 1–2 phút`) and of the old result card (`Trình độ của bạn`, `đã nảy mầm`) are replaced; the error-copy pins move to the design §7 strings (`Cậu vừa gửi quá nhiều lần…`, `Máy chủ AI đang bận…`, `Máy chủ không nhận thông tin đã gửi…`, `Không tạo được hành trình. Thử lại.`); keep every goal/time/plant-name/quiz pin. New: while the POST is pending the grading panel shows "Tớ đang xem lại các lượt của cậu…" and a `DrawingBar`; a 202 `{status:'generating', assessed_level:'B1', pet_state}` → the result step: `RankPlate` "Cấp B1", `CompanionSprite` with `react="levelup"` once (the hatch), the line "Tớ nở rồi! Cậu ở cấp B1. Giờ tớ đang vẽ bản đồ cho cậu…", eyebrow "Bản đồ ngày 1", `DrawingBar`, caption "Đóng ứng dụng cũng được — bản đồ vẫn được vẽ tiếp.", button "Vào trại →"; `auth.setLevel('B1')` called; `readAssessment()` equals the posted request + level + a `submitted_at`; a 200/201 `status:'success'` → the same step with "Bản đồ đã sẵn sàng." and no `DrawingBar`, no record saved; "Vào trại →" → `navigateTo('/', {replace:true})` without an awaited `quest.load()`; on mount, a `quest.load()` that yields `roadmapJob 'generating'` redirects to `/`, while `'failed'` stays (the hub's "Làm lại trận đầu tiên"); `grep`-style assertion: the rendered page never contains "Đừng đóng trang".
- [ ] **Step 2:** `npm run test:unit -- onboardingPage` — FAIL.
- [ ] **Step 3:** `onboarding.vue`: keep the goal and quiz steps as they are on `main` (their retro restyle is retro plan 4 of 6). Replace the pending hint under the last-question button with the design §3 grading panel (`RetroPanel speaker={plant name or 'Mầm Non'}` + seed `CompanionSprite` + `SpeechBox` line + `DrawingBar`); replace the `result` card with the design §3 result layout (`RetroPanel`, `CompanionSprite`, `RankPlate`, `SpeechBox`, eyebrow, `DrawingBar` or the ready caption, `RetroButton primary` "Vào trại →"). `next()`: on success call `auth.setLevel(res.assessed_level)`; `status === 'generating'` → `saveAssessment({request, assessed_level, submitted_at: Date.now()})`; `finish()` → `navigateTo('/', { replace: true })`. `assessErrorMessage` → the §7 strings. `onMounted`: redirect when `quest.daily || quest.roadmapJob === 'generating'`. Reduced motion: pass `reduced` from `matchMedia('(prefers-reduced-motion: reduce)')` to `CompanionSprite`/`DrawingBar`.
- [ ] **Step 4:** `npm run lint && npm run typecheck && npm run test:unit` PASS; `grep -rn "Đừng đóng trang" frontend/pages frontend/components` empty.
- [ ] **Step 5:** Commit: `pwa: onboarding shows the level and the sprout at the grade, not after the roadmap`.

### Task 11: Hub — drawing, failed and reveal states (design §1, §3 "Hub — drawing/failed", §4, §5, §7; acceptance 2, 4, 5, 6, 7, 9)

**Files:** `frontend/pages/index.vue`, `tests/unit/indexPage.test.ts`.

- [ ] **Step 1 (tests first):** extend `indexPage.test.ts` (it already stubs `useApi` and the stores; keep the growth-moment and streak-shield cases green):
  - drawing: `quest.load` → 404 `roadmap_generating`/`generating`, `auth.user.cefr_current = 'B1'` → plant card shows "Cấp B1" and the drawing line "Tớ đang vẽ bản đồ cho cậu… Đóng ứng dụng cũng được."; the quest region is replaced by eyebrow "Bản đồ ngày 1", a `DrawingBar`, a `RetroPanel fog` with three `aria-hidden` placeholders and the caption "Ba nhiệm vụ đầu tiên sẽ hiện ở đây."; no `SegmentedProgress`, no `QuestRow`, no bottom button; before the first `load()` settles neither the eyebrow nor the `DrawingBar` renders.
  - reload: a stored record with `submitted_at` 100 s ago → the drawing state, the level from `auth.user`, the next poll 10 s later, and **no** `POST /onboarding/assessment` on mount.
  - reveal: the stub answers 404 `generating` then 200 → in one tick the fog is gone, the quest region renders the three tasks with task 1 as `next`, the line "Vẽ xong rồi! Phòng đầu tiên ở ngay đây, cậu.", a bottom `RetroButton` "Vào nhiệm vụ 1 →" linking `/learn/<first task id>`, and `readAssessment()` is `null`; a mount that gets 200 first shows the ordinary hub, no reveal line, no button.
  - failed: 404 `status:'failed'` → the ember panel "Không tạo được hành trình. Thử lại." + "Thử lại", line "Bản đồ chưa xong. Cậu bấm thử lại giúp tớ nhé.", the fog panel still under it; tap → exactly one `POST /api/v1/onboarding/assessment` with body deep-equal to the stored `request`, button disabled while pending; a 202 → drawing state with `submitted_at` reset; a 200/201 → `quest.load()`; `rate_limited` / `ai_*` / `invalid_request` / network → the panel text swaps to the §7 string and the button re-enables; offline (`navigator.onLine = false`) → toast "Cần mạng để thử lại.", no POST. No stored record → the button reads "Làm lại trận đầu tiên" and navigates to `/onboarding`.
  - stale: 420 s of `generating` (fake timers) → the failed panel with "Vẽ bản đồ lâu hơn bình thường. Thử lại.".
  - network error while drawing → the drawing state stays, exactly one toast "Đang ngoại tuyến — bản đồ vẫn được vẽ, tớ sẽ xem lại khi có mạng.", the "Không tải được nhiệm vụ." panel never renders; `no_active_roadmap` → the existing empty state unchanged.
  - reduced motion: `DrawingBar` static and no reveal animation class.
- [ ] **Step 2:** `npm run test:unit -- indexPage` — FAIL.
- [ ] **Step 3:** `index.vue`: keep `start(pet.consumeDelta())` before first paint and the plant-name speaker exactly as on `main`. Add a `drawing = computed(() => quest.roadmapJob !== 'none' || stale.value)` branch **before** the `quest.noRoadmap` branch (so `quest.error` never shows over it while `roadmapJob === 'generating'` — keep the last known `roadmapJob` on a network error, which the Task 8 store already does); `usePollRoadmap({ load: quest.load, submittedAt: () => readAssessment()?.submitted_at ?? null, active: () => quest.roadmapJob === 'generating' })`, started on mount when `roadmapJob === 'generating'` after the first load; a `revealed` ref set by a `watch` on `quest.roadmapJob` going `'generating' → 'none'` with `quest.daily` non-null (clears the record); the speech line is overridden by the drawing / failed / reveal lines while those apply; the failed panel's retry posts `readAssessment().request` through `useOnboardingApi().assess` and, on 202, `saveAssessment({...record, submitted_at: Date.now()})` + `quest.load()` + `poll.start()`. The level label under the health bar: `auth.user?.cefr_current` shown only in the drawing/failed states.
  **Adaptation (see Notes):** the reveal renders today's quest region (`SegmentedProgress` + the `QuestRow` list, task 1 `next`) in place of the fog; the retro-hub `QuestNode` path, connector lighting and the `DayBar` land with retro plan 2 (hub).
- [ ] **Step 4:** `npm run lint && npm run typecheck && npm run test:unit && npm run build` PASS.
- [ ] **Step 5:** Commit: `pwa: the hub draws the map while the roadmap job runs, then reveals day 1`.

## Notes
- **Estimate 8 h, over the ~6 h norm.** Splitting into two plans is not allowed today, so the tasks are ordered backend-first (Tasks 1–7, ≈ 5 h, green on their own at Task 7) then frontend (Tasks 8–11, ≈ 3 h). The backend must not ship without Tasks 9–11: today's onboarding page would treat the 202 as a finished onboarding and land on a hub that errors on `roadmap_generating`.
- **Inbox bugs this plan delivers the guard for (both stay `selected`; the reviewer closes them after checking):** `harness/ideas/_inbox/two-concurrent-assessment-submits-double-spend-the-ai-and-or.md` — the `roadmap:gen:{user_id}` acquire means a concurrent submit answers 202 without a second roadmap generation (at most one extra grading call in the double-tap window, which the bug's Expected output allows) and no orphaned roadmap; the partial unique index on `roadmaps (user_id) WHERE is_active` it also asks for is **not** in this plan. `harness/ideas/_inbox/assessment-request-has-no-overall-cap-malformed-output-retry.md` — the request is now bounded by grading (2 × 30 s) and the job by `RoadmapJobTimeout` (360 s, pinned against `TaskTimeout` and the key TTL); `server.go`'s comment and `CLAUDE.md` state it (Task 7).
- **Design departure — acceptance 5 is adapted.** The design composes with `retro-hub.md` / `retro-onboarding.md`, but only the retro kit (plan 1 of 6) is built; the hub and onboarding page plans (2 and 4 of 6) are not written. This plan puts the new states, built from the kit, into today's `pages/index.vue` and `pages/onboarding.vue`: the reveal swaps the fog for today's quest region with task 1 as `next`, the reveal line and the "Vào nhiệm vụ 1 →" button, but the `QuestNode` connector lighting and the `DayBar` wait for retro plan 2, which must keep this plan's states. The last-question button keeps its current label ("Hoàn thành") until retro plan 4 renames it "Kết thúc".
- **Wire decision the design left open (§0.3):** one code `roadmap_generating` with a `status` field, not a sibling `roadmap_failed`. 201 disappears from the assessment route (the job always answers 202); the client keeps accepting 201.
- **TTL choice:** `failed` lives 6 min, like `generating` (the idea's 2 × 180 s). After that the hub shows the existing `no_active_roadmap` empty state even though the staged level still lives 2 h; a longer `failed` TTL (= `PlacementQuizTTL`) is a one-line follow-up if the reviewer sees learners hitting it.
- **Shutdown:** a deploy during a job cancels it through the root context, the job marks the key `failed`, `main` waits up to 3 s; a hard kill leaves `generating` until the TTL and the client's 420 s stale rule offers "Thử lại".
- `POST /roadmaps/regenerate` (59-of-84) stays synchronous; making it a job too is the design's §0.5 follow-up.

## Verification
```
# gate (Depends on) first — GATE-OK, else stop and report
cd backend && go build ./... && gofmt -l . && go vet ./... && go test -timeout 180s ./... -count=1 -race
COMPOSE_PROJECT_NAME=<slug> make up && go test -timeout 300s ./... -run Integration -p 1 -count=1 && make down
grep -n 'RoadmapGenKey' internal/store/keys.go                                  # 1 hit
grep -n 'StatusAccepted' internal/onboarding/handler.go                         # 1 hit
grep -n 'roadmap_generating' internal/quests/handler.go "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md" ../harness/CODEMAP.md   # >= 3 hits
git diff origin/main --stat -- internal/airouter/router.go internal/airouter/timeouts.go   # empty
cd ../frontend && npm run lint && npm run typecheck && npm run test:unit && npm run build
grep -rn "Đừng đóng trang" pages components                                      # empty
git push -u origin harness/2026-09-27-medium-level-result-in-30-seconds-grade-first-write-the-roadmap-in-
gh run list --branch harness/2026-09-27-medium-level-result-in-30-seconds-grade-first-write-the-roadmap-in-   # every job green
```
Design acceptance (`harness/designs/level-result-in-30-seconds-grade-first-write-the-roadmap-in-.md`), each proven by the named test:
1. With a stubbed assessment that answers 202 `{status: "generating", assessed_level: "B1", pet_state}` and a `quests/daily` stub that never completes, the result step shows the rank plate "Cấp B1", the sprout (hatch once) and the `DrawingBar` — and no string in `frontend/` tells the learner to keep the page open. *(Task 10)*
2. "Vào trại →" navigates to `/`; while `GET /quests/daily` answers `404 roadmap_generating` (`status: generating`) the hub shows the companion panel with "Cấp B1", the drawing line, the eyebrow "Bản đồ ngày 1", the `DrawingBar`, the fogged placeholders and no bottom button. *(Tasks 10, 11)*
3. Polling (fake timers): `quests/daily` is requested at 5-s intervals for the first 60 s, 10-s until 240 s, 30-s after; no request fires while the document is hidden; one fires immediately on `visibilitychange` → visible and on `online`; at 420 s of `generating` the failed panel appears with the stale string. *(Tasks 9, 11)*
4. A reload or PWA reopen during generation shows the same drawing state, the level from `auth.user.cefr_current`, and the backoff continues from the stored `submitted_at`; no assessment POST is sent on load. A fresh load after completion renders the plain first-time hub with no reveal. *(Task 11)*
5. When a poll returns 200 the fog is replaced by the three `QuestNode`s and the `DayBar` within one frame, the connector to tile 1 lights, the cursor lands on tile 1, the bottom button reads "Vào nhiệm vụ 1 →", the line is the reveal line, and `aelp.assessment` is removed — once per mounted page. *(Task 11 — **adapted**: today's quest region with task 1 as `next` stands in for `QuestNode`/`DayBar`/connector until retro plan 2; see Notes)*
6. `status: failed` shows the ember panel "Không tạo được hành trình. Thử lại." with "Thử lại"; tapping it sends exactly one `POST /onboarding/assessment` whose body equals the stored request (`target_goal`, `notification_time`, `timezone`, `answers`, `plant_name` when present), disables while in flight, and on 202 returns to the drawing state. With no stored record the button reads "Làm lại trận đầu tiên" and opens `/onboarding`. *(Task 11; backend retry semantics: Task 4 `TestRetryAfterFailedJobReusesTheStagedLevel`)*
7. A network error during polling keeps the drawing state and shows exactly one offline toast; the quest error panel never appears while `roadmapJob === 'generating'`. *(Task 11)*
8. A 201/200 assessment (roadmap already active) renders the result step with the "Bản đồ đã sẵn sàng." caption and "Vào trại →" lands on a hub with quests; `no_active_roadmap` still renders the shell empty state. *(Tasks 10, 11)*
9. Under `prefers-reduced-motion` (and `reduced`) no keyframe runs: the hatch is a frame swap, the `DrawingBar` block is static, the reveal is the final frame, and every state is distinguishable by text and glyph. *(Tasks 9, 10, 11)*
10. `npm run lint`, `npm run typecheck`, `npm run test:unit`, `npm run build` green; CI green on the pushed branch. *(commands above)*
