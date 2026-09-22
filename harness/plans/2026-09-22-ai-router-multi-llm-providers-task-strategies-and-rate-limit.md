---
idea: harness/ideas/2026-09-22-run-02/ai-router-multi-llm-providers-task-strategies-and-rate-limit.md
status: draft
priority: high
merged: false
order: 5
---
# AI Router: multi-LLM providers, task strategies and rate limit — Plan

**Idea:** `harness/ideas/2026-09-22-run-02/ai-router-multi-llm-providers-task-strategies-and-rate-limit.md`
**Goal:** Add `backend/internal/airouter` — a compiled, tested port of the 1st-thinking doc's §6.2 (`TaskType`, `ProviderType`, `LLMProvider`, `Router` with task→provider strategies and deterministic fallback, Gemini and OpenAI-compatible providers over injectable base URLs), the §4 `ratelimit:ai:{user_id}` limiter (5/min), the §6.1 system prompt as a constant, and a strict validator for the 4×7×3 roadmap JSON that onboarding (6) will persist. No HTTP routes.

**Spec precedence:** §6.2 of the 1st-thinking doc is pseudocode (AGENTS.md → *Reading the spec*): its names and wire formats are kept, its map-iteration fallback and query-string API key are not. §4 fixes the limiter key, TTL and limit; §6.1 fixes the prompt and the 4/7/3 constraints; backend spec §9 lists the env vars (`GEMINI_API_KEY`, `OPENAI_API_KEY`, `DEEPSEEK_API_KEY`). Neither spec states the roadmap JSON schema §6.1 refers to — this plan defines it (Task 6) and onboarding reuses it.

**Architecture:** `LLMProvider` is one method, `GenerateContent(ctx, systemPrompt, userPrompt) (string, error)`. Two implementations (`GeminiProvider`, `OpenAICompatibleProvider`) share an `http.Client` with a 30 s timeout and take their base URL as a constructor argument so tests point them at `httptest` servers. `Router` holds `providers` and `strategies`; `Route` tries the strategy's provider, then the others in the fixed `FallbackOrder`, on absence *or* error. `Config`/`ConfigFromEnv(lookup)` decide which providers exist — a provider registers only when its API key is set. `RedisRateLimiter.Allow` does `INCR` + `EXPIRE NX` in one pipeline. `ParseRoadmap` decodes strictly (no fences, no trailing tokens, exact counts) and `Roadmap.Exercises()` flattens to the 84 rows onboarding inserts, whose `content_json` **is** the task object — so the `title` / `duration_minutes` keys quests' `toTask` reads are always present.

**Tech stack:** Go 1.25 (`backend/go.mod`), `net/http`, `encoding/json`, `go-redis/v9`. No new dependencies. No tables, no migration. Redis 7 is required for `EXPIRE … NX` (`redis:7-alpine` in `docker-compose.yml` and `ci.yml`).

**Depends on:** `store` (1), merged: `*store.Redis{Client}`, `store.AIRateLimitKey`, `store.AIRateLimitTTL` (`internal/store/keys.go:14,35`). Independent of quests/pet; can be executed in parallel with them. `cmd/api/main.go` gets a two-line wiring (Task 8) that touches nothing another slice edits.

**Run every command from `backend/`** unless the step says otherwise. `rg` is not installed — use `grep -n`.

## File structure

| Path | Responsibility |
| --- | --- |
| `backend/internal/airouter/types.go` | `TaskType`, `ProviderType`, `LLMProvider`, sentinel errors |
| `backend/internal/airouter/router.go` `router_test.go` | `Router`, `DefaultStrategies`, `FallbackOrder`, `Route` |
| `backend/internal/airouter/gemini.go` `gemini_test.go` | `GeminiProvider` |
| `backend/internal/airouter/openai.go` `openai_test.go` | `OpenAICompatibleProvider` (OpenAI and DeepSeek) |
| `backend/internal/airouter/config.go` `config_test.go` | `Config`, `ConfigFromEnv`, `NewRouter` |
| `backend/internal/airouter/ratelimit.go` `ratelimit_test.go` | `RateLimiter`, `RedisRateLimiter`, `TestIntegrationRateLimiterAllowsFiveThenBlocks` |
| `backend/internal/airouter/prompt.go` `prompt_test.go` | `RoadmapSystemPrompt` (§6.1 verbatim), `RoadmapSchema`, `RoadmapUserPrompt` |
| `backend/internal/airouter/roadmap.go` `roadmap_test.go` | `Roadmap` types, `ParseRoadmap`, `Exercises()` |
| `backend/cmd/api/main.go` | build the router from env, log configured providers |
| `backend/.env.example` | commented provider variables |
| `harness/CODEMAP.md` | `airouter` paragraph |

---

## Tasks

### Task 1: Types and the router

**Files:**
- Create: `backend/internal/airouter/types.go`
- Create: `backend/internal/airouter/router.go`
- Test: `backend/internal/airouter/router_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/airouter/router_test.go`:
```go
package airouter

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// scripted is an LLMProvider that records calls and returns a fixed answer.
type scripted struct {
	name  string
	out   string
	err   error
	calls []string
}

func (s *scripted) GenerateContent(_ context.Context, system, user string) (string, error) {
	s.calls = append(s.calls, system+"|"+user)
	return s.out, s.err
}

func TestDefaultStrategiesMatchSpec62(t *testing.T) {
	want := map[TaskType]ProviderType{
		TaskRoadmapGen:    ProviderGemini,
		TaskExerciseGen:   ProviderDeepSeek,
		TaskPlacementTest: ProviderGemini,
		TaskEssayGrading:  ProviderOpenAI,
	}
	if !reflect.DeepEqual(DefaultStrategies(), want) {
		t.Errorf("DefaultStrategies() = %v, want %v", DefaultStrategies(), want)
	}
	if got := FallbackOrder; !reflect.DeepEqual(got, []ProviderType{ProviderGemini, ProviderOpenAI, ProviderDeepSeek}) {
		t.Errorf("FallbackOrder = %v", got)
	}
}

func TestRouteUsesTheStrategyProvider(t *testing.T) {
	gemini := &scripted{name: "gemini", out: `{"from":"gemini"}`}
	openai := &scripted{name: "openai", out: `{"from":"openai"}`}
	r := NewRouterWithProviders(map[ProviderType]LLMProvider{ProviderGemini: gemini, ProviderOpenAI: openai})

	out, err := r.Route(context.Background(), TaskEssayGrading, "sys", "usr")
	if err != nil || out != `{"from":"openai"}` {
		t.Fatalf("Route(essay) = %q, %v; want openai", out, err)
	}
	if len(gemini.calls) != 0 || len(openai.calls) != 1 || openai.calls[0] != "sys|usr" {
		t.Errorf("calls: gemini=%v openai=%v", gemini.calls, openai.calls)
	}
}

func TestRouteFallsBackInFixedOrderWhenThePreferredProviderIsMissing(t *testing.T) {
	openai := &scripted{name: "openai", out: "o"}
	deepseek := &scripted{name: "deepseek", out: "d"}
	r := NewRouterWithProviders(map[ProviderType]LLMProvider{ProviderOpenAI: openai, ProviderDeepSeek: deepseek})

	// roadmap_generation prefers gemini (absent) → openai before deepseek.
	out, err := r.Route(context.Background(), TaskRoadmapGen, "s", "u")
	if err != nil || out != "o" {
		t.Fatalf("Route = %q, %v; want the openai fallback", out, err)
	}
	if len(deepseek.calls) != 0 {
		t.Error("deepseek was called although openai succeeded")
	}
}

func TestRouteFallsBackWhenThePreferredProviderErrors(t *testing.T) {
	gemini := &scripted{name: "gemini", err: errors.New("503 overloaded")}
	deepseek := &scripted{name: "deepseek", out: "d"}
	r := NewRouterWithProviders(map[ProviderType]LLMProvider{ProviderGemini: gemini, ProviderDeepSeek: deepseek})

	out, err := r.Route(context.Background(), TaskPlacementTest, "s", "u")
	if err != nil || out != "d" {
		t.Fatalf("Route = %q, %v; want deepseek after gemini failed", out, err)
	}
	if len(gemini.calls) != 1 || len(deepseek.calls) != 1 {
		t.Errorf("calls: gemini=%d deepseek=%d, want 1 and 1", len(gemini.calls), len(deepseek.calls))
	}
}

func TestRouteJoinsEveryErrorWhenAllProvidersFail(t *testing.T) {
	gemini := &scripted{err: errors.New("gemini down")}
	openai := &scripted{err: errors.New("openai down")}
	r := NewRouterWithProviders(map[ProviderType]LLMProvider{ProviderGemini: gemini, ProviderOpenAI: openai})

	_, err := r.Route(context.Background(), TaskRoadmapGen, "s", "u")
	if err == nil {
		t.Fatal("err = nil, want every provider's error")
	}
	if !errors.Is(err, ErrAllProvidersFailed) {
		t.Errorf("err = %v, want ErrAllProvidersFailed", err)
	}
	for _, want := range []string{"gemini down", "openai down"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v, missing %q", err, want)
		}
	}
}

func TestRouteWithNoProvidersReturnsErrNoProviders(t *testing.T) {
	r := NewRouterWithProviders(nil)
	if _, err := r.Route(context.Background(), TaskRoadmapGen, "s", "u"); !errors.Is(err, ErrNoProviders) {
		t.Fatalf("err = %v, want ErrNoProviders", err)
	}
	if got := r.Providers(); len(got) != 0 {
		t.Errorf("Providers() = %v, want empty", got)
	}
}

func TestRouteStopsFallingBackOnceTheContextIsDone(t *testing.T) {
	gemini := &scripted{err: errors.New("slow")}
	openai := &scripted{out: "o"}
	r := NewRouterWithProviders(map[ProviderType]LLMProvider{ProviderGemini: gemini, ProviderOpenAI: openai})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := r.Route(ctx, TaskRoadmapGen, "s", "u"); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if len(openai.calls) != 0 {
		t.Error("fell back to openai on a cancelled context")
	}
}

func TestUnknownTaskDefaultsToGemini(t *testing.T) {
	gemini := &scripted{out: "g"}
	r := NewRouterWithProviders(map[ProviderType]LLMProvider{ProviderGemini: gemini})
	if out, err := r.Route(context.Background(), TaskType("something_new"), "s", "u"); err != nil || out != "g" {
		t.Fatalf("Route(unknown) = %q, %v; want gemini (§6.2 default)", out, err)
	}
}

func TestProvidersListsInFallbackOrder(t *testing.T) {
	r := NewRouterWithProviders(map[ProviderType]LLMProvider{ProviderDeepSeek: &scripted{}, ProviderGemini: &scripted{}})
	if got := r.Providers(); !reflect.DeepEqual(got, []ProviderType{ProviderGemini, ProviderDeepSeek}) {
		t.Errorf("Providers() = %v", got)
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
mkdir -p internal/airouter && go test ./internal/airouter/...
```
Expected: build failure, `undefined: TaskType`.

