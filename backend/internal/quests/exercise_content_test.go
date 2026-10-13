package quests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// --- fakes, local to this file (do not touch fakes_test.go: the plan's
// Global Constraints reserve it for other branches). ---

type fakeContentRepo struct {
	target        ContentTarget
	targetErr     error
	saveContentFn func(exerciseID string, content json.RawMessage) (json.RawMessage, error)
	loseCAS       bool
	winner        json.RawMessage
	savedLevel    string
	targetCalls   int
	saveCalls     int
	savedContent  json.RawMessage
}

func (f *fakeContentRepo) ContentTarget(ctx context.Context, userID, exerciseID string) (ContentTarget, error) {
	f.targetCalls++
	return f.target, f.targetErr
}

func (f *fakeContentRepo) SaveContent(ctx context.Context, userID, exerciseID string, original, content json.RawMessage, level string) (json.RawMessage, bool, error) {
	f.saveCalls++
	f.savedContent = content
	f.savedLevel = level
	if f.loseCAS {
		f.target.Exercise.ContentJSON = f.winner
		return nil, false, nil
	}
	if f.saveContentFn != nil {
		out, err := f.saveContentFn(exerciseID, content)
		f.target.Exercise.ContentJSON = out
		return nil, false, err
	}
	f.target.Exercise.ContentJSON = generatedSnapshot(original, content, level)
	return f.target.Exercise.ContentJSON, true, nil
}

var _ ContentRepo = (*fakeContentRepo)(nil)

// fakeContentAI scripts Route: answers[i]/errs[i] for the i-th call (0-based).
// When block is true, Route waits on ctx.Done() and returns its error — used
// to simulate a provider slower than the deadline without a real sleep.
type fakeContentAI struct {
	answers     []string
	errs        []error
	block       bool
	calls       int
	tasks       []airouter.TaskType
	userPrompts []string
}

func (f *fakeContentAI) Route(ctx context.Context, task airouter.TaskType, system, user string) (string, error) {
	i := f.calls
	f.calls++
	f.tasks = append(f.tasks, task)
	f.userPrompts = append(f.userPrompts, user)
	if f.block {
		<-ctx.Done()
		return "", ctx.Err()
	}
	var err error
	if i < len(f.errs) {
		err = f.errs[i]
	}
	if err != nil {
		return "", err
	}
	if i < len(f.answers) {
		return f.answers[i], nil
	}
	return "", errors.New("fakeContentAI: no scripted answer")
}

var _ ContentAI = (*fakeContentAI)(nil)

type fakeContentLimiter struct {
	err   error
	calls int
}

func (f *fakeContentLimiter) Allow(ctx context.Context, userID string) error {
	f.calls++
	return f.err
}

var _ ContentLimiter = (*fakeContentLimiter)(nil)

// contentJSON builds a roadmap-style exercises.content_json: title,
// duration_minutes and a free-form "content" draft, optionally typed
// (content_schema: 1).
func contentJSON(title string, draft string, typed bool) json.RawMessage {
	schema := ""
	if typed {
		schema = `,"content_schema":1`
	}
	return json.RawMessage(fmt.Sprintf(`{"type":"vocabulary","title":%q,"duration_minutes":10,"content":%s%s}`, title, draft, schema))
}

func baseTarget(now time.Time) ContentTarget {
	return ContentTarget{
		Exercise:         Exercise{ID: "ex-1", DayNumber: 1, TaskType: "vocabulary", IsCompleted: false, ContentJSON: contentJSON("Vocabulary warm-up", `{"topic":"greetings"}`, false)},
		RoadmapCreatedAt: now,
		Timezone:         "UTC",
		Level:            "B1",
		Goal:             "pass an interview",
		ModuleFocus:      "Workplace English",
		DayTitle:         "Meetings",
	}
}

