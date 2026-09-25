---
idea: harness/ideas/2026-09-24-run-01/typed-task-content-with-answer-keys-so-every-quest-renders-a.md
status: done
priority: medium
merged: false
branch: harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a
worktree: .worktrees/typed-task-content-with-answer-keys-so-every-quest-renders-a
---
# Typed task content (backend half): a per-type `content` schema with answer keys, asked for in the prompt and enforced by `ParseRoadmap` — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Feature team — ticket **F2** of 2026-09-24. **Estimate:** 5 h. **Branch:** `harness/2026-09-24-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a`.
**Approval:** medium feature, approved by the owner's 2026-09-24 planning instruction (override of the auto-approve rule; recorded in the idea's `## Evaluation`).

**Idea:** `harness/ideas/2026-09-24-run-01/typed-task-content-with-answer-keys-so-every-quest-renders-a.md` — this plan is **part 1 of 2 (backend)**. Part 2 (frontend rendering with instant feedback and a score) is a separate plan once this has merged.

**Goal:** Every task the roadmap model returns carries `content` in a fixed, validated shape per task type — vocabulary `{words[], questions[]}`, reading `{passage, questions[]}`, practice `{questions[]}`, each question with `options{A..D}`, `answer` ∈ options and a one-line `explanation` — so no new roadmap can ever reach the client as unrenderable JSON, and today's frontend (`utils/content.ts`: `words` → word list, `questions` → quiz) renders all of it without a change.

**Architecture:** `airouter` only. A new file `content.go` owns the typed content structs, the bounds and `validateContent(task Task) error`; `ParseRoadmap` (in `roadmap.go`) gains exactly **one call** in its task loop. `prompt.go`'s `RoadmapSchema` string states the shape and the bounds. `Task.Content` stays `json.RawMessage`, so `Exercises()`, `quests.toTask` and `roadmaps.roadmap_json` are byte-for-byte unaffected — the validator decodes a copy and discards it. Roadmaps already stored (free-form content) are untouched; the frontend keeps its `raw` fallback for them.

**Decisions (from the evaluation):** `answer` and `explanation` travel to the client (offline-capable, low stakes) — no server-side grading; **no score persistence** here (nothing reads it; `POST /quests/progress` already accepts `user_answers` unpersisted) — part 2 decides where a score lives; validation in a new file so Bug ticket B2's edits to the same loop do not collide.

**Tech stack:** Go stdlib (`encoding/json`, `strings`, `unicode/utf8`).

**Spec:** 1st-thinking §6.1 (three task categories; "requested schema"); §3.2 `exercises.content_json`; Frontend spec §5 (7.3 task view renders the content). The shapes below are a **superset** of what `frontend/utils/content.ts` already recognises (`words[{term, definition}]`, `questions[{id, prompt, options}]`).

**⚠ Conflict note:** Bug ticket B2 (`harness/plans/2026-09-24-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md`) edits `ParseRoadmap`'s loop and appends rows to `TestParseRoadmapRejects` the same day. Keep this plan's `roadmap.go` change to the single `validateContent` call and put **all** new tests in `content_test.go` (none in `roadmap_test.go` except the fixture change in Task 4). If `origin/main` has B2 when you start: `git fetch origin main && git merge origin/main --no-edit`, keep both sides.

## Global Constraints

- Work in `.worktrees/<slug>`; never edit the main checkout (AGENTS.md). Run from `backend/`.
- `rg`/`timeout` not installed: `grep -n`, `go test -timeout 60s`.
- Every rejection wraps `ErrInvalidRoadmap` via the existing `invalid(...)` and names module/day/task 1-based like the existing messages.
- Unknown extra fields inside `content` are tolerated (same policy as the rest of `ParseRoadmap`); missing or wrong-typed **required** fields are rejected. Nothing is repaired.
- `content` is **required** for every task from now on (`{}`/absent → reject). Both test fixtures (`airouter/roadmap_test.go` `validRoadmapJSON`, `onboarding/fakes_test.go` `fixtureRoadmap`) must produce typed content.
- The prompt keeps `RoadmapSystemPrompt` verbatim (§6.1; `prompt_test.go` pins it) — only `RoadmapSchema` changes.
- `gofmt -l internal/airouter` prints nothing.

