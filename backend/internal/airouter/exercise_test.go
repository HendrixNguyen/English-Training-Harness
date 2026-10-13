package airouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExerciseUserPromptNamesOnlyItsShape(t *testing.T) {
	base := ExerciseBrief{Level: "C2", Goal: "pass an interview", ModuleFocus: "workplace English", DayTitle: "Meetings", TaskTitle: "Vocabulary warm-up"}

	t.Run("vocabulary", func(t *testing.T) {
		b := base
		b.TaskType = "vocabulary"
		p := ExerciseUserPrompt(b)
		for _, want := range []string{b.Level, b.Goal, b.ModuleFocus, b.DayTitle, b.TaskTitle, "words", fmt.Sprintf("%d-%d", MinWords, MaxWords), fmt.Sprintf("%d-%d", MinVocabQuestions, MaxVocabQuestions)} {
			if !strings.Contains(p, want) {
				t.Errorf("vocabulary prompt missing %q:\n%s", want, p)
			}
		}
		if strings.Contains(p, `"passage":`) {
			t.Errorf("vocabulary prompt should not request a passage field:\n%s", p)
		}
	})

	t.Run("reading", func(t *testing.T) {
		b := base
		b.TaskType = "reading"
		p := ExerciseUserPrompt(b)
		lo, hi := PassageWords("C2")
		for _, want := range []string{b.Level, b.Goal, b.ModuleFocus, b.DayTitle, b.TaskTitle, "passage", fmt.Sprintf("%d-%d words", lo, hi), fmt.Sprintf("%d-%d", MinReadingQuestions, MaxReadingQuestions)} {
			if !strings.Contains(p, want) {
				t.Errorf("reading prompt missing %q:\n%s", want, p)
			}
		}
	})

	t.Run("practice", func(t *testing.T) {
		b := base
		b.TaskType = "practice"
		p := ExerciseUserPrompt(b)
		for _, want := range []string{b.Level, b.Goal, b.ModuleFocus, b.DayTitle, b.TaskTitle, fmt.Sprintf("%d-%d", MinPracticeQuestions, MaxPracticeQuestions)} {
			if !strings.Contains(p, want) {
				t.Errorf("practice prompt missing %q:\n%s", want, p)
			}
		}
		if strings.Contains(p, `"passage":`) || strings.Contains(p, `"words":`) {
			t.Errorf("practice prompt should not request passage or words fields:\n%s", p)
		}
	})

	t.Run("draft line only when non-empty and not {}", func(t *testing.T) {
		const marker = "curriculum's draft for this task"
		b := base
		b.TaskType = "practice"
		b.Draft = ""
		if strings.Contains(ExerciseUserPrompt(b), marker) {
			t.Error("empty draft produced a draft line")
		}
		b.Draft = "{}"
		if strings.Contains(ExerciseUserPrompt(b), marker) {
			t.Error("{} draft produced a draft line")
		}
		b.Draft = `{"topic":"small talk"}`
		if !strings.Contains(ExerciseUserPrompt(b), marker) {
			t.Error("a real draft produced no draft line")
		}
	})

	t.Run("a 2,000-rune draft is cut to 600", func(t *testing.T) {
		b := base
		b.TaskType = "practice"
		b.Draft = strings.Repeat("d", 2000)
		p := ExerciseUserPrompt(b)
		if strings.Contains(p, strings.Repeat("d", 601)) {
			t.Error("draft was not cut to 600 runes")
		}
		if !strings.Contains(p, strings.Repeat("d", 600)) {
			t.Error("draft was cut to fewer than 600 runes")
		}
	})
}

func TestParseExerciseContent(t *testing.T) {
	for _, tt := range TaskTypes {
		t.Run(tt, func(t *testing.T) {
			out, err := ParseExerciseContent(tt, "B1", string(SampleContent(tt)))
			if err != nil {
				t.Fatalf("SampleContent(%s) rejected: %v", tt, err)
			}
			if len(out) == 0 {
				t.Fatal("empty output")
			}
		})
	}

	sample := string(SampleContent("practice"))
	cases := map[string]string{
		"fenced":           "```json\n" + sample + "\n```",
		"preamble":         "Here is the content:\n" + sample,
		"trailing content": sample + "}",
		"json array":       "[" + sample + "]",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseExerciseContent("practice", "B1", raw); !errors.Is(err, ErrInvalidContent) {
				t.Errorf("%s: err = %v, want ErrInvalidContent", name, err)
			}
		})
	}

	t.Run("valid answer comes back compact", func(t *testing.T) {
		var buf bytes.Buffer
		if err := json.Indent(&buf, SampleContent("practice"), "", "  "); err != nil {
			t.Fatalf("indent: %v", err)
		}
		out, err := ParseExerciseContent("practice", "B1", buf.String())
		if err != nil {
			t.Fatalf("indented sample rejected: %v", err)
		}
		if bytes.Contains(out, []byte("\n")) || bytes.Contains(out, []byte("  ")) {
			t.Errorf("output not compact: %s", out)
		}
	})
}

