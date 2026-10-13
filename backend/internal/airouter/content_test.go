package airouter

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

func changeQuestions(t *testing.T, taskType string, change func([]Question)) json.RawMessage {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.Unmarshal(SampleContent(taskType), &body); err != nil {
		t.Fatal(err)
	}
	var questions []Question
	if err := json.Unmarshal(body["questions"], &questions); err != nil {
		t.Fatal(err)
	}
	change(questions)
	body["questions"], _ = json.Marshal(questions)
	out, _ := json.Marshal(body)
	return out
}

// Assert the exact rule through both entry points: a generic rejection would
// otherwise hide a broken duplicate or placeholder check.
func TestContentQuestionQualityRejections(t *testing.T) {
	cases := []struct {
		name, reason string
		change       func([]Question)
	}{
		{"same prompt", "question q2 repeats prompt of question q1", func(q []Question) { q[1].Prompt = q[0].Prompt }},
		{"normalized prompt", "question q2 repeats prompt of question q1", func(q []Question) {
			q[0].Prompt = "Which word fits the sentence?"
			q[1].Prompt = " \tWHICH  word\nfits\u00a0the sentence? "
		}},
		{"same visible choices", "question q1 option C repeats option A", func(q []Question) { q[0].Options["C"] = q[0].Options["A"] }},
		{"normalized visible choices", "question q1 option B repeats option A", func(q []Question) {
			q[0].Options["A"] = "A new library"
			q[0].Options["B"] = " \tA NEW\n library\u00a0 "
		}},
		{"copied prompt schema", "question q1 has placeholder prompt", func(q []Question) { q[0].Prompt = "<a complete distinct question>" }},
		{"copied explanation schema", "question q1 has placeholder explanation", func(q []Question) {
			q[0].Explanation = "<a meaningful sentence explaining the correct answer in context>"
		}},
	}
	for _, field := range []string{"prompt", "explanation"} {
		for _, placeholder := range []string{"TODO", " tBd. ", "…", " _?! — ", "<" + field + ">", "[insert " + field + " here]", "{{" + field + "}}"} {
			field, placeholder := field, placeholder
			cases = append(cases, struct {
				name, reason string
				change       func([]Question)
			}{field + " " + placeholder, "question q1 has placeholder " + field, func(q []Question) {
				if field == "prompt" {
					q[0].Prompt = placeholder
				} else {
					q[0].Explanation = placeholder
				}
			}})
		}
	}
	for _, taskType := range TaskTypes {
		for _, c := range cases {
			t.Run(taskType+"/"+c.name, func(t *testing.T) {
				body := changeQuestions(t, taskType, c.change)
				want := "content: " + c.reason + ": " + ErrInvalidContent.Error()
				for name, check := range map[string]func() error{
					"validator": func() error { return ValidateContent(taskType, "B1", body) },
					"parser":    func() error { _, err := ParseExerciseContent(taskType, "B1", string(body)); return err },
				} {
					err := check()
					if !errors.Is(err, ErrInvalidContent) || err.Error() != want {
						t.Errorf("%s: err = %v, want %q", name, err, want)
					}
				}
			})
		}
	}
}

func TestContentQuestionQualityPreservesLegitimateChoices(t *testing.T) {
	body := changeQuestions(t, "practice", func(q []Question) {
		q[0].Prompt = "Which punctuation mark ends a question?"
		q[0].Options = map[string]string{"A": ".", "B": "?", "C": "!", "D": ";"}
		q[1].Prompt = "Which abbreviation means to be determined (TBD)?"
		q[1].Options = map[string]string{"A": "ETA", "B": "TBD", "C": "NASA", "D": "UK"}
		q[1].Explanation = "TBD means to be determined, so the date is not yet decided."
		// The remaining distinct grammar prompts can reuse am/is/are/be and B.
	})
	if _, err := ParseExerciseContent("practice", "B1", string(body)); err != nil {
		t.Fatalf("valid punctuation, acronym, short and cross-question repeated choices rejected: %v", err)
	}
}

