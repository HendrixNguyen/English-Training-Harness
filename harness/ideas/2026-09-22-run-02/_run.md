# Ideation run 2026-09-22-run-02

**Mode:** mvp
**Read:** `AGENTS.md`; `.agents/roles/ideator.md`; `harness/CODEMAP.md` (Planned backend packages, Planned frontend areas); spec `1st-thinking-architecture-doc.md` §2.1–2.2 (lines 7–66), §3.1–3.2 (68–256), §4 (258–266), §5.1–5.2 (268–332), §6.1 (334–350), §6.2 identifiers only (352–667, pseudocode per AGENTS.md), §7 (668–685, escaped heading `\#\# 7\.`), §8 (686–705); prior run `harness/ideas/2026-09-22-run-01/_run.md` (only one prior run exists); both inbox bug files. No web research (mvp mode: evidence = spec + CODEMAP). Memory skill not queried — run-01 recorded that it held nothing about this project.
**Inbox swept:**
- `harness/ideas/2026-09-22-run-02/reconcile-pet-states-stage-between-erd-and-ddl-wilted-defaul.md` — type bug, source reviewer, priority low (kept)
- `harness/ideas/2026-09-22-run-02/spec-never-states-when-the-pet-states-row-is-created.md` — type bug, source reviewer, priority low (kept)

## Proposed
MVP slices, in dependency order (`order:` 1–8):
- `harness/ideas/2026-09-22-run-02/store-go-module-postgres-and-redis-clients-migration-0001.md` — Store: Go module, Postgres and Redis clients, migration 0001 (order 1)
- `harness/ideas/2026-09-22-run-02/auth-google-oauth-code-exchange-and-jwt-sessions.md` — Auth: Google OAuth code exchange and JWT sessions (order 2)
- `harness/ideas/2026-09-22-run-02/quests-daily-quest-suite-and-progress-recording.md` — Quests: daily quest suite and progress recording (order 3)
- `harness/ideas/2026-09-22-run-02/pet-health-streak-and-stage-engine-with-revive.md` — Pet: health, streak and stage engine with revive (order 4)
- `harness/ideas/2026-09-22-run-02/ai-router-multi-llm-providers-task-strategies-and-rate-limit.md` — AI Router: multi-LLM providers, task strategies and rate limit (order 5)
- `harness/ideas/2026-09-22-run-02/google-one-way-calendar-and-tasks-sync.md` — Google: one-way Calendar and Tasks sync (order 6)
- `harness/ideas/2026-09-22-run-02/notify-web-push-subscriptions-and-delayed-reminder-queue.md` — Notify: Web Push subscriptions and delayed reminder queue (order 7)
- `harness/ideas/2026-09-22-run-02/frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md` — Frontend Shell: Nuxt 3 PWA with auth, daily quest and pet screens (order 8)

Bugs carried from the inbox (not slices):
- `harness/ideas/2026-09-22-run-02/reconcile-pet-states-stage-between-erd-and-ddl-wilted-defaul.md` — Reconcile pet_states stage between ERD and DDL (wilted, default)
- `harness/ideas/2026-09-22-run-02/spec-never-states-when-the-pet-states-row-is-created.md` — Spec never states when the pet_states row is created

## Notes
- **`onboarding` has no slice.** CODEMAP lists eight backend packages including `onboarding` (`POST /api/v1/onboarding/assessment`, `quiz:placement:*`), but this run was asked for exactly eight slices with `onboarding` omitted and `frontend-shell` as #8. Consequence: `POST /api/v1/onboarding/assessment` and the `quiz:placement:{user_id}` key are the only §7/§4 items not covered; `roadmaps`/`exercises` rows are only produced by the quests slice's dev fixture. Question for the human: add an `onboarding` slice (order 6, after airouter, before google — it needs airouter for grading and feeds google with the roadmap) in a follow-up `--mvp` run, or fold it into airouter?
- Spec gaps the executors will have to decide and record in CODEMAP (each is named in the relevant slice's Expected output): `users.target_goal NOT NULL` but unknown at login (auth inserts `''`); `JWT_SECRET` is not in the §8 env list; pet stage thresholds and decay amount are unspecified (proposed -20/day mirroring +20); the active revive challenge needs a Redis key not in §4 (`pet:revive:{user_id}`); Google event/tasklist ids have no column in §3.2 (store in `roadmap_json` or migration 0002). These are executor decisions, not new APIs.
- Cross-slice contract set here so later slices do not force re-consent: auth (2) must request `calendar.events` + `tasks` scopes with `access_type=offline`; frontend-shell (8) reuses the same list.
- The two swept bugs are spec-document fixes with `priority: low`; the pet slice (4) states how the code behaves (idempotent `ON CONFLICT DO NOTHING` row creation; `wilted` reachable at health 0) so it does not block on them.
- Considered and dropped: a separate `onboarding` slice (out of the requested list — see first note); a `cron` package (both cron users, pet decay and notify worker, hook into the single in-process worker the spec names, so no extra slice); a `GET /settings/vapid-public-key` endpoint (not in §7; runtime config instead); WebSocket for pet growth animation (§2.2 shows "REST / WebSocket" but §5.2 step 5 is a plain response).
