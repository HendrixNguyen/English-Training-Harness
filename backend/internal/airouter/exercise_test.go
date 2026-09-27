package airouter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
		if strings.Contains(p, "passage") {
			t.Errorf("vocabulary prompt should not mention passage:\n%s", p)
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
		if strings.Contains(p, "passage") || strings.Contains(p, "words") {
			t.Errorf("practice prompt should not mention passage or words:\n%s", p)
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
// the largest valid content per type, at C2 (the widest bounds), must still
// fit comfortably inside ExerciseTimeout.
func TestExerciseAnswerFitsTheBudget(t *testing.T) {
	longQuestion := func(id string) Question {
		return Question{
			ID:     id,
			Prompt: strings.Repeat("p", 120),
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

	cases := map[string]any{
		"vocabulary": VocabularyContent{Words: longWords(MaxWords), Questions: longQuestions(MaxVocabQuestions)},
		"reading":    ReadingContent{Passage: strings.Repeat("a", maxRunes), Questions: longQuestions(MaxReadingQuestions)},
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
			// ≤ 6 KB ≈ 1.5k tokens at ~4 bytes/token; at the measured ~42 tok/s
			// (production, 2026-09-25) that is ≤ ~36 s, well inside the 60 s
			// ExerciseTimeout.
			if len(out) > 6144 {
				t.Errorf("%s: %d bytes, want ≤ 6144", tt, len(out))
			}
		})
	}
}
