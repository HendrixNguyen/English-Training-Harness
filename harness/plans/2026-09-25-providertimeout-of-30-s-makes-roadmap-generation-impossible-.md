---
idea: harness/ideas/_inbox/providertimeout-of-30-s-makes-roadmap-generation-impossible-.md
status: done
priority: high
merged: true
branch: harness/2026-09-25-high-providertimeout-of-30-s-makes-roadmap-generation-impossible-
worktree: .worktrees/providertimeout-of-30-s-makes-roadmap-generation-impossible-
pr: "https://github.com/HendrixNguyen/English-Training-Harness/pull/30"
---
# Per-task AI deadlines (180 s roadmap / 30 s others), the graded level survives a failed roadmap step, Gemini default model off the retired name, one retry on 503, per-call provider logs — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea (head):** `harness/ideas/_inbox/providertimeout-of-30-s-makes-roadmap-generation-impossible-.md`
**Also planned here (its frontmatter points at this plan):**
- `harness/ideas/_inbox/gemini-default-model-gemini-2-5-flash-is-retired-for-new-acc.md` → Task 4 (default model, retry on 429/502/503/504, upstream-error and token-usage log lines) and Task 5 (docs)

**Goal:** `POST /api/v1/onboarding/assessment` succeeds when the roadmap takes up to three minutes on a fallback provider, keeps the placement grade when the roadmap step fails so one re-submit finishes without re-grading, boots on a Gemini model that still exists, survives one 503 from a busy provider, and logs enough per provider call (elapsed, tokens, upstream status + message) that the next failure is diagnosable from `railway logs`.

**Why now (`priority: high`):** Live onboarding on 2026-09-25 (owner): Gemini 503 → fallback to `openai/gpt-4o-mini` → placement graded in 2 s → `POST …/assessment` **502 after 33.87 s** because `airouter.ProviderTimeout = 30 s` (`backend/internal/airouter/gemini.go:15`) is the `http.Client{Timeout}` of both drivers (`gemini.go:42`, `openai.go:39`) and the §6.1 roadmap (~4.5 k output tokens) measured 53 s on `gpt-4o-mini` and 78 s on `deepseek-chat`. The grade dies with the request (`onboarding/service.go:72-88` keeps `level` in a local and writes nothing on failure). Earlier the same day a fresh deploy 502'd in 1.08 s because `DefaultGeminiModel = "gemini-2.5-flash"` (`gemini.go:21`) is retired for new accounts (404) and no log line says so. This is the §5.1 happy path for every new learner. Root-cause detail is in both ideas' `## Evaluation`.

**Architecture:** The deadline moves out of the `http.Client` and into the context: a new `airouter/timeouts.go` owns `TaskTimeout(task)` (180 s for `roadmap_generation`, 30 s otherwise), `onboarding.routeJSON` sets it per `Route` call, and `Router.Route` applies it itself when a caller's context has no deadline (so no future caller regresses to unbounded); the drivers' `http.Client` gets no `Timeout`. A deadline hit becomes `onboarding.ErrAITimeout` → **504 `ai_timeout`** (today `Route`'s `ctx.Err()` reaches the handler as 500 `internal_error`). The graded level is staged as one extra field (`_level`) of the existing `quiz:placement:{user_id}` hash (same 2 h TTL, deleted on success), and a re-submit with identical answers skips the placement call — the §6.1 DTO is unchanged, no new endpoint, and the "nothing durable is written before the roadmap exists" invariant that `TestAssessFailsWithoutWritingWhenTheRoadmapIsBadTwice` pins stays true (see the idea's Evaluation for why not `users.cefr_current` or a 202). `postJSON` gains one retry after a 2 s backoff on 429/502/503/504 (bounded by the same context), a 200-char truncation of upstream error bodies, and the drivers log `airouter: <provider>(<model>) task=<task> ok <secs>s tokens prompt=<n> completion=<n>`; the router logs each provider failure with elapsed time and the upstream error. `DefaultGeminiModel` becomes `gemini-3.8-flash`. Frontend: one line of waiting copy under the last quiz button (there is no client fetch timeout to raise — `frontend/utils/apiClient.ts` sets none).

**Tech stack:** Go 1.2x, `net/http` + `context`, `httptest`, `go-redis/v9` (`HSET`/`HGETALL`/`EXPIRE` in a `TxPipeline`), Gin; Nuxt 3 / Vitest for the one frontend line. No new dependencies.

**Base branch and conflict note:** the executor bases its worktree on freshly fetched `origin/main` (harness-execute skill, step 5). Two things may or may not be on that base:
- `harness/2026-09-24-medium-geminiprovider-drops-every-response-part-after-the-first-so-` (done, unmerged today) rewrites `GeminiProvider.GenerateContent`'s response struct (adds `FinishReason`, `PromptFeedback`, joins parts, `maxOutputTokens`). If `grep -n finishReason backend/internal/airouter/gemini.go` hits, that branch has merged: **keep every one of its fields and behaviours** and only add the `UsageMetadata` field, the timing and the log call from Task 4. If it does not hit, apply Task 4 to the struct as it is on main. Never merge or cherry-pick that branch.
- `deploy/README.md` exists only on `harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook-` today (`git ls-tree origin/main deploy` is empty). Task 5's runbook row is conditional on `test -f deploy/README.md` and is **skipped, not improvised**, when absent — the evaluator re-files it if still missing after that branch merges.

Run every command from the worktree root unless a step says `backend/` or `frontend/`; `rg` is not installed — use `grep -n`; `timeout` is not installed — use the tools' own flags (`go test -timeout`, `curl --max-time`). `make check` runs `gofmt`, `go vet` and `go test ./... -count=1 -race`; keep it green after every task.

---

## File structure

| File | Change |
| --- | --- |
| `backend/internal/airouter/timeouts.go` | **New**: `RoadmapTimeout`, `DefaultTaskTimeout`, `TaskTimeout(task)`, `ensureDeadline`, `withTask`/`taskFrom` (task label for log lines) |
| `backend/internal/airouter/timeouts_test.go` | **New**: budget table, `budgeted` fake, Route deadline tests, "context governs, not a client timeout" test for both drivers |
| `backend/internal/airouter/router.go` | `Route` applies `ensureDeadline` + `withTask`, logs each provider failure with elapsed time |
| `backend/internal/airouter/gemini.go` | drop `ProviderTimeout`; client without `Timeout`; `DefaultGeminiModel = "gemini-3.8-flash"`; `usageMetadata` parsed; success log line; `postJSON` split into `postJSON` (retry loop) + `doJSON`; `errorBodyChars = 200`; `retryBackoff` |
| `backend/internal/airouter/openai.go` | client without `Timeout`; `usage` parsed; success log line |
| `backend/internal/airouter/gemini_test.go`, `openai_test.go` | defaults tests updated (no client timeout, new model); retry / no-retry / give-up / backoff-respects-ctx / log-line tests |
| `backend/internal/onboarding/service.go` | `ErrAITimeout`; `route()` sets the per-task deadline and maps its own deadline; `Assess` reuses a staged level, stages it after grading |
| `backend/internal/onboarding/quiz.go` | `QuizStore.StageLevel` / `StagedLevel`; Redis implementation |
| `backend/internal/onboarding/handler.go` | `ErrAITimeout` → 504 `ai_timeout` |
| `backend/internal/onboarding/fakes_test.go`, `service_test.go`, `handler_test.go` | fake quiz level field; scripted provider records budgets and can time out; new tests |
| `backend/internal/onboarding/quiz_integration_test.go` | **New**: `TEST_REDIS_URL`-gated test of `StageLevel`/`StagedLevel`/`Clear` |
| `backend/cmd/api/server.go` | comment on `newServer` re-worded (no `ProviderTimeout`) |
| `backend/.env.example` | `GEMINI_MODEL` default + why |
| `CLAUDE.md` | router paragraph: deadlines, model, `*_MODEL` overrides |
| `project-base/1st-thinking-architecture-doc.md` | one-paragraph §6.2 addendum before §7 |
| `deploy/README.md` | **only if present** — `GEMINI_MODEL` row in the env table |
| `frontend/pages/onboarding.vue`, `frontend/tests/unit/onboardingPage.test.ts` | waiting copy while the assessment POST is pending + one test |
| `harness/CODEMAP.md` | **airouter**, **onboarding**, **cmd/api** bullets |

---

### Task 1: Per-task deadlines in `airouter`; the context governs, not the `http.Client`