- [ ] **Step 3: Write `types.go`**

```go
// Package airouter routes AI tasks to LLM providers with fallback
// (1st-thinking doc §6.2), rate-limits AI calls per user (§4), and owns the
// roadmap prompt (§6.1) and the JSON shape it must produce.
package airouter

import (
	"context"
	"errors"
)

// TaskType names an AI job (§6.2).
type TaskType string

const (
	TaskPlacementTest TaskType = "placement_test"
	TaskRoadmapGen    TaskType = "roadmap_generation"
	TaskExerciseGen   TaskType = "exercise_generation"
	TaskEssayGrading  TaskType = "essay_grading"
)

// ProviderType names a vendor (§6.2).
type ProviderType string

const (
	ProviderGemini   ProviderType = "gemini"
	ProviderOpenAI   ProviderType = "openai"
	ProviderDeepSeek ProviderType = "deepseek"
)

// LLMProvider is one JSON-mode completion. Implementations must return the
// model's raw text; parsing and validation are the caller's job.
type LLMProvider interface {
	GenerateContent(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

var (
	// ErrNoProviders means no provider env var was set at boot. Callers map it
	// to 503.
	ErrNoProviders = errors.New("airouter: no providers configured")
	// ErrAllProvidersFailed wraps every provider's error after fallback ran out.
	ErrAllProvidersFailed = errors.New("airouter: all providers failed")
	// ErrRateLimited is returned by RateLimiter.Allow past 5 calls/min (§4).
	// Callers map it to 429.
	ErrRateLimited = errors.New("airouter: rate limited")
	// ErrInvalidRoadmap wraps every ParseRoadmap rejection.
	ErrInvalidRoadmap = errors.New("airouter: invalid roadmap")
)
```

- [ ] **Step 4: Write `router.go`**

```go
package airouter

import (
	"context"
	"errors"
	"fmt"
	"log"
)

// FallbackOrder is the deterministic order Route tries providers other than
// the strategy's. §6.2 iterated a Go map (random); this is fixed so behaviour
// under an outage is predictable and testable.
var FallbackOrder = []ProviderType{ProviderGemini, ProviderOpenAI, ProviderDeepSeek}

// DefaultStrategies is the §6.2 task → preferred provider table.
func DefaultStrategies() map[TaskType]ProviderType {
	return map[TaskType]ProviderType{
		TaskRoadmapGen:    ProviderGemini,
		TaskExerciseGen:   ProviderDeepSeek,
		TaskPlacementTest: ProviderGemini,
		TaskEssayGrading:  ProviderOpenAI,
	}
}

// Router picks a provider per task and falls back across the rest.
type Router struct {
	providers  map[ProviderType]LLMProvider
	strategies map[TaskType]ProviderType
}

// NewRouterWithProviders builds a router over already-constructed providers
// with the default strategies. NewRouter (config.go) is the env-driven path.
func NewRouterWithProviders(providers map[ProviderType]LLMProvider) *Router {
	if providers == nil {
		providers = map[ProviderType]LLMProvider{}
	}
	return &Router{providers: providers, strategies: DefaultStrategies()}
}

// Providers lists the configured providers in FallbackOrder.
func (r *Router) Providers() []ProviderType {
	out := make([]ProviderType, 0, len(r.providers))
	for _, p := range FallbackOrder {
		if _, ok := r.providers[p]; ok {
			out = append(out, p)
		}
	}
	return out
}

// Route sends the prompt to the task's preferred provider, then to the others
// in FallbackOrder, on absence or error. An unknown task prefers Gemini
// (§6.2). With no providers it returns ErrNoProviders; when every attempt
// fails it returns ErrAllProvidersFailed joined with each provider's error.
// It stops as soon as ctx is done.
func (r *Router) Route(ctx context.Context, task TaskType, systemPrompt, userPrompt string) (string, error) {
	if len(r.providers) == 0 {
		return "", ErrNoProviders
	}
	preferred, ok := r.strategies[task]
	if !ok {
		preferred = ProviderGemini
	}

	order := []ProviderType{preferred}
	for _, p := range FallbackOrder {
		if p != preferred {
			order = append(order, p)
		}
	}

	errs := []error{ErrAllProvidersFailed}
	for _, p := range order {
		provider, exists := r.providers[p]
		if !exists {
			continue
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if p != preferred {
			log.Printf("airouter: fallback from %s to %s for task %s", preferred, p, task)
		}
		out, err := provider.GenerateContent(ctx, systemPrompt, userPrompt)
		if err == nil {
			return out, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", p, err))
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return "", errors.Join(errs...)
}
```

- [ ] **Step 5: Run and confirm it passes**

