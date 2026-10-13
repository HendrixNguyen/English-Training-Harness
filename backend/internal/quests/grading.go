package quests

import (
	"encoding/json"
	"errors"
	"math"
	"strings"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
)

// ErrInvalidAnswers maps malformed selections to HTTP 400 before tracking writes.
var ErrInvalidAnswers = errors.New("quests: invalid answers")

// ErrExerciseContentChanged means the client or completion snapshot is stale.
var ErrExerciseContentChanged = errors.New("quests: exercise content changed")

// ProgressSubmission contains selections only. Keys and scores come from storage.
type ProgressSubmission struct {
	Revision *string
	Answers  map[string]string
}

type QuizAnswer struct {
	QuestionID     string  `json:"question_id"`
	SelectedOption string  `json:"selected_option"`
	CorrectOption  *string `json:"correct_option"`
	IsCorrect      *bool   `json:"is_correct"`
	Explanation    *string `json:"explanation"`
}

type QuizResult struct {
	Status          string       `json:"status"`
	AnsweredCount   int          `json:"answered_count"`
	QuestionCount   int          `json:"question_count"`
	CorrectCount    *int         `json:"correct_count"`
	ScorePercent    *int         `json:"score_percent"`
	DurationSeconds int64        `json:"duration_seconds"`
	Answers         []QuizAnswer `json:"answers"`
}

// gradeSubmission is pure: it validates every selection and produces an ordered
// result. Revisionless compatibility never discloses grading fields.
func gradeSubmission(e Exercise, seconds int64, in ProgressSubmission) (json.RawMessage, error) {
	_, revision, body, contractErr := ContentContract(e)
	if in.Revision != nil {
		if contractErr != nil {
			return nil, contractErr
		}
		if revision == nil {
			return nil, ErrExerciseContentInvalid
		}
		if *in.Revision != *revision {
			return nil, ErrExerciseContentChanged
		}
	}
	var content struct {
		Questions []airouter.Question `json:"questions"`
	}
	// Old time-only outline calls must remain usable. Supplied answers still need
	// an unambiguous usable question set to validate against.

	if contractErr != nil || revision == nil {
		// Compatibility validates recognizable IDs/options only. Unrelated invalid
		// generated study fields cannot break an older client's valid selections.
		questions, err := compatibilityQuestions(body)
		if err != nil && len(in.Answers) > 0 {
			return nil, ErrInvalidAnswers
		}
		content.Questions = questions
	} else if err := json.Unmarshal(body, &content); err != nil {
		return nil, ErrInvalidAnswers
	}

	byID := make(map[string]airouter.Question, len(content.Questions))
	allKeyed := len(content.Questions) > 0
	for _, q := range content.Questions {
		byID[q.ID] = q
		if q.Answer == "" {
			allKeyed = false
		}
	}
	for id, choice := range in.Answers {
		q, ok := byID[id]
		if !ok {
			return nil, ErrInvalidAnswers
		}
		if _, ok = q.Options[choice]; !ok {
			return nil, ErrInvalidAnswers
		}
	}
	if in.Revision != nil && len(in.Answers) != len(content.Questions) {
		return nil, ErrInvalidAnswers
	}
	out := QuizResult{Status: "ungraded", AnsweredCount: len(in.Answers), QuestionCount: len(content.Questions), DurationSeconds: min(seconds, MaxDurationSeconds), Answers: make([]QuizAnswer, 0, len(in.Answers))}
	graded := in.Revision != nil && allKeyed
	correct := 0
	for _, q := range content.Questions {
		choice, ok := in.Answers[q.ID]
		if !ok {
			continue
		}
		a := QuizAnswer{QuestionID: q.ID, SelectedOption: choice}
		// A matching legacy revision may have optional per-question keys; only
		// unknown fields stay null. Aggregate grade requires every key.
		if in.Revision != nil && q.Answer != "" {
			key := q.Answer
			isCorrect := choice == key
			a.CorrectOption = &key
			a.IsCorrect = &isCorrect
			if q.Explanation != "" {
				explanation := q.Explanation
				a.Explanation = &explanation
			}
			if isCorrect {
				correct++
			}
		}
		out.Answers = append(out.Answers, a)
	}
	if graded {
		score := int(math.Round(100 * float64(correct) / float64(len(content.Questions))))
		out.Status = "graded"
		out.CorrectCount = &correct
		out.ScorePercent = &score
	}
	return json.Marshal(out)
}

// compatibilityQuestions deliberately ignores study fields, prompts, keys and
// explanations: revisionless calls validate choices, never grade untrusted data.
func compatibilityQuestions(body json.RawMessage) ([]airouter.Question, error) {
	var envelope map[string]json.RawMessage
	if json.Unmarshal(body, &envelope) != nil || envelope == nil {
		return nil, ErrInvalidAnswers
	}
	raw, ok := envelope["questions"]
	if !ok {
		return nil, nil
	}
	var questions []struct {
		ID      string            `json:"id"`
		Options map[string]string `json:"options"`
	}
	if string(raw) == "null" || json.Unmarshal(raw, &questions) != nil {
		return nil, ErrInvalidAnswers
	}
	out := make([]airouter.Question, 0, len(questions))
	seen := map[string]bool{}
	for _, q := range questions {
		id := strings.TrimSpace(q.ID)
		if id == "" || seen[id] || len(q.Options) < 2 || len(q.Options) > 4 {
			return nil, ErrInvalidAnswers
		}
		seen[id] = true
		visible := map[string]bool{}
		for key, value := range q.Options {
			text := airouter.NormalizeQuestionText(value)
			if (key != "A" && key != "B" && key != "C" && key != "D") || text == "" || visible[text] {
				return nil, ErrInvalidAnswers
			}
			visible[text] = true
		}
		out = append(out, airouter.Question{ID: q.ID, Options: q.Options})
	}
	return out, nil
}
