package onboarding

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

func newRouter(svc *Service, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1", func(c *gin.Context) { c.Set(auth.ContextUserID, userID); c.Next() })
	g.GET("/onboarding/quiz", QuizHandler())
	g.POST("/onboarding/assessment", AssessmentHandler(svc))
	g.POST("/roadmaps/regenerate", RegenerateHandler(svc))
	return r
}

func post(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/assessment", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// postRegenerate posts to POST /api/v1/roadmaps/regenerate. body == nil sends
// no body at all (ContentLength 0), matching a bare POST with no JSON.
func postRegenerate(r *gin.Engine, body *string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		reader = strings.NewReader(*body)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/roadmaps/regenerate", reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func strPtr(s string) *string { return &s }

// spec61Request is the backend spec §6.1 example body with the bank's ids.
const spec61Request = `{"target_goal": "IELTS 7.0 Preparation", "notification_time": "20:00:00", "timezone": "Asia/Ho_Chi_Minh", "answers": [{ "question_id": "q1", "selected_option": "B" }, { "question_id": "q2", "selected_option": "A" }]}`

func TestQuizReturnsTheBankWithoutAnswers(t *testing.T) {
	h := newHarness(t)
	w := httptest.NewRecorder()
	newRouter(h.svc, "u1").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/quiz", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body QuizResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Questions) != len(Bank) || strings.Contains(w.Body.String(), `"correct"`) {
		t.Errorf("body = %s", w.Body.String())
	}
}

func TestAssessmentReturns201WithTheSpec61Body(t *testing.T) {
	h := newHarness(t)
	w := post(newRouter(h.svc, "u1"), spec61Request)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	want := `{"status":"success","assessed_level":"B1","roadmap_id":"rm-new","pet_state":{"plant_name":"My Green Buddy","health_points":100,"stage":"sprout"}}`
	if strings.TrimSpace(w.Body.String()) != want {
		t.Errorf("body =\n%s\nwant\n%s", w.Body.String(), want)
	}
}

func TestAssessmentReturns200WhenARoadmapAlreadyExists(t *testing.T) {
	h := newHarness(t)
	h.repo.activeID = "rm-existing"
	w := post(newRouter(h.svc, "u1"), spec61Request)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"roadmap_id":"rm-existing"`) {
		t.Errorf("status = %d body = %s, want 200 with the existing roadmap", w.Code, w.Body.String())
	}
}

func TestAssessmentErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(h *harness)
		body   string
		status int
		code   string
	}{
		{"malformed json", nil, `nonsense`, 400, "invalid_request"},
		{"validation", nil, `{"target_goal":"","notification_time":"20:00:00","timezone":"UTC","answers":[{"question_id":"q1","selected_option":"B"}]}`, 400, "invalid_request"},
		{"rate limited", func(h *harness) { h.limiter.err = airouter.ErrRateLimited }, spec61Request, 429, "rate_limited"},
		{"no providers", func(h *harness) {
			h.svc = NewService(h.repo, h.quiz, h.limiter, airouter.NewRouterWithProviders(nil), h.pet, fixedClock(sept22))
		}, spec61Request, 503, "ai_unavailable"},
		{"all providers failed", func(h *harness) { h.ai.replies[airouter.TaskPlacementTest] = nil }, spec61Request, 502, "ai_upstream_failed"},
		{"bad output twice", func(h *harness) { h.ai.replies[airouter.TaskPlacementTest] = []string{"x", "y"} }, spec61Request, 502, "ai_bad_output"},
		{"repo failure", func(h *harness) { h.repo.saveErr = errors.New("pg") }, spec61Request, 500, "internal_error"},
		{"ai timed out", func(h *harness) { h.ai.timeout[airouter.TaskRoadmapGen] = true }, spec61Request, 504, "ai_timeout"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			if tc.setup != nil {
				tc.setup(h)
			}
			w := post(newRouter(h.svc, "u1"), tc.body)
			if w.Code != tc.status || !strings.Contains(w.Body.String(), `"error":"`+tc.code+`"`) {
				t.Errorf("status = %d body = %s, want %d %s", w.Code, w.Body.String(), tc.status, tc.code)
			}
		})
	}
}

func TestRegenerateHandlerNoBodyKeepsTheCurrentLevel(t *testing.T) {
	h := newHarness(t)
	h.repo.activeID = "rm-existing"
	h.repo.profile = Profile{CEFRCurrent: "B1", TargetGoal: "goal"}

	w := postRegenerate(newRouter(h.svc, "u1"), nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	want := `{"status":"success","assessed_level":"B1","roadmap_id":"rm-new"}`
	if strings.TrimSpace(w.Body.String()) != want {
		t.Errorf("body =\n%s\nwant\n%s", w.Body.String(), want)
	}
}

func TestRegenerateHandlerWithLevelReturns201(t *testing.T) {
	h := newHarness(t)
	h.repo.activeID = "rm-existing"
	h.repo.profile = Profile{CEFRCurrent: "B1", TargetGoal: "goal"}

	w := postRegenerate(newRouter(h.svc, "u1"), strPtr(`{"cefr_level":"B2"}`))
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	want := `{"status":"success","assessed_level":"B2","roadmap_id":"rm-new"}`
	if strings.TrimSpace(w.Body.String()) != want {
		t.Errorf("body =\n%s\nwant\n%s", w.Body.String(), want)
	}
}

func TestRegenerateHandlerErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(h *harness)
		body   *string
		status int
		code   string
	}{
		{"two steps away", func(h *harness) {
			h.repo.activeID = "rm-existing"
			h.repo.profile = Profile{CEFRCurrent: "B1", TargetGoal: "goal"}
		}, strPtr(`{"cefr_level":"C1"}`), 400, "invalid_request"},
		{"malformed json", func(h *harness) {
			h.repo.activeID = "rm-existing"
		}, strPtr(`nonsense`), 400, "invalid_request"},
		{"no active roadmap", nil, nil, 404, "no_active_roadmap"},
		{"rate limited", func(h *harness) {
			h.repo.activeID = "rm-existing"
			h.limiter.err = airouter.ErrRateLimited
		}, nil, 429, "rate_limited"},
		{"no providers", func(h *harness) {
			h.repo.activeID = "rm-existing"
			h.svc = NewService(h.repo, h.quiz, h.limiter, airouter.NewRouterWithProviders(nil), h.pet, fixedClock(sept22))
		}, nil, 503, "ai_unavailable"},
		{"bad output twice", func(h *harness) {
			h.repo.activeID = "rm-existing"
			h.ai.replies[airouter.TaskRoadmapGen] = []string{"x", "y"}
		}, nil, 502, "ai_bad_output"},
		{"ai timed out", func(h *harness) {
			h.repo.activeID = "rm-existing"
			h.ai.timeout[airouter.TaskRoadmapGen] = true
		}, nil, 504, "ai_timeout"},
		{"all providers failed", func(h *harness) {
			h.repo.activeID = "rm-existing"
			h.ai.replies[airouter.TaskRoadmapGen] = nil
		}, nil, 502, "ai_upstream_failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			if tc.setup != nil {
				tc.setup(h)
			}
			w := postRegenerate(newRouter(h.svc, "u1"), tc.body)
			if w.Code != tc.status || !strings.Contains(w.Body.String(), `"error":"`+tc.code+`"`) {
				t.Errorf("status = %d body = %s, want %d %s", w.Code, w.Body.String(), tc.status, tc.code)
			}
		})
	}
}

func TestRegenerateHandlerRequiresAuth(t *testing.T) {
	h := newHarness(t)
	w := postRegenerate(newRouter(h.svc, ""), nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}
