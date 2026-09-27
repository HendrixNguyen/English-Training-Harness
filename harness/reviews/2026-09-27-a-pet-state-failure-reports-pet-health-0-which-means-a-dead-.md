---
plan: harness/plans/2026-09-25-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md
verdict: pass
bugs: []
---
# Review — quests: a failed pet read omits `pet_health`/`streak_count` instead of reporting a dead plant

**Plan:** `harness/plans/2026-09-25-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md`
**Branch/worktree:** `harness/2026-09-25-medium-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-` / `.worktrees/a-pet-state-failure-reports-pet-health-0-which-means-a-dead-`
**Diff:** `git diff main...harness/2026-09-25-medium-a-pet-state-failure-reports-pet-health-0-which-means-a-dead- --stat`

## Plan vs idea
Delivered. The idea's Expected output (first option): a `Pet.State` failure never fabricates a plant state — both fields are `*int,omitempty`, nil on error, so the §6.2 body omits them and the 200 stays. The client (`usePetStore.applyProgress`) now assigns only present numbers, so the plant hub keeps the last state from `GET /pet/status`. The `Pet` interface doc now says a missing `pet_states` row is not an error, as the idea asked. `NopPet`'s (100, 0) defaults unchanged.

## Code vs plan
Reviewed at origin head `200d54b` in a detached scratch worktree (diff base `eefe92f`, 9 files, +93/-33).
- Task 1 (backend): followed exactly — `service.go` pointer fields + nil on error (`PetState{}` substitution gone), `pet.go` doc, `petOf` helper at the three existing assertions, rewritten `TestAPetStateFailureDoesNotFailTheRequest`, new `TestProgressHandlerOmitsThePetFieldsWhenTheReadFails` (asserts on the JSON body, not the struct — honest).
- Task 2 (frontend): followed — optional `ProgressResponse` fields, `typeof === 'number'` guards, both new store tests; the "real 0" test pins that a genuine wilt is not dropped by a falsy check (executor mutation-checked this).
- Task 3 (CODEMAP): followed; the quests sentence is accurate.
- Deviations (both justified, both necessary): `integration_test.go` adapted to `petOf` (it would not compile otherwise); `petStore.test.ts` `applyProgress` cases load a copy of the shared fixture — fixes a real order-dependence the old test had.

Re-run evidence:
```
$ gh run list --branch <branch> --limit 1
completed success harness: CODEMAP — omitted pet fields on a failed read  CI ... 36093609127
$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL make check
go vet ./...
go test ./... -count=1 -race
ok  .../cmd/api ... ok .../internal/quests ... ok .../internal/store   (all 13 packages ok)
$ go test ./internal/quests/ -count=1 -race -v | grep ...
--- PASS: TestProgressHandlerOmitsThePetFieldsWhenTheReadFails
--- PASS: TestAPetStateFailureDoesNotFailTheRequest
ok  .../internal/quests 7.258s
$ grep -n omitempty internal/quests/service.go
24:	PetHealth   *int `json:"pet_health,omitempty"`
25:	StreakCount *int `json:"streak_count,omitempty"`
$ grep -rn 'PetState{}' internal/quests/service.go     -> (no output)
$ go vet -tags integration ./internal/quests/          -> ok
$ npm ci && npm run lint && npm run typecheck && npm run test:unit
eslint . (clean) / nuxi typecheck (clean)
 ✓ tests/unit/petStore.test.ts (8 tests)
 Test Files  16 passed (16)   Tests  80 passed (80)
```
The integration suite was not re-run locally; CI's `backend-integration` job on the head commit is green and the only integration change is the `petOf` compile fix.

## Quality
- Boundaries: change stays inside `quests` + the pet/quest stores; no cross-package reads. `grep` confirms nothing outside `internal/quests` reads `PetHealth`/`StreakCount`; the only frontend consumer is `pages/learn/[id].vue` passing the whole response to `applyProgress`.
- Pointer capture: `&pet.Health` escapes an `if`-scoped variable — safe in Go, allocation is per-request and trivial.
- Documentation: backend spec §6.2 still shows the fields as always present; the plan deliberately records the error-path omission in CODEMAP only (the spec does not define the error path). Acceptable; noted, not filed.
- Merge note: `git merge-tree origin/main <branch>` reports a content conflict in `harness/CODEMAP.md` only (the branch is based on an older main). Code files merge cleanly. The daily integration merge must resolve the CODEMAP paragraph by hand.

## Bugs filed
None.

## Verdict
pass — plan and idea delivered, suite and verification reproduce, CI green. Only `harness/CODEMAP.md` conflicts with current `main` at integration time.
