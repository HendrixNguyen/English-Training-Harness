package notify

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunWorkerTicksAndStopsWhenTheContextIsCancelled(t *testing.T) {
	h := dueHarness()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		RunWorker(ctx, h.svc, 5*time.Millisecond)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for {
		if len(h.sender.Sent()) >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("worker never sent; calls: %v", h.log.calls)
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunWorker did not return after cancel")
	}
	if !strings.Contains(strings.Join(h.log.calls, ";"), "queue.Due(") {
		t.Error("the worker never asked the queue")
	}
}
