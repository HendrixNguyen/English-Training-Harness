---
idea: harness/ideas/2026-09-25-run-01/name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o.md
status: executing
priority: medium
merged: false
design: harness/designs/plant-name.md
branch: harness/2026-09-26-medium-name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o
worktree: .worktrees/name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o
---
# Name your plant at onboarding and see it greet you by name on the hub — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/2026-09-25-run-01/name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o.md`
**Design:** `harness/designs/plant-name.md`

**Goal:** A learner can name their plant on the onboarding goal step (or skip it and get "Mầm Non"), the name is stored once in `pet_states.plant_name` through the existing assessment request, and the result step, the hub, the wilted banner, the `/revive` heading and two speech-bubble lines say that name instead of the DDL's English "My Green Buddy".

**Why now (`priority: medium`):** Confirmed on `origin/main` today: `pet_states.plant_name` is only ever the DDL default (`backend/internal/pet/repo.go` `ensureSQL` inserts `(user_id)` alone; `stateColumns` reads `COALESCE(p.plant_name, 'My Green Buddy')`), `onboarding.AssessmentRequest` has no name field, and the first Vietnamese screen after the quiz renders `frontend/pages/onboarding.vue:143` "{{ result.pet_state.plant_name }} đã nảy mầm" → "My Green Buddy đã nảy mầm". The plant is the retention mechanism (1st-thinking §1); naming is the cheapest ownership mechanic and this fixes a visible localisation wart on the happy path. Feature slot 5 of 5 today (two-cap rule, owner 2026-09-25).

**Design decisions (taken by the evaluator; do not re-litigate):**
1. **Wire:** `AssessmentRequest` gains optional `plant_name`. `validate()` trims it and rejects a trimmed length outside 1..30 **runes** (`utf8.RuneCountInString`) with the existing `ErrInvalidRequest` → `400 invalid_request`; blank/absent is valid and means "default". Length only on the server — the letters/digits/spaces character class is enforced inline by the client (design §1), so a later rename endpoint inherits one server rule.
2. **Default lives in `onboarding`:** `const DefaultPlantName = "Mầm Non"`, applied on the create path when the trimmed name is empty. The DDL default `'My Green Buddy'` is untouched (backend spec §3.2 must equal `0001_init`; no migration).
3. **`onboarding.Pet` is extended, not siblinged:** `Ensure(ctx, userID, plantName string)`. The interface has one method, one adapter (`petForOnboarding` in `main.go`) and one fake, so widening it is the smaller change; a sibling would double all three. Contract: `plantName == ""` creates the row if missing and never touches an existing name — the **re-submit path passes `""`** and stays a no-op by decision (the selected inbox bug `a-re-submitted-assessment-silently-discards-target-goal-noti.md` owns that path).
4. **`pet` gains a sibling, not a wider `Ensure`:** `Service.EnsureNamed(ctx, userID, plantName) (State, error)` and `Repo.EnsureNamed(ctx, userID, plantName) error`. `Service.Ensure` has four in-package callers (`handler`, `questhook`, `Revive`, `OnTargetMet`) and dozens of tests; they stay untouched. `PgRepo.EnsureNamed` runs `ensureSQL` when the name is empty, otherwise
   `INSERT INTO pet_states (user_id, plant_name) VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET plant_name = EXCLUDED.plant_name WHERE pet_states.plant_name IS DISTINCT FROM EXCLUDED.plant_name` — idempotent, still 1:1 (UNIQUE `user_id`), a no-op write when the name is unchanged. `pet` stores what it is given; length is the caller's rule (the column is VARCHAR(100)).
5. **Specs:** backend spec §6.1 request example gains `"plant_name": "Mầm Non"`, its description names the field and rule, and its 201 example's `plant_name` becomes `"Mầm Non"` so the one exchange stays consistent; 1st-thinking §7's `POST /api/v1/onboarding/assessment` bullet mentions the optional `plant\_name` in the document's escaped style. Backend spec wins for the wire. §6.3's `GET /pet/status` example keeps "My Green Buddy" (a pet created by `GET /pet/status` before onboarding still has the DDL default — true today, true after).
6. **Frontend per design:** one optional field on the goal step, trimmed, `^[\p{L}\p{N} ]{1,30}$`, rejected inline before submit (`aria-invalid`, `role="note"`, start button disabled — same pattern as the reminder time); `plant_name` is put in the body **only when set**; the result step reads the name from the response. `speechLine` gains an optional `name` whose default keeps every current output byte-identical. Hub: a caption line above the plant, the banner, and the bubble call. `/revive`: the alert band.
7. **Same-day overlaps:** the growth-moment plan also edits `utils/plant.ts` `speechLine` (new inputs) and `pages/index.vue`; the streak-shield plan edits `pages/index.vue` and `stores/pet.ts`. This plan's `speechLine` change is one added parameter with a default, and its `index.vue` change is three one-line edits. **Execute this plan after those two** (sync from `origin/main` first; if `speechLine` already has more inputs, add `name` beside them).

