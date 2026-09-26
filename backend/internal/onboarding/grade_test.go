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

func answersFor(correctIDs ...string) []Answer {
	want := map[string]bool{}
	for _, id := range correctIDs {
		want[id] = true
	}
	var out []Answer
	for _, q := range Bank {
		opt := "Z" // guaranteed wrong: options are A-D
		if want[q.ID] {
			opt = q.Correct
		}
		out = append(out, Answer{QuestionID: q.ID, SelectedOption: opt})
	}
	return out
}

func TestGradeFloor(t *testing.T) {
	cases := []struct {
		name    string
		answers []Answer
		want    string
	}{
		{"all correct", answersFor("q1", "q2", "q3", "q4", "q5", "q6", "q7", "q8", "q9", "q10"), "C1"},
		{"A1+A2 right, one B1 right, rest wrong", answersFor("q1", "q2", "q3", "q4", "q5"), "A2"},
		{"through B1, one B2 right", answersFor("q1", "q2", "q3", "q4", "q5", "q6", "q7"), "B1"},
		{"only one A1 item right", answersFor("q1"), "A1"},
		{"all wrong", answersFor(), "A1"},
		{"unknown ids only", []Answer{{QuestionID: "q99", SelectedOption: "A"}, {QuestionID: "q98", SelectedOption: "B"}}, "A1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := GradeFloor(tc.answers); got != tc.want {
				t.Errorf("GradeFloor(%s) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestMaxLevel(t *testing.T) {
	cases := []struct{ a, b, want string }{
		{"A2", "B1", "B1"},
		{"C1", "A1", "C1"},
		{"xx", "B1", "B1"},
		{"B1", "xx", "B1"},
	}
	for _, tc := range cases {
		if got := maxLevel(tc.a, tc.b); got != tc.want {
			t.Errorf("maxLevel(%q, %q) = %q, want %q", tc.a, tc.b, got, tc.want)
		}
	}
}
