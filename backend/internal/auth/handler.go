package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// signInRequest is the §7 POST /api/v1/auth/google body.
type signInRequest struct {
	Code        string `json:"code" binding:"required"`
	RedirectURI string `json:"redirect_uri" binding:"required"`
}

// signInResponse is the backend spec §6.1 200 body for POST /api/v1/auth/google.
// expires_in is derived from TokenTTL so it can never disagree with the JWT exp
// or the Redis session TTL; token_type is always "Bearer" — the value auth.Require
// expects in the Authorization header.
type signInResponse struct {
	AccessToken string     `json:"access_token"`
	TokenType   string     `json:"token_type"`
	ExpiresIn   int        `json:"expires_in"`
	User        signInUser `json:"user"`
}

type signInUser struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	FullName    string `json:"full_name"`
	CEFRCurrent string `json:"cefr_current"`
}

// Handler serves POST /api/v1/auth/google (backend spec §6.1). Any failure on
// Google's side is a 401: the client's only sensible response is to restart the
// consent flow.
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

		c.JSON(http.StatusOK, signInResponse{
			AccessToken: out.Token,
			TokenType:   "Bearer",
			ExpiresIn:   int(TokenTTL / time.Second),
			User: signInUser{
				ID:          out.User.ID,
				Email:       out.User.Email,
				FullName:    out.User.FullName,
				CEFRCurrent: out.User.CEFRCurrent,
			},
		})
	}
}
