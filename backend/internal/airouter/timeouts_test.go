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
