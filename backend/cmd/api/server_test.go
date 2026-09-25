package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	go func() { done <- serve(ctx, newServer(h), ln, ShutdownGrace) }()

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
	if err := serve(context.Background(), newServer(http.NotFoundHandler()), ln, ShutdownGrace); err == nil {
		t.Fatal("serve returned nil for a dead listener")
	}
}

// A redeploy during a long-running request (e.g. a 60 s google.SyncTimeout
// call) must not hang the process forever: serve force-closes what remains
// once the grace runs out and returns ErrDrainTimedOut instead of the fatal
// generic error the fast path used to return, so main can log it and still
// run the deferred pg/rdb Close calls (exit 0 — the stop was planned).
func TestServeReturnsErrDrainTimedOutAndClosesTheStragglerWhenTheGraceRunsOut(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusOK)
	})
	ln := listen(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, newServer(h), ln, 100*time.Millisecond) }()

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
		if !errors.Is(err, ErrDrainTimedOut) {
			t.Fatalf("serve returned %v, want ErrDrainTimedOut", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not return after the grace ran out")
	}
	// The straggler was force-closed: the client sees an error, not a 200.
	if got := <-code; got != -1 {
		t.Fatalf("in-flight request got %d, want a transport error after srv.Close()", got)
	}
	unblock()
}

// WriteTimeout must stay unset: google.SyncTimeout (60 s) and onboarding's AI
// calls legitimately outlive any sane server-wide write deadline.
func TestNewServerBoundsReadsButNotWrites(t *testing.T) {
	srv := newServer(http.NotFoundHandler())
	if srv.ReadHeaderTimeout <= 0 || srv.IdleTimeout <= 0 {
		t.Fatalf("ReadHeaderTimeout/IdleTimeout unset: %+v", srv)
	}
	if srv.ReadTimeout != ReadTimeout || ReadTimeout <= 0 {
		t.Fatalf("ReadTimeout = %s, want %s (see newServer doc)", srv.ReadTimeout, ReadTimeout)
	}
	if srv.WriteTimeout != 0 {
		t.Fatalf("WriteTimeout = %s, want 0 (see newServer doc)", srv.WriteTimeout)
	}
}

// A client that sends headers promptly and then trickles a body (or never
// finishes it) must not hold a goroutine indefinitely: POST /api/v1/auth/google
// needs no token, so anyone can open one. ReadTimeout bounds the whole read —
// headers and body — so the handler's blocked r.Body.Read unblocks with an
// i/o timeout once the deadline elapses, instead of waiting forever for the
// rest of the body.
//
// Deviation from the plan's literal assertion: ReadTimeout only arms
// net/http's *read* deadline (net/http/server.go readRequest:
// `c.rwc.SetReadDeadline(wholeReqDeadline)`), never a write deadline. Verified
// against go1.27.1: once the deadline fires, io.ReadAll(r.Body) returns an
// error and the handler's goroutine is freed (the actual bug this task
// fixes), but the connection itself is not force-closed — if the handler
// still writes a response (as this test's handler does, matching a real
// gin handler that would reply 400), the client receives it normally (with
// `Connection: close` appended) rather than a bare transport error. So the
// observable, deadline-independent guarantee is that the handler's read
// unblocks — that is what this test asserts, instead of the plan's "read =
// transport error, not 200" (a stdlib behavior the plan's snippet assumed
// but net/http does not provide with ReadTimeout alone).
func TestASlowBodyIsCutOffAtReadTimeout(t *testing.T) {
	readErr := make(chan error, 1)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		readErr <- err
		w.WriteHeader(http.StatusOK)
	})
	ln := listen(t)
	srv := newServer(h)
	srv.ReadTimeout = 200 * time.Millisecond // production is 30 s; the deadline is what is under test
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = serve(ctx, srv, ln, ShutdownGrace) }()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprint(conn, "POST /x HTTP/1.1\r\nHost: x\r\nContent-Length: 100\r\n\r\n{") // headers + 1 byte, then nothing

	select {
	case err := <-readErr:
		if err == nil {
			t.Fatal("handler's body read returned nil, want an i/o timeout once ReadTimeout elapses")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler's body read did not unblock within 2s of a 200ms ReadTimeout; the goroutine is held indefinitely")
	}
}

func TestWaitWithinReturnsTrueWhenTheGroupFinishes(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done() }()
	start := time.Now()
	if !waitWithin(&wg, time.Second) {
		t.Fatal("waitWithin returned false, want true when the group finishes")
	}
	if time.Since(start) >= time.Second {
		t.Fatalf("waitWithin took %s, want well under the 1 s bound", time.Since(start))
	}
}

func TestWaitWithinReturnsFalseWhenItDoesNot(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	blocked := make(chan struct{})
	t.Cleanup(func() { close(blocked) })
	go func() { defer wg.Done(); <-blocked }()
	if waitWithin(&wg, 50*time.Millisecond) {
		t.Fatal("waitWithin returned true, want false when the group is still running")
	}
}
