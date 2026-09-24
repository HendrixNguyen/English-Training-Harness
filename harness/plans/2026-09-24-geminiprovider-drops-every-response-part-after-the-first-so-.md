---
idea: harness/ideas/_inbox/geminiprovider-drops-every-response-part-after-the-first-so-.md
status: approved
priority: medium
merged: false
---
# GeminiProvider joins every response part, names a non-STOP finish reason, surfaces a safety block, and asks for enough output tokens — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Bug team — ticket **B3** of 2026-09-24. **Estimate:** 3 h. **Branch:** `harness/2026-09-24-medium-geminiprovider-drops-every-response-part-after-the-first-so-`.

**Idea:** `harness/ideas/_inbox/geminiprovider-drops-every-response-part-after-the-first-so-.md`

**Goal:** A 28-day roadmap that Gemini splits across several `parts` arrives whole; a truncated (`MAX_TOKENS`) or refused (`SAFETY`, `promptFeedback.blockReason`) answer fails with an error that says so instead of a `ParseRoadmap` decode error; and the request asks for an output budget sized for 84 tasks so truncation is prevented, not just detected.

**Architecture:** Everything is inside `GeminiProvider.GenerateContent` (`backend/internal/airouter/gemini.go`): the response struct gains `finishReason` and `promptFeedback`, the parts are joined with a `strings.Builder`, and `generationConfig` gains `maxOutputTokens`. The `LLMProvider` interface, the router and all callers are unchanged — a non-STOP finish is just another `error` that `Route` treats like any provider failure (fallback order, then `ErrAllProvidersFailed`).

**Tech stack:** Go stdlib; `httptest` fakes as in `gemini_test.go`.

**Spec:** 1st-thinking §6.2 (`GeminiProvider`, `response_mime_type: application/json`, temperature 0.2); §6.1 output shape (84 tasks, the largest answer the product requests).

**Root cause (re-read on `main` @ 9517f25):** `gemini.go:64-79` — response struct decodes `parts` as a slice and returns `Parts[0].Text`; no `finishReason`, no `promptFeedback`; `generationConfig` (`gemini.go:48-51`) has no `maxOutputTokens`. Every fake in `gemini_test.go` has exactly one part.

## Global Constraints

- Work in `.worktrees/<slug>`; never touch the main checkout (AGENTS.md).
- `rg`/`timeout` not installed: `grep -n`, `go test -timeout 60s`.
- `TestGeminiSendsTheSpec62RequestAndReturnsTheFirstPart` pins path, header, `response_mime_type`, `temperature` — those must not change; its name is updated in Task 1 because "first part" is no longer the behaviour.
- Error strings stay prefixed `gemini:` (the router joins them into `ErrAllProvidersFailed`; onboarding logs them).
- `gofmt -l internal/airouter` prints nothing.

## Review Focus

1. A candidate with `finishReason: "STOP"` **and** several parts → joined text, no error. Task 1.
2. `finishReason` absent (older API responses omit it) → treated as STOP. Task 2 test "no finishReason".
3. `finishReason: "MAX_TOKENS"` with a non-empty partial text → error naming `MAX_TOKENS`, never the partial text as success. Task 2.
4. `candidates: []` with `promptFeedback.blockReason: "SAFETY"` → error mentions `SAFETY`/`blockReason`, not the generic `empty response`. Task 3.
5. `maxOutputTokens` must be present in the request body the fake receives, as a number; the constant is named and documented. Task 4.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/airouter/gemini.go` | `GeminiMaxOutputTokens` const; response struct with `finishReason`, `promptFeedback`; parts joined; `maxOutputTokens` in `generationConfig` |
| `backend/internal/airouter/gemini_test.go` | Rename first test; add multi-part, finishReason, blockReason and maxOutputTokens cases |
| `harness/CODEMAP.md` | `airouter` bullet: one clause on Gemini response handling |

Run from `backend/` in the worktree.

---

## Tasks

### Task 1: Join every part of the first candidate

**Files:**
- Modify: `backend/internal/airouter/gemini.go:60-79`
- Test: `backend/internal/airouter/gemini_test.go`

- [ ] **Step 1: Write the failing test** (append):

```go
func TestGeminiJoinsEveryPartOfTheFirstCandidate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// The Generative Language API may split one candidate's answer across
		// parts; a roadmap split mid-object is not JSON unless re-joined.
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"a\":1,"},{"text":"\"b\":2}"}]},"finishReason":"STOP"}]}`))
	}))
	defer srv.Close()
	p := NewGeminiProvider("k", srv.URL, "m", srv.Client())
	out, err := p.GenerateContent(context.Background(), "s", "u")
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	if out != `{"a":1,"b":2}` {
		t.Fatalf("out = %q, want the two parts concatenated in order", out)
	}
}
```