**Files:**
- Create: `backend/internal/airouter/timeouts.go`
- Create: `backend/internal/airouter/timeouts_test.go`
- Modify: `backend/internal/airouter/router.go` (`Route`)
- Modify: `backend/internal/airouter/gemini.go:14-15,41-43`, `backend/internal/airouter/openai.go:38-40`
- Modify: `backend/internal/airouter/gemini_test.go:84-95`, `backend/internal/airouter/openai_test.go:77-81`

- [ ] **Step 1: Write the failing tests.** Create `backend/internal/airouter/timeouts_test.go`:

```go
package airouter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// budgeted is a provider whose answer would take `needs`: it succeeds only
// when the context it is given still has at least that long, and never
// sleeps — so a "31-second model" is tested in microseconds.
type budgeted struct {
	needs time.Duration
	out   string
	calls int
}

func (b *budgeted) GenerateContent(ctx context.Context, _, _ string) (string, error) {
	b.calls++
	dl, ok := ctx.Deadline()
	if !ok {
		return "", errors.New("budgeted: the context has no deadline")
	}
	if left := time.Until(dl); left < b.needs {
		return "", fmt.Errorf("budgeted: %s left, need %s: %w", left.Round(time.Second), b.needs, context.DeadlineExceeded)
	}
	return b.out, nil
}

func TestTaskTimeoutsMatchTheMeasuredProviders(t *testing.T) {
	// 2026-09-25 measurements of the §6.1 roadmap: gpt-4o-mini 53 s, deepseek-chat 78 s.
	if TaskTimeout(TaskRoadmapGen) < 120*time.Second || TaskTimeout(TaskRoadmapGen) != RoadmapTimeout {
		t.Errorf("TaskTimeout(roadmap) = %s, want RoadmapTimeout ≥ 120 s", TaskTimeout(TaskRoadmapGen))
	}
	for _, task := range []TaskType{TaskPlacementTest, TaskExerciseGen, TaskEssayGrading, TaskType("something_new")} {
		if TaskTimeout(task) != 30*time.Second || TaskTimeout(task) != DefaultTaskTimeout {
			t.Errorf("TaskTimeout(%s) = %s, want 30 s (§6.2)", task, TaskTimeout(task))
		}
	}
}

func TestRouteGivesTheRoadmapMoreThan30SecondsAndOtherTasksExactly30(t *testing.T) {
	slow := &budgeted{needs: 31 * time.Second, out: "{}"}
	r := NewRouterWithProviders(map[ProviderType]LLMProvider{ProviderGemini: slow})

	if out, err := r.Route(context.Background(), TaskRoadmapGen, "s", "u"); err != nil || out != "{}" {
		t.Fatalf("roadmap on a 31 s provider: %q, %v; want success", out, err)
	}
	_, err := r.Route(context.Background(), TaskPlacementTest, "s", "u")
	if !errors.Is(err, ErrAllProvidersFailed) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("placement on a 31 s provider: %v; want ErrAllProvidersFailed wrapping DeadlineExceeded", err)
	}
}

func TestRouteKeepsACallerDeadlineInsteadOfWideningIt(t *testing.T) {
	slow := &budgeted{needs: 31 * time.Second, out: "{}"}
	r := NewRouterWithProviders(map[ProviderType]LLMProvider{ProviderGemini: slow})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := r.Route(ctx, TaskRoadmapGen, "s", "u"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a caller's 10 s deadline was widened to the task default: %v", err)
	}
}

func TestProvidersObeyTheContextDeadlineNotAClientTimeout(t *testing.T) {
	// One body that satisfies both parsers.
	const late = `{"candidates":[{"content":{"parts":[{"text":"late"}]}}],"choices":[{"message":{"content":"late"}}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(300 * time.Millisecond):
			_, _ = w.Write([]byte(late))
		}
	}))
	defer srv.Close()
	// nil client → the production default, which must carry no Timeout.
	providers := map[string]LLMProvider{
		"gemini": NewGeminiProvider("k", srv.URL, "m", nil),
		"openai": NewOpenAICompatibleProvider(srv.URL, "k", "m", nil),
	}
	for name, p := range providers {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			started := time.Now()
			if _, err := p.GenerateContent(ctx, "s", "u"); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("err = %v, want context.DeadlineExceeded", err)
			}
			if took := time.Since(started); took > 250*time.Millisecond {
				t.Errorf("took %s; the context deadline did not cut the call", took)
			}
			ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel2()
			if out, err := p.GenerateContent(ctx2, "s", "u"); err != nil || out != "late" {
				t.Fatalf("with a 2 s deadline: %q, %v; want the answer", out, err)
			}
		})
	}
}
```

In `gemini_test.go` replace `TestGeminiDefaultsToTheRealEndpoint`'s timeout check (lines 92-94) with:

```go
	if p.client.Timeout != 0 {
		t.Errorf("client.Timeout = %v, want 0: the per-task deadline travels in the context (timeouts.go)", p.client.Timeout)
	}
```

In `openai_test.go` `TestOpenAICompatibleDefaults` (line 79) replace `p.client.Timeout != ProviderTimeout` with `p.client.Timeout != 0`.

- [ ] **Step 2: Run them to see them fail.** `cd backend && go test ./internal/airouter/ -run 'TestTaskTimeouts|TestRouteGives|TestRouteKeeps|TestProvidersObey|Defaults' -count=1`
Expected: compile error `undefined: TaskTimeout` / `RoadmapTimeout` / `DefaultTaskTimeout`.

- [ ] **Step 3: Create `backend/internal/airouter/timeouts.go`.**

```go
package airouter

import (
	"context"
	"time"
)

// Per-task deadlines. §6.2's pseudocode gave every call one 30 s http.Client
// timeout; a §6.1 roadmap is ~4.5 k output tokens and measured 53 s on
// gpt-4o-mini and 78 s on deepseek-chat (2026-09-25), so only Gemini Flash
// ever finished inside it. The budget now travels in the context: callers set
// it per Route call (onboarding.routeJSON), Route adds it when a caller did
// not, and the drivers' http.Client carries no Timeout of its own.
const (
	// RoadmapTimeout bounds one Route call for TaskRoadmapGen, fallbacks
	// included: room for one fast-failing provider plus one slow success.
	RoadmapTimeout = 180 * time.Second
	// DefaultTaskTimeout is §6.2's 30 s for every other task.
	DefaultTaskTimeout = 30 * time.Second
)

// TaskTimeout is the budget for one Route call of task.
func TaskTimeout(task TaskType) time.Duration {
	if task == TaskRoadmapGen {
		return RoadmapTimeout
	}
	return DefaultTaskTimeout
}

// ensureDeadline returns ctx as is when it already has a deadline, else a
// child bounded by TaskTimeout(task). Route never widens a caller's budget.
func ensureDeadline(ctx context.Context, task TaskType) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, TaskTimeout(task))
}

// The task rides along in the context so a driver's log line can name it
// without widening the LLMProvider interface.
type taskKey struct{}

func withTask(ctx context.Context, task TaskType) context.Context {
	return context.WithValue(ctx, taskKey{}, task)
}

func taskFrom(ctx context.Context) TaskType {
	task, _ := ctx.Value(taskKey{}).(TaskType)
	return task
}
```

- [ ] **Step 4: `Route` applies it.** In `router.go`, after the `preferred` lookup (line 63) and before building `order`, add:

```go
	ctx, cancel := ensureDeadline(ctx, task)
	defer cancel()
	ctx = withTask(ctx, task)
```

Update the doc comment above `Route`: append `Every attempt runs under the caller's deadline, or TaskTimeout(task) when the caller set none.`

- [ ] **Step 5: Drivers drop the client timeout.** In `gemini.go` delete lines 14-15 (`ProviderTimeout`) and change the constructor default (line 42) to:

```go
	if client == nil {
		client = &http.Client{} // no Timeout: the per-task deadline is in the context (timeouts.go)
	}
```

Same in `openai.go` line 39. Remove the now-unused `"time"` import from `gemini.go` **only if** nothing else in the file uses it (Task 4 adds a use; `go vet` will tell you).

- [ ] **Step 6: Run the package.** `cd backend && go test ./internal/airouter/ -count=1 -race`
Expected: PASS, including the four new tests (the deadline test takes ~0.5 s).

- [ ] **Step 7: Nothing else names `ProviderTimeout`.** `grep -rn ProviderTimeout backend/ CLAUDE.md harness/CODEMAP.md` → only `backend/cmd/api/server.go:28` (a comment; Task 5 rewords it) and `harness/CODEMAP.md` (Task 7).

