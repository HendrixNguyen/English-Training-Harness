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
