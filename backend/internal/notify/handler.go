package notify

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// settingsRequest is the backend spec §6.4 POST /settings/notifications body:
// notification_time plus a FLAT push_subscription {endpoint, p256dh, auth}
// (not the browser's nested `keys` object — the PWA flattens it). timezone is
// additive (§3.2 users.timezone; optional).
type settingsRequest struct {
	NotificationTime string   `json:"notification_time" binding:"required"`
	Timezone         string   `json:"timezone"`
	PushSubscription *pushSub `json:"push_subscription"`
}

type pushSub struct {
	Endpoint string `json:"endpoint" binding:"required"`
	P256dh   string `json:"p256dh" binding:"required"`
	Auth     string `json:"auth" binding:"required"`
}

// SettingsHandler serves POST /api/v1/settings/notifications. It must be
// mounted behind auth.Require().
func SettingsHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var req settingsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}
		in := SettingsRequest{NotificationTime: req.NotificationTime, Timezone: req.Timezone}
		if req.PushSubscription != nil {
			in.Subscription = &Subscription{Endpoint: req.PushSubscription.Endpoint, P256dh: req.PushSubscription.P256dh, Auth: req.PushSubscription.Auth}
		}

		res, err := svc.UpdateSettings(c.Request.Context(), userID, in)
		switch {
		case errors.Is(err, ErrInvalidRequest):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		case errors.Is(err, ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			c.JSON(http.StatusOK, res)
		}
	}
}