## Review Focus

1. `answer: "e"` (lower-case, or a key not in `options`) → rejected with a message naming the question id. Task 2.
2. `options` with 3 keys, or keys other than `A..D` → rejected (the frontend renders `options` as-is, so the shape must be exact). Task 2.
3. A vocabulary `words[]` entry whose `example` is missing → rejected? **No** — `example` is optional (today's frontend `Word` has only `term`/`definition`); `term` and `definition` are required and non-blank. Task 2 test "example optional".
4. A reading `passage` of 4000+ characters (a model that pastes an article) → rejected by `maxPassageRunes`; test uses `strings.Repeat`. Task 2.
5. The prompt's schema text and the validator's bounds must agree — Task 3's test asserts the numbers in `RoadmapSchema` are the constants, so one cannot drift from the other.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/airouter/content.go` | **New**: `Question`, `Word`, `VocabularyContent`, `ReadingContent`, `PracticeContent`; bounds constants; `validateContent(task Task, where string) error` |
| `backend/internal/airouter/content_test.go` | **New**: acceptance per type, rejection table, bounds |
| `backend/internal/airouter/roadmap.go` | One call: `if err := validateContent(*task, fmt.Sprintf("module %d day %d task %d", mi+1, di+1, ti+1)); err != nil { return Roadmap{}, err }` |
| `backend/internal/airouter/roadmap_test.go` | `validRoadmapJSON` builds typed content via `sampleContent(tt)` |
| `backend/internal/airouter/prompt.go` | `RoadmapSchema` states the three content shapes and bounds |
| `backend/internal/airouter/prompt_test.go` | Asserts the schema text carries the bounds constants |
| `backend/internal/onboarding/fakes_test.go` | `fixtureRoadmap` uses `airouter.SampleContent(tt)` |
| `harness/CODEMAP.md` | `airouter` bullet: content contract |

---

## Tasks

### Task 1: The types, the bounds and the happy path

**Files:**
- Create: `backend/internal/airouter/content.go`
- Test: `backend/internal/airouter/content_test.go`

**Interfaces (produces):**
```go
type Question struct { ID, Prompt string; Options map[string]string; Answer, Explanation string }
type Word struct { Term, Definition, Example string }            // Example optional
type VocabularyContent struct { Words []Word; Questions []Question }
type ReadingContent struct { Passage string; Questions []Question }
type PracticeContent struct { Questions []Question }
const (
	MinWords, MaxWords                     = 5, 8
	MinVocabQuestions, MaxVocabQuestions   = 3, 5
	MinReadingQuestions, MaxReadingQuestions = 3, 5
	MinPracticeQuestions, MaxPracticeQuestions = 4, 8
	minPassageRunes, maxPassageRunes       = 200, 2000
)
var optionKeys = []string{"A", "B", "C", "D"}
func validateContent(task Task, where string) error
func SampleContent(taskType string) json.RawMessage   // exported: test fixtures in onboarding use it
```

- [ ] **Step 1: Write the failing acceptance test** (`content_test.go`):

```go
package airouter

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestValidateContentAcceptsEachTypeAndTheSamples(t *testing.T) {
	for _, tt := range TaskTypes {
		t.Run(tt, func(t *testing.T) {
			task := Task{Type: tt, Title: "t", DurationMinutes: 10, Content: SampleContent(tt)}
			if err := validateContent(task, "module 1 day 1 task 1"); err != nil {
				t.Fatalf("sample %s content rejected: %v", tt, err)
			}
		})
	}
	// The samples are also what today's frontend renders: vocabulary has words,
	// reading and practice have questions (frontend/utils/content.ts).
	var v VocabularyContent
	if err := json.Unmarshal(SampleContent("vocabulary"), &v); err != nil || len(v.Words) < MinWords {
		t.Fatalf("vocabulary sample: %v / %d words", err, len(v.Words))
	}
	var r ReadingContent
	if err := json.Unmarshal(SampleContent("reading"), &r); err != nil || r.Passage == "" || len(r.Questions) < MinReadingQuestions {
		t.Fatalf("reading sample: %v", err)
	}
}
```

- [ ] **Step 2: Run** — `go test -timeout 60s ./internal/airouter -run TestValidateContent -v` → FAIL (undefined).

- [ ] **Step 3: Implement** `content.go`:

```go
package airouter

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Task content shapes. These are what the §6.1 "requested schema" leaves
// unstated and what exercises.content_json therefore carries. They are a
// superset of what frontend/utils/content.ts already renders (words[] →
// word list, questions[] → quiz), so a roadmap that passes here renders
// today; answer/explanation enable instant feedback in the next slice.

// Question is one multiple-choice item. Options has exactly the keys A..D;
// Answer is one of them.
type Question struct {
	ID          string            `json:"id"`
	Prompt      string            `json:"prompt"`
	Options     map[string]string `json:"options"`
	Answer      string            `json:"answer"`
	Explanation string            `json:"explanation"`
}

// Word is one vocabulary item; Example is optional.
type Word struct {
	Term       string `json:"term"`
	Definition string `json:"definition"`
	Example    string `json:"example,omitempty"`
}

type VocabularyContent struct {
	Words     []Word     `json:"words"`
	Questions []Question `json:"questions"`
}

type ReadingContent struct {
	Passage   string     `json:"passage"`
	Questions []Question `json:"questions"`
}

type PracticeContent struct {
	Questions []Question `json:"questions"`
}

// Bounds: a ~10-minute task (§6.1). RoadmapSchema quotes these numbers;
// TestRoadmapSchemaStatesTheContentBounds keeps them in step.
const (
	MinWords, MaxWords                         = 5, 8
	MinVocabQuestions, MaxVocabQuestions       = 3, 5
	MinReadingQuestions, MaxReadingQuestions   = 3, 5
	MinPracticeQuestions, MaxPracticeQuestions = 4, 8
	minPassageRunes, maxPassageRunes           = 200, 2000
)

var optionKeys = []string{"A", "B", "C", "D"}

// validateContent decodes a copy of task.Content into the shape for task.Type
// and checks counts, option keys, answers and non-blank text. where names the
// task ("module 1 day 2 task 3") for the error. task.Content itself is never
// modified — roadmap_json and content_json store the model's bytes.
func validateContent(task Task, where string) error {
	if len(task.Content) == 0 || strings.TrimSpace(string(task.Content)) == "{}" {
		return invalid("%s has no content", where)
	}
	switch task.Type {
	case "vocabulary":
		var c VocabularyContent
		if err := json.Unmarshal(task.Content, &c); err != nil {
			return invalid("%s content: %v", where, err)
		}
		if n := len(c.Words); n < MinWords || n > MaxWords {
			return invalid("%s has %d words, want %d..%d", where, n, MinWords, MaxWords)
		}
		for i, w := range c.Words {
			if strings.TrimSpace(w.Term) == "" || strings.TrimSpace(w.Definition) == "" {
				return invalid("%s word %d needs term and definition", where, i+1)
			}
		}
		return validateQuestions(c.Questions, MinVocabQuestions, MaxVocabQuestions, where)
	case "reading":
		var c ReadingContent
		if err := json.Unmarshal(task.Content, &c); err != nil {
			return invalid("%s content: %v", where, err)
		}
		if n := utf8.RuneCountInString(strings.TrimSpace(c.Passage)); n < minPassageRunes || n > maxPassageRunes {
			return invalid("%s passage is %d characters, want %d..%d", where, n, minPassageRunes, maxPassageRunes)
		}
		return validateQuestions(c.Questions, MinReadingQuestions, MaxReadingQuestions, where)
	case "practice":
		var c PracticeContent
		if err := json.Unmarshal(task.Content, &c); err != nil {
			return invalid("%s content: %v", where, err)
		}
		return validateQuestions(c.Questions, MinPracticeQuestions, MaxPracticeQuestions, where)
	}
	return invalid("%s has type %q", where, task.Type) // unreachable: isTaskType ran first
}

func validateQuestions(qs []Question, min, max int, where string) error {
	if n := len(qs); n < min || n > max {
		return invalid("%s has %d questions, want %d..%d", where, n, min, max)
	}
	seen := map[string]bool{}
	for i, q := range qs {
		id := strings.TrimSpace(q.ID)
		if id == "" || seen[id] {
			return invalid("%s question %d needs a unique id", where, i+1)
		}
		seen[id] = true
		if strings.TrimSpace(q.Prompt) == "" {
			return invalid("%s question %s has no prompt", where, id)
		}
		if len(q.Options) != len(optionKeys) {
			return invalid("%s question %s has %d options, want %s", where, id, len(q.Options), strings.Join(optionKeys, ""))
		}
		for _, k := range optionKeys {
			if strings.TrimSpace(q.Options[k]) == "" {
				return invalid("%s question %s is missing option %s", where, id, k)
			}
		}
		if _, ok := q.Options[q.Answer]; !ok {
			return invalid("%s question %s answer %q is not one of its options", where, id, q.Answer)
		}
		if strings.TrimSpace(q.Explanation) == "" {
			return invalid("%s question %s has no explanation", where, id)
		}
	}
	return nil
}

// SampleContent is a minimal valid content object per task type, for test
// fixtures here and in onboarding. It is not a prompt example.
func SampleContent(taskType string) json.RawMessage {
	q := func(id string) Question {
		return Question{ID: id, Prompt: "Choose the correct form: She ___ a teacher.", Options: map[string]string{"A": "am", "B": "is", "C": "are", "D": "be"}, Answer: "B", Explanation: "Third person singular takes 'is'."}
	}
	questions := func(n int) []Question {
		out := make([]Question, 0, n)
		for i := 1; i <= n; i++ {
			out = append(out, q(fmt.Sprintf("q%d", i)))
		}
		return out
	}
	var v any
	switch taskType {
	case "vocabulary":
		words := make([]Word, 0, MinWords)
		for i := 1; i <= MinWords; i++ {
			words = append(words, Word{Term: fmt.Sprintf("term%d", i), Definition: "a definition", Example: "An example sentence."})
		}
		v = VocabularyContent{Words: words, Questions: questions(MinVocabQuestions)}
	case "reading":
		v = ReadingContent{Passage: strings.Repeat("The learner reads a short passage about daily habits. ", 5), Questions: questions(MinReadingQuestions)}
	default:
		v = PracticeContent{Questions: questions(MinPracticeQuestions)}
	}
	b, _ := json.Marshal(v)
	return b
}
```

- [ ] **Step 4: Run** → PASS. **Step 5: Commit** — `gofmt -l internal/airouter; git add internal/airouter/content.go internal/airouter/content_test.go && git commit -m "airouter: typed task content shapes, bounds and validator"`.

### Task 2: The rejection table

**Files:**
- Test: `backend/internal/airouter/content_test.go` (append)

- [ ] **Step 1: Write the table** (these already pass against Task 1's code — run them to prove the validator rejects what it must; any that passes validation is a bug to fix in `content.go` before committing):

```go
func TestValidateContentRejects(t *testing.T) {
	mutQ := func(taskType string, f func(qs []Question)) json.RawMessage {
		var m map[string]any
		_ = json.Unmarshal(SampleContent(taskType), &m)
		raw, _ := json.Marshal(m["questions"])
		var qs []Question
		_ = json.Unmarshal(raw, &qs)
		f(qs)
		m["questions"] = qs
		b, _ := json.Marshal(m)
		return b
	}
	cases := map[string]Task{
		"absent content":              {Type: "practice"},
		"empty object":                {Type: "practice", Content: json.RawMessage(`{}`)},
		"not an object":               {Type: "practice", Content: json.RawMessage(`[1,2]`)},
		"too few words":               {Type: "vocabulary", Content: func() json.RawMessage { var c VocabularyContent; _ = json.Unmarshal(SampleContent("vocabulary"), &c); c.Words = c.Words[:2]; b, _ := json.Marshal(c); return b }()},
		"blank definition":            {Type: "vocabulary", Content: func() json.RawMessage { var c VocabularyContent; _ = json.Unmarshal(SampleContent("vocabulary"), &c); c.Words[0].Definition = "  "; b, _ := json.Marshal(c); return b }()},
		"short passage":               {Type: "reading", Content: func() json.RawMessage { var c ReadingContent; _ = json.Unmarshal(SampleContent("reading"), &c); c.Passage = "Too short."; b, _ := json.Marshal(c); return b }()},
		"article-length passage":      {Type: "reading", Content: func() json.RawMessage { var c ReadingContent; _ = json.Unmarshal(SampleContent("reading"), &c); c.Passage = strings.Repeat("word ", 900); b, _ := json.Marshal(c); return b }()},
		"blank question id":           {Type: "practice", Content: mutQ("practice", func(qs []Question) { qs[0].ID = "" })},
		"three practice questions":    {Type: "practice", Content: func() json.RawMessage { var c PracticeContent; _ = json.Unmarshal(SampleContent("practice"), &c); c.Questions = c.Questions[:3]; b, _ := json.Marshal(c); return b }()},
		"answer not in options":       {Type: "practice", Content: mutQ("practice", func(qs []Question) { qs[1].Answer = "E" })},
		"lower-case answer":           {Type: "practice", Content: mutQ("practice", func(qs []Question) { qs[1].Answer = "b" })},
		"three options":               {Type: "reading", Content: mutQ("reading", func(qs []Question) { delete(qs[0].Options, "D") })},
		"option key outside A..D":     {Type: "reading", Content: mutQ("reading", func(qs []Question) { qs[0].Options["E"] = qs[0].Options["D"]; delete(qs[0].Options, "D") })},
		"blank option text":           {Type: "vocabulary", Content: mutQ("vocabulary", func(qs []Question) { qs[0].Options["C"] = "" })},
		"duplicate question id":       {Type: "practice", Content: mutQ("practice", func(qs []Question) { qs[2].ID = qs[1].ID })},
		"no explanation":              {Type: "practice", Content: mutQ("practice", func(qs []Question) { qs[3].Explanation = "" })},
		"no prompt":                   {Type: "practice", Content: mutQ("practice", func(qs []Question) { qs[0].Prompt = " " })},
	}
	for name, task := range cases {
		t.Run(name, func(t *testing.T) {
			if task.Title == "" {
				task.Title, task.DurationMinutes = "t", 10
			}
			err := validateContent(task, "module 1 day 1 task 1")
			if err == nil {
				t.Fatal("validateContent accepted it")
			}
			if !errors.Is(err, ErrInvalidRoadmap) || !strings.Contains(err.Error(), "module 1 day 1 task 1") {
				t.Errorf("err = %v, want ErrInvalidRoadmap naming the task", err)
			}
		})
	}
}

func TestValidateContentToleratesOptionalExampleAndUnknownFields(t *testing.T) {
	var c VocabularyContent
	_ = json.Unmarshal(SampleContent("vocabulary"), &c)
	for i := range c.Words {
		c.Words[i].Example = ""
	}
	b, _ := json.Marshal(c)
	// add an unknown field the way a model might
	b = append(b[:len(b)-1], []byte(`,"difficulty":"B1"}`)...)
	if err := validateContent(Task{Type: "vocabulary", Title: "t", DurationMinutes: 10, Content: b}, "module 1 day 1 task 1"); err != nil {
		t.Fatalf("optional example / unknown field rejected: %v", err)
	}
}
```

- [ ] **Step 2: Run** — `go test -timeout 60s ./internal/airouter -run TestValidateContent -v` → every row PASS (i.e. each is rejected). If a row is accepted, fix `content.go` — do not delete the row.

- [ ] **Step 3: Commit** — `git add internal/airouter/content_test.go && git commit -m "airouter: content validator rejection table"`.

### Task 3: The prompt states the shapes; `ParseRoadmap` enforces them

**Files:**
- Modify: `backend/internal/airouter/prompt.go` (`RoadmapSchema`)
- Modify: `backend/internal/airouter/roadmap.go` (one call in the task loop; doc comment)
- Test: `backend/internal/airouter/prompt_test.go`, `backend/internal/airouter/roadmap_test.go` (`validRoadmapJSON`)

- [ ] **Step 1: Write the failing tests** — append to `prompt_test.go`:

```go
func TestRoadmapSchemaStatesTheContentShapesAndBounds(t *testing.T) {
	for _, want := range []string{
		`"words"`, `"term"`, `"definition"`, `"example"`, `"passage"`, `"questions"`, `"options"`, `"A"`, `"D"`, `"answer"`, `"explanation"`,
		fmt.Sprintf("%d-%d words", MinWords, MaxWords),
		fmt.Sprintf("%d-%d questions", MinReadingQuestions, MaxReadingQuestions),
		fmt.Sprintf("%d-%d questions", MinPracticeQuestions, MaxPracticeQuestions),
		fmt.Sprintf("%d-%d characters", minPassageRunes, maxPassageRunes),
	} {
		if !strings.Contains(RoadmapSchema, want) {
			t.Errorf("RoadmapSchema missing %q", want)
		}
	}
	if strings.Contains(RoadmapSchema, "free-form") {
		t.Error("RoadmapSchema still calls content free-form")
	}
}
```

(add `fmt` to the imports), and in `roadmap_test.go` change `validRoadmapJSON`'s task line to `Content: SampleContent(tt)` and add a rejection row at the **end** of `TestParseRoadmapRejects`:

```go
		"free-form content": validRoadmapJSON(t, func(r *Roadmap) { r.Modules[0].Days[0].Tasks[2].Content = json.RawMessage(`{"items":[]}`) }),
```

- [ ] **Step 2: Run** — `go test -timeout 60s ./internal/airouter` → `TestRoadmapSchemaStates…` FAIL; `free-form content` row FAIL (accepted).

- [ ] **Step 3: Implement** — `RoadmapSchema` becomes:

```go
const RoadmapSchema = `{
  "title": "string",
  "cefr_level": "A1|A2|B1|B2|C1|C2",
  "modules": [
    {
      "week": 1,
      "title": "string",
      "focus": "string",
      "days": [
        {
          "title": "string",
          "tasks": [
            {"type": "vocabulary", "title": "string", "duration_minutes": 10,
             "content": {"words": [{"term": "string", "definition": "string", "example": "string"}],
                         "questions": [QUESTION]}},
            {"type": "reading",    "title": "string", "duration_minutes": 10,
             "content": {"passage": "string", "questions": [QUESTION]}},
            {"type": "practice",   "title": "string", "duration_minutes": 10,
             "content": {"questions": [QUESTION]}}
          ]
        }
      ]
    }
  ]
}
QUESTION = {"id": "q1", "prompt": "string", "options": {"A": "string", "B": "string", "C": "string", "D": "string"}, "answer": "A|B|C|D", "explanation": "string"}
"modules" has exactly 4 entries, each "days" exactly 7, each "tasks" exactly 3 with the three types in that order.
"content" is required and typed by task: vocabulary has 5-8 words (each with term and definition; example optional) and 3-5 questions; reading has one passage of 200-2000 characters in English at the learner's level and 3-5 questions about it; practice has 4-8 questions. Every question has exactly options A, B, C, D, an "answer" that is one of those four letters, and a one-sentence "explanation". Question ids are unique within a task.`
```

Then make the numbers come from the constants so the test's `fmt.Sprintf` and the text agree: since Go `const` strings cannot interpolate, define `RoadmapSchema` as a `var` built once with `fmt.Sprintf` over a template containing `%d-%d words` etc. — **or** keep the literal and make the test compare literals to the constants (`MinWords == 5 && MaxWords == 8 …`). Choose the `var RoadmapSchema = fmt.Sprintf(roadmapSchemaTemplate, MinWords, MaxWords, MinVocabQuestions, MaxVocabQuestions, minPassageRunes, maxPassageRunes, MinReadingQuestions, MaxReadingQuestions, MinPracticeQuestions, MaxPracticeQuestions)` form: `RoadmapUserPrompt` already uses `%s` with it and `prompt_test.go` references it as a value, so a `var` is a drop-in. Escape the template's literal `%` (none exist) and keep `QUESTION` as plain text.

In `roadmap.go`, inside the task loop after the duration checks (last statement of the loop body):

```go
				if err := validateContent(*task, fmt.Sprintf("module %d day %d task %d", mi+1, di+1, ti+1)); err != nil {
					return Roadmap{}, err
				}
```

and extend the `ParseRoadmap` doc comment with "…, and typed, bounded `content` per task (see content.go)".

- [ ] **Step 4: Run** — `go test -timeout 60s ./internal/airouter` → PASS. **Step 5: Commit** — `git add internal/airouter/prompt.go internal/airouter/prompt_test.go internal/airouter/roadmap.go internal/airouter/roadmap_test.go && git commit -m "airouter: the roadmap prompt asks for typed content and ParseRoadmap enforces it"`.

### Task 4: Onboarding's fixture and the whole backend still pass

**Files:**
- Modify: `backend/internal/onboarding/fakes_test.go:118-127` (`fixtureRoadmap`)

- [ ] **Step 1: Run** `go test -timeout 120s ./internal/onboarding` → FAIL (`… has no content`), proving the fixture is now held to the contract.
- [ ] **Step 2: Change** the task line to `Content: airouter.SampleContent(tt)`.
- [ ] **Step 3: Run** `go test -timeout 300s ./...` → PASS (quests reads only `title`/`duration_minutes` from `content_json`; google reads `->>'title'`; neither cares about `content`).
- [ ] **Step 4: Commit** — `git add internal/onboarding/fakes_test.go && git commit -m "onboarding: roadmap fixture carries typed content"`.

### Task 5: CODEMAP

- [ ] **Step 1:** In the `airouter` bullet, replace `RoadmapSchema is the JSON shape §6.1 leaves unstated — {title, cefr_level, modules[4]{week,title,focus,days[7]{title,tasks[3]{type,title,duration_minutes,content}}}}` with `RoadmapSchema is the JSON shape §6.1 leaves unstated — {title, cefr_level, modules[4]{week,title,focus,days[7]{title,tasks[3]{type,title,duration_minutes,content}}}} where **content is typed per task** (content.go): vocabulary {words[5..8]{term, definition, example?}, questions[3..5]}, reading {passage 200..2000 chars, questions[3..5]}, practice {questions[4..8]}; every question {id (unique), prompt, options{A,B,C,D}, answer ∈ A..D, explanation}. answer/explanation reach the client in content_json (instant feedback is the frontend's next slice); the frontend's words/questions renderers already match these shapes, and its raw fallback covers roadmaps stored before this contract` and add `validateContent` to the `ParseRoadmap rejects …` list as `content that misses its type's shape or bounds`.
- [ ] **Step 2: Commit** — `git add harness/CODEMAP.md && git commit -m "codemap: typed task content contract"`.

## Verification

```bash
cd backend && gofmt -l . ; go vet ./... && go test -timeout 120s ./internal/airouter -count=1 -run 'Content|Schema|ParseRoadmap' -v 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
go test -timeout 300s ./...
```

Expected: no gofmt output; `ok …/internal/airouter`; whole backend PASS. Push; `gh run list --branch <branch>` green. Optional live check with `GEMINI_API_KEY` set: run one `POST /onboarding/assessment` against a scratch DB and `psql -c "select content_json->'content'->'questions'->0->>'answer' from exercises where task_type='practice' limit 1"` → a letter.

## Notes and open questions

- **Part 2 (frontend)** — a separate plan after this merges: `utils/content.ts` gains typed branches (`reading` shows the passage; questions grade locally against `answer`, show `explanation`, end with "4/5"); decide then whether a score is persisted (new `daily_progress` field or `exercises` column + spec DDL) — nothing here forecloses either.
- Larger answers: Bug ticket B3 (Gemini `maxOutputTokens 32768`) makes this safer; if `ai_bad_output` rates rise after both land, the first knob is `MaxPracticeQuestions`/passage length, the second is the model.
- Spaced repetition (`spaced-repetition-vocabulary-review…`, selected low) can now key on `words[].term`; re-rank it after this merges.

## Execution summary

Built exactly per plan: Tasks 1–5 implemented and committed one per plan step, no deviations from the interfaces, bounds, or file structure specified. Branch derived per skill step 4 using today's date (2026-09-25) rather than the plan header's 2026-09-24: `harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a`. One extra commit beyond the plan's five: `gofmt -w` had reformatted the long `cases := map[string]Task{...}` literal in Task 2's rejection table after the initial commit (multi-line struct literals gofmt normally collapses only when they fit; this one didn't) — committed separately as "airouter: gofmt content_test.go" rather than silently amending. `origin/main` did not yet have Bug ticket B2 when the worktree was cut, so the plan's conflict-note merge step was not needed.

**Plan verification (as specified):**
```
$ cd backend && gofmt -l . ; go vet ./... && go test -timeout 120s ./internal/airouter -count=1 -run 'Content|Schema|ParseRoadmap' -v 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter	0.414s

$ go test -timeout 300s ./...
ok  	.../backend/cmd/api
ok  	.../backend/internal/airouter
ok  	.../backend/internal/auth
ok  	.../backend/internal/config
ok  	.../backend/internal/google
ok  	.../backend/internal/health
ok  	.../backend/internal/middleware
ok  	.../backend/internal/notify
ok  	.../backend/internal/onboarding
ok  	.../backend/internal/pet
ok  	.../backend/internal/quests
ok  	.../backend/internal/secrets
ok  	.../backend/internal/store
```
No gofmt output; whole backend PASS.

**Runtime proof (Definition of done, step 8):**
1. **Build** — `go build -o /tmp/typed-content-api ./cmd/api` — clean, no errors/warnings.
2. **Whole suite, clean shell** — `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL -u GEMINI_API_KEY -u OPENAI_API_KEY -u DEEPSEEK_API_KEY -u JWT_SECRET -u ENCRYPTION_SECRET_KEY -u GOOGLE_CLIENT_ID -u GOOGLE_CLIENT_SECRET go test -timeout 300s ./... -count=1` — all 13 packages `ok`. Also ran `make check` (CI's exact `fmt-check && vet && go test ./... -count=1 -race`) — green.
3. **Boots and answers** — scratch `backend/.env` with `COMPOSE_PROJECT_NAME=typed-content`, `POSTGRES_PORT=55442`, `REDIS_PORT=56442`, `PORT=8142`; `docker compose up -d --wait --wait-timeout 120` (project `typed-content` only); built binary run with those vars — logged `migrations applied: [0001_init 0002_google_sync 0003_pet_verdict_dates]` then `listening on [::]:8142`. `curl --max-time 10 http://localhost:8142/healthz` → `200 {"postgres":"ok","redis":"ok","status":"ok"}`. `curl .../api/v1/onboarding/quiz` → `401 {"error":"unauthorized"}` (correct — no session), proving routing/middleware live.
4. **Documented commands** — `make check` (green, above); `make test-integration` with `TEST_DATABASE_URL=postgres://english:english@localhost:55442/...` / `TEST_REDIS_URL=redis://localhost:56442/0` pointed at the scratch stack — all Integration tests passed, including `TestIntegrationSaveAssessmentPersists84ExercisesAndDeactivatesPrevious` in `internal/onboarding`, which exercises exactly the fixture this plan changed (`fixtureRoadmap` → `airouter.SampleContent`) end to end against a live Postgres. No GEMINI/OPENAI/DEEPSEEK key was available in this environment, so the plan's *optional* live `POST /onboarding/assessment` + `psql` check was not run; `/healthz` plus the onboarding integration test are the real path exercised instead.
5. **CI on the branch** — green, all 4 jobs (`harness-tooling`, `backend-unit`, `frontend`, `backend-integration`): https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36106459659
6. **Cleanup** — killed the app process (`pgrep -fl typed-content-api` empty afterward), `docker compose down` for project `typed-content` (`docker ps --filter name=typed-content` empty afterward), deleted the scratch `backend/.env`; `git status --short` in the worktree clean before pushing.

**Branch:** `harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a` — pushed, no PR opened.