func TestContentServiceQuestionQualityRetry(t *testing.T) {
	now := time.Date(2026, time.October, 12, 10, 0, 0, 0, time.UTC)
	valid := string(airouter.SampleContent("vocabulary"))
	cases := []struct {
		name, reason string
		change       func([]airouter.Question)
	}{
		{"duplicate prompts", "question q2 repeats prompt of question q1", func(q []airouter.Question) { q[1].Prompt = q[0].Prompt }},
		{"duplicate choices", "question q1 option B repeats option A", func(q []airouter.Question) { q[0].Options["B"] = q[0].Options["A"] }},
		{"placeholder", "question q1 has placeholder explanation", func(q []airouter.Question) { q[0].Explanation = "TODO" }},
	}
	for _, c := range cases {
		var content airouter.VocabularyContent
		if err := json.Unmarshal([]byte(valid), &content); err != nil {
			t.Fatal(err)
		}
		c.change(content.Questions)
		invalid, _ := json.Marshal(content)
		for _, recover := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/recover=%t", c.name, recover), func(t *testing.T) {
				second := string(invalid)
				if recover {
					second = valid
				}
				repo := &fakeContentRepo{target: baseTarget(now)}
				ai := &fakeContentAI{answers: []string{string(invalid), second, valid}}
				lim := &fakeContentLimiter{}
				svc := NewContentService(repo, ai, lim, func() time.Time { return now })
				_, err := svc.Exercise(context.Background(), "u1", "ex-1")
				if ai.calls != 2 || lim.calls != 1 {
					t.Errorf("AI calls=%d, limiter calls=%d, want 2/1", ai.calls, lim.calls)
				}
				if recover {
					if err != nil || repo.saveCalls != 1 || string(repo.savedContent) != valid {
						t.Errorf("valid second response: err=%v, saves=%d, saved=%s", err, repo.saveCalls, repo.savedContent)
					}
				} else if !errors.Is(err, ErrBadAIOutput) || !strings.Contains(err.Error(), c.reason) || repo.saveCalls != 0 {
					t.Errorf("two invalid responses: err=%v, saves=%d, want ErrBadAIOutput with %q and no save", err, repo.saveCalls, c.reason)
				}
			})
		}
	}
}

