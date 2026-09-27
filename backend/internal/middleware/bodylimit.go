package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes bounds every /api/v1 request body. The largest legitimate
// body is an onboarding assessment (ten answers, ~1 KiB) or a push
// subscription (a ≤ 2048-byte endpoint plus two keys); 64 KiB leaves room
// for §6.2's free-form user_answers without letting one bearer token stream
// megabytes into encoding/json's slice growth before validate() ever runs.
const MaxBodyBytes int64 = 64 << 10

// BodyLimit wraps the request body in http.MaxBytesReader, so a body past
// max makes the handler's ShouldBindJSON fail — which every handler already
// answers with 400 invalid_request — instead of being allocated. Mount it on
// the /api/v1 group *before* the routes are registered: Gin copies a group's
// middleware into each route at registration time.
//
// MaxBytesReader's connection-close hook never fires here — net/http checks
// an unexported interface that gin's ResponseWriter does not satisfy — so an
// over-limit request answers 400 on a connection that stays open; cmd/api's
// ReadTimeout is what bounds a slow sender.
func BodyLimit(max int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
		}
		c.Next()
	}
}
