package onboarding

import (
	"errors"
	"strings"
	"testing"
)

func TestPlacementUserPromptCarriesItemsLevelsCorrectAndGivenAnswers(t *testing.T) {
	p := PlacementUserPrompt([]Answer{{QuestionID: "q1", SelectedOption: "B"}, {QuestionID: "q9", SelectedOption: "A"}})
	for _, want := range []string{`"id":"q1"`, `"level":"A1"`, `"correct":"B"`, `"answer":"B"`, `"id":"q9"`, `"level":"C1"`, `"answer":"A"`, `"is_correct":true`, `"is_correct":false`} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt missing %s:\n%s", want, p)
		}
	}
	if strings.Contains(p, "q2") {
		t.Error("prompt includes an unanswered item; only answered items are graded")
	}
}

func TestPlacementSystemPromptDemandsJSONOnly(t *testing.T) {
	for _, want := range []string{"CEFR", `"cefr_level"`, "ONLY"} {
		if !strings.Contains(PlacementSystemPrompt, want) {
			t.Errorf("PlacementSystemPrompt missing %q", want)
		}
	}
}

func TestParsePlacement(t *testing.T) {
	good := map[string]struct{ raw, want string }{
		"plain":       {`{"cefr_level":"B1"}`, "B1"},
		"extra field": {`{"cefr_level":"C2","reasoning":"…"}`, "C2"},
		"whitespace":  {"\n {\"cefr_level\": \"A2\"}\n", "A2"},
	}
	for name, tc := range good {
		t.Run(name, func(t *testing.T) {
			lvl, err := ParsePlacement(tc.raw)
			if err != nil || lvl != tc.want {
				t.Fatalf("ParsePlacement = %q, %v; want %q", lvl, err, tc.want)
			}
		})
	}
	bad := map[string]string{
		"fence":    "```json\n{\"cefr_level\":\"B1\"}\n```",
		"preamble": `The learner is B1: {"cefr_level":"B1"}`,
		"bad enum": `{"cefr_level":"B7"}`,
		"lower":    `{"cefr_level":"b1"}`,
		"missing":  `{"level":"B1"}`,
		"trailing": `{"cefr_level":"B1"} {}`,
		"empty":    ``,
	}
	for name, raw := range bad {
		t.Run(name, func(t *testing.T) {
			if _, err := ParsePlacement(raw); !errors.Is(err, ErrBadAIOutput) {
				t.Errorf("err = %v, want ErrBadAIOutput", err)
			}
		})
	}
}
