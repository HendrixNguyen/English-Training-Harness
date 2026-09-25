---
type: bug
status: proposed
source: human
run: _inbox
priority: high
---
# Gemini default model gemini-2.5-flash is retired for new accounts, so a fresh deploy fails every placement test with 502

## Why
First production placement test (owner, 2026-09-25): `POST /api/v1/onboarding/assessment` → 502 `ai_upstream_failed` after 1.08 s; the PWA shows "Máy chủ AI đang bận, chưa chấm được bài". The key is valid (`GET /v1beta/models` lists 30+ models), but `airouter.DefaultGeminiModel = "gemini-2.5-flash"` gets HTTP 404 from Google: *"This model models/gemini-2.5-flash is no longer available to new users. Please update your code to use models/gemini-3.8-flash…"*. Every new Gemini account hits this on the happy path (§5.1 onboarding), and the API logs nothing about the upstream status or message, so the operator only sees a bare 502.

Workaround applied on the live service: `GEMINI_MODEL=gemini-3.8-flash` (the provider honours the env override; `backend/.env.example` line 34 documents it). That model answered 503 UNAVAILABLE "high demand" on several probes the same hour, so the router's behaviour on a busy provider matters too.

## Expected output
- `DefaultGeminiModel` moved off the retired name (Google's suggested `gemini-3.8-flash`, or `gemini-flash-latest` if the project prefers an alias that tracks releases); `.env.example`, CLAUDE.md's router paragraph and `deploy/README.md`'s env table say which and why.
- The Gemini and OpenAI-compatible providers log the upstream HTTP status and the first ~200 chars of Google's `error.message` on failure (never the key), so a retired model or a quota error is diagnosable from `railway logs`.
- One bounded retry (or a fallback to the next configured provider) on upstream 503 UNAVAILABLE / 429, per the router's existing fallback design (`router.go:82`), so a transient capacity spike does not fail the learner's placement test outright. Keep the 5 req/min per-user limit.
- Unit tests for the retry/fallback path and for the error logging (httptest server returning 404/503 bodies).

## Evidence
```
$ curl -s -H "x-goog-api-key: …" -d '{"contents":[{"parts":[{"text":"OK"}]}]}' https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent
{"error":{"code":404,"message":"This model models/gemini-2.5-flash is no longer available to new users. Please update your code to use models/gemini-3.8-flash for the latest features and improvements. We recommend you to use the Interactions API…"}}
$ … models/gemini-3.8-flash:generateContent
{"error":{"code":503,"message":"This model is currently experiencing high demand. Spikes in demand are usually temporary. Please try again later.","status":"UNAVAILABLE"}}
$ railway logs --service api --deployment | tail -2
[GIN] 2026/09/25 - 08:23:16 | 502 | 1.08s | POST "/api/v1/onboarding/assessment"      # no provider line before it
```
`backend/internal/airouter/gemini.go:21` (`DefaultGeminiModel`), `:48` (`v1beta/models/{model}:generateContent`); `backend/internal/onboarding/handler.go:49–53` (503 `ai_unavailable`, 502 `ai_bad_output` / `ai_upstream_failed`); `backend/internal/airouter/router.go:82` (fallback log).
