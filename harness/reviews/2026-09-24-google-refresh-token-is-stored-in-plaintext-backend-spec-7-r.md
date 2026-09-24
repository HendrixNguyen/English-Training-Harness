---
plan: harness/plans/2026-09-24-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/a-wrong-or-rotated-encryption-secret-key-silently-turns-ever.md, harness/ideas/_inbox/the-sealed-refresh-token-is-not-bound-to-its-users-row-so-a-.md, harness/ideas/_inbox/refresh-token-sealing-tests-miss-the-sealer-error-path-and-a.md, harness/ideas/_inbox/config-go-still-says-jwt-secret-is-not-in-the-1st-thinking-e.md]
---
# Review — Secrets at rest and at boot: AES-256-GCM for `users.google_refresh_token`, and a `JWT_SECRET` that must be 32 bytes

**Plan:** `harness/plans/2026-09-24-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md`
**Branch/worktree:** `harness/2026-09-24-high-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r` / `.worktrees/google-refresh-token-is-stored-in-plaintext-backend-spec-7-r`
**Diff:** `git diff origin/main...harness/2026-09-24-high-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r --stat`

## Plan vs idea

Head idea (`google-refresh-token-is-stored-in-plaintext-backend-spec-7-r`): every Expected-output bullet is delivered. There are two deviations from the idea's wording, both argued in the plan's design decisions and both sound:
- The cipher lives in a new `internal/secrets` package rather than an `auth` helper. `auth` only sees `Sealer` and `google` only sees `Opener`, and neither ever sees the key.
- Sealing happens in `auth.PgUserRepo`, not `Service.SignIn`, so `UserRepo` fakes are unchanged and the empty-token `COALESCE(NULLIF(…))` rule still holds (verified at runtime below).

`ENCRYPTION_SECRET_KEY` is required at boot and documented in `.env.example`. The integration test asserts that the column is a `v1:` value, not the plaintext, and that it opens back to it. CI needs no new env, because each integration test builds its own `Box`.

Folded idea (`jwt-secret-is-accepted-at-any-length…`): delivered. A `JWT_SECRET` shorter than 32 bytes is refused with the exact message the idea proposed, and the 1/31/32/64-byte table covers the boundary. From the rejected JWT-verify finding, the plan takes `WithExpirationRequired` and a case-insensitive `Bearer`, and leaves out `iss`/`aud` binding with a stated reason. That is reasonable for a single-issuer, single-audience HS256 setup.

## Code vs plan

All five tasks were followed with no code deviations. There are five commits, one per task, each with a `Co-Authored-By` trailer. HEAD is `629f6f3`, the same commit as the remote branch. The only deviation is the environmental worktree path, which the orchestrator has since corrected with `git worktree move`.

### Evidence re-run (worktree `backend/`, clean `env -u` shell, diffed against `origin/main` after `git fetch`)

```
gofmt -l ./internal/secrets ./internal/config ./internal/auth ./internal/google ./cmd/api   # no output
  (gofmt -l . also lists internal/quests/{handler_test,repo}.go: untouched here, pre-existing on main, owned by today's gofmt plan)
go build ./... && go vet ./...                                   # clean
go test ./... -count=1                                           # ok ×11 packages (airouter auth config google health notify onboarding pet quests secrets store)
go test ./internal/secrets/ -v                                   # PASS ×7 (names match the plan)
go test ./internal/config/ -v -run 'JWT|Encryption'              # PASS: RejectsAShortJWTSecret 1/31 rejected, 32/64 accepted; RequiresAWellFormedEncryptionKey ×5
go test ./internal/auth/ -v -run 'WithoutExp|CaseInsensitiveBearer'   # PASS ×2
go test ./internal/google/ -v -run OpenStored                    # PASS ×2
Verification greps: repo.go no `refreshToken, defaultTargetGoal`; WithExpirationRequired=1; main.go 3 lines (54/93/153);
  no "plaintext today"; ENCRYPTION_SECRET_KEY .env.example=2 CODEMAP=3; "AES-256-GCM via ENCRYPTION"=1; JWT\_SECRET 1/1; 5 commits; cli validate exit 0
gh run list --branch harness/2026-09-24-high-…  → 35958815050, headSha 629f6f3, completed/success
```

