---
type: bug
status: proposed
source: human
run: _inbox
priority: high
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
