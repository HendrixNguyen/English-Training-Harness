# Harness state

_Generated 2026-09-23 09:58. Do not edit — run `python3 tools/harness/cli.py state`._


## Invalid

_none_

## Blockers (merge refused until fixed)

_none_

## Inbox (reviewer bugs awaiting evaluation)

- `harness/ideas/_inbox/a-passed-revival-is-knocked-from-50-back-to-20-by-the-same-l.md` — A passed revival is knocked from 50 back to 20 by the same local day's miss penalty [medium]
- `harness/ideas/_inbox/a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md` — A Pet.State failure reports pet_health 0 which means a dead plant [medium]
- `harness/ideas/_inbox/a-re-submitted-assessment-silently-discards-target-goal-noti.md` — A re-submitted assessment silently discards target_goal notification_time and timezone and still answers status success [medium]
- `harness/ideas/_inbox/auth-reports-postgres-and-redis-failures-as-401-and-logs-not.md` — auth reports Postgres and Redis failures as 401 and logs nothing [medium]
- `harness/ideas/_inbox/backend-env-example-omits-the-app-s-own-database-url-redis-u.md` — backend/.env.example omits the app's own DATABASE_URL REDIS_URL PORT [low]
- `harness/ideas/_inbox/branch-carries-its-own-execution-summary-so-the-ai-router-pl.md` — Branch carries its own execution summary so the ai-router plan file conflicts on merge [high]
- `harness/ideas/_inbox/ci-jobs-have-no-timeout-minutes-and-the-harness-job-floats-p.md` — CI jobs have no timeout-minutes and the harness job floats python-version 3.x [low]
- `harness/ideas/_inbox/cmd-api-has-no-graceful-shutdown-so-its-deferred-close-calls.md` — cmd/api has no graceful shutdown so its deferred Close calls are unreachable [low]
- `harness/ideas/_inbox/gemini-test-assertion-that-the-api-key-is-not-in-the-query-s.md` — Gemini test assertion that the API key is not in the query string can never fail [low]
- `harness/ideas/_inbox/geminiprovider-drops-every-response-part-after-the-first-so-.md` — GeminiProvider drops every response part after the first so a long roadmap arrives truncated [medium]
- `harness/ideas/_inbox/get-onboarding-quiz-is-shipped-but-absent-from-backend-spec-.md` — GET onboarding quiz is shipped but absent from backend spec 6.1 and 1st-thinking 7 [low]
- `harness/ideas/_inbox/gin-default-ships-debug-mode-and-all-proxies-trusted-to-prod.md` — gin.Default ships debug mode and all-proxies-trusted to production [medium]
- `harness/ideas/_inbox/google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md` — google_refresh_token is stored in plaintext; backend spec 7 requires AES-256-GCM via ENCRYPTION_SECRET_KEY [high]
- `harness/ideas/_inbox/healthz-leaks-postgres-and-redis-driver-error-strings-public.md` — healthz leaks Postgres and Redis driver error strings publicly [medium]
- `harness/ideas/_inbox/healthz-shares-one-2s-deadline-across-two-sequential-pings.md` — healthz shares one 2s deadline across two sequential pings [low]
- `harness/ideas/_inbox/integration-gate-tests-only-prove-the-skip-and-would-pass-if.md` — Integration gate tests only prove the skip and would pass if the gate always skipped [medium]
- `harness/ideas/_inbox/integration-tests-share-one-database-and-p-1-is-a-workaround.md` — integration tests share one database and -p 1 is a workaround not isolation [medium]
- `harness/ideas/_inbox/jwt-secret-is-accepted-at-any-length-including-one-character.md` — JWT_SECRET is accepted at any length including one character [medium]
- `harness/ideas/_inbox/jwt-verify-does-not-require-exp-or-bind-iss-aud-and-bearer-i.md` — JWT verify does not require exp or bind iss/aud and Bearer is case-sensitive [low]
- `harness/ideas/_inbox/main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` — main.go installs a signal handler with no server shutdown so the API now ignores SIGINT and SIGTERM [high]
- `harness/ideas/_inbox/meeting-the-daily-target-during-a-revive-challenge-answers-4.md` — Meeting the daily target during a revive challenge answers 409 and leaks the Redis key for 24h [medium]
- `harness/ideas/_inbox/migrate-is-only-tested-against-the-single-embedded-migration.md` — Migrate is only tested against the single embedded migration [medium]
- `harness/ideas/_inbox/migrate-s-advisory-lock-can-hang-boot-forever-with-no-bound-.md` — Migrate's advisory lock can hang boot forever with no bound and no log [medium]
- `harness/ideas/_inbox/migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md` — Migrate's advisory lock leaks into the pool when unlock runs on a cancelled context [high]
- `harness/ideas/_inbox/migrations-run-on-every-boot-with-no-advisory-lock.md` — migrations run on every boot with no advisory lock [medium]
- `harness/ideas/_inbox/module-week-is-never-validated-and-day-number-comes-from-arr.md` — Module.week is never validated and day_number comes from array position [medium]
- `harness/ideas/_inbox/no-index-supports-the-quests-lookups-on-roadmaps-and-exercis.md` — No index supports the quests lookups on roadmaps and exercises [low]
- `harness/ideas/_inbox/no-post-handler-bounds-the-request-body-so-one-jwt-can-decod.md` — No POST handler bounds the request body so one JWT can decode an unbounded answers array into memory [medium]
- `harness/ideas/_inbox/onboarding-service-stores-a-now-clock-it-never-uses-so-every.md` — onboarding Service stores a now clock it never uses so every caller passes a dead dependency [low]
- `harness/ideas/_inbox/ontargetmet-is-lost-forever-if-a-write-after-the-incrby-fail.md` — OnTargetMet is lost forever if a write after the INCRBY fails [medium]
- `harness/ideas/_inbox/parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md` — ParseRoadmap accepts a 90-minute daily quest so the 30-minute day is unenforced [medium]
- `harness/ideas/_inbox/pet-sweep-tests-cannot-fail-on-the-branches-they-are-named-f.md` — Pet sweep tests cannot fail on the branches they are named for [low]
- `harness/ideas/_inbox/progress-request-validation-is-incomplete-outside-the-gin-bi.md` — Progress request validation is incomplete outside the Gin binding tags [low]
- `harness/ideas/_inbox/quests-reads-the-users-table-directly-for-the-timezone.md` — quests reads the users table directly for the timezone [medium]
- `harness/ideas/_inbox/redisratelimiter-has-no-documented-behaviour-when-redis-is-d.md` — RedisRateLimiter has no documented behaviour when Redis is down and a zero Limit blocks everything [low]
- `harness/ideas/_inbox/root-gitignore-env-silently-swallows-every-module-s-env-exam.md` — Root .gitignore .env* silently swallows every module's .env.example [medium]
- `harness/ideas/_inbox/route-falls-back-on-terminal-4xx-so-one-bad-prompt-buys-thre.md` — Route falls back on terminal 4xx so one bad prompt buys three paid provider calls [medium]
- `harness/ideas/_inbox/route-has-no-overall-deadline-so-one-call-can-take-90-second.md` — Route has no overall deadline so one call can take 90 seconds [low]
- `harness/ideas/_inbox/route-returns-upstream-provider-error-bodies-verbatim-to-its.md` — Route returns upstream provider error bodies verbatim to its caller [medium]
- `harness/ideas/_inbox/seeddemoroadmap-is-not-transactional-and-can-leave-a-partial.md` — SeedDemoRoadmap is not transactional and can leave a partial active roadmap [low]
- `harness/ideas/_inbox/service-and-require-failure-paths-are-untested-the-fakes-err.md` — Service and Require failure paths are untested; the fakes' err fields are never set [low]
- `harness/ideas/_inbox/service-ontargetmet-ignores-localdate-so-pet-has-no-idempote.md` — Service.OnTargetMet ignores localDate so pet has no idempotency of its own [medium]
- `harness/ideas/_inbox/shared-http-helpers-live-in-gemini-go-and-an-empty-base-url-.md` — Shared HTTP helpers live in gemini.go and an empty base URL silently means OpenAI [low]
- `harness/ideas/_inbox/signing-in-on-a-second-device-silently-logs-the-first-one-ou.md` — signing in on a second device silently logs the first one out [medium]
- `harness/ideas/_inbox/spec-3-2-leaves-user-id-nullable-on-three-child-tables.md` — spec 3.2 leaves user_id nullable on three child tables [low]
- `harness/ideas/_inbox/stale-main-go-header-comment-and-a-double-prefixed-config-er.md` — stale main.go header comment and a double-prefixed config error [low]
- `harness/ideas/_inbox/sweep-reads-updated-at-then-writes-unconditionally-so-two-sw.md` — Sweep reads updated_at then writes unconditionally so two sweeps in one local hour double the miss penalty [medium]
- `harness/ideas/_inbox/the-hourly-sweep-loads-every-pet-in-a-zone-into-memory-and-r.md` — The hourly sweep loads every pet in a zone into memory and reparses tzdata per user [medium]
- `harness/ideas/_inbox/the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` — The miss sweep judges the day from volatile Redis and ignores the durable daily_progress row [medium]
- `harness/ideas/_inbox/two-comments-in-the-new-progress-validation-misstate-the-cod.md` — Two comments in the new progress validation misstate the code they describe [low]
- `harness/ideas/_inbox/two-concurrent-assessment-submits-double-spend-the-ai-and-or.md` — Two concurrent assessment submits double-spend the AI and orphan a roadmap with its 84 exercises [medium]
- `harness/ideas/_inbox/two-recordprogress-error-branches-are-uncovered-and-the-redi.md` — Two RecordProgress error branches are uncovered and the Redis-down test's comment overstates it [medium]
- `harness/ideas/_inbox/zones-that-skip-local-midnight-on-spring-forward-are-never-s.md` — Zones that skip local midnight on spring-forward are never swept that day [medium]