Integration tests (`COMPOSE_PROJECT_NAME=rev-secrets`, Postgres 55444, Redis 56394), `go test ./... -run Integration -p 1`: 13 of 13 `--- PASS`, including `TestIntegrationUpsertCreatesThenPreservesTheLearnerState` and `TestIntegrationSyncStateIsOneRowPerUser`.

Boot refusals (the built binary under `env -i`; exit code 1 confirmed):
```
JWT_SECRET=x                          → config: config: JWT_SECRET must be at least 32 bytes (generate one with: openssl rand -base64 32)
JWT_SECRET=<31 bytes>                 → same
ENCRYPTION_SECRET_KEY unset           → config: config: ENCRYPTION_SECRET_KEY is required — 32 bytes as 64 hex characters (generate one with: openssl rand -hex 32)
ENCRYPTION_SECRET_KEY=abc / 62 hex / base64-32 → config: config: ENCRYPTION_SECRET_KEY must be 32 bytes as 64 hex characters (…)
```
All of these reproduce. The only difference is the doubled `config: config:` prefix, which the execution summary left out (filed as low).

Runtime proof (real binary on :18093 with a random `openssl rand -base64 32` / `-hex 32`; `/healthz` → `200 {"postgres":"ok","redis":"ok","status":"ok"}`, migrations applied). I then ran a throwaway program through the **real** constructors (`config.Load` → `secrets.New` → `auth.NewPgUserRepo` / `google.NewPgRefreshTokenSource`, same key as the server). It lived in an underscore directory and was deleted afterwards:
```
stored column: prefix="v1:" len=65 containsPlaintext=false
RefreshToken(sealed) = "1//rev-plain-token", err=<nil>
re-login with empty token kept sealed value: true
re-seal same plaintext gives new ciphertext: true
RefreshToken(legacy) err=google: no refresh token on file: stored value unusable (secrets: value is not a v1 ciphertext)
POST /api/v1/integrations/google/sync, legacy plaintext row, header "authorization: bearer <jwt>" → {"error":"reauth_required"} HTTP 409
POST … with "Bearer x.y.z"                                                                     → {"error":"unauthorized"} HTTP 401
grep -c 'rev-plain\|legacy-plaintext' api.log → 0   (no plaintext token reaches the log)
```
Cleanup: the server was stopped, `docker compose -p rev-secrets down` removed the containers and network, `git status --short` in the worktree is empty, and the scratch key file was deleted. Note, outside this diff: after its dependencies went away, the binary did not exit on SIGTERM within about 7s and needed `kill -9`. The shutdown plan is not on this branch, so I did not file it.

**The executor's Definition of done holds. There is no gate failure.**

### Mutations (mine, each reverted with `git checkout`, tree clean after)

| Mutation | Killed by |
| --- | --- |
| `Seal`: constant zero nonce | `TestSealingTwiceGivesDifferentCiphertexts` |
| `Open`: skip the prefix check | `TestOpenRefusesAPlaintextRow`, `TestOpenStoredMapsEmptyLegacyAndTamperedToErrNoRefreshToken` |
| `config`: drop the `MinJWTSecretBytes` check | `TestLoadRejectsAShortJWTSecret` |
| `token.go`: remove `WithExpirationRequired()` | `TestVerifyRejectsATokenWithoutExp` |
| `middleware.go`: case-sensitive `"Bearer "` | `TestRequireAcceptsACaseInsensitiveBearerScheme` |
| `google/token.go`: return `stored` without `Open` | both `OpenStored` tests |

I reasoned through the two integration-only mutations (repo passes `refreshToken`; repo seals `""`) from the assertions rather than running them: `HasPrefix(refresh,"v1:")` and `box.Open(refresh)=="rt-1"` after the empty re-login would fail on each. The execution summary does not record the plan's mutation table at all.

## Quality

