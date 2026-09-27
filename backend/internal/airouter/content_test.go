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