## Proposed

- `harness/ideas/2026-09-22-run-01/adaptive-reminder-timing-and-pre-decay-rescue-push.md` — Adaptive Reminder Timing and Pre-Decay Rescue Push
- `harness/ideas/2026-09-22-run-01/pet-streak-shield-earned-by-target-days.md` — Pet Streak Shield Earned by Target Days
- `harness/ideas/2026-09-22-run-01/spaced-repetition-vocabulary-review-in-the-daily-quest.md` — Spaced Repetition Vocabulary Review in the Daily Quest
- `harness/ideas/2026-09-22-run-02/reconcile-pet-states-stage-between-erd-and-ddl-wilted-defaul.md` — Reconcile pet_states stage between ERD and DDL (wilted, default) [low]
- `harness/ideas/2026-09-22-run-02/spec-never-states-when-the-pet-states-row-is-created.md` — Spec never states when the pet_states row is created [low]

## Selected

_none_

## Planned (awaiting approval)

_none_

## Approved

- `harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md` — Notify: Web Push subscriptions and delayed reminder queue — Plan [high]
- `harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md` — Frontend Shell: Nuxt 3 PWA with auth, daily quest and pet screens — Plan [high]
- `harness/plans/2026-09-23-auth-google-response-returns-token-and-omits-token-type-and-.md` — auth/google response: answer the backend spec §6.1 sign-in body — Plan [high]

