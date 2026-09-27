---
type: bug
status: planned
source: human
run: _inbox
priority: high
plan: harness/plans/2026-09-25-providertimeout-of-30-s-makes-roadmap-generation-impossible-.md
---
# ProviderTimeout of 30 s makes roadmap generation impossible on every non-Gemini provider, so onboarding 502s after the placement test

## Why
Live onboarding on 2026-09-25 (owner): with Gemini refusing on capacity (503) the router correctly fell back to the OpenAI-compatible provider (OpenRouter, `openai/gpt-4o-mini`), graded the placement test in ~2 s, then `POST /api/v1/onboarding/assessment` answered **502 after 33.87 s** — `airouter.ProviderTimeout = 30 * time.Second` (`gemini.go:15`, used by both drivers) cut the roadmap call off mid-generation, so `ErrAllProvidersFailed` → `ai_upstream_failed`. Measured with the same JSON shape (4 modules × 7 days × 3 tasks, §6.1): `openai/gpt-4o-mini` 53 s / 4 712 completion tokens, `deepseek/deepseek-chat` 78 s / 4 494 tokens. No mainstream model streams ~4.5 k tokens in 30 s except Gemini Flash on a good day, so the §6.2 timeout is wrong for `TaskRoadmapGen` and every learner whose Gemini call fails is stuck at the end of the quiz — the placement result is discarded too, so they re-take the quiz on retry.

## Expected output
- Per-task timeouts instead of one constant: placement/exercise/grading ≈ 30 s, roadmap generation ≥ 120 s (context deadline passed through `routeJSON`; the Gin request has no server-side write timeout that would cut it shorter — verify `http.Server` settings from the 2026-09-24 shutdown plan). Constants documented in CLAUDE.md's router paragraph and the backend spec §6.2 addendum.
- Onboarding does not lose the graded level when the roadmap step fails: persist `cefr_current` after grading (or return `202` with `assessed_level` and let the client request roadmap generation separately) so a retry skips the quiz. Pick the smaller change that keeps the §6.2 DTO stable; say which in the plan.
- Providers log elapsed time and token usage per call (`airouter: gemini roadmap_generation 18.2s 4310 tokens`) so the operator can see this from `railway logs`.
- Frontend: the assessment screen's spinner copy tells the learner it can take a minute or two; the 30-second client fetch timeout, if any, is raised accordingly.
- Tests: a fake provider that sleeps past 30 s but under the roadmap deadline succeeds for `TaskRoadmapGen` and fails for `TaskPlacementTest`.

## Evidence
```
2026/09/25 09:14:12 airouter: fallback from gemini to openai for task placement_test
2026/09/25 09:14:14 airouter: fallback from gemini to openai for task roadmap_generation
[GIN] 2026/09/25 - 09:14:44 | 502 | 33.87s | POST "/api/v1/onboarding/assessment"
$ # same prompt shape via OpenRouter, response_format json_object, temperature 0.2
openai/gpt-4o-mini:     http=200 time=53s output_chars=23980 completion_tokens=4712 finish=stop
deepseek/deepseek-chat: http=200 time=78s output_chars=22819 completion_tokens=4494 finish=stop
```
`backend/internal/airouter/gemini.go:14-15,42`, `openai.go:39` (`http.Client{Timeout: ProviderTimeout}`), `backend/internal/onboarding/service.go:73-108` (placement then roadmap in one request, level not persisted between them). Related inbox bug: "Gemini default model gemini-2.5-flash is retired…" (retry/fallback on 503, error logging) — plan together if the evaluator prefers one airouter change.

## Evaluation
**Verdict: select, `priority: high`.** The *Why* is real and measured: every learner whose Gemini call fails (today: every fresh account, because the default model is retired — see the folded bug) reaches the end of the placement quiz and gets a 502 after ~34 s, and the graded level is thrown away with it. That is the §5.1 happy path for a first-time user; nothing outranks it in the inbox.

**Root cause (read-only on `origin/main`; this checkout's `backend/` is identical for `airouter` and `onboarding`):**
- `backend/internal/airouter/gemini.go:15` `ProviderTimeout = 30 * time.Second` is the `http.Client{Timeout}` for both drivers (`gemini.go:42`, `openai.go:39`). `http.Client.Timeout` is an absolute per-request budget that includes reading the body, so a 53–78 s JSON answer can never complete; `client.Do` fails with `Client.Timeout exceeded`, `postJSON` wraps it, `Router.Route` falls through to the next provider (each capped at 30 s in turn) and returns `ErrAllProvidersFailed` → `onboarding/handler.go:52` → 502 `ai_upstream_failed`. The measured 33.87 s is Gemini's fast 503 + one 30 s OpenAI timeout (DeepSeek unset).
- `backend/internal/onboarding/service.go:72-88` grades and generates in one call and keeps `level` in a local; the roadmap error returns before any write, so the grade is gone. `quiz.go` only stages the answers (`HSET quiz:placement:{user_id}`, 2 h TTL).
- `backend/cmd/api/server.go:31`: `newServer` sets no `WriteTimeout` (the 2026-09-23 shutdown plan chose that deliberately) and the Gin request context carries no deadline, so nothing server-side cuts a longer AI call short — confirmed; only the doc comment on `server.go:28` needs re-wording once `ProviderTimeout` goes.
- A consequence the fix must handle: when a context deadline fires, `Route` returns `ctx.Err()` (`router.go:78,91`), which `handler.go` maps to **500 `internal_error`**. A per-task deadline needs its own mapping (504 `ai_timeout`).

**Decision on keeping the grade — the Redis quiz hash, not `users.cefr_current`, not a 202.** The graded level is written as one extra field (`_level`) of the existing `quiz:placement:{user_id}` hash (same 2 h TTL, deleted on success as today) and reused by a re-submit with identical answers, which skips the placement call. It keeps the §6.1 DTO byte-for-byte, adds no endpoint, and preserves the invariant `TestAssessFailsWithoutWritingWhenTheRoadmapIsBadTwice` pins ("the users update must be in the same tx as the roadmap and never run alone"); a durable `cefr_current` write before a roadmap exists would break that invariant and outlive any retry. The PWA already keeps the answers on an `ai_*` error (`pages/onboarding.vue`: `// answers are kept`), so the retry is one tap on "Hoàn thành" — a page reload still loses the client-side answers; persisting them in the client is out of scope.

**Deadlines:** `airouter.TaskTimeout(task)` — 180 s for `roadmap_generation` (slowest measured provider 78 s, room for one fast-failing provider plus one slow success), 30 s for everything else — set by `onboarding.routeJSON` per `Route` call, and applied by `Route` itself when a caller passes a context without a deadline (future callers cannot regress to "no bound"). The `http.Client.Timeout` is removed so the context governs. Worst case for one assessment is 2 × 30 s + 2 × 180 s (the malformed-body retry) — accepted; unverified whether Railway's edge enforces a shorter proxy timeout (Railway documents none), noted in the plan.

**Folded in:** `gemini-default-model-gemini-2-5-flash-is-retired-for-new-acc.md` — same file (`gemini.go`/`postJSON`), same log lines, same tests; together they stay under half a day. **Dependencies:** none. **Conflict note:** `harness/2026-09-24-medium-geminiprovider-drops-every-response-part-after-the-first-so-` (done, unmerged) also edits `gemini.go`'s response struct; the plan tells the executor how to land on either base.
