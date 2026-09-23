package pet

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// newPetRouter stands in for auth.Require() by injecting the user id under
// auth.ContextUserID (Require itself is covered in the auth slice).
func newPetRouter(svc *Service, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1", func(c *gin.Context) { c.Set(auth.ContextUserID, userID); c.Next() })
	g.GET("/pet/status", StatusHandler(svc))
	g.POST("/pet/revive", ReviveHandler(svc))
	return r
}

func do(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestStatusReturnsTheSpec63BodyAndCreatesTheRow(t *testing.T) {
	h := newHarness(sept22)

	w := do(newPetRouter(h.svc, "u1"), http.MethodGet, "/api/v1/pet/status", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var body struct {
		PlantName       string  `json:"plant_name"`
		Stage           string  `json:"stage"`
		HealthPoints    int     `json:"health_points"`
		CurrentStreak   int     `json:"current_streak"`
		LastPracticedAt *string `json:"last_practiced_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding: %v (%s)", err, w.Body.String())
	}
	if body.PlantName != "My Green Buddy" || body.Stage != "sprout" || body.HealthPoints != 100 || body.CurrentStreak != 0 || body.LastPracticedAt != nil {
		t.Errorf("body = %+v, want the fresh-pet defaults with last_practiced_at null", body)
	}
	if !strings.Contains(w.Body.String(), `"last_practiced_at":null`) {
		t.Errorf("last_practiced_at must be present and null on a fresh pet: %s", w.Body.String())
	}
	if h.repo.ensured != 1 {
		t.Errorf("Ensure called %d times, want 1", h.repo.ensured)
	}
}

func TestStatusFormatsLastPracticedAtAsUTCRFC3339(t *testing.T) {
	h := newHarness(sept22)
	at := time.Date(2026, time.September, 21, 20, 15, 0, 0, time.UTC)
	h.repo.states["u1"] = State{PlantName: "My Green Buddy", HealthPoints: 80, Stage: StageSprout, CurrentStreak: 5, LastPracticedAt: &at}

	w := do(newPetRouter(h.svc, "u1"), http.MethodGet, "/api/v1/pet/status", "")
	// Exactly the §6.3 example.
	want := `{"plant_name":"My Green Buddy","stage":"sprout","health_points":80,"current_streak":5,"last_practiced_at":"2026-09-21T20:15:00Z"}`
	if strings.TrimSpace(w.Body.String()) != want {
		t.Errorf("body =\n%s\nwant\n%s", w.Body.String(), want)
	}
}

func TestReviveReturns409WhileNotWilted(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 30, Stage: StageSprout}

	w := do(newPetRouter(h.svc, "u1"), http.MethodPost, "/api/v1/pet/revive", `{"answers":{"q1":"C"}}`)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"error":"pet_not_wilted"`) {
		t.Errorf("status = %d body = %s, want 409 pet_not_wilted", w.Code, w.Body.String())
	}
}

func TestReviveStartsThenPassesWithTheSpec63Body(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}
	r := newPetRouter(h.svc, "u1")

	// §6.3 request body; an empty body is accepted too (the client may have no answers yet).
	w := do(r, http.MethodPost, "/api/v1/pet/revive", `{"answers":{"q1":"C","q2":"B"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("start: status = %d body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"revival_passed":false`) || !strings.Contains(w.Body.String(), `"health_points":0`) {
		t.Errorf("start body = %s, want revival_passed false and health 0", w.Body.String())
	}

	h.study.set("u1", "2026-09-22", 900)
	w = do(r, http.MethodPost, "/api/v1/pet/revive", "")
	if w.Code != http.StatusOK {
		t.Fatalf("pass: status = %d body = %s", w.Code, w.Body.String())
	}
	want := `{"revival_passed":true,"pet_state":{"health_points":50,"stage":"sprout","current_streak":0}}`
	if strings.TrimSpace(w.Body.String()) != want {
		t.Errorf("pass body =\n%s\nwant\n%s", w.Body.String(), want)
	}
}

func TestReviveRejectsMalformedJSON(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}

	w := do(newPetRouter(h.svc, "u1"), http.MethodPost, "/api/v1/pet/revive", `nonsense`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandlersReturn500OnRepoFailure(t *testing.T) {
	h := newHarness(sept22)
	h.repo.ensureErr = errBoom
	r := newPetRouter(h.svc, "u1")

	if w := do(r, http.MethodGet, "/api/v1/pet/status", ""); w.Code != http.StatusInternalServerError {
		t.Errorf("status: %d, want 500", w.Code)
	}
	if w := do(r, http.MethodPost, "/api/v1/pet/revive", ""); w.Code != http.StatusInternalServerError {
		t.Errorf("revive: %d, want 500", w.Code)
	}
}
