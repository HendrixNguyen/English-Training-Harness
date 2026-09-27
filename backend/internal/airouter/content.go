package airouter

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Task content shapes. These are what the §6.1 "requested schema" leaves
// unstated and what exercises.content_json therefore carries. They are a
// superset of what frontend/utils/content.ts already renders (words[] →
// word list, questions[] → quiz), so content that passes here renders today;
// answer/explanation enable instant feedback in the next slice.

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

// Bounds: a ~10-minute task (§6.1). The exercise prompt (exercise.go) quotes
// these numbers; TestExerciseUserPromptNamesOnlyItsShape keeps them in step.
const (
	MinWords, MaxWords                         = 5, 8
	MinVocabQuestions, MaxVocabQuestions       = 3, 5
	MinReadingQuestions, MaxReadingQuestions   = 3, 5
	MinPracticeQuestions, MaxPracticeQuestions = 4, 8
)

// PassageWords is the one table of reading-passage lengths per CEFR level.
// The exercise prompt quotes it and ValidateContent enforces it (in runes).
var passageWords = map[string][2]int{"A1": {60, 90}, "A2": {90, 130}, "B1": {130, 180}, "B2": {180, 250}, "C1": {250, 320}, "C2": {300, 380}}

// PassageWords returns level's word bounds; an unknown level falls back to B1.
func PassageWords(level string) (min, max int) {
	b, ok := passageWords[level]
	if !ok {
		b = passageWords["B1"]
	}
	return b[0], b[1]
}

// passageRunes converts PassageWords to a rune-count range: min×4, max×8.
// That is deliberately generous on both ends (English runs well under 4
// runes/word and rarely past 8 with punctuation and spaces) so it rejects
// only the egregiously short or long passage, not a plausible one that runs
// a little tight or loose on words.
func passageRunes(level string) (min, max int) {
	lo, hi := PassageWords(level)
	return lo * 4, hi * 8
}

// ErrInvalidContent wraps every ValidateContent rejection.
var ErrInvalidContent = errors.New("airouter: invalid exercise content")

var optionKeys = []string{"A", "B", "C", "D"}

// invalidContent wraps ErrInvalidContent with a "content: <rule>" message.
// Unlike the roadmap's invalid(), it never names a module/day/task location:
// the caller already knows which exercise it is.
func invalidContent(format string, args ...any) error {
	return fmt.Errorf("content: %s: %w", fmt.Sprintf(format, args...), ErrInvalidContent)
}

// ValidateContent decodes content into the shape for taskType and checks
// counts, option keys, answers, non-blank text and, for reading, the
// level's passage bounds. content itself is never modified — content_json
// stores the model's bytes.
func ValidateContent(taskType, level string, content json.RawMessage) error {
	if len(content) == 0 || strings.TrimSpace(string(content)) == "{}" {
		return invalidContent("no content")
	}
	switch taskType {
	case "vocabulary":
		var c VocabularyContent
		if err := json.Unmarshal(content, &c); err != nil {
			return invalidContent("%v", err)
		}
		if n := len(c.Words); n < MinWords || n > MaxWords {
			return invalidContent("has %d words, want %d..%d", n, MinWords, MaxWords)
		}
		for i, w := range c.Words {
			if strings.TrimSpace(w.Term) == "" || strings.TrimSpace(w.Definition) == "" {
				return invalidContent("word %d needs term and definition", i+1)
			}
		}
		return validateQuestions(c.Questions, MinVocabQuestions, MaxVocabQuestions)
	case "reading":
		var c ReadingContent
		if err := json.Unmarshal(content, &c); err != nil {
			return invalidContent("%v", err)
		}
		min, max := passageRunes(level)
		if n := utf8.RuneCountInString(strings.TrimSpace(c.Passage)); n < min || n > max {
			return invalidContent("passage is %d characters, want %d..%d for %s", n, min, max, level)
		}
		return validateQuestions(c.Questions, MinReadingQuestions, MaxReadingQuestions)
	case "practice":
		var c PracticeContent
		if err := json.Unmarshal(content, &c); err != nil {
			return invalidContent("%v", err)
		}
		return validateQuestions(c.Questions, MinPracticeQuestions, MaxPracticeQuestions)
	}
	return invalidContent("unknown task type %q", taskType) // unreachable: caller already checked isTaskType
}

func validateQuestions(qs []Question, min, max int) error {
	if n := len(qs); n < min || n > max {
		return invalidContent("has %d questions, want %d..%d", n, min, max)
	}
	seen := map[string]bool{}
	for i, q := range qs {
		id := strings.TrimSpace(q.ID)
		if id == "" || seen[id] {
			return invalidContent("question %d needs a unique id", i+1)
		}
		seen[id] = true
		if strings.TrimSpace(q.Prompt) == "" {
			return invalidContent("question %s has no prompt", id)
		}
		if len(q.Options) != len(optionKeys) {
			return invalidContent("question %s has %d options, want %s", id, len(q.Options), strings.Join(optionKeys, ""))
		}
		for _, k := range optionKeys {
			if strings.TrimSpace(q.Options[k]) == "" {
				return invalidContent("question %s is missing option %s", id, k)
			}
		}
		if _, ok := q.Options[q.Answer]; !ok {
			return invalidContent("question %s answer %q is not one of its options", id, q.Answer)
		}
		if strings.TrimSpace(q.Explanation) == "" {
			return invalidContent("question %s has no explanation", id)
		}
	}
	return nil
}

// SampleContent is a minimal valid content object per task type, for test
// fixtures here and in onboarding. It is not a prompt example. Its reading
// passage is sized for B1 (520..1440 runes).
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
		v = ReadingContent{Passage: strings.Repeat("The learner reads a short passage about daily habits and practises new words in context. ", 6), Questions: questions(MinReadingQuestions)}
	default:
		v = PracticeContent{Questions: questions(MinPracticeQuestions)}
	}
	b, _ := json.Marshal(v)
	return b
}