```sh
go test ./internal/airouter/... -v
```
Expected: every `--- PASS`.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend/internal/airouter && git commit -m "airouter: task/provider types and Router with deterministic fallback"
```

---

### Task 2: Gemini provider

**Files:**
- Create: `backend/internal/airouter/gemini.go`
- Test: `backend/internal/airouter/gemini_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/airouter/gemini_test.go`:
```go
package airouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiSendsTheSpec62RequestAndReturnsTheFirstPart(t *testing.T) {
	var gotPath, gotKey, gotCT string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotKey, gotCT = r.URL.Path, r.Header.Get("x-goog-api-key"), r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"ok\":true}"}]}}]}`))
	}))
	defer srv.Close()

	p := NewGeminiProvider("k-123", srv.URL, "gemini-2.5-flash", srv.Client())
	out, err := p.GenerateContent(context.Background(), "SYS", "USR")
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	if out != `{"ok":true}` {
		t.Errorf("out = %q", out)
	}
	if gotPath != "/v1beta/models/gemini-2.5-flash:generateContent" {
		t.Errorf("path = %q", gotPath)
	}
	if gotKey != "k-123" || gotCT != "application/json" {
		t.Errorf("headers: key=%q content-type=%q", gotKey, gotCT)
	}
	if strings.Contains(srv.URL+gotPath, "key=") {
		t.Error("API key must travel in a header, not the query string")
	}

	sys := gotBody["system_instruction"].(map[string]any)["parts"].([]any)[0].(map[string]any)["text"]
	if sys != "SYS" {
		t.Errorf("system_instruction text = %v", sys)
	}
	content := gotBody["contents"].([]any)[0].(map[string]any)
	if content["role"] != "user" || content["parts"].([]any)[0].(map[string]any)["text"] != "USR" {
		t.Errorf("contents[0] = %v", content)
	}
	gen := gotBody["generationConfig"].(map[string]any)
	if gen["response_mime_type"] != "application/json" || gen["temperature"] != 0.2 {
		t.Errorf("generationConfig = %v, want application/json and 0.2", gen)
	}
}

func TestGeminiRejectsNon2xxEmptyCandidatesAndBadJSON(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
		want   string
	}{
		"500":              {500, `{"error":"boom"}`, "status 500"},
		"429":              {429, `rate`, "status 429"},
		"empty candidates": {200, `{"candidates":[]}`, "empty"},
		"empty parts":      {200, `{"candidates":[{"content":{"parts":[]}}]}`, "empty"},
		"not json":         {200, `<html>`, "decoding"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			p := NewGeminiProvider("k", srv.URL, "m", srv.Client())
			_, err := p.GenerateContent(context.Background(), "s", "u")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestGeminiDefaultsToTheRealEndpoint(t *testing.T) {
	p := NewGeminiProvider("k", "", "", nil)
	if p.baseURL != DefaultGeminiBaseURL || p.model != DefaultGeminiModel || p.client == nil {
		t.Errorf("defaults = %q %q client=%v", p.baseURL, p.model, p.client)
	}
	if DefaultGeminiBaseURL != "https://generativelanguage.googleapis.com" {
		t.Errorf("DefaultGeminiBaseURL = %q", DefaultGeminiBaseURL)
	}
	if p.client.Timeout != ProviderTimeout {
		t.Errorf("timeout = %v, want %v (§6.2: 30s)", p.client.Timeout, ProviderTimeout)
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/airouter/... -run Gemini
```
Expected: build failure, `undefined: NewGeminiProvider`.

- [ ] **Step 3: Implement**

`backend/internal/airouter/gemini.go`:
```go
package airouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ProviderTimeout is the §6.2 HTTP client timeout.
const ProviderTimeout = 30 * time.Second

// Gemini defaults. §6.2's "gemini.api.internal" was a placeholder; this is the
// real Generative Language API. The model is configurable because names churn.
const (
	DefaultGeminiBaseURL = "https://generativelanguage.googleapis.com"
	DefaultGeminiModel   = "gemini-2.5-flash"
)

// GeminiProvider calls models/{model}:generateContent in JSON mode.
type GeminiProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewGeminiProvider builds a provider. Empty baseURL/model/client take the
// defaults; tests pass an httptest server URL.
func NewGeminiProvider(apiKey, baseURL, model string, client *http.Client) *GeminiProvider {
	if baseURL == "" {
		baseURL = DefaultGeminiBaseURL
	}
	if model == "" {
		model = DefaultGeminiModel
	}
	if client == nil {
		client = &http.Client{Timeout: ProviderTimeout}
	}
	return &GeminiProvider{apiKey: apiKey, baseURL: strings.TrimRight(baseURL, "/"), model: model, client: client}
}

func (g *GeminiProvider) GenerateContent(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", g.baseURL, g.model)
	reqBody := map[string]any{
		"system_instruction": map[string]any{"parts": []map[string]string{{"text": systemPrompt}}},
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]string{{"text": userPrompt}}},
		},
		"generationConfig": map[string]any{
			"response_mime_type": "application/json",
			"temperature":        0.2,
		},
	}
	body, err := postJSON(ctx, g.client, url, reqBody, map[string]string{"x-goog-api-key": g.apiKey})
	if err != nil {
		return "", fmt.Errorf("gemini: %w", err)
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("gemini: decoding response: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}
	return parsed.Candidates[0].Content.Parts[0].Text, nil
}

// postJSON is shared by both providers: marshal, POST, require 2xx, return the
// body. Error bodies are truncated so a verbose upstream cannot flood logs.
func postJSON(ctx context.Context, client *http.Client, url string, reqBody any, headers map[string]string) ([]byte, error) {
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(body, 512))
	}
	return body, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}

var _ LLMProvider = (*GeminiProvider)(nil)
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/airouter/... -run Gemini -v
```
Expected: every `--- PASS`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/airouter/gemini.go backend/internal/airouter/gemini_test.go && git commit -m "airouter: Gemini provider over an injectable base URL, key in header, JSON mode"
```

---

### Task 3: OpenAI-compatible provider (OpenAI and DeepSeek)

**Files:**
- Create: `backend/internal/airouter/openai.go`
- Test: `backend/internal/airouter/openai_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/airouter/openai_test.go`:
```go
package airouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAICompatibleSendsTheSpec62RequestAndReturnsTheFirstChoice(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"cefr_level\":\"B1\"}"}}]}`))
	}))
	defer srv.Close()

	p := NewOpenAICompatibleProvider(srv.URL+"/v1/", "sk-1", "deepseek-chat", srv.Client())
	out, err := p.GenerateContent(context.Background(), "SYS", "USR")
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	if out != `{"cefr_level":"B1"}` {
		t.Errorf("out = %q", out)
	}
	if gotPath != "/v1/chat/completions" {
		t.Errorf("path = %q (trailing slash on the base URL must not double up)", gotPath)
	}
	if gotAuth != "Bearer sk-1" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotBody["model"] != "deepseek-chat" || gotBody["temperature"] != 0.2 {
		t.Errorf("model/temperature = %v/%v", gotBody["model"], gotBody["temperature"])
	}
	if gotBody["response_format"].(map[string]any)["type"] != "json_object" {
		t.Errorf("response_format = %v", gotBody["response_format"])
	}
	msgs := gotBody["messages"].([]any)
	if len(msgs) != 2 || msgs[0].(map[string]any)["role"] != "system" || msgs[0].(map[string]any)["content"] != "SYS" ||
		msgs[1].(map[string]any)["role"] != "user" || msgs[1].(map[string]any)["content"] != "USR" {
		t.Errorf("messages = %v", msgs)
	}
}

func TestOpenAICompatibleRejectsNon2xxEmptyChoicesAndBadJSON(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
		want   string
	}{
		"401":           {401, `{"error":{"message":"bad key"}}`, "status 401"},
		"empty choices": {200, `{"choices":[]}`, "empty"},
		"not json":      {200, `oops`, "decoding"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			p := NewOpenAICompatibleProvider(srv.URL, "k", "m", srv.Client())
			_, err := p.GenerateContent(context.Background(), "s", "u")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestOpenAICompatibleDefaults(t *testing.T) {
	p := NewOpenAICompatibleProvider("", "k", "", nil)
	if p.baseURL != DefaultOpenAIBaseURL || p.model != DefaultOpenAIModel || p.client.Timeout != ProviderTimeout {
		t.Errorf("defaults = %q %q %v", p.baseURL, p.model, p.client.Timeout)
	}
	if DefaultOpenAIBaseURL != "https://api.openai.com/v1" || DefaultDeepSeekBaseURL != "https://api.deepseek.com/v1" {
		t.Errorf("base URLs = %q %q", DefaultOpenAIBaseURL, DefaultDeepSeekBaseURL)
	}
	if DefaultOpenAIModel != "gpt-4o-mini" || DefaultDeepSeekModel != "deepseek-chat" {
		t.Errorf("models = %q %q (§6.2)", DefaultOpenAIModel, DefaultDeepSeekModel)
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/airouter/... -run OpenAI
```
Expected: build failure, `undefined: NewOpenAICompatibleProvider`.

