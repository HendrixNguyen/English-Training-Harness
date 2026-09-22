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