**Tech stack:** Go 1.25 / Gin / pgx (backend), Nuxt 3 + Vue 3 + Pinia + Vitest (frontend). No new dependencies.

**Run every command from the worktree root** unless a step says otherwise; `backend/` and `frontend/` commands say so. `rg` and `timeout` are not installed — use `grep -n`; bound Go with `-timeout`. Backend unit tests run with `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL` in front so nothing touches a live service. Frontend checks are `npm run lint && npm run typecheck && npm run test:unit` from `frontend/`.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/onboarding/types.go` | `AssessmentRequest.PlantName` (`json:"plant_name"`); `Pet.Ensure` takes `plantName`; doc the `""` contract |
| `backend/internal/onboarding/service.go` | `DefaultPlantName`, `MaxPlantNameRunes = 30`; `validate()` length rule; create path resolves the name, re-submit path passes `""` |
| `backend/internal/onboarding/fakes_test.go` | `fakePet` records every name and echoes a non-empty one into `state.PlantName` |
| `backend/internal/onboarding/service_test.go` | validation bounds; `TestAssessNamesThePlant`; the happy/idempotent tests assert the recorded names |
| `backend/internal/onboarding/handler_test.go` | `spec61Request` and the 201 body carry "Mầm Non"; one 400 case for a 31-rune name |
| `backend/internal/pet/repo.go` | `Repo.EnsureNamed` + `ensureNamedSQL`; `PgRepo.EnsureNamed` |
| `backend/internal/pet/service.go` | `Service.EnsureNamed` |
| `backend/internal/pet/fakes_test.go` | `fakeRepo.EnsureNamed` |
| `backend/internal/pet/service_test.go` | `TestEnsureNamedSetsTheNameAndAnEmptyNameKeepsIt` |
| `backend/internal/pet/integration_test.go` | **new** `TestIntegrationEnsureNamedWritesOnceAndKeepsTheNameOnRepeat` (gated on `TEST_DATABASE_URL`) |
| `backend/cmd/api/main.go` | `petForOnboarding.Ensure(ctx, userID, plantName)` → `svc.EnsureNamed` |
| `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` | §6.1 description, request and 201 examples |
| `project-base/1st-thinking-architecture-doc.md` | §7 assessment bullet |
| `frontend/composables/useOnboardingApi.ts` | `AssessmentRequest.plant_name?: string` |
| `frontend/pages/onboarding.vue` | the field, validation, request body, error copy |
| `frontend/utils/plant.ts` | `speechLine` `name?` with default |
| `frontend/pages/index.vue` | caption line, banner text, bubble call |
| `frontend/pages/revive.vue` | alert band text |
| `frontend/tests/unit/onboardingPage.test.ts` | field validation; body carries `plant_name` only when set |
| `frontend/tests/unit/plant.test.ts` | `speechLine` with and without a name |
| `frontend/tests/unit/revivePage.test.ts` | the alarm names the plant |
| `harness/CODEMAP.md` | `onboarding`, `pet`, `shell` paragraphs |

---

## Tasks

### Task 1: `pet` — `EnsureNamed` (repo, service, fake, unit test)

**Files:**
- Modify: `backend/internal/pet/repo.go`, `backend/internal/pet/service.go`, `backend/internal/pet/fakes_test.go`, `backend/internal/pet/service_test.go`

- [ ] **Step 1: Write the failing test** in `service_test.go` after `TestEnsureCreatesTheRowOnceAndReturnsTheDefaults`:

```go
func TestEnsureNamedSetsTheNameAndAnEmptyNameKeepsIt(t *testing.T) {
	h := newHarness(sept22)

	st, err := h.svc.EnsureNamed(ctx, "u1", "Mầm Non")
	if err != nil || st.PlantName != "Mầm Non" || st.HealthPoints != 100 {
		t.Fatalf("EnsureNamed = %+v, %v; want a fresh row named Mầm Non", st, err)
	}
	// "" is onboarding's re-submit path: create if missing, never rename.
	if st, err = h.svc.EnsureNamed(ctx, "u1", ""); err != nil || st.PlantName != "Mầm Non" {
		t.Errorf("EnsureNamed(\"\") = %+v, %v; want the name kept", st, err)
	}
	if st, err = h.svc.EnsureNamed(ctx, "u1", "Lá Xanh"); err != nil || st.PlantName != "Lá Xanh" {
		t.Errorf("EnsureNamed(Lá Xanh) = %+v, %v; want the name replaced", st, err)
	}
	if len(h.repo.states) != 1 {
		t.Errorf("rows = %d, want 1 (still 1:1)", len(h.repo.states))
	}
	if _, err := h.svc.EnsureNamed(ctx, "u2", ""); err != nil || h.repo.states["u2"].PlantName != "My Green Buddy" {
		t.Errorf("EnsureNamed(\"\") on a missing row must create it with the DDL default; got %+v, %v", h.repo.states["u2"], err)
	}
}
```

- [ ] **Step 2: Run red:** from `backend/`: `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./internal/pet/ -run TestEnsureNamed -count=1` → compile error (`h.svc.EnsureNamed undefined`).
- [ ] **Step 3: Make it pass.**
  - `repo.go`, in the `Repo` interface right after `Ensure`:

```go
	// EnsureNamed is Ensure plus a name: creates the row with plantName, or
	// renames an existing row when plantName differs. An empty plantName is
	// exactly Ensure — the row is created if missing and an existing name is
	// never touched (onboarding's re-submit path). Length is the caller's
	// rule; the column is VARCHAR(100).
	EnsureNamed(ctx context.Context, userID, plantName string) error
