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
