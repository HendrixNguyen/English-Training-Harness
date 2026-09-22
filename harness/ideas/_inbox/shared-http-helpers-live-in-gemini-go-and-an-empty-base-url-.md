---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Shared HTTP helpers live in gemini.go and an empty base URL silently means OpenAI

## Why
Two small structural things that a maintainer will trip over in six months.

**Shared helpers are filed under one vendor.** `postJSON`, `truncate` and the `ProviderTimeout`
constant live in `gemini.go` (`:14-15`, `:82-117`) but are used by `openai.go` too — every request
either provider makes goes through `postJSON`, and `NewOpenAICompatibleProvider` reads
`ProviderTimeout`. The plan's own File structure table says `gemini.go` is "`GeminiProvider`" and
`openai.go` is "`OpenAICompatibleProvider`", so the layout the plan documents and the layout on disk
disagree. Anyone changing the body-size cap, the 30-second timeout or the error format for "the
OpenAI provider" has to know to open the Gemini file; anyone deleting or rewriting the Gemini
provider takes both HTTP clients with them.

**An empty base URL silently means OpenAI.** `NewOpenAICompatibleProvider("", key, "", nil)` defaults
to `DefaultOpenAIBaseURL` and `DefaultOpenAIModel` (`openai.go:31-42`) — reasonable for the type's
name, dangerous for its second user. `NewRouter` has to hand-roll DeepSeek's defaults to compensate
(`config.go:63-72`), which is why the two branches are asymmetric: the OpenAI branch trusts the
constructor, the DeepSeek branch repeats its work. Those lines are dead whenever `ConfigFromEnv`
built the `Config` (it already fills every default) and load-bearing only for a hand-built one — a
`Config{DeepSeekAPIKey: "x"}` without them would register a "DeepSeek" provider pointed at
`api.openai.com` and authenticated with a DeepSeek key. That is a confusing failure to debug, and
nothing prevents a future caller from constructing the provider directly.

Neither is a defect in behaviour today — every path through `ConfigFromEnv` → `NewRouter` is correct
and tested (`TestNewRouterWiresDeepSeekOntoTheOpenAIDialect`). Both are the kind of shape that turns
into a defect the first time someone extends the package, which the spec says will happen: §2.1 lists
Gemini **Pro** as well as Flash, and essay grading and exercise generation are still unsliced.

## Expected output
- `postJSON`, `truncate` and `ProviderTimeout` move to a shared file — `http.go` (or `provider.go`) —
  leaving `gemini.go` and `openai.go` each holding exactly the one provider its name promises. The
  plan's File structure table and `harness/CODEMAP.md` name the new file.
- `NewOpenAICompatibleProvider` stops guessing a vendor: either it requires a non-empty `baseURL`
  and `model` (returning an error, or documented as a programming error), or the two vendors get
  thin named constructors — `NewOpenAIProvider(key, baseURL, model, client)` and
  `NewDeepSeekProvider(...)` — that apply their own defaults. `NewRouter`'s two branches then look
  the same and `config.go:64-71` disappears.
- Behaviour is unchanged; the existing tests must pass untouched apart from renames.

## Evidence
- Plan under review: `harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md` — *File structure* table (`gemini.go` → `GeminiProvider`), Tasks 2–4.
- `backend/internal/airouter/gemini.go:14-15,82-117` — `ProviderTimeout`, `postJSON`, `truncate`.
- `backend/internal/airouter/openai.go:39,54` — consumes both from the Gemini file.
- `backend/internal/airouter/openai.go:31-37` — empty `baseURL`/`model` default to OpenAI's.
- `backend/internal/airouter/config.go:60-72` — asymmetric OpenAI/DeepSeek branches compensating for it.
