# Harness state

_Generated 2026-09-27 21:26. Do not edit — run `python3 tools/harness/cli.py state`._


## Invalid

_none_

## Blockers (merge refused until fixed)

- `harness/ideas/_inbox/settings-saving-the-reminder-time-replaces-the-no-key-unsupp.md` — Settings: saving the reminder time replaces the no-key, unsupported and denied states with off, exposing a switch that cannot work — blocks `harness/plans/2026-09-24-settings-screen-wires-web-push-reminders-and-google-calendar.md`
- `harness/ideas/_inbox/a-typed-content-roadmap-needs-about-20k-output-tokens-so-gen.md` — A typed-content roadmap needs about 20k output tokens so generation cannot finish inside the 180 s budget or gpt-4o-mini's 16k cap — blocks `harness/plans/2026-09-24-typed-task-content-with-answer-keys-so-every-quest-renders-a.md`
- `harness/ideas/_inbox/roadmap-page-never-refetches-today-goes-stale-and-a-new-lear.md` — Roadmap page never refetches: today goes stale and a new learner keeps seeing 'no roadmap' until a full reload — blocks `harness/plans/2026-09-25-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md`
- `harness/ideas/_inbox/onboarding-declares-cefrorder-in-both-the-regenerate-and-the.md` — onboarding declares cefrOrder in both the regenerate and the level-floor branches so the daily merge does not compile — blocks `harness/plans/2026-09-26-a-session-a-learner-wants-to-finish-level-true-content-do-to.md`
- `harness/ideas/_inbox/session-renewal-signs-the-learner-out-when-a-non-2xx-or-a-pa.md` — Session renewal signs the learner out when a non-2xx or a parallel request races the rotated token — blocks `harness/plans/2026-09-26-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve.md`
- `harness/ideas/_inbox/orphan-window-test-fails-once-the-google-403-branch-is-merge.md` — Orphan-window test fails once the google 403 branch is merged: the fake now names the second Google-assigned insert evt_second-2 — blocks `harness/plans/2026-09-27-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn.md`

## Inbox (reviewer bugs awaiting evaluation)

