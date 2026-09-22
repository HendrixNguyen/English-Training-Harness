---
plan: harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/branch-carries-its-own-execution-summary-so-the-ai-router-pl.md, harness/ideas/_inbox/geminiprovider-drops-every-response-part-after-the-first-so-.md, harness/ideas/_inbox/route-falls-back-on-terminal-4xx-so-one-bad-prompt-buys-thre.md, harness/ideas/_inbox/route-returns-upstream-provider-error-bodies-verbatim-to-its.md, harness/ideas/_inbox/parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md, harness/ideas/_inbox/module-week-is-never-validated-and-day-number-comes-from-arr.md, harness/ideas/_inbox/route-has-no-overall-deadline-so-one-call-can-take-90-second.md, harness/ideas/_inbox/redisratelimiter-has-no-documented-behaviour-when-redis-is-d.md, harness/ideas/_inbox/gemini-test-assertion-that-the-api-key-is-not-in-the-query-s.md, harness/ideas/_inbox/shared-http-helpers-live-in-gemini-go-and-an-empty-base-url-.md]
---
# Review — AI Router: multi-LLM providers, task strategies and rate limit

**Plan:** `harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md`
**Branch/worktree:** `harness/2026-09-22-high-ai-router-multi-llm-providers-task-strategies-and-rate-limit` / `.worktrees/ai-router-multi-llm-providers-task-strategies-and-rate-limit`
**Diff:** `git diff main...harness/2026-09-22-high-ai-router-multi-llm-providers-task-strategies-and-rate-limit --stat`

## Plan vs idea

The idea's *Expected output* is delivered item for item.

| Idea asks for | Delivered |
| --- | --- |
| §6.2 ported: `TaskType`, `ProviderType`, `LLMProvider`, `Router`, strategies, fallback, error when none configured | `types.go`, `router.go`, `config.go` — all four task types, all three provider types, the one-method interface with §6.2's exact signature |
| Strategies `roadmap→gemini`, `exercise→deepseek`, `placement→gemini`, `essay→openai` | `router.go:16-23`, pinned by `TestDefaultStrategiesMatchSpec62` against a literal map |
| `GeminiProvider` on the real Generative Language API; `OpenAICompatibleProvider(baseURL, apiKey, model)` for both vendors; 30 s timeouts | `gemini.go`, `openai.go`; `ProviderTimeout = 30 * time.Second` |
| `RateLimit(ctx, userID)` — `INCR` + `EXPIRE` 60 s, max 5/min, `ErrRateLimited` | `ratelimit.go` as `RateLimiter` interface + `RedisRateLimiter.Allow`. **Deviation, justified:** an interface instead of a package-level func, so callers can fake it; the wire behaviour is what the idea asked for |
| §6.1 prompt constant, `Roadmap` struct, strict `ParseRoadmap` that strips nothing | `prompt.go`, `roadmap.go`; fences, preamble and trailing tokens all rejected |
| Tests: routing/fallback, `httptest` per provider, limiter 5-then-block, parser rejections | present; 15 rejection subtests, both providers against real `httptest` servers driving the production clients |

The two deviations the evaluator pre-authorised are both in the code and both improvements over the
pseudocode: the API key travels in `x-goog-api-key` instead of `?key=`, and the fallback order is the
fixed `FallbackOrder` slice rather than Go map iteration. `NewRouter` never failing the boot is
delivered and proved at runtime.

## Code vs plan

All nine tasks landed as described; the executor's "no deviations from the plan's code" holds against
the diff. Ten commits on the branch (nine task commits plus the summary artefact, below).

| Task | Result |
| --- | --- |
| 1 Types and the router | followed — `types.go`, `router.go`, `router_test.go` |
| 2 Gemini provider | followed — `gemini.go` + the shared `postJSON` |
| 3 OpenAI-compatible provider | followed — `openai.go` |
| 4 Env-driven configuration | followed — `config.go`, no `os.Getenv` in package code |
| 5 Rate limiter | followed — `ratelimit.go`, `TxPipeline` INCR + ExpireNX |
| 6 Schema, prompts, strict parser | followed — `prompt.go`, `roadmap.go` |
| 7 `.env.example` | followed — nine commented variables |
| 8 Boot wiring | followed — `cmd/api/main.go`, 9 lines |
| 9 CODEMAP | followed, one word imprecise (see *Quality*) |

### Verification re-run (reviewer, in the worktree)

Scratch stack on `POSTGRES_PORT=5435 REDIS_PORT=6383`, brought up under an explicit Compose project
name (`-p airouter-review`) rather than the default `backend`, because another executor's stack —
project `backend`, ports 5434/6382 — came up on this machine during the run and the default project
name is the directory basename, identical across every worktree. Torn down afterwards, volume
removed, scratch `backend/.env` deleted; the owner's `scio3-redis-1` on 6379 was never touched.