- [ ] **Step 8: Commit.**
```bash
git add backend/internal/airouter/timeouts.go backend/internal/airouter/timeouts_test.go backend/internal/airouter/router.go backend/internal/airouter/gemini.go backend/internal/airouter/openai.go backend/internal/airouter/gemini_test.go backend/internal/airouter/openai_test.go
git commit -m "airouter: per-task deadlines in the context (180 s roadmap, 30 s others); drivers carry no client timeout"
```

---

### Task 2: `onboarding` sets the deadline per call and reports a hit as 504 `ai_timeout`

**Files:**
- Modify: `backend/internal/onboarding/service.go:111-132` (`routeJSON`), new `route` helper, `ErrAITimeout`
- Modify: `backend/internal/onboarding/handler.go:42-60`
- Modify: `backend/internal/onboarding/fakes_test.go:84-115` (scripted provider records budgets, can time out)
- Modify: `backend/internal/onboarding/service_test.go`, `backend/internal/onboarding/handler_test.go:74-95`

- [ ] **Step 1: Teach the scripted provider to report its budget and to time out.** In `fakes_test.go` change `scriptedProvider`:

```go
type scriptedProvider struct {
	replies map[airouter.TaskType][]string
	calls   map[airouter.TaskType]int
	prompts map[airouter.TaskType][]string        // system|user per call
	budgets map[airouter.TaskType][]time.Duration // time left on ctx at each call
	timeout map[airouter.TaskType]bool            // answer as a model that outlives its deadline
}

func newScripted() *scriptedProvider {
	return &scriptedProvider{replies: map[airouter.TaskType][]string{}, calls: map[airouter.TaskType]int{},
		prompts: map[airouter.TaskType][]string{}, budgets: map[airouter.TaskType][]time.Duration{}, timeout: map[airouter.TaskType]bool{}}
}

func (p *scriptedProvider) GenerateContent(ctx context.Context, system, user string) (string, error) {
	task := airouter.TaskRoadmapGen
	if system == PlacementSystemPrompt {
		task = airouter.TaskPlacementTest
	}
	p.prompts[task] = append(p.prompts[task], system+"|"+user)
	if dl, ok := ctx.Deadline(); ok {
		p.budgets[task] = append(p.budgets[task], time.Until(dl))
	}
	i := p.calls[task]
	p.calls[task]++
	if p.timeout[task] {
		return "", fmt.Errorf("scripted: %s outlived its deadline: %w", task, context.DeadlineExceeded)
	}
	if i >= len(p.replies[task]) {
		return "", fmt.Errorf("scripted: no reply %d for %s", i, task)
	}
	return p.replies[task][i], nil
}
```

- [ ] **Step 2: Write the failing tests.** Append to `service_test.go`:

```go
func TestAssessGivesEachAICallItsOwnDeadline(t *testing.T) {
	h := newHarness(t)
	if _, err := h.svc.Assess(ctx, "u1", validRequest()); err != nil {
		t.Fatalf("Assess: %v", err)
	}
	within := func(got, want time.Duration) bool { return got > want-2*time.Second && got <= want }
	if b := h.ai.budgets[airouter.TaskPlacementTest]; len(b) != 1 || !within(b[0], airouter.DefaultTaskTimeout) {
		t.Errorf("placement budget = %v, want ≈ %s", b, airouter.DefaultTaskTimeout)
	}
	if b := h.ai.budgets[airouter.TaskRoadmapGen]; len(b) != 1 || !within(b[0], airouter.RoadmapTimeout) {
		t.Errorf("roadmap budget = %v, want ≈ %s (not the 30 s that 502'd on 2026-09-25)", b, airouter.RoadmapTimeout)
	}
}

func TestAssessReportsADeadlineHitAsAITimeoutWithoutWriting(t *testing.T) {
	h := newHarness(t)
	h.ai.timeout[airouter.TaskRoadmapGen] = true

	_, err := h.svc.Assess(ctx, "u1", validRequest())
	if !errors.Is(err, ErrAITimeout) {
		t.Fatalf("err = %v, want ErrAITimeout", err)
	}
	if len(h.repo.saved) != 0 {
		t.Error("a timed-out roadmap wrote an assessment")
	}
}

func TestAssessPassesACallerCancellationThroughUnchanged(t *testing.T) {
	h := newHarness(t)
	gone, cancel := context.WithCancel(ctx)
	cancel() // the client disconnected
	_, err := h.svc.Assess(gone, "u1", validRequest())
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrAITimeout) {
		t.Fatalf("err = %v, want context.Canceled and not ErrAITimeout", err)
	}
}
```

Add a row to `TestAssessmentErrorMapping` in `handler_test.go`:

```go
		{"ai timed out", func(h *harness) { h.ai.timeout[airouter.TaskRoadmapGen] = true }, spec61Request, 504, "ai_timeout"},
```

- [ ] **Step 3: Run to see them fail.** `cd backend && go test ./internal/onboarding/ -run 'Deadline|AITimeout|Cancellation|ErrorMapping' -count=1`
Expected: `undefined: ErrAITimeout`.

- [ ] **Step 4: Implement.** In `service.go` add next to `ErrInvalidRequest`:

```go
// ErrAITimeout means an AI call did not finish inside airouter.TaskTimeout.
// The handler maps it to 504 ai_timeout; a caller's own cancellation is
// passed through untouched.
var ErrAITimeout = errors.New("onboarding: AI call timed out")
```

Change `routeJSON`'s `s.ai.Route(ctx, task, system, user)` call to `s.route(ctx, task, system, user)` and add:

```go
// route runs one Route call under the task's own budget (airouter.TaskTimeout:
// 180 s for the roadmap, 30 s otherwise) and names a deadline of ours
// ErrAITimeout. If the parent context is done the client is gone, and that
// error is returned as is.
func (s *Service) route(ctx context.Context, task airouter.TaskType, system, user string) (string, error) {
	budget := airouter.TaskTimeout(task)
	actx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	raw, err := s.ai.Route(actx, task, system, user)
	if err != nil && ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
		log.Printf("onboarding: %s did not finish inside %s", task, budget)
		return "", fmt.Errorf("%w: %s after %s", ErrAITimeout, task, budget)
	}
	return raw, err
}
```

Update the `Assess` doc comment's "both AI calls before any write" sentence to mention the per-task budgets. In `handler.go` add, **before** the `ErrAllProvidersFailed` case:

```go
		case errors.Is(err, ErrAITimeout):
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "ai_timeout"})
```

- [ ] **Step 5: Run the package.** `cd backend && go test ./internal/onboarding/ -count=1 -race` → PASS.

- [ ] **Step 6: Commit.**
```bash
git add backend/internal/onboarding/service.go backend/internal/onboarding/handler.go backend/internal/onboarding/fakes_test.go backend/internal/onboarding/service_test.go backend/internal/onboarding/handler_test.go
git commit -m "onboarding: each AI call runs under its task deadline; a deadline hit is 504 ai_timeout, not 500"
```

---

### Task 3: The graded level survives a failed roadmap step

**Files:**
- Modify: `backend/internal/onboarding/quiz.go` (`QuizStore` + Redis implementation)
- Modify: `backend/internal/onboarding/service.go:62-88` (`Assess`)
- Modify: `backend/internal/onboarding/fakes_test.go:43-61` (`fakeQuiz`)
- Modify: `backend/internal/onboarding/service_test.go`
- Create: `backend/internal/onboarding/quiz_integration_test.go`

- [ ] **Step 1: Fake quiz store learns the level.** In `fakes_test.go`:

```go
type fakeQuiz struct {
	staged  map[string][]Answer
	level   map[string]string
	lastTTL time.Duration
	cleared int
}

func newFakeQuiz() *fakeQuiz { return &fakeQuiz{staged: map[string][]Answer{}, level: map[string]string{}} }

func (f *fakeQuiz) StageAnswers(_ context.Context, userID string, answers []Answer, ttl time.Duration) error {
	f.staged[userID] = answers
	delete(f.level, userID) // DEL + HSET in the real store
	f.lastTTL = ttl
	return nil
}

func (f *fakeQuiz) StageLevel(_ context.Context, userID, level string, _ time.Duration) error {
	f.level[userID] = level
	return nil
}

func (f *fakeQuiz) StagedLevel(_ context.Context, userID string, answers []Answer) (string, error) {
	level, staged := f.level[userID], f.staged[userID]
	if level == "" || len(staged) != len(answers) {
		return "", nil
	}
	for i := range answers {
		if staged[i] != answers[i] {
			return "", nil
		}
	}
	return level, nil
}

func (f *fakeQuiz) Clear(_ context.Context, userID string) error {
	f.cleared++
	delete(f.staged, userID)
	delete(f.level, userID)
	return nil
}
```

