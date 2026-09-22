package quests

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// DailyHandler serves GET /api/v1/quests/daily (backend spec §6.2). It must be
// mounted behind auth.Require().
func DailyHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		suite, err := svc.Daily(c.Request.Context(), userID)
		if errors.Is(err, ErrNoActiveRoadmap) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no_active_roadmap"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		c.JSON(http.StatusOK, suite)
	}
}

// progressRequest is the backend spec §6.2 POST /api/v1/quests/progress body.
type progressRequest struct {
	ExerciseID      string `json:"exercise_id" binding:"required"`
	DurationSeconds int64  `json:"duration_seconds" binding:"required,gt=0"`
	// UserAnswers is part of the §6.2 request and is accepted so a
	// spec-conformant client is never rejected, but it is not persisted: no
	// §3.2 table stores answers and §6.2 does not say what becomes of them.
	// See the plan's Reconciliation section.
	UserAnswers json.RawMessage `json:"user_answers"`
}

// ProgressHandler serves POST /api/v1/quests/progress. It must be mounted
// behind auth.Require().
func ProgressHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req progressRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}

		out, err := svc.RecordProgress(c.Request.Context(), userID, req.ExerciseID, req.DurationSeconds)
		switch {
		case errors.Is(err, ErrNoActiveRoadmap):
			c.JSON(http.StatusNotFound, gin.H{"error": "no_active_roadmap"})
		case errors.Is(err, ErrExerciseNotFound):
			// 404 rather than 403 so other users' exercise ids stay unprobeable.
			c.JSON(http.StatusNotFound, gin.H{"error": "exercise_not_found"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			c.JSON(http.StatusOK, out)
		}
	}
}
