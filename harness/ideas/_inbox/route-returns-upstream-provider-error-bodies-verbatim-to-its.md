---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Route returns upstream provider error bodies verbatim to its caller

## Why
`postJSON` puts the first 512 bytes of a non-2xx response body into the error
(`backend/internal/airouter/gemini.go:106-108`), each provider wraps it with its own prefix, and
`Route` joins all three with `errors.Join` (`router.go:72,88,93`). The error a caller receives is
therefore a concatenation of up to three vendors' raw error payloads. Reproduced:

```
airouter: all providers failed
gemini: gemini: status 400: {"error":{"message":"Invalid prompt: the learner goal 'PROMPT-ECHO-SECRETISH' is not supported"}}
openai: openai-compat(m): status 400: {"error":{"message":"bad request"}}
deepseek: openai-compat(m): status 400: {"error":{"message":"bad request"}}
```

Two things are wrong with that as a value handed across a package boundary. First, LLM vendors
routinely echo the offending input back in `error.message` — content-filter and validation errors
quote the prompt — and this package's prompts carry learner-supplied text (`RoadmapUserPrompt`'s
`targetGoal`, and later the placement answers and essays). Second, the string names internal
infrastructure: the `%w`-wrapped transport error from `client.Do` is a `*url.Error` whose message
contains the configured base URL, so a proxy or self-hosted gateway address ends up in it. (The API
keys themselves are safe — verified: they travel in headers, never in the URL, and no error or log
line contains them.)

Nothing in this slice exposes HTTP, so nothing leaks today. But onboarding (order 6) is the next
slice and this is the error it will hold, and the single most common Gin idiom in a young codebase
is `c.JSON(500, gin.H{"error": err.Error()})` — this repo has already filed
`healthz-leaks-postgres-and-redis-driver-error-strings-public.md` for exactly that shape in
`health`. The package should make the safe thing the easy thing before a caller exists, not after.

Related: the same joined string is what makes the error unusable for classification — see
`route-falls-back-on-terminal-4xx-so-one-bad-prompt-buys-thre.md`.

## Expected output
`airouter` separates the operator-facing error from the caller-facing one:

- Provider errors keep the full body for logs (unchanged), but carry it in a typed error whose
  `Error()` is safe: `gemini: status 400` — provider, status, nothing else. The body lives in a
  field (`Body string`) and/or is logged at the provider, not returned.
- `Route` returns `ErrAllProvidersFailed` wrapping the typed per-provider errors, so
  `errors.Is`/`errors.As` still work and `err.Error()` is `airouter: all providers failed:
  gemini: status 400; openai: status 429; deepseek: status 500` — no vendor payload, no URL.
- The package documents on `Route` that its error is safe to log but must still not be returned
  verbatim to an end user, and that callers map `ErrNoProviders` → 503, `ErrRateLimited` → 429,
  everything else → 502/500 with a generic body.
- A test asserts `err.Error()` does **not** contain a marker string planted in the fake's response
  body, and that it *does* still identify provider and status.

## Evidence
- Plan under review: `harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md` (Tasks 1–3).
- `backend/internal/airouter/gemini.go:102-108` — body read (4 MiB cap) then `truncate(body, 512)` into the error.
- `backend/internal/airouter/router.go:72,88,93` — `errors.Join(ErrAllProvidersFailed, per-provider errors…)`.
- Reviewer probe (scratch test, removed): a fake 400 body containing `PROMPT-ECHO-SECRETISH` appears verbatim in `Route`'s returned error; the base URL does not appear on the 400 path but does on the transport-failure path via `*url.Error`.
- Prior art in this repo: `harness/ideas/_inbox/healthz-leaks-postgres-and-redis-driver-error-strings-public.md`.
