package pet

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// statusResponse is the GET /api/v1/pet/status 200 body — backend spec §6.3,
// field for field. last_practiced_at is null until the first target is met.
type statusResponse struct {
	PlantName       string     `json:"plant_name"`
	Stage           string     `json:"stage"`
	HealthPoints    int        `json:"health_points"`
	CurrentStreak   int        `json:"current_streak"`
	LastPracticedAt *time.Time `json:"last_practiced_at"`
}

func toStatus(s State) statusResponse {
	out := statusResponse{PlantName: s.PlantName, Stage: s.Stage, HealthPoints: s.HealthPoints, CurrentStreak: s.CurrentStreak}
	if s.LastPracticedAt != nil {
		u := s.LastPracticedAt.UTC()
		out.LastPracticedAt = &u
	}
	return out
}

// reviveRequest is the §6.3 POST /api/v1/pet/revive body. answers is accepted
// so a spec-conformant client is never rejected, but neither spec says how it
// is graded, so it is not read — the pass condition is 15 minutes of recorded
// study (see Service.Revive and the plan's open questions).
type reviveRequest struct {
	Answers json.RawMessage `json:"answers"`
}

// reviveResponse is the §6.3 200 body.
type reviveResponse struct {
	RevivalPassed bool `json:"revival_passed"`
	PetState      struct {
		HealthPoints  int    `json:"health_points"`
		Stage         string `json:"stage"`
		CurrentStreak int    `json:"current_streak"`
	} `json:"pet_state"`
}

// StatusHandler serves GET /api/v1/pet/status. Mount behind auth.Require().
func StatusHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		st, err := svc.Ensure(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		c.JSON(http.StatusOK, toStatus(st))
	}
}

// ReviveHandler serves POST /api/v1/pet/revive. Mount behind auth.Require().
func ReviveHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if c.Request.ContentLength != 0 {
			var req reviveRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
				return
			}
		}

		out, err := svc.Revive(c.Request.Context(), userID)
		switch {
		case errors.Is(err, ErrNotWilted):
			c.JSON(http.StatusConflict, gin.H{"error": "pet_not_wilted"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			var resp reviveResponse
			resp.RevivalPassed = out.Passed
			resp.PetState.HealthPoints = out.State.HealthPoints
			resp.PetState.Stage = out.State.Stage
			resp.PetState.CurrentStreak = out.State.CurrentStreak
			c.JSON(http.StatusOK, resp)
		}
	}
}