```
$ go build ./... && go vet ./...
BUILD+VET CLEAN

$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1
ok  	.../internal/airouter	0.716s
ok  	.../internal/auth	2.963s
ok  	.../internal/config	0.839s
ok  	.../internal/health	1.454s
ok  	.../internal/store	2.184s

$ go test ./internal/airouter/... -run 'Route|Strategies|Providers' -v   → 11 PASS, ok
$ go test ./internal/airouter/... -run 'Gemini|OpenAI' -v                → 16 PASS, ok
$ go test ./internal/airouter/... -run 'Config|NewRouter' -v             →  4 PASS, ok
$ go test ./internal/airouter/... -run 'Roadmap|Exercises|Prompt' -v     → 22 PASS, ok (15 reject subtests)

$ grep -rn --include='*.go' --exclude='*_test.go' 'https://\|http://' internal/airouter/
internal/airouter/gemini.go:20:	DefaultGeminiBaseURL = "https://generativelanguage.googleapis.com"
internal/airouter/openai.go:13:	DefaultOpenAIBaseURL   = "https://api.openai.com/v1"
internal/airouter/openai.go:15:	DefaultDeepSeekBaseURL = "https://api.deepseek.com/v1"

$ grep -rn --include='*_test.go' 'googleapis.com\|api.openai.com\|api.deepseek.com' internal/airouter/ | grep -v Default
(no hits)

$ grep -n 'x-goog-api-key' internal/airouter/gemini.go ; grep -c 'key=' internal/airouter/gemini.go
gemini.go:59  →  0

$ grep -n '"response_mime_type": "application/json"' gemini.go ; grep -n '"type": "json_object"' openai.go
one hit each

$ grep -c 'Output ONLY valid JSON' internal/airouter/prompt.go   → 1
$ grep -rn 'os\.Getenv' internal/airouter/ --include='*.go' --exclude='*_test.go'
config.go:27 (comment only)
$ grep -c '^func TestIntegration' internal/airouter/ratelimit_test.go  → 1

$ make tidy   → no go.mod/go.sum drift
$ make test   → every package ok
$ make up / make down (as -p airouter-review)  → healthy, then removed cleanly

$ TEST_DATABASE_URL=… TEST_REDIS_URL=… go test ./... -count=1 -v -run Integration -p 1
--- PASS: TestIntegrationRateLimiterAllowsFiveThenBlocks (0.01s)
--- PASS: TestIntegrationUpsertCreatesThenPreservesTheLearnerState (0.12s)
--- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (0.09s)
--- PASS: TestIntegrationConcurrentMigrateDoesNotRace (0.12s)
--- PASS: TestIntegrationPetStatesRejectsASecondRowForTheSameUser (0.07s)
--- PASS: TestIntegrationRedisRoundTrip (0.00s)
(no --- SKIP)

$ python3 tools/harness/cli.py validate; echo exit=$?   → exit=0
$ git status --short (worktree)                          → clean
$ gh run list --branch harness/2026-09-22-high-…         → completed success, run 35753914102
```

**Runtime proof, re-run.** Booted `./cmd/api` against the scratch stack on port 18211 with no AI keys:

```
2026/09/22 23:33:30 migrations applied: [0001_init]
2026/09/22 23:33:30 airouter: no provider API keys set; AI-backed routes will answer 503
2026/09/22 23:33:30 listening on :18211
$ curl -s http://localhost:18211/healthz   → {"postgres":"ok","redis":"ok","status":"ok"} HTTP 200
```

Then again with `GEMINI_API_KEY`/`DEEPSEEK_API_KEY` set to marked fake values:
`airouter: providers [gemini deepseek]`, and `grep -c 'SECRET-' api.log` → **0**. Processes killed,
ports confirmed down.

The executor's step 4 (router → `httptest` fake → `ParseRoadmap`) was reproduced with a throwaway
in-package test, removed immediately (`git status` clean afterwards):

```
PASS e2e: 4 modules, 84 exercises
PASS reject: airouter: invalid roadmap: 3 modules, want 4
```

Everything in the execution summary reproduces. No executor gate failure. One cosmetic discrepancy:
the summary reports `grep -n 'store.AIRateLimitKey\|store.AIRateLimitTTL\|ExpireNX' ratelimit.go` as
"3 hits" — it is 3 matches on 2 lines, because `ExpireNX` and `AIRateLimitTTL` share line 38.

## Quality

