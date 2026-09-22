---
type: mvp-slice
status: proposed
source: ideator
run: 2026-09-22-run-02
order: 5
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