- [ ] **Step 2: Write the failing tests.** Append to `service_test.go`:

```go
func TestAssessKeepsTheGradeWhenTheRoadmapFailsAndSkipsGradingOnTheRetry(t *testing.T) {
	h := newHarness(t)
	h.ai.replies[airouter.TaskRoadmapGen] = nil // every roadmap call fails → ErrAllProvidersFailed

	if _, err := h.svc.Assess(ctx, "u1", validRequest()); !errors.Is(err, airouter.ErrAllProvidersFailed) {
		t.Fatalf("first Assess: %v, want ErrAllProvidersFailed", err)
	}
	if h.quiz.level["u1"] != "B1" || len(h.repo.saved) != 0 || h.quiz.cleared != 0 {
		t.Fatalf("after the failed roadmap: level=%q saved=%d cleared=%d; want the level staged, nothing written, hash kept", h.quiz.level["u1"], len(h.repo.saved), h.quiz.cleared)
	}

	h.ai.replies[airouter.TaskRoadmapGen] = []string{fixtureRoadmapJSON(t)}
	h.ai.calls[airouter.TaskRoadmapGen] = 0
	out, err := h.svc.Assess(ctx, "u1", validRequest())
	if err != nil || !out.Created || out.AssessedLevel != "B1" {
		t.Fatalf("retry: %+v, %v; want a created roadmap at the staged level", out, err)
	}
	if h.ai.calls[airouter.TaskPlacementTest] != 1 {
		t.Errorf("placement graded %d times, want 1 — the retry must reuse the staged level", h.ai.calls[airouter.TaskPlacementTest])
	}
	if h.limiter.calls != 2 {
		t.Errorf("limiter calls = %d, want 2 (one slot per assessment call, retry included)", h.limiter.calls)
	}
	if len(h.repo.saved) != 1 || h.repo.saved[0].CEFRLevel != "B1" || h.quiz.cleared != 1 || h.quiz.level["u1"] != "" {
		t.Errorf("after the retry: saved=%d cleared=%d level=%q", len(h.repo.saved), h.quiz.cleared, h.quiz.level["u1"])
	}
}

func TestAssessGradesAgainWhenTheRetryChangesAnAnswer(t *testing.T) {
	h := newHarness(t)
	h.ai.replies[airouter.TaskRoadmapGen] = nil
	if _, err := h.svc.Assess(ctx, "u1", validRequest()); err == nil {
		t.Fatal("first Assess succeeded; the fixture should fail at the roadmap")
	}

	h.ai.replies[airouter.TaskPlacementTest] = []string{`{"cefr_level":"B1"}`, `{"cefr_level":"A2"}`}
	h.ai.replies[airouter.TaskRoadmapGen] = []string{fixtureRoadmapJSON(t)}
	h.ai.calls[airouter.TaskRoadmapGen] = 0
	req := validRequest()
	req.Answers[0].SelectedOption = "A" // q1 has options A-D; B is correct
	out, err := h.svc.Assess(ctx, "u1", req)
	if err != nil || out.AssessedLevel != "A2" {
		t.Fatalf("retry with changed answers: %+v, %v; want a fresh grade", out, err)
	}
	if h.ai.calls[airouter.TaskPlacementTest] != 2 {
		t.Errorf("placement graded %d times, want 2", h.ai.calls[airouter.TaskPlacementTest])
	}
}
```

Also extend the existing `TestAssessHappyPathGradesGeneratesAndPersistsOnce`: after its success assertions add `if h.quiz.level["u1"] != "" { t.Error("the staged level must go with the hash on success") }`.

