package quests

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
)

// ContentContract is the shared hydration/daily/grading boundary. Generated
// content keeps its provenance even when invalid, but has no usable revision.
// Legacy root fields and explicit legacy bodies may contain study-only content;
// unmarked nested bodies must have a complete quiz to escape outline status.
func ContentContract(e Exercise) (origin string, revision *string, body json.RawMessage, err error) {
	var meta struct {
		Content       json.RawMessage `json:"content"`
		ContentSchema int             `json:"content_schema"`
		ContentLevel  string          `json:"content_level"`
		ContentOrigin string          `json:"content_origin"`
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(e.ContentJSON, &root) != nil || root == nil {
		return "outline", nil, nil, nil
	}
	if err := json.Unmarshal(e.ContentJSON, &meta); err != nil {
		if string(root["content_schema"]) == "1" {
			return "generated", nil, root["content"], fmt.Errorf("%w: invalid metadata", ErrExerciseContentInvalid)
		}
		return "outline", nil, nil, nil
	}
	if meta.ContentSchema == 1 {
		body = meta.Content
		if err = validateSavedGenerated(e.TaskType, meta.ContentLevel, body); err != nil {
			return "generated", nil, body, err
		}
		return "generated", contentRevision(e.TaskType, body), body, nil
	}
	body = meta.Content
	if len(body) == 0 || bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
		// Recognized historical flattened shapes only. Outer bookkeeping does not
		// belong to actual learning content or its revision.
		flattened := map[string]json.RawMessage{}
		for _, key := range []string{"passage", "glossary", "words", "questions"} {
			if v, ok := root[key]; ok {
				flattened[key] = v
			}
		}
		body, _ = json.Marshal(flattened)
		if validLegacy(body, true) {
			return "legacy", contentRevision(e.TaskType, body), body, nil
		}
	} else if validLegacy(body, meta.ContentOrigin == "legacy") {
		return "legacy", contentRevision(e.TaskType, body), body, nil
	}
	return "outline", nil, body, nil
}

// Persisted levels are authoritative. Historical schema-1 rows without level
// use the union of all supported CEFR bounds (240..3040 reading runes), retaining
// generated counts/options/keys/degeneracy/glossary checks;
// no later learner profile participates. These intervals overlap.
func validateSavedGenerated(taskType, level string, body json.RawMessage) error {
	levels := []string{"A1", "A2", "B1", "B2", "C1", "C2"}
	if level != "" {
		known := false
		for _, l := range levels {
			if level == l {
				known = true
			}
		}
		if !known {
			return fmt.Errorf("%w: unknown saved content_level %q", ErrExerciseContentInvalid, level)
		}
		levels = []string{level}
	}
	// Pick one compatible reading interval before validation, so a valid
	// historical size cannot mask a precise shape/quality rejection behind a
	// later level's passage-size error. Non-reading content has no level bounds.
	validationLevel := "B1"
	if level != "" {
		validationLevel = level
	} else if taskType == "reading" {
		var c airouter.ReadingContent
		if json.Unmarshal(body, &c) == nil {
			n := utf8.RuneCountInString(strings.TrimSpace(c.Passage))
			validationLevel = "C2"
			if n < 240 {
				validationLevel = "A1"
			}
			for _, l := range levels {
				lo, hi := airouter.PassageWords(l)
				if n >= lo*4 && n <= hi*8 {
					validationLevel = l
					break
				}
			}
		}
	}
	if err := airouter.ValidateContent(taskType, validationLevel, body); err != nil {
		return fmt.Errorf("%w: %v", ErrExerciseContentInvalid, err)
	}
	return nil
}

func contentRevision(taskType string, body json.RawMessage) *string {
	// UseNumber preserves arbitrary numeric fields; encoding/json sorts object
	// keys recursively. Array order and every nested content field remain real.
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil {
		return nil
	}
	canonical, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	digest := sha256.Sum256(append(append([]byte(taskType), 0), canonical...))
	revision := hex.EncodeToString(digest[:])
	return &revision
}

func validLegacy(body json.RawMessage, studyOnly bool) bool {
	if len(body) > airouter.MaxContentBytes {
		return false
	}
	var c struct {
		Passage   string              `json:"passage"`
		Words     []airouter.Word     `json:"words"`
		Questions []airouter.Question `json:"questions"`
		Glossary  []airouter.Gloss    `json:"glossary"`
	}
	if json.Unmarshal(body, &c) != nil {
		return false
	}
	for _, w := range c.Words {
		if strings.TrimSpace(w.Term) == "" || strings.TrimSpace(w.Definition) == "" {
			return false
		}
	}
	if len(c.Questions) == 0 {
		return studyOnly && (strings.TrimSpace(c.Passage) != "" || len(c.Words) > 0)
	}
	seenIDs := map[string]bool{}
	seenPrompts := map[string]bool{}
	for _, q := range c.Questions {
		id := strings.TrimSpace(q.ID)
		prompt := normalizeLegacyText(q.Prompt)
		if id == "" || seenIDs[id] || placeholderLegacyText(q.Prompt) || seenPrompts[prompt] {
			return false
		}
		seenIDs[id] = true
		seenPrompts[prompt] = true
		if len(q.Options) < 2 || len(q.Options) > 4 {
			return false
		}
		seenOptions := map[string]bool{}
		for key, text := range q.Options {
			normalized := normalizeLegacyText(text)
			if (key != "A" && key != "B" && key != "C" && key != "D") || normalized == "" || seenOptions[normalized] {
				return false
			}
			seenOptions[normalized] = true
		}
		// Keys/explanations may both be absent. A supplied key must match an
		// existing option; a supplied explanation must be substantive and keyed.
		if q.Answer != "" {
			if _, ok := q.Options[q.Answer]; !ok {
				return false
			}
		}
		if q.Explanation != "" && (q.Answer == "" || placeholderLegacyText(q.Explanation)) {
			return false
		}
	}
	return true
}

func normalizeLegacyText(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
func placeholderLegacyText(s string) bool {
	s = normalizeLegacyText(s)
	if !strings.ContainsFunc(s, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) }) {
		return true
	}
	switch strings.TrimRight(s, ".:!?…") {
	case "todo", "tbd":
		return true
	}
	switch s {
	case "<a complete distinct question>", "<a meaningful sentence explaining the correct answer in context>":
		return true
	}
	for _, label := range []string{"prompt", "question", "question prompt", "explanation"} {
		if s == "<"+label+">" || s == "["+label+"]" || s == "{{"+label+"}}" || s == "[insert "+label+" here]" || s == "insert "+label+" here" || s == label+" goes here" {
			return true
		}
	}
	return false
}
