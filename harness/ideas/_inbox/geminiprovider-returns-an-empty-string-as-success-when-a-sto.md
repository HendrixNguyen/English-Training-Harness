---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# GeminiProvider returns an empty string as success when a STOP candidate has only empty-text parts

## Why
The B3 fix's goal is that an unusable Gemini answer fails as a named provider error (so `Route` falls
back to the next provider) instead of reaching `ParseRoadmap` and surfacing as a decode error / paid
onboarding retry. One shape still slips through: `candidates[0]` with `finishReason: "STOP"` (or absent)
and parts that are all empty text, e.g. `{"candidates":[{"content":{"parts":[{"text":""}]},"finishReason":"STOP"}]}`.
The length check passes (one part), the finishReason check passes, and the joined string `""` is
returned as success. Onboarding then parses `""`, fails, retries the whole router call once, and reports
`ai_bad_output` — without ever trying the fallback providers.

## Expected output
After joining, if the text is empty (after `strings.TrimSpace`), `GenerateContent` returns
`gemini: empty response` so the router falls back. A `gemini_test.go` row covers a STOP candidate with
one empty-text part.

## Evidence
- Plan under review: `harness/plans/2026-09-24-geminiprovider-drops-every-response-part-after-the-first-so-.md` (Tasks 1–2).
- `backend/internal/airouter/gemini.go` on branch `harness/2026-09-24-medium-geminiprovider-drops-every-response-part-after-the-first-so-` (@87ea60f): emptiness is checked on `len(Parts)`, never on the joined text.
- `backend/internal/onboarding/service.go:113-131` (`routeJSON`): a malformed body is retried once and never falls back across providers; router errors do fall back.
- Inferred from reading the code; no test exercises the shape.

## Evaluation
_Evaluator, 2026-09-26 — **deferred** (bug cap of 5 reached; status left `proposed`)._ Gemini unused in production; three-line fix + one test row — fold into the Gemini thinking plan tomorrow.