**1. Spec fidelity — clean.** `DefaultStrategies()` is `roadmap_generation`+`placement_test`→gemini,
`exercise_generation`→deepseek, `essay_grading`→openai, byte-identical to the 1st-thinking doc
§6.2 (lines 419-427) and the backend spec §5.1 (lines 221-226). `RoadmapSystemPrompt` is §6.1 lines
338-350 verbatim with the rich-text backslash escapes removed; `prompt_test.go` pins all five
constraints. The Gemini body carries `system_instruction`, `contents[{role:user,parts}]` and
`generationConfig{response_mime_type: application/json, temperature: 0.2}` exactly as §6.2 writes it,
and the response is read from `candidates[0].content.parts[0].text`. The OpenAI-compatible body is
`model` + system/user `messages` + `response_format{type: json_object}` + `temperature: 0.2` against
`{baseURL}/chat/completions`. The only intentional wire change is §6.2's `?key=` → the
`x-goog-api-key` header, which is the right call. `§6.2`'s strict `StatusCode != 200` is relaxed to
2xx — harmless.

**2. Fallback semantics — deterministic, but undiscriminating.** Order is fixed (preferred, then
`FallbackOrder` minus preferred) and the log line names both providers. `Route` does **not** touch
the rate limiter — the limiter is a separate interface the caller drives — so a fallback cannot
double-charge a slot; the cost it does multiply is the provider bill. It falls back on *any* error,
including a 400 that will fail identically everywhere: measured, one 400 from Gemini produces three
upstream calls. Bounded (3 per `Route`, and §4 caps a user at 5 `Route`s/min → 15 calls/min/user), so
not a merge blocker, but filed. It also returns each provider's error body joined together, which is
where prompt fragments could travel if a caller does `err.Error()` — filed. Context handling is
correct: `ctx.Err()` is checked before each attempt and `TestRouteStopsFallingBackOnceTheContextIsDone`
pins it. There is no *total* budget, so three 30-second timeouts can stack to 90 s — filed.

**3. Rate limiter — correct.** `INCR` and `EXPIRE NX` go through `TxPipeline` (MULTI/EXEC), so the
"INCR succeeded, EXPIRE failed" window the plan set out to close really is closed: either both apply
or `Exec` returns an error and `Allow` reports it. `EXPIRE NX` also means the window is **not**
extended by the calls that are being rejected — the integration test checks `TTL ∈ (0, 60s]` after
six INCRs, which is the assertion that matters. Window semantics are a fixed 60 s from the first hit,
so up to 10 calls can straddle a boundary; that is what §4 specifies and is fine at this limit. The
Redis-down path returns a non-`ErrRateLimited` error and leaves fail-open vs fail-closed to an
undocumented caller decision — for a cost-bearing call the contract should be written down; filed
(low, with the zero-`Limit` footgun).

**4. Secrets and logging — clean, verified.** The Gemini key is only ever an `x-goog-api-key` header
value and the OpenAI/DeepSeek key only ever `Authorization: Bearer …`; no key appears in any URL
(`grep -c 'key=' gemini.go` → 0), in any log line (verified at runtime with marked fake keys), or in
any error string — provider errors carry status and response body, never the credential. `.env.example`
ships the variables commented and empty. The one thing to watch is the *other* direction: upstream
error bodies do travel outward in `Route`'s error; filed as medium, because onboarding is next.

**5. `ParseRoadmap` — strict where it looks, silent where it does not.** It rejects fences, preamble,
trailing tokens, non-object input, 3 and 5 modules, 6 days, 2 and 4 tasks, duplicate and unknown task
types, empty task titles, out-of-range durations and bad CEFR — 15 subtests, all real. Unknown extra
fields are tolerated by design and documented, and that is the right call for LLM output: a renamed
key still fails, because the count checks catch it. `Exercises()` is deterministic
(`day_number = mi*7+di+1`, 84 rows, 28 of each type) and `content_json` is the whole task object, so
the `title` and `duration_minutes` keys quests' `toTask` reads are present by construction and
asserted per row. Two gaps: the per-task `1..30` range lets a "daily quest" total 90 minutes against
§6.1's 30, and `Module.Week` is decoded but never validated while `day_number` comes from array
position — both filed, both reproduced.

**6. Timeouts, context, body size.** 30 s per provider client; `http.NewRequestWithContext`
everywhere; response bodies capped at 4 MiB via `io.LimitReader` on both the success and error paths,
and error bodies truncated to 512 bytes before entering a message. Silent truncation at 4 MiB
surfaces as a decode error rather than a distinct one — acceptable. Missing piece is the whole-`Route`
budget (filed).

