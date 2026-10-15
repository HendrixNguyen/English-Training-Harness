package notify

import (
	"context"
	"errors"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// SavedSettings is preference readback. NextReminderAt is prospective, not a
// delivery receipt. PushConfigured reports server sender wiring, not device
// permission, subscription, or Google synchronization.
type SavedSettings struct {
	NotificationTime string `json:"notification_time"`
	Timezone         string `json:"timezone"`
	NextReminderAt   string `json:"next_reminder_at"`
	PushConfigured   bool   `json:"push_configured"`
}

func (s *Service) ReadSettings(ctx context.Context, owner string) (SavedSettings, error) {
	prefs, err := s.repo.Preferences(ctx, owner)
	if err != nil {
		return SavedSettings{}, err
	}
	clock, err := NormalizeClock(prefs.NotificationTime)
	if err != nil {
		return SavedSettings{}, err
	}
	loc := Location(prefs.Timezone)
	next, err := NextSendTime(s.now(), clock, loc)
	if err != nil {
		return SavedSettings{}, err
	}
	return SavedSettings{NotificationTime: clock, Timezone: loc.String(), NextReminderAt: next.UTC().Format(time.RFC3339), PushConfigured: s.sender != nil}, nil
}

// ReadSettingsHandler serves GET /api/v1/settings behind auth.Require().
func ReadSettingsHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		owner := auth.UserID(c)
		if owner == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		out, err := svc.ReadSettings(c.Request.Context(), owner)
		switch {
		case errors.Is(err, ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			c.JSON(http.StatusOK, out)
		}
	}
}
