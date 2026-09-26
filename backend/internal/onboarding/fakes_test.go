package onboarding

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
)

type fakeRepo struct {
	activeID     string // "" == none
	profile      Profile
	saved        []Assessment
	nextID       string
	saveErr      error
	activeErr    error
	replaceCalls []struct {
		level   string
		roadmap airouter.Roadmap
	}
	replaceErr error
}

func newFakeRepo() *fakeRepo { return &fakeRepo{profile: Profile{CEFRCurrent: "A1"}, nextID: "rm-new"} }

func (f *fakeRepo) ActiveRoadmapID(context.Context, string) (string, bool, error) {
	if f.activeErr != nil {
		return "", false, f.activeErr
	}
	return f.activeID, f.activeID != "", nil
}

func (f *fakeRepo) Profile(context.Context, string) (Profile, error) { return f.profile, nil }

func (f *fakeRepo) SaveAssessment(_ context.Context, _ string, a Assessment) (string, error) {
	if f.saveErr != nil {
		return "", f.saveErr
	}
	f.saved = append(f.saved, a)
	f.activeID = f.nextID
	f.profile.CEFRCurrent = a.CEFRLevel
	return f.nextID, nil
}

func (f *fakeRepo) ReplaceRoadmap(_ context.Context, _, level string, roadmap airouter.Roadmap) (string, error) {
	if f.replaceErr != nil {
		return "", f.replaceErr
	}
	f.replaceCalls = append(f.replaceCalls, struct {
		level   string
		roadmap airouter.Roadmap
	}{level, roadmap})
	f.activeID = f.nextID
	f.profile.CEFRCurrent = level
	return f.nextID, nil
}

type fakeQuiz struct {
	staged  map[string][]Answer
	level   map[string]string
	lastTTL time.Duration
	cleared int
}

func newFakeQuiz() *fakeQuiz {
	return &fakeQuiz{staged: map[string][]Answer{}, level: map[string]string{}}
}

func (f *fakeQuiz) StageAnswers(_ context.Context, userID string, answers []Answer, ttl time.Duration) error {
	f.staged[userID] = answers
	delete(f.level, userID) // DEL + HSET in the real store
	f.lastTTL = ttl
	return nil
}

func (f *fakeQuiz) StageLevel(_ context.Context, userID, level string, _ time.Duration) error {
	f.level[userID] = level
	return nil
}

func (f *fakeQuiz) StagedLevel(_ context.Context, userID string, answers []Answer) (string, error) {
	level, staged := f.level[userID], f.staged[userID]
	if level == "" || len(staged) != len(answers) {
		return "", nil
	}
	for i := range answers {
		if staged[i] != answers[i] {
			return "", nil
		}
	}
	return level, nil
}

func (f *fakeQuiz) Clear(_ context.Context, userID string) error {
	f.cleared++
	delete(f.staged, userID)
	delete(f.level, userID)
	return nil
}

type fakeLimiter struct {
	calls int
	err   error
}

func (f *fakeLimiter) Allow(context.Context, string) error {
	f.calls++
	return f.err
}

type fakePet struct {
	ensured int
	state   PetState
	err     error
}

func (f *fakePet) Ensure(context.Context, string) (PetState, error) {
	f.ensured++
	return f.state, f.err
}

// scriptedProvider answers per task type from a queue, so a test can make the
// first placement answer malformed and the second valid. It is wrapped in a
// real airouter.Router, so the Generator seam is the production type.
type scriptedProvider struct {
	replies map[airouter.TaskType][]string
	calls   map[airouter.TaskType]int
	prompts map[airouter.TaskType][]string        // system|user per call
	budgets map[airouter.TaskType][]time.Duration // time left on ctx at each call
	timeout map[airouter.TaskType]bool            // answer as a model that outlives its deadline
}

func newScripted() *scriptedProvider {
	return &scriptedProvider{replies: map[airouter.TaskType][]string{}, calls: map[airouter.TaskType]int{},
		prompts: map[airouter.TaskType][]string{}, budgets: map[airouter.TaskType][]time.Duration{}, timeout: map[airouter.TaskType]bool{}}
}

// task is smuggled through the system prompt: the placement and roadmap
// prompts are distinct constants, so the fake tells them apart by content.
func (p *scriptedProvider) GenerateContent(ctx context.Context, system, user string) (string, error) {
	task := airouter.TaskRoadmapGen
	if system == PlacementSystemPrompt {
		task = airouter.TaskPlacementTest
	}
	p.prompts[task] = append(p.prompts[task], system+"|"+user)
	if dl, ok := ctx.Deadline(); ok {
		p.budgets[task] = append(p.budgets[task], time.Until(dl))
	}
	i := p.calls[task]
	p.calls[task]++
	if p.timeout[task] {
		return "", fmt.Errorf("scripted: %s outlived its deadline: %w", task, context.DeadlineExceeded)
	}
	if i >= len(p.replies[task]) {
		return "", fmt.Errorf("scripted: no reply %d for %s", i, task)
	}
	return p.replies[task][i], nil
}

func routerOver(p *scriptedProvider) *airouter.Router {
	return airouter.NewRouterWithProviders(map[airouter.ProviderType]airouter.LLMProvider{airouter.ProviderGemini: p})
}

// fixtureRoadmap is a valid 4x7x3 roadmap.
func fixtureRoadmap() airouter.Roadmap {
	r := airouter.Roadmap{Title: "Fixture", CEFRLevel: "B1"}
	for m := 1; m <= airouter.Modules; m++ {
		mod := airouter.Module{Week: m, Title: fmt.Sprintf("Week %d", m), Focus: "fixture"}
		for d := 1; d <= airouter.DaysPerModule; d++ {
			day := airouter.Day{Title: fmt.Sprintf("Day %d", (m-1)*airouter.DaysPerModule+d)}
			for _, tt := range airouter.TaskTypes {
				day.Tasks = append(day.Tasks, airouter.Task{Type: tt, Title: tt + " task", DurationMinutes: 10, Content: json.RawMessage(`{}`)})
			}
			mod.Days = append(mod.Days, day)
		}
		r.Modules = append(r.Modules, mod)
	}
	return r
}

func fixtureRoadmapJSON(t *testing.T) string {
	t.Helper()
	b, err := json.Marshal(fixtureRoadmap())
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return string(b)
}

// validRequest answers every bank item correctly.
func validRequest() AssessmentRequest {
	req := AssessmentRequest{TargetGoal: "IELTS 7.0 Preparation", NotificationTime: "20:00:00", Timezone: "Asia/Ho_Chi_Minh"}
	for _, q := range Bank {
		req.Answers = append(req.Answers, Answer{QuestionID: q.ID, SelectedOption: q.Correct})
	}
	return req
}

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }
