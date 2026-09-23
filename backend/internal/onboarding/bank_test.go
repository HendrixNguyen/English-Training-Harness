package onboarding

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBankIsWellFormed(t *testing.T) {
	if len(Bank) < 10 {
		t.Fatalf("len(Bank) = %d, want at least 10 items", len(Bank))
	}
	ids := map[string]bool{}
	levels := map[string]bool{"A1": true, "A2": true, "B1": true, "B2": true, "C1": true, "C2": true}
	for _, q := range Bank {
		if ids[q.ID] {
			t.Errorf("duplicate id %q", q.ID)
		}
		ids[q.ID] = true
		if !levels[q.Level] {
			t.Errorf("%s: level %q is not a CEFR level", q.ID, q.Level)
		}
		if _, ok := q.Options[q.Correct]; !ok {
			t.Errorf("%s: correct option %q is not among %v", q.ID, q.Correct, q.Options)
		}
		if len(q.Options) != 4 || strings.TrimSpace(q.Prompt) == "" {
			t.Errorf("%s: want 4 options and a prompt, got %d / %q", q.ID, len(q.Options), q.Prompt)
		}
	}
}

func TestPublicBankNeverLeaksAnswersOrLevels(t *testing.T) {
	b, err := json.Marshal(PublicBank())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	for _, leak := range []string{`"correct"`, `"level"`, `"Correct"`, `"Level"`} {
		if strings.Contains(s, leak) {
			t.Errorf("public quiz leaks %s: %s", leak, s)
		}
	}
	var out struct {
		Questions []struct {
			ID      string            `json:"id"`
			Prompt  string            `json:"prompt"`
			Options map[string]string `json:"options"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Questions) != len(Bank) || out.Questions[0].ID != Bank[0].ID || len(out.Questions[0].Options) != 4 {
		t.Errorf("public shape = %+v", out.Questions[0])
	}
}

func TestLookupFindsQuestionsByID(t *testing.T) {
	q, ok := Lookup(Bank[3].ID)
	if !ok || q.ID != Bank[3].ID {
		t.Errorf("Lookup(%q) = %+v, %t", Bank[3].ID, q, ok)
	}
	if _, ok := Lookup("nope"); ok {
		t.Error("Lookup(nope) found something")
	}
}
