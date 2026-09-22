package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ContextUserID is the Gin context key holding the authenticated user's UUID.
const ContextUserID = "user_id"

// Require verifies the bearer token and that the matching Redis session is
// still present. Every per-user route in spec §7 mounts behind it.
//
// A token is accepted only when it (a) verifies against JWT_SECRET, (b) is
// within its exp window, and (c) is byte-for-byte the token stored at
// sess:{user_id}:token. (c) is what makes DEL a revocation and what makes a new
// sign-in supersede the previous token.
func Require(tokens *TokenIssuer, sessions SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c.GetHeader("Authorization"))
		if raw == "" {
			abortUnauthorized(c)
			return
		}
		userID, err := tokens.Verify(raw)
		if err != nil {
			abortUnauthorized(c)
			return
		}
		stored, err := sessions.Get(c.Request.Context(), userID)
		if err != nil || stored != raw {
			abortUnauthorized(c)
			return
		}
		c.Set(ContextUserID, userID)
		c.Next()
	}
}

// UserID returns the authenticated user set by Require, or "" outside it.
func UserID(c *gin.Context) string {
	v, ok := c.Get(ContextUserID)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func abortUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
}
