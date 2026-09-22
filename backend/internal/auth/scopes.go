// Package auth owns Google sign-in, session tokens and the Gin middleware that
// every per-user route in spec §7 sits behind.
package auth

import (
	"net/url"
	"strings"
)

// AuthEndpoint is Google's consent screen. The frontend builds its login link
// from AuthCodeURL so the scope list never diverges from the one the backend
// exchanges against.
const AuthEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"

// Scopes is the full consent set. calendar.events and tasks are requested at
// first sign-in — not later, when the google slice lands — so that no existing
// user has to re-consent (spec §5.1 steps 6-7).
var Scopes = []string{
	"openid",
	"email",
	"profile",
	"https://www.googleapis.com/auth/calendar.events",
	"https://www.googleapis.com/auth/tasks",
}

// ScopeString joins Scopes the way Google's `scope` parameter expects.
func ScopeString() string { return strings.Join(Scopes, " ") }

// AuthCodeURL builds the consent URL. access_type=offline plus prompt=consent
// is what makes Google return a refresh token (users.google_refresh_token).
func AuthCodeURL(clientID, redirectURI, state string) string {
	q := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {ScopeString()},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
		"state":         {state},
	}
	return AuthEndpoint + "?" + q.Encode()
}
