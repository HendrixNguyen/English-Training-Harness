---
idea: harness/ideas/2026-09-22-run-02/onboarding-placement-test-cefr-grading-and-roadmap-generatio.md
status: approved
priority: high
merged: false
order: 6
---
# Onboarding placement test CEFR grading and roadmap generation — Plan

**Idea:** `harness/ideas/2026-09-22-run-02/onboarding-placement-test-cefr-grading-and-roadmap-generatio.md`
**Goal:** Add `backend/internal/onboarding` — `GET /api/v1/onboarding/quiz` and `POST /api/v1/onboarding/assessment` behind `auth.Require()` — that stages answers in `quiz:placement:{user_id}` (2 h), grades CEFR through `airouter` (`TaskPlacementTest`), generates and validates the 28-day roadmap (`TaskRoadmapGen` + `airouter.ParseRoadmap`), and persists `users.cefr_current`/`target_goal`/`timezone`/`notification_time`, one active `roadmaps` row and its 84 `exercises` in a single transaction — with the **backend spec §6.1 DTOs field for field** — then retires the quests slice's TEMPORARY `store.SeedDemoRoadmap`.

**Spec precedence:** the *Backend Technical Specification* §6.1 is the wire contract for `POST /api/v1/onboarding/assessment` (request and 201 body). Neither spec lists `GET /api/v1/onboarding/quiz` (backend §6.1 has two endpoints; 1st-thinking §7 the same list) — **it is kept as an addition** because a client cannot answer questions it was never given; its shape is this plan's, recorded in CODEMAP for the spec's owner. §4 fixes the Redis hash and TTL; §5.1 the flow; §3.2 the columns; §6.1 of the 1st-thinking doc the roadmap constraints (enforced by `airouter.ParseRoadmap`).

**Architecture:** `Service.Assess` runs: validate → idempotency check (active roadmap exists → return it, no AI, no write) → rate-limit slot → stage answers in Redis → grade (retry once on bad shape) → generate (retry once) → **one transaction** (`UPDATE users`, deactivate old roadmaps, `INSERT roadmaps`, 84 × `INSERT exercises`) → `Pet.Ensure` for the §6.1 `pet_state` → clear the hash → 201. Both AI calls happen before any write, so a 502 writes nothing. Collaborators are interfaces (`Repo`, `QuizStore`, `airouter.RateLimiter`, `Generator` = `*airouter.Router`'s `Route`, `Pet`) so the whole path tests with fakes and a scripted `LLMProvider`. `onboarding` does **not** import `pet`: its `Pet` interface returns its own `PetState`, and `cmd/api/main.go` adapts `*pet.Service` — this keeps `internal/quests`'s in-package integration test free to import `onboarding` without an import cycle (`pet` imports `quests`).

**Tech stack:** Go 1.25 (`backend/go.mod`), Gin, `pgx/v5` (transactions + `pgx.Batch`), `go-redis/v9`. No new dependencies, no migration.

**Depends on:** `store` (1) and `auth` (2) merged; **`quests` (3) merged** (this plan edits `internal/quests/integration_test.go` and deletes `internal/store/seed.go`); **`pet` (4) merged** (`*pet.Service.Ensure(ctx, userID) (pet.State, error)` is adapted in `main.go`); **`airouter` (5) merged** (`Router.Route`, `RateLimiter`, `ErrRateLimited`, `ErrNoProviders`, `ErrAllProvidersFailed`, `RoadmapSystemPrompt`, `RoadmapUserPrompt`, `ParseRoadmap`, `Roadmap`, `Roadmap.Exercises()`, `TaskPlacementTest`, `TaskRoadmapGen`). Verify each symbol against `main` before starting; if any of 3/4/5 is not merged, stop and say so.

**Run every command from `backend/`** unless the step says otherwise. `rg` is not installed — use `grep -n`.

## File structure

| Path | Responsibility |
| --- | --- |
| `backend/internal/onboarding/bank.go` `bank_test.go` | the placement question bank (server-side answers) and its public DTO |
| `backend/internal/onboarding/grade.go` `grade_test.go` | placement prompt + strict `ParsePlacement` |
| `backend/internal/onboarding/types.go` | §6.1 `AssessmentRequest`/`AssessmentResult`/`PetState`, `Pet` and `Generator` seams |
| `backend/internal/onboarding/repo.go` | `Repo` interface + `PgRepo` (`ActiveRoadmapID`, `Profile`, `SaveAssessment` tx) |
| `backend/internal/onboarding/quiz.go` | `QuizStore` + `RedisQuizStore` (`quiz:placement:*`) |
| `backend/internal/onboarding/service.go` `service_test.go` | `Assess`, validation, retry-once, idempotency |
| `backend/internal/onboarding/handler.go` `handler_test.go` | the two routes, §6.1 bodies, error mapping |
| `backend/internal/onboarding/fakes_test.go` | fakes + scripted provider + roadmap fixture |
| `backend/internal/onboarding/integration_test.go` | `TestIntegrationSaveAssessmentPersists84ExercisesAndDeactivatesPrevious` |
| `backend/internal/store/seed.go` `seed_test.go` | **deleted** |
| `backend/internal/quests/integration_test.go` | seed through `onboarding.PgRepo` instead |
| `backend/cmd/api/main.go` | wire the service, adapt `*pet.Service`, mount routes |
| `harness/CODEMAP.md` | `onboarding` paragraph; drop the seed sentence from `store` |

---

## Tasks

### Task 1: The question bank and its public shape

**Files:**
- Create: `backend/internal/onboarding/bank.go`
- Test: `backend/internal/onboarding/bank_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/onboarding/bank_test.go`:
```go
package onboarding

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBankIsWellFormed(t *testing.T) {
	if len(Bank) < 10 {
		t.Fatalf("len(Bank) = %d, want at least 10 items", len(Bank))
	}
	ids := map[string]bool{}
	levels := map[string]bool{"A1": true, "A2": true, "B1": true, "B2": true, "C1": true, "C2": true}
	for _, q := range Bank {
		if ids[q.ID] {
			t.Errorf("duplicate id %q", q.ID)
		}
		ids[q.ID] = true
		if !levels[q.Level] {
			t.Errorf("%s: level %q is not a CEFR level", q.ID, q.Level)
		}
		if _, ok := q.Options[q.Correct]; !ok {
			t.Errorf("%s: correct option %q is not among %v", q.ID, q.Correct, q.Options)
		}
		if len(q.Options) != 4 || strings.TrimSpace(q.Prompt) == "" {
			t.Errorf("%s: want 4 options and a prompt, got %d / %q", q.ID, len(q.Options), q.Prompt)
		}
	}
}

func TestPublicBankNeverLeaksAnswersOrLevels(t *testing.T) {
	b, err := json.Marshal(PublicBank())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	for _, leak := range []string{`"correct"`, `"level"`, `"Correct"`, `"Level"`} {
		if strings.Contains(s, leak) {
			t.Errorf("public quiz leaks %s: %s", leak, s)
		}
	}
	var out struct {
		Questions []struct {
			ID      string            `json:"id"`
			Prompt  string            `json:"prompt"`
			Options map[string]string `json:"options"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Questions) != len(Bank) || out.Questions[0].ID != Bank[0].ID || len(out.Questions[0].Options) != 4 {
		t.Errorf("public shape = %+v", out.Questions[0])
	}
}