```

  - the constant, next to `ensureSQL`:

```go
	// The WHERE makes an unchanged name a no-op write; UNIQUE(user_id) keeps it 1:1.
	ensureNamedSQL = `
INSERT INTO pet_states (user_id, plant_name) VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET plant_name = EXCLUDED.plant_name
WHERE pet_states.plant_name IS DISTINCT FROM EXCLUDED.plant_name`
```

  - `PgRepo.EnsureNamed` after `PgRepo.Ensure`:

```go
func (r *PgRepo) EnsureNamed(ctx context.Context, userID, plantName string) error {
	if plantName == "" {
		return r.Ensure(ctx, userID)
	}
	if _, err := r.Pool.Exec(ctx, ensureNamedSQL, userID, plantName); err != nil {
		return fmt.Errorf("pet: ensuring named pet_states row: %w", err)
	}
	return nil
}
```

  - `service.go`, after `Ensure`:

```go
// EnsureNamed is Ensure with the learner's chosen name (onboarding's create
// path). An empty name is exactly Ensure. See Repo.EnsureNamed.
func (s *Service) EnsureNamed(ctx context.Context, userID, plantName string) (State, error) {
	if err := s.repo.EnsureNamed(ctx, userID, plantName); err != nil {
		return State{}, err
	}
	return s.repo.Get(ctx, userID)
}
```

  - `fakes_test.go`: `fakeRepo.EnsureNamed` = the body of `Ensure` (same `ensureErr`, `ensured++`, create with `defaultState`) plus `if plantName != "" { st := f.states[userID]; st.PlantName = plantName; f.states[userID] = st }` under the same lock.
- [ ] **Step 4: Run green:** `env -u … go test ./internal/pet/ -count=1 -race` → `ok`. `go vet ./...` clean.
- [ ] **Step 5: Commit:** `git commit -am "pet: EnsureNamed creates or renames the 1:1 row; empty name is Ensure"`.

### Task 2: `pet` — integration test for the SQL

**Files:**
- Modify: `backend/internal/pet/integration_test.go`

- [ ] **Step 1: Add** `TestIntegrationEnsureNamedWritesOnceAndKeepsTheNameOnRepeat` next to `TestIntegrationEnsureCreatesExactlyOnePetRow`, same gate/comment/user-seeding shape (`gid = "google-pet-named-integration"`, its own `DELETE … WHERE google_id` cleanup):
  1. eight concurrent `svc.EnsureNamed(ctx, userID, "Mầm Non")` → zero errors, `SELECT count(*) FROM pet_states WHERE user_id = $1` = 1, `repo.Get` → `PlantName == "Mầm Non"`, `HealthPoints == 100`, `Stage == StageSprout`;
  2. `svc.EnsureNamed(ctx, userID, "")` → name still "Mầm Non";
  3. `svc.EnsureNamed(ctx, userID, "Lá Xanh")` → "Lá Xanh", still one row;
  4. `svc.EnsureNamed(ctx, userID, "Lá Xanh")` again → `pg.Pool.Exec(ctx, ensureNamedSQL, …)` directly and assert `RowsAffected() == 0` (the `IS DISTINCT FROM` guard makes the repeat a no-op write).
- [ ] **Step 2: Run:** without `TEST_DATABASE_URL` it must `--- SKIP` with the standard message: `env -u … go test ./internal/pet/ -run TestIntegrationEnsureNamed -count=1 -v | grep -E 'SKIP|ok'`. With the dev stack (`COMPOSE_PROJECT_NAME=<slug>` in the scratch `backend/.env`, `make up`, export `TEST_DATABASE_URL`): `go test ./internal/pet/ -run 'TestIntegrationEnsure' -count=1 -p 1 -v -timeout 120s` → both pass; `make down` the same project after. Record which of the two you ran in the plan's `## Verification` notes on the branch commit message.
- [ ] **Step 3: Commit:** `git commit -am "pet: integration test — EnsureNamed writes once and keeps the name on repeat"`.

