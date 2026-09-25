package airouter

import (
	"encoding/json"
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
