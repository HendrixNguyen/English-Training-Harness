package onboarding

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// ErrInvalidRequest wraps every validation failure (400).
var ErrInvalidRequest = errors.New("onboarding: invalid request")

// ErrAITimeout means an AI call did not finish inside airouter.TaskTimeout.
// The handler maps it to 504 ai_timeout; a caller's own cancellation is
// passed through untouched.
var ErrAITimeout = errors.New("onboarding: AI call timed out")

// DailyMinutes is the study commitment the roadmap prompt is built for (§1).
const DailyMinutes = 30

// Service runs the §5.1 steps 4-5 flow. The request/response DTOs and the
// Pet/Generator seams are in types.go.
type Service struct {
	repo    Repo
	quiz    QuizStore
	limiter airouter.RateLimiter
	ai      Generator
	pet     Pet
	now     func() time.Time
}

// NewService wires the collaborators.
func NewService(repo Repo, quiz QuizStore, limiter airouter.RateLimiter, ai Generator, pet Pet, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, quiz: quiz, limiter: limiter, ai: ai, pet: pet, now: now}
}

// Assess validates, short-circuits when a roadmap is already active, then
// grades, generates and persists — both AI calls before any write, so a
// failure writes nothing. Each AI call runs under its own airouter.TaskTimeout
// budget (180 s for the roadmap, 30 s otherwise); a budget hit is ErrAITimeout.
func (s *Service) Assess(ctx context.Context, userID string, req AssessmentRequest) (AssessmentResult, error) {
	if err := validate(req); err != nil {
		return AssessmentResult{}, err
	}

	if id, ok, err := s.repo.ActiveRoadmapID(ctx, userID); err != nil {
		return AssessmentResult{}, err
	} else if ok {
		profile, err := s.repo.Profile(ctx, userID)
		if err != nil {
			return AssessmentResult{}, err
		}
		pet, err := s.pet.Ensure(ctx, userID)
		if err != nil {
			return AssessmentResult{}, err
		}
		return AssessmentResult{Status: "success", AssessedLevel: profile.CEFRCurrent, RoadmapID: id, PetState: pet, Created: false}, nil
	}

	// One slot per assessment, not per LLM call, so the retry-once path can
	// never trip the §4 limit on its own.
	if err := s.limiter.Allow(ctx, userID); err != nil {
		return AssessmentResult{}, err
	}

	if err := s.quiz.StageAnswers(ctx, userID, req.Answers, store.PlacementQuizTTL); err != nil {
		return AssessmentResult{}, err
	}

	var level string
	if err := s.routeJSON(ctx, airouter.TaskPlacementTest, PlacementSystemPrompt, PlacementUserPrompt(req.Answers), func(raw string) error {
		lvl, err := ParsePlacement(raw)
		level = lvl
		return err
	}); err != nil {
		return AssessmentResult{}, err
	}

	var roadmap airouter.Roadmap
	if err := s.routeJSON(ctx, airouter.TaskRoadmapGen, airouter.RoadmapSystemPrompt, airouter.RoadmapUserPrompt(level, req.TargetGoal, DailyMinutes), func(raw string) error {
		rm, err := airouter.ParseRoadmap(raw)
		roadmap = rm
		return err
	}); err != nil {
		return AssessmentResult{}, err
	}

	roadmapID, err := s.repo.SaveAssessment(ctx, userID, Assessment{
		CEFRLevel:        level,
		TargetGoal:       strings.TrimSpace(req.TargetGoal),
		Timezone:         req.Timezone,
		NotificationTime: req.NotificationTime,
		Roadmap:          roadmap,
	})
	if err != nil {
		return AssessmentResult{}, err
	}

	pet, err := s.pet.Ensure(ctx, userID)
	if err != nil {
		return AssessmentResult{}, err
	}
	if err := s.quiz.Clear(ctx, userID); err != nil {
		log.Printf("onboarding: clearing quiz hash for %s: %v", userID, err) // it expires anyway
	}
	return AssessmentResult{Status: "success", AssessedLevel: level, RoadmapID: roadmapID, PetState: pet, Created: true}, nil
}

// routeJSON calls the router and parses; a malformed body is retried once,
// then reported as ErrBadAIOutput. Provider/router errors are returned as-is
// (the router has already fallen back across providers).
func (s *Service) routeJSON(ctx context.Context, task airouter.TaskType, system, user string, parse func(string) error) error {
	var last error
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := s.route(ctx, task, system, user)
		if err != nil {
			return err
		}
		if err := parse(raw); err != nil {
			last = err
			log.Printf("onboarding: %s attempt %d returned a malformed body: %v", task, attempt+1, err)
			continue
		}
		return nil
	}
	if errors.Is(last, ErrBadAIOutput) {
		return last
	}
	return fmt.Errorf("%w: %v", ErrBadAIOutput, last)
}

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

func validate(req AssessmentRequest) error {
	goal := strings.TrimSpace(req.TargetGoal)
	if goal == "" || len(goal) > 255 {
		return fmt.Errorf("%w: target_goal must be 1..255 characters", ErrInvalidRequest)
	}
	if req.Timezone == "" {
		return fmt.Errorf("%w: timezone is required", ErrInvalidRequest)
	}
	if _, err := time.LoadLocation(req.Timezone); err != nil {
		return fmt.Errorf("%w: timezone %q is not an IANA zone", ErrInvalidRequest, req.Timezone)
	}
	if _, err := time.Parse("15:04:05", req.NotificationTime); err != nil {
		return fmt.Errorf("%w: notification_time must be HH:MM:SS", ErrInvalidRequest)
	}
	if len(req.Answers) == 0 {
		return fmt.Errorf("%w: answers must not be empty", ErrInvalidRequest)
	}
	seen := map[string]bool{}
	for _, a := range req.Answers {
		q, ok := Lookup(a.QuestionID)
		if !ok {
			return fmt.Errorf("%w: unknown question_id %q", ErrInvalidRequest, a.QuestionID)
		}
		if _, ok := q.Options[a.SelectedOption]; !ok {
			return fmt.Errorf("%w: %s has no option %q", ErrInvalidRequest, a.QuestionID, a.SelectedOption)
		}
		if seen[a.QuestionID] {
			return fmt.Errorf("%w: %s answered twice", ErrInvalidRequest, a.QuestionID)
		}
		seen[a.QuestionID] = true
	}
	return nil
}