- [ ] **Step 3: Implement**

`backend/internal/airouter/openai.go`:
```go
package airouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Defaults for the two OpenAI-compatible vendors (§2.1, §6.2).
const (
	DefaultOpenAIBaseURL   = "https://api.openai.com/v1"
	DefaultOpenAIModel     = "gpt-4o-mini"
	DefaultDeepSeekBaseURL = "https://api.deepseek.com/v1"
	DefaultDeepSeekModel   = "deepseek-chat"
)

// OpenAICompatibleProvider speaks the chat/completions dialect shared by
// OpenAI and DeepSeek.
type OpenAICompatibleProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewOpenAICompatibleProvider builds a provider. An empty baseURL/model means
// OpenAI's; DeepSeek callers pass DefaultDeepSeekBaseURL/DefaultDeepSeekModel
// (see config.go).
func NewOpenAICompatibleProvider(baseURL, apiKey, model string, client *http.Client) *OpenAICompatibleProvider {
	if baseURL == "" {
		baseURL = DefaultOpenAIBaseURL
	}
	if model == "" {
		model = DefaultOpenAIModel
	}
	if client == nil {
		client = &http.Client{Timeout: ProviderTimeout}
	}
	return &OpenAICompatibleProvider{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, model: model, client: client}
}

func (o *OpenAICompatibleProvider) GenerateContent(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := map[string]any{
		"model": o.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.2,
	}
	body, err := postJSON(ctx, o.client, o.baseURL+"/chat/completions", reqBody,
		map[string]string{"Authorization": "Bearer " + o.apiKey})
	if err != nil {
		return "", fmt.Errorf("openai-compat(%s): %w", o.model, err)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("openai-compat(%s): decoding response: %w", o.model, err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("openai-compat(%s): empty choices", o.model)
	}
	return parsed.Choices[0].Message.Content, nil
}

var _ LLMProvider = (*OpenAICompatibleProvider)(nil)
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/airouter/... -run OpenAI -v
```
Expected: every `--- PASS`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/airouter/openai.go backend/internal/airouter/openai_test.go && git commit -m "airouter: OpenAI-compatible provider shared by OpenAI and DeepSeek"
```

---

### Task 4: Env-driven configuration — providers register only when their key is set

**Files:**
- Create: `backend/internal/airouter/config.go`
- Test: `backend/internal/airouter/config_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/airouter/config_test.go`:
```go
package airouter

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func lookup(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestConfigFromEnvReadsTheSpec9VariablesAndDefaults(t *testing.T) {
	cfg := ConfigFromEnv(lookup(map[string]string{
		"GEMINI_API_KEY":   "g",
		"OPENAI_API_KEY":   "o",
		"DEEPSEEK_API_KEY": "d",
		"DEEPSEEK_BASE_URL": "http://ds.local/v1",
		"GEMINI_MODEL":     "gemini-2.5-pro",
	}))
	want := Config{
		GeminiAPIKey: "g", GeminiBaseURL: DefaultGeminiBaseURL, GeminiModel: "gemini-2.5-pro",
		OpenAIAPIKey: "o", OpenAIBaseURL: DefaultOpenAIBaseURL, OpenAIModel: DefaultOpenAIModel,
		DeepSeekAPIKey: "d", DeepSeekBaseURL: "http://ds.local/v1", DeepSeekModel: DefaultDeepSeekModel,
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("ConfigFromEnv =\n%+v\nwant\n%+v", cfg, want)
	}
}

func TestNewRouterRegistersOnlyProvidersWithAKey(t *testing.T) {
	cases := []struct {
		env  map[string]string
		want []ProviderType
	}{
		{map[string]string{}, nil},
		{map[string]string{"GEMINI_API_KEY": "g"}, []ProviderType{ProviderGemini}},
		{map[string]string{"DEEPSEEK_API_KEY": "d", "OPENAI_BASE_URL": "http://x"}, []ProviderType{ProviderDeepSeek}}, // a base URL alone registers nothing
		{map[string]string{"GEMINI_API_KEY": "g", "OPENAI_API_KEY": "o", "DEEPSEEK_API_KEY": "d"}, []ProviderType{ProviderGemini, ProviderOpenAI, ProviderDeepSeek}},
	}
	for _, tc := range cases {
		r := NewRouter(ConfigFromEnv(lookup(tc.env)))
		got := r.Providers()
		if len(got) == 0 && len(tc.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("env %v → Providers() = %v, want %v", tc.env, got, tc.want)
		}
	}
}

func TestNewRouterWithNoKeysBootsButRoutesToErrNoProviders(t *testing.T) {
	r := NewRouter(ConfigFromEnv(lookup(nil)))
	if r == nil {
		t.Fatal("NewRouter returned nil; the binary must boot without AI keys")
	}
	if _, err := r.Route(context.Background(), TaskRoadmapGen, "s", "u"); !errors.Is(err, ErrNoProviders) {
		t.Errorf("err = %v, want ErrNoProviders", err)
	}
}

func TestNewRouterWiresDeepSeekOntoTheOpenAIDialect(t *testing.T) {
	r := NewRouter(Config{DeepSeekAPIKey: "d", DeepSeekBaseURL: "http://ds", DeepSeekModel: "deepseek-chat"})
	p, ok := r.providers[ProviderDeepSeek].(*OpenAICompatibleProvider)
	if !ok {
		t.Fatalf("deepseek provider is %T, want *OpenAICompatibleProvider", r.providers[ProviderDeepSeek])
	}
	if p.baseURL != "http://ds" || p.model != "deepseek-chat" || p.apiKey != "d" {
		t.Errorf("deepseek = %+v", p)
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/airouter/... -run 'Config|NewRouter'
```
Expected: build failure, `undefined: ConfigFromEnv`.

- [ ] **Step 3: Implement**

`backend/internal/airouter/config.go`:
```go
package airouter

import "net/http"

// Config is everything NewRouter needs. Keys are the backend spec §9 / §8
// variables; base URLs and models are optional overrides (tests, proxies,
// model churn). It lives here rather than in internal/config so the package
// is testable with a plain lookup func and config.Load's "required variables"
// contract is unchanged — AI keys are optional.
type Config struct {
	GeminiAPIKey  string
	GeminiBaseURL string
	GeminiModel   string

	OpenAIAPIKey  string
	OpenAIBaseURL string
	OpenAIModel   string

	DeepSeekAPIKey  string
	DeepSeekBaseURL string
	DeepSeekModel   string

	// HTTPClient, when set, is shared by every provider (tests).
	HTTPClient *http.Client
}

// ConfigFromEnv reads the provider variables through lookup (os.Getenv in
// main) and fills defaults for every optional value.
func ConfigFromEnv(lookup func(string) string) Config {
	or := func(v, def string) string {
		if v == "" {
			return def
		}
		return v
	}
	return Config{
		GeminiAPIKey:  lookup("GEMINI_API_KEY"),
		GeminiBaseURL: or(lookup("GEMINI_BASE_URL"), DefaultGeminiBaseURL),
		GeminiModel:   or(lookup("GEMINI_MODEL"), DefaultGeminiModel),

		OpenAIAPIKey:  lookup("OPENAI_API_KEY"),
		OpenAIBaseURL: or(lookup("OPENAI_BASE_URL"), DefaultOpenAIBaseURL),
		OpenAIModel:   or(lookup("OPENAI_MODEL"), DefaultOpenAIModel),

		DeepSeekAPIKey:  lookup("DEEPSEEK_API_KEY"),
		DeepSeekBaseURL: or(lookup("DEEPSEEK_BASE_URL"), DefaultDeepSeekBaseURL),
		DeepSeekModel:   or(lookup("DEEPSEEK_MODEL"), DefaultDeepSeekModel),
	}
}

// NewRouter registers one provider per configured API key (§6.2 NewRouter).
// Unlike the pseudocode it never fails: with no keys the router boots and
// Route returns ErrNoProviders, so a developer without AI credentials can
// still run every non-AI route.
func NewRouter(cfg Config) *Router {
	providers := map[ProviderType]LLMProvider{}
	if cfg.GeminiAPIKey != "" {
		providers[ProviderGemini] = NewGeminiProvider(cfg.GeminiAPIKey, cfg.GeminiBaseURL, cfg.GeminiModel, cfg.HTTPClient)
	}
	if cfg.OpenAIAPIKey != "" {
		providers[ProviderOpenAI] = NewOpenAICompatibleProvider(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.HTTPClient)
	}
	if cfg.DeepSeekAPIKey != "" {
		base, model := cfg.DeepSeekBaseURL, cfg.DeepSeekModel
		if base == "" {
			base = DefaultDeepSeekBaseURL
		}
		if model == "" {
			model = DefaultDeepSeekModel
		}
		providers[ProviderDeepSeek] = NewOpenAICompatibleProvider(base, cfg.DeepSeekAPIKey, model, cfg.HTTPClient)
	}
	return NewRouterWithProviders(providers)
}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/airouter/... -run 'Config|NewRouter' -v
```
Expected: every `--- PASS`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/airouter/config.go backend/internal/airouter/config_test.go && git commit -m "airouter: env-driven Config; providers register only when their API key is set"
```

---

### Task 5: The `ratelimit:ai:{user_id}` limiter

**Files:**
- Create: `backend/internal/airouter/ratelimit.go`
- Test: `backend/internal/airouter/ratelimit_test.go`

The live test is named `TestIntegration…` so CI's `backend-integration` job counts it and fails if it
skips. It is gated on `TEST_REDIS_URL` only — never `REDIS_URL` — matching `internal/store`.

- [ ] **Step 1: Write the failing tests**

`backend/internal/airouter/ratelimit_test.go`:
```go
package airouter

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

func TestRateLimitConstantsMatchSpec4(t *testing.T) {
	if AILimitPerMinute != 5 {
		t.Errorf("AILimitPerMinute = %d, want 5 (spec §4)", AILimitPerMinute)
	}
	if store.AIRateLimitTTL != time.Minute {
		t.Errorf("store.AIRateLimitTTL = %v, want 1m", store.AIRateLimitTTL)
	}
	if store.AIRateLimitKey("u") != "ratelimit:ai:u" {
		t.Errorf("store.AIRateLimitKey = %q", store.AIRateLimitKey("u"))
	}
}

// Gated on TEST_REDIS_URL, never the production REDIS_URL (spec §9). CI exports
// TEST_* and fails on --- SKIP; run with -p 1 (make test-integration).
func TestIntegrationRateLimiterAllowsFiveThenBlocks(t *testing.T) {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL is unset; run `make up` and export it to run integration tests")
	}
	ctx := context.Background()
	rdb, err := store.NewRedis(ctx, url)
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	const user = "airouter-integration-user"
	key := store.AIRateLimitKey(user)
	rdb.Client.Del(ctx, key)
	t.Cleanup(func() { rdb.Client.Del(ctx, key) })

	rl := NewRedisRateLimiter(rdb)
	for i := 1; i <= AILimitPerMinute; i++ {
		if err := rl.Allow(ctx, user); err != nil {
			t.Fatalf("call %d: %v, want allowed", i, err)
		}
	}
	if err := rl.Allow(ctx, user); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("call %d: err = %v, want ErrRateLimited", AILimitPerMinute+1, err)
	}

	ttl, err := rdb.Client.TTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if ttl <= 0 || ttl > store.AIRateLimitTTL {
		t.Errorf("TTL = %v, want within (0, %v] — the key must expire even after six INCRs", ttl, store.AIRateLimitTTL)
	}

	// Another user is unaffected.
	if err := rl.Allow(ctx, user+"-2"); err != nil {
		t.Errorf("second user: %v, want allowed", err)
	}
	rdb.Client.Del(ctx, store.AIRateLimitKey(user+"-2"))
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/airouter/... -run RateLimit
```
Expected: build failure, `undefined: AILimitPerMinute`.

- [ ] **Step 3: Implement**

`backend/internal/airouter/ratelimit.go`:
```go
package airouter

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// AILimitPerMinute is the §4 cap on ratelimit:ai:{user_id}.
const AILimitPerMinute = 5

// RateLimiter gates AI calls per user. Allow returns nil when the call may
// proceed and ErrRateLimited when the user is over the limit.
type RateLimiter interface {
	Allow(ctx context.Context, userID string) error
}

// RedisRateLimiter is the real RateLimiter: INCR the §4 key and set its TTL
// only if it has none (EXPIRE NX, Redis ≥ 7), both in one pipeline, so a
// crash between the two commands can never leave a key that never expires.
type RedisRateLimiter struct {
	Client *redis.Client
	Limit  int64
}

// NewRedisRateLimiter builds a limiter over an existing client at the §4 limit.
func NewRedisRateLimiter(r *store.Redis) *RedisRateLimiter {
	return &RedisRateLimiter{Client: r.Client, Limit: AILimitPerMinute}
}

func (l *RedisRateLimiter) Allow(ctx context.Context, userID string) error {
	key := store.AIRateLimitKey(userID)
	pipe := l.Client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, store.AIRateLimitTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("airouter: rate limit: %w", err)
	}
	if incr.Val() > l.Limit {
		return ErrRateLimited
	}
	return nil
}

var _ RateLimiter = (*RedisRateLimiter)(nil)
```

- [ ] **Step 4: Confirm the unit test passes and the live one skips**

```sh
env -u REDIS_URL -u TEST_REDIS_URL go test ./internal/airouter/... -run RateLimit -v
```
Expected: `--- PASS: TestRateLimitConstantsMatchSpec4`, `--- SKIP: TestIntegrationRateLimiterAllowsFiveThenBlocks`.

- [ ] **Step 5: Run it for real if Docker is available**

```sh
POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose up -d --wait
export TEST_DATABASE_URL='postgres://english:english@localhost:5433/english?sslmode=disable'
export TEST_REDIS_URL='redis://localhost:6380/0'
make test-integration
docker compose down
```
Expected: `--- PASS: TestIntegrationRateLimiterAllowsFiveThenBlocks` among the others, no `--- SKIP`.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend/internal/airouter/ratelimit.go backend/internal/airouter/ratelimit_test.go && git commit -m "airouter: ratelimit:ai:{user_id} — INCR + EXPIRE NX, 5/min, ErrRateLimited"
```

---

### Task 6: Roadmap schema, prompt constants and the strict parser

**Files:**
- Create: `backend/internal/airouter/prompt.go`
- Create: `backend/internal/airouter/roadmap.go`
- Test: `backend/internal/airouter/prompt_test.go`
- Test: `backend/internal/airouter/roadmap_test.go`

§6.1 says "matching the requested schema" and never states one. This is the schema, kept next to the
validator so they cannot drift. `content_json` for each exercise is the **task object itself**, so the
`title` and `duration_minutes` keys the quests slice's `toTask` reads are present by construction.

- [ ] **Step 1: Write the failing tests**

`backend/internal/airouter/prompt_test.go`:
```go
package airouter

import (
	"strings"
	"testing"
)

func TestRoadmapSystemPromptIsSpec61Verbatim(t *testing.T) {
	for _, want := range []string{
		"You are an elite AI Language Curriculum Architect.",
		"CRITICAL CONSTRAINTS:",
		"1. Output ONLY valid JSON matching the requested schema. No markdown backticks, no code blocks, no conversational preamble.",
		"2. Structure the output into 4 distinct Modules (Weeks).",
		"3. Each Module must contain 7 Daily Quests (Total 28 Days).",
		"4. Each Daily Quest MUST be calculated to take approximately 30 minutes to complete, split into 3 distinct tasks (10 mins each): Vocabulary/Grammar, Reading/Listening, and Practice/Interactive.",
		"5. Difficulty must scale progressively across the modules.",
	} {
		if !strings.Contains(RoadmapSystemPrompt, want) {
			t.Errorf("RoadmapSystemPrompt missing %q", want)
		}
	}
}

func TestRoadmapUserPromptCarriesTheLearnerAndTheSchema(t *testing.T) {
	p := RoadmapUserPrompt("B1", "IELTS 7.0 Preparation", 30)
	for _, want := range []string{"B1", "IELTS 7.0 Preparation", "30 minutes", RoadmapSchema, `"vocabulary"`, `"reading"`, `"practice"`} {
		if !strings.Contains(p, want) {
			t.Errorf("RoadmapUserPrompt missing %q", want)
		}
	}
}
```

`backend/internal/airouter/roadmap_test.go`:
```go
package airouter

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// validRoadmapJSON builds a 4x7x3 roadmap; mutate lets a test break one thing.
func validRoadmapJSON(t *testing.T, mutate func(r *Roadmap)) string {
	t.Helper()
	r := Roadmap{Title: "Road to IELTS 7", CEFRLevel: "B1"}
	for m := 1; m <= Modules; m++ {
		mod := Module{Week: m, Title: fmt.Sprintf("Week %d", m), Focus: "focus"}
		for d := 1; d <= DaysPerModule; d++ {
			day := Day{Title: fmt.Sprintf("Day %d", (m-1)*DaysPerModule+d)}
			for _, tt := range TaskTypes {
				day.Tasks = append(day.Tasks, Task{Type: tt, Title: tt + " task", DurationMinutes: 10, Content: json.RawMessage(`{"items":[]}`)})
			}
			mod.Days = append(mod.Days, day)
		}
		r.Modules = append(r.Modules, mod)
	}
	if mutate != nil {
		mutate(&r)
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func TestParseRoadmapAcceptsTheExactShape(t *testing.T) {
	r, err := ParseRoadmap(validRoadmapJSON(t, nil))
	if err != nil {
		t.Fatalf("ParseRoadmap: %v", err)
	}
	if len(r.Modules) != 4 || len(r.Modules[3].Days) != 7 || len(r.Modules[3].Days[6].Tasks) != 3 {
		t.Errorf("shape = %d modules, %d days, %d tasks", len(r.Modules), len(r.Modules[3].Days), len(r.Modules[3].Days[6].Tasks))
	}
	if r.CEFRLevel != "B1" {
		t.Errorf("CEFRLevel = %q", r.CEFRLevel)
	}
}

func TestParseRoadmapToleratesSurroundingWhitespaceAndUnknownFields(t *testing.T) {
	raw := "\n  " + strings.Replace(validRoadmapJSON(t, nil), `{"title"`, `{"model_note":"hi","title"`, 1) + "\n"
	if _, err := ParseRoadmap(raw); err != nil {
		t.Fatalf("ParseRoadmap: %v", err)
	}
}

func TestParseRoadmapDefaultsAMissingDurationToTen(t *testing.T) {
	r, err := ParseRoadmap(validRoadmapJSON(t, func(r *Roadmap) { r.Modules[0].Days[0].Tasks[0].DurationMinutes = 0 }))
	if err != nil {
		t.Fatalf("ParseRoadmap: %v", err)
	}
	if got := r.Modules[0].Days[0].Tasks[0].DurationMinutes; got != DefaultTaskMinutes {
		t.Errorf("DurationMinutes = %d, want %d", got, DefaultTaskMinutes)
	}
}

func TestParseRoadmapRejects(t *testing.T) {
	cases := map[string]string{
		"markdown fence":      "```json\n" + validRoadmapJSON(t, nil) + "\n```",
		"preamble":            "Here is your roadmap: " + validRoadmapJSON(t, nil),
		"trailing garbage":    validRoadmapJSON(t, nil) + " {}",
		"empty":               "",
		"not an object":       `[1,2,3]`,
		"three modules":       validRoadmapJSON(t, func(r *Roadmap) { r.Modules = r.Modules[:3] }),
		"five modules":        validRoadmapJSON(t, func(r *Roadmap) { r.Modules = append(r.Modules, r.Modules[0]) }),
		"six days":            validRoadmapJSON(t, func(r *Roadmap) { r.Modules[1].Days = r.Modules[1].Days[:6] }),
		"two tasks":           validRoadmapJSON(t, func(r *Roadmap) { r.Modules[2].Days[3].Tasks = r.Modules[2].Days[3].Tasks[:2] }),
		"four tasks":          validRoadmapJSON(t, func(r *Roadmap) { d := &r.Modules[2].Days[3]; d.Tasks = append(d.Tasks, d.Tasks[0]) }),
		"duplicate task type": validRoadmapJSON(t, func(r *Roadmap) { r.Modules[0].Days[0].Tasks[1].Type = "vocabulary" }),
		"unknown task type":   validRoadmapJSON(t, func(r *Roadmap) { r.Modules[0].Days[0].Tasks[2].Type = "listening" }),
		"empty task title":    validRoadmapJSON(t, func(r *Roadmap) { r.Modules[3].Days[6].Tasks[0].Title = "" }),
		"absurd duration":     validRoadmapJSON(t, func(r *Roadmap) { r.Modules[3].Days[6].Tasks[0].DurationMinutes = 120 }),
		"bad cefr":            validRoadmapJSON(t, func(r *Roadmap) { r.CEFRLevel = "B7" }),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseRoadmap(raw)
			if err == nil {
				t.Fatal("ParseRoadmap accepted it")
			}
			if !errors.Is(err, ErrInvalidRoadmap) {
				t.Errorf("err = %v, want it to wrap ErrInvalidRoadmap", err)
			}
		})
	}
}

func TestExercisesFlattensTo84RowsCarryingTitleAndDuration(t *testing.T) {
	r, err := ParseRoadmap(validRoadmapJSON(t, nil))
	if err != nil {
		t.Fatalf("ParseRoadmap: %v", err)
	}
	ex := r.Exercises()
	if len(ex) != 84 {
		t.Fatalf("len = %d, want 84", len(ex))
	}
	if ex[0].DayNumber != 1 || ex[83].DayNumber != 28 || ex[21].DayNumber != 8 {
		t.Errorf("day numbers: first %d, [21] %d, last %d; want 1, 8, 28", ex[0].DayNumber, ex[21].DayNumber, ex[83].DayNumber)
	}
	seen := map[string]int{}
	for _, e := range ex {
		seen[e.TaskType]++
		var meta struct {
			Type            string `json:"type"`
			Title           string `json:"title"`
			DurationMinutes int    `json:"duration_minutes"`
		}
		if err := json.Unmarshal(e.ContentJSON, &meta); err != nil {
			t.Fatalf("content_json is not JSON: %v", err)
		}
		if meta.Title == "" || meta.DurationMinutes != 10 || meta.Type != e.TaskType {
			t.Errorf("content_json = %s, want title, duration_minutes=10 and type=%s (what quests' toTask reads)", e.ContentJSON, e.TaskType)
		}
	}
	if seen["vocabulary"] != 28 || seen["reading"] != 28 || seen["practice"] != 28 {
		t.Errorf("task type counts = %v, want 28 each", seen)
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/airouter/... -run 'Roadmap|Exercises|Prompt'
```
Expected: build failure, `undefined: RoadmapSystemPrompt`.

- [ ] **Step 3: Write `prompt.go`**

```go
package airouter

import "fmt"

// RoadmapSystemPrompt is the 1st-thinking doc §6.1 system prompt, verbatim
// (backslash escapes removed).
const RoadmapSystemPrompt = `You are an elite AI Language Curriculum Architect. Your job is to create a structured, highly personalized learning roadmap for an English learner based on their current CEFR level, target goal, and daily study time commitment.

CRITICAL CONSTRAINTS:

1. Output ONLY valid JSON matching the requested schema. No markdown backticks, no code blocks, no conversational preamble.

2. Structure the output into 4 distinct Modules (Weeks).

3. Each Module must contain 7 Daily Quests (Total 28 Days).

4. Each Daily Quest MUST be calculated to take approximately 30 minutes to complete, split into 3 distinct tasks (10 mins each): Vocabulary/Grammar, Reading/Listening, and Practice/Interactive.

5. Difficulty must scale progressively across the modules.`

// RoadmapSchema is "the requested schema" §6.1 refers to. ParseRoadmap
// enforces it; onboarding persists what passes. The three task types are the
// §3.2 task_category values.
const RoadmapSchema = `{
  "title": "string",
  "cefr_level": "A1|A2|B1|B2|C1|C2",
  "modules": [
    {
      "week": 1,
      "title": "string",
      "focus": "string",
      "days": [
        {
          "title": "string",
          "tasks": [
            {"type": "vocabulary", "title": "string", "duration_minutes": 10, "content": {}},
            {"type": "reading",    "title": "string", "duration_minutes": 10, "content": {}},
            {"type": "practice",   "title": "string", "duration_minutes": 10, "content": {}}
          ]
        }
      ]
    }
  ]
}
"modules" has exactly 4 entries, each "days" exactly 7, each "tasks" exactly 3 with the three types in that order. "content" is free-form JSON for the task material (word lists, passages, prompts).`

// RoadmapUserPrompt is the user turn for TaskRoadmapGen.
func RoadmapUserPrompt(cefrLevel, targetGoal string, dailyMinutes int) string {
	return fmt.Sprintf(`Learner profile:
- Current CEFR level: %s
- Target goal: %s
- Daily study time: %d minutes

Produce the 28-day roadmap as JSON with exactly this schema:
%s`, cefrLevel, targetGoal, dailyMinutes, RoadmapSchema)
}
```

- [ ] **Step 4: Write `roadmap.go`**

```go
package airouter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// The §6.1 shape.
const (
	Modules            = 4
	DaysPerModule      = 7
	TasksPerDay        = 3
	DefaultTaskMinutes = 10
	maxTaskMinutes     = 30
)

// TaskTypes are the §3.2 task_category values in the §6.1 order
// (Vocabulary/Grammar, Reading/Listening, Practice/Interactive).
var TaskTypes = []string{"vocabulary", "reading", "practice"}

var cefrLevels = map[string]bool{"A1": true, "A2": true, "B1": true, "B2": true, "C1": true, "C2": true}

// Roadmap is the validated §6.1 output; it is also what roadmaps.roadmap_json stores.
type Roadmap struct {
	Title     string   `json:"title"`
	CEFRLevel string   `json:"cefr_level"`
	Modules   []Module `json:"modules"`
}

// Module is one week.
type Module struct {
	Week  int    `json:"week"`
	Title string `json:"title"`
	Focus string `json:"focus"`
	Days  []Day  `json:"days"`
}

// Day is one daily quest.
type Day struct {
	Title string `json:"title"`
	Tasks []Task `json:"tasks"`
}

// Task is one ~10-minute exercise. Marshalled whole into exercises.content_json,
// so its title and duration_minutes are what GET /quests/daily exposes.
type Task struct {
	Type            string          `json:"type"`
	Title           string          `json:"title"`
	DurationMinutes int             `json:"duration_minutes"`
	Content         json.RawMessage `json:"content,omitempty"`
}

// Exercise is one §3.2 exercises row, ready to insert.
type Exercise struct {
	DayNumber   int
	TaskType    string
	ContentJSON json.RawMessage
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidRoadmap, fmt.Sprintf(format, args...))
}

// ParseRoadmap decodes a model response strictly: no markdown fences or
// preamble (§6.1 constraint 1), no trailing tokens, exactly 4 modules × 7 days
// × 3 tasks with the three task types each present once, non-empty task
// titles, durations within (0, 30] (a missing duration becomes 10), and a
// valid cefr_level. Unknown extra fields are tolerated. Nothing is stripped or
// repaired — a non-conforming answer is the caller's cue to retry.
func ParseRoadmap(raw string) (Roadmap, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Roadmap{}, invalid("empty response")
	}
	if strings.HasPrefix(trimmed, "```") {
		return Roadmap{}, invalid("response is wrapped in a markdown fence")
	}
	if !strings.HasPrefix(trimmed, "{") {
		return Roadmap{}, invalid("response does not start with a JSON object")
	}

	dec := json.NewDecoder(strings.NewReader(trimmed))
	var r Roadmap
	if err := dec.Decode(&r); err != nil {
		return Roadmap{}, invalid("decoding: %v", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return Roadmap{}, invalid("trailing content after the JSON object")
	}

	if !cefrLevels[r.CEFRLevel] {
		return Roadmap{}, invalid("cefr_level %q is not a CEFR level", r.CEFRLevel)
	}
	if len(r.Modules) != Modules {
		return Roadmap{}, invalid("%d modules, want %d", len(r.Modules), Modules)
	}
	for mi := range r.Modules {
		m := &r.Modules[mi]
		if len(m.Days) != DaysPerModule {
			return Roadmap{}, invalid("module %d has %d days, want %d", mi+1, len(m.Days), DaysPerModule)
		}
		for di := range m.Days {
			d := &m.Days[di]
			if len(d.Tasks) != TasksPerDay {
				return Roadmap{}, invalid("module %d day %d has %d tasks, want %d", mi+1, di+1, len(d.Tasks), TasksPerDay)
			}
			seen := map[string]bool{}
			for ti := range d.Tasks {
				task := &d.Tasks[ti]
				if !isTaskType(task.Type) {
					return Roadmap{}, invalid("module %d day %d task %d has type %q", mi+1, di+1, ti+1, task.Type)
				}
				if seen[task.Type] {
					return Roadmap{}, invalid("module %d day %d repeats task type %q", mi+1, di+1, task.Type)
				}
				seen[task.Type] = true
				if strings.TrimSpace(task.Title) == "" {
					return Roadmap{}, invalid("module %d day %d task %d has no title", mi+1, di+1, ti+1)
				}
				if task.DurationMinutes == 0 {
					task.DurationMinutes = DefaultTaskMinutes
				}
				if task.DurationMinutes < 1 || task.DurationMinutes > maxTaskMinutes {
					return Roadmap{}, invalid("module %d day %d task %d duration %d is outside 1..%d", mi+1, di+1, ti+1, task.DurationMinutes, maxTaskMinutes)
				}
			}
		}
	}
	return r, nil
}

func isTaskType(s string) bool {
	for _, t := range TaskTypes {
		if s == t {
			return true
		}
	}
	return false
}

// Exercises flattens a validated roadmap into its 84 exercises rows.
// day_number = (module-1)*7 + day; content_json is the whole task object.
func (r Roadmap) Exercises() []Exercise {
	out := make([]Exercise, 0, Modules*DaysPerModule*TasksPerDay)
	for mi, m := range r.Modules {
		for di, d := range m.Days {
			for _, t := range d.Tasks {
				content, _ := json.Marshal(t) // a validated Task always marshals
				out = append(out, Exercise{
					DayNumber:   mi*DaysPerModule + di + 1,
					TaskType:    t.Type,
					ContentJSON: content,
				})
			}
		}
	}
	return out
}
```

- [ ] **Step 5: Run and confirm it passes**

```sh
go test ./internal/airouter/... -v
```
Expected: every `--- PASS`, including all `TestParseRoadmapRejects` subtests.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend/internal/airouter/prompt.go backend/internal/airouter/prompt_test.go backend/internal/airouter/roadmap.go backend/internal/airouter/roadmap_test.go && git commit -m "airouter: §6.1 system prompt, roadmap schema, strict 4x7x3 parser and 84-row flattener"
```

---

### Task 7: `.env.example`

**Files:**
- Modify: `backend/.env.example`

- [ ] **Step 1: Append**

```
# AI providers (backend spec §9). Each provider registers only when its key is
# set; with none set the API boots and AI routes answer 503. Base URLs and
# models are optional overrides.
#GEMINI_API_KEY=
#GEMINI_BASE_URL=https://generativelanguage.googleapis.com
#GEMINI_MODEL=gemini-2.5-flash
#OPENAI_API_KEY=
#OPENAI_BASE_URL=https://api.openai.com/v1
#OPENAI_MODEL=gpt-4o-mini
#DEEPSEEK_API_KEY=
#DEEPSEEK_BASE_URL=https://api.deepseek.com/v1
#DEEPSEEK_MODEL=deepseek-chat
```

- [ ] **Step 2: Commit**

```sh
cd .. && git add backend/.env.example && git commit -m "env: document the optional AI provider variables"
```

---

### Task 8: Build the router at boot

**Files:**
- Modify: `backend/cmd/api/main.go`

Onboarding (6) will consume the router; this slice only proves it boots and reports what is configured.

- [ ] **Step 1: Wire it**

Add `"os"` and `"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"` to the imports. After the `store.Migrate` block and before `r := gin.Default()`:
```go
	aiRouter := airouter.NewRouter(airouter.ConfigFromEnv(os.Getenv))
	if providers := aiRouter.Providers(); len(providers) == 0 {
		log.Printf("airouter: no provider API keys set; AI-backed routes will answer 503")
	} else {
		log.Printf("airouter: providers %v", providers)
	}
```

- [ ] **Step 2: Confirm build, vet and tests**

```sh
go build ./... && go vet ./... && go test ./... -count=1
grep -n 'airouter.NewRouter(airouter.ConfigFromEnv(os.Getenv))' cmd/api/main.go
```
Expected: clean; one grep hit.

- [ ] **Step 3: Commit**

```sh
cd .. && git add backend/cmd/api/main.go && git commit -m "airouter: build the router from the environment at boot"
```

---

### Task 9: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (the `**airouter**` bullet)

- [ ] **Step 1: Replace the bullet**

```
- **airouter** — LLM routing (1st-thinking §6.2 as pseudocode; §4 limiter; §6.1 prompt). `LLMProvider.GenerateContent(ctx, system, user) (string, error)`; `GeminiProvider` (real Generative Language API, `x-goog-api-key` header, `response_mime_type: application/json`, temperature 0.2) and `OpenAICompatibleProvider` (`/chat/completions`, `response_format: json_object`, temperature 0.2) shared by OpenAI (`gpt-4o-mini`) and DeepSeek (`deepseek-chat`); 30 s client timeout; base URLs and models overridable (`*_BASE_URL`, `*_MODEL`) so tests use `httptest`. `NewRouter(ConfigFromEnv(os.Getenv))` registers a provider **only when its `GEMINI_API_KEY` / `OPENAI_API_KEY` / `DEEPSEEK_API_KEY` is set** and never fails the boot — with none, `Route` returns `ErrNoProviders` (callers → 503). Strategies: `roadmap_generation`, `placement_test` → gemini; `exercise_generation` → deepseek; `essay_grading` → openai; unknown → gemini. `Route` tries the preferred provider then the others in the fixed `FallbackOrder` (gemini, openai, deepseek) on absence **or** error, stops on a done context, and returns `ErrAllProvidersFailed` joined with each provider's error. `RedisRateLimiter.Allow` = `INCR ratelimit:ai:{user_id}` + `EXPIRE NX 60s` in one pipeline (Redis 7), `ErrRateLimited` past 5/min (callers → 429). `RoadmapSystemPrompt` is §6.1 verbatim; `RoadmapSchema` is the JSON shape §6.1 leaves unstated — `{title, cefr_level, modules[4]{week,title,focus,days[7]{title,tasks[3]{type,title,duration_minutes,content}}}}`; `ParseRoadmap` rejects fences, preamble, trailing tokens, wrong counts, duplicate/unknown task types, empty titles, durations outside 1..30 and bad CEFR (missing duration → 10), wrapping `ErrInvalidRoadmap`; `Roadmap.Exercises()` yields the 84 `(day_number, task_type, content_json)` rows where **`content_json` is the task object**, so quests' `toTask` finds `title`/`duration_minutes`. No routes, no tables. Tests: `httptest` providers, fake providers for routing, `TestIntegrationRateLimiterAllowsFiveThenBlocks` gated on `TEST_REDIS_URL` (skips locally, must pass in CI).
```

- [ ] **Step 2: Verify and commit**

```sh
grep -n 'ErrNoProviders\|FallbackOrder\|ParseRoadmap' harness/CODEMAP.md
git add harness/CODEMAP.md && git commit -m "codemap: airouter — providers, strategies, fallback, limiter, roadmap schema"
```
Expected: three hits.

---

## Verification

Run from the worktree root.

```sh
cd backend && go build ./... && go vet ./...
# expect: no output

env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1
# expect: ok for every package incl. internal/airouter — no live service, no network

go test ./internal/airouter/... -run 'Route|Strategies|Providers' -v
# expect: --- PASS — strategy table, fixed fallback order, error join, ErrNoProviders, context stop

go test ./internal/airouter/... -run 'Gemini|OpenAI' -v
# expect: --- PASS — request shapes (header key, JSON mode, temperature 0.2), error cases

go test ./internal/airouter/... -run 'Config|NewRouter' -v
# expect: --- PASS — providers register only with a key

go test ./internal/airouter/... -run 'Roadmap|Exercises|Prompt' -v
# expect: --- PASS — every reject subtest, 84 rows with title/duration_minutes

grep -rn --include='*.go' --exclude='*_test.go' 'https://\|http://' internal/airouter/
# expect: exactly the three Default*BaseURL constants — nothing else hard-codes a host

grep -rn --include='*_test.go' 'googleapis.com\|api.openai.com\|api.deepseek.com' internal/airouter/ | grep -v 'Default'
# expect: no hits — tests never name a real host except through the Default* constants

grep -n 'store.AIRateLimitKey\|store.AIRateLimitTTL\|ExpireNX' internal/airouter/ratelimit.go
# expect: 3 hits

grep -n 'x-goog-api-key' internal/airouter/gemini.go && grep -c 'key=' internal/airouter/gemini.go
# expect: one hit; then 0 — the key never goes in the query string

grep -n '"response_mime_type": "application/json"' internal/airouter/gemini.go
grep -n '"type": "json_object"' internal/airouter/openai.go
# expect: one hit each

grep -c 'Output ONLY valid JSON' internal/airouter/prompt.go
# expect: 1

grep -rn 'Getenv' internal/airouter/ --include='*.go'
# expect: no hits — the package reads env only through the lookup func main passes

grep -c '^func TestIntegration' internal/airouter/ratelimit_test.go
# expect: 1

cd .. && python3 tools/harness/cli.py validate; echo exit=$?
# expect: exit=0

git log --oneline main..HEAD
# expect: 9 commits, one per task, each with the Co-Authored-By trailer

git status --short
# expect: clean
```

After pushing: `gh run list --branch <branch>` must show all three jobs green; `backend-integration` is where the limiter test runs for real.

## Notes and open questions

- **Fallback on error, not only on absence.** §6.2 falls back only when the preferred provider is unconfigured. Falling back on a failed call too is what makes an outage survivable, but it doubles cost in the failure case and can mask a broken key. If the owner wants the literal §6.2 behaviour, remove the `errs = append` loop's continuation.
- **The roadmap schema is this plan's invention.** §6.1 references "the requested schema" without stating it. Onboarding depends on `RoadmapSchema`/`ParseRoadmap`; the frontend's roadmap view (frontend spec §5) should be checked against it before that slice is planned.
- **`content` is free-form.** The validator does not check task `content` (word lists, passages) — its shape differs per task type and no spec defines it. The frontend will need a per-type contract eventually.
- **Model names churn.** `gemini-2.5-flash` and `gpt-4o-mini` are defaults, overridable by `*_MODEL`. Spec §2.1 mentions "Gemini Flash & Pro"; Pro is not wired (no task needs it yet).
- **`EXPIRE NX` needs Redis ≥ 7.0.** Compose and CI use `redis:7-alpine`. Railway's Redis plugin version should be confirmed before deploy (spec §9 step 1).
- **One rate-limit slot = one `Allow`.** Whether an assessment (two LLM calls, possibly a retry) consumes one slot or one per call is onboarding's decision (its plan: one per request).
- **No placement prompt here.** Grading needs the question bank, which is onboarding's; only the roadmap prompt is shared infrastructure.
- **`Route` logs fallbacks with `log.Printf`** — same as the rest of the backend; structured logging is a cross-cutting change not for this slice.
