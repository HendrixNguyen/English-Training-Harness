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
