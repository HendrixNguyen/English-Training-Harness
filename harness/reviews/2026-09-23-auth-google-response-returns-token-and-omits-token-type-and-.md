---
plan: harness/plans/2026-09-23-auth-google-response-returns-token-and-omits-token-type-and-.md
verdict: pass
bugs: []
---
# Review — auth/google response: answer the backend spec §6.1 sign-in body

**Plan:** `harness/plans/2026-09-23-auth-google-response-returns-token-and-omits-token-type-and-.md`
**Branch/worktree:** `harness/2026-09-23-high-auth-google-response-returns-token-and-omits-token-type-and-` / `.worktrees/auth-google-response-returns-token-and-omits-token-type-and-`
**Diff:** `git diff main...harness/2026-09-23-high-auth-google-response-returns-token-and-omits-token-type-and- --stat`

## Plan vs idea

Delivered, point for point. The idea's *Expected output* asked for four things:

- `auth.Handler` writes exactly the §6.1 shape — `backend/internal/auth/handler.go:20-32,51-61`: a typed
  `signInResponse`/`signInUser` pair with tags `access_token`, `token_type`, `expires_in`, `user{id, email,
  full_name, cefr_current}`. Matches the spec literal at
  `project-base/Adaptive English Learning Platform - Backend Technical Specification.md:251` key for key.
- No `token` alias — `grep -n '"token"' internal/auth/handler.go` is empty; the only `"token"` left in the
  package is the test's negative assertion.
- `handler_test.go` asserts the wire shape — see *Quality* below; verified independently, not taken on trust.
- CODEMAP names the fields — `harness/CODEMAP.md:10`.

## Code vs plan

Task 1 (handler + test) and Task 2 (CODEMAP): **followed**, no deviations. `git diff main...<branch>
--name-only` is exactly `backend/internal/auth/handler.go`, `backend/internal/auth/handler_test.go`,
`harness/CODEMAP.md` — nothing else. Worktree is clean and `HEAD == @{u}` (`6566205`).

Plan *Verification* re-run by the reviewer in the worktree (`backend/`):

```
$ gofmt -l ./internal/auth                       # (no output)
$ go build ./... && go vet ./...                 # VET_OK, exit 0
$ go test ./... -count=1 -timeout 300s
ok  .../internal/airouter   0.685s
ok  .../internal/auth       1.579s
ok  .../internal/config     0.978s
ok  .../internal/health     2.064s
ok  .../internal/pet        2.633s
ok  .../internal/quests     3.323s
ok  .../internal/store      3.834s          # 0 FAIL
```

CI on the pushed branch: `gh run list --branch harness/2026-09-23-high-auth-…` → one run,
`completed success` (`35813839556`, 49s). Green.

No PR exists (`gh pr create` 403s — the `gh` CLI is authenticated as a work account that is not a
collaborator on this repo). Step 8 of the review skill is skipped for that reason; not a finding.

## Quality

**`expires_in` is honest.** `int(TokenTTL / time.Second)` (`handler.go:54`), and `TokenTTL` is the single
value that governs both sides of the session: the JWT `exp` (`token.go:36` — `now.Add(TokenTTL)`) and the
Redis `sess:{user_id}:token` TTL (`service.go:50` — `sessions.Put(..., TokenTTL)`). `token.go:14` defines it
as `store.SessionTTL` (= 24h, `store/keys.go:12`) and `token_test.go:26-29` pins that equality, so the
advertised number cannot drift from the real session lifetime without a failing test. Observed in the test's
own JWT payload: `iat 1800000000`, `exp 1800086400` — exactly the advertised 86400.

**Test honesty — checked independently, not accepted from the summary.** The assertion is real: the body is
decoded into `map[string]json.RawMessage` first (`handler_test.go:48-63`), so it is about wire keys, not Go
struct tags, and `len(keys) != 4` is a genuine exactness check. I verified it catches an *added* field, which
a missing-key loop alone would not: with a transient fifth field (`Extra string \`json:"extra"\``) added to
`signInResponse`, the test failed with

```
handler_test.go:62: body has 5 top-level keys, want 4: {"access_token":"eyJ…","token_type":"Bearer",
  "expires_in":86400,"user":{…},"extra":""}
```

and I also confirmed the executor's reverted-handler claim by restoring `handler.go` from `f37175a`, which
failed with all six documented diagnostics (`body has a "token" key`, three `body is missing …`, `body has 2
top-level keys, want 4`, `access_token does not verify … invalid number of segments`). Both edits were
reverted immediately; `git status --short` is clean and the suite is green again.

**Conventions and boundaries.** The typed response struct matches the later `pet` handler's pattern; nothing
outside `handler.go`/`handler_test.go` moved, so `Service`, `TokenIssuer`, `auth.Require` and the session
store are untouched and no package boundary is crossed. `gofmt`/`go vet` clean. The doc comment on
`signInResponse` states the derivation rule, which is the thing a maintainer would otherwise re-derive.

**CODEMAP accuracy.** `harness/CODEMAP.md:10` now reads "answers the backend spec §6.1 body
`{access_token, token_type: "Bearer", expires_in: 86400, user: {id, email, full_name, cefr_current}}` —
`expires_in` is `int(TokenTTL / time.Second)`, and there is no `token` key (contract fix 2026-09-23)". That
matches the code exactly, including the derivation. No correction needed.

**Commit attribution.** Both commits (`67b1597`, `6566205`) end with
`Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`. Correct; no finding.

## Bugs filed

None.

## Verdict

**pass.** Nothing blocks the merge. The §6.1 shape is exactly right — four top-level keys, no `token` alias,
`token_type: "Bearer"`, `expires_in` derived from `TokenTTL` and equal to the real JWT/Redis lifetime — and
the test pins the wire keys in a way that fails on both a missing and an added field. MVP slice 9
(frontend-shell) is unblocked once this lands on `main`.

**Follow-up for the orchestrator (not a finding against this branch):** once this merges,
`harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md` becomes stale in
two places — its *Merge blocker* header paragraph and the matrix row describing auth as "live on `main`,
wrong shape". Both should read "live on `main`" before slice 9 is reviewed.
