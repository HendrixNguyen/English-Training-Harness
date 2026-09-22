---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# GeminiProvider drops every response part after the first so a long roadmap arrives truncated

## Why
`GeminiProvider.GenerateContent` returns `parsed.Candidates[0].Content.Parts[0].Text` and discards
every other part (`backend/internal/airouter/gemini.go:76-79`). The Generative Language API is free
to split one candidate's answer across several `parts`, and does so more often as the answer grows.
The answer this provider exists to fetch is the largest one the product ever asks for: a 28-day
roadmap with 84 tasks, each carrying a free-form `content` object of word lists or passages.

When Gemini splits it, the caller does not get a degraded roadmap — it gets a prefix that is not
valid JSON, so `ParseRoadmap` rejects it with `decoding: unexpected end of JSON input`. Onboarding
then either retries (three more paid provider calls, same odds) or fails the user's very first
interaction with the product. The failure is invisible in tests because every fake returns one part.
Reproduced against an `httptest` fake:

```
PROBE multi-part -> "{\"a\":1," (dropped second part: true)
```

Two smaller gaps sit in the same place. `finishReason` is never read, so a `MAX_TOKENS` truncation is
indistinguishable from a complete answer — again surfacing as a `ParseRoadmap` decode error rather
than a diagnosable one. And `promptFeedback.blockReason` (a safety block) leaves `candidates` empty,
which the code reports as the generic `gemini: empty response`, hiding the only piece of information
that would tell an operator the prompt itself was refused.

## Expected output
`GenerateContent` concatenates the text of every part of `candidates[0].content`, in order, before
returning — one `strings.Builder` over `parsed.Candidates[0].Content.Parts`. A `gemini_test.go` case
serves a two-part candidate and asserts the joined string.

`finishReason` is decoded; anything other than `STOP` (notably `MAX_TOKENS`, `SAFETY`,
`RECITATION`) returns an error naming it, so the caller can tell "the model was cut off" from "the
model returned junk". `promptFeedback.blockReason` is decoded and included when `candidates` is
empty. Both get a test case.

`generationConfig` gains an explicit `maxOutputTokens` sized for the 84-task roadmap (and
`thinkingConfig` if the configured 2.5-class model spends its output budget on thinking), so the
truncation is prevented rather than only detected.

## Evidence
- Plan under review: `harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md` (Task 2).
- `backend/internal/airouter/gemini.go:64-79` — the response struct decodes `parts` as a slice, then indexes `[0]`.
- `backend/internal/airouter/gemini_test.go:13-54,56-82` — every fake response has exactly one part; no case covers two.
- Reviewer probe (scratch test, removed): a fake returning `parts:[{"text":"{\"a\":1,"},{"text":"\"b\":2}"}]` yields `{"a":1,` — the second part is dropped and the result is not valid JSON.
