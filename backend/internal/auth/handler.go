package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// signInRequest is the §7 POST /api/v1/auth/google body.
type signInRequest struct {
	Code        string `json:"code" binding:"required"`
	RedirectURI string `json:"redirect_uri" binding:"required"`
}

// Handler serves POST /api/v1/auth/google. Any failure on Google's side is a
// 401: the client's only sensible response is to restart the consent flow.
func Handler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req signInRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}

		out, err := svc.SignIn(c.Request.Context(), req.Code, req.RedirectURI)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "google_auth_failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": out.Token,
			"user": gin.H{
				"id":           out.User.ID,
				"email":        out.User.Email,
				"full_name":    out.User.FullName,
				"cefr_current": out.User.CEFRCurrent,
			},
		})
	}
}
