package google

import (
	"context"
	"errors"
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
		var up *UpstreamError
		switch {
		case errors.Is(err, ErrReauthRequired):
			// The refresh token is gone or revoked: the client sends the user
			// back through /login (auth.AuthCodeURL re-requests consent).
			c.JSON(http.StatusConflict, gin.H{"error": "reauth_required"})
		case errors.As(err, &up), errors.Is(err, context.DeadlineExceeded):
			c.JSON(http.StatusBadGateway, gin.H{"error": "google_unavailable"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			c.JSON(http.StatusOK, res)
		}
	}
}
