---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Folded into the plan for geminiprovider-drops-every-response-part-after-the-first-so-.md, which rewrites gemini_test.go and adds the RawQuery assertion there."
---
# Gemini test assertion that the API key is not in the query string can never fail

## Why
Keeping the Gemini API key out of the URL is the one deliberate security deviation this slice makes
from §6.2 (which builds `fmt.Sprintf("%s?key=%s", baseUrl, g.apiKey)`), and the plan's *Decisions*
call it out: "key in the `x-goog-api-key` header rather than the query string so it never lands in
access logs". The test that claims to guard that cannot fail:

```go
// backend/internal/airouter/gemini_test.go:17
gotPath, gotKey, gotCT = r.URL.Path, r.Header.Get("x-goog-api-key"), r.Header.Get("Content-Type")
...
// :38
if strings.Contains(srv.URL+gotPath, "key=") {
    t.Error("API key must travel in a header, not the query string")
}
```

The handler records `r.URL.Path`, which by definition excludes the query string, and `srv.URL` is
`http://127.0.0.1:<port>`. The concatenation can never contain `key=` no matter what the provider
sends. Reintroduce the §6.2 query-string form and this assertion stays green.

The invariant is not unguarded — the plan's verification greps `grep -c 'key=' internal/airouter/gemini.go`
and expects 0, which the reviewer re-ran and confirmed — but a grep in a plan file is not a
regression test: it runs when someone re-reads the plan, not on every `go test`. The only *test*
covering the repo's one security-motivated design decision asserts nothing, which is precisely the
vacuous-pass shape this repo has already filed once
(`integration-gate-tests-only-prove-the-skip-and-would-pass-if.md`).

## Expected output
The handler captures the query, and the test asserts on it:

```go
gotRawQuery = r.URL.RawQuery
...
if gotRawQuery != "" {
    t.Errorf("request carried a query string %q; the API key must travel in the x-goog-api-key header", gotRawQuery)
}
```

plus an explicit `if strings.Contains(gotRawQuery, apiKey)` guard so the intent survives a future
non-secret query parameter. A mutation check goes in the amending plan's verification: switching
`gemini.go` back to `?key=` must make this test fail.

While there: `TestOpenAICompatibleSendsTheSpec62Request…` has no equivalent assertion that the
DeepSeek/OpenAI key appears only in `Authorization` and never in the URL — worth adding the same
`RawQuery` check there.

## Evidence
- Plan under review: `harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md` (Task 2; the decision is in the idea's *Decisions taken against the pseudocode*).
- `backend/internal/airouter/gemini_test.go:17` (captures `r.URL.Path` only) and `:38-40` (the unfalsifiable assertion).
- `backend/internal/airouter/gemini.go:48,59` — the URL has no query; the key goes in `x-goog-api-key`. The behaviour is correct; only its test is not.
- `project-base/1st-thinking-architecture-doc.md` §6.2 — `url := fmt.Sprintf("%s?key=%s", baseUrl, g.apiKey)`, the form this test is meant to prevent coming back.
- Reviewer verification: `grep -c 'key=' internal/airouter/gemini.go` → 0; a boot with fake keys produced no log line containing them (`grep -c 'SECRET-' api.log` → 0). The invariant holds today.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Reject — folded.** The invariant holds (`grep -c 'key=' gemini.go` → 0); only the test is vacuous. The Gemini multi-part plan touches `gemini_test.go` and takes the `RawQuery` assertion with it.
