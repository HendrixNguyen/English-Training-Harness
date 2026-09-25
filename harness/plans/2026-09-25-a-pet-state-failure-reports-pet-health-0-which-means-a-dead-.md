---
idea: harness/ideas/_inbox/a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md
status: done
priority: medium
merged: false
branch: harness/2026-09-25-medium-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-
worktree: .worktrees/a-pet-state-failure-reports-pet-health-0-which-means-a-dead-
---
# quests: a failed pet read omits `pet_health`/`streak_count` instead of reporting a dead plant — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/_inbox/a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md`

**Goal:** `POST /api/v1/quests/progress` never fabricates a plant state. When the pet read fails after a recorded session, the response still says `200` with the progress fields, but carries no `pet_health` / `streak_count`, and the PWA keeps showing the last plant it knew.

**Why now (`priority: medium`):** Confirmed on `origin/main` today: `backend/internal/quests/service.go` `RecordProgress` does `pet = PetState{}` when `s.pet.State` fails, `ProgressResult.PetHealth`/`StreakCount` are plain `int` (`service.go:22-23`), and `frontend/stores/pet.ts` `applyProgress` writes `res.pet_health` straight into `status.health_points`. Backend spec §8 defines health `0` as a dead plant and §6.3's revive challenge as the way back; Frontend spec §5 drives the plant animation from exactly these numbers. So a transient `pet_states` read failure shows a death animation right after a successful session — the worst message the retention mechanism can send, on a `200`. Small, self-contained, no overlap with today's other four plans.

**Root cause:** the quests slice left this as an open question ("`Pet.State` failure → log and report (0, 0) … Confirm") and the zero value of `int` happens to be the one number with domain meaning.

**Design decisions:**
1. **Omit, do not substitute.** `PetHealth`/`StreakCount` become `*int` with `json:",omitempty"`; on a `State` error both stay `nil` and the JSON has neither key. The success shape is byte-for-byte what backend spec §6.2 shows (the spec wins for the success path; it does not specify the error path, and CODEMAP records the omission).
2. **Keep the 200.** The progress write is already committed; a 5xx would invite a retry and a double count. Unchanged — the idea agrees.
3. **The client skips what is absent.** `usePetStore.applyProgress` assigns each field only when it is a number; `ProgressResponse` marks both optional. No visual change on the happy path.
4. **`Pet.State`'s contract is written down:** a user with no `pet_states` row is not an error (`pet.QuestHook.State` goes through `Service.Ensure`, which creates the row); an error means the read itself failed.

**Tech stack:** Go 1.25 / Gin (backend), Nuxt 3 + Pinia + Vitest (frontend). No new dependencies.

**Run every command from the worktree root** unless a step says otherwise. `rg` is not installed — use `grep -n`. Backend unit tests run with `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL` in front.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/quests/service.go` | `ProgressResult.PetHealth`/`StreakCount` → `*int,omitempty`; nil on a failed read |
| `backend/internal/quests/pet.go` | `Pet.State` doc: no row ≠ error; a failed read omits the fields |
| `backend/internal/quests/service_test.go` | the four `PetHealth`/`StreakCount` assertions use a helper; the failure test asserts `nil` |
| `backend/internal/quests/handler_test.go` | new: the failure body has neither key; the §6.2 body test unchanged |
| `frontend/stores/quest.ts` | `ProgressResponse.pet_health?` / `streak_count?` |
| `frontend/stores/pet.ts` | `applyProgress` assigns only present numbers |
| `frontend/tests/unit/petStore.test.ts` | new: absent fields leave the plant alone; `0` still applies |
| `harness/CODEMAP.md` | `quests` paragraph: the failure shape; `shell` paragraph: one clause on `applyProgress` |

---

## Tasks

### Task 1: Backend — nil, not zero

**Files:**
- Modify: `backend/internal/quests/service.go`, `backend/internal/quests/pet.go`, `backend/internal/quests/service_test.go`, `backend/internal/quests/handler_test.go`

- [ ] **Step 1: Write the failing tests.**
  - In `service_test.go`, change `TestAPetStateFailureDoesNotFailTheRequest` (line ~305) to assert `out.DailySecondsSpent == 600 && out.PetHealth == nil && out.StreakCount == nil`, and its comment to "Pet fields are omitted, never fabricated: 0 means a dead plant (spec §8)". Add a package-level helper `func petOf(out ProgressResult) (int, int)` that derefs with `-1` for nil, and use it in the three assertions at lines ~70, ~107, ~137 (`h, s := petOf(out); if h != 80 || s != 4 {…}`).
  - In `handler_test.go`, add `TestProgressHandlerOmitsThePetFieldsWhenTheReadFails`: `h.pet.stateErr = errors.New("pet_states unreachable")`, post the §6.2 body, expect `200`, body contains `"daily_seconds_spent"` and `"is_target_met"`, and `!strings.Contains(body, "pet_health") && !strings.Contains(body, "streak_count")`.
- [ ] **Step 2: Run red:** from `backend/`: `env -u … go test ./internal/quests/ -run 'TestAPetStateFailure|TestProgressHandlerOmits' -count=1` → compile errors (`*int` vs `int`) / the handler test fails on `"pet_health":0`.
- [ ] **Step 3: Make them pass.** In `service.go`:

```go
	// §6.2's two pet fields. Pointers so a failed read is *omitted* (decision
	// below), never reported as 0 — health 0 is a dead plant in spec §8.
	PetHealth   *int `json:"pet_health,omitempty"`
	StreakCount *int `json:"streak_count,omitempty"`