func TestContentServiceExercise(t *testing.T) {
	now := time.Date(2026, time.September, 27, 10, 0, 0, 0, time.UTC)
	nowFn := func() time.Time { return now }

	t.Run("cached: no limiter call, no AI call", func(t *testing.T) {
		target := baseTarget(now)
		target.Exercise.ContentJSON = contentJSON("Vocabulary warm-up", string(airouter.SampleContent("vocabulary")), true)
		repo := &fakeContentRepo{target: target}
		ai := &fakeContentAI{}
		lim := &fakeContentLimiter{}
		svc := NewContentService(repo, ai, lim, nowFn)

		task, err := svc.Exercise(context.Background(), "u1", "ex-1")
		if err != nil {
			t.Fatalf("Exercise: %v", err)
		}
		if lim.calls != 0 || ai.calls != 0 {
			t.Errorf("limiter calls = %d, AI calls = %d, want 0 and 0", lim.calls, ai.calls)
		}
		if repo.saveCalls != 0 {
			t.Errorf("SaveContent called %d times, want 0", repo.saveCalls)
		}
		if task.Title != "Vocabulary warm-up" || task.DurationMinutes != 10 {
			t.Errorf("task = %+v, want the stored title/duration", task)
		}
		if string(task.ContentJSON) != string(target.Exercise.ContentJSON) {
			t.Errorf("ContentJSON = %s, want the stored body unchanged", task.ContentJSON)
		}
	})

	t.Run("first open: 1 limiter call, 1 AI call naming the exercise, 1 save", func(t *testing.T) {
		target := baseTarget(now)
		repo := &fakeContentRepo{target: target}
		ai := &fakeContentAI{answers: []string{string(airouter.SampleContent("vocabulary"))}}
		lim := &fakeContentLimiter{}
		svc := NewContentService(repo, ai, lim, nowFn)

		task, err := svc.Exercise(context.Background(), "u1", "ex-1")
		if err != nil {
			t.Fatalf("Exercise: %v", err)
		}
		if lim.calls != 1 {
			t.Errorf("limiter calls = %d, want 1", lim.calls)
		}
		if ai.calls != 1 {
			t.Errorf("AI calls = %d, want 1", ai.calls)
		}
		if len(ai.tasks) != 1 || ai.tasks[0] != airouter.TaskExerciseGen {
			t.Errorf("task = %v, want TaskExerciseGen", ai.tasks)
		}
		prompt := ai.userPrompts[0]
		for _, want := range []string{target.Level, target.Goal, target.ModuleFocus, target.DayTitle, "Vocabulary warm-up"} {
			if !strings.Contains(prompt, want) {
				t.Errorf("user prompt missing %q:\n%s", want, prompt)
			}
		}
		if repo.saveCalls != 1 {
			t.Errorf("SaveContent called %d times, want 1", repo.saveCalls)
		}
		var parsed struct {
			Words []airouter.Word `json:"words"`
		}
		_ = json.Unmarshal(repo.savedContent, &parsed)
		if len(parsed.Words) == 0 {
			t.Errorf("saved content has no words: %s", repo.savedContent)
		}
		if task.Title != "Vocabulary warm-up" || task.DurationMinutes != 10 {
			t.Errorf("task = %+v, want the stored title/duration", task)
		}
	})

	t.Run("malformed then valid: 2 AI calls, 1 save", func(t *testing.T) {
		target := baseTarget(now)
		repo := &fakeContentRepo{target: target}
		ai := &fakeContentAI{answers: []string{"not json", string(airouter.SampleContent("vocabulary"))}}
		lim := &fakeContentLimiter{}
		svc := NewContentService(repo, ai, lim, nowFn)

		if _, err := svc.Exercise(context.Background(), "u1", "ex-1"); err != nil {
			t.Fatalf("Exercise: %v", err)
		}
		if ai.calls != 2 {
			t.Errorf("AI calls = %d, want 2", ai.calls)
		}
		if repo.saveCalls != 1 {
			t.Errorf("SaveContent called %d times, want 1", repo.saveCalls)
		}
	})

	t.Run("malformed twice: ErrBadAIOutput, 0 saves", func(t *testing.T) {
		target := baseTarget(now)
		repo := &fakeContentRepo{target: target}
		ai := &fakeContentAI{answers: []string{"not json", "still not json"}}
		lim := &fakeContentLimiter{}
		svc := NewContentService(repo, ai, lim, nowFn)

		_, err := svc.Exercise(context.Background(), "u1", "ex-1")
		if !errors.Is(err, ErrBadAIOutput) {
			t.Fatalf("err = %v, want ErrBadAIOutput", err)
		}
		if repo.saveCalls != 0 {
			t.Errorf("SaveContent called %d times, want 0", repo.saveCalls)
		}
	})

	t.Run("future day: ErrExerciseNotFound, 0 limiter calls", func(t *testing.T) {
		target := baseTarget(now) // roadmap created "now" -> today is day 1
		target.Exercise.DayNumber = 3
		repo := &fakeContentRepo{target: target}
		ai := &fakeContentAI{}
		lim := &fakeContentLimiter{}
		svc := NewContentService(repo, ai, lim, nowFn)

		_, err := svc.Exercise(context.Background(), "u1", "ex-1")
		if !errors.Is(err, ErrExerciseNotFound) {
			t.Fatalf("err = %v, want ErrExerciseNotFound", err)
		}
		if lim.calls != 0 {
			t.Errorf("limiter calls = %d, want 0", lim.calls)
		}
	})

	t.Run("past day: generated", func(t *testing.T) {
		target := baseTarget(now.AddDate(0, 0, -4)) // created 4 days ago -> today is day 5
		target.Exercise.DayNumber = 1
		repo := &fakeContentRepo{target: target}
		ai := &fakeContentAI{answers: []string{string(airouter.SampleContent("vocabulary"))}}
		lim := &fakeContentLimiter{}
		svc := NewContentService(repo, ai, lim, nowFn)

		if _, err := svc.Exercise(context.Background(), "u1", "ex-1"); err != nil {
			t.Fatalf("Exercise: %v", err)
		}
		if ai.calls != 1 || repo.saveCalls != 1 {
			t.Errorf("AI calls = %d, saves = %d, want 1 and 1", ai.calls, repo.saveCalls)
		}
	})

	t.Run("limiter refuses: 0 AI calls", func(t *testing.T) {
		target := baseTarget(now)
		repo := &fakeContentRepo{target: target}
		ai := &fakeContentAI{}
		lim := &fakeContentLimiter{err: airouter.ErrRateLimited}
		svc := NewContentService(repo, ai, lim, nowFn)

		_, err := svc.Exercise(context.Background(), "u1", "ex-1")
		if !errors.Is(err, airouter.ErrRateLimited) {
			t.Fatalf("err = %v, want ErrRateLimited", err)
		}
		if ai.calls != 0 {
			t.Errorf("AI calls = %d, want 0", ai.calls)
		}
	})

	t.Run("AI blocks until its deadline: ErrAITimeout", func(t *testing.T) {
		target := baseTarget(now)
		repo := &fakeContentRepo{target: target}
		ai := &fakeContentAI{block: true}
		lim := &fakeContentLimiter{}
		svc := NewContentService(repo, ai, lim, nowFn)
		svc.timeout = 20 * time.Millisecond

		_, err := svc.Exercise(context.Background(), "u1", "ex-1")
		if !errors.Is(err, ErrAITimeout) {
			t.Fatalf("err = %v, want ErrAITimeout", err)
		}
	})

	t.Run("router error is returned as is, no retry", func(t *testing.T) {
		target := baseTarget(now)
		repo := &fakeContentRepo{target: target}
		ai := &fakeContentAI{errs: []error{airouter.ErrAllProvidersFailed}}
		lim := &fakeContentLimiter{}
		svc := NewContentService(repo, ai, lim, nowFn)

		_, err := svc.Exercise(context.Background(), "u1", "ex-1")
		if !errors.Is(err, airouter.ErrAllProvidersFailed) {
			t.Fatalf("err = %v, want ErrAllProvidersFailed", err)
		}
		if ai.calls != 1 {
			t.Errorf("AI calls = %d, want 1 (no retry on a router error)", ai.calls)
		}
	})

	t.Run("race lost: the first writer's content comes back", func(t *testing.T) {
		target := baseTarget(now)
		firstWriter := contentJSON("Vocabulary warm-up", string(airouter.SampleContent("vocabulary")), true)
		repo := &fakeContentRepo{target: target, saveContentFn: func(exerciseID string, content json.RawMessage) (json.RawMessage, error) {
			return firstWriter, nil
		}}
		ai := &fakeContentAI{answers: []string{string(airouter.SampleContent("vocabulary"))}}
		lim := &fakeContentLimiter{}
		svc := NewContentService(repo, ai, lim, nowFn)

		task, err := svc.Exercise(context.Background(), "u1", "ex-1")
		if err != nil {
			t.Fatalf("Exercise: %v", err)
		}
		if string(task.ContentJSON) != string(firstWriter) {
			t.Errorf("ContentJSON = %s, want the first writer's content", task.ContentJSON)
		}
	})
}

