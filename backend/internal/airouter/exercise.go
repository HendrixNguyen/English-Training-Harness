package airouter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// ExerciseSystemPrompt is the system turn for TaskExerciseGen: one task's
// typed content, not the whole roadmap.
const ExerciseSystemPrompt = `You are an English exercise writer for a 10-minute task inside a learner's daily quest. Output ONLY one JSON object matching the requested schema: no markdown, no code fences, no commentary. Write at the learner's CEFR level. Every multiple-choice question has exactly the options A, B, C and D, exactly one correct "answer", and a one-sentence "explanation" of why that answer is right.`

// exerciseDraftRunes bounds how much of the roadmap's free-form draft rides
// into the prompt: enough to carry the topic, never enough to dominate the
// token budget the size probe (TestExerciseAnswerFitsTheBudget) counts on.
const exerciseDraftRunes = 600

// ExerciseBrief names one exercise for ExerciseUserPrompt: the learner's
// level and goal, where in the curriculum the task sits, and the roadmap's
// own free-form content for that task as a topic draft.
type ExerciseBrief struct {
	TaskType, Level, Goal, ModuleFocus, DayTitle, TaskTitle string
	// Draft is the roadmap's free-form content for this task, compact JSON.
	// It may be "" or "{}" (no draft). ExerciseUserPrompt cuts it to
	// exerciseDraftRunes runes; it is never rejected for being long.
	Draft string
}

const questionShape = `{"id":"q1","prompt":"…","options":{"A":"…","B":"…","C":"…","D":"…"},"answer":"A|B|C|D","explanation":"…"}`

// ExerciseUserPrompt names the learner (level, goal), where the task sits in
// the curriculum (module focus, day title, task title), then the one JSON
// shape for b.TaskType with its counts taken from content.go's constants
// (and, for reading, PassageWords(b.Level)).
func ExerciseUserPrompt(b ExerciseBrief) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Learner profile:\n- CEFR level: %s\n- Target goal: %s\n\n", b.Level, b.Goal)
	fmt.Fprintf(&sb, "This task sits in module focus %q, day %q, task %q.\n\n", b.ModuleFocus, b.DayTitle, b.TaskTitle)
	sb.WriteString("Produce ONE JSON object with exactly this schema:\n")
	switch b.TaskType {
	case "vocabulary":
		fmt.Fprintf(&sb, `{"words":[{"term":"…","definition":"…","example":"…"}] × %d-%d, "questions":[%s] × %d-%d}`,
			MinWords, MaxWords, questionShape, MinVocabQuestions, MaxVocabQuestions)
	case "reading":
		lo, hi := PassageWords(b.Level)
		fmt.Fprintf(&sb, `{"passage":"<%d-%d words>", "questions":[%s] × %d-%d}`,
			lo, hi, questionShape, MinReadingQuestions, MaxReadingQuestions)
		sb.WriteString("\nThe questions are about the passage.")
	case "practice":
		fmt.Fprintf(&sb, `{"questions":[%s] × %d-%d}`, questionShape, MinPracticeQuestions, MaxPracticeQuestions)
	}
	if draft := cutRunes(strings.TrimSpace(b.Draft), exerciseDraftRunes); draft != "" && draft != "{}" {
		fmt.Fprintf(&sb, "\n\nThe curriculum's draft for this task (use it as the topic; rewrite it into the schema): %s", draft)
	}
	return sb.String()
}

// cutRunes truncates s to at most n runes; it never errors, it just cuts.
func cutRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n])
}

// ParseExerciseContent decodes a model response for one exercise strictly:
// no markdown fence or preamble, no trailing tokens, a JSON object (not an
// array or scalar). It then runs ValidateContent for taskType/level and
// returns compact bytes. Every failure wraps ErrInvalidContent.
func ParseExerciseContent(taskType, level, raw string) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, invalidContent("empty response")
	}
	if strings.HasPrefix(trimmed, "```") {
		return nil, invalidContent("response is wrapped in a markdown fence")
	}
	if !strings.HasPrefix(trimmed, "{") {
		return nil, invalidContent("response does not start with a JSON object")
	}

	dec := json.NewDecoder(strings.NewReader(trimmed))
	var content json.RawMessage
	if err := dec.Decode(&content); err != nil {
		return nil, invalidContent("decoding: %v", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, invalidContent("trailing content after the JSON object")
	}

	if err := ValidateContent(taskType, level, content); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := json.Compact(&buf, content); err != nil {
		return nil, invalidContent("compacting: %v", err)
	}
	return json.RawMessage(buf.Bytes()), nil
}
