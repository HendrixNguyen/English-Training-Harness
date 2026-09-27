---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# GEMINI_THINKING_BUDGET is missing from both env examples and thinkingConfig cannot be omitted

## Why
Two loose ends from the per-attempt-deadline branch:
1. `GEMINI_THINKING_BUDGET` is read by `airouter.ConfigFromEnv` and documented in CLAUDE.md/CODEMAP, but neither `backend/.env.example` nor `deploy/.env.example` lists it. The executor held off because unmerged branches edit those files, and flagged it as a follow-up. An operator reading the env contract will not know the knob exists.
2. `thinkingConfig.thinkingBudget` is sent on **every** Gemini request, and no value means "omit it". The branch's own comment says Pro-class models reject `0`. An operator can work around that with a positive value. A model with no thinking support at all may also reject a `thinkingConfig`, and then no value works except removing the code. **Uncertain:** I could not check this against the live API because there is no Gemini key in this run. Production has no Gemini key today, so the impact is latent.

## Expected output
`#GEMINI_THINKING_BUDGET=0` (with a one-line comment: 0 none, -1 dynamic, N cap) is added to both env examples once the unmerged edits land. Optionally, an empty or `off` value omits `thinkingConfig` from the request, with a `gemini_test.go` row asserting the key is absent.

## Evidence
- `harness/plans/2026-09-27-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s.md`, Execution summary "Follow-ups for the reviewer".
- `backend/internal/airouter/gemini.go` `generationConfig` (`"thinkingConfig": map[string]any{"thinkingBudget": g.thinkingBudget}` unconditional); `config.go` `ConfigFromEnv` (`strconv.Atoi`, else default 0).
- `grep -n GEMINI_THINKING_BUDGET backend/.env.example deploy/.env.example` on branch `harness/2026-09-27-medium-per-task-…` @ b6f33af finds no match.
