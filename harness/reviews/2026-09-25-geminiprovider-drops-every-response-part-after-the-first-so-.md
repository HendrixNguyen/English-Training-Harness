---
plan: harness/plans/2026-09-24-geminiprovider-drops-every-response-part-after-the-first-so-.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/gemini-thinking-tokens-share-maxoutputtokens-so-the-32768-bu.md, harness/ideas/_inbox/geminiprovider-returns-an-empty-string-as-success-when-a-sto.md]
---
# Review — GeminiProvider joins every response part, names a non-STOP finish reason, surfaces a safety block, and asks for enough output tokens

**Plan:** `harness/plans/2026-09-24-geminiprovider-drops-every-response-part-after-the-first-so-.md`
**Branch/worktree:** `harness/2026-09-24-medium-geminiprovider-drops-every-response-part-after-the-first-so-` / `.worktrees/geminiprovider-drops-every-response-part-after-the-first-so-`
**Diff:** `git diff main...harness/2026-09-24-medium-geminiprovider-drops-every-response-part-after-the-first-so- --stat`

## Plan vs idea
Delivers two of the idea's three promises in full: every part of `candidates[0]` is joined in order, `finishReason` other than STOP/absent is a named `gemini:` error, and `promptFeedback.blockReason` is reported on an empty answer — each with a test. The third — "truncation is prevented rather than only detected" — is only half-delivered: `maxOutputTokens: 32768` is set, but the plan declined the idea's conditional `thinkingConfig` on the premise that 2.5 Flash applies `maxOutputTokens` to the visible answer only. Public reports say thinking tokens count against that budget (filed, medium; not reproduced against the live API — no key here). Impact is bounded: production is OpenRouter-only, Gemini is not configured.

## Code vs plan
Reviewed at `origin/<branch>` = `87ea60f`, fresh detached worktree `.worktrees/geminiprovider-drops-every-response-part-after-the-first-so-` (the branch itself is checked out in a nested evaluator worktree, left untouched).

- Task 1 (join parts, rename first test) — followed.
- Task 2 (non-STOP finishReason error; order empty-parts → finishReason → join) — followed.
- Task 3 (`blockReason` in empty-response error + `prompt blocked` row) — followed.
- Task 4 (`GeminiMaxOutputTokens = 32768`, in `generationConfig`, asserted in request test) — followed.
- Task 5 (CODEMAP airouter clause) — followed; accurate.
- Global constraints: `gemini:` prefix kept; `gofmt -l internal/airouter` empty; the spec §6.2 request pins (path, header, mime type, temperature) unchanged.

Re-run evidence (all reproduced the executor's summary):

```
$ gofmt -l .                      # internal/quests/handler_test.go, internal/quests/repo.go — pre-existing, untouched by this diff
$ go vet ./...                    # clean
$ go test -count=1 -run Gemini -v ./internal/airouter
--- PASS: TestGeminiSendsTheSpec62RequestAndReturnsTheText
--- PASS: TestGeminiJoinsEveryPartOfTheFirstCandidate
--- PASS: TestGeminiNamesANonStopFinishReason (5 subtests: MAX_TOKENS, SAFETY, RECITATION, no finishReason, STOP)
--- PASS: TestGeminiRejectsNon2xxEmptyCandidatesAndBadJSON (6 subtests incl. prompt_blocked)
--- PASS: TestGeminiDefaultsToTheRealEndpoint / TestUnknownTaskDefaultsToGemini
$ go test -count=1 -timeout 300s ./...   # all 10 packages ok
$ go build ./cmd/api                     # ok
Runtime: docker compose -p revb3gemini (ports 15491/16391) → API on :18291 → GET /healthz → 200 {"postgres":"ok","redis":"ok","status":"ok"}; compose down, containers and network removed.
```

CI: `gh run list --branch <branch>` → run 36026652849 on head `87ea60f`, `success` (backend-unit, backend-integration, frontend, harness-tooling).

**Integration notes for the daily branch (not defects of this plan):**
1. `git merge-tree origin/main <branch>` conflicts in `harness/CODEMAP.md` only — adjacency, not content: main edited the quests/pet/google bullets, the branch edited only the airouter bullet (main's airouter bullet is unchanged since the merge base). Resolve by taking main's lines and re-applying this branch's airouter clause.
2. It also conflicts with `origin/harness/2026-09-25-high-providertimeout-of-30-s-makes-roadmap-generation-impossible-` in `backend/internal/airouter/gemini.go` (3 hunks). Correct resolution: keep both — `DefaultGeminiModel = "gemini-3.8-flash"` **and** the `GeminiMaxOutputTokens` const (then its "2.5-class models accept up to 65536" comment is stale; reword to the configured model); keep both `UsageMetadata` and `PromptFeedback` in the response struct; call `logCall(...)` **before** the finishReason check so truncated answers still log token usage, then the finishReason check, then the `strings.Builder` join. Run `go test ./internal/airouter` after.
3. On this branch's (stale) base, SIGTERM does not stop the API (`signal.NotifyContext` + `r.Run`); main already fixed this with `server.go`/`ShutdownGrace`. Observed while tearing down the runtime check; nothing to file.

## Quality
- Design/boundaries: change is confined to `GeminiProvider.GenerateContent`; `LLMProvider`, router and callers unchanged. A non-STOP finish becomes an ordinary provider error, so `Route` falls back across providers and onboarding's `routeJSON` returns it as `ai_upstream_failed` without a paid malformed-body retry — the right failure mode.
- Correctness on untried inputs: a STOP candidate whose parts are all empty text still returns `""` as success, reaching `ParseRoadmap` and the retry path the plan set out to avoid (filed, low). `finishReason` values like `OTHER`/`FINISH_REASON_UNSPECIFIED` correctly error. `thought: true` parts would be joined into the answer, but they are only returned when `includeThoughts` is set, which it is not — noted, not filed.
- Performance/cost: 32768 output tokens against a 30 s `ProviderTimeout` cannot complete on this base; the sibling ProviderTimeout branch removes the client timeout in favour of a per-task deadline, so the combination is fine once both land.
- Tests: honest — each new behaviour has a failing-first test with a meaningful assertion (joined string equality; error must name both `finishReason` and the reason). The `GeminiMaxOutputTokens < 16384` clause asserts a constant, harmless. No case combines several parts with `MAX_TOKENS`; the single-part case covers the branch.
- Conventions: idiomatic Go, matches the file's anonymous-struct decode style; comments explain why.
- Security: none affected.

## Bugs filed
- `harness/ideas/_inbox/gemini-thinking-tokens-share-maxoutputtokens-so-the-32768-bu.md` — medium: thinking tokens consume `maxOutputTokens`; add a named `thinkingConfig.thinkingBudget`.
- `harness/ideas/_inbox/geminiprovider-returns-an-empty-string-as-success-when-a-sto.md` — low: STOP with empty joined text is returned as success.

No blockers.

## Verdict
**pass-with-bugs.** Plan executed exactly, verification and runtime proof reproduced, CI green on head. Two non-blocking bugs filed. Needs the CODEMAP (vs main) and gemini.go (vs the ProviderTimeout branch) conflict resolution described above at daily integration.
