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
	// included: Route splits what is left of it evenly over the configured
	// providers not yet tried (attemptBudget), so room for the fallback is
	// enforced per attempt rather than assumed.
	RoadmapTimeout = 180 * time.Second
	// ExerciseTimeout bounds one Route call for TaskExerciseGen: one task's
	// typed content is ≤ ~1.5k output tokens (TestExerciseAnswerFitsTheBudget),
	// and 30 s left no headroom at production's ~42 tok/s (2026-09-25).
	ExerciseTimeout = 60 * time.Second
	// DefaultTaskTimeout is §6.2's 30 s for every other task.
	DefaultTaskTimeout = 30 * time.Second
)

// attemptBudget is one provider's share of what is left of the Route budget:
// remaining split evenly over the configured providers not yet tried (this
// one included), so a preferred provider that hangs cannot spend the
// fallbacks' time. With one provider left it is everything, so a
// single-provider router behaves exactly as before.
func attemptBudget(remaining time.Duration, providersLeft int) time.Duration {
	if providersLeft <= 1 {
		return remaining
	}
	return remaining / time.Duration(providersLeft)
}

// TaskTimeout is the budget for one Route call of task.
func TaskTimeout(task TaskType) time.Duration {
	switch task {
	case TaskRoadmapGen:
		return RoadmapTimeout
	case TaskExerciseGen:
		return ExerciseTimeout
	default:
		return DefaultTaskTimeout
	}
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