func TestContentAcceptsSavedGenuineB1Reading(t *testing.T) {
	// Existing genuine 9Router English-bot-free sample, saved 2026-10-12.
	// This regression preserves compatible content; it is not fresh model proof.
	body, err := os.ReadFile("testdata/genuine_b1_reading.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseExerciseContent("reading", "B1", string(body)); err != nil {
		t.Fatalf("saved genuine B1 reading rejected: %v", err)
	}
}

func TestContentQuestionQualityDoesNotUseFuzzyMatching(t *testing.T) {
	body := changeQuestions(t, "practice", func(q []Question) {
		q[0].Prompt = "Which verb fits: I ___ at the library?"
		q[1].Prompt = "Which verb fits: She ___ at the library?"
		for i := 0; i < 2; i++ {
			q[i].Options = map[string]string{"A": "work", "B": "works", "C": "working", "D": "worked"}
		}
		q[0].Answer = "A"
		q[0].Explanation = "Use the base form work with I in the present simple."
		q[1].Explanation = "Use works with the third-person singular subject she in the present simple."
	})
	if err := ValidateContent("practice", "B1", body); err != nil {
		t.Fatalf("distinct prompts and closely related grammar choices rejected: %v", err)
	}
}

func TestValidateContentAcceptsEachTypeAndTheSamples(t *testing.T) {
	for _, tt := range TaskTypes {
		t.Run(tt, func(t *testing.T) {
			if err := ValidateContent(tt, "B1", SampleContent(tt)); err != nil {
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
	type row struct {
		taskType string
		content  json.RawMessage
		want     string
	}
	cases := map[string]row{
		"absent content": {taskType: "practice", content: nil, want: "no content"},
		"empty object":   {taskType: "practice", content: json.RawMessage(`{}`), want: "no content"},
		"not an object":  {taskType: "practice", content: json.RawMessage(`[1,2]`), want: "content:"},
		"too few words": {taskType: "vocabulary", want: "words, want", content: func() json.RawMessage {
			var c VocabularyContent
			_ = json.Unmarshal(SampleContent("vocabulary"), &c)
			c.Words = c.Words[:2]
			b, _ := json.Marshal(c)
			return b
		}()},
		"blank definition": {taskType: "vocabulary", want: "needs term and definition", content: func() json.RawMessage {
			var c VocabularyContent
			_ = json.Unmarshal(SampleContent("vocabulary"), &c)
			c.Words[0].Definition = "  "
			b, _ := json.Marshal(c)
			return b
		}()},
		"short passage": {taskType: "reading", want: "passage is", content: func() json.RawMessage {
			var c ReadingContent
			_ = json.Unmarshal(SampleContent("reading"), &c)
			c.Passage = "Too short."
			b, _ := json.Marshal(c)
			return b
		}()},
		"article-length passage": {taskType: "reading", want: "passage is", content: func() json.RawMessage {
			var c ReadingContent
			_ = json.Unmarshal(SampleContent("reading"), &c)
			c.Passage = strings.Repeat("word ", 900)
			b, _ := json.Marshal(c)
			return b
		}()},
		"blank question id":     {taskType: "practice", want: "unique id", content: mutQ("practice", func(qs []Question) { qs[0].ID = "" })},
		"duplicate question id": {taskType: "practice", want: "unique id", content: mutQ("practice", func(qs []Question) { qs[2].ID = qs[1].ID })},
		"three practice questions": {taskType: "practice", want: "questions, want", content: func() json.RawMessage {
			var c PracticeContent
			_ = json.Unmarshal(SampleContent("practice"), &c)
			c.Questions = c.Questions[:3]
			b, _ := json.Marshal(c)
			return b
		}()},
		"answer not in options": {taskType: "practice", want: "answer", content: mutQ("practice", func(qs []Question) { qs[1].Answer = "E" })},
		"three options":         {taskType: "reading", want: "options, want", content: mutQ("reading", func(qs []Question) { delete(qs[0].Options, "D") })},
		"blank option text":     {taskType: "vocabulary", want: "missing option", content: mutQ("vocabulary", func(qs []Question) { qs[0].Options["C"] = "" })},
		"no explanation":        {taskType: "practice", want: "no explanation", content: mutQ("practice", func(qs []Question) { qs[3].Explanation = "" })},
		"no prompt":             {taskType: "practice", want: "no prompt", content: mutQ("practice", func(qs []Question) { qs[0].Prompt = " " })},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := ValidateContent(c.taskType, "B1", c.content)
			if err == nil {
				t.Fatal("ValidateContent accepted it")
			}
			if !errors.Is(err, ErrInvalidContent) {
				t.Errorf("err = %v, want ErrInvalidContent", err)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %q, want it to contain %q", err.Error(), c.want)
			}
			if !strings.HasPrefix(err.Error(), "content:") {
				t.Errorf("err = %q, want it to start with %q", err.Error(), "content:")
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
	if err := ValidateContent("vocabulary", "B1", b); err != nil {
		t.Fatalf("optional example / unknown field rejected: %v", err)
	}
}

func TestPassageBoundsFitEveryLevel(t *testing.T) {
	for level := range passageWords {
		t.Run(level, func(t *testing.T) {
			min, max := PassageWords(level)
			runeMin, runeMax := passageRunes(level)
			if got := min * 5; got < runeMin {
				t.Errorf("%s: min words ×5 = %d, want ≥ %d (passageRunes min)", level, got, runeMin)
			}
			if got := max * 6; got > runeMax {
				t.Errorf("%s: max words ×6 = %d, want ≤ %d (passageRunes max)", level, got, runeMax)
			}
		})
	}
}

func TestValidateContentPassagePerLevel(t *testing.T) {
	t.Run("C2 accepts a 380-word passage", func(t *testing.T) {
		var c ReadingContent
		_ = json.Unmarshal(SampleContent("reading"), &c)
		c.Passage = strings.Repeat("abcde ", 380) // 380 words × 6 runes (5-letter word + space) = 2,280 runes
		b, _ := json.Marshal(c)
		if err := ValidateContent("reading", "C2", b); err != nil {
			t.Fatalf("C2 380-word passage rejected: %v", err)
		}
	})
	t.Run("A1 rejects a 2,000-character passage", func(t *testing.T) {
		var c ReadingContent
		_ = json.Unmarshal(SampleContent("reading"), &c)
		c.Passage = strings.Repeat("a", 2000)
		b, _ := json.Marshal(c)
		err := ValidateContent("reading", "A1", b)
		if err == nil || !strings.Contains(err.Error(), "for A1") {
			t.Fatalf("err = %v, want rejection naming A1", err)
		}
	})
	t.Run("unknown level uses B1's bounds", func(t *testing.T) {
		if err := ValidateContent("reading", "Z9", SampleContent("reading")); err != nil {
			t.Fatalf("B1-sample passage rejected for unknown level: %v", err)
		}
	})
}

// All returned bytes are persisted, including unknown fields. Counts alone
// must not let one oversized text field become an unbounded cached exercise.
func TestValidateContentRejectsOversizedContent(t *testing.T) {
	for _, taskType := range TaskTypes {
		t.Run(taskType, func(t *testing.T) {
			b := SampleContent(taskType)
			b = append(b[:len(b)-1], []byte(`,"extra":"`+strings.Repeat("x", 32768)+`"}`)...)
			if err := ValidateContent(taskType, "B1", b); !errors.Is(err, ErrInvalidContent) {
				t.Fatalf("oversized content: %v, want ErrInvalidContent", err)
			}
			if _, err := ParseExerciseContent(taskType, "B1", string(b)); !errors.Is(err, ErrInvalidContent) {
				t.Fatalf("oversized response: %v, want ErrInvalidContent", err)
			}
		})
	}
}
