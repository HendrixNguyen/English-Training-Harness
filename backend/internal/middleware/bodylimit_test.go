package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type seen struct {
	answers int
	err     error
}

// newLimitedRouter mirrors cmd/api: the limit sits on the /api/v1 group and
// the handler binds JSON exactly as every real handler does.
func newLimitedRouter(t *testing.T, max int64) (*gin.Engine, *seen) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	s := &seen{}
	v1 := r.Group("/api/v1")
	v1.Use(BodyLimit(max))
	v1.POST("/echo", func(c *gin.Context) {
		var body struct {
			Answers []string `json:"answers"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			s.err = err
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}
		s.answers = len(body.Answers)
		c.JSON(http.StatusOK, gin.H{"n": len(body.Answers)})
	})
	return r, s
}

func post(t *testing.T, r *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/echo", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestABodyPastTheLimitIsRejectedBeforeItIsDecoded(t *testing.T) {
	r, s := newLimitedRouter(t, 64)
	body := `{"answers":[` + strings.Repeat(`"A",`, 1000) + `"A"]}` // ~4 KiB against a 64-byte limit

	w := post(t, r, body)

	if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"invalid_request"}` {
		t.Fatalf("status %d body %s; want 400 invalid_request", w.Code, w.Body.String())
	}
	if s.err == nil || !strings.Contains(s.err.Error(), "request body too large") {
		t.Errorf("handler saw %v, want http.MaxBytesReader's \"request body too large\"", s.err)
	}
	if s.answers != 0 {
		t.Errorf("handler decoded %d answers past the limit, want none", s.answers)
	}
}

func TestABodyUnderTheLimitIsDecodedNormally(t *testing.T) {
	r, s := newLimitedRouter(t, MaxBodyBytes)
	w := post(t, r, `{"answers":["A","B"]}`)
	if w.Code != http.StatusOK || s.answers != 2 {
		t.Errorf("status %d, decoded %d; want 200 and 2", w.Code, s.answers)
	}
}

func TestTheProductionLimitIs64KiB(t *testing.T) {
	if MaxBodyBytes != 64<<10 {
		t.Errorf("MaxBodyBytes = %d, want 65536 — the largest §6 body is an assessment or a push subscription, a few KiB", MaxBodyBytes)
	}
}