// TestExerciseAnswerFitsTheBudget is the size probe the blocker asked for:
// representative generous text sizes per type, at C2 (the widest bounds),
// should fit comfortably inside ExerciseTimeout. Text fields are not bounded
// individually, so this is an estimate, not an upper bound on valid content.
func TestExerciseAnswerFitsTheBudget(t *testing.T) {
	longQuestion := func(id string) Question {
		return Question{
			ID:     id,
			Prompt: id + " " + strings.Repeat("p", 120),
			Options: map[string]string{
				"A": strings.Repeat("a", 40),
				"B": strings.Repeat("b", 40),
				"C": strings.Repeat("c", 40),
				"D": strings.Repeat("d", 40),
			},
			Answer:      "A",
			Explanation: strings.Repeat("e", 120),
		}
	}
	longQuestions := func(n int) []Question {
		out := make([]Question, 0, n)
		for i := 1; i <= n; i++ {
			out = append(out, longQuestion(fmt.Sprintf("q%d", i)))
		}
		return out
	}
	_, maxRunes := passageRunes("C2")

	longWords := func(n int) []Word {
		out := make([]Word, 0, n)
		for i := 1; i <= n; i++ {
			out = append(out, Word{Term: strings.Repeat("t", 20), Definition: strings.Repeat("d", 80), Example: strings.Repeat("x", 100)})
		}
		return out
	}
	longGlossary := func(n int) []Gloss {
		out := make([]Gloss, 0, n)
		for i := 1; i <= n; i++ {
			out = append(out, Gloss{Term: strings.Repeat("t", 20), MeaningVI: strings.Repeat("nghĩa ", 8), Definition: strings.Repeat("d", 100)})
		}
		return out
	}

	cases := map[string]any{
		"vocabulary": VocabularyContent{Words: longWords(MaxWords), Questions: longQuestions(MaxVocabQuestions)},
		"reading":    ReadingContent{Passage: strings.Repeat("a", maxRunes), Glossary: longGlossary(MaxGlossary), Questions: longQuestions(MaxReadingQuestions)},
		"practice":   PracticeContent{Questions: longQuestions(MaxPracticeQuestions)},
	}
	for tt, v := range cases {
		t.Run(tt, func(t *testing.T) {
			b, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			out, err := ParseExerciseContent(tt, "C2", string(b))
			if err != nil {
				t.Fatalf("largest %s content rejected: %v", tt, err)
			}
			// ≤ 8 KiB ≈ 2k tokens at ~4 bytes/token; at the measured ~42 tok/s
			// (production, 2026-09-25) that is ~49 s, inside the 60 s
			// ExerciseTimeout. This single-exercise estimate includes Vietnamese
			// glossary bytes; it does not multiply content over a whole roadmap.
			t.Logf("%s representative content: %d bytes", tt, len(out))
			if len(out) > 8192 {
				t.Errorf("%s: %d bytes, want ≤ 8192", tt, len(out))
			}
		})
	}
}

func TestExerciseUserPromptPreservesLevelGuidance(t *testing.T) {
	for _, level := range []string{"A1", "A2", "B1", "B2", "C1", "C2"} {
		if got := ExerciseUserPrompt(ExerciseBrief{TaskType: "reading", Level: level}); !strings.Contains(got, LevelGuidance(level)) {
			t.Errorf("%s exercise prompt omits current level guidance", level)
		}
	}
}

func TestExercisePromptsRequestDistinctGroundedQuestions(t *testing.T) {
	for _, want := range []string{"distinct learning point", "distinct prompt", "plausible", "distinct option texts", "meaningful sentence", "answer letters may repeat"} {
		if !strings.Contains(ExerciseSystemPrompt, want) {
			t.Errorf("system prompt omits %q", want)
		}
	}
	for taskType, grounding := range map[string]string{
		"reading":    "answers and explanations in the actual passage",
		"vocabulary": "studied words and their examples",
		"practice":   "actual module focus and learner's target goal",
	} {
		prompt := ExerciseUserPrompt(ExerciseBrief{TaskType: taskType, Level: "B1"})
		if !strings.Contains(prompt, grounding) {
			t.Errorf("%s prompt omits grounding %q", taskType, grounding)
		}
		if strings.Contains(prompt, "…") || strings.Contains(prompt, `"explanation":"<explanation>"`) {
			t.Errorf("%s prompt still uses meaningless template text", taskType)
		}
	}
}

// Exercise questions disappear on Continue, so practice cannot depend on a
// passage displayed only by an earlier question. Inspect the actual outbound
// provider messages, including a curriculum draft that suggests this trap.
func TestPracticeProviderMessagesRequireIndependentQuestionContext(t *testing.T) {
	var got struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode prepared messages: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": string(SampleContent("practice"))}}}})
	}))
	defer srv.Close()
	brief := ExerciseBrief{TaskType: "practice", Level: "B1", Goal: "Read together", ModuleFocus: "Community services", DayTitle: "Library", TaskTitle: "Community library", Draft: `{"topic":"Tell a library story in q1, then ask about it in q2"}`}
	provider := NewOpenAICompatibleProvider(srv.URL, "fixture-only", "fixture-model", srv.Client())
	if _, err := provider.GenerateContent(context.Background(), ExerciseSystemPrompt, ExerciseUserPrompt(brief)); err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 2 || got.Messages[1].Role != "user" {
		t.Fatalf("prepared messages: %+v", got.Messages)
	}
	prompt := got.Messages[1].Content
	for _, want := range []string{"independently answerable", "all context needed", "Do not put a shared story or passage only in q1", "do not refer back to earlier questions", "Shared passages belong to reading tasks", brief.Goal, brief.ModuleFocus, brief.DayTitle, brief.TaskTitle, brief.Draft} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prepared practice message missing %q", want)
		}
	}
	for _, kind := range []string{"reading", "vocabulary"} {
		other := brief
		other.TaskType = kind
		if strings.Contains(ExerciseUserPrompt(other), "independently answerable") {
			t.Errorf("practice constraint leaked into %s", kind)
		}
	}
}