```

  and at the end of `RecordProgress`:

```go
	// Also best-effort: the write is done, and a 500 here would make the client
	// retry and double-count. A failed read leaves both fields nil — omitted on
	// the wire — so the client keeps the last state it knew; GET /pet/status
	// (pet slice) is the authoritative read.
	res := ProgressResult{ /* the existing fields */ }
	if pet, err := s.pet.State(ctx, userID); err != nil {
		log.Printf("quests: reading pet state for user %s (pet fields omitted): %v", userID, err)
	} else {
		res.PetHealth, res.StreakCount = &pet.Health, &pet.Streak
	}
	return res, nil
```

  In `pet.go`, extend the `Pet` doc: "State: a user with no `pet_states` row is not an error — implementations create or default it (`pet.QuestHook` goes through `Service.Ensure`). An error means the read itself failed; `RecordProgress` then omits `pet_health`/`streak_count` rather than fabricate a dead plant."
- [ ] **Step 4: Run green:** `env -u … go test ./internal/quests/ -count=1` → `ok`; `cd backend && make check` → all `ok` under `-race`. Check `grep -rn 'PetHealth\|StreakCount' backend --include='*.go' | grep -v internal/quests` → no output (nothing outside quests reads these fields; if something does, adapt it in this task).
- [ ] **Step 5: Commit:** `git commit -am "quests: a failed pet read omits pet_health/streak_count instead of reporting 0"`.

### Task 2: Frontend — skip what is absent

**Files:**
- Modify: `frontend/stores/quest.ts`, `frontend/stores/pet.ts`, `frontend/tests/unit/petStore.test.ts`

- [ ] **Step 1: Write the failing tests** in `petStore.test.ts` next to the existing `applyProgress` case:
  - `applyProgress leaves health and streak alone when the response omits them`: load `status` (health 80, streak 4), `pet.applyProgress({})`, expect `toMatchObject({ health_points: 80, current_streak: 4 })`.
  - `applyProgress still applies a real 0`: `pet.applyProgress({ pet_health: 0, streak_count: 0 })` → `health_points: 0` (a genuine wilt from the backend must not be dropped by a falsy check).
- [ ] **Step 2: Run red:** from `frontend/`: `npm run test:unit -- petStore` → the first new case fails (`health_points: undefined`); the type check also complains once `ProgressResponse` is changed — do Step 3 in one go.
- [ ] **Step 3: Make them pass.** `quest.ts`: `pet_health?: number` / `streak_count?: number` with a comment "omitted when the backend's pet read failed (CODEMAP quests); never 0-for-unknown". `pet.ts`:

```ts
    /** §6.2 progress response. Either field is absent when the backend's pet read failed; keep the last known plant then. */
    applyProgress(res: { pet_health?: number, streak_count?: number }) {
      if (!this.status) return
      if (typeof res.pet_health === 'number') this.status.health_points = res.pet_health
      if (typeof res.streak_count === 'number') this.status.current_streak = res.streak_count
    },
