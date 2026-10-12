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
const ExerciseSystemPrompt = `You are an English exercise writer for a 10-minute task inside a learner's daily quest. Output ONLY one JSON object matching the requested schema: no markdown, no code fences, no commentary. Write at the learner's CEFR level. Each question must test a distinct learning point with a distinct prompt. Every multiple-choice question has exactly the options A, B, C and D with distinct option texts, exactly one correct "answer", and plausible distractors that are wrong for the given context. Write each "explanation" as a meaningful sentence explaining why the answer fits that context and teaching the relevant point. Correct answer letters may repeat across questions. Replace every schema description with actual learning content; never emit TODO, TBD, ellipses or template placeholders as prompts or explanations.`

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

const questionShape = `{"id":"q1","prompt":"<a complete distinct question>","options":{"A":"<choice A>","B":"<choice B>","C":"<choice C>","D":"<choice D>"},"answer":"<one option key: A, B, C or D>","explanation":"<a meaningful sentence explaining the correct answer in context>"}`

// ExerciseUserPrompt names the learner (level, goal), where the task sits in
// the curriculum (module focus, day title, task title), then the one JSON
// shape for b.TaskType with its counts taken from content.go's constants
// (and, for reading, PassageWords(b.Level)).
func ExerciseUserPrompt(b ExerciseBrief) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Learner profile:\n- CEFR level: %s\n- Target goal: %s\n\n", b.Level, b.Goal)
	fmt.Fprintf(&sb, "This task sits in module focus %q, day %q, task %q.\n\n", b.ModuleFocus, b.DayTitle, b.TaskTitle)
	fmt.Fprintf(&sb, "Level guidance:\n%s\n\n", LevelGuidance(b.Level))
	sb.WriteString("Produce ONE JSON object with exactly this schema:\n")
	switch b.TaskType {
	case "vocabulary":
		fmt.Fprintf(&sb, `{"words":[{"term":"<a word to study>","definition":"<the word's meaning>","example":"<a sentence using the word in context>"}] × %d-%d, "questions":[%s] × %d-%d}`,
			MinWords, MaxWords, questionShape, MinVocabQuestions, MaxVocabQuestions)
		sb.WriteString("\nGround the questions, answers and explanations in the studied words and their examples. Test different word meanings or uses, with plausible distractors.")
	case "reading":
		lo, hi := PassageWords(b.Level)
		fmt.Fprintf(&sb, `{"passage":"<%d-%d words>", "glossary":[{"term":"<exact passage term>","meaning_vi":"<short Vietnamese meaning>","definition":"<one-sentence English definition>"}] × %d-%d, "questions":[%s] × %d-%d}`,
			lo, hi, MinGlossary, MaxGlossary, questionShape, MinReadingQuestions, MaxReadingQuestions)
		sb.WriteString("\nGround the questions, answers and explanations in the actual passage. Cover distinct details, meanings or inferences supported by the text, with plausible distractors that the passage does not support.")
		sb.WriteString("\nFor the glossary, choose distinct harder terms appropriate to the learner's CEFR level. Each term must use the exact spelling from the passage, with a short Vietnamese meaning and a one-sentence English definition.")
	case "practice":
		fmt.Fprintf(&sb, `{"questions":[%s] × %d-%d}`, questionShape, MinPracticeQuestions, MaxPracticeQuestions)
		sb.WriteString("\nUse the actual module focus and learner's target goal to write varied practice situations. Ground each answer and explanation in its situation and relevant language rule.")
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
	if len(raw) > MaxContentBytes {
		return nil, invalidContent("body exceeds %d bytes", MaxContentBytes)
	}
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
