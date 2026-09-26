package onboarding

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrBadAIOutput means the model answered with something that is not the
// requested JSON, twice. Callers map it to 502 and must have written nothing.
var ErrBadAIOutput = errors.New("onboarding: AI output did not match the requested shape")

// Answer is one entry of the §6.1 request's answers array.
type Answer struct {
	QuestionID     string `json:"question_id" binding:"required"`
	SelectedOption string `json:"selected_option" binding:"required"`
}

// PlacementSystemPrompt is the grader's system turn (TaskPlacementTest). §5.1
// only says "Grade"; this is the pet slice-style decision recorded in CODEMAP.
const PlacementSystemPrompt = `You are a CEFR examiner. You receive multiple-choice placement items, each tagged with the CEFR level it probes and its correct option, together with the learner's chosen options. Estimate the learner's overall CEFR level from the pattern of correct and incorrect answers: a learner who is reliably correct up to a level and mostly wrong above it is at that level.

Output ONLY a JSON object of the form {"cefr_level": "<A1|A2|B1|B2|C1|C2>"}. No markdown, no commentary.`

// PlacementUserPrompt serialises the answered items with their level, correct
// option and the learner's choice. Unknown ids are skipped (validation has
// already rejected them).
func PlacementUserPrompt(answers []Answer) string {
	type item struct {
		ID        string `json:"id"`
		Level     string `json:"level"`
		Prompt    string `json:"prompt"`
		Correct   string `json:"correct"`
		Answer    string `json:"answer"`
		IsCorrect bool   `json:"is_correct"`
	}
	items := make([]item, 0, len(answers))
	for _, a := range answers {
		q, ok := Lookup(a.QuestionID)
		if !ok {
			continue
		}
		items = append(items, item{ID: q.ID, Level: q.Level, Prompt: q.Prompt, Correct: q.Correct, Answer: a.SelectedOption, IsCorrect: q.Correct == a.SelectedOption})
	}
	b, _ := json.Marshal(map[string]any{"items": items})
	return "Placement items and the learner's answers:\n" + string(b)
}

var cefrLevels = map[string]bool{"A1": true, "A2": true, "B1": true, "B2": true, "C1": true, "C2": true}

// cefrOrder is the fixed ladder GradeFloor and maxLevel walk. C2 is reachable
// only through the AI grader — the placement Bank tops out at C1, so the
// floor never reaches it (see GradeFloor).
var cefrOrder = []string{"A1", "A2", "B1", "B2", "C1", "C2"}

// GradeFloor derives a deterministic floor from the bank's own correct
// answers, independent of the AI grader: a learner who answers both items of
// a level correctly has proven at least that level. It walks the ladder from
// A1, requiring both items of a level correct to advance past it, and stops
// at the first level that fails — so a wrong A1 item caps the floor at A1
// even if every later item is right. It never returns below A1 and, because
// the bank has no C2 items, never above C1; unknown question ids are
// skipped (validate has already rejected them upstream; this stays
// defensive).
func GradeFloor(answers []Answer) string {
	correct := map[string]int{}
	total := map[string]int{}
	for _, q := range Bank {
		total[q.Level]++
	}
	for _, a := range answers {
		q, ok := Lookup(a.QuestionID)
		if !ok {
			continue
		}
		if q.Correct == a.SelectedOption {
			correct[q.Level]++
		}
	}
	floor := "A1"
	for _, level := range cefrOrder {
		if total[level] == 0 || correct[level] != total[level] {
			break
		}
		floor = level
	}
	return floor
}

// maxLevel returns whichever of a, b sits higher in cefrOrder. An unknown
// level loses to a known one so a stray value never wins by accident.
func maxLevel(a, b string) string {
	ai, aok := cefrIndex(a)
	bi, bok := cefrIndex(b)
	if !aok {
		return b
	}
	if !bok {
		return a
	}
	if ai >= bi {
		return a
	}
	return b
}

func cefrIndex(level string) (int, bool) {
	for i, l := range cefrOrder {
		if l == level {
			return i, true
		}
	}
	return 0, false
}

// ParsePlacement decodes {"cefr_level": "..."} strictly: no fence, no
// preamble, no trailing tokens, level in the §3.2 enum (case-sensitive).
func ParsePlacement(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "{") {
		return "", fmt.Errorf("%w: not a JSON object", ErrBadAIOutput)
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	var out struct {
		Level string `json:"cefr_level"`
	}
	if err := dec.Decode(&out); err != nil {
		return "", fmt.Errorf("%w: %v", ErrBadAIOutput, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return "", fmt.Errorf("%w: trailing content", ErrBadAIOutput)
	}
	if !cefrLevels[out.Level] {
		return "", fmt.Errorf("%w: cefr_level %q", ErrBadAIOutput, out.Level)
	}
	return out.Level, nil
}
