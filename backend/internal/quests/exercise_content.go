package quests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// ErrBadAIOutput means two attempts at TaskExerciseGen both returned a body
// ParseExerciseContent rejected. The handler maps it to 502 ai_bad_output.
var ErrBadAIOutput = errors.New("quests: AI output did not match the exercise schema")

// ErrAITimeout means one TaskExerciseGen call did not finish inside
// airouter.TaskTimeout(TaskExerciseGen). The handler maps it to 504 ai_timeout;
// a caller's own cancellation is returned as is (see route).
var ErrAITimeout = errors.New("quests: AI call timed out")

// ContentAI is the one Route call ContentService needs — the same shape as
// onboarding.Generator, narrowed to what generating one exercise's content
// takes.
type ContentAI interface {
	Route(ctx context.Context, task airouter.TaskType, system, user string) (string, error)
}

// ContentLimiter gates the one AI call a first open spends — airouter.RateLimiter
// narrowed to the method ContentService calls.
type ContentLimiter interface {
	Allow(ctx context.Context, userID string) error
}

// ContentService implements GET /api/v1/quests/exercises/:id: the first open
// of an exercise generates its typed content and caches it; every later open
// is a DB read (see Exercise).
type ContentService struct {
	repo    ContentRepo
	ai      ContentAI
	limiter ContentLimiter
	now     func() time.Time
	// timeout overrides airouter.TaskTimeout(TaskExerciseGen) for tests that
	// need an AI call to block until its deadline in milliseconds, not 60s.
	// Zero means "use the real budget".
	timeout time.Duration
}

// NewContentService wires the collaborators.
func NewContentService(repo ContentRepo, ai ContentAI, limiter ContentLimiter, now func() time.Time) *ContentService {
	if now == nil {
		now = time.Now
	}
	return &ContentService{repo: repo, ai: ai, limiter: limiter, now: now}
}

// exerciseMeta is the subset of one exercises.content_json this service reads
// before generation: the roadmap's own title/duration_minutes (present since
// the roadmap was generated) and, once typed, content/content_schema.
type exerciseMeta struct {
	Title           string          `json:"title"`
	DurationMinutes int             `json:"duration_minutes"`
	Content         json.RawMessage `json:"content"`
	ContentSchema   int             `json:"content_schema"`
}

// Exercise returns exerciseID's typed content, generating it on the first
// open. See the plan's Task 4 for the numbered steps this follows.
func (s *ContentService) Exercise(ctx context.Context, userID, exerciseID string) (Task, error) {
	target, err := s.repo.ContentTarget(ctx, userID, exerciseID)
	if err != nil {
		return Task{}, err
	}

	today := DayNumber(target.RoadmapCreatedAt, s.now(), Location(target.Timezone))
	if target.Exercise.DayNumber > today {
		return Task{}, ErrExerciseNotFound
	}

	var meta exerciseMeta
	_ = json.Unmarshal(target.Exercise.ContentJSON, &meta)
	if meta.ContentSchema == 1 {
		return toTask(target.Exercise), nil
	}

	if err := s.limiter.Allow(ctx, userID); err != nil {
		return Task{}, err
	}

	brief := airouter.ExerciseBrief{
		TaskType:    target.Exercise.TaskType,
		Level:       target.Level,
		Goal:        target.Goal,
		ModuleFocus: target.ModuleFocus,
		DayTitle:    target.DayTitle,
		TaskTitle:   meta.Title,
		Draft:       compactDraft(meta.Content),
	}

	content, err := s.generate(ctx, brief)
	if err != nil {
		return Task{}, err
	}

	saved, err := s.repo.SaveContent(ctx, exerciseID, content)
	if err != nil {
		return Task{}, err
	}
	log.Printf("quests: generated content for exercise %s (%s, %s)", exerciseID, target.Exercise.TaskType, target.Level)

	out := target.Exercise
	out.ContentJSON = saved
	return toTask(out), nil
}

// compactDraft compacts content (the roadmap's own free-form draft for this
// task) to a single line; "" when there is none. ExerciseUserPrompt is what
// actually bounds it to 600 runes.
func compactDraft(content json.RawMessage) string {
	if len(content) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, content); err != nil {
		return ""
	}
	return buf.String()
}

// generate runs at most 2 attempts of TaskExerciseGen: a malformed answer is
// logged and retried once; a router error (rate limit, no providers, every
// provider failed, our own timeout) returns immediately with no retry.
func (s *ContentService) generate(ctx context.Context, brief airouter.ExerciseBrief) (json.RawMessage, error) {
	var last error
	for attempt := 1; attempt <= 2; attempt++ {
		raw, err := s.route(ctx, brief)
		if err != nil {
			return nil, err
		}
		content, err := airouter.ParseExerciseContent(brief.TaskType, brief.Level, raw)
		if err != nil {
			last = err
			log.Printf("quests: exercise_generation attempt %d returned a malformed body: %v", attempt, err)
			continue
		}
		return content, nil
	}
	return nil, fmt.Errorf("%w: %v", ErrBadAIOutput, last)
}

// route runs one Route call under our own TaskExerciseGen budget (or s.timeout
// in a test) and names a deadline of ours ErrAITimeout — mirroring
// onboarding.Service.route. If the parent context is done the client is gone,
// and that error is returned as is.
func (s *ContentService) route(ctx context.Context, brief airouter.ExerciseBrief) (string, error) {
	budget := s.timeout
	if budget <= 0 {
		budget = airouter.TaskTimeout(airouter.TaskExerciseGen)
	}
	actx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	raw, err := s.ai.Route(actx, airouter.TaskExerciseGen, airouter.ExerciseSystemPrompt, airouter.ExerciseUserPrompt(brief))
	if err != nil && ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
		return "", fmt.Errorf("%w: exercise_generation after %s", ErrAITimeout, budget)
	}
	return raw, err
}

// ExerciseHandler serves GET /api/v1/quests/exercises/:id (backend spec
// §6.2). It must be mounted behind auth.Require().
func ExerciseHandler(svc *ContentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		task, err := svc.Exercise(c.Request.Context(), userID, c.Param("id"))
		switch {
		case errors.Is(err, ErrExerciseNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "exercise_not_found"})
		case errors.Is(err, airouter.ErrRateLimited):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
		case errors.Is(err, airouter.ErrNoProviders):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ai_unavailable"})
		case errors.Is(err, ErrBadAIOutput):
			c.JSON(http.StatusBadGateway, gin.H{"error": "ai_bad_output"})
		case errors.Is(err, ErrAITimeout):
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "ai_timeout"})
		case errors.Is(err, airouter.ErrAllProvidersFailed):
			c.JSON(http.StatusBadGateway, gin.H{"error": "ai_upstream_failed"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			c.JSON(http.StatusOK, task)
		}
	}
}
