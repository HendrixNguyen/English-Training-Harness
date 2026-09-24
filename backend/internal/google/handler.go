package google

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// SyncTimeout bounds one sync: a token refresh, one Calendar call and up to
// 30 Tasks calls, sequential. §6.4 says "asynchronously" but its 200 body
// carries the ids Google returns, so the work runs inside the request.
const SyncTimeout = 60 * time.Second

// SyncHandler serves POST /api/v1/integrations/google/sync (backend spec
// §6.4). It must be mounted behind auth.Require().
func SyncHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), SyncTimeout)
		defer cancel()

		res, err := svc.Sync(ctx, userID)
		if err != nil {
			logSyncFailure(userID, err)
		}
		var up *UpstreamError
		switch {
		case errors.Is(err, ErrReauthRequired):
			// The refresh token is gone or revoked, or Google rejected the
			// scopes: the client sends the user back through /login
			// (auth.AuthCodeURL re-requests consent).
			c.JSON(http.StatusConflict, gin.H{"error": "reauth_required"})
		case errors.As(err, &up), errors.Is(err, context.DeadlineExceeded), errors.Is(err, ErrAlreadyExists):
			// Quota/throttle, 5xx, our 60 s deadline, or a 409 the insert path
			// did not consume (a concurrent patch, any Tasks conflict): Google
			// was the problem and a retry is the answer — 502, never 500.
			c.JSON(http.StatusBadGateway, gin.H{"error": "google_unavailable"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			log.Printf("google: sync user=%s ok event=%s tasks=%d", userID, res.CalendarEventID, res.TasksCreatedCount)
			c.JSON(http.StatusOK, res)
		}
	}
}

// logSyncFailure records why a sync failed, server-side only. For an
// UpstreamError that is Google's own reason (service, status, body) — the
// single most useful line when a user reports "sync does nothing". Tokens
// are never part of any error value in this package (token.go, oauth.go);
// TestSyncHandlerLogsTheFailureServerSideOnly keeps it that way.
func logSyncFailure(userID string, err error) {
	var up *UpstreamError
	if errors.As(err, &up) {
		log.Printf("google: sync user=%s failed: %s returned %d: %s", userID, up.Service, up.Status, truncate(up.Body, 512))
		return
	}
	log.Printf("google: sync user=%s failed: %v", userID, err)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