func TestLookupFindsQuestionsByID(t *testing.T) {
	q, ok := Lookup(Bank[3].ID)
	if !ok || q.ID != Bank[3].ID {
		t.Errorf("Lookup(%q) = %+v, %t", Bank[3].ID, q, ok)
	}
	if _, ok := Lookup("nope"); ok {
		t.Error("Lookup(nope) found something")
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
mkdir -p internal/onboarding && go test ./internal/onboarding/...
```
Expected: build failure, `undefined: Bank`.

- [ ] **Step 3: Implement**

`backend/internal/onboarding/bank.go`:
```go
// Package onboarding runs the placement test and kicks off the roadmap
// (1st-thinking §5.1 steps 4-5; backend spec §6.1 for the wire DTOs; §4 for
// the quiz:placement hash).
package onboarding

// Question is one placement item. Correct and Level never leave the server;
// they are what the grader prompt is built from.
type Question struct {
	ID      string
	Prompt  string
	Options map[string]string // "A".."D"
	Correct string
	Level   string // the CEFR level the item probes
}

// Bank is the fixed placement set: two items per level A1-C1, ordered by
// difficulty. Ten items keep the grader prompt small and the quiz under five
// minutes; the LLM grader (not a score table) turns the pattern of answers
// into a level, which is what §6.1 asks for.
var Bank = []Question{
	{ID: "q1", Level: "A1", Prompt: "She ___ a teacher.", Options: map[string]string{"A": "am", "B": "is", "C": "are", "D": "be"}, Correct: "B"},
	{ID: "q2", Level: "A1", Prompt: "I ___ to the cinema yesterday.", Options: map[string]string{"A": "go", "B": "goes", "C": "went", "D": "gone"}, Correct: "C"},
	{ID: "q3", Level: "A2", Prompt: "There isn't ___ milk left in the fridge.", Options: map[string]string{"A": "some", "B": "any", "C": "many", "D": "a few"}, Correct: "B"},
	{ID: "q4", Level: "A2", Prompt: "If it rains tomorrow, we ___ at home.", Options: map[string]string{"A": "stay", "B": "will stay", "C": "stayed", "D": "would stay"}, Correct: "B"},
	{ID: "q5", Level: "B1", Prompt: "I've lived here ___ 2019.", Options: map[string]string{"A": "for", "B": "since", "C": "during", "D": "from"}, Correct: "B"},
	{ID: "q6", Level: "B1", Prompt: "The report ___ by the team last week.", Options: map[string]string{"A": "wrote", "B": "was written", "C": "has written", "D": "is writing"}, Correct: "B"},
	{ID: "q7", Level: "B2", Prompt: "Choose the word closest in meaning to 'reluctant'.", Options: map[string]string{"A": "eager", "B": "unwilling", "C": "careless", "D": "confident"}, Correct: "B"},
	{ID: "q8", Level: "B2", Prompt: "Hardly ___ the meeting started when the fire alarm went off.", Options: map[string]string{"A": "had", "B": "has", "C": "did", "D": "was"}, Correct: "A"},
	{ID: "q9", Level: "C1", Prompt: "Her argument was so ___ that even her critics conceded the point.", Options: map[string]string{"A": "tenuous", "B": "cogent", "C": "verbose", "D": "tentative"}, Correct: "B"},
	{ID: "q10", Level: "C1", Prompt: "Had it not been for the delay, we ___ the deadline.", Options: map[string]string{"A": "would meet", "B": "had met", "C": "would have met", "D": "met"}, Correct: "C"},
}

// Lookup finds a bank item by id.
func Lookup(id string) (Question, bool) {
	for _, q := range Bank {
		if q.ID == id {
			return q, true
		}
	}
	return Question{}, false
}

// PublicQuestion is what GET /api/v1/onboarding/quiz shows: no answer, no level.
type PublicQuestion struct {
	ID      string            `json:"id"`
	Prompt  string            `json:"prompt"`
	Options map[string]string `json:"options"`
}

// QuizResponse is the GET /api/v1/onboarding/quiz 200 body. This endpoint is
// not in either spec; see CODEMAP.
type QuizResponse struct {
	Questions []PublicQuestion `json:"questions"`
}

// PublicBank strips the server-only fields.
func PublicBank() QuizResponse {
	out := QuizResponse{Questions: make([]PublicQuestion, 0, len(Bank))}
	for _, q := range Bank {
		out.Questions = append(out.Questions, PublicQuestion{ID: q.ID, Prompt: q.Prompt, Options: q.Options})
	}
	return out
}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/onboarding/... -v
```
Expected: three `--- PASS`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/onboarding && git commit -m "onboarding: placement question bank and its answer-free public shape"
```

---

### Task 2: Grading prompt and strict placement parser

**Files:**
- Create: `backend/internal/onboarding/grade.go`
- Test: `backend/internal/onboarding/grade_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/onboarding/grade_test.go`:
```go
package onboarding

import (
	"errors"
	"strings"
	"testing"
)

func TestPlacementUserPromptCarriesItemsLevelsCorrectAndGivenAnswers(t *testing.T) {
	p := PlacementUserPrompt([]Answer{{QuestionID: "q1", SelectedOption: "B"}, {QuestionID: "q9", SelectedOption: "A"}})
	for _, want := range []string{`"id":"q1"`, `"level":"A1"`, `"correct":"B"`, `"answer":"B"`, `"id":"q9"`, `"level":"C1"`, `"answer":"A"`, `"is_correct":true`, `"is_correct":false`} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt missing %s:\n%s", want, p)
		}
	}
	if strings.Contains(p, "q2") {
		t.Error("prompt includes an unanswered item; only answered items are graded")
	}
}

func TestPlacementSystemPromptDemandsJSONOnly(t *testing.T) {
	for _, want := range []string{"CEFR", `"cefr_level"`, "ONLY"} {
		if !strings.Contains(PlacementSystemPrompt, want) {
			t.Errorf("PlacementSystemPrompt missing %q", want)
		}
	}
}

func TestParsePlacement(t *testing.T) {
	good := map[string]struct{ raw, want string }{
		"plain":       {`{"cefr_level":"B1"}`, "B1"},
		"extra field": {`{"cefr_level":"C2","reasoning":"…"}`, "C2"},
		"whitespace":  {"\n {\"cefr_level\": \"A2\"}\n", "A2"},
	}
	for name, tc := range good {
		t.Run(name, func(t *testing.T) {
			lvl, err := ParsePlacement(tc.raw)
			if err != nil || lvl != tc.want {
				t.Fatalf("ParsePlacement = %q, %v; want %q", lvl, err, tc.want)
			}
		})
	}
	bad := map[string]string{
		"fence":    "```json\n{\"cefr_level\":\"B1\"}\n```",
		"preamble": `The learner is B1: {"cefr_level":"B1"}`,
		"bad enum": `{"cefr_level":"B7"}`,
		"lower":    `{"cefr_level":"b1"}`,
		"missing":  `{"level":"B1"}`,
		"trailing": `{"cefr_level":"B1"} {}`,
		"empty":    ``,
	}
	for name, raw := range bad {
		t.Run(name, func(t *testing.T) {
			if _, err := ParsePlacement(raw); !errors.Is(err, ErrBadAIOutput) {
				t.Errorf("err = %v, want ErrBadAIOutput", err)
			}
		})
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/onboarding/... -run 'Placement'
```
Expected: build failure, `undefined: PlacementUserPrompt`.

- [ ] **Step 3: Implement**

`backend/internal/onboarding/grade.go`:
```go
package onboarding

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrBadAIOutput means the model answered with something that is not the
// requested JSON, twice. Callers map it to 502 and must have written nothing.
var ErrBadAIOutput = errors.New("onboarding: AI output did not match the requested shape")

// Answer is one entry of the §6.1 request's answers array.
type Answer struct {
	QuestionID     string `json:"question_id" binding:"required"`
	SelectedOption string `json:"selected_option" binding:"required"`
}

// PlacementSystemPrompt is the grader's system turn (TaskPlacementTest). §5.1
// only says "Grade"; this is the pet slice-style decision recorded in CODEMAP.
const PlacementSystemPrompt = `You are a CEFR examiner. You receive multiple-choice placement items, each tagged with the CEFR level it probes and its correct option, together with the learner's chosen options. Estimate the learner's overall CEFR level from the pattern of correct and incorrect answers: a learner who is reliably correct up to a level and mostly wrong above it is at that level.

Output ONLY a JSON object of the form {"cefr_level": "<A1|A2|B1|B2|C1|C2>"}. No markdown, no commentary.`

// PlacementUserPrompt serialises the answered items with their level, correct
// option and the learner's choice. Unknown ids are skipped (validation has
// already rejected them).
func PlacementUserPrompt(answers []Answer) string {
	type item struct {
		ID        string `json:"id"`
		Level     string `json:"level"`
		Prompt    string `json:"prompt"`
		Correct   string `json:"correct"`
		Answer    string `json:"answer"`
		IsCorrect bool   `json:"is_correct"`
	}
	items := make([]item, 0, len(answers))
	for _, a := range answers {
		q, ok := Lookup(a.QuestionID)
		if !ok {
			continue
		}
		items = append(items, item{ID: q.ID, Level: q.Level, Prompt: q.Prompt, Correct: q.Correct, Answer: a.SelectedOption, IsCorrect: q.Correct == a.SelectedOption})
	}
	b, _ := json.Marshal(map[string]any{"items": items})
	return "Placement items and the learner's answers:\n" + string(b)
}

var cefrLevels = map[string]bool{"A1": true, "A2": true, "B1": true, "B2": true, "C1": true, "C2": true}

// ParsePlacement decodes {"cefr_level": "..."} strictly: no fence, no
// preamble, no trailing tokens, level in the §3.2 enum (case-sensitive).
func ParsePlacement(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "{") {
		return "", fmt.Errorf("%w: not a JSON object", ErrBadAIOutput)
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	var out struct {
		Level string `json:"cefr_level"`
	}
	if err := dec.Decode(&out); err != nil {
		return "", fmt.Errorf("%w: %v", ErrBadAIOutput, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return "", fmt.Errorf("%w: trailing content", ErrBadAIOutput)
	}
	if !cefrLevels[out.Level] {
		return "", fmt.Errorf("%w: cefr_level %q", ErrBadAIOutput, out.Level)
	}
	return out.Level, nil
}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/onboarding/... -run 'Placement' -v
```
Expected: every `--- PASS` including all `bad` subtests.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/onboarding/grade.go backend/internal/onboarding/grade_test.go && git commit -m "onboarding: placement grading prompt and strict cefr_level parser"
```

---

### Task 3: Types, repository (transactional save) and the quiz hash store

**Files:**
- Create: `backend/internal/onboarding/types.go`
- Create: `backend/internal/onboarding/repo.go`
- Create: `backend/internal/onboarding/quiz.go`

- [ ] **Step 0: Write `types.go`** — the §6.1 DTOs and the two seams onboarding owns

```go
package onboarding

import (
	"context"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
)

// AssessmentRequest is the backend spec §6.1 POST /onboarding/assessment body.
type AssessmentRequest struct {
	TargetGoal       string   `json:"target_goal"`
	NotificationTime string   `json:"notification_time"`
	Timezone         string   `json:"timezone"`
	Answers          []Answer `json:"answers"`
}

// PetState is the §6.1 pet_state object. onboarding defines its own type and
// interface so it never imports pet (see the plan's architecture note).
type PetState struct {
	PlantName    string `json:"plant_name"`
	HealthPoints int    `json:"health_points"`
	Stage        string `json:"stage"`
}

// Pet creates the pet_states row idempotently and reports it. Satisfied in
// cmd/api/main.go by an adapter over *pet.Service.Ensure.
type Pet interface {
	Ensure(ctx context.Context, userID string) (PetState, error)
}

// Generator is the one airouter method onboarding needs; *airouter.Router
// satisfies it.
type Generator interface {
	Route(ctx context.Context, task airouter.TaskType, systemPrompt, userPrompt string) (string, error)
}

// AssessmentResult is the §6.1 response plus Created (201 vs 200), which
// never reaches the wire.
type AssessmentResult struct {
	Status        string   `json:"status"`
	AssessedLevel string   `json:"assessed_level"`
	RoadmapID     string   `json:"roadmap_id"`
	PetState      PetState `json:"pet_state"`
	Created       bool     `json:"-"`
}
```

- [ ] **Step 1: Write `repo.go`**

```go
package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
)

// ErrUnknownUser means the authenticated id has no users row.
var ErrUnknownUser = errors.New("onboarding: unknown user")

// Profile is the slice of users the idempotent path reports back.
type Profile struct {
	CEFRCurrent string
}

// Assessment is everything one successful onboarding writes, in one tx.
type Assessment struct {
	CEFRLevel        string
	TargetGoal       string
	Timezone         string
	NotificationTime string // HH:MM:SS
	Roadmap          airouter.Roadmap
}

// Repo is the Postgres side. onboarding owns roadmaps/exercises writes and
// the users columns onboarding fills (cefr_current, target_goal, timezone,
// notification_time); quests reads roadmaps/exercises afterwards.
type Repo interface {
	// ActiveRoadmapID returns ok=false when the user has no active roadmap.
	ActiveRoadmapID(ctx context.Context, userID string) (id string, ok bool, err error)
	Profile(ctx context.Context, userID string) (Profile, error)
	// SaveAssessment updates the user, deactivates previous roadmaps, inserts
	// the new active roadmap and its 84 exercises, atomically. Returns the
	// roadmap id.
	SaveAssessment(ctx context.Context, userID string, a Assessment) (string, error)
}

const (
	activeRoadmapSQL = `SELECT id FROM roadmaps WHERE user_id = $1 AND is_active = TRUE ORDER BY created_at DESC LIMIT 1`

	profileSQL = `SELECT COALESCE(cefr_current::text, 'A1') FROM users WHERE id = $1`

	updateUserSQL = `
UPDATE users
SET cefr_current = $2::cefr_level, target_goal = $3, timezone = $4, notification_time = $5::time
WHERE id = $1`

	deactivateSQL = `UPDATE roadmaps SET is_active = FALSE WHERE user_id = $1 AND is_active = TRUE`

	insertRoadmapSQL = `INSERT INTO roadmaps (user_id, roadmap_json, is_active) VALUES ($1, $2::jsonb, TRUE) RETURNING id`

	insertExerciseSQL = `
INSERT INTO exercises (roadmap_id, day_number, task_type, content_json)
VALUES ($1, $2, $3::task_category, $4::jsonb)`
)

// PgRepo is the real Repo.
type PgRepo struct{ Pool *pgxpool.Pool }

// NewPgRepo builds a repo over an existing pool.
func NewPgRepo(pool *pgxpool.Pool) *PgRepo { return &PgRepo{Pool: pool} }

func (r *PgRepo) ActiveRoadmapID(ctx context.Context, userID string) (string, bool, error) {
	var id string
	err := r.Pool.QueryRow(ctx, activeRoadmapSQL, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("onboarding: reading active roadmap: %w", err)
	}
	return id, true, nil
}

func (r *PgRepo) Profile(ctx context.Context, userID string) (Profile, error) {
	var p Profile
	err := r.Pool.QueryRow(ctx, profileSQL, userID).Scan(&p.CEFRCurrent)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrUnknownUser
	}
	if err != nil {
		return Profile{}, fmt.Errorf("onboarding: reading profile: %w", err)
	}
	return p, nil
}

func (r *PgRepo) SaveAssessment(ctx context.Context, userID string, a Assessment) (string, error) {
	roadmapJSON, err := json.Marshal(a.Roadmap)
	if err != nil {
		return "", fmt.Errorf("onboarding: encoding roadmap: %w", err)
	}

	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("onboarding: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, updateUserSQL, userID, a.CEFRLevel, a.TargetGoal, a.Timezone, a.NotificationTime)
	if err != nil {
		return "", fmt.Errorf("onboarding: updating user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return "", ErrUnknownUser
	}
	if _, err := tx.Exec(ctx, deactivateSQL, userID); err != nil {
		return "", fmt.Errorf("onboarding: deactivating roadmaps: %w", err)
	}
	var roadmapID string
	if err := tx.QueryRow(ctx, insertRoadmapSQL, userID, roadmapJSON).Scan(&roadmapID); err != nil {
		return "", fmt.Errorf("onboarding: inserting roadmap: %w", err)
	}

	batch := &pgx.Batch{}
	exercises := a.Roadmap.Exercises()
	for _, e := range exercises {
		batch.Queue(insertExerciseSQL, roadmapID, e.DayNumber, e.TaskType, e.ContentJSON)
	}
	results := tx.SendBatch(ctx, batch)
	for i := range exercises {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return "", fmt.Errorf("onboarding: inserting exercise %d: %w", i, err)
		}
	}
	if err := results.Close(); err != nil {
		return "", fmt.Errorf("onboarding: closing batch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("onboarding: commit: %w", err)
	}
	return roadmapID, nil
}

var _ Repo = (*PgRepo)(nil)
```

- [ ] **Step 2: Write `quiz.go`**

```go
package onboarding

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// QuizStore is the §4 quiz:placement:{user_id} Hash — "transient storage for
// active placement test answers before grading". StageAnswers writes
// question_id → selected_option and sets the TTL; Clear removes the hash
// after a successful assessment. A failed grading leaves the answers
// inspectable for the TTL.
type QuizStore interface {
	StageAnswers(ctx context.Context, userID string, answers []Answer, ttl time.Duration) error
	Clear(ctx context.Context, userID string) error
}

// RedisQuizStore is the real QuizStore, keyed by store.PlacementQuizKey.
type RedisQuizStore struct{ Client *redis.Client }

// NewRedisQuizStore builds a store over an existing client.
func NewRedisQuizStore(r *store.Redis) *RedisQuizStore { return &RedisQuizStore{Client: r.Client} }

func (s *RedisQuizStore) StageAnswers(ctx context.Context, userID string, answers []Answer, ttl time.Duration) error {
	key := store.PlacementQuizKey(userID)
	fields := make([]any, 0, 2*len(answers))
	for _, a := range answers {
		fields = append(fields, a.QuestionID, a.SelectedOption)
	}
	pipe := s.Client.TxPipeline()
	pipe.Del(ctx, key) // a re-take replaces, never merges with, stale answers
	pipe.HSet(ctx, key, fields...)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("onboarding: staging answers: %w", err)
	}
	return nil
}

func (s *RedisQuizStore) Clear(ctx context.Context, userID string) error {
	if err := s.Client.Del(ctx, store.PlacementQuizKey(userID)).Err(); err != nil {
		return fmt.Errorf("onboarding: clearing quiz: %w", err)
	}
	return nil
}

var _ QuizStore = (*RedisQuizStore)(nil)
```

- [ ] **Step 3: Confirm it compiles**

```sh
go build ./... && go vet ./internal/onboarding/...
```
Expected: no output.

- [ ] **Step 4: Commit**

```sh
cd .. && git add backend/internal/onboarding/types.go backend/internal/onboarding/repo.go backend/internal/onboarding/quiz.go && git commit -m "onboarding: §6.1 DTOs, Pet/Generator seams, transactional repository and quiz:placement hash store"
```

---

### Task 4: Fakes and the scripted provider

**Files:**
- Create: `backend/internal/onboarding/fakes_test.go`

- [ ] **Step 1: Write the fakes**

```go
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
	activeID  string // "" == none
	profile   Profile
	saved     []Assessment
	nextID    string
	saveErr   error
	activeErr error
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

type fakeQuiz struct {
	staged  map[string][]Answer
	lastTTL time.Duration
	cleared int
}

func newFakeQuiz() *fakeQuiz { return &fakeQuiz{staged: map[string][]Answer{}} }

func (f *fakeQuiz) StageAnswers(_ context.Context, userID string, answers []Answer, ttl time.Duration) error {
	f.staged[userID] = answers
	f.lastTTL = ttl
	return nil
}

func (f *fakeQuiz) Clear(_ context.Context, userID string) error {
	f.cleared++
	delete(f.staged, userID)
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
	prompts map[airouter.TaskType][]string // system|user per call
}

func newScripted() *scriptedProvider {
	return &scriptedProvider{replies: map[airouter.TaskType][]string{}, calls: map[airouter.TaskType]int{}, prompts: map[airouter.TaskType][]string{}}
}

// task is smuggled through the system prompt: the placement and roadmap
// prompts are distinct constants, so the fake tells them apart by content.
func (p *scriptedProvider) GenerateContent(_ context.Context, system, user string) (string, error) {
	task := airouter.TaskRoadmapGen
	if system == PlacementSystemPrompt {
		task = airouter.TaskPlacementTest
	}
	p.prompts[task] = append(p.prompts[task], system+"|"+user)
	i := p.calls[task]
	p.calls[task]++
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
```

- [ ] **Step 2: Confirm it vets**

```sh
go vet ./internal/onboarding/...
```
Expected: no output (`AssessmentRequest`, `PetState` come from Task 3's `types.go`; `go vet` does not flag unused package-level test helpers).

- [ ] **Step 3: Commit**

```sh
cd .. && git add backend/internal/onboarding/fakes_test.go && git commit -m "onboarding: fakes, scripted provider behind a real Router, roadmap fixture"
```

---

### Task 5: `Service.Assess`

**Files:**
- Create: `backend/internal/onboarding/service.go`
- Test: `backend/internal/onboarding/service_test.go`

- [ ] **Step 1: Write the failing tests**

`backend/internal/onboarding/service_test.go`:
```go
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

	out, err := h.svc.Assess(ctx, "u1", validRequest())
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if !out.Created || out.Status != "success" || out.AssessedLevel != "B1" || out.RoadmapID != "rm-new" {
		t.Errorf("out = %+v", out)
	}
	if out.PetState != (PetState{PlantName: "My Green Buddy", HealthPoints: 100, Stage: "sprout"}) {
		t.Errorf("pet_state = %+v", out.PetState)
	}
	if len(h.repo.saved) != 1 {
		t.Fatalf("saved %d assessments, want 1", len(h.repo.saved))
	}
	a := h.repo.saved[0]
	if a.CEFRLevel != "B1" || a.TargetGoal != "IELTS 7.0 Preparation" || a.Timezone != "Asia/Ho_Chi_Minh" || a.NotificationTime != "20:00:00" {
		t.Errorf("assessment = %+v, want the §6.1 request fields", a)
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
	if h.quiz.lastTTL != store.PlacementQuizTTL || h.quiz.lastTTL != 2*time.Hour {
		t.Errorf("quiz TTL = %v, want store.PlacementQuizTTL (2h, §4)", h.quiz.lastTTL)
	}
	if h.quiz.cleared != 1 || len(h.quiz.staged) != 0 {
		t.Error("quiz hash not cleared after success")
	}
	// The generation prompt is the airouter constant and carries the graded level and goal.
	gen := h.ai.prompts[airouter.TaskRoadmapGen][0]
	if !strings.HasPrefix(gen, airouter.RoadmapSystemPrompt+"|") || !strings.Contains(gen, "B1") || !strings.Contains(gen, "IELTS 7.0 Preparation") {
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
		"empty goal":          func(r *AssessmentRequest) { r.TargetGoal = "   " },
		"goal too long":       func(r *AssessmentRequest) { r.TargetGoal = strings.Repeat("x", 256) },
		"bad timezone":        func(r *AssessmentRequest) { r.Timezone = "Mars/Olympus" },
		"empty timezone":      func(r *AssessmentRequest) { r.Timezone = "" },
		"bad time":            func(r *AssessmentRequest) { r.NotificationTime = "8pm" },
		"no answers":          func(r *AssessmentRequest) { r.Answers = nil },
		"unknown question":    func(r *AssessmentRequest) { r.Answers[0].QuestionID = "q99" },
		"unknown option":      func(r *AssessmentRequest) { r.Answers[0].SelectedOption = "E" },
		"duplicate question":  func(r *AssessmentRequest) { r.Answers[1].QuestionID = r.Answers[0].QuestionID },
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
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/onboarding/... -run Assess
```
Expected: build failure, `undefined: NewService`.

- [ ] **Step 3: Implement**

`backend/internal/onboarding/service.go`:
```go
package onboarding

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// ErrInvalidRequest wraps every validation failure (400).
var ErrInvalidRequest = errors.New("onboarding: invalid request")

// DailyMinutes is the study commitment the roadmap prompt is built for (§1).
const DailyMinutes = 30

// Service runs the §5.1 steps 4-5 flow. The request/response DTOs and the
// Pet/Generator seams are in types.go.
type Service struct {
	repo    Repo
	quiz    QuizStore
	limiter airouter.RateLimiter
	ai      Generator
	pet     Pet
	now     func() time.Time
}

// NewService wires the collaborators.
func NewService(repo Repo, quiz QuizStore, limiter airouter.RateLimiter, ai Generator, pet Pet, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, quiz: quiz, limiter: limiter, ai: ai, pet: pet, now: now}
}

// Assess validates, short-circuits when a roadmap is already active, then
// grades, generates and persists — both AI calls before any write, so a
// failure writes nothing.
func (s *Service) Assess(ctx context.Context, userID string, req AssessmentRequest) (AssessmentResult, error) {
	if err := validate(req); err != nil {
		return AssessmentResult{}, err
	}

	if id, ok, err := s.repo.ActiveRoadmapID(ctx, userID); err != nil {
		return AssessmentResult{}, err
	} else if ok {
		profile, err := s.repo.Profile(ctx, userID)
		if err != nil {
			return AssessmentResult{}, err
		}
		pet, err := s.pet.Ensure(ctx, userID)
		if err != nil {
			return AssessmentResult{}, err
		}
		return AssessmentResult{Status: "success", AssessedLevel: profile.CEFRCurrent, RoadmapID: id, PetState: pet, Created: false}, nil
	}

	// One slot per assessment, not per LLM call, so the retry-once path can
	// never trip the §4 limit on its own.
	if err := s.limiter.Allow(ctx, userID); err != nil {
		return AssessmentResult{}, err
	}

	if err := s.quiz.StageAnswers(ctx, userID, req.Answers, store.PlacementQuizTTL); err != nil {
		return AssessmentResult{}, err
	}

	var level string
	if err := s.routeJSON(ctx, airouter.TaskPlacementTest, PlacementSystemPrompt, PlacementUserPrompt(req.Answers), func(raw string) error {
		lvl, err := ParsePlacement(raw)
		level = lvl
		return err
	}); err != nil {
		return AssessmentResult{}, err
	}

	var roadmap airouter.Roadmap
	if err := s.routeJSON(ctx, airouter.TaskRoadmapGen, airouter.RoadmapSystemPrompt, airouter.RoadmapUserPrompt(level, req.TargetGoal, DailyMinutes), func(raw string) error {
		rm, err := airouter.ParseRoadmap(raw)
		roadmap = rm
		return err
	}); err != nil {
		return AssessmentResult{}, err
	}

	roadmapID, err := s.repo.SaveAssessment(ctx, userID, Assessment{
		CEFRLevel:        level,
		TargetGoal:       strings.TrimSpace(req.TargetGoal),
		Timezone:         req.Timezone,
		NotificationTime: req.NotificationTime,
		Roadmap:          roadmap,
	})
	if err != nil {
		return AssessmentResult{}, err
	}

	pet, err := s.pet.Ensure(ctx, userID)
	if err != nil {
		return AssessmentResult{}, err
	}
	if err := s.quiz.Clear(ctx, userID); err != nil {
		log.Printf("onboarding: clearing quiz hash for %s: %v", userID, err) // it expires anyway
	}
	return AssessmentResult{Status: "success", AssessedLevel: level, RoadmapID: roadmapID, PetState: pet, Created: true}, nil
}

// routeJSON calls the router and parses; a malformed body is retried once,
// then reported as ErrBadAIOutput. Provider/router errors are returned as-is
// (the router has already fallen back across providers).
func (s *Service) routeJSON(ctx context.Context, task airouter.TaskType, system, user string, parse func(string) error) error {
	var last error
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := s.ai.Route(ctx, task, system, user)
		if err != nil {
			return err
		}
		if err := parse(raw); err != nil {
			last = err
			log.Printf("onboarding: %s attempt %d returned a malformed body: %v", task, attempt+1, err)
			continue
		}
		return nil
	}
	if errors.Is(last, ErrBadAIOutput) {
		return last
	}
	return fmt.Errorf("%w: %v", ErrBadAIOutput, last)
}

func validate(req AssessmentRequest) error {
	goal := strings.TrimSpace(req.TargetGoal)
	if goal == "" || len(goal) > 255 {
		return fmt.Errorf("%w: target_goal must be 1..255 characters", ErrInvalidRequest)
	}
	if req.Timezone == "" {
		return fmt.Errorf("%w: timezone is required", ErrInvalidRequest)
	}
	if _, err := time.LoadLocation(req.Timezone); err != nil {
		return fmt.Errorf("%w: timezone %q is not an IANA zone", ErrInvalidRequest, req.Timezone)
	}
	if _, err := time.Parse("15:04:05", req.NotificationTime); err != nil {
		return fmt.Errorf("%w: notification_time must be HH:MM:SS", ErrInvalidRequest)
	}
	if len(req.Answers) == 0 {
		return fmt.Errorf("%w: answers must not be empty", ErrInvalidRequest)
	}
	seen := map[string]bool{}
	for _, a := range req.Answers {
		q, ok := Lookup(a.QuestionID)
		if !ok {
			return fmt.Errorf("%w: unknown question_id %q", ErrInvalidRequest, a.QuestionID)
		}
		if _, ok := q.Options[a.SelectedOption]; !ok {
			return fmt.Errorf("%w: %s has no option %q", ErrInvalidRequest, a.QuestionID, a.SelectedOption)
		}
		if seen[a.QuestionID] {
			return fmt.Errorf("%w: %s answered twice", ErrInvalidRequest, a.QuestionID)
		}
		seen[a.QuestionID] = true
	}
	return nil
}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/onboarding/... -v
```
Expected: every `--- PASS`, including all validation subtests.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/onboarding/service.go backend/internal/onboarding/service_test.go && git commit -m "onboarding: Assess — validate, idempotent, rate-limited, grade + generate with retry-once, single-tx persist"
```

---

### Task 6: The two routes

**Files:**
- Create: `backend/internal/onboarding/handler.go`
- Test: `backend/internal/onboarding/handler_test.go`

- [ ] **Step 1: Write the failing tests**

`backend/internal/onboarding/handler_test.go`:
```go
package onboarding

import (
	"encoding/json"
	"errors"
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
	return r
}

func post(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/assessment", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

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
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/onboarding/... -run 'Quiz|Assessment'
```
Expected: build failure, `undefined: QuizHandler`.

- [ ] **Step 3: Implement**

`backend/internal/onboarding/handler.go`:
```go
package onboarding

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// QuizHandler serves GET /api/v1/onboarding/quiz — the placement items
// without answers. Not in either spec (see CODEMAP); mount behind
// auth.Require() so the bank is not public.
func QuizHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if auth.UserID(c) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusOK, PublicBank())
	}
}

// AssessmentHandler serves POST /api/v1/onboarding/assessment (backend spec
// §6.1): 201 on a new roadmap, 200 when one was already active. Mount behind
// auth.Require().
func AssessmentHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var req AssessmentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}

		out, err := svc.Assess(c.Request.Context(), userID, req)
		switch {
		case errors.Is(err, ErrInvalidRequest):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		case errors.Is(err, airouter.ErrRateLimited):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
		case errors.Is(err, airouter.ErrNoProviders):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ai_unavailable"})
		case errors.Is(err, ErrBadAIOutput):
			c.JSON(http.StatusBadGateway, gin.H{"error": "ai_bad_output"})
		case errors.Is(err, airouter.ErrAllProvidersFailed):
			c.JSON(http.StatusBadGateway, gin.H{"error": "ai_upstream_failed"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		case out.Created:
			c.JSON(http.StatusCreated, out)
		default:
			c.JSON(http.StatusOK, out)
		}
	}
}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/onboarding/... -v
```
Expected: every `--- PASS`, including the exact §6.1 body comparison.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/onboarding/handler.go backend/internal/onboarding/handler_test.go && git commit -m "onboarding: GET /onboarding/quiz and POST /onboarding/assessment with the §6.1 body and error mapping"
```

---

### Task 7: Integration test for the transaction

**Files:**
- Create: `backend/internal/onboarding/integration_test.go`

Gated on `TEST_DATABASE_URL` only; `TestIntegration…` so CI counts it.

- [ ] **Step 1: Write the test**

```go
package onboarding

import (
	"context"
	"os"
	"testing"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// Gated on TEST_DATABASE_URL, never the production DATABASE_URL (spec §9);
// internal/store's tests drop every table in the database they point at. CI
// exports TEST_* and fails on --- SKIP; run with -p 1 (make test-integration).
func TestIntegrationSaveAssessmentPersists84ExercisesAndDeactivatesPrevious(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is unset; run `make up` and export it to run integration tests")
	}
	ctx := context.Background()
	pg, err := store.NewPostgres(ctx, url)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	t.Cleanup(pg.Close)
	if _, err := store.Migrate(ctx, pg.Migrator(), store.MigrationsFS); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	const gid = "google-onboarding-integration"
	_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)
	var userID string
	if err := pg.Pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, target_goal) VALUES ($1, $2, '') RETURNING id`,
		"onboarding@example.com", gid).Scan(&userID); err != nil {
		t.Fatalf("inserting user: %v", err)
	}
	t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid) })

	repo := NewPgRepo(pg.Pool)
	if _, ok, err := repo.ActiveRoadmapID(ctx, userID); err != nil || ok {
		t.Fatalf("ActiveRoadmapID before = ok:%t err:%v, want none", ok, err)
	}

	first, err := repo.SaveAssessment(ctx, userID, Assessment{CEFRLevel: "B1", TargetGoal: "IELTS 7.0", Timezone: "Asia/Ho_Chi_Minh", NotificationTime: "20:00:00", Roadmap: fixtureRoadmap()})
	if err != nil {
		t.Fatalf("first SaveAssessment: %v", err)
	}
	second, err := repo.SaveAssessment(ctx, userID, Assessment{CEFRLevel: "B2", TargetGoal: "TOEFL 100", Timezone: "UTC", NotificationTime: "07:30:00", Roadmap: fixtureRoadmap()})
	if err != nil {
		t.Fatalf("second SaveAssessment: %v", err)
	}
	if first == second {
		t.Fatal("second save returned the first roadmap id")
	}

	id, ok, err := repo.ActiveRoadmapID(ctx, userID)
	if err != nil || !ok || id != second {
		t.Errorf("ActiveRoadmapID = %q ok:%t err:%v, want the second roadmap", id, ok, err)
	}
	var active, total int
	if err := pg.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE is_active), count(*) FROM roadmaps WHERE user_id = $1`, userID).Scan(&active, &total); err != nil {
		t.Fatalf("counting roadmaps: %v", err)
	}
	if active != 1 || total != 2 {
		t.Errorf("roadmaps: active=%d total=%d, want 1 of 2", active, total)
	}

	var exercises, days, vocab int
	if err := pg.Pool.QueryRow(ctx,
		`SELECT count(*), count(DISTINCT day_number), count(*) FILTER (WHERE task_type = 'vocabulary') FROM exercises WHERE roadmap_id = $1`, second).Scan(&exercises, &days, &vocab); err != nil {
		t.Fatalf("counting exercises: %v", err)
	}
	if exercises != 84 || days != 28 || vocab != 28 {
		t.Errorf("exercises=%d days=%d vocabulary=%d, want 84/28/28", exercises, days, vocab)
	}
	var title string
	var minutes int
	if err := pg.Pool.QueryRow(ctx,
		`SELECT content_json->>'title', (content_json->>'duration_minutes')::int FROM exercises WHERE roadmap_id = $1 AND day_number = 1 AND task_type = 'reading'`, second).Scan(&title, &minutes); err != nil {
		t.Fatalf("reading content_json: %v", err)
	}
	if title != "reading task" || minutes != 10 {
		t.Errorf("content_json title/duration = %q/%d — quests' toTask reads exactly these keys", title, minutes)
	}

	var cefr, goal, tz, notif string
	if err := pg.Pool.QueryRow(ctx,
		`SELECT cefr_current::text, target_goal, timezone, notification_time::text FROM users WHERE id = $1`, userID).Scan(&cefr, &goal, &tz, &notif); err != nil {
		t.Fatalf("reading user: %v", err)
	}
	if cefr != "B2" || goal != "TOEFL 100" || tz != "UTC" || notif != "07:30:00" {
		t.Errorf("user = %s/%s/%s/%s, want the second assessment's values", cefr, goal, tz, notif)
	}
	p, err := repo.Profile(ctx, userID)
	if err != nil || p.CEFRCurrent != "B2" {
		t.Errorf("Profile = %+v, %v", p, err)
	}
}
```

