---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Gemini thinking tokens share maxOutputTokens so the 32768 budget does not prevent roadmap truncation

## Why
The B3 plan sets `generationConfig.maxOutputTokens = 32768` so the 84-task roadmap is "prevented, not
just detected" from truncating, and deliberately leaves `thinkingConfig` unset on the premise that
"`gemini-2.5-flash` applies `maxOutputTokens` to the visible answer". That premise is wrong: on the
2.5 and 3.x Flash models Google counts thinking tokens against `maxOutputTokens`, and Flash's default
thinking budget is dynamic (up to ~24k tokens). A roadmap prompt is exactly the "non-trivial" request on
which the model thinks heavily, so the visible budget can fall to a fraction of 32768.

The branch still fails honestly (the new `finishReason MAX_TOKENS` error), but the idea's third
promise — truncation prevented — is not delivered, and every thinking token is billed and adds latency to
the user's first interaction. The idea itself anticipated this ("and `thinkingConfig` if the configured
2.5-class model spends its output budget on thinking"); the plan declined it on a factual error.

Impact today is limited: production runs OpenRouter only (Gemini is not configured), so this bites
whoever re-enables Gemini.

## Expected output
`generationConfig` carries a `thinkingConfig` with an explicit, named `thinkingBudget` (0 for this
strict-JSON generation task, or a small fixed budget) so the visible answer gets the whole
`GeminiMaxOutputTokens`. The `GeminiMaxOutputTokens` doc comment states that the budget includes
thinking. The request test asserts `thinkingConfig.thinkingBudget` is present. If the configured model
rejects `thinkingBudget: 0` (some Pro models require thinking), the value is configurable or the
comment says which models it applies to.

## Evidence
- Plan under review: `harness/plans/2026-09-24-geminiprovider-drops-every-response-part-after-the-first-so-.md` — *Notes and open questions*, first bullet; Task 4.
- `backend/internal/airouter/gemini.go` on branch `harness/2026-09-24-medium-geminiprovider-drops-every-response-part-after-the-first-so-` (@87ea60f): `generationConfig` has `maxOutputTokens` and no `thinkingConfig`.
- Public reports that Flash thinking tokens consume `maxOutputTokens` and produce empty/`MAX_TOKENS` answers: https://github.com/valentinfrlch/ha-llmvision/issues/609, https://discuss.ai.google.dev/t/max-output-tokens-isnt-respected-when-using-gemini-2-5-flash-model/106708, https://medium.com/@devanshtiwari365/gemini-2-5-flash-was-returning-37-tokens-i-spent-a-day-figuring-out-why-c7c22ac3734f. Not reproduced against the live API (no Gemini key in this environment) — confirm with one real roadmap call reading `usageMetadata.thoughtsTokenCount`.
