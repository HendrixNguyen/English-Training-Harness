---
type: mvp-slice
status: planned
source: ideator
run: 2026-09-22-run-02
order: 5
priority: high
plan: harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md
---
# AI Router: multi-LLM providers, task strategies and rate limit

## Why
CEFR placement and the 28-day roadmap — the product's whole "adaptive" claim — are LLM calls, and a single-vendor integration turns any provider outage into a broken onboarding. §6.2 already specifies a task → provider strategy map with fallback across Gemini, OpenAI and DeepSeek; implementing it as one package means onboarding, exercise generation and essay grading all get vendor failover and cost control for free. The `ratelimit:ai:*` limiter is the only thing standing between a hostile client and an unbounded AI bill.

## Expected output
Delivers (Go package `backend/internal/airouter`, no HTTP routes of its own):
- Port of §6.2 to compiling, tested Go: `TaskType` (`placement_test`, `roadmap_generation`, `exercise_generation`, `essay_grading`), `ProviderType` (`gemini`, `openai`, `deepseek`), `LLMProvider` interface `GenerateContent(ctx, systemPrompt, userPrompt) (string, error)`, `Router{providers, strategies}`, `NewRouter()` reading `GEMINI_API_KEY`, `OPENAI_API_KEY`/`OPENAI_BASE_URL`, `DEEPSEEK_API_KEY`/`DEEPSEEK_BASE_URL`; strategies `roadmap_generation→gemini`, `exercise_generation→deepseek`, `placement_test→gemini`, `essay_grading→openai`; `Route()` falls back to any other configured provider when the preferred one is missing or errors; error when none configured.
- `GeminiProvider` against the real Generative Language API URL (the spec's `gemini.api.internal` is a placeholder) and `OpenAICompatibleProvider(baseURL, apiKey, model)` used for both `gpt-4o-mini` and `deepseek-chat`; 30s HTTP timeouts as in §6.2.
- `airouter.RateLimit(ctx, userID)` — `INCR ratelimit:ai:{user_id}` with `EXPIRE` 60s on first hit, max 5/min, returns `ErrRateLimited` for callers to map to HTTP 429.
- The §6.1 system prompt as a Go constant, a `Roadmap` struct (4 modules × 7 daily quests × 3 tasks × ~10 min, task types matching `task_category`), and `ParseRoadmap(string)` that strips nothing — it rejects markdown fences and any output not matching the 4/7/3 shape, per constraint 1.
- Tests: strategy routing and fallback order; each provider against `httptest` servers (success, non-2xx, empty candidates/choices); rate limiter allows 5 then blocks within a minute (miniredis or the store test Redis); roadmap parser rejects wrong module/quest counts and fenced output.
- Tables: none. Redis keys: `ratelimit:ai:{user_id}`. Endpoints: none (consumed by onboarding — unsliced this run — and by future exercise generation). Screens: none.

Depends on: store (1) for the Redis client.

## Evidence
- Spec §6.1 (lines 336–350) system prompt and the 4 modules / 7 quests / 3 tasks constraints.
- Spec §6.2 (lines 352–667) `package airouter`: `TaskType`, `ProviderType`, `LLMProvider`, `Router`, `NewRouter`, strategies map, `GeminiProvider`, `OpenAICompatibleProvider` (treated as pseudocode per `AGENTS.md`).
- Spec §2.1 (line 15) provider suite: Gemini Flash & Pro, GPT-4o-mini, DeepSeek-Chat.
- Spec §4 (line 266) `ratelimit:ai:{user_id}` String(Int), 1 minute, max 5 req/min.
- Spec §8 (line 698) `GEMINI_API_KEY`, `OPENAI_API_KEY`, `DEEPSEEK_API_KEY`.
- `harness/CODEMAP.md` → `airouter`, and `onboarding` ("CEFR grading via airouter").

## Evaluation

**Verdict: select — priority `high`.**

**Is the *Why* real?** Yes. The product's two defining outputs — the CEFR placement and the 28-day roadmap (§5.1 step 5) — are LLM calls, and the onboarding slice (order 6) cannot be planned without a provider abstraction to stub. §6.2 of the 1st-thinking doc already draws the design (task → preferred provider, fallback, two wire formats); §4 supplies the only abuse control on the AI bill (`ratelimit:ai:{user_id}`, 5/min). One package, no routes, consumed by onboarding now and by exercise generation / essay grading later.

**Is the *Expected output* achievable in one plan?** Yes. Two HTTP clients, a router with a strategy table, a rate limiter, two constants (prompt, schema) and one validator. All of it tests against `httptest` servers and a fake Redis-free limiter; the limiter itself gets one `TestIntegration*` on `TEST_REDIS_URL`.

**Dependencies:** `store` (1, merged) for `*store.Redis`, `store.AIRateLimitKey` and `store.AIRateLimitTTL` — all already on `main` (`internal/store/keys.go:14,35`). Nothing else. `NewRouter` never fails the boot: with no provider env var set it returns a router whose `Route` returns `ErrNoProviders`, so a developer without API keys can still run the binary and every non-AI route.

**Decisions taken against the pseudocode (§6.2 is non-binding per AGENTS.md):**
- **Fallback is deterministic and also covers errors.** The pseudocode iterates a Go map (random order) and falls back only when the preferred provider is *missing*. This plan tries the preferred provider, then the others in the fixed order `gemini, openai, deepseek`, on *either* absence or error, and returns the joined errors when all fail — that is the outage resilience the *Why* argues for.
- **Gemini** calls the real Generative Language API (`https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent`, model from `GEMINI_MODEL`, default `gemini-2.5-flash`), key in the `x-goog-api-key` header rather than the query string so it never lands in access logs; `GEMINI_BASE_URL` overrides the host for tests. Body keeps the pseudocode's `system_instruction`, `contents`, `generationConfig{response_mime_type: application/json, temperature: 0.2}`.
- **OpenAI-compatible** provider is shared by OpenAI (`OPENAI_BASE_URL` default `https://api.openai.com/v1`, `OPENAI_MODEL` default `gpt-4o-mini`) and DeepSeek (`DEEPSEEK_BASE_URL` default `https://api.deepseek.com/v1`, `DEEPSEEK_MODEL` default `deepseek-chat`), `response_format {type: json_object}`, temperature 0.2, 30 s client timeout as in §6.2.
- **Rate limiter:** `INCR` + `EXPIRE NX 60s` in one pipeline (Redis 7 — `redis:7-alpine` in compose and CI), so a crash between the two commands can never leave a key without a TTL; count > 5 → `ErrRateLimited`. Callers map it to 429.
- **Roadmap schema.** §6.1 tells the model to match "the requested schema" but never states one, so this slice owns it: `{title, cefr_level, modules[4]{week, title, focus, days[7]{title, tasks[3]{type, title, duration_minutes, content}}}}` with each day's three `type`s being exactly `vocabulary`, `reading`, `practice` (the §3.2 `task_category` values). `ParseRoadmap` rejects markdown fences, trailing tokens, and any count other than 4/7/3, and `Roadmap.Exercises()` flattens to 84 `(day_number, task_type, content_json)` rows whose `content_json` **is the task object** — so the `title` and `duration_minutes` keys the quests plan's `toTask` reads are present by construction.
- **Config** lives in `airouter.Config` / `airouter.ConfigFromEnv(lookup)` rather than `internal/config`, so the package is testable with a plain `func(string) string` and `config.Load` keeps its "required variables" contract untouched (provider keys are optional).

**Priority rationale:** `high` — MVP `order` 5 and a hard dependency of onboarding (6).

**Test strategy:** strategy table and fallback order with fake providers; each HTTP provider against `httptest.NewServer` (request body/headers asserted; success, non-2xx, empty candidates/choices, malformed JSON); `ParseRoadmap` table test (accept; reject fences, 3 modules, 6 days, 2 tasks, duplicate task type, trailing garbage); `ConfigFromEnv` with a map lookup; `TestIntegrationRateLimiterAllowsFiveThenBlocks` on `TEST_REDIS_URL` asserting the sixth call fails and `TTL ≤ 60s`.
