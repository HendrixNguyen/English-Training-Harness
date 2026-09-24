// Package health serves GET /healthz: a liveness probe that reports whether
// Postgres and Redis are reachable. The route is unauthenticated (Railway
// probes it), so the body names the failing dependency with a fixed marker
// and never carries the driver's error — pgx embeds `user=… database=…` and
// the host:port in it. The full error goes to the server log instead.
package health

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Pinger is anything that can be checked for reachability. store.Postgres and
// store.Redis both satisfy it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// PingTimeout is each dependency's own budget: two slow dependencies cost
// 2×PingTimeout, and neither can starve the other by spending a shared
// deadline. A var so tests can shrink it.
var PingTimeout = 2 * time.Second

// unavailable is the only thing the body ever says about a failing dependency.
const unavailable = "unavailable"

// Handler returns 200 when both dependencies answer, 503 otherwise, with
// "unavailable" against the dependency that did not.
func Handler(db, cache Pinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := gin.H{"status": "ok", "postgres": "ok", "redis": "ok"}
		healthy := true

		if err := ping(c.Request.Context(), db); err != nil {
			log.Printf("health: postgres ping failed: %v", err)
			body["postgres"] = unavailable
			healthy = false
		}
		if err := ping(c.Request.Context(), cache); err != nil {
			log.Printf("health: redis ping failed: %v", err)
			body["redis"] = unavailable
			healthy = false
		}
		if !healthy {
			body["status"] = unavailable
			c.JSON(http.StatusServiceUnavailable, body)
			return
		}
		c.JSON(http.StatusOK, body)
	}
}

func ping(parent context.Context, p Pinger) error {
	ctx, cancel := context.WithTimeout(parent, PingTimeout)
	defer cancel()
	return p.Ping(ctx)
}