- [ ] **Step 3: Run to see them fail.** `cd backend && go test ./internal/onboarding/ -run 'KeepsTheGrade|GradesAgain' -count=1`
Expected: it compiles (the fake's extra methods are harmless until the interface names them) and fails at `after the failed roadmap: level="" …` — `Assess` never stages the level yet.

- [ ] **Step 4: The store.** In `quiz.go` extend the interface and implementation:

```go
// QuizStore is the §4 quiz:placement:{user_id} Hash — "transient storage for
// active placement test answers before grading". StageAnswers writes
// question_id → selected_option and sets the TTL; StageLevel adds the graded
// level under levelField (same TTL) so a re-submit after a failed roadmap step
// is not graded again; StagedLevel returns it only for exactly the staged
// answers; Clear removes the hash after a successful assessment. A failed
// grading leaves the answers inspectable for the TTL.
type QuizStore interface {
	StageAnswers(ctx context.Context, userID string, answers []Answer, ttl time.Duration) error
	StageLevel(ctx context.Context, userID, level string, ttl time.Duration) error
	// StagedLevel is "" when nothing is staged or the answers differ.
	StagedLevel(ctx context.Context, userID string, answers []Answer) (string, error)
	Clear(ctx context.Context, userID string) error
}

// levelField holds the graded level beside the answers. Bank ids are q1..q10
// and validate() rejects anything else, so it can never collide with one.
const levelField = "_level"
```

```go
func (s *RedisQuizStore) StageLevel(ctx context.Context, userID, level string, ttl time.Duration) error {
	key := store.PlacementQuizKey(userID)
	pipe := s.Client.TxPipeline()
	pipe.HSet(ctx, key, levelField, level)
	pipe.Expire(ctx, key, ttl) // HSET on an expired key would otherwise create one without a TTL
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("onboarding: staging level: %w", err)
	}
	return nil
}

func (s *RedisQuizStore) StagedLevel(ctx context.Context, userID string, answers []Answer) (string, error) {
	fields, err := s.Client.HGetAll(ctx, store.PlacementQuizKey(userID)).Result() // empty map for a missing key
	if err != nil {
		return "", fmt.Errorf("onboarding: reading staged level: %w", err)
	}
	level := fields[levelField]
	if level == "" || len(fields)-1 != len(answers) {
		return "", nil
	}
	for _, a := range answers {
		if fields[a.QuestionID] != a.SelectedOption {
			return "", nil
		}
	}
	return level, nil
}
```

- [ ] **Step 5: `Assess` reuses it.** Replace `service.go` lines 68-79 (stage + grade) with:

```go
	// A re-submit with the same answers after a failed roadmap step reuses
	// the level graded then (it sits in the quiz hash for the TTL).
	level, err := s.quiz.StagedLevel(ctx, userID, req.Answers)
	if err != nil {
		return AssessmentResult{}, err
	}
	if level == "" {
		if err := s.quiz.StageAnswers(ctx, userID, req.Answers, store.PlacementQuizTTL); err != nil {
			return AssessmentResult{}, err
		}
		if err := s.routeJSON(ctx, airouter.TaskPlacementTest, PlacementSystemPrompt, PlacementUserPrompt(req.Answers), func(raw string) error {
			lvl, err := ParsePlacement(raw)
			level = lvl
			return err
		}); err != nil {
			return AssessmentResult{}, err
		}
		if err := s.quiz.StageLevel(ctx, userID, level, store.PlacementQuizTTL); err != nil {
			log.Printf("onboarding: staging level for %s: %v", userID, err) // a retry grades again
		}
	} else {
		log.Printf("onboarding: reusing the staged level %s for %s", level, userID)
	}
```

Keep `var level string` semantics (declare `var level string` before if you prefer; the `:=` above declares it). The limiter call stays where it is (one slot per assessment call — the retry costs a slot, never a second grading). Update the `Assess` doc comment accordingly.

- [ ] **Step 6: Run the package.** `cd backend && go test ./internal/onboarding/ -count=1 -race` → PASS.

- [ ] **Step 7: Integration test for the Redis store.** Create `backend/internal/onboarding/quiz_integration_test.go` following `airouter/ratelimit_test.go`'s gate:

```go
package onboarding

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// Gated on TEST_REDIS_URL, never the production REDIS_URL (spec §9). CI exports
// TEST_* and fails on --- SKIP; run with -p 1 (make test-integration).
func TestIntegrationQuizStoreStagesAndReusesTheGradedLevel(t *testing.T) {
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
	const user = "onboarding-integration-quiz-user"
	qs := NewRedisQuizStore(rdb)
	t.Cleanup(func() { _ = qs.Clear(ctx, user) })

	answers := []Answer{{QuestionID: "q1", SelectedOption: "B"}, {QuestionID: "q2", SelectedOption: "C"}}
	if err := qs.StageAnswers(ctx, user, answers, time.Minute); err != nil {
		t.Fatal(err)
	}
	if lvl, err := qs.StagedLevel(ctx, user, answers); err != nil || lvl != "" {
		t.Fatalf("before StageLevel: %q, %v; want \"\"", lvl, err)
	}
	if err := qs.StageLevel(ctx, user, "B1", time.Minute); err != nil {
		t.Fatal(err)
	}
	if lvl, err := qs.StagedLevel(ctx, user, answers); err != nil || lvl != "B1" {
		t.Fatalf("same answers: %q, %v; want B1", lvl, err)
	}
	changed := []Answer{{QuestionID: "q1", SelectedOption: "A"}, {QuestionID: "q2", SelectedOption: "C"}}
	if lvl, _ := qs.StagedLevel(ctx, user, changed); lvl != "" {
		t.Errorf("changed answers: %q, want \"\"", lvl)
	}
	if lvl, _ := qs.StagedLevel(ctx, user, answers[:1]); lvl != "" {
		t.Errorf("fewer answers: %q, want \"\"", lvl)
	}
	if ttl := rdb.Client.TTL(ctx, store.PlacementQuizKey(user)).Val(); ttl <= 0 || ttl > time.Minute {
		t.Errorf("TTL after StageLevel = %s, want (0, 1m]", ttl)
	}
	if err := qs.StageAnswers(ctx, user, changed, time.Minute); err != nil {
		t.Fatal(err)
	}
	if lvl, _ := qs.StagedLevel(ctx, user, changed); lvl != "" {
		t.Errorf("re-staging answers must drop the old level, got %q", lvl)
	}
	if err := qs.Clear(ctx, user); err != nil {
		t.Fatal(err)
	}
	if n := rdb.Client.Exists(ctx, store.PlacementQuizKey(user)).Val(); n != 0 {
		t.Errorf("key survives Clear")
	}
}
```

Check `store.Redis` exposes `Client` the way `ratelimit_test.go` / `quiz.go:27` use it; adjust the field name if it differs. Run it locally if Redis is up (`COMPOSE_PROJECT_NAME=<slug> make up`, `TEST_REDIS_URL=redis://127.0.0.1:<port>/0 go test ./internal/onboarding/ -run Integration -count=1 -p 1`); otherwise it skips here and **must pass in CI's `backend-integration`** (which fails on `--- SKIP`).

- [ ] **Step 8: Commit.**
```bash
git add backend/internal/onboarding/quiz.go backend/internal/onboarding/service.go backend/internal/onboarding/fakes_test.go backend/internal/onboarding/service_test.go backend/internal/onboarding/quiz_integration_test.go
git commit -m "onboarding: the graded level is staged in the quiz hash, so a retry after a failed roadmap step is not graded again"
```

---

### Task 4: Provider logs (elapsed, tokens, upstream status + message), one retry on 429/502/503/504, Gemini default model `gemini-3.8-flash`

**Files:**
- Modify: `backend/internal/airouter/gemini.go` (constants, `GenerateContent`, `postJSON` → `postJSON` + `doJSON`, `truncate`)
- Modify: `backend/internal/airouter/openai.go` (`GenerateContent`)
- Modify: `backend/internal/airouter/router.go:84-88` (failure log)
- Modify: `backend/internal/airouter/gemini_test.go`, `backend/internal/airouter/openai_test.go`

- [ ] **Step 1: Write the failing tests.** Add to `gemini_test.go` (imports: add `"bytes"`, `"log"`, `"sync/atomic"`, `"time"`):

```go
// captureLog routes the standard logger into a buffer for one test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

func noBackoff(t *testing.T) {
	t.Helper()
	prev := retryBackoff
	retryBackoff = 0
	t.Cleanup(func() { retryBackoff = prev })
}

const geminiOK = `{"candidates":[{"content":{"parts":[{"text":"ok"}]}}],"usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":34,"totalTokenCount":46}}`

func TestGeminiRetriesOnceOn503AndLogsElapsedAndTokens(t *testing.T) {
	noBackoff(t)
	logs := captureLog(t)
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"code":503,"message":"This model is currently experiencing high demand. Spikes in demand are usually temporary. Please try again later.","status":"UNAVAILABLE"}}`))
			return
		}
		_, _ = w.Write([]byte(geminiOK))
	}))
	defer srv.Close()

	p := NewGeminiProvider("k-123", srv.URL, "gemini-3.8-flash", srv.Client())
	out, err := p.GenerateContent(withTask(context.Background(), TaskPlacementTest), "s", "u")
	if err != nil || out != "ok" {
		t.Fatalf("GenerateContent = %q, %v; want ok after one retry", out, err)
	}
	if calls != 2 {
		t.Errorf("upstream calls = %d, want 2", calls)
	}
	got := logs.String()
	for _, want := range []string{"retrying once", "status 503", "high demand", "gemini(gemini-3.8-flash) task=placement_test ok ", "tokens prompt=12 completion=34"} {
		if !strings.Contains(got, want) {
			t.Errorf("log missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "k-123") {
		t.Errorf("the API key reached the log:\n%s", got)
	}
}

func TestGeminiDoesNotRetryA404AndReportsGooglesMessageTruncated(t *testing.T) {
	noBackoff(t)
	var calls int32
	msg := "This model models/gemini-2.5-flash is no longer available to new users. Please update your code to use models/gemini-3.8-flash for the latest features and improvements. We recommend you to use the Interactions API " + strings.Repeat("x", 300)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":404,"message":"` + msg + `","status":"NOT_FOUND"}}`))
	}))
	defer srv.Close()

	p := NewGeminiProvider("k", srv.URL, "gemini-2.5-flash", srv.Client())
	_, err := p.GenerateContent(context.Background(), "s", "u")
	if err == nil || !strings.Contains(err.Error(), "status 404") || !strings.Contains(err.Error(), "no longer available to new users") {
		t.Fatalf("err = %v, want status 404 with Google's message", err)
	}
	if strings.Contains(err.Error(), "xxxxxxxxxx") || !strings.Contains(err.Error(), "…") {
		t.Errorf("error body not cut at %d chars: %v", errorBodyChars, err)
	}
	if calls != 1 {
		t.Errorf("upstream calls = %d, want 1 (404 is not retried)", calls)
	}
}

func TestGeminiGivesUpAfterTheSecond503(t *testing.T) {
	noBackoff(t)
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"status":"UNAVAILABLE"}}`))
	}))
	defer srv.Close()
	p := NewGeminiProvider("k", srv.URL, "m", srv.Client())
	if _, err := p.GenerateContent(context.Background(), "s", "u"); err == nil || !strings.Contains(err.Error(), "status 503") {
		t.Fatalf("err = %v, want status 503", err)
	}
	if calls != 2 {
		t.Errorf("upstream calls = %d, want exactly 2", calls)
	}
}

func TestGeminiRetryBackoffRespectsTheDeadline(t *testing.T) {
	prev := retryBackoff
	retryBackoff = time.Minute
	t.Cleanup(func() { retryBackoff = prev })
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	p := NewGeminiProvider("k", srv.URL, "m", srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := p.GenerateContent(ctx, "s", "u")
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "status 503") {
		t.Fatalf("err = %v, want DeadlineExceeded that still names the 503", err)
	}
	if time.Since(started) > time.Second || calls != 1 {
		t.Errorf("took %s with %d calls; the backoff must end with the context", time.Since(started), calls)
	}
}
```

