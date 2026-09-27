package auth

import (
	"errors"
	"log"
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

// Handler serves POST /api/v1/auth/google (backend spec §6.1). Google
// rejected → 401; the email belongs to another account → 409; the session
// store is down → 503; anything else → 500. Every failure is logged once.
func Handler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req signInRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}

		out, err := svc.SignIn(c.Request.Context(), req.Code, req.RedirectURI)
		if err != nil {
			log.Printf("auth: sign-in failed: %v", err) // never a token: see errors.go and the handler test
			switch {
			case errors.Is(err, ErrGoogleRejected):
				c.JSON(http.StatusUnauthorized, gin.H{"error": "google_auth_failed"})
			case errors.Is(err, ErrEmailTaken):
				c.JSON(http.StatusConflict, gin.H{"error": "email_in_use"})
			case errors.Is(err, ErrSessionStoreUnavailable):
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "unavailable"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			}
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
