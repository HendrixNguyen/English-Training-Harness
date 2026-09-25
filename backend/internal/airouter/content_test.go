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
		"absent content":           {Type: "practice"},
		"empty object":             {Type: "practice", Content: json.RawMessage(`{}`)},
		"not an object":            {Type: "practice", Content: json.RawMessage(`[1,2]`)},
		"too few words":            {Type: "vocabulary", Content: func() json.RawMessage { var c VocabularyContent; _ = json.Unmarshal(SampleContent("vocabulary"), &c); c.Words = c.Words[:2]; b, _ := json.Marshal(c); return b }()},
		"blank definition":         {Type: "vocabulary", Content: func() json.RawMessage { var c VocabularyContent; _ = json.Unmarshal(SampleContent("vocabulary"), &c); c.Words[0].Definition = "  "; b, _ := json.Marshal(c); return b }()},
		"short passage":            {Type: "reading", Content: func() json.RawMessage { var c ReadingContent; _ = json.Unmarshal(SampleContent("reading"), &c); c.Passage = "Too short."; b, _ := json.Marshal(c); return b }()},
		"article-length passage":   {Type: "reading", Content: func() json.RawMessage { var c ReadingContent; _ = json.Unmarshal(SampleContent("reading"), &c); c.Passage = strings.Repeat("word ", 900); b, _ := json.Marshal(c); return b }()},
		"blank question id":        {Type: "practice", Content: mutQ("practice", func(qs []Question) { qs[0].ID = "" })},
		"three practice questions": {Type: "practice", Content: func() json.RawMessage { var c PracticeContent; _ = json.Unmarshal(SampleContent("practice"), &c); c.Questions = c.Questions[:3]; b, _ := json.Marshal(c); return b }()},
		"answer not in options":    {Type: "practice", Content: mutQ("practice", func(qs []Question) { qs[1].Answer = "E" })},
		"lower-case answer":        {Type: "practice", Content: mutQ("practice", func(qs []Question) { qs[1].Answer = "b" })},
		"three options":            {Type: "reading", Content: mutQ("reading", func(qs []Question) { delete(qs[0].Options, "D") })},
		"option key outside A..D":  {Type: "reading", Content: mutQ("reading", func(qs []Question) { qs[0].Options["E"] = qs[0].Options["D"]; delete(qs[0].Options, "D") })},
		"blank option text":        {Type: "vocabulary", Content: mutQ("vocabulary", func(qs []Question) { qs[0].Options["C"] = "" })},
		"duplicate question id":    {Type: "practice", Content: mutQ("practice", func(qs []Question) { qs[2].ID = qs[1].ID })},
		"no explanation":           {Type: "practice", Content: mutQ("practice", func(qs []Question) { qs[3].Explanation = "" })},
		"no prompt":                {Type: "practice", Content: mutQ("practice", func(qs []Question) { qs[0].Prompt = " " })},
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