// newContentRouter mounts only GET /quests/exercises/:id, the same way
// newQuestRouter (handler_test.go) stands in for auth.Require().
func newContentRouter(svc *ContentService, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	inject := func(c *gin.Context) {
		if userID != "" {
			c.Set(auth.ContextUserID, userID)
		}
		c.Next()
	}
	g := r.Group("/api/v1", inject)
	g.GET("/quests/exercises/:id", ExerciseHandler(svc))
	return r
}

func TestExerciseHandlerMapsEveryCode(t *testing.T) {
	now := time.Date(2026, time.September, 27, 10, 0, 0, 0, time.UTC)
	nowFn := func() time.Time { return now }

	cases := []struct {
		name       string
		userID     string
		repo       *fakeContentRepo
		ai         *fakeContentAI
		lim        *fakeContentLimiter
		wantStatus int
		wantError  string
	}{
		{
			name:       "no user id",
			userID:     "",
			repo:       &fakeContentRepo{},
			ai:         &fakeContentAI{},
			lim:        &fakeContentLimiter{},
			wantStatus: http.StatusUnauthorized,
			wantError:  "unauthorized",
		},
		{
			name:       "exercise not found",
			userID:     "u1",
			repo:       &fakeContentRepo{targetErr: ErrExerciseNotFound},
			ai:         &fakeContentAI{},
			lim:        &fakeContentLimiter{},
			wantStatus: http.StatusNotFound,
			wantError:  "exercise_not_found",
		},
		{
			name:       "rate limited",
			userID:     "u1",
			repo:       &fakeContentRepo{target: baseTarget(now)},
			ai:         &fakeContentAI{},
			lim:        &fakeContentLimiter{err: airouter.ErrRateLimited},
			wantStatus: http.StatusTooManyRequests,
			wantError:  "rate_limited",
		},
		{
			name:       "no providers",
			userID:     "u1",
			repo:       &fakeContentRepo{target: baseTarget(now)},
			ai:         &fakeContentAI{errs: []error{airouter.ErrNoProviders}},
			lim:        &fakeContentLimiter{},
			wantStatus: http.StatusServiceUnavailable,
			wantError:  "ai_unavailable",
		},
		{
			name:       "bad AI output",
			userID:     "u1",
			repo:       &fakeContentRepo{target: baseTarget(now)},
			ai:         &fakeContentAI{answers: []string{"nope", "still nope"}},
			lim:        &fakeContentLimiter{},
			wantStatus: http.StatusBadGateway,
			wantError:  "ai_bad_output",
		},
		{
			name:       "all providers failed",
			userID:     "u1",
			repo:       &fakeContentRepo{target: baseTarget(now)},
			ai:         &fakeContentAI{errs: []error{airouter.ErrAllProvidersFailed}},
			lim:        &fakeContentLimiter{},
			wantStatus: http.StatusBadGateway,
			wantError:  "ai_upstream_failed",
		},
		{
			name:       "internal error",
			userID:     "u1",
			repo:       &fakeContentRepo{targetErr: errors.New("boom")},
			ai:         &fakeContentAI{},
			lim:        &fakeContentLimiter{},
			wantStatus: http.StatusInternalServerError,
			wantError:  "internal_error",
		},
		{
			name:   "cached ok",
			userID: "u1",
			repo: &fakeContentRepo{target: func() ContentTarget {
				t := baseTarget(now)
				t.Exercise.ContentJSON = contentJSON("Vocabulary warm-up", string(airouter.SampleContent("vocabulary")), true)
				return t
			}()},
			ai:         &fakeContentAI{},
			lim:        &fakeContentLimiter{},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewContentService(tc.repo, tc.ai, tc.lim, nowFn)
			w := httptest.NewRecorder()
			newContentRouter(svc, tc.userID).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/quests/exercises/ex-1", nil))
			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", w.Code, tc.wantStatus, w.Body.String())
			}
			if tc.wantError != "" && !strings.Contains(w.Body.String(), fmt.Sprintf(`"error":"%s"`, tc.wantError)) {
				t.Errorf("body = %s, want error %q", w.Body.String(), tc.wantError)
			}
		})
	}

	t.Run("504 ai_timeout", func(t *testing.T) {
		repo := &fakeContentRepo{target: baseTarget(now)}
		ai := &fakeContentAI{block: true}
		lim := &fakeContentLimiter{}
		svc := NewContentService(repo, ai, lim, nowFn)
		svc.timeout = 20 * time.Millisecond

		w := httptest.NewRecorder()
		newContentRouter(svc, "u1").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/quests/exercises/ex-1", nil))
		if w.Code != http.StatusGatewayTimeout {
			t.Fatalf("status = %d, want 504; body = %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"error":"ai_timeout"`) {
			t.Errorf("body = %s, want ai_timeout", w.Body.String())
		}
	})
}