- `harness/ideas/_inbox/a-failed-google-sync-on-a-bad-refresh-token-logs-two-lines-f.md` — A failed Google sync on a bad refresh token logs two lines for one event [low]
- `harness/ideas/_inbox/a-malformed-frontend-origin-is-refused-only-after-migrations.md` — A malformed FRONTEND_ORIGIN is refused only after migrations have run against the database [low]
- `harness/ideas/_inbox/a-typed-content-roadmap-needs-about-20k-output-tokens-so-gen.md` — A typed-content roadmap needs about 20k output tokens so generation cannot finish inside the 180 s budget or gpt-4o-mini's 16k cap [high]
- `harness/ideas/_inbox/already-onboarded-learners-now-see-the-english-default-my-gr.md` — Already-onboarded learners now see the English default 'My Green Buddy' as the plant's name on the hub, banners and bubble [medium]
- `harness/ideas/_inbox/auth-session-store-flattens-the-redis-error-with-v-so-a-clie.md` — auth session store flattens the Redis error with %v so a client cancel is logged as an outage [low]
- `harness/ideas/_inbox/backend-spec-is-silent-on-push-fail-key-subscription-key-val.md` — Backend spec is silent on push fail key, subscription key validation and the 10-subscription cap [low]
- `harness/ideas/_inbox/boot-store-migrate-still-has-no-deadline-now-that-the-adviso.md` — Boot store.Migrate still has no deadline now that the advisory-lock leak is fixed [low]
- `harness/ideas/_inbox/concurrent-subscription-saves-for-one-user-overshoot-the-10-.md` — Concurrent subscription saves for one user overshoot the 10-row cap [low]
- `harness/ideas/_inbox/gemini-thinking-budget-is-missing-from-both-env-examples-and.md` — GEMINI_THINKING_BUDGET is missing from both env examples and thinkingConfig cannot be omitted [low]
- `harness/ideas/_inbox/google-403-allow-list-refactor-dropped-the-pins-for-userrate.md` — Google 403 allow-list refactor dropped the pins for userRateLimitExceeded and quotaExceeded and logs a dangling space for a no-reason 403 [low]
- `harness/ideas/_inbox/growth-moment-plant-name-and-roadmap-tree-ui-is-built-on-kit.md` — Growth moment, plant-name and roadmap-tree UI is built on kit v1; kit v2 on main forbids its radii and eased scale motion [low]
- `harness/ideas/_inbox/hub-bubble-tells-a-learner-who-practised-yesterday-hom-qua-t.md` — Hub bubble tells a learner who practised yesterday 'Hôm qua tớ nhớ bạn' — missed-day check uses elapsed hours, not calendar days [medium]
- `harness/ideas/_inbox/infisical-push-back-step-leaves-production-secrets-in-a-worl.md` — Infisical push-back step leaves production secrets in a world-readable /tmp file when the upload fails [low]
- `harness/ideas/_inbox/learnpage-test-finds-the-completion-button-by-its-mt-4-spaci.md` — learnPage test finds the completion button by its mt-4 spacing class [low]
- `harness/ideas/_inbox/level-guidance-asks-c2-for-300-380-word-passages-but-parsero.md` — Level guidance asks C2 for 300-380-word passages but ParseRoadmap caps passages at 2000 characters [medium]
- `harness/ideas/_inbox/login-shows-generic-retry-copy-for-409-email-in-use-so-a-loc.md` — login shows generic retry copy for 409 email_in_use so a locked-out learner retries forever [low]
- `harness/ideas/_inbox/migration-lock-wait-log-and-error-always-quote-30s-even-when.md` — Migration lock wait log and error always quote 30s even when the caller deadline or a cancel is the bound [low]
- `harness/ideas/_inbox/notify-tick-counts-a-subscription-pruned-and-clears-its-fail.md` — notify Tick counts a subscription pruned and clears its failure counter even when DeleteSubscription fails [low]
- `harness/ideas/_inbox/onboarding-declares-cefrorder-in-both-the-regenerate-and-the.md` — onboarding declares cefrOrder in both the regenerate and the level-floor branches so the daily merge does not compile [high]
- `harness/ideas/_inbox/orphan-window-test-fails-once-the-google-403-branch-is-merge.md` — Orphan-window test fails once the google 403 branch is merged: the fake now names the second Google-assigned insert evt_second-2 [high]
- `harness/ideas/_inbox/pages-without-404-html-answers-a-missing-nuxt-chunk-with-ind.md` — Pages without 404.html answers a missing /_nuxt/ chunk with index.html as 200 under the one-year immutable header [medium]
- `harness/ideas/_inbox/parseorigins-still-accepts-an-empty-port-an-empty-host-a-zer.md` — ParseOrigins still accepts an empty port, an empty host, a zero-padded or out-of-range port and a non-ASCII host [low]
- `harness/ideas/_inbox/plant-name-field-rejects-vietnamese-typed-in-decomposed-unic.md` — Plant-name field rejects Vietnamese typed in decomposed Unicode, including the suggested 'Mầm Non' [low]
- `harness/ideas/_inbox/post-roadmaps-regenerate-answers-400-to-a-chunked-request-wi.md` — POST roadmaps regenerate answers 400 to a chunked request with an empty body [low]
- `harness/ideas/_inbox/retro-amend-2-test-gaps-dark-focus-ring-cascade-unguarded-sp.md` — Retro amend 2 test gaps: dark focus-ring cascade unguarded, SpeechBox flip-back case asserts no emit count [low]
- `harness/ideas/_inbox/rls-migration-leaves-supabase-default-privileges-so-future-t.md` — RLS migration leaves Supabase default privileges so future tables views and functions are granted to anon again [medium]
- `harness/ideas/_inbox/roadmap-hom-nay-chip-and-hoc-ngay-button-put-white-text-on-g.md` — Roadmap 'HÔM NAY' chip and 'Học ngay' button put white text on growth (2.5:1 on the branch, 1.7:1 on main's kit) [medium]
- `harness/ideas/_inbox/roadmap-page-never-refetches-today-goes-stale-and-a-new-lear.md` — Roadmap page never refetches: today goes stale and a new learner keeps seeing 'no roadmap' until a full reload [high]
- `harness/ideas/_inbox/roadmap-tree-marks-day-28-as-today-forever-once-the-28-days-.md` — Roadmap tree marks day 28 as today forever once the 28 days are over (DayNumber clamps) [low]
- `harness/ideas/_inbox/session-renewal-signs-the-learner-out-when-a-non-2xx-or-a-pa.md` — Session renewal signs the learner out when a non-2xx or a parallel request races the rotated token [high]
- `harness/ideas/_inbox/settings-page-misses-the-design-hero-time-and-breaks-ui-kit-.md` — Settings page misses the design hero time and breaks UI kit v2 rules (rounded blurred switch, 3 to 1 status text, invisible time-picker icon) [low]
- `harness/ideas/_inbox/settings-re-consent-with-google-returns-the-learner-to-the-h.md` — Settings re-consent with Google returns the learner to the hub instead of settings, so the sync has to be found and tapped again [low]
- `harness/ideas/_inbox/settings-reminder-switch-can-disagree-with-the-real-push-sub.md` — Settings reminder switch can disagree with the real push subscription after a failed time save or a cleared subscription [medium]
- `harness/ideas/_inbox/settings-reminders-requesting-renders-the-switch-on-a-stale-.md` — Settings reminders: requesting renders the switch on, a stale saveError can mislabel a failed enable, and several guards are mutation-proof in tests [low]
- `harness/ideas/_inbox/settings-saving-the-reminder-time-replaces-the-no-key-unsupp.md` — Settings: saving the reminder time replaces the no-key, unsupported and denied states with off, exposing a switch that cannot work [high]
- `harness/ideas/_inbox/settings-sync-status-always-says-hom-nay-for-a-sync-persiste.md` — Settings sync status always says hom nay for a sync persisted from an earlier day [low]
- `harness/ideas/_inbox/stale-worktrees-has-no-test-for-the-legacy-basename-match-on.md` — stale-worktrees has no test for the legacy basename match on git's list or for skipping an ancestor of the cwd [low]
- `harness/ideas/_inbox/store-reset-refusal-still-has-no-direct-test-deferred-by-the.md` — store reset refusal still has no direct test (deferred by the integration-gate plan until integration_test.go is free) [low]
- `harness/ideas/_inbox/streak-shield-integration-test-never-scans-a-fresh-row-s-shi.md` — Streak-shield integration test never scans a fresh row's shields and races no shielded PenaliseMiss [low]
- `harness/ideas/_inbox/sw-header-strip-comment-and-codemap-say-a-cached-token-is-ne.md` — SW header-strip comment and CODEMAP say a cached token is newer when it is older [low]
- `harness/ideas/_inbox/task-timer-credits-idle-wall-clock-time-one-10-minute-task-l.md` — Task timer credits idle wall-clock time: one 10-minute task left open posts up to 60 minutes and can meet the day alone [medium]
- `harness/ideas/_inbox/testassessraisestheaileveltothefloor-s-prompt-check-is-vacuo.md` — TestAssessRaisesTheAILevelToTheFloor's prompt check is vacuous because RoadmapSchema always contains C1 [low]
- `harness/ideas/_inbox/testvalidatecontentrejects-asserts-only-the-task-location-so.md` — TestValidateContentRejects asserts only the task location so a row can pass for the wrong rule [low]
- `harness/ideas/_inbox/the-drain-overrun-log-line-reads-server-server-drain-grace-e.md` — The drain overrun log line reads server: server: drain grace exceeded [low]
- `harness/ideas/_inbox/the-nuxt-immutable-cache-rule-also-covers-the-unhashed-build.md` — The /_nuxt/* immutable cache rule also covers the unhashed builds/latest.json app manifest [low]
- `harness/ideas/_inbox/the-shield-earn-line-shows-every-day-the-streak-sits-on-a-mu.md` — The shield earn line shows every day the streak sits on a multiple of 7 including the next morning before practice [medium]
- `harness/ideas/_inbox/the-worker-join-after-a-drain-overrun-waits-a-fresh-8-s-grac.md` — The worker join after a drain overrun waits a fresh 8 s grace so worst-case shutdown is 16 s past the Railway window [low]
- `harness/ideas/_inbox/two-done-branches-both-add-migration-0004-pet-shields-and-rl.md` — Two done branches both add migration 0004 pet shields and rls [medium]
- `harness/ideas/_inbox/two-unmerged-branches-both-add-migration-0004-0004-rls-and-0.md` — Two unmerged branches both add migration 0004 (0004_rls and 0004_pet_shields) [low]
- `harness/ideas/_inbox/typed-reading-tasks-show-questions-without-their-passage-and.md` — Typed reading tasks show questions without their passage and vocabulary tasks hide their questions until the frontend slice lands [medium]
- `harness/ideas/_inbox/v1contrast-source-guard-only-scans-class-attributes-so-scrip.md` — v1Contrast source guard only scans class attributes, so script-level class maps can put text-white on a growth fill unnoticed [low]

## Proposed

- `harness/ideas/2026-09-27-run-01/adaptive-placement-a-30-item-a1c2-ladder-replaces-the-fixed-.md` — Adaptive placement: a 30-item A1–C2 ladder replaces the fixed ten questions every learner and every day-28 re-check sees
- `harness/ideas/2026-09-27-run-01/answers-are-remembered-server-graded-accuracy-per-task-feeds.md` — Answers are remembered: server-graded accuracy per task feeds the world map, task swaps and the day-28 checkpoint
- `harness/ideas/2026-09-27-run-01/region-clear-an-end-of-week-recap-on-the-hub-with-days-met-m.md` — Region clear: an end-of-week recap on the hub with days met, minutes, words and the plant's stage

## Selected

- `harness/ideas/2026-09-22-run-02/reconcile-pet-states-stage-between-erd-and-ddl-wilted-defaul.md` — Reconcile pet_states stage between ERD and DDL (wilted, default) [low]
- `harness/ideas/_inbox/a-multi-day-sweep-outage-collapses-every-missed-day-into-one.md` — A multi-day sweep outage collapses every missed day into one penalty [medium]
- `harness/ideas/_inbox/a-non-uuid-exercise-id-on-post-quests-progress-answers-500-i.md` — A non-UUID exercise_id on POST quests progress answers 500 instead of 400 or 404 [low]
- `harness/ideas/_inbox/a-re-submitted-assessment-silently-discards-target-goal-noti.md` — A re-submitted assessment silently discards target_goal notification_time and timezone and still answers status success [medium]
- `harness/ideas/_inbox/a-rejected-cachestorage-delete-now-fails-sign-in-for-a-user-.md` — A rejected CacheStorage delete now fails sign-in for a user who is already signed in [low]
- `harness/ideas/_inbox/a-user-deleted-tasks-list-is-never-rebuilt-and-sync-keeps-an.md` — A user-deleted Tasks list is never rebuilt and sync keeps answering synced with a stale count [low]
- `harness/ideas/_inbox/assessment-request-has-no-overall-cap-malformed-output-retry.md` — Assessment request has no overall cap; malformed-output retry doubles the 180 s roadmap budget [low]
- `harness/ideas/_inbox/companionsprite-motion-is-off-the-pixel-grid-at-every-size-b.md` — CompanionSprite motion is off the pixel grid at every size but 128 px, levelup is a CSS filter not a palette swap, and down lacks its translate [low]
- `harness/ideas/_inbox/config-gin-mode-validation-is-unreachable-in-the-binary-beca.md` — config GIN_MODE validation is unreachable in the binary because gin init panics first, and its comments claim otherwise [low]
- `harness/ideas/_inbox/config-go-still-says-jwt-secret-is-not-in-the-1st-thinking-e.md` — config.go still says JWT_SECRET is not in the 1st-thinking env list, and boot refusals print config: config: [low]
- `harness/ideas/_inbox/due-plus-re-slot-is-not-atomic-so-two-api-instances-double-s.md` — Due plus re-slot is not atomic so two API instances double-send the same reminder [medium]
- `harness/ideas/_inbox/durable-flag-and-live-row-success-tests-leave-updated-at-and.md` — Durable-flag and live-row success tests leave updated_at and the Daily error path unasserted [low]
- `harness/ideas/_inbox/fontsource-per-subset-css-has-no-unicode-range-so-all-nine-v.md` — Fontsource per-subset CSS has no unicode-range, so all nine VT323 and Nunito faces claim every codepoint [low]
- `harness/ideas/_inbox/get-onboarding-quiz-is-shipped-but-absent-from-backend-spec-.md` — GET onboarding quiz is shipped but absent from backend spec 6.1 and 1st-thinking 7 [low]
- `harness/ideas/_inbox/no-index-supports-the-quests-lookups-on-roadmaps-and-exercis.md` — No index supports the quests lookups on roadmaps and exercises [low]
- `harness/ideas/_inbox/no-test-pins-that-login-awaits-signin-s-cache-clear-before-n.md` — No test pins that login awaits signIn's cache clear before navigating to the hub [low]
- `harness/ideas/_inbox/onboarding-s-invalid-request-copy-asks-the-learner-to-fix-fi.md` — Onboarding's invalid_request copy asks the learner to fix fields the quiz step no longer shows [low]
- `harness/ideas/_inbox/pgrefreshtokensource-has-no-sentinel-for-a-missing-user-so-a.md` — PgRefreshTokenSource has no sentinel for a missing user, so a deleted user gets 500 not 409 [low]
- `harness/ideas/_inbox/refresh-token-sealing-tests-miss-the-sealer-error-path-and-a.md` — Refresh-token sealing tests miss the sealer-error path and a short-but-valid v1 payload, and one can panic instead of fail [low]
- `harness/ideas/_inbox/retro-kit-small-contract-gaps-hpbar-shows-unclamped-values-b.md` — Retro kit small contract gaps: HpBar shows unclamped values, bar steps use total not delta, band panels keep their ring, reduced QuestNode drops its cursor, MapNode missed ring is not ember [low]
- `harness/ideas/_inbox/retro-kit-tests-miss-boundaries-and-one-asserts-nothing-ches.md` — Retro kit tests miss boundaries and one asserts nothing: Chest unmount, SpeechBox mid-type unmount, empty inputs, out-of-range values [low]
- `harness/ideas/_inbox/retrobutton-keeps-its-4-px-depth-shadow-when-disabled-or-loa.md` — RetroButton keeps its 4 px depth shadow when disabled or loading [low]
- `harness/ideas/_inbox/revive-s-absolute-save-erases-a-concurrent-ontargetmet-s-20-.md` — Revive's absolute Save erases a concurrent OnTargetMet's +20 and streak [low]
- `harness/ideas/_inbox/route-falls-back-on-terminal-4xx-so-one-bad-prompt-buys-thre.md` — Route falls back on terminal 4xx so one bad prompt buys three paid provider calls [low]
- `harness/ideas/_inbox/signing-in-on-a-second-device-silently-logs-the-first-one-ou.md` — signing in on a second device silently logs the first one out [medium]
- `harness/ideas/_inbox/spec-3-2-leaves-user-id-nullable-on-three-child-tables.md` — spec 3.2 leaves user_id nullable on three child tables [low]
- `harness/ideas/_inbox/store-reset-hard-codes-the-down-migration-list-so-migration-.md` — store reset hard-codes the down-migration list so migration 0003 will silently not roll back [low]
- `harness/ideas/_inbox/synctimeout-gives-thirty-sequential-google-calls-a-two-secon.md` — SyncTimeout gives thirty sequential Google calls a two-second mean budget and no resumption [low]
- `harness/ideas/_inbox/the-api-state-cache-still-has-no-expiry-bound-so-a-stale-res.md` — The api-state cache still has no expiry bound so a stale response can be served indefinitely [low]
- `harness/ideas/_inbox/the-hourly-sweep-loads-every-pet-in-a-zone-into-memory-and-r.md` — The hourly sweep loads every pet in a zone into memory and reparses tzdata per user [low]
- `harness/ideas/_inbox/the-sealed-refresh-token-is-not-bound-to-its-users-row-so-a-.md` — The sealed refresh token is not bound to its users row, so a ciphertext can be moved to another account [low]
- `harness/ideas/_inbox/tick-sends-serially-with-a-10s-client-timeout-so-one-slow-pu.md` — Tick sends serially with a 10s client timeout so one slow push service delays every other reminder [medium]
- `harness/ideas/_inbox/two-concurrent-assessment-submits-double-spend-the-ai-and-or.md` — Two concurrent assessment submits double-spend the AI and orphan a roadmap with its 84 exercises [medium]
- `harness/ideas/_inbox/two-recordprogress-error-branches-are-uncovered-and-the-redi.md` — Two RecordProgress error branches are uncovered and the Redis-down test's comment overstates it [low]
- `harness/ideas/_inbox/web-image-runs-caddy-as-root-and-reinstalls-npm-deps-on-ever.md` — Web image runs Caddy as root and reinstalls npm deps on every source change [low]

## Planned (awaiting approval)

_none_

## Approved

- `harness/plans/2026-09-27-tap-a-hard-word-vietnamese-glosses-on-every-reading-passage-.md` — The reading passage finally renders — with tap-a-word Vietnamese glosses — Plan [high]
- `harness/plans/2026-09-27-kho-qua-de-qua-swap-one-task-for-a-level-fitted-replacement-.md` — Khó quá / Dễ quá — swap one task for a level-fitted replacement — Plan [medium]
- `harness/plans/2026-09-27-level-result-in-30-seconds-grade-first-write-the-roadmap-in-.md` — Level result in 30 seconds: grade first, write the roadmap in the background (`202 generating`, `roadmap:gen:{user_id}`, `404 roadmap_generating`, the onboarding result step and the hub drawing state) — Plan [medium]
- `harness/plans/2026-09-27-hear-it-read-aloud-for-passages-questions-and-vocabulary-wit.md` — Hear it — read-aloud for passages, questions and flashcards (`useSpeech`, `SpeakButton`) on `/learn/:id` — Plan [medium]
- `harness/plans/2026-09-27-google-tasks-tick-themselves-when-the-day-s-target-is-met-an.md` — Google Tasks tick themselves when the day's target is met, and the calendar event opens the app — Plan [medium]
- `harness/plans/2026-09-27-missed-days-are-not-lost-catch-up-on-any-earlier-day-s-tasks.md` — Missed days are not lost — "Học bù" on any earlier day, minutes count toward today — Plan [medium]
- `harness/plans/2026-09-27-install-and-remind-nudge-after-the-first-met-day-add-to-home.md` — Install-and-remind nudge — the companion asks to call you tomorrow after your first met day — Plan [medium]
- `harness/plans/2026-09-26-day-28-checkpoint-cefr-re-assessment-and-the-next-roadmap.md` — Day-28 checkpoint (backend): `POST /api/v1/roadmaps/next` re-grades the level and replaces the roadmap; `GET /quests/daily` says `roadmap_complete` — Plan [medium]
- `harness/plans/2026-09-27-ai-graded-writing-practice-through-the-essay-grading-route.md` — AI-graded writing practice (backend): `writing` practice tasks in the roadmap and `POST /api/v1/quests/writing/grade` through the `essay_grading` route — Plan [medium]
- `harness/plans/2026-09-27-spaced-repetition-vocabulary-review-in-the-daily-quest.md` — Spaced-repetition vocabulary review (backend): `vocab_reviews`, `reviews[]` on `GET /quests/daily`, `review_grades[]` on `POST /quests/progress` — Plan [low]

## Executing

_none_

## Done (last 10)

- `harness/plans/2026-09-27-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn.md` — The 404-on-patch fallback re-opens the orphan window and Sync's new doc says the Calendar half is covered — Plan [low] (review: pass-with-bugs)
- `harness/plans/2026-09-27-retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md` — Retro kit amend 2: pin the v1 aliases, ground-0 ink on growth/alert fills, scheme-split v1 green text, and SpeechBox mid-line reduced flip — Plan [high] (review: pass) (merged)
- `harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md` — Retro kit amend: the kit paints its own dark ground (StateBlock/CountdownTimer readable in light scheme), palette fallback, OS reduced motion, static sprite reactions, torch focus and the 44 px MapNode — Plan [high] (review: pass) (merged)
- `harness/plans/2026-09-27-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s.md` — Per-task AI deadline is shared across the fallback chain; a slow preferred provider starves the fallback — Plan [medium] (review: pass-with-bugs) (merged)
- `harness/plans/2026-09-27-no-per-user-subscription-cap-and-no-length-bound-on-endpoint.md` — Push subscriptions: validate `p256dh`/`auth`, cap 10 per user (evict oldest), prune after 3 consecutive send failures — Plan [medium] (review: pass-with-bugs) (merged)
- `harness/plans/2026-09-27-migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md` — Migrate's advisory lock leaks into the pool when unlock runs on a cancelled context — Plan [medium] (review: pass-with-bugs) (merged)
- `harness/plans/2026-09-27-integration-gate-tests-only-prove-the-skip-and-would-pass-if.md` — Integration gate tests only prove the skip and would pass if the gate always skipped — Plan [low] (review: pass-with-bugs) (merged)
- `harness/plans/2026-09-27-google-403-accessnotconfigured-api-disabled-still-maps-to-re.md` — Google 403 accessNotConfigured (API disabled) still maps to reauth_required, looping users through re-consent — Plan [low] (review: pass-with-bugs) (merged)
- `harness/plans/2026-09-27-executor-worktree-nested-under-claude-worktrees-while-plan-f.md` — `stale-worktrees` reads `git worktree list`, the executor records the real worktree path, and amend re-reviews count for the amended plan — Plan [low] (review: pass-with-bugs) (merged)
- `harness/plans/2026-09-27-codemap-does-not-document-the-config-and-health-packages-and.md` — CODEMAP does not document the config and health packages and still says three CI jobs — Plan [low] (review: pass) (merged)

## Failed

- `harness/plans/2026-09-27-adaptive-reminder-timing-and-pre-decay-rescue-push.md` — Pre-decay rescue push: one "your plant needs {n} more minutes" Web Push at local 22:00 on an unmet day — Plan [low]
