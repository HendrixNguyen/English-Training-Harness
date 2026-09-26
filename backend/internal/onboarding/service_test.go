package onboarding

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

type harness struct {
	svc     *Service
	repo    *fakeRepo
	quiz    *fakeQuiz
	limiter *fakeLimiter
	ai      *scriptedProvider
	pet     *fakePet
}

var (
	ctx    = context.Background()
	sept22 = time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
)

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{repo: newFakeRepo(), quiz: newFakeQuiz(), limiter: &fakeLimiter{}, ai: newScripted(),
		pet: &fakePet{state: PetState{PlantName: "My Green Buddy", HealthPoints: 100, Stage: "sprout"}}}
	h.ai.replies[airouter.TaskPlacementTest] = []string{`{"cefr_level":"B1"}`}
	h.ai.replies[airouter.TaskRoadmapGen] = []string{fixtureRoadmapJSON(t)}
	h.svc = NewService(h.repo, h.quiz, h.limiter, routerOver(h.ai), h.pet, fixedClock(sept22))
	return h
}

func TestAssessHappyPathGradesGeneratesAndPersistsOnce(t *testing.T) {
	h := newHarness(t)

	// validRequest() answers every bank item correctly, so GradeFloor is C1 —
	// higher than the scripted grader's B1 — and the placement floor raises
	// the assessed level to C1 (a 10/10 learner cannot be graded below what
	// their own answers prove).
	out, err := h.svc.Assess(ctx, "u1", validRequest())
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if !out.Created || out.Status != "success" || out.AssessedLevel != "C1" || out.RoadmapID != "rm-new" {
		t.Errorf("out = %+v", out)
	}
	if out.PetState != (PetState{PlantName: "My Green Buddy", HealthPoints: 100, Stage: "sprout"}) {
		t.Errorf("pet_state = %+v", out.PetState)
	}
	if len(h.repo.saved) != 1 {
		t.Fatalf("saved %d assessments, want 1", len(h.repo.saved))
	}
	a := h.repo.saved[0]
	if a.CEFRLevel != "C1" || a.TargetGoal != "IELTS 7.0 Preparation" || a.Timezone != "Asia/Ho_Chi_Minh" || a.NotificationTime != "20:00:00" {
		t.Errorf("assessment = %+v, want the §6.1 request fields (raised to the floor C1)", a)
	}
	if ex := a.Roadmap.Exercises(); len(ex) != 84 || !strings.Contains(string(ex[0].ContentJSON), `"title"`) || !strings.Contains(string(ex[0].ContentJSON), `"duration_minutes":10`) {
		t.Errorf("exercises = %d rows, first content %s; want 84 with title and duration_minutes", len(ex), ex[0].ContentJSON)
	}
	if h.ai.calls[airouter.TaskPlacementTest] != 1 || h.ai.calls[airouter.TaskRoadmapGen] != 1 {
		t.Errorf("AI calls = %v, want one of each", h.ai.calls)
	}
	if h.limiter.calls != 1 {
		t.Errorf("rate limiter consulted %d times, want 1 per assessment", h.limiter.calls)
	}
	if h.pet.ensured != 1 {
		t.Errorf("pet.Ensure called %d times, want 1", h.pet.ensured)
	}
	if h.quiz.lastTTL != store.PlacementQuizTTL {
		t.Errorf("quiz TTL = %v, want store.PlacementQuizTTL (2h, §4)", h.quiz.lastTTL)
	}
	if store.PlacementQuizTTL != 2*time.Hour {
		t.Errorf("store.PlacementQuizTTL = %v, want 2h per §4", store.PlacementQuizTTL)
	}
	if h.quiz.cleared != 1 || len(h.quiz.staged) != 0 {
		t.Error("quiz hash not cleared after success")
	}
	if h.quiz.level["u1"] != "" {
		t.Error("the staged level must go with the hash on success")
	}
	// The generation prompt is the airouter constant and carries the raised
	// (floored) level and goal.
	gen := h.ai.prompts[airouter.TaskRoadmapGen][0]
	if !strings.HasPrefix(gen, airouter.RoadmapSystemPrompt+"|") || !strings.Contains(gen, "C1") || !strings.Contains(gen, "IELTS 7.0 Preparation") {
		t.Errorf("roadmap prompt = %.120s…", gen)
	}
}