**Security focus (as the orchestrator asked):**
- *Nonce generation and reuse.* Each `Seal` draws a fresh 96-bit nonce from `crypto/rand`. With one key and about one seal per sign-in, the random-nonce collision bound (about 2^32 messages) is nowhere near reach. A `rand.Read` failure returns an error and never falls back to a zero nonce. `aead.Seal(nonce, nonce, …)` appends to the nonce slice correctly, and there is no aliasing bug.
- *Key parsing.* `ParseHexKey` trims whitespace and needs exactly 32 decoded bytes. `New` re-checks the length, so a key that bypasses `config` still cannot build a weaker cipher. Uppercase hex is accepted, which is fine but untested.
- *Can plaintext be written or logged?* No write path found: `auth.Service.SignIn` → `PgUserRepo.UpsertByGoogleID` is the only writer of the column, and `google.PgRefreshTokenSource` is the only reader (checked with grep over the non-test code). Only `""` passes unsealed, on purpose. Nothing in `auth` or `google` logs anything, and error strings carry sentinel text, never the value. The runtime log contained no token.
- *Versioned prefix.* An exact, case-sensitive `v1:` check comes before any decode. `ErrNotSealed` versus `ErrOpen` is deterministic, and a v2 format can be added without ambiguity.
- *Constant time and error oracles.* GCM's tag check is constant-time. The prefix and length checks are on non-secret structure. Decode, length and authentication failures all collapse to one `ErrOpen`, and every client-visible outcome is the same 409, so there is no padding- or format-oracle surface. The flip side is operability: that same collapse makes a wrong key *silent* (bug 1).
- *No AAD.* Ciphertexts are not bound to their row, so they can be transplanted between accounts by someone with DB write access (bug 2, low).
- *Key rollout.* `.env.example` documents both variables with generate hints, both are required, and it states what rotating or losing the key does. Backend spec §9 now lists `JWT_SECRET (at least 32 bytes)` next to `ENCRYPTION_SECRET_KEY`. What is missing is any detection of a wrong key at boot or at runtime (bug 1).
- *JWT.* The 32-byte floor is on byte length, which matches RFC 7518 §3.2 for raw bytes. `openssl rand -base64 32` gives 44 characters of 256-bit entropy. `WithExpirationRequired` closes the no-exp token. The `Bearer` parsing is safe: it slices after a length check and uses `EqualFold` on 7 bytes, and `TrimSpace` handles extra spaces.

**Boundaries.** `secrets` is imported by `config`, `cmd/api` and the tests only. `auth` and `google` depend on their own one-method interfaces, and no package reads another's table. `config` → `secrets` is a pure leaf dependency, with no cycle.

**Tests.** The suite is honest, and every assertion I mutated turned red. Gaps are the sealer-error path, a short-but-valid payload, a possible nil-deref panic in `google/token_test.go:41`, and the no-exp test not asserting its reason (bug 3, low).

**Docs.** CODEMAP (`secrets` bullet, `auth`, `google`, the CI note) is accurate against the code, so I made no correction. Specs: the §7 "AES-256-GCM" restoration and the §9 and 1st-thinking §8 additions keep the documents' escaping. `config.go:29` still says `JWT_SECRET` is absent from 1st-thinking §8, which this same branch contradicts (bug 4, low).

## Bugs filed

- `harness/ideas/_inbox/a-wrong-or-rotated-encryption-secret-key-silently-turns-ever.md` (**medium**): a wrong or rotated key and a tampered row are both silent 409 `reauth_required`s, with no log line and no boot check.
- `harness/ideas/_inbox/the-sealed-refresh-token-is-not-bound-to-its-users-row-so-a-.md` (low): no AAD, so a `v1:` value can be moved to another account.
- `harness/ideas/_inbox/refresh-token-sealing-tests-miss-the-sealer-error-path-and-a.md` (low): test gaps, and one test that can panic.
- `harness/ideas/_inbox/config-go-still-says-jwt-secret-is-not-in-the-1st-thinking-e.md` (low): a stale config comment, and boot refusals print `config: config:`.

None is a blocker. Nothing here loses data, breaks a workflow, or opens a hole the plan was meant to close. The plaintext-at-rest defect is fixed.

## Verdict

**pass-with-bugs.** The plan and both ideas are delivered. The build, the full and integration suites, CI, the boot refusals and the runtime proof all reproduce, including the sealed round-trip and legacy → 409. Four non-blocking bugs are filed. Merge with `/harness merge harness/plans/2026-09-24-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md` (daily-PR flow, suggested order: shutdown → API-edge → this plan, per the plan's conflict note).