// contentProviderFunc keeps deadline regressions on the real router's
// per-provider budgets and joined errors, rather than scripting Route itself.
type contentProviderFunc func(context.Context, string, string) (string, error)

func (f contentProviderFunc) GenerateContent(ctx context.Context, system, user string) (string, error) {
	return f(ctx, system, user)
}

func TestContentServiceProviderShareTimeoutPreservesUpstream(t *testing.T) {
	for _, mode := range []string{"service", "handler"} {
		t.Run(mode, func(t *testing.T) {
			now := time.Date(2026, time.September, 27, 10, 0, 0, 0, time.UTC)
			repo := &fakeContentRepo{target: baseTarget(now)}
			limiter := &fakeContentLimiter{}
			preferredCalls, fallbackCalls := 0, 0
			router := airouter.NewRouterWithProviders(map[airouter.ProviderType]airouter.LLMProvider{
				airouter.ProviderDeepSeek: contentProviderFunc(func(ctx context.Context, _, _ string) (string, error) {
					preferredCalls++
					<-ctx.Done()
					return "", ctx.Err()
				}),
				airouter.ProviderGemini: contentProviderFunc(func(_ context.Context, _, _ string) (string, error) {
					fallbackCalls++
					return "", errors.New("fallback upstream failure")
				}),
			})
			svc := NewContentService(repo, router, limiter, func() time.Time { return now })
			svc.timeout = 200 * time.Millisecond
			if mode == "service" {
				_, err := svc.Exercise(context.Background(), "u1", "ex-1")
				if !errors.Is(err, airouter.ErrAllProvidersFailed) || !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrAITimeout) {
					t.Fatalf("provider-share timeout: %v, want upstream failure containing DeadlineExceeded, without ErrAITimeout", err)
				}
			} else {
				response := httptest.NewRecorder()
				newContentRouter(svc, "u1").ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/quests/exercises/ex-1", nil))
				if response.Code != http.StatusBadGateway || !strings.Contains(response.Body.String(), `"error":"ai_upstream_failed"`) {
					t.Fatalf("response = %d %s, want 502 ai_upstream_failed", response.Code, response.Body)
				}
			}
			if preferredCalls != 1 || fallbackCalls != 1 || limiter.calls != 1 || repo.saveCalls != 0 {
				t.Errorf("preferred=%d fallback=%d limiter=%d saves=%d, want 1/1/1/0", preferredCalls, fallbackCalls, limiter.calls, repo.saveCalls)
			}
		})
	}
}

func generatedSnapshot(original, content json.RawMessage, level string) json.RawMessage {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(original, &m)
	if m == nil {
		m = map[string]json.RawMessage{}
	}
	m["content"] = content
	m["content_schema"] = json.RawMessage("1")
	m["content_level"], _ = json.Marshal(level)
	out, _ := json.Marshal(m)
	return out
}