### Task 3: `onboarding` — the field, the default, the interface

**Files:**
- Modify: `backend/internal/onboarding/types.go`, `backend/internal/onboarding/service.go`, `backend/internal/onboarding/fakes_test.go`, `backend/internal/onboarding/service_test.go`, `backend/internal/onboarding/handler_test.go`

- [ ] **Step 1: Write the failing tests.**
  - `fakes_test.go`: `fakePet` gains `names []string`; `Ensure(_ context.Context, _ string, plantName string)` appends `plantName`, increments `ensured`, and when `plantName != ""` returns `f.state` with `PlantName = plantName` (so the response reflects the write, as the real adapter does).
  - `service_test.go`:
    - in `TestAssessHappyPathGradesGeneratesAndPersistsOnce`, after the `h.pet.ensured != 1` check: `if got := h.pet.names; len(got) != 1 || got[0] != DefaultPlantName { t.Errorf("pet.Ensure names = %q, want [%q] (blank plant_name → the Vietnamese default)", got, DefaultPlantName) }` and change the `PetState` expectation to `PlantName: "Mầm Non"`.
    - in `TestAssessIsIdempotentWhileARoadmapIsActive`: set `req.PlantName = "Lá Xanh"` on the request and assert `h.pet.names[0] == ""` — "the re-submit path never renames (deliberate; see the selected inbox bug)".
    - in `TestAssessValidatesTheRequestBeforeTouchingAnything` add `"plant name too long": func(r *AssessmentRequest) { r.PlantName = strings.Repeat("ă", 31) }` (a multi-byte rune, so a byte-length check would wrongly reject 11 of them — it must count runes).
    - new table test:

```go
func TestAssessNamesThePlant(t *testing.T) {
	cases := map[string]struct{ in, want string }{
		"absent":          {"", DefaultPlantName},
		"whitespace only": {"   ", DefaultPlantName},
		"trimmed":         {"  Lá Xanh  ", "Lá Xanh"},
		"one rune":        {"A", "A"},
		"thirty runes":    {strings.Repeat("ă", 30), strings.Repeat("ă", 30)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			req := validRequest()
			req.PlantName = tc.in
			out, err := h.svc.Assess(ctx, "u1", req)
			if err != nil {
				t.Fatalf("Assess: %v", err)
			}
			if h.pet.names[0] != tc.want || out.PetState.PlantName != tc.want {
				t.Errorf("ensured %q, response %q; want %q", h.pet.names[0], out.PetState.PlantName, tc.want)
			}
		})
	}
}
```

  - `handler_test.go`: `spec61Request` gains `"plant_name": "Mầm Non", ` after `"timezone": …` (mirror the spec edit in Task 5) and the `want` body in `TestAssessmentReturns201WithTheSpec61Body` becomes `"plant_name":"Mầm Non"`; add to `TestAssessmentErrorMapping`: `{"plant name too long", nil, strings.Replace(spec61Request, `"Mầm Non"`, `"`+strings.Repeat("a", 31)+`"`, 1), 400, "invalid_request"}`.