func TestAssessIsIdempotentWhileARoadmapIsActive(t *testing.T) {
	h := newHarness(t)
	h.repo.activeID = "rm-existing"
	h.repo.profile.CEFRCurrent = "B2"

	out, err := h.svc.Assess(ctx, "u1", validRequest())
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if out.Created || out.RoadmapID != "rm-existing" || out.AssessedLevel != "B2" || out.Status != "success" {
		t.Errorf("out = %+v, want the existing roadmap, not created", out)
	}
	if len(h.ai.calls) != 0 || h.limiter.calls != 0 || len(h.repo.saved) != 0 || len(h.quiz.staged) != 0 {
		t.Errorf("idempotent path had side effects: ai=%v limiter=%d saved=%d staged=%d", h.ai.calls, h.limiter.calls, len(h.repo.saved), len(h.quiz.staged))
	}
	if h.pet.ensured != 1 {
		t.Errorf("pet.Ensure called %d times, want 1 (the response still needs pet_state)", h.pet.ensured)
	}
}

func TestAssessRetriesOnceOnABadGradeThenFailsWithoutWriting(t *testing.T) {
	h := newHarness(t)
	h.ai.replies[airouter.TaskPlacementTest] = []string{"```json\n{\"cefr_level\":\"B1\"}\n```", `{"level":"B1"}`}

	_, err := h.svc.Assess(ctx, "u1", validRequest())
	if !errors.Is(err, ErrBadAIOutput) {
		t.Fatalf("err = %v, want ErrBadAIOutput", err)
	}
	if h.ai.calls[airouter.TaskPlacementTest] != 2 || h.ai.calls[airouter.TaskRoadmapGen] != 0 {
		t.Errorf("AI calls = %v, want 2 placement, 0 roadmap", h.ai.calls)
	}
	if len(h.repo.saved) != 0 {
		t.Error("a failed grading wrote an assessment")
	}
	if h.quiz.cleared != 0 || len(h.quiz.staged["u1"]) == 0 {
		t.Error("staged answers must survive a failed grading for the TTL")
	}
}

func TestAssessRecoversWhenTheSecondRoadmapAttemptIsValid(t *testing.T) {
	h := newHarness(t)
	h.ai.replies[airouter.TaskRoadmapGen] = []string{`{"modules":[]}`, fixtureRoadmapJSON(t)}

	out, err := h.svc.Assess(ctx, "u1", validRequest())
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if !out.Created || h.ai.calls[airouter.TaskRoadmapGen] != 2 || len(h.repo.saved) != 1 {
		t.Errorf("out=%+v calls=%v saved=%d", out, h.ai.calls, len(h.repo.saved))
	}
}

func TestAssessFailsWithoutWritingWhenTheRoadmapIsBadTwice(t *testing.T) {
	h := newHarness(t)
	h.ai.replies[airouter.TaskRoadmapGen] = []string{`{"modules":[]}`, "Sure! Here is the roadmap: {}"}

	_, err := h.svc.Assess(ctx, "u1", validRequest())
	if !errors.Is(err, ErrBadAIOutput) {
		t.Fatalf("err = %v, want ErrBadAIOutput", err)
	}
	if len(h.repo.saved) != 0 {
		t.Error("a bad roadmap wrote something — the users update must be in the same tx as the roadmap and never run alone")
	}
}

func TestAssessPropagatesProviderFailures(t *testing.T) {
	h := newHarness(t)
	h.ai.replies[airouter.TaskPlacementTest] = nil // scripted returns an error → router: all providers failed

	_, err := h.svc.Assess(ctx, "u1", validRequest())
	if !errors.Is(err, airouter.ErrAllProvidersFailed) {
		t.Fatalf("err = %v, want ErrAllProvidersFailed passed through", err)
	}
	if len(h.repo.saved) != 0 {
		t.Error("wrote on provider failure")
	}
}

func TestAssessWithNoProvidersConfigured(t *testing.T) {
	h := newHarness(t)
	h.svc = NewService(h.repo, h.quiz, h.limiter, airouter.NewRouterWithProviders(nil), h.pet, fixedClock(sept22))

	if _, err := h.svc.Assess(ctx, "u1", validRequest()); !errors.Is(err, airouter.ErrNoProviders) {
		t.Fatalf("err = %v, want ErrNoProviders", err)
	}
}

func TestAssessIsRateLimitedBeforeAnyAICall(t *testing.T) {
	h := newHarness(t)
	h.limiter.err = airouter.ErrRateLimited

	if _, err := h.svc.Assess(ctx, "u1", validRequest()); !errors.Is(err, airouter.ErrRateLimited) {
		t.Fatalf("err = %v, want ErrRateLimited", err)
	}
	if len(h.ai.calls) != 0 || len(h.quiz.staged) != 0 || len(h.repo.saved) != 0 {
		t.Error("rate-limited request had side effects")
	}
}