Update `TestGeminiDefaultsToTheRealEndpoint`: add `if DefaultGeminiModel != "gemini-3.8-flash" { t.Errorf("DefaultGeminiModel = %q; gemini-2.5-flash is retired for new accounts (404, 2026-09-25)", DefaultGeminiModel) }`. Change the model in `TestGeminiSendsTheSpec62RequestAndReturnsTheFirstPart` from `gemini-2.5-flash` to `gemini-3.8-flash` (path assertion too). In `TestGeminiRejectsNon2xxEmptyCandidatesAndBadJSON` call `noBackoff(t)` first (the `429` case is now retried once, and the case's expectation stays `status 429`).

In `openai_test.go` extend the first test's response with `"usage":{"prompt_tokens":7,"completion_tokens":9,"total_tokens":16}` and, with `logs := captureLog(t)` before the call and `withTask(context.Background(), TaskEssayGrading)` as the context, assert the log contains `openai-compat(deepseek-chat) task=essay_grading ok ` and `tokens prompt=7 completion=9` and not `sk-1`. Add `noBackoff(t)` to the OpenAI non-2xx test if it has a 429/5xx case.

- [ ] **Step 2: Run to see them fail.** `cd backend && go test ./internal/airouter/ -count=1` → `undefined: retryBackoff`, `withTask` exists (Task 1), `errorBodyChars` undefined.

- [ ] **Step 3: `gemini.go` — constants, model, usage, log.**

```go
// Gemini defaults. §6.2's "gemini.api.internal" was a placeholder; this is the
// real Generative Language API. The model is configurable (GEMINI_MODEL)
// because names churn: gemini-2.5-flash answers 404 "no longer available to
// new users" since 2026-09; Google points at gemini-3.8-flash.
const (
	DefaultGeminiBaseURL = "https://generativelanguage.googleapis.com"
	DefaultGeminiModel   = "gemini-3.8-flash"
)
```

In `GenerateContent`: `label := fmt.Sprintf("gemini(%s)", g.model)`; `started := time.Now()` before `postJSON(ctx, g.client, label, url, reqBody, headers)`; extend the parsed struct with

```go
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
```

(keep `FinishReason`/`PromptFeedback` if the base has them — see the conflict note) and, just before the successful `return`, `logCall(ctx, label, started, parsed.UsageMetadata.PromptTokenCount, parsed.UsageMetadata.CandidatesTokenCount)`.

- [ ] **Step 4: `postJSON` with the retry, and `logCall`.** Replace `postJSON`/`truncate` with:

```go
// errorBodyChars bounds how much of an upstream error body reaches errors
// and logs: enough for Google's "model retired" sentence, never a flood.
const errorBodyChars = 200

// retryBackoff is the pause before the single retry of a 429/502/503/504
// answer ("Spikes in demand are usually temporary"). A var so tests need not wait.
var retryBackoff = 2 * time.Second

func retryable(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// postJSON is shared by both providers: marshal, POST, require 2xx, return the
// body. A retryable status is tried once more after retryBackoff, still under
// ctx; any other failure is returned with the status and a truncated body.
// label names the provider and model in log lines and never includes the key.
func postJSON(ctx context.Context, client *http.Client, label, url string, reqBody any, headers map[string]string) ([]byte, error) {
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	for attempt := 0; ; attempt++ {
		body, status, err := doJSON(ctx, client, url, raw, headers)
		if err != nil {
			return nil, err
		}
		if status >= 200 && status <= 299 {
			return body, nil
		}
		upstream := fmt.Errorf("status %d: %s", status, truncate(body, errorBodyChars))
		if attempt > 0 || !retryable(status) {
			return nil, upstream
		}
		log.Printf("airouter: %s: %v; retrying once in %s", label, upstream, retryBackoff)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("%w while waiting to retry: %v", ctx.Err(), upstream)
		case <-time.After(retryBackoff):
		}
	}
}

// doJSON is one POST: transport errors are returned, any status is reported.
func doJSON(ctx context.Context, client *http.Client, url string, raw []byte, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, 0, fmt.Errorf("read body: %w", err)
	}
	return body, resp.StatusCode, nil
}

// logCall is the one line an operator reads per successful call.
func logCall(ctx context.Context, label string, started time.Time, promptTokens, completionTokens int) {
	log.Printf("airouter: %s task=%s ok %.1fs tokens prompt=%d completion=%d", label, taskFrom(ctx), time.Since(started).Seconds(), promptTokens, completionTokens)
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
```

Add `"log"` and `"time"` to `gemini.go`'s imports.

- [ ] **Step 5: `openai.go`.** `label := fmt.Sprintf("openai-compat(%s)", o.model)`; `started := time.Now()`; call `postJSON(ctx, o.client, label, o.baseURL+"/chat/completions", reqBody, …)`; extend the parsed struct with

```go
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
```

and `logCall(ctx, label, started, parsed.Usage.PromptTokens, parsed.Usage.CompletionTokens)` before the successful return. Add `"time"` to its imports.

- [ ] **Step 6: The router names every failure.** In `router.go` replace lines 84-88 with:

```go
		started := time.Now()
		out, err := provider.GenerateContent(ctx, systemPrompt, userPrompt)
		if err == nil {
			return out, nil
		}
		log.Printf("airouter: %s failed for task %s after %.1fs: %v", p, task, time.Since(started).Seconds(), err)
		errs = append(errs, fmt.Errorf("%s: %w", p, err))
```

Add `"time"` to its imports. Add a test in `router_test.go`: with `captureLog(t)`, `TestRouteJoinsEveryErrorWhenAllProvidersFail` also asserts the log contains `gemini failed for task roadmap_generation after` and `openai failed for task roadmap_generation after`.

- [ ] **Step 7: Run the package with race.** `cd backend && go test ./internal/airouter/ -count=1 -race` → PASS. Then `cd backend && make check` → PASS (the onboarding fakes never hit `postJSON`, so nothing else changes).

- [ ] **Step 8: Commit.**
```bash
git add backend/internal/airouter/
git commit -m "airouter: gemini-3.8-flash default, one retry on 429/502/503/504, per-call logs with elapsed time, tokens and the upstream error"
```

---

### Task 5: Configuration and documentation

**Files:**
- Modify: `backend/.env.example:32-34`
- Modify: `backend/cmd/api/server.go:25-29`
- Modify: `CLAUDE.md:50-56,64`
- Modify: `project-base/1st-thinking-architecture-doc.md` (insert before the `\#\# 7\. Core REST API Endpoint Specifications` line)
- Modify (**only if present**): `deploy/README.md` env table

- [ ] **Step 1: `.env.example`.** Replace line 34 with:

```
# gemini-2.5-flash answers 404 "no longer available to new users" (2026-09-25); override only to pin a model.
#GEMINI_MODEL=gemini-3.8-flash
```

- [ ] **Step 2: `server.go` comment.** Replace the `newServer` doc comment (lines 25-29) with:

```go
// newServer wraps the router in an http.Server that main can shut down.
// WriteTimeout is deliberately unset: POST /integrations/google/sync runs up
// to google.SyncTimeout (60 s) and an onboarding assessment runs each
// airouter.Route call under airouter.TaskTimeout (180 s for the roadmap,
// 30 s otherwise, one malformed-body retry each); a server-wide write
// deadline would cut those responses off mid-flight.
```

`cmd/api`'s test that asserts `WriteTimeout == 0` stays as is.

- [ ] **Step 3: `CLAUDE.md`.** After the paragraph ending `…differing only in base URL and model).` (line 50) add:

```
Per-task deadlines live in `airouter/timeouts.go` and travel in the context: `TaskTimeout` is 180 s for roadmap generation (the §6.1 answer is ~4.5 k output tokens and takes 53–78 s on the OpenAI/DeepSeek fallbacks) and 30 s for every other task; `onboarding.routeJSON` sets it per `Route` call, `Route` applies it when a caller passes no deadline, and the drivers' `http.Client` has no timeout of its own. A deadline hit surfaces as 504 `ai_timeout`. Each driver retries once after 2 s on 429/502/503/504 and logs elapsed time, token usage and, on failure, the upstream status plus the first 200 chars of the body (never the key). The Gemini default model is `gemini-3.8-flash` — `gemini-2.5-flash` is retired for new accounts.
```

On line 64 append `and \`GEMINI_MODEL\`, \`OPENAI_MODEL\`, \`DEEPSEEK_MODEL\`` to the sentence about `*_BASE_URL`.

- [ ] **Step 4: Spec addendum.** In `project-base/1st-thinking-architecture-doc.md`, immediately before the line `\#\# 7\. Core REST API Endpoint Specifications`, insert (blank line before and after):

```
**Addendum (2026-09-25, evaluator):** the `http.Client{Timeout: 30 * time.Second}` in the snippets above is superseded. Deadlines are per task and travel in the context (`airouter.TaskTimeout`: 180 s for `roadmap_generation`, 30 s otherwise); the drivers' client has no timeout. `DefaultGeminiModel` is `gemini-3.8-flash` (`gemini-2.5-flash` answers 404 for accounts created after 2026-09). Each provider call retries once on 429/502/503/504 and logs elapsed time, token usage and the upstream error. The onboarding assessment keeps the graded level in `quiz:placement:{user_id}` (§4) so a re-submit after a failed roadmap step is not graded again.
```

- [ ] **Step 5: Runbook row — conditional.** `test -f deploy/README.md || echo 'Task 5 step 5 skipped: deploy/ not on base'`. If present, add after the `GEMINI_API_KEY` row of the env table:

```
| `GEMINI_MODEL` | no | Gemini model name; default `gemini-3.8-flash` (`gemini-2.5-flash` answers 404 for accounts created after 2026-09) | set only to pin a model | same |
```

If absent, write `Task 5 step 5 skipped: deploy/ not on base` in the execution summary. Do not merge or cherry-pick the deploy branch.

- [ ] **Step 6: Check and commit.** `cd backend && make check` → PASS. `grep -rn 'gemini-2.5-flash' backend/ CLAUDE.md` → only the "retired" remarks. Commit:

```bash
git add backend/.env.example backend/cmd/api/server.go CLAUDE.md project-base/1st-thinking-architecture-doc.md
test -f deploy/README.md && git add deploy/README.md
git commit -m "docs: per-task AI deadlines, gemini-3.8-flash default, GEMINI_MODEL override"
```

---

### Task 6: Frontend — tell the learner the wait is normal

**Files:**
- Modify: `frontend/pages/onboarding.vue` (quiz card, after the last `AppButton`)
- Modify: `frontend/tests/unit/onboardingPage.test.ts`

There is no client-side fetch timeout to raise (`frontend/utils/apiClient.ts` sets none; the browser waits), so this is copy only.

- [ ] **Step 1: Failing test.** Append inside the `describe` in `onboardingPage.test.ts`:

```ts
  it('says the grading and roadmap can take a minute or two while the POST is pending', async () => {
    let finish!: (v: unknown) => void
    api.post.mockImplementation(() => new Promise((resolve) => { finish = resolve }))
    const w = mountPage()
    await flushPromises()
    await completeQuiz(w)
    expect(w.text()).toContain('thường mất 1–2 phút')
    finish(ASSESSED)
    await flushPromises()
    expect(w.text()).not.toContain('thường mất 1–2 phút')
    expect(w.text()).toContain('C1')
  })
```

Run: `cd frontend && npx vitest run tests/unit/onboardingPage.test.ts` → the new case fails on the first `toContain`.

- [ ] **Step 2: The copy.** In `onboarding.vue`, directly after the `AppButton` that renders `{{ isLast ? 'Hoàn thành' : 'Tiếp tục' }}` (inside `<template v-else-if="current">`), add:

```vue
        <p v-if="loading && isLast" class="mt-3 text-center text-sm text-mute" role="status">
          Đang chấm bài và soạn lộ trình 28 ngày — thường mất 1–2 phút. Đừng đóng trang.
        </p>
```

- [ ] **Step 3: Verify.** `cd frontend && npm run lint && npm run test:unit` → all green (the suite had 78+ cases before this plan; nothing else changes). Commit:

```bash
git add frontend/pages/onboarding.vue frontend/tests/unit/onboardingPage.test.ts
git commit -m "onboarding page: say the grading and roadmap take a minute or two while the POST is pending"
```

---

### Task 7: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` — the **airouter**, **onboarding** and **cmd/api** bullets

- [ ] **Step 1:** In the **airouter** bullet replace `30 s client timeout;` with: `per-task deadlines in the context (`timeouts.go`: `TaskTimeout` = 180 s `roadmap_generation`, 30 s otherwise — set by `onboarding.routeJSON`, applied by `Route` when the caller set none; the drivers' `http.Client` has no timeout); `postJSON` retries once after 2 s on 429/502/503/504 under the same context, error bodies are cut at 200 chars; each success logs `airouter: <provider>(<model>) task=<task> ok <s>s tokens prompt=<n> completion=<n>` and `Route` logs each provider failure with elapsed time and the upstream error; Gemini default model `gemini-3.8-flash` (`gemini-2.5-flash` is retired for new accounts);`. In the **onboarding** bullet: after `HSET quiz:placement:{user_id}` … `(§4; a failed grading leaves them inspectable)` add `; after grading the level is staged in the same hash (`_level`, same TTL) and a re-submit with identical answers reuses it instead of grading again`; after `ErrAllProvidersFailed → 502 ai_upstream_failed` add `; a task deadline hit (`ErrAITimeout`) → 504 `ai_timeout`; each AI call runs under `airouter.TaskTimeout`, so one assessment is bounded by 2 × 30 s + 2 × 180 s`. In the **cmd/api** bullet replace `google.SyncTimeout and onboarding's AI calls outlive any sane one` with `google.SyncTimeout and onboarding's per-task AI deadlines (up to 180 s a call) outlive any sane one`.

- [ ] **Step 2:** `python3 tools/harness/cli.py validate` → exit 0. Commit: `git add harness/CODEMAP.md && git commit -m "harness: CODEMAP — per-task AI deadlines, staged placement level, provider retry and logs"`. This is the only `harness/` file the branch may change.

---

## Verification

Run from the worktree root after the last task; paste real output into the execution summary.

```bash
cd backend && make check                                    # gofmt, vet, go test ./... -count=1 -race → ok for every package
cd backend && grep -rn ProviderTimeout . ; echo "exit=$?"    # no matches (exit=1)
cd backend && grep -rn 'gemini-2.5-flash' . ../CLAUDE.md     # only comments/tests that say it is retired
cd backend && go test ./internal/airouter/ ./internal/onboarding/ -run 'Timeout|Deadline|Retr|KeepsTheGrade|GradesAgain|Logs' -count=1 -race -v | grep -E '^(--- |ok|FAIL)'
cd frontend && npm run lint && npm run test:unit             # lint clean; every case passes incl. the waiting-copy one
git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP   # empty
test -f deploy/README.md && grep -n GEMINI_MODEL deploy/README.md || echo 'Task 5 step 5 skipped: deploy/ not on base'
```

**Curl proof against a local boot with a fake provider (do it if `make up` works on this machine; otherwise record why and rely on the unit tests + CI):**

1. In the scratchpad, write a throwaway HTTP server (Python `http.server` is fine) on `127.0.0.1:18090` that answers `POST /v1/chat/completions`: if the request body's system prompt is the placement prompt (contains `cefr_level` and `placement`), reply `{"choices":[{"message":{"content":"{\"cefr_level\":\"B1\"}"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`; otherwise **sleep 35 s** and reply with a valid 4 × 7 × 3 roadmap (generate it the way `fixtureRoadmap()` does — `airouter.RoadmapSchema` documents the shape) plus a `usage` object.
2. `cd backend && cp .env.example .env`, set `COMPOSE_PROJECT_NAME=<slug>`, unique `REDIS_PORT`/`POSTGRES_PORT`, `make up`; export `DATABASE_URL`/`REDIS_URL`/`JWT_SECRET`/`ENCRYPTION_SECRET_KEY`/`GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET` (dummies are fine for these two), **no `GEMINI_API_KEY`**, `OPENAI_API_KEY=fake`, `OPENAI_BASE_URL=http://127.0.0.1:18090/v1`, `FRONTEND_ORIGIN=http://localhost:3000`; `make run` in the background (`make run > /tmp/api.log 2>&1 &`), wait for `curl -sf --max-time 2 http://127.0.0.1:8080/healthz`.
3. Insert a user row with `psql` (`INSERT INTO users (id, email, google_id, …)` — read `0001_init.up.sql` for the NOT NULL columns) and mint a JWT with a throwaway `backend/cmd/minttoken/main.go` calling `auth.NewTokenIssuer(os.Getenv("JWT_SECRET"), time.Now).Issue(<user id>)` (`go run ./cmd/minttoken`; **delete the directory afterwards** so `make check` and the commit never see it).
4. `curl -s --max-time 240 -o /tmp/assess.json -w '%{http_code} %{time_total}s\n' -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"target_goal":"IELTS 7.0 Preparation","notification_time":"20:00:00","timezone":"Asia/Ho_Chi_Minh","answers":[{"question_id":"q1","selected_option":"B"}]}' http://127.0.0.1:8080/api/v1/onboarding/assessment`
   Expected: `201 3x.xxxs` (≥ 35 s — before this plan the same call was `502 3x.xx s`), `/tmp/assess.json` has `"assessed_level":"B1"` and a `roadmap_id`; `/tmp/api.log` shows `airouter: fallback from gemini to openai for task placement_test`, `airouter: openai-compat(gpt-4o-mini) task=placement_test ok 0.0s tokens prompt=1 completion=1`, and `… task=roadmap_generation ok 35.0s …`.
5. Restart the fake so the roadmap answer is `503 {"error":{"message":"high demand"}}` on the first call and the roadmap on the second; `DELETE FROM roadmaps` for the user; repeat the curl → `201` after ≈ 37 s, log shows `retrying once in 2s`. Then make the fake answer 503 to every roadmap call, delete the roadmap again, repeat → `502 {"error":"ai_upstream_failed"}` within ~5 s, log shows `openai failed for task roadmap_generation after 2.0s: … status 503: {"error":{"message":"high demand"}}`, `redis-cli -p <port> HGETALL quiz:placement:<user id>` shows `_level B1`; repeat the same curl once more with the fake healthy → `201` and the log shows `reusing the staged level B1` and **no** placement call.
6. `make down` for the same `COMPOSE_PROJECT_NAME`; remove `cmd/minttoken`; `git status --short` shows nothing untracked.

The proof on the live host is the owner's after the daily PR merges: one real onboarding with `GEMINI_API_KEY` unset and `OPENAI_API_KEY` set must return 201 in well under 180 s, and `railway logs` must show the `ok …s tokens …` line for the roadmap.

CI on the pushed `harness/*` branch (`gh run list --branch <branch>`) must be green: `backend-unit`, `backend-integration` (runs the new `TEST_REDIS_URL` test and fails on a skip), `harness-tooling`.

## Execution summary

Built and executed exactly as the plan's 7 tasks specify, in
`.worktrees/providertimeout-of-30-s-makes-roadmap-generation-impossible-` on
`harness/2026-09-25-high-providertimeout-of-30-s-makes-roadmap-generation-impossible-`,
based on freshly fetched `origin/main` (`959cb5f`). One commit per task, in order.

**Base-branch conflict note resolved as predicted:** `grep -n finishReason backend/internal/airouter/gemini.go`
on `origin/main` had no hit, so the geminiprovider response-struct branch had not merged; Task 4 applied to
`gemini.go`'s struct as it stood on main. `git ls-tree origin/main deploy` was empty, so Task 5 Step 5 (the
`deploy/README.md` env-table row) was skipped and recorded, per the plan's own conditional rule — no merge or
cherry-pick of either branch.