```

- [ ] **Step 4: Run green:** `npm run lint && npm run typecheck && npm run test:unit` → clean; `pages/learn/[id].vue` still compiles (it passes the whole `ProgressResponse`).
- [ ] **Step 5: Commit:** `git commit -am "frontend: applyProgress keeps the last plant when the progress response omits the pet fields"`.

### Task 3: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md`

- [ ] **Step 1:** `quests` paragraph — after "answers `{daily_seconds_spent, daily_minutes_spent, is_target_met, pet_health, streak_count}`", add: "(`pet_health`/`streak_count` are `*int,omitempty`: when `Pet.State` fails after the write they are **omitted**, never `0` — health 0 is a dead plant in spec §8 — and the failure is logged; the 200 stays because the session is already recorded)". `shell` paragraph — in the `stores/pet.ts` clause add "`applyProgress` assigns only the pet fields the progress response carries".
- [ ] **Step 2:** `python3 tools/harness/cli.py validate` → 0. Commit: `git commit -am "harness: CODEMAP — omitted pet fields on a failed read"`.

---

## Verification

```bash
cd backend
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./internal/quests/ -count=1 -race -v 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
# expect: one `ok` line, no FAIL
grep -n 'omitempty' internal/quests/service.go
# expect: 2 lines (pet_health, streak_count)
grep -rn 'PetState{}' internal/quests/service.go
# expect: no output (the zero-value substitution is gone)
make check
# expect: fmt-check silent, vet silent, ok for every package under -race
cd ../frontend
npm run lint && npm run typecheck && npm run test:unit
# expect: all clean; petStore.test.ts shows 2 new passing cases
grep -n 'pet_health?' stores/quest.ts
# expect: 1 line
cd ..
grep -n 'omitempty\|omitted' harness/CODEMAP.md
# expect: the quests sentence
git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP
# expect: no output
python3 tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0
gh run list --branch "$(git branch --show-current)" --limit 1
# expect: backend-unit, backend-integration, harness-tooling, frontend green
```

Mutation check (record the result): change `if typeof res.pet_health === 'number'` to `if (res.pet_health)` in `pet.ts` → the "still applies a real 0" case must go red; revert.

## Notes and open questions

- **Why not carry the last known value server-side?** quests has no cache of pet state and must not read `pet_states` (CODEMAP boundary). Omission plus the client's last known state gives the same outcome with no new state.
- **`GET /quests/daily` is untouched** — it never carried pet fields; the hub reads `GET /pet/status`.
- **The §6.2 sample body still applies** to the success path; `TestProgressHandlerReturnsTheSpec62Body` stays as the contract test.
- **The hub self-heals:** `pages/index.vue` calls `pet.load()` in `onMounted` (confirmed on `origin/main`), so after `/learn/:id` navigates back the plant is re-read from `GET /pet/status` anyway; the omitted fields only avoid the flash of a dead plant in between. No change there.

## Execution summary

**Built:** all three tasks, in order, each with a failing test written first.

- Task 1 (backend): `service.go`'s `PetHealth`/`StreakCount` → `*int,omitempty`; `RecordProgress` now leaves both nil (and logs) on a failed `Pet.State` read instead of substituting `PetState{}`. `pet.go`'s `Pet` doc extended per the plan. `service_test.go` got the `petOf` helper (derefs to `-1` for nil) used at the three existing assertions plus the rewritten `TestAPetStateFailureDoesNotFailTheRequest`. `handler_test.go` got `TestProgressHandlerOmitsThePetFieldsWhenTheReadFails`.
- Task 2 (frontend): `quest.ts`'s `ProgressResponse.pet_health`/`streak_count` → optional; `pet.ts`'s `applyProgress` assigns each field only when `typeof … === 'number'`. `petStore.test.ts` got the two new cases from the plan.
- Task 3: `harness/CODEMAP.md`'s `quests` and `shell` paragraphs updated per the plan.