## Executing

- `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md` — Google: one-way Calendar and Tasks sync — Plan [high]

## Done (last 10)

- `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md` — Store: Go module, Postgres and Redis clients, migration 0001 — Plan [high] (review: pass-with-bugs) (merged)
- `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md` — Quests: daily quest suite and progress recording — Plan [high] (review: fail) (merged)
- `harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md` — Pet: health, streak and stage engine with revive — Plan [high] (review: pass-with-bugs) (merged)
- `harness/plans/2026-09-22-onboarding-placement-test-cefr-grading-and-roadmap-generatio.md` — Onboarding placement test CEFR grading and roadmap generation — Plan [high] (review: pass-with-bugs)
- `harness/plans/2026-09-22-go-test-drops-every-table-in-whatever-database-url-points-at.md` — go test drops every table in whatever DATABASE_URL points at — Plan [high] (unreviewed) (merged)
- `harness/plans/2026-09-22-docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md` — docker-compose hard-codes host ports so make up fails locally — Plan [high] (unreviewed) (merged)
- `harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md` — CI on GitHub Actions for backend and harness tooling — Plan [high] (review: pass-with-bugs) (merged)
- `harness/plans/2026-09-22-cancel-in-progress-cancels-ci-on-main-the-only-ref-ci-actual.md` — cancel-in-progress cancels CI on main, the only ref CI actually runs on — Plan [high] (unreviewed) (merged)
- `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md` — Auth: Google OAuth code exchange and JWT sessions — Plan [high] (review: pass-with-bugs) (merged)
- `harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md` — AI Router: multi-LLM providers, task strategies and rate limit — Plan [high] (review: pass-with-bugs) (merged)

## Failed

_none_
