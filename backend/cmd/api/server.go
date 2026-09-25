package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
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
// to google.SyncTimeout (60 s) and an onboarding assessment runs each
// airouter.Route call under airouter.TaskTimeout (180 s for the roadmap,
// 30 s otherwise, one malformed-body retry each); a server-wide write
// deadline would cut those responses off mid-flight.
func newServer(h http.Handler) *http.Server {
	return &http.Server{Handler: h, ReadHeaderTimeout: ReadHeaderTimeout, IdleTimeout: IdleTimeout}
}

// serve runs srv on ln until ctx is done, then drains it within ShutdownGrace.
// It returns nil on a clean shutdown (Serve's own http.ErrServerClosed is the
// normal exit, not an error) and the listener error otherwise, so main can
// return — letting its deferred pg/rdb Close calls run — instead of Fatalf-ing.
func serve(ctx context.Context, srv *http.Server, ln net.Listener) error {
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

	log.Printf("shutting down: draining in-flight requests for up to %s", ShutdownGrace)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownGrace)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close() // grace exceeded: close what is left so the process still exits
		return fmt.Errorf("shutdown: %w", err)
	}
	<-errc // Serve has returned http.ErrServerClosed
	return nil
}
