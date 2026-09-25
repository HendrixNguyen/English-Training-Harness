package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	// ShutdownGrace bounds srv.Shutdown once SIGINT/SIGTERM arrives: in-flight
	// requests get this long to finish before the process exits. Railway sends
	// SIGTERM and kills after its own grace window, so this stays under 10 s.
	ShutdownGrace = 8 * time.Second
	// ReadHeaderTimeout caps how long a client may take to send request
	// headers (slowloris). Request bodies are bounded per handler, not here.
	ReadHeaderTimeout = 10 * time.Second
	// IdleTimeout closes keep-alive connections that sit idle.
	IdleTimeout = 120 * time.Second
)

// newServer wraps the router in an http.Server that main can shut down.
// WriteTimeout is deliberately unset: POST /integrations/google/sync runs up
// to google.SyncTimeout (60 s) and an onboarding assessment may spend several
// airouter.Route calls of up to len(FallbackOrder) × ProviderTimeout each; a
// server-wide write deadline would cut those responses off mid-flight.
func newServer(h http.Handler) *http.Server {
	return &http.Server{Handler: h, ReadHeaderTimeout: ReadHeaderTimeout, IdleTimeout: IdleTimeout}
}

// ErrDrainTimedOut is serve's answer when in-flight requests outlive the grace:
// the remaining connections were force-closed. main logs it and returns
// normally (exit 0 — the stop was planned; the log line is the signal), so the
// deferred pg/rdb Close calls run, unlike a log.Fatalf.
var ErrDrainTimedOut = errors.New("server: drain grace exceeded; remaining connections were closed")

// serve runs srv on ln until ctx is done, then drains it within grace.
// It returns nil on a clean shutdown (Serve's own http.ErrServerClosed is the
// normal exit, not an error), ErrDrainTimedOut if the grace ran out before the
// drain finished, and the listener error otherwise, so main can return —
// letting its deferred pg/rdb Close calls run — instead of Fatalf-ing.
func serve(ctx context.Context, srv *http.Server, ln net.Listener, grace time.Duration) error {
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	log.Printf("shutting down: draining in-flight requests for up to %s (press Ctrl-C again to force)", grace)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close() // grace exceeded: close what is left so the process still exits
		<-errc          // Serve has returned; do not leak its goroutine
		return fmt.Errorf("%w: %v", ErrDrainTimedOut, err)
	}
	<-errc // Serve has returned http.ErrServerClosed
	return nil
}

// waitWithin waits for wg up to d and reports whether it finished. main uses it
// to join the background workers after the drain without letting a stuck
// worker hold the process past Railway's kill window.
func waitWithin(wg *sync.WaitGroup, d time.Duration) bool {
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		return true
	case <-time.After(d):
		return false
	}
}