- [ ] **Step 2: Confirm it skips without services**

```sh
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1
go test ./internal/onboarding/... -count=1 -run Integration -v
```
Expected: `ok` everywhere; `--- SKIP: TestIntegrationSaveAssessmentPersists84ExercisesAndDeactivatesPrevious`.

- [ ] **Step 3: Commit**

```sh
cd .. && git add backend/internal/onboarding/integration_test.go && git commit -m "onboarding: integration test — one active roadmap, 84 exercises, users columns"
```

---

### Task 8: Retire `store.SeedDemoRoadmap`; seed the quests integration test through onboarding

**Files:**
- Delete: `backend/internal/store/seed.go`, `backend/internal/store/seed_test.go`
- Modify: `backend/internal/quests/integration_test.go`

`internal/quests/integration_test.go` is `package quests` and may import `onboarding` because
`onboarding` imports only `airouter`, `auth`, `store` — never `quests` or `pet`. Verify:
`grep -n '"github.com/HendrixNguyen/English-Training-Harness/backend/internal/' internal/onboarding/*.go`
must show no `quests` or `pet` import.

- [ ] **Step 1: Delete the seed**

```sh
git rm internal/store/seed.go internal/store/seed_test.go
grep -rn 'SeedDemoRoadmap\|DemoRoadmapDays\|DemoTaskTypes' --include='*.go' .
```
Expected: the only remaining hits are in `internal/quests/integration_test.go`.

