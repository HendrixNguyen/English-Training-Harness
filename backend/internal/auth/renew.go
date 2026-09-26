package auth

import (
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// RenewBelow is how much of TokenTTL may remain before Require issues a
// fresh token: half. A learner who shows up daily therefore never expires;
// 24 h of absence still does (backend spec §7, read as an idle window).
const RenewBelow = TokenTTL / 2

const (
	HeaderSessionToken     = "X-Session-Token"
	HeaderSessionExpiresIn = "X-Session-Expires-In"
)

// maybeRenew runs after the presented token has verified and matched the
// stored session. Below RenewBelow it issues a new token, stores it with the
// full TTL (the old one is invalid from that moment — the key holds one
// token) and returns it in the response headers. A store failure is logged
// and the request proceeds on the old token.
func maybeRenew(c *gin.Context, tokens *TokenIssuer, sessions SessionStore, userID string, exp time.Time) {
	if exp.Sub(tokens.now()) >= RenewBelow {
		return
	}
	fresh, err := tokens.Issue(userID)
	if err != nil {
		log.Printf("auth: renewing session for user=%s: issuing token: %v", userID, err)
		return
	}
	if err := sessions.Put(c.Request.Context(), userID, fresh, TokenTTL); err != nil {
		log.Printf("auth: renewing session for user=%s: storing token: %v", userID, err)
		return
	}
	c.Header(HeaderSessionToken, fresh)
	c.Header(HeaderSessionExpiresIn, strconv.Itoa(int(TokenTTL/time.Second)))
}
