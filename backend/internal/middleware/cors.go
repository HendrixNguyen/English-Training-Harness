// Package middleware holds the request-edge guards cmd/api mounts once: the
// CORS allow-list for the PWA's origin and the request-body bound. It knows
// nothing about routes, users or tables.
package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// ErrBadOrigin is ParseOrigins' error for an entry that is not a bare
// scheme://host[:port] origin, or for an empty list.
var ErrBadOrigin = errors.New("middleware: FRONTEND_ORIGIN entries must be http(s)://host[:port] with no path, query, fragment, userinfo, default port, wildcard or trailing dot")

// ParseOrigins turns the FRONTEND_ORIGIN value into the exact-match allow-list:
// comma-separated, whitespace trimmed, one trailing slash dropped, scheme and
// host lower-cased (browsers send Origin lower-cased). Empty entries are
// skipped; an empty result is an error — a CORS middleware with no origins
// would silently block the PWA, which is the bug this package exists to fix.
func ParseOrigins(raw string) ([]string, error) {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSuffix(strings.TrimSpace(part), "/")
		if p == "" {
			continue
		}
		u, err := url.Parse(p)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") ||
			u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil || u.Opaque != "" {
			return nil, fmt.Errorf("%w: %q", ErrBadOrigin, strings.TrimSpace(part))
		}
		// The three shapes url.Parse accepts that no browser ever puts in an
		// Origin header. Refuse them rather than normalise: boot must say what
		// the operator typed and why it can never match.
		if reason := neverSentByABrowser(u); reason != "" {
			return nil, fmt.Errorf("%w: %q (%s)", ErrBadOrigin, strings.TrimSpace(part), reason)
		}
		out = append(out, u.Scheme+"://"+strings.ToLower(u.Host))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no origins in %q", ErrBadOrigin, raw)
	}
	return out, nil
}

// neverSentByABrowser names why a parsed origin can never equal a browser's
// Origin header, or "" when it can. Browsers omit the scheme's default port
// (RFC 6454 §6.1 serialises host:port only when the port is not the default),
// never send a wildcard, and strip a trailing dot.
func neverSentByABrowser(u *url.URL) string {
	host, port := u.Hostname(), u.Port()
	switch {
	case strings.Contains(host, "*"):
		return "wildcard hosts are not supported: list each preview origin explicitly"
	case strings.HasSuffix(host, "."):
		return "trailing dot: browsers send the host without it"
	case (u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80"):
		return "default port: browsers omit :" + port + " from Origin, so this entry would never match"
	}
	return ""
}

// CORS answers cross-origin browser requests from the allow-listed origins
// (spec §8 deploys the PWA and the API as two Railway services on two
// origins). Mount it globally with engine.Use before any route: Gin rebuilds
// its NoRoute chain from the global middleware, so a preflight for a path
// that has only a GET route (Gin registers no OPTIONS routes) still reaches
// this handler and is answered 204, not 404.
//
// Allowed origin: echo it in Access-Control-Allow-Origin (never "*"), add
// Vary: Origin; a preflight (OPTIONS + Access-Control-Request-Method) also
// gets Allow-Methods / Allow-Headers / Max-Age and stops here with 204.
// Origin not on the list: a preflight is answered 403; an actual request
// passes through with no CORS headers, which the browser then refuses to
// read, while non-browser clients are unaffected. No Origin header
// (same-origin, curl, the Railway health probe): untouched.
//
// There is deliberately no Access-Control-Allow-Credentials: the session is
// a bearer Authorization header, not a cookie.
func CORS(allowed []string) gin.HandlerFunc {
	allow := make(map[string]bool, len(allowed))
	for _, o := range allowed {
		allow[o] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		c.Writer.Header().Add("Vary", "Origin")
		preflight := c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != ""
		if !allow[origin] {
			if preflight {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}
		c.Header("Access-Control-Allow-Origin", origin)
		if preflight {
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Max-Age", "600")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
