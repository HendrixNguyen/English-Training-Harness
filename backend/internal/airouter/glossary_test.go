package airouter

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func readingWithGlossary(t *testing.T, glossary any, absent bool) json.RawMessage {
	t.Helper()
	var c map[string]any
	if err := json.Unmarshal(SampleContent("reading"), &c); err != nil {
		t.Fatal(err)
	}
	delete(c, "glossary")
	if !absent {
		c["glossary"] = glossary
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestReadingGlossaryAcceptsLenientContent(t *testing.T) {
	entry := map[string]any{"term": "learner", "meaning_vi": "người học", "definition": "A person who is learning."}
	excess := make([]any, 12)
	for i := range excess {
		excess[i] = entry
	}
	for name, glossary := range map[string]any{
		"absent":                   nil,
		"null":                     nil,
		"empty":                    []any{},
		"one entry":                []any{entry},
		"blank strings":            []any{map[string]any{"term": " ", "meaning_vi": "", "definition": " "}},
		"missing fields":           []any{map[string]any{}},
		"null fields":              []any{map[string]any{"term": nil, "meaning_vi": nil, "definition": nil}},
		"null entry":               []any{nil},
		"duplicate terms":          []any{entry, entry},
		"term absent from passage": []any{map[string]any{"term": "absent", "meaning_vi": "vắng mặt", "definition": "Not present."}},
		"excess entries":           excess,
	} {
		t.Run(name, func(t *testing.T) {
			b := readingWithGlossary(t, glossary, name == "absent")
			if err := ValidateContent("reading", "B1", b); err != nil {
				t.Fatalf("lenient content rejected: %v", err)
			}
			out, err := ParseExerciseContent("reading", "B1", string(b))
			if err != nil {
				t.Fatalf("lenient response rejected: %v", err)
			}
			if !bytes.Equal(out, b) {
				t.Fatalf("content changed during validation:\ngot %s\nwant %s", out, b)
			}
		})
	}
}

func TestReadingGlossaryRejectsWrongTypes(t *testing.T) {
	for name, tc := range map[string]struct {
		glossary any
		want     string
	}{
		"object":            {map[string]any{"term": "learner"}, "cannot unmarshal object"},
		"string":            {"learner", "cannot unmarshal string"},
		"number":            {7, "cannot unmarshal number"},
		"boolean":           {true, "cannot unmarshal bool"},
		"string entry":      {[]any{"learner"}, "cannot unmarshal string"},
		"array entry":       {[]any{[]any{}}, "cannot unmarshal array"},
		"numeric term":      {[]any{map[string]any{"term": 7}}, "cannot unmarshal number"},
		"boolean meaning":   {[]any{map[string]any{"meaning_vi": false}}, "cannot unmarshal bool"},
		"object definition": {[]any{map[string]any{"definition": map[string]any{}}}, "cannot unmarshal object"},
		"array meaning":     {[]any{map[string]any{"meaning_vi": []any{}}}, "cannot unmarshal array"},
	} {
		t.Run(name, func(t *testing.T) {
			b := readingWithGlossary(t, tc.glossary, false)
			for _, err := range []error{ValidateContent("reading", "B1", b), func() error {
				_, err := ParseExerciseContent("reading", "B1", string(b))
				return err
			}()} {
				if !errors.Is(err, ErrInvalidContent) {
					t.Fatalf("err = %v, want ErrInvalidContent", err)
				}
				if !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "glossary") {
					t.Errorf("err = %q, want glossary rejection containing %q", err, tc.want)
				}
			}
		})
	}
}

func TestReadingSampleIncludesPassageGlossary(t *testing.T) {
	var c ReadingContent
	if err := json.Unmarshal(SampleContent("reading"), &c); err != nil {
		t.Fatal(err)
	}
	if len(c.Glossary) != 5 {
		t.Fatalf("sample has %d glossary entries, want 5", len(c.Glossary))
	}
	seen := map[string]bool{}
	for _, g := range c.Glossary {
		if g.Term == "" || g.MeaningVI == "" || g.Definition == "" || !strings.Contains(c.Passage, g.Term) || seen[g.Term] {
			t.Errorf("invalid sample glossary entry: %+v", g)
		}
		seen[g.Term] = true
	}
}