**7. Boundaries, tests, CODEMAP, idiom.** `airouter` imports exactly one internal package — `store`,
for `AIRateLimitKey`/`AIRateLimitTTL`/`*store.Redis` — and nothing else; no routes, no tables, no
migration, no cross-package table access. The `httptest` tests drive the real `GeminiProvider` and
`OpenAICompatibleProvider` (not a mock of them) and assert the actual request shape, which is honest
work; `ConfigFromEnv` is tested through an injected lookup with no environment mutation. One test
assertion is vacuous — `gemini_test.go:38` checks `srv.URL + r.URL.Path` for `key=`, a string that
can never contain a query — and it happens to be the one guarding the slice's security decision;
filed. Test gaps worth naming, all filed: no multi-part Gemini response, no 4xx-vs-5xx distinction,
no dead-Redis case, no day-budget or `week` cases. `CODEMAP.md` is accurate and unusually complete for
one paragraph, with a single imprecision: "rejects … empty titles" should be "empty **task** titles",
since roadmap/module/day titles are unchecked. The reviewer's one-word correction commit on the
branch was refused by the permission system, so it is folded into the `ParseRoadmap` bug instead.
Go idiom is good — sentinel errors with `%w`, `var _ Interface = (*T)(nil)` assertions, constructor
defaults, interfaces defined at the consumer. Two structural nits (shared `postJSON`/`ProviderTimeout`
filed under `gemini.go`; an empty base URL silently meaning OpenAI) are filed as one low bug.

**Merge artefact.** Commit `0966115` puts the `## Execution summary` into the plan file *on the
branch*, against the convention ROOT commit `6d55426` states. It matters for the merge:
`git merge-tree --write-tree --name-only main HEAD` reports **`CONFLICT (content)`** on
`harness/plans/2026-09-22-ai-router-….md` and on nothing else — every Go file merges cleanly. The
branch's copy is the *older* one (`status: approved`, no `branch:`/`worktree:` keys, no `### CI`
section), so resolving it "take theirs" would roll the plan's frontmatter back and desynchronise
`STATE.md`. Filed `priority: high` (not a blocker — the resolution is mechanical) with the exact
instruction for `/harness merge`: keep `main`'s copy of that one file.

## Bugs filed

Ten, none blocking. `python3 tools/harness/cli.py blockers --plan <plan>` → exit 0.

| Priority | Bug |
| --- | --- |
| high | `branch-carries-its-own-execution-summary-so-the-ai-router-pl.md` — commit `0966115` makes the plan file conflict on merge; resolve in favour of `main`'s copy |
| medium | `geminiprovider-drops-every-response-part-after-the-first-so-.md` — only `parts[0]` is read; a split 84-task roadmap arrives as truncated JSON. Also no `finishReason`/`blockReason`/`maxOutputTokens` |
| medium | `route-falls-back-on-terminal-4xx-so-one-bad-prompt-buys-thre.md` — a 400 costs three paid provider calls and masks a bad key |
| medium | `route-returns-upstream-provider-error-bodies-verbatim-to-its.md` — vendor error payloads (which echo prompts) are joined into the returned error |
| medium | `parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md` — per-task `1..30` with no day sum; roadmap/module/day titles unchecked |
| medium | `module-week-is-never-validated-and-day-number-comes-from-arr.md` — reversed or duplicate `week` accepted; `roadmap_json` then disagrees with `exercises.day_number` |
| low | `route-has-no-overall-deadline-so-one-call-can-take-90-second.md` — three 30 s timeouts stack |
| low | `redisratelimiter-has-no-documented-behaviour-when-redis-is-d.md` — fail-open/closed undocumented; zero `Limit` blocks everything |
| low | `gemini-test-assertion-that-the-api-key-is-not-in-the-query-s.md` — the key-in-query assertion cannot fail |
| low | `shared-http-helpers-live-in-gemini-go-and-an-empty-base-url-.md` — `postJSON`/`ProviderTimeout` filed under one vendor; empty base URL means OpenAI |

## Verdict

**`pass-with-bugs`.** The slice delivers its idea and its plan, with two deliberate deviations from
§6.2's pseudocode that both improve on it. Everything in the execution summary reproduces in the
worktree — build, vet, the full unit suite in a stripped environment, every targeted test group,
every verification grep, the integration suite against a live Redis 7 with no skips, the no-keys boot
and `/healthz`, and the end-to-end route-and-validate proof. CI is green on HEAD
(run 35753914102). No blockers: no secret reaches a URL, a log or an error string; the cost of a
failing call is bounded at three attempts under a 5/min cap; and nothing the validator accepts would
write a malformed roadmap — the 84 rows are always well-formed, well-numbered and carry the keys
quests reads.

Mergeable. The one thing the merge must handle is the plan-file conflict from commit `0966115`:
take `main`'s copy of `harness/plans/2026-09-22-ai-router-….md` and everything else from the branch.