func TestAssessValidatesTheRequestBeforeTouchingAnything(t *testing.T) {
	cases := map[string]func(r *AssessmentRequest){
		"empty goal":         func(r *AssessmentRequest) { r.TargetGoal = "   " },
		"goal too long":      func(r *AssessmentRequest) { r.TargetGoal = strings.Repeat("x", 256) },
		"bad timezone":       func(r *AssessmentRequest) { r.Timezone = "Mars/Olympus" },
		"empty timezone":     func(r *AssessmentRequest) { r.Timezone = "" },
		"bad time":           func(r *AssessmentRequest) { r.NotificationTime = "8pm" },
		"no answers":         func(r *AssessmentRequest) { r.Answers = nil },
		"unknown question":   func(r *AssessmentRequest) { r.Answers[0].QuestionID = "q99" },
		"unknown option":     func(r *AssessmentRequest) { r.Answers[0].SelectedOption = "E" },
		"duplicate question": func(r *AssessmentRequest) { r.Answers[1].QuestionID = r.Answers[0].QuestionID },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			req := validRequest()
			mutate(&req)
			_, err := h.svc.Assess(ctx, "u1", req)
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("err = %v, want ErrInvalidRequest", err)
			}
			if len(h.ai.calls) != 0 || h.limiter.calls != 0 || len(h.quiz.staged) != 0 || len(h.repo.saved) != 0 || h.pet.ensured != 0 {
				t.Error("invalid request had side effects")
			}
		})
	}
}

func TestAssessAcceptsAPartialAnswerSet(t *testing.T) {
	h := newHarness(t)
	req := validRequest()
	req.Answers = req.Answers[:3]
	if _, err := h.svc.Assess(ctx, "u1", req); err != nil {
		t.Fatalf("Assess with 3 answers: %v", err)
	}
	if !strings.Contains(h.ai.prompts[airouter.TaskPlacementTest][0], `"id":"q3"`) || strings.Contains(h.ai.prompts[airouter.TaskPlacementTest][0], `"id":"q4"`) {
		t.Error("grader prompt must contain exactly the answered items")
	}
}

func TestAssessSurfacesARepoFailure(t *testing.T) {
	h := newHarness(t)
	h.repo.saveErr = errors.New("pg down")
	if _, err := h.svc.Assess(ctx, "u1", validRequest()); err == nil || errors.Is(err, ErrBadAIOutput) {
		t.Fatalf("err = %v, want the repo error", err)
	}
}

func TestAssessGivesEachAICallItsOwnDeadline(t *testing.T) {
	h := newHarness(t)
	if _, err := h.svc.Assess(ctx, "u1", validRequest()); err != nil {
		t.Fatalf("Assess: %v", err)
	}
	within := func(got, want time.Duration) bool { return got > want-2*time.Second && got <= want }
	if b := h.ai.budgets[airouter.TaskPlacementTest]; len(b) != 1 || !within(b[0], airouter.DefaultTaskTimeout) {
		t.Errorf("placement budget = %v, want ≈ %s", b, airouter.DefaultTaskTimeout)
	}
	if b := h.ai.budgets[airouter.TaskRoadmapGen]; len(b) != 1 || !within(b[0], airouter.RoadmapTimeout) {
		t.Errorf("roadmap budget = %v, want ≈ %s (not the 30 s that 502'd on 2026-09-25)", b, airouter.RoadmapTimeout)
	}
}

func TestAssessReportsADeadlineHitAsAITimeoutWithoutWriting(t *testing.T) {
	h := newHarness(t)
	h.ai.timeout[airouter.TaskRoadmapGen] = true

	_, err := h.svc.Assess(ctx, "u1", validRequest())
	if !errors.Is(err, ErrAITimeout) {
		t.Fatalf("err = %v, want ErrAITimeout", err)
	}
	if len(h.repo.saved) != 0 {
		t.Error("a timed-out roadmap wrote an assessment")
	}
}

func TestAssessPassesACallerCancellationThroughUnchanged(t *testing.T) {
	h := newHarness(t)
	gone, cancel := context.WithCancel(ctx)
	cancel() // the client disconnected
	_, err := h.svc.Assess(gone, "u1", validRequest())
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrAITimeout) {
		t.Fatalf("err = %v, want context.Canceled and not ErrAITimeout", err)
	}
}

