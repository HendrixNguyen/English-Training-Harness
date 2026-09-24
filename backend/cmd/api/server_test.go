package main

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

// The signal arrives while a request is in flight: serve must let it finish,
// close the listener, and return nil — the shape a Railway SIGTERM produces.
func TestServeStopsOnContextCancelAndDrainsInFlightRequests(t *testing.T) {
	started := make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(300 * time.Millisecond) // well under ShutdownGrace
		w.WriteHeader(http.StatusOK)
	})
	ln := listen(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, newServer(h), ln) }()

	code := make(chan int, 1)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String() + "/")
		if err != nil {
			code <- -1
			return
		}
		_ = resp.Body.Close()
		code <- resp.StatusCode
	}()

	<-started
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v, want nil on a clean shutdown", err)
		}
	case <-time.After(ShutdownGrace + time.Second):
		t.Fatal("serve did not return after the context was cancelled")
	}
	if got := <-code; got != http.StatusOK {
		t.Fatalf("in-flight request got %d, want 200 (drained, not cut off)", got)
	}
	if _, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second); err == nil {
		t.Fatal("listener still accepting connections after shutdown")
	}
}

func TestServeReturnsAListenerError(t *testing.T) {
	ln := listen(t)
	_ = ln.Close() // Serve on a closed listener fails at once
	if err := serve(context.Background(), newServer(http.NotFoundHandler()), ln); err == nil {
		t.Fatal("serve returned nil for a dead listener")
	}
}

// WriteTimeout must stay unset: google.SyncTimeout (60 s) and onboarding's AI
// calls legitimately outlive any sane server-wide write deadline.
func TestNewServerBoundsHeaderReadsButNotWrites(t *testing.T) {
	srv := newServer(http.NotFoundHandler())
	if srv.ReadHeaderTimeout <= 0 || srv.IdleTimeout <= 0 {
		t.Fatalf("ReadHeaderTimeout/IdleTimeout unset: %+v", srv)
	}
	if srv.WriteTimeout != 0 {
		t.Fatalf("WriteTimeout = %s, want 0 (see newServer doc)", srv.WriteTimeout)
	}
}