**Deviations from the plan's file list (both required for the build/tests to pass, not optional polish):**
1. `backend/internal/quests/integration_test.go` (not in the plan's File structure table) also referenced `out.PetHealth`/`out.StreakCount` as plain `int` and failed to compile after the `*int` change. Fixed with the same `petOf` helper the plan introduced for `service_test.go` (one assertion, `TestIntegrationEnsureCreatesExactlyOnePetRow`'s neighbor).
2. `frontend/tests/unit/petStore.test.ts`'s existing `applyProgress updates health and streak…` case, and the two new cases, all read the module-level `status` fixture object by reference via `api.get.mockResolvedValue(status)`. `applyProgress` mutates `this.status` (== that same object) in place, so the first `applyProgress` test permanently corrupted the shared fixture (80/5 → 100/6) for every test running after it in the same file. Fixed by passing `{ ...status }` (a fresh copy) in all three `applyProgress` tests, so each is independent of execution order.

**Verification (plan's Verification section, all as specified):**
- `go test ./internal/quests/ -count=1 -race -v` → `ok`, no FAIL.
- `grep -n 'omitempty' internal/quests/service.go` → 2 lines (`pet_health`, `streak_count`).
- `grep -rn 'PetState{}' internal/quests/service.go` → no output.
- `make check` (fmt-check, vet, `go test ./... -race`) → all 13 packages `ok`.
- `npm run lint && npm run typecheck && npm run test:unit` → all clean; 16 files / 80 tests, `petStore.test.ts` at 8 (2 new + the pre-existing case now isolated from the shared-fixture bug above).
- `grep -n 'pet_health?' stores/quest.ts` → 1 line.
- `grep -n 'omitempty\|omitted' harness/CODEMAP.md` → the quests sentence.
- `git diff --stat origin/main...HEAD -- harness/` → only `harness/CODEMAP.md` (2 insertions, 2 deletions).
- `python3 tools/harness/cli.py validate` → `exit=0`.
- Mutation check: flipped `typeof res.pet_health === 'number'` to `if (res.pet_health)` in `pet.ts` → "applyProgress still applies a real 0" went red as expected (`health_points: 80` instead of `0`); reverted, confirmed clean (`git status --short` empty).

**Runtime proof (step 8, beyond the plan's own Verification):**
- **Build:** `go build ./...` → clean.
- **Full test suite from a clean shell:** `make check` (`env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL` implied by the shell used) → all 13 backend packages `ok` under `-race`; frontend `npm run lint && npm run typecheck && npm run test:unit` → clean, 80/80.
- **Boot + real infra, scratch Docker project `bf-pet` (`COMPOSE_PROJECT_NAME=bf-pet`, `POSTGRES_PORT=55441`, `REDIS_PORT=56441`, scratch `backend/.env`):**
  - `docker compose -p bf-pet up -d --wait --wait-timeout 120` → both `postgres`/`redis` healthy.
  - Gated integration suite against the real services: `go test ./... -count=1 -v -run Integration -p 1` with `TEST_DATABASE_URL`/`TEST_REDIS_URL` pointed at the scratch stack → all pass, including `TestIntegrationDailyAndProgressAgainstRealServices` (the exact `RecordProgress` path this plan changed, against a real Postgres + Redis).
  - Booted the real binary: `go run ./cmd/api` with the scratch env (`JWT_SECRET`, `ENCRYPTION_SECRET_KEY`, `GOOGLE_CLIENT_ID/SECRET` scratch values) → log shows `migrations applied: [0001_init 0002_google_sync 0003_pet_verdict_dates]` then `listening on [::]:8080`. `curl http://localhost:8080/healthz` → `200 {"postgres":"ok","redis":"ok","status":"ok"}`.
  - **Cleanup:** killed the server (parent `go run` PID and the compiled child binary — `go run` spawns a child, so both had to be killed), confirmed with `pgrep -fl cmd/api`/`pgrep -fl go-build.*api` (both empty) and a failed `curl` (connection refused). `docker compose -p bf-pet down` removed both containers, the network, and the named volume was untouched (no `-v`, matches `make down`'s semantics). Removed the scratch `backend/.env`. `docker ps --filter name=bf-pet` → empty.

**CI on the pushed branch:** `harness/2026-09-25-medium-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-`
Run https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36093609127 — **success**. All four jobs green: `frontend` (33s), `backend-unit` (1m12s), `backend-integration` (46s), `harness-tooling` (8s).
