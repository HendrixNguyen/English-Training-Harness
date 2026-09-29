package airouter

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
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

// hang is an LLMProvider that blocks until its context is done, simulating a
// preferred provider that never fails fast. It never widens its own budget:
// it only reports what ctx gave it.
type hang struct {
	mu    sync.Mutex
	calls int
}

func (h *hang) GenerateContent(ctx context.Context, _, _ string) (string, error) {
	h.mu.Lock()
	h.calls++
	h.mu.Unlock()
	<-ctx.Done()
	return "", ctx.Err()
}

func (h *hang) callCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.calls
}

// deadlineProbe is an LLMProvider that records how much time was left on each
// context it was handed, then returns a fixed out/err without doing any real
// work — so the 60s/90s/180s split can be asserted in microseconds.
type deadlineProbe struct {
	mu   sync.Mutex
	seen []time.Duration
	out  string
	err  error
}

func (d *deadlineProbe) GenerateContent(ctx context.Context, _, _ string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	dl, ok := ctx.Deadline()
	if !ok {
		d.seen = append(d.seen, -1)
	} else {
		d.seen = append(d.seen, time.Until(dl))
	}
	return d.out, d.err
}

func (d *deadlineProbe) lastSeen() time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.seen) == 0 {
		return -1
	}
	return d.seen[len(d.seen)-1]
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
	logs := captureLog(t)
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
	got := logs.String()
	if !strings.Contains(got, "gemini failed for task roadmap_generation after") || !strings.Contains(got, "openai failed for task roadmap_generation after") {
		t.Errorf("log missing per-provider failure lines:\n%s", got)
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

func TestRouteCutsAHangingPreferredProviderSoTheFallbackStillAnswers(t *testing.T) {
	gemini := &hang{}
	openai := &scripted{out: "o"}
	r := NewRouterWithProviders(map[ProviderType]LLMProvider{ProviderGemini: gemini, ProviderOpenAI: openai})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	started := time.Now()
	out, err := r.Route(ctx, TaskPlacementTest, "s", "u")
	elapsed := time.Since(started)

	if err != nil || out != "o" {
		t.Fatalf("Route = %q, %v; want the openai fallback to answer", out, err)
	}
	if gemini.callCount() != 1 {
		t.Errorf("gemini.calls = %d, want 1", gemini.callCount())
	}
	if len(openai.calls) != 1 {
		t.Errorf("openai.calls = %d, want 1", len(openai.calls))
	}
	if elapsed >= 200*time.Millisecond {
		t.Errorf("elapsed = %s, want well inside the 200ms caller deadline", elapsed)
	}
}

func TestRouteSplitsTheRemainingBudgetAcrossTheProvidersLeft(t *testing.T) {
	within := func(t *testing.T, got, want time.Duration) {
		t.Helper()
		diff := got - want
		if diff < 0 {
			diff = -diff
		}
		if diff > time.Second {
			t.Errorf("saw %s, want %s ± 1s", got, want)
		}
	}

	t.Run("three configured providers", func(t *testing.T) {
		gemini := &deadlineProbe{err: errors.New("down")}
		openai := &deadlineProbe{err: errors.New("down")}
		deepseek := &deadlineProbe{out: "d"}
		r := NewRouterWithProviders(map[ProviderType]LLMProvider{
			ProviderGemini: gemini, ProviderOpenAI: openai, ProviderDeepSeek: deepseek,
		})
		out, err := r.Route(context.Background(), TaskRoadmapGen, "s", "u")
		if err != nil || out != "d" {
			t.Fatalf("Route = %q, %v; want deepseek's answer", out, err)
		}
		within(t, gemini.lastSeen(), 60*time.Second)
		within(t, openai.lastSeen(), 90*time.Second)
		within(t, deepseek.lastSeen(), 180*time.Second)
	})

	t.Run("only gemini configured", func(t *testing.T) {
		gemini := &deadlineProbe{out: "g"}
		r := NewRouterWithProviders(map[ProviderType]LLMProvider{ProviderGemini: gemini})
		out, err := r.Route(context.Background(), TaskRoadmapGen, "s", "u")
		if err != nil || out != "g" {
			t.Fatalf("Route = %q, %v; want gemini's answer", out, err)
		}
		within(t, gemini.lastSeen(), 180*time.Second)
	})

	t.Run("gemini and deepseek configured, caller deadline 10s", func(t *testing.T) {
		gemini := &deadlineProbe{out: "g"}
		deepseek := &deadlineProbe{out: "d"}
		r := NewRouterWithProviders(map[ProviderType]LLMProvider{ProviderGemini: gemini, ProviderDeepSeek: deepseek})
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		out, err := r.Route(ctx, TaskRoadmapGen, "s", "u")
		if err != nil || out != "g" {
			t.Fatalf("Route = %q, %v; want gemini's answer", out, err)
		}
		within(t, gemini.lastSeen(), 5*time.Second)
	})
}

func TestRouteReportsDeadlineExceededWhenEveryAttemptTimesOut(t *testing.T) {
	gemini := &hang{}
	openai := &hang{}
	r := NewRouterWithProviders(map[ProviderType]LLMProvider{ProviderGemini: gemini, ProviderOpenAI: openai})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := r.Route(ctx, TaskPlacementTest, "s", "u")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if err != context.DeadlineExceeded && !errors.Is(err, ErrAllProvidersFailed) {
		t.Errorf("err = %v, want it to also satisfy ErrAllProvidersFailed unless it is bare DeadlineExceeded", err)
	}
	if gemini.callCount() != 1 || openai.callCount() != 1 {
		t.Errorf("calls: gemini=%d openai=%d, want 1 and 1", gemini.callCount(), openai.callCount())
	}
}
