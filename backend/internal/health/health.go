// Package health serves GET /healthz: a liveness probe that reports whether
// Postgres and Redis are reachable.
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Pinger is anything that can be checked for reachability. store.Postgres and
// store.Redis both satisfy it.
type Pinger interface {
	Ping(ctx context.Context) error
}

const pingTimeout = 2 * time.Second

// Handler returns 200 when both dependencies answer, 503 otherwise, naming the
// failing dependency in the body.
func Handler(db, cache Pinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), pingTimeout)
		defer cancel()

		body := gin.H{"status": "ok", "postgres": "ok", "redis": "ok"}
		healthy := true

		if err := db.Ping(ctx); err != nil {
			body["postgres"] = err.Error()
			healthy = false
		}
		if err := cache.Ping(ctx); err != nil {
			body["redis"] = err.Error()
			healthy = false
		}
		if !healthy {
			body["status"] = "unavailable"
			c.JSON(http.StatusServiceUnavailable, body)
			return
		}
		c.JSON(http.StatusOK, body)
	}
}