### Deviations
- After Task 3's edit, `gofmt` flagged `internal/onboarding/fakes_test.go`'s `newFakeQuiz` one-liner (too long
  for gofmt's wrap rule) when `make check` ran during Task 4. Fixed with `gofmt -w` and folded into the Task 4
  commit (`b09919e`) with a note, since `make check` must stay green after every task and the file had no other
  content changes.
- No other deviations. All test names, function bodies, log-line formats, and file paths matched the plan
  verbatim; no task needed reinterpretation.

### Verification (plan's `## Verification` block, run from the worktree root)

```
cd backend && make check
  → ok for cmd/api, airouter, auth, config, google, health, middleware, notify, onboarding, pet, quests, secrets, store

cd backend && grep -rn ProviderTimeout . ; echo "exit=$?"
  → no matches, exit=1

cd backend && grep -rn 'gemini-2.5-flash' . ../CLAUDE.md
  → only in gemini.go's comment, gemini_test.go's retirement-message fixtures/assertions, .env.example's comment,
    and CLAUDE.md's new paragraph — all "retired" remarks, no live default

cd backend && go test ./internal/airouter/ ./internal/onboarding/ -run 'Timeout|Deadline|Retr|KeepsTheGrade|GradesAgain|Logs' -count=1 -race -v | grep -E '^(--- |ok|FAIL)'
  --- PASS: TestGeminiRetriesOnceOn503AndLogsElapsedAndTokens (0.00s)
  --- PASS: TestGeminiDoesNotRetryA404AndReportsGooglesMessageTruncated (0.00s)
  --- PASS: TestGeminiRetryBackoffRespectsTheDeadline (0.05s)
  --- PASS: TestTaskTimeoutsMatchTheMeasuredProviders (0.00s)
  --- PASS: TestRouteKeepsACallerDeadlineInsteadOfWideningIt (0.00s)
  --- PASS: TestProvidersObeyTheContextDeadlineNotAClientTimeout (0.71s)
  ok  	.../internal/airouter	2.110s
  --- PASS: TestAssessRetriesOnceOnABadGradeThenFailsWithoutWriting (0.00s)
  --- PASS: TestAssessGivesEachAICallItsOwnDeadline (0.00s)
  --- PASS: TestAssessReportsADeadlineHitAsAITimeoutWithoutWriting (0.00s)
  --- PASS: TestAssessKeepsTheGradeWhenTheRoadmapFailsAndSkipsGradingOnTheRetry (0.00s)
  --- PASS: TestAssessGradesAgainWhenTheRetryChangesAnAnswer (0.00s)
  ok  	.../internal/onboarding	1.631s

cd frontend && npm run lint && npm run test:unit
  → eslint clean; 16 files, 79/79 tests passed (incl. the new waiting-copy test)

git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP
  → empty (only harness/CODEMAP.md touched, as required)

test -f deploy/README.md && grep -n GEMINI_MODEL deploy/README.md || echo 'Task 5 step 5 skipped: deploy/ not on base'
  → Task 5 step 5 skipped: deploy/ not on base
```

### Runtime proof (definition-of-done §8, backend `make up` was available)

Ran the full local-boot curl proof with a throwaway Python fake provider on `127.0.0.1:18090` (scratchpad,
never committed), `COMPOSE_PROJECT_NAME=aelp-timeout-rp`, `POSTGRES_PORT=15532`, `REDIS_PORT=16479`, API on
`PORT=18081` (8080 was already held by another local instance), `GEMINI_API_KEY` unset, `OPENAI_API_KEY=fake`,
`OPENAI_BASE_URL=http://127.0.0.1:18090/v1`. Session auth needed one addition the plan's step 3 didn't spell
out: `Require` also checks the Redis `sess:{user_id}:token` key, so the throwaway `cmd/minttoken` JWT alone
401'd until `redis-cli SET sess:<user id>:token <jwt> EX 86400` matched it.

1. **Fresh onboarding, fake sleeps 35 s on the roadmap call:**
   `201 35.087206s` (`assessed_level: B1`, a `roadmap_id`). API log:
   ```
   airouter: fallback from gemini to openai for task placement_test
   airouter: openai-compat(gpt-4o-mini) task=placement_test ok 0.0s tokens prompt=1 completion=1
   airouter: fallback from gemini to openai for task roadmap_generation
   airouter: openai-compat(gpt-4o-mini) task=roadmap_generation ok 35.0s tokens prompt=100 completion=4500
   [GIN] 201 | 35.08s | POST "/api/v1/onboarding/assessment"
   ```
   Before this plan the same shape of call 502'd at ~34 s (the old 30 s `ProviderTimeout`).

2. **Roadmap 503s once then succeeds** (`DELETE FROM roadmaps` first): `201 4.048188s`, log shows
   `airouter: openai-compat(gpt-4o-mini): status 503: {"error": {"message": "high demand"}}; retrying once in 2s`
   followed by a successful `ok 4.0s` line.

3. **Roadmap 503s on every call** (`DELETE FROM roadmaps` first): `502 {"error":"ai_upstream_failed"}` in
   `2.051670s`, log shows
   `airouter: openai failed for task roadmap_generation after 2.0s: openai-compat(gpt-4o-mini): status 503: {"error": {"message": "high demand"}}`.
   `redis-cli HGETALL quiz:placement:<user id>` → `q1 B _level B1` (the grade survived the failed roadmap step).

4. **Same request repeated, fake healthy:** `201 0.035096s`. Log shows
   `onboarding: reusing the staged level B1 for <user id>` and **no** `task=placement_test` line — the retry
   skipped grading. `HGETALL quiz:placement:<user id>` afterward is empty (cleared on success).

The 504 `ai_timeout` path (Task 2) was not re-proven live — it would need a real ~180 s wait for no additional
signal — and is instead covered by `TestAssessReportsADeadlineHitAsAITimeoutWithoutWriting` and
`TestAssessmentErrorMapping/ai_timed_out`, both passing above.

**Cleanup verified:** `pkill`'d the fake provider and the `go run`-spawned `exe/api` binary (the latter needed
an extra `lsof -t :18081 | kill` — `pkill -f "go run"` alone leaves the compiled child running), `docker compose
down` + `docker volume rm aelp-timeout-rp_postgres_data`, `docker ps` / `pgrep -fl "cmd/api\|fake_provider"`
both empty afterward, `rm -rf backend/cmd/minttoken`, `rm -f backend/.env`, `git status --short` clean.

### CI

Pushed `harness/2026-09-25-high-providertimeout-of-30-s-makes-roadmap-generation-impossible-`.
Run: https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36120925682 — **green**:
`backend-unit` (1m13s), `frontend` (44s), `harness-tooling` (7s), `backend-integration` (40s, ran the new
`TEST_REDIS_URL`-gated `TestIntegrationQuizStoreStagesAndReusesTheGradedLevel` without skipping).
