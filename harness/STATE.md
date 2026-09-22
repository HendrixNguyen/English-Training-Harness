# Harness state

_Generated 2026-09-22 17:32. Do not edit — run `python3 tools/harness/cli.py state`._


## Invalid

_none_

## Blockers (merge refused until fixed)

_none_

## Inbox

- `harness/ideas/_inbox/backend-env-example-omits-the-app-s-own-database-url-redis-u.md` — backend/.env.example omits the app's own DATABASE_URL REDIS_URL PORT [low]
- `harness/ideas/_inbox/cmd-api-has-no-graceful-shutdown-so-its-deferred-close-calls.md` — cmd/api has no graceful shutdown so its deferred Close calls are unreachable [low]
- `harness/ideas/_inbox/docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md` — docker-compose hard-codes host ports so make up fails locally [high]
- `harness/ideas/_inbox/gin-default-ships-debug-mode-and-all-proxies-trusted-to-prod.md` — gin.Default ships debug mode and all-proxies-trusted to production [medium]
- `harness/ideas/_inbox/go-test-drops-every-table-in-whatever-database-url-points-at.md` — go test drops every table in whatever DATABASE_URL points at [high]
- `harness/ideas/_inbox/healthz-leaks-postgres-and-redis-driver-error-strings-public.md` — healthz leaks Postgres and Redis driver error strings publicly [medium]
- `harness/ideas/_inbox/healthz-shares-one-2s-deadline-across-two-sequential-pings.md` — healthz shares one 2s deadline across two sequential pings [low]
- `harness/ideas/_inbox/integration-gate-tests-only-prove-the-skip-and-would-pass-if.md` — Integration gate tests only prove the skip and would pass if the gate always skipped [medium]
- `harness/ideas/_inbox/migrate-is-only-tested-against-the-single-embedded-migration.md` — Migrate is only tested against the single embedded migration [medium]
- `harness/ideas/_inbox/migrations-run-on-every-boot-with-no-advisory-lock.md` — migrations run on every boot with no advisory lock [medium]
- `harness/ideas/_inbox/no-ci-runs-the-integration-suite-so-pgmigrator-is-never-veri.md` — no CI runs the integration suite so PgMigrator is never verified [medium]
- `harness/ideas/_inbox/root-gitignore-env-silently-swallows-every-module-s-env-exam.md` — Root .gitignore .env* silently swallows every module's .env.example [medium]
- `harness/ideas/_inbox/spec-3-2-leaves-user-id-nullable-on-three-child-tables.md` — spec 3.2 leaves user_id nullable on three child tables [low]

## Proposed

- `harness/ideas/2026-09-22-run-01/adaptive-reminder-timing-and-pre-decay-rescue-push.md` — Adaptive Reminder Timing and Pre-Decay Rescue Push
- `harness/ideas/2026-09-22-run-01/pet-streak-shield-earned-by-target-days.md` — Pet Streak Shield Earned by Target Days
- `harness/ideas/2026-09-22-run-01/spaced-repetition-vocabulary-review-in-the-daily-quest.md` — Spaced Repetition Vocabulary Review in the Daily Quest
- `harness/ideas/2026-09-22-run-02/ai-router-multi-llm-providers-task-strategies-and-rate-limit.md` — AI Router: multi-LLM providers, task strategies and rate limit
- `harness/ideas/2026-09-22-run-02/frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md` — Frontend Shell: Nuxt 3 PWA with auth, daily quest and pet screens
- `harness/ideas/2026-09-22-run-02/google-one-way-calendar-and-tasks-sync.md` — Google: one-way Calendar and Tasks sync
- `harness/ideas/2026-09-22-run-02/notify-web-push-subscriptions-and-delayed-reminder-queue.md` — Notify: Web Push subscriptions and delayed reminder queue
- `harness/ideas/2026-09-22-run-02/onboarding-placement-test-cefr-grading-and-roadmap-generatio.md` — Onboarding placement test CEFR grading and roadmap generation
- `harness/ideas/2026-09-22-run-02/pet-health-streak-and-stage-engine-with-revive.md` — Pet: health, streak and stage engine with revive
- `harness/ideas/2026-09-22-run-02/reconcile-pet-states-stage-between-erd-and-ddl-wilted-defaul.md` — Reconcile pet_states stage between ERD and DDL (wilted, default) [low]
- `harness/ideas/2026-09-22-run-02/spec-never-states-when-the-pet-states-row-is-created.md` — Spec never states when the pet_states row is created [low]

## Selected

_none_

## Planned (awaiting approval)

_none_

## Approved

- `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md` — Auth: Google OAuth code exchange and JWT sessions — Plan [high]
- `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md` — Quests: daily quest suite and progress recording — Plan [high]
- `harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md` — CI on GitHub Actions for backend and harness tooling — Plan [high]

## Executing

_none_

## Done (last 10)

- `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md` — Store: Go module, Postgres and Redis clients, migration 0001 — Plan [high] (review: pass-with-bugs) (merged)
- `harness/plans/2026-09-22-go-test-drops-every-table-in-whatever-database-url-points-at.md` — go test drops every table in whatever DATABASE_URL points at — Plan [high] (unreviewed) (merged)
- `harness/plans/2026-09-22-docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md` — docker-compose hard-codes host ports so make up fails locally — Plan [high] (unreviewed) (merged)
- `harness/plans/2026-09-22-add-unique-user-id-to-pet-states-ddl.md` — Add UNIQUE user_id to pet_states DDL — Plan [high] (review: pass-with-bugs) (merged)

## Failed

_none_
