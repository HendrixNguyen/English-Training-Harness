package main

import (
	"context"
	"net"
	"net/http"
	"sync"
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

// The signal arrives while a request is in flight: serve must stop accepting,
// keep running until that request finishes, then return nil — the shape a
// Railway SIGTERM produces. The ordering is the point: serve returning lets
// main run pg.Close()/rdb.Close() and exit, so it must not return while a
// handler is still working. The handler blocks on release so the test can
// observe the drain window instead of racing a sleep.
func TestServeStopsOnContextCancelAndDrainsInFlightRequests(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock) // a failing assertion must not leave the handler wedged
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
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

	// Drain window: the handler is still blocked, so serve must still be running…
	select {
	case err := <-done:
		t.Fatalf("serve returned (%v) while a request was still in flight", err)
	case <-time.After(200 * time.Millisecond):
	}
	// …but new connections must already be refused (Shutdown closes listeners first).
	if c, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second); err == nil {
		_ = c.Close()
		t.Fatal("listener still accepting new connections during the drain")
	}

	unblock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v, want nil on a clean shutdown", err)
		}
	case <-time.After(ShutdownGrace + time.Second):
		t.Fatal("serve did not return after the in-flight request finished")
	}
	if got := <-code; got != http.StatusOK {
		t.Fatalf("in-flight request got %d, want 200 (drained, not cut off)", got)
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
