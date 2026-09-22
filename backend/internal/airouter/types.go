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