- [ ] **Step 2: Re-point the quests integration test**

In `internal/quests/integration_test.go` (as merged from the quests plan's Task 7), add imports:
```go
	"encoding/json"
	"fmt"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/onboarding"
```
Replace:
```go
	if _, err := store.SeedDemoRoadmap(ctx, pg.Pool, userID); err != nil {
		t.Fatalf("SeedDemoRoadmap: %v", err)
	}
```
with:
```go
	// Seeded the way production does it: through onboarding's repository, so
	// this test breaks if onboarding stops writing title/duration_minutes.
	if _, err := onboarding.NewPgRepo(pg.Pool).SaveAssessment(ctx, userID, onboarding.Assessment{
		CEFRLevel: "B1", TargetGoal: "integration", Timezone: "UTC", NotificationTime: "20:00:00",
		Roadmap: integrationRoadmap(),
	}); err != nil {
		t.Fatalf("SaveAssessment: %v", err)
	}
```
and append to the file:
```go
// integrationRoadmap is a valid 4x7x3 roadmap whose tasks carry the title and
// duration_minutes the §6.2 daily response exposes.
func integrationRoadmap() airouter.Roadmap {
	r := airouter.Roadmap{Title: "Integration", CEFRLevel: "B1"}
	for m := 1; m <= airouter.Modules; m++ {
		mod := airouter.Module{Week: m, Title: fmt.Sprintf("Week %d", m), Focus: "integration"}
		for d := 1; d <= airouter.DaysPerModule; d++ {
			day := airouter.Day{Title: fmt.Sprintf("Day %d", (m-1)*airouter.DaysPerModule+d)}
			for _, tt := range airouter.TaskTypes {
				day.Tasks = append(day.Tasks, airouter.Task{Type: tt, Title: "Day task: " + tt, DurationMinutes: 10, Content: json.RawMessage(`{}`)})
			}
			mod.Days = append(mod.Days, day)
		}
		r.Modules = append(r.Modules, mod)
	}
	return r
}
```
The existing assertion `suite.Tasks[0].Title == "" || suite.Tasks[0].DurationMinutes != 10` keeps
passing because `content_json` is the task object.

- [ ] **Step 3: Confirm build and the skip-locally property**

```sh
go build ./... && go vet ./... && env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1
grep -rn 'SeedDemoRoadmap' --include='*.go' . ; echo "hits above must be zero"
```
Expected: clean; no hits.

- [ ] **Step 4: Run the integration suite for real if Docker is available**

```sh
POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose up -d --wait
export TEST_DATABASE_URL='postgres://english:english@localhost:5433/english?sslmode=disable'
export TEST_REDIS_URL='redis://localhost:6380/0'
make test-integration
docker compose down
```
Expected: `--- PASS` for every `TestIntegration*` (store, auth, quests, pet, airouter, onboarding), no `--- SKIP`. If Docker is unavailable, say so; CI is the gate.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend/internal/store/seed.go backend/internal/store/seed_test.go backend/internal/quests/integration_test.go && git commit -m "onboarding: retire store.SeedDemoRoadmap; quests integration test seeds through onboarding's repo"
```

---

### Task 9: Wire it in `main.go`

**Files:**
- Modify: `backend/cmd/api/main.go`

After the pet plan's Task 9 and the airouter plan's Task 8, `main.go` has `aiRouter`, `petSvc`, `guarded`. Verify with `grep -n 'aiRouter\|petSvc\|guarded' cmd/api/main.go`.

- [ ] **Step 1: Add the adapter and the wiring**

Add `"github.com/HendrixNguyen/English-Training-Harness/backend/internal/onboarding"` to the imports. Below `func main()` (package level):
```go
// petForOnboarding adapts *pet.Service to onboarding.Pet. onboarding defines
// its own PetState so it never imports pet (pet imports quests; quests' tests
// import onboarding — an import here would be a cycle).
type petForOnboarding struct{ svc *pet.Service }

func (p petForOnboarding) Ensure(ctx context.Context, userID string) (onboarding.PetState, error) {
	st, err := p.svc.Ensure(ctx, userID)
	if err != nil {
		return onboarding.PetState{}, err
	}
	return onboarding.PetState{PlantName: st.PlantName, HealthPoints: st.HealthPoints, Stage: st.Stage}, nil
}
```
After the `guarded.POST("/pet/revive", …)` line:
```go
	onboardingSvc := onboarding.NewService(
		onboarding.NewPgRepo(pg.Pool),
		onboarding.NewRedisQuizStore(rdb),
		airouter.NewRedisRateLimiter(rdb),
		aiRouter,
		petForOnboarding{svc: petSvc},
		time.Now,
	)
	guarded.GET("/onboarding/quiz", onboarding.QuizHandler())
	guarded.POST("/onboarding/assessment", onboarding.AssessmentHandler(onboardingSvc))
```

- [ ] **Step 2: Confirm build, vet and tests**

```sh
go build ./... && go vet ./... && go test ./... -count=1
grep -n '/onboarding/quiz\|/onboarding/assessment\|petForOnboarding' cmd/api/main.go
```
Expected: clean; four hits.

- [ ] **Step 3: Commit**

```sh
cd .. && git add backend/cmd/api/main.go && git commit -m "onboarding: mount quiz and assessment routes; adapt pet.Service for the §6.1 pet_state"
```

---

### Task 10: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (`**onboarding**` bullet; remove the `SeedDemoRoadmap` sentence from `**store**`)

- [ ] **Step 1: Replace the onboarding bullet**

```
- **onboarding** — placement and roadmap kickoff (1st-thinking §5.1 steps 4–5; wire contract = backend spec §6.1). `GET /api/v1/onboarding/quiz` (**not in either spec — added here** so the client has questions to answer; the spec's owner should add it to §6.1/§7) returns `{questions[{id, prompt, options{A..D}}]}` from the fixed ten-item `Bank` (two per level A1–C1); correct options and levels never leave the server. `POST /api/v1/onboarding/assessment` takes `{target_goal, notification_time, timezone, answers[{question_id, selected_option}]}` and answers `201 {status: "success", assessed_level, roadmap_id, pet_state{plant_name, health_points, stage}}`. Order: validate (400 `invalid_request`) → if an active roadmap exists return it with `200`, no AI call, no write → one `ratelimit:ai` slot per assessment (429 `rate_limited`) → `HSET quiz:placement:{user_id}` with the answers + `EXPIRE` 2h (§4; a failed grading leaves them inspectable) → `TaskPlacementTest` with `PlacementSystemPrompt` (items with level + correct option + learner's choice) parsed by `ParsePlacement` → `TaskRoadmapGen` with `airouter.RoadmapSystemPrompt`/`RoadmapUserPrompt(level, goal, 30)` validated by `airouter.ParseRoadmap` → **one transaction**: `UPDATE users SET cefr_current, target_goal, timezone, notification_time`, `UPDATE roadmaps SET is_active = FALSE …`, `INSERT roadmaps (roadmap_json = the validated JSON, is_active = TRUE)`, 84 × `INSERT exercises` via `pgx.Batch` with **`content_json` = the task object** (`type, title, duration_minutes, content` — what quests' `toTask` reads) → `Pet.Ensure` → `DEL` the hash. Each AI call is retried **once** on a malformed body, then 502 `ai_bad_output` with nothing written; `ErrNoProviders` → 503 `ai_unavailable`; `ErrAllProvidersFailed` → 502 `ai_upstream_failed`. `onboarding` does not import `pet`: `cmd/api/main.go`'s `petForOnboarding` adapts `*pet.Service.Ensure` to `onboarding.Pet`, which keeps `quests`' in-package integration test free to seed through `onboarding.PgRepo.SaveAssessment` (the TEMPORARY `store.SeedDemoRoadmap` is gone). Tests are pure (scripted `LLMProvider` behind a real `airouter.Router`, fakes for repo/quiz/limiter/pet); `TestIntegrationSaveAssessmentPersists84ExercisesAndDeactivatesPrevious` is gated on `TEST_DATABASE_URL` (skips locally, must pass in CI).
```

- [ ] **Step 2: Remove the seed sentence from the store bullet**

Delete the sentence beginning `` `store.SeedDemoRoadmap` inserts a 28-day … `` from the `**store**` bullet.

- [ ] **Step 3: Verify and commit**

```sh
grep -c 'SeedDemoRoadmap' harness/CODEMAP.md
grep -n 'petForOnboarding\|onboarding/quiz' harness/CODEMAP.md
git add harness/CODEMAP.md && git commit -m "codemap: onboarding — §6.1 flow, quiz endpoint addition, seed retired"
```
Expected: `1` (the mention that it is gone), then two hits.

---

## Verification

Run from the worktree root.

```sh
cd backend && go build ./... && go vet ./...
# expect: no output

env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1
# expect: ok for every package incl. internal/onboarding — no live service, no network

go test ./internal/onboarding/... -run 'Assess' -v
# expect: --- PASS — happy path (84 rows, users fields, TTL 2h, one limiter slot), idempotent (no AI/no write), retry-once then 502 with zero writes, 429, validation

go test ./internal/onboarding/... -run 'Placement|Bank|Public' -v
# expect: --- PASS — strict cefr_level parser, bank shape, no answer leak

go test ./internal/onboarding/... -run 'Quiz|Assessment|ErrorMapping' -v
# expect: --- PASS — exact §6.1 201 body, 200 on re-submit, 400/429/502/503/500 codes

grep -n '"assessed_level"\|"roadmap_id"\|"pet_state"\|"plant_name"\|"health_points"\|"stage"' internal/onboarding/types.go
# expect: 6 hits — §6.1 response fields

grep -n '"target_goal"\|"notification_time"\|"timezone"\|"answers"\|"question_id"\|"selected_option"' internal/onboarding/types.go internal/onboarding/grade.go
# expect: 6 hits — §6.1 request fields

grep -n 'store.PlacementQuizKey\|store.PlacementQuizTTL' internal/onboarding/quiz.go internal/onboarding/service.go
# expect: 3 hits — onboarding never hand-builds the §4 key or TTL

grep -n 'airouter.ParseRoadmap\|airouter.RoadmapSystemPrompt\|airouter.TaskPlacementTest\|airouter.TaskRoadmapGen' internal/onboarding/service.go
# expect: 4 hits

grep -n 'Begin(ctx)\|Commit(ctx)\|SendBatch' internal/onboarding/repo.go
# expect: 3 hits — the single transaction

grep -rn '"github.com/HendrixNguyen/English-Training-Harness/backend/internal/pet"\|internal/quests"' internal/onboarding/
# expect: no hits — onboarding imports neither pet nor quests

grep -rn 'SeedDemoRoadmap' --include='*.go' . ; ls internal/store/seed.go 2>&1 | grep -c 'No such file'
# expect: no Go hits; then 1

grep -n 'onboarding.NewPgRepo' internal/quests/integration_test.go
# expect: 1 hit

grep -rn 'Getenv("DATABASE_URL")\|Getenv("REDIS_URL")' internal/onboarding/
# expect: no hits

grep -c '^func TestIntegration' internal/onboarding/integration_test.go
# expect: 1

grep -n 'petForOnboarding\|/onboarding/assessment' cmd/api/main.go
# expect: 3 hits

cd .. && python3 tools/harness/cli.py validate; echo exit=$?
# expect: exit=0

git log --oneline main..HEAD
# expect: 10 commits, one per task, each with the Co-Authored-By trailer

git status --short
# expect: clean
```

After pushing: `gh run list --branch <branch>` must show all three jobs green; `backend-integration` runs the new test and the re-pointed quests test.

## Notes and open questions

- **`GET /onboarding/quiz` is a spec addition.** Both specs list only `/auth/google` and `/onboarding/assessment` for this area. Kept because the §6.1 request references `question_id`s the client must obtain somewhere; the spec owner should add the endpoint to backend §6.1 and 1st-thinking §7 (a low spec bug for the reviewer to file if not).
- **Retaking the placement.** An active roadmap makes re-submission a no-op `200`. If the product wants "retake and regenerate", add an explicit `force` flag or a `DELETE`; the deactivate-previous step in the tx already supports it.
- **Grading by LLM rather than a score table.** §6.1 says "invoking Gemini AI for CEFR grading"; with a ten-item bank a deterministic mapping would be trivial and cheaper. Kept LLM-based per spec; the prompt gives it the per-item level and correctness so the answer is grounded.
- **The bank is ten hand-written items.** Enough for MVP flow; content quality and item count are product work. Items live in Go, not the database — no §3.2 table for them.
- **`quiz:placement` holds answers, not questions.** §4 says "answers before grading", so the hash is written by the POST, not the GET. If the frontend needs "resume an in-progress quiz", the GET would also write, and the TTL would then mean "quiz session".
- **One rate-limit slot per assessment** (two LLM calls, up to two retries). Per-call accounting would let one bad-shape retry consume 40 % of a user's minute.
- **`notification_time` and `timezone` are written here** because §6.1 puts them on this request; `POST /settings/notifications` (notify slice) will also write `notification_time`. Both writers, same column, last wins.
- **Placement uses `TaskPlacementTest` → gemini; generation `TaskRoadmapGen` → gemini** — one provider outage still degrades gracefully via the router's fallback.
- **Pet row creation timing.** `Pet.Ensure` runs after the roadmap commit, so a pet failure returns 500 after the roadmap exists; the next call is the idempotent path and returns both. The spec bug `spec-never-states-when-the-pet-states-row-is-created` can be closed with "on first `GET /pet/status` or at onboarding, idempotently".