- [ ] **Step 2: Run red:** from `backend/`: `env -u … go test ./internal/onboarding/ -count=1` → compile errors (`PlantName`, `DefaultPlantName`, the fake's signature).
- [ ] **Step 3: Make them pass.**
  - `types.go`: `PlantName string \`json:"plant_name"\`` on `AssessmentRequest` with the comment `// Optional; trimmed, 1..MaxPlantNameRunes runes, blank → DefaultPlantName.`; `Pet` becomes

```go
// Pet creates the pet_states row idempotently and reports it. plantName is
// the name to create the row with (or rename it to); "" creates the row if
// missing and never touches an existing name — the re-submit path passes "".
// Satisfied in cmd/api/main.go by an adapter over *pet.Service.EnsureNamed.
type Pet interface {
	Ensure(ctx context.Context, userID, plantName string) (PetState, error)
}
```

  - `service.go`: next to `DailyMinutes`:

```go
// DefaultPlantName is the name a blank plant_name gets. Decided here, not by
// the DDL default ('My Green Buddy', spec §3.2 — kept, since it must equal
// 0001_init), because the product speaks Vietnamese.
const DefaultPlantName = "Mầm Non"

// MaxPlantNameRunes bounds plant_name after trimming (runes, not bytes).
const MaxPlantNameRunes = 30
```

    In `validate()`, after the goal check: `if n := utf8.RuneCountInString(strings.TrimSpace(req.PlantName)); n > MaxPlantNameRunes { return fmt.Errorf("%w: plant_name must be at most %d characters", ErrInvalidRequest, MaxPlantNameRunes) }` (import `unicode/utf8`). Re-submit path: `s.pet.Ensure(ctx, userID, "")` with the comment `// Deliberately no rename: the active-roadmap path writes nothing (see the inbox bug on re-submits).` Create path: `s.pet.Ensure(ctx, userID, plantNameOrDefault(req.PlantName))` with

```go
func plantNameOrDefault(raw string) string {
	if name := strings.TrimSpace(raw); name != "" {
		return name
	}
	return DefaultPlantName
}
```

- [ ] **Step 4: Run green:** `env -u … go test ./internal/onboarding/ -count=1 -race` → `ok`. `go build ./...` now fails only in `cmd/api` (`petForOnboarding` no longer satisfies `onboarding.Pet`) — fixed in Task 4; run `go vet ./internal/onboarding/` for now.
- [ ] **Step 5: Commit:** `git commit -am "onboarding: optional plant_name (1..30 runes, blank → Mầm Non) passed to Pet.Ensure; re-submit never renames"`.

### Task 4: `cmd/api` — the adapter

**Files:**
- Modify: `backend/cmd/api/main.go`

- [ ] **Step 1:** `petForOnboarding.Ensure(ctx context.Context, userID, plantName string)` calls `p.svc.EnsureNamed(ctx, userID, plantName)`; update its comment ("adapts `*pet.Service.EnsureNamed`").
- [ ] **Step 2: Run:** from `backend/`: `make check` → `gofmt -l` silent, `go vet` silent, every package `ok` under `-race`. `grep -rn 'Ensure(' internal/onboarding/*.go cmd/api/main.go | grep -v _test` → the two service call sites and the adapter, all three-argument.
- [ ] **Step 3: Commit:** `git commit -am "cmd/api: petForOnboarding carries the plant name to pet.EnsureNamed"`.

### Task 5: Specs

**Files:**
- Modify: `project-base/Adaptive English Learning Platform - Backend Technical Specification.md`, `project-base/1st-thinking-architecture-doc.md`

- [ ] **Step 1: Backend spec §6.1** (`grep -n 'onboarding/assessment' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md"` → the bullet at ~269): append to the Description line: ` plant_name is optional — trimmed, 1–30 characters; blank or absent names the plant "Mầm Non" (the DDL default is not used by this endpoint).` In the request example insert `"plant_name": "Mầm Non", ` after `"timezone": "Asia/Ho_Chi_Minh", `; in the 201 example change `"plant_name": "My Green Buddy"` to `"plant_name": "Mầm Non"`. Leave §6.3's example and the §3.2 DDL untouched.
- [ ] **Step 2: 1st-thinking §7** (`grep -n 'POST /api/v1/onboarding/assessment' project-base/1st-thinking-architecture-doc.md` → line ~672): the bullet becomes `\* \*\*POST /api/v1/onboarding/assessment\*\*: Submits placement quiz answers and an optional plant\_name, grades level, triggers AI roadmap generation.` (escaped style, as the file writes `target\_goal`).
- [ ] **Step 3: Check:** `grep -c 'plant_name' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md"` → 6 (was 4); `grep -c 'plant\\_name' project-base/1st-thinking-architecture-doc.md` → 3 (was 2); `diff <(sed -n '/^```sql/,/^```/p' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md" | grep -n 'plant_name') <(grep -n 'plant_name' backend/internal/store/migrations/0001_init.up.sql)` shows only line-number noise — the DDL text is unchanged.
- [ ] **Step 4: Commit:** `git commit -am "spec: §6.1 assessment request carries optional plant_name; §7 bullet"`.

### Task 6: Frontend — the field and the request

**Files:**
- Modify: `frontend/composables/useOnboardingApi.ts`, `frontend/pages/onboarding.vue`, `frontend/tests/unit/onboardingPage.test.ts`

- [ ] **Step 1: Write the failing tests** in `onboardingPage.test.ts`:
  - `'rejects a plant name that is too long or has symbols inline, and accepts a trimmed plain one'`: mount, click "IELTS 7.0", `await w.find('input[name="plant_name"]').setValue('a'.repeat(31))` → start button `disabled` defined and `w.find('[role="note"]').text()` contains `'Tên cây'`; `setValue('Mầm Non!')` → still disabled with the note; `setValue('  Mầm Non ')` → enabled, no note; `setValue('   ')` → enabled, no note.
  - `'sends plant_name trimmed when the learner typed one'`: `api.post.mockResolvedValue(ASSESSED)`; mount; set the field to `'  Lá Xanh '`; `completeQuiz(w)`; `expect(body).toMatchObject({ plant_name: 'Lá Xanh' })`.
  - The existing `'posts the §6.1 assessment body …'` test already asserts the exact body **without** `plant_name` when the field is blank, and `'Cây Thử đã nảy mầm'` already proves the result step uses the returned name — leave both as they are; they are the "only when set" and "response, not field" proofs.
- [ ] **Step 2: Run red:** from `frontend/`: `npm run test:unit -- onboardingPage` → the two new cases fail (no such input).
- [ ] **Step 3: Make them pass.**
  - `useOnboardingApi.ts`: `plant_name?: string` on `AssessmentRequest` with the comment `/** Optional; trimmed 1–30 chars; omitted when blank → server default "Mầm Non". */`.
  - `onboarding.vue` script: `const plantName = ref('')`; `const PLANT_NAME_RE = /^[\p{L}\p{N} ]{1,30}$/u`; `const plantNameTrimmed = computed(() => plantName.value.trim())`; `const plantNameValid = computed(() => plantNameTrimmed.value === '' || PLANT_NAME_RE.test(plantNameTrimmed.value))`; `canStart` adds `&& plantNameValid.value`; in `next()` the body spreads `...(plantNameTrimmed.value ? { plant_name: plantNameTrimmed.value } : {})`. `assessErrorMessage`'s `invalid_request` copy becomes `'Máy chủ không nhận thông tin đã gửi. Kiểm tra lại mục tiêu, tên cây và giờ nhắc học rồi thử lại.'` (the existing test asserts `'giờ nhắc học'`, still true).
  - Template, between the goal cards and the time label (design §1):

```vue
      <label class="mt-6 block">
        <span class="text-sm text-mute">Đặt tên cho cây của bạn (không bắt buộc)</span>
        <input v-model="plantName" name="plant_name" type="text" maxlength="30" autocomplete="off" enterkeyhint="done" placeholder="Mầm Non" :aria-invalid="!plantNameValid || undefined" :aria-describedby="plantNameValid ? undefined : 'plant-name-note'" class="mt-1 block w-full rounded-btn border border-ink/15 bg-transparent px-3 py-2 dark:border-paper/15">
        <span v-if="!plantNameValid" id="plant-name-note" class="mt-1 block text-sm text-alert" role="note">Tên cây dài 1–30 ký tự, chỉ gồm chữ, số và dấu cách.</span>
      </label>
```

- [ ] **Step 4: Run green:** `npm run lint && npm run typecheck && npm run test:unit` → clean; `onboardingPage.test.ts` shows 8 cases (was 6).
- [ ] **Step 5: Commit:** `git commit -am "frontend: optional plant name on the onboarding goal step, validated inline, sent only when set"`.

### Task 7: Frontend — the name is spoken (hub, revive, bubble)

**Files:**
- Modify: `frontend/utils/plant.ts`, `frontend/pages/index.vue`, `frontend/pages/revive.vue`, `frontend/tests/unit/plant.test.ts`, `frontend/tests/unit/revivePage.test.ts`

- [ ] **Step 0: Overlap check.** `git fetch origin main && git merge origin/main --no-edit` first. `grep -n 'export function speechLine' utils/plant.ts` — if the growth-moment plan has landed, its input object is wider; add `name` beside the existing inputs and keep every existing branch's text unchanged.
- [ ] **Step 1: Write the failing tests.**
  - `plant.test.ts`, new case `'speaks the plant's name where "tớ" would be, and stays byte-identical without one'`: `speechLine({ stage: 'sapling', health: 10, targetMet: true, name: 'Mầm Non' })` → `'Cảm ơn bạn, hôm nay Mầm Non đủ nước rồi 🌿'`; `speechLine({ stage: 'sapling', health: 10, targetMet: false, name: 'Mầm Non' })` → `'Mầm Non sắp héo mất! Học một chút nhé?'`; `speechLine({ stage: 'sprout', health: 80, targetMet: false, name: 'Mầm Non' })` → `'Tưới cho tớ 10 phút học đi!'` (unchanged line); `name: '  '` behaves as no name. The existing four expectations stay untouched — they are the byte-identical proof.
  - `revivePage.test.ts`: in the existing test that reaches the wilted alarm (`'retry reloads the status and then shows the real wilted state'`), add `expect(w.find('[role="alert"]').text()).toContain('My Green Buddy đang bị héo rũ')` (the `WILTED` fixture's name).
- [ ] **Step 2: Run red:** `npm run test:unit -- plant revivePage` → the new expectations fail.
- [ ] **Step 3: Make them pass.**
  - `plant.ts`:

```ts
/** `name` replaces "tớ" in the two lines that address the plant by name (design plant-name §2); blank keeps every line as before. */
export function speechLine(o: { stage: string, health: number, targetMet: boolean, name?: string }): string {
  const name = o.name?.trim() || 'tớ'
  const Name = name === 'tớ' ? 'Tớ' : name
  if (o.stage === 'wilted' || o.health <= 0) return '…'
  if (o.targetMet) return `Cảm ơn bạn, hôm nay ${name} đủ nước rồi 🌿`
  if (o.health >= 60) return 'Tưới cho tớ 10 phút học đi!'
  if (o.health >= 30) return 'Tớ hơi khát rồi… 10 phút thôi?'
  return `${Name} sắp héo mất! Học một chút nhé?`
}
```

  - `index.vue` — three one-line edits: the bubble call adds `name: pet.status.plant_name` as its last property; the banner span becomes `<span>⚠️ {{ pet.status?.plant_name || 'Cây xanh' }} đang bị héo rũ!</span>`; one new line directly above `<PlantSvg …>` inside the `v-else-if="pet.status"` block: `<p v-if="pet.status.plant_name" class="text-center font-display text-lg">{{ pet.status.plant_name }}</p>`.
  - `revive.vue` — the alert band's text becomes `⚠️ <span class="normal-case">{{ pet.status.plant_name || 'Cây xanh' }}</span> đang bị héo rũ!` (the band is `uppercase`; the name is not shouted — design §2).
- [ ] **Step 4: Run green:** `npm run lint && npm run typecheck && npm run test:unit` → clean; `plant.test.ts` 6 cases (was 5). Then `npm run build` → succeeds.
- [ ] **Step 5: Commit:** `git commit -am "frontend: the hub, the wilted banners and two bubble lines say the plant's name"`.

### Task 8: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md`

- [ ] **Step 1:**
  - `onboarding` paragraph: the request becomes `{target_goal, notification_time, timezone, plant_name?, answers[…]}`; after "validate (400 `invalid_request`)" add "(`plant_name` optional, trimmed 1..30 **runes**; blank → `DefaultPlantName` "Mầm Non" — decided here, never the DDL's 'My Green Buddy')"; "`Pet.Ensure`" becomes "`Pet.Ensure(userID, name)` (the re-submit path passes `""` and never renames — deliberate, see the inbox re-submit bug)"; the adapter clause becomes "`petForOnboarding` adapts `*pet.Service.EnsureNamed` to `onboarding.Pet`".
  - `pet` paragraph: after the `Service.Ensure` sentence add "`Service.EnsureNamed(user, name)` / `Repo.EnsureNamed` is the same row creation with a name — `INSERT … (user_id, plant_name) ON CONFLICT (user_id) DO UPDATE SET plant_name = EXCLUDED.plant_name WHERE … IS DISTINCT FROM …`, so a repeat is a no-op write and `""` is plain `Ensure`; onboarding is its only caller; `TestIntegrationEnsureNamedWritesOnceAndKeepsTheNameOnRepeat` is gated like its sibling".
  - `shell` paragraph: in the `/onboarding` clause add "an optional plant-name field (`^[\p{L}\p{N} ]{1,30}$` after trim, inline `role="note"`, sent as `plant_name` only when set; design `harness/designs/plant-name.md`)"; in the `/` clause add "(the pet card captions the plant with `plant_name`; the wilted banners and `utils/plant.ts` `speechLine`'s `name` input use it)".
- [ ] **Step 2:** `python3 tools/harness/cli.py validate` → 0. Commit: `git commit -am "harness: CODEMAP — plant_name through onboarding, pet.EnsureNamed, shell"`.

---

## Verification

```bash
cd backend
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./internal/onboarding/ ./internal/pet/ ./cmd/... -count=1 -race -v 2>&1 | grep -E '^(--- FAIL|--- SKIP|ok|FAIL)'
# expect: three `ok` lines, no FAIL; the SKIP lines are exactly the TestIntegration* functions of the two packages (3 in pet, 1 in onboarding)
grep -rhn '^func TestIntegration' --include='*_test.go' . | wc -l
# expect: 13 (was 12)
grep -n 'EnsureNamed' internal/pet/repo.go internal/pet/service.go cmd/api/main.go | wc -l
# expect: >= 5 (interface, SQL const comment/def, PgRepo method, Service method, adapter)
grep -n 'DefaultPlantName\|MaxPlantNameRunes' internal/onboarding/service.go | head -3
# expect: both constants defined
grep -rn 'My Green Buddy' internal/onboarding/*.go
# expect: no output outside _test.go files (the DDL default is never written by onboarding); test fakes may still carry it
make check
# expect: fmt-check silent, vet silent, ok for every package under -race
cd ../frontend
npm run lint && npm run typecheck && npm run test:unit && npm run build
# expect: all clean; onboardingPage.test.ts 8 cases, plant.test.ts 6 cases, revivePage.test.ts 6 cases
grep -c 'plant_name' pages/onboarding.vue pages/index.vue pages/revive.vue composables/useOnboardingApi.ts
# expect: onboarding.vue >= 3 (was 1), index.vue 3 (was 0), revive.vue 1 (was 0), useOnboardingApi.ts 2 (was 1)
grep -n 'name?: string' utils/plant.ts
# expect: 1 line
cd ..
grep -c 'plant_name' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md"
# expect: 6 (was 4)
grep -c 'plant\\_name' project-base/1st-thinking-architecture-doc.md
# expect: 3 (was 2)
grep -n "DEFAULT 'My Green Buddy'" backend/internal/store/migrations/0001_init.up.sql "project-base/Adaptive English Learning Platform - Backend Technical Specification.md" | wc -l
# expect: 2 — the DDL default is untouched in both places
grep -n 'EnsureNamed\|DefaultPlantName\|plant-name' harness/CODEMAP.md | wc -l
# expect: >= 3
git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP
# expect: no output
python3 tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0
gh run list --branch "$(git branch --show-current)" --limit 1
# expect: backend-unit, backend-integration (runs the new TestIntegration* against the CI database), harness-tooling, frontend green
```

Mutation checks (record the results): (a) change `utf8.RuneCountInString` to `len` in `validate()` → `"thirty runes"` in `TestAssessNamesThePlant` must go red (30 × "ă" is 60 bytes); revert. (b) drop the `WHERE … IS DISTINCT FROM` clause from `ensureNamedSQL` → the integration test's `RowsAffected() == 0` assertion must go red when run against a database; revert. (c) change `plantNameTrimmed.value ? { plant_name: … } : {}` to always send `plant_name` → the existing exact-body `toEqual` case must go red; revert.

## Notes and open questions

- **Why the default is not the DDL's.** Spec §3.2 must equal `0001_init` (AGENTS.md), and changing the column default would also rename every pet created by `GET /pet/status` before onboarding. Deciding the default in `onboarding` keeps the migration untouched and the rule in one Go constant.
- **Why length-only on the server.** The character class is a UI rule for the one field that exists today; putting it on the server would make a future rename endpoint (on `/settings`) re-argue it. 30 runes is well inside VARCHAR(100).
- **Pets created before onboarding** (a learner opens `/` before `/onboarding`: the hub's `pet.load()` runs `Ensure`) are renamed by `EnsureNamed` at assessment time — that is the `DO UPDATE` branch and why `DO NOTHING` was not enough.
- **Re-submit stays a no-op** for the name, matching `target_goal`/`timezone`/`notification_time` today; the selected inbox bug `harness/ideas/_inbox/a-re-submitted-assessment-silently-discards-target-goal-noti.md` decides what a re-submit does, and when it lands it should carry `plant_name` with the other three (its plan should call `Pet.Ensure(userID, plantNameOrDefault(...))` on whichever path it chooses to write).
- **`GET /pet/status` example in §6.3** keeps "My Green Buddy": correct for a pet whose owner never onboarded.
- **e2e:** `frontend/tests/e2e/login.spec.ts` stubs `PET` with "My Green Buddy" and asserts nothing about the name — untouched; Playwright is local-only (CODEMAP `frontend` job).
- **Rename UI** is a follow-up on `/settings` (approved 2026-09-24), not this plan.
