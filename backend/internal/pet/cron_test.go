package pet

import (
	"context"
	"testing"
	"time"
)

func TestNextTopOfHour(t *testing.T) {
	tests := []struct {
		now, want time.Time
	}{
		{time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC), time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC)},
		{time.Date(2026, 9, 22, 10, 0, 0, 1, time.UTC), time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC)},
		{time.Date(2026, 9, 22, 10, 59, 59, 0, time.UTC), time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC)},
		{time.Date(2026, 9, 22, 23, 30, 0, 0, time.UTC), time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		if got := NextTopOfHour(tt.now); !got.Equal(tt.want) {
			t.Errorf("NextTopOfHour(%v) = %v, want %v", tt.now, got, tt.want)
		}
	}
}

func TestRunHourlyStopsWhenTheContextIsCancelled(t *testing.T) {
	h := newHarness(sept22)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		RunHourly(ctx, h.svc)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunHourly did not return after cancel")
	}
}
