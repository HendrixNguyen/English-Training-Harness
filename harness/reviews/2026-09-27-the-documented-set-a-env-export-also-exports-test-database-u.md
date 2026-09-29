---
plan: harness/plans/2026-09-25-the-documented-set-a-env-export-also-exports-test-database-u.md
verdict: pass
bugs: []
---
# Review — Dev loop and CI mirror: `make run` sources `.env` in its own shell, `make check` refuses service variables, `fmt-check` fails on a parse error, integration tests run `-race`

**Plan:** `harness/plans/2026-09-25-the-documented-set-a-env-export-also-exports-test-database-u.md`
**Branch/worktree:** `harness/2026-09-25-medium-the-documented-set-a-env-export-also-exports-test-database-u` / `.worktrees/the-documented-set-a-env-export-also-exports-test-database-u`
**Diff:** `git diff main...harness/2026-09-25-medium-the-documented-set-a-env-export-also-exports-test-database-u --stat`

## Plan vs idea
Delivered for all three ideas folded into this plan. Head idea (documented `set -a` loop exports `TEST_DATABASE_URL` into the dev shell): `.env.example` now ships `TEST_*` commented out, and `make run` sources `.env` in its own recipe shell, so no documented command exports anything into the interactive shell. `make check` now mirrors `backend-unit`'s service-variable guard. Integration tests run under `-race` in CI and in `make test-integration`. The gofmt parse-error hole is closed too.

## Code vs plan
Reviewed at the branch head `fb9f78c` (4 commits, base `f942e64`, 138 behind `origin/main`), then again with `origin/main` merged in a scratch worktree (never pushed).

- Task 1 (`make run` owns the export; `TEST_*` commented out): followed.
- Task 2 (`no-service-vars`, `build`, `fmt-check` exit status, `check` order): followed.
- Task 3 (`-race` on `backend-integration` + `make test-integration`): followed.
- Task 4 (CODEMAP): followed.

Branch head:
```
$ grep -n '^TEST_' .env.example; echo grep-exit=$?
grep-exit=1
$ sh -c 'set -a; . ./.env.example; set +a; env | grep -c "^TEST_"'
0
$ make -n run | tail -1
set -a; . ./.env; set +a; exec go run ./cmd/api
$ grep -n 'set -a' Makefile .env.example
Makefile:14:	set -a; . ./.env; set +a; exec go run ./cmd/api
$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL make check
go build ./... / go vet ./... / go test ./... -count=1 -race → 13 packages ok; exit=0
$ TEST_REDIS_URL=x make check
TEST_REDIS_URL is set: make check mirrors CI's backend-unit, ... ; make: *** [no-service-vars] Error 1; exit=2
$ printf 'package probe\n\nfunc F( {}\n' > internal/zz_probe.go; make fmt-check
internal/zz_probe.go:3:9: expected ')', found '{'
gofmt -l failed (see above); exit=2   (probe removed, tree clean)
$ grep -n '\-race' Makefile .github/workflows/ci.yml   → Makefile:44, :70; ci.yml:54, :109 (4 commands)
$ mv .env away; make run
.env missing: cp .env.example .env, then edit it; exit=2
$ gh run list --branch <branch> --limit 1
completed success ... CI push 36094885712 2m36s
```
With `origin/main` merged (CODEMAP conflict resolved to main's side, for testing only). Stack: `COMPOSE_PROJECT_NAME=rv-deploy`, pg 5447, redis 6397; torn down with `down -v` afterwards.
```
$ env -u ... make check → exit=0 (all packages ok under -race)
$ TEST_DATABASE_URL=...:5447/english TEST_REDIS_URL=...:6397/0 make test-integration
go test ./... -count=1 -v -run Integration -p 1 -race   → exit 0
want=13 pass=13 skip=0 "DATA RACE"=0
```
So the `-race` switch holds on today's main, including `onboarding/quiz_integration_test.go`, which landed after this branch was cut.

## Quality
- The Makefile idiom is sound. The `$$(gofmt -l .) || …` assignment status is POSIX. `printenv` is portable. `exec` keeps signal delivery unchanged. The guard runs as the first prerequisite, so it fires before any test binary.
- Test honesty: there are no new unit tests, which is right for Makefile/CI plumbing. The proofs are the probe runs above.
- **Merge note for the orchestrator:** `harness/CODEMAP.md` conflicts with `origin/main` because main has since rewritten the `cmd/api` paragraph (onboarding's 180 s AI deadlines) and the CI section. When this goes into the daily branch, keep main's paragraphs and re-apply this branch's two sentences: `make run` sources `.env`, and `-race` plus `no-service-vars` in `make check`. `ci.yml` and `.env.example` auto-merge cleanly.
- CODEMAP accuracy (branch side): correct.

## Bugs filed
None.

## Verdict
`pass`. It may go into today's daily PR; the CODEMAP conflict needs resolving there.