Also rename `TestGeminiSendsTheSpec62RequestAndReturnsTheFirstPart` → `TestGeminiSendsTheSpec62RequestAndReturnsTheText` (behaviour unchanged for one part).

- [ ] **Step 2: Run to see it fail** — `go test -timeout 60s ./internal/airouter -run TestGeminiJoinsEveryPart -v` → FAIL `out = "{\"a\":1,"`.

- [ ] **Step 3: Implement** — replace the decode/return block in `GenerateContent`:

```go
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		PromptFeedback struct {
			BlockReason string `json:"blockReason"`
		} `json:"promptFeedback"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("gemini: decoding response: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}
	var sb strings.Builder
	for _, part := range parsed.Candidates[0].Content.Parts {
		sb.WriteString(part.Text)
	}
	return sb.String(), nil
```

(`strings` is already imported.) `FinishReason`/`PromptFeedback` are decoded here and used in Tasks 2–3.

- [ ] **Step 4: Run the package** → PASS. **Step 5: Commit**

```bash
gofmt -l internal/airouter && go vet ./internal/airouter
git add internal/airouter/gemini.go internal/airouter/gemini_test.go
git commit -m "airouter: GeminiProvider concatenates every part of the candidate"
```

### Task 2: A non-STOP `finishReason` is an error that names itself

**Files:**
- Modify: `backend/internal/airouter/gemini.go` (after the empty-response check)
- Test: `backend/internal/airouter/gemini_test.go`

- [ ] **Step 1: Write the failing test** (append):

```go
func TestGeminiNamesANonStopFinishReason(t *testing.T) {
	cases := map[string]struct {
		body    string
		wantErr string // "" = success
	}{
		"no finishReason (older responses)": {`{"candidates":[{"content":{"parts":[{"text":"{}"}]}}]}`, ""},
		"STOP":                              {`{"candidates":[{"content":{"parts":[{"text":"{}"}]},"finishReason":"STOP"}]}`, ""},
		"MAX_TOKENS with partial text":      {`{"candidates":[{"content":{"parts":[{"text":"{\"title\":\"Road"}]},"finishReason":"MAX_TOKENS"}]}`, "MAX_TOKENS"},
		"SAFETY":                            {`{"candidates":[{"content":{"parts":[{"text":""}]},"finishReason":"SAFETY"}]}`, "SAFETY"},
		"RECITATION":                        {`{"candidates":[{"content":{"parts":[{"text":"x"}]},"finishReason":"RECITATION"}]}`, "RECITATION"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(tc.body)) }))
			defer srv.Close()
			p := NewGeminiProvider("k", srv.URL, "m", srv.Client())
			out, err := p.GenerateContent(context.Background(), "s", "u")
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("err = %v, want success", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) || !strings.Contains(err.Error(), "finishReason") {
				t.Fatalf("err = %v (out %q), want an error naming finishReason %s", err, out, tc.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2: Run** → `MAX_TOKENS`, `SAFETY`, `RECITATION` FAIL (they currently succeed with partial text).

- [ ] **Step 3: Implement** — right after the empty-response check:

```go
	// STOP (or absent, on older responses) is the only complete answer. Anything
	// else — MAX_TOKENS, SAFETY, RECITATION, … — would otherwise surface downstream
	// as "ParseRoadmap: unexpected end of JSON input" and burn a paid retry.
	if fr := parsed.Candidates[0].FinishReason; fr != "" && fr != "STOP" {
		return "", fmt.Errorf("gemini: finishReason %s (answer incomplete or refused)", fr)
	}
```

Note the `SAFETY` case above has one empty part, so the emptiness check must come **after**... no: an empty `parts` slice is still "empty response", but a slice with one empty-text part passes the length check and reaches the finishReason check — exactly what the test needs. Keep the order: empty-parts check, then finishReason, then join.

- [ ] **Step 4: Run the package** → PASS. **Step 5: Commit** — `git add internal/airouter/gemini.go internal/airouter/gemini_test.go && git commit -m "airouter: Gemini non-STOP finishReason is a named error, not a truncated success"`.

### Task 3: A prompt block is reported as such

**Files:**
- Modify: `backend/internal/airouter/gemini.go` (the empty-response branch)
- Test: `backend/internal/airouter/gemini_test.go` (add a row to `TestGeminiRejectsNon2xxEmptyCandidatesAndBadJSON`)

- [ ] **Step 1: Add the failing row** to the `cases` map in `TestGeminiRejectsNon2xxEmptyCandidatesAndBadJSON`:

```go
		"prompt blocked": {200, `{"promptFeedback":{"blockReason":"SAFETY","safetyRatings":[]},"candidates":[]}`, "blockReason SAFETY"},
```

- [ ] **Step 2: Run** → FAIL: the error says `empty` and not `blockReason SAFETY`.

- [ ] **Step 3: Implement** — the empty-response branch becomes:

```go
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		if br := parsed.PromptFeedback.BlockReason; br != "" {
			// The prompt itself was refused: the one fact an operator needs.
			return "", fmt.Errorf("gemini: empty response: blockReason %s", br)
		}
		return "", fmt.Errorf("gemini: empty response")
	}
```

(The existing `"empty candidates"` and `"empty parts"` rows still match `"empty"`.)

- [ ] **Step 4: Run the package** → PASS. **Step 5: Commit** — `git commit -am "airouter: Gemini surfaces promptFeedback.blockReason on an empty answer"`.

### Task 4: Ask for enough output tokens

**Files:**
- Modify: `backend/internal/airouter/gemini.go:15-22` (constants), `:48-52` (`generationConfig`)
- Test: `backend/internal/airouter/gemini_test.go` (`TestGeminiSendsTheSpec62RequestAndReturnsTheText`)

- [ ] **Step 1: Extend the request test** — after the `generationConfig` assertion add:

```go
	if gen["maxOutputTokens"] != float64(GeminiMaxOutputTokens) || GeminiMaxOutputTokens < 16384 {
		t.Errorf("maxOutputTokens = %v, want %d (≥ 16384: 84 tasks with content)", gen["maxOutputTokens"], GeminiMaxOutputTokens)
	}
```

- [ ] **Step 2: Run** → FAIL (`maxOutputTokens = <nil>`; `GeminiMaxOutputTokens` undefined — compile error is the failure).

- [ ] **Step 3: Implement** — add to the Gemini defaults block:

```go
	// GeminiMaxOutputTokens sizes the answer for the largest thing we ask for:
	// a 28-day roadmap, 84 tasks each carrying content (word lists, passages,
	// questions). Without it the model's default budget truncates long
	// roadmaps, which ParseRoadmap then rejects as malformed JSON. 2.5-class
	// models accept up to 65536; 32768 leaves headroom for typed content.
	GeminiMaxOutputTokens = 32768
```

and in `generationConfig`: `"maxOutputTokens": GeminiMaxOutputTokens,`.

- [ ] **Step 4: Run the package** → PASS. **Step 5: Commit** — `git add internal/airouter/gemini.go internal/airouter/gemini_test.go && git commit -m "airouter: Gemini requests maxOutputTokens sized for an 84-task roadmap"`.

### Task 5: CODEMAP

- [ ] **Step 1:** In the `airouter` bullet, after `GeminiProvider (real Generative Language API, x-goog-api-key header, response_mime_type: application/json, temperature 0.2)` insert `, maxOutputTokens 32768, joins every part of candidates[0], errors on any finishReason other than STOP and on promptFeedback.blockReason — so a truncated or refused answer is a named provider error, not a ParseRoadmap decode error`.
- [ ] **Step 2: Commit** — `git add harness/CODEMAP.md && git commit -m "codemap: Gemini response handling"`.

## Verification

```bash
cd backend && gofmt -l . ; go vet ./... && go test -timeout 120s ./internal/airouter -count=1 -run 'Gemini' -v 2>&1 | grep -E '^(--- FAIL|ok|FAIL|=== RUN)' | head -40
go test -timeout 300s ./...
```

Expected: no gofmt output; every `TestGemini*` PASS; whole backend PASS. Push; `gh run list --branch <branch>` green.

## Notes and open questions

- `thinkingConfig` is deliberately not set: `gemini-2.5-flash` applies `maxOutputTokens` to the visible answer. If production logs show `finishReason MAX_TOKENS` on roadmaps despite this budget, the fix is `GEMINI_MODEL`/config, or a `thinkingBudget` field added then — not now.
- This ticket makes feature F2 (typed content, larger answers) safer but F2 does not depend on it.
