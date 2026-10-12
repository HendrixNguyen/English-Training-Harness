package airouter

import (
	"fmt"
	"strings"
	"testing"
)

func TestExerciseReadingPromptRequestsGlossary(t *testing.T) {
	for _, level := range []string{"A1", "A2", "B1", "B2", "C1", "C2"} {
		t.Run(level, func(t *testing.T) {
			p := ExerciseUserPrompt(ExerciseBrief{TaskType: "reading", Level: level})
			lo, hi := PassageWords(level)
			for _, want := range []string{`"glossary":[{"term":"<exact passage term>","meaning_vi":"<short Vietnamese meaning>","definition":"<one-sentence English definition>"}] × 5-8`, "harder terms", "learner's CEFR level", "short Vietnamese meaning", "one-sentence English definition", "exact spelling from the passage", fmt.Sprintf("%d-%d words", lo, hi), questionShape, LevelGuidance(level)} {
				if !strings.Contains(p, want) {
					t.Errorf("%s reading prompt missing %q", level, want)
				}
			}
		})
	}
}

func TestExerciseGlossaryOnlyRequestedForReading(t *testing.T) {
	for _, taskType := range []string{"vocabulary", "practice"} {
		if p := ExerciseUserPrompt(ExerciseBrief{TaskType: taskType, Level: "B1"}); strings.Contains(p, "glossary") || strings.Contains(p, "meaning_vi") {
			t.Errorf("%s prompt unexpectedly requests a glossary", taskType)
		}
	}
}
