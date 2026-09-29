package onboarding

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// QuizHandler serves GET /api/v1/onboarding/quiz — the placement items
// without answers. Not in either spec (see CODEMAP); mount behind
// auth.Require() so the bank is not public.
func QuizHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if auth.UserID(c) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusOK, PublicBank())
	}
}

// AssessmentHandler serves POST /api/v1/onboarding/assessment (backend spec
// §6.1): 201 on a new roadmap, 200 when one was already active. Mount behind
// auth.Require().
func AssessmentHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var req AssessmentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}

		out, err := svc.Assess(c.Request.Context(), userID, req)
		switch {
		case errors.Is(err, ErrInvalidRequest):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		case errors.Is(err, airouter.ErrRateLimited):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
		case errors.Is(err, airouter.ErrNoProviders):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ai_unavailable"})
		case errors.Is(err, ErrBadAIOutput):
			c.JSON(http.StatusBadGateway, gin.H{"error": "ai_bad_output"})
		case errors.Is(err, ErrAITimeout):
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "ai_timeout"})
		case errors.Is(err, airouter.ErrAllProvidersFailed):
			c.JSON(http.StatusBadGateway, gin.H{"error": "ai_upstream_failed"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		case out.Created:
			c.JSON(http.StatusCreated, out)
		default:
			c.JSON(http.StatusOK, out)
		}
	}
}

// RegenerateHandler serves POST /api/v1/roadmaps/regenerate (backend spec
// §6.1.3, added by this plan): 201 with a fresh roadmap. Mount behind
// auth.Require(). The body is optional — an absent or empty body keeps the
// current CEFR level — so ShouldBindJSON only runs when one was sent.
func RegenerateHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var req RegenerateRequest
		if c.Request.ContentLength != 0 {
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
				return
			}
		}

		out, err := svc.Regenerate(c.Request.Context(), userID, req)
		switch {
		case errors.Is(err, ErrInvalidRequest):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		case errors.Is(err, ErrNoActiveRoadmap):
			c.JSON(http.StatusNotFound, gin.H{"error": "no_active_roadmap"})
		case errors.Is(err, airouter.ErrRateLimited):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
		case errors.Is(err, airouter.ErrNoProviders):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ai_unavailable"})
		case errors.Is(err, ErrBadAIOutput):
			c.JSON(http.StatusBadGateway, gin.H{"error": "ai_bad_output"})
		case errors.Is(err, ErrAITimeout):
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "ai_timeout"})
		case errors.Is(err, airouter.ErrAllProvidersFailed):
			c.JSON(http.StatusBadGateway, gin.H{"error": "ai_upstream_failed"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			c.JSON(http.StatusCreated, out)
		}
	}
}