func TestAssessKeepsTheGradeWhenTheRoadmapFailsAndSkipsGradingOnTheRetry(t *testing.T) {
	h := newHarness(t)
	h.ai.replies[airouter.TaskRoadmapGen] = nil // every roadmap call fails → ErrAllProvidersFailed

	if _, err := h.svc.Assess(ctx, "u1", validRequest()); !errors.Is(err, airouter.ErrAllProvidersFailed) {
		t.Fatalf("first Assess: %v, want ErrAllProvidersFailed", err)
	}
	// validRequest() answers every bank item correctly, so the placement
	// floor (C1) raises the graded B1 before it is staged.
	if h.quiz.level["u1"] != "C1" || len(h.repo.saved) != 0 || h.quiz.cleared != 0 {
		t.Fatalf("after the failed roadmap: level=%q saved=%d cleared=%d; want the raised level staged, nothing written, hash kept", h.quiz.level["u1"], len(h.repo.saved), h.quiz.cleared)
	}

	h.ai.replies[airouter.TaskRoadmapGen] = []string{fixtureRoadmapJSON(t)}
	h.ai.calls[airouter.TaskRoadmapGen] = 0
	out, err := h.svc.Assess(ctx, "u1", validRequest())
	if err != nil || !out.Created || out.AssessedLevel != "C1" {
		t.Fatalf("retry: %+v, %v; want a created roadmap at the staged (raised) level", out, err)
	}
	if h.ai.calls[airouter.TaskPlacementTest] != 1 {
		t.Errorf("placement graded %d times, want 1 — the retry must reuse the staged level", h.ai.calls[airouter.TaskPlacementTest])
	}
	if h.limiter.calls != 2 {
		t.Errorf("limiter calls = %d, want 2 (one slot per assessment call, retry included)", h.limiter.calls)
	}
	if len(h.repo.saved) != 1 || h.repo.saved[0].CEFRLevel != "C1" || h.quiz.cleared != 1 || h.quiz.level["u1"] != "" {
		t.Errorf("after the retry: saved=%d cleared=%d level=%q", len(h.repo.saved), h.quiz.cleared, h.quiz.level["u1"])
	}
}

// TestAssessRaisesTheAILevelToTheFloor: the AI grades A2 but the bank
// answers are all correct (GradeFloor == C1), so the deterministic floor
// wins — a 10/10 learner cannot be staged, generated for or persisted below
// what their own answers prove.
func TestAssessRaisesTheAILevelToTheFloor(t *testing.T) {
	h := newHarness(t)
	h.ai.replies[airouter.TaskPlacementTest] = []string{`{"cefr_level":"A2"}`}

	out, err := h.svc.Assess(ctx, "u1", validRequest())
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if out.AssessedLevel != "C1" {
		t.Errorf("AssessedLevel = %q, want the floor C1", out.AssessedLevel)
	}
	if h.repo.saved[0].CEFRLevel != "C1" {
		t.Errorf("saved CEFRLevel = %q, want C1", h.repo.saved[0].CEFRLevel)
	}
	gen := h.ai.prompts[airouter.TaskRoadmapGen][0]
	if !strings.Contains(gen, "C1") {
		t.Errorf("roadmap prompt = %.160s…, want the raised level C1", gen)
	}
	if h.quiz.level["u1"] != "" {
		t.Error("the staged level must go with the hash on success")
	}
}

// TestAssessKeepsAHigherAILevel: the floor never lowers the AI's own grade.
func TestAssessKeepsAHigherAILevel(t *testing.T) {
	h := newHarness(t)
	h.ai.replies[airouter.TaskPlacementTest] = []string{`{"cefr_level":"B2"}`}
	req := validRequest()
	// Only the A1 pair right: GradeFloor stops at A1, well below the AI's B2.
	for i := range req.Answers {
		if req.Answers[i].QuestionID == "q1" || req.Answers[i].QuestionID == "q2" {
			continue
		}
		q, _ := Lookup(req.Answers[i].QuestionID)
		for opt := range q.Options {
			if opt != q.Correct {
				req.Answers[i].SelectedOption = opt
				break
			}
		}
	}

	out, err := h.svc.Assess(ctx, "u1", req)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if out.AssessedLevel != "B2" {
		t.Errorf("AssessedLevel = %q, want the AI's own B2 kept", out.AssessedLevel)
	}
}

func TestAssessGradesAgainWhenTheRetryChangesAnAnswer(t *testing.T) {
	h := newHarness(t)
	h.ai.replies[airouter.TaskRoadmapGen] = nil
	if _, err := h.svc.Assess(ctx, "u1", validRequest()); err == nil {
		t.Fatal("first Assess succeeded; the fixture should fail at the roadmap")
	}

	h.ai.replies[airouter.TaskPlacementTest] = []string{`{"cefr_level":"B1"}`, `{"cefr_level":"A2"}`}
	h.ai.replies[airouter.TaskRoadmapGen] = []string{fixtureRoadmapJSON(t)}
	h.ai.calls[airouter.TaskRoadmapGen] = 0
	req := validRequest()
	req.Answers[0].SelectedOption = "A" // q1 has options A-D; B is correct
	out, err := h.svc.Assess(ctx, "u1", req)
	if err != nil || out.AssessedLevel != "A2" {
		t.Fatalf("retry with changed answers: %+v, %v; want a fresh grade", out, err)
	}
	if h.ai.calls[airouter.TaskPlacementTest] != 2 {
		t.Errorf("placement graded %d times, want 2", h.ai.calls[airouter.TaskPlacementTest])
	}
}
