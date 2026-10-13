package quests

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"strings"
	"testing"
	"time"
)

func gradingExercise() Exercise {
	return Exercise{ID: "ex-2-reading", TaskType: "reading", ContentJSON: json.RawMessage(`{"content":{"questions":[{"id":"q1","prompt":"What is the first item?","options":{"A":"First","B":"Second"},"answer":"A","explanation":"The first item appears before the second."},{"id":"q2","prompt":"What comes after the first?","options":{"A":"First","B":"Second"},"answer":"B","explanation":"The second item follows the first."},{"id":"q3","prompt":"Which item begins the sequence?","options":{"A":"First","B":"Second"},"answer":"A","explanation":"The sequence begins with the first item."}]}}`)}
}
func gradingHarness(t *testing.T) (*harness, string) {
	h := newHarness(t, time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC))
	e := gradingExercise()
	h.quests.exercises[2] = []Exercise{e}
	_, r, _, err := ContentContract(e)
	if err != nil || r == nil {
		t.Fatal("bad fixture", err)
	}
	return h, *r
}
func TestProgressStrictAnswersAndRevision(t *testing.T) {
	for _, tc := range []struct {
		name, extra string
		status      int
	}{
		{"graded", `,"user_answers":{"q1":"A","q2":"A","q3":"A"}`, 200},
		{"partial", `,"user_answers":{"q1":"A"}`, 400},
		{"extra", `,"user_answers":{"q1":"A","q2":"A","q3":"A","q4":"B"}`, 400},
		{"option", `,"user_answers":{"q1":"Z","q2":"A","q3":"A"}`, 400},
		{"duplicate_id", `,"user_answers":{"q1":"A","q1":"B","q2":"A","q3":"A"}`, 400},
		{"number_choice", `,"user_answers":{"q1":1}`, 400}, {"null_choice", `,"user_answers":{"q1":null}`, 400}, {"null_answers", `,"user_answers":null`, 400}, {"array_answers", `,"user_answers":[]`, 400},
		{"score", `,"score_percent":100`, 400}, {"duplicate_field", `,"duration_seconds":9`, 400}, {"unknown", `,"correct_option":"A"`, 400},
		{"changed", `,"user_answers":{"q1":"A","q2":"B","q3":"A"}`, 409},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, rev := gradingHarness(t)
			if tc.name == "changed" {
				rev = "stale"
			}
			body := fmt.Sprintf(`{"exercise_id":"ex-2-reading","duration_seconds":47,"content_revision":%q%s}`, rev, tc.extra)
			w := postJSON(newQuestRouter(h.svc, "u1"), "/api/v1/quests/progress", body)
			if w.Code != tc.status {
				t.Fatalf("status %d body %s", w.Code, w.Body)
			}
			if tc.status != 200 {
				if len(h.log.calls) != 0 {
					t.Fatal("rejected request wrote", h.log.calls)
				}
				return
			}
			var out struct {
				QuizResult QuizResult `json:"quiz_result"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
				t.Fatal(err)
			}
			q := out.QuizResult
			if q.Status != "graded" || q.CorrectCount == nil || *q.CorrectCount != 2 || q.ScorePercent == nil || *q.ScorePercent != 67 || q.DurationSeconds != 47 || len(q.Answers) != 3 || q.Answers[1].SelectedOption != "A" || q.Answers[1].IsCorrect == nil || *q.Answers[1].IsCorrect {
				t.Fatalf("result %+v", q)
			}
		})
	}
}
func TestProgressRejectsTrailingOversizedAndWrongRevisionType(t *testing.T) {
	for _, body := range []string{`{} {}`,
		`{"exercise_id":null,"duration_seconds":1}`,
		`{"exercise_id":12,"duration_seconds":1}`,
		`{"exercise_id":"ex-2-reading","duration_seconds":1.5}`,
		`{"exercise_id":"ex-2-reading","duration_seconds":"1"}`,
		`{"exercise_id":"ex-2-reading","duration_seconds":1}` + strings.Repeat(" ", 70000), `{"exercise_id":"ex-2-reading","duration_seconds":1} {}`, `{"exercise_id":"ex-2-reading","duration_seconds":1,"content_revision":null}`, `{"exercise_id":"ex-2-reading","duration_seconds":1,"content_revision":4}`, `{"exercise_id":"ex-2-reading","duration_seconds":1,"padding":"` + strings.Repeat("x", 70000) + `"}`} {
		h, _ := gradingHarness(t)
		w := postJSON(newQuestRouter(h.svc, "u1"), "/api/v1/quests/progress", body)
		if w.Code != 400 || len(h.log.calls) != 0 {
			t.Fatal(w.Code, h.log.calls)
		}
	}
}
func TestRevisionlessAnswersStayUngradedAndRetainCompletedResult(t *testing.T) {
	h, rev := gradingHarness(t)
	ctx := context.Background()
	out, err := h.svc.RecordProgress(ctx, "u1", "ex-2-reading", 23, ProgressSubmission{Answers: map[string]string{"q2": "B"}})
	if err != nil {
		t.Fatal(err)
	}
	var q QuizResult
	if err := json.Unmarshal(out.QuizResult, &q); err != nil {
		t.Fatal(err)
	}
	if q.Status != "ungraded" || q.AnsweredCount != 1 || q.QuestionCount != 3 || q.CorrectCount != nil || q.ScorePercent != nil || q.DurationSeconds != 23 || len(q.Answers) != 1 || q.Answers[0].CorrectOption != nil || q.Answers[0].IsCorrect != nil || q.Answers[0].Explanation != nil {
		t.Fatalf("result %+v", q)
	}
	again, err := h.svc.RecordProgress(ctx, "u1", "ex-2-reading", 7, ProgressSubmission{Revision: &rev, Answers: map[string]string{"q1": "A", "q2": "B", "q3": "A"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(again.QuizResult) != string(out.QuizResult) {
		t.Fatalf("overwrote completed result %s", again.QuizResult)
	}
}
func TestRevisedInvalidContentFailsBeforeWrites(t *testing.T) {
	h, rev := gradingHarness(t)
	h.quests.exercises[2][0].ContentJSON = json.RawMessage(`{"content_schema":1,"content":{"questions":[]}}`)
	_, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 9, ProgressSubmission{Revision: &rev})
	if !strings.Contains(fmt.Sprint(err), ErrExerciseContentInvalid.Error()) {
		t.Fatal(err)
	}
	if len(h.log.calls) != 0 {
		t.Fatal(h.log.calls)
	}
}

func TestGradingCompatibilityAndEmptyHistory(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		answers map[string]string
		revised bool
		count   int
	}{
		{"time_only", string(gradingExercise().ContentJSON), nil, false, 3},
		{"keyless", `{"content":{"questions":[{"id":"q1","prompt":"Choose a beginning","options":{"A":"First","B":"Last"}}]}}`, map[string]string{"q1": "A"}, true, 1},
		{"study_only", `{"passage":"A historical study passage without a quiz."}`, nil, true, 0},
		{"outline_compatibility", `{"content":{"passage":"A short outline"}}`, nil, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := Exercise{ID: "x", TaskType: "reading", ContentJSON: json.RawMessage(tc.body)}
			in := ProgressSubmission{Answers: tc.answers}
			if tc.revised {
				_, in.Revision, _, _ = ContentContract(e)
			}
			raw, err := gradeSubmission(e, 31, in)
			if err != nil {
				t.Fatal(err)
			}
			var q QuizResult
			_ = json.Unmarshal(raw, &q)
			if q.Status != "ungraded" || q.CorrectCount != nil || q.ScorePercent != nil || q.QuestionCount != tc.count || q.DurationSeconds != 31 || q.Answers == nil {
				t.Fatalf("%s", raw)
			}
			for _, a := range q.Answers {
				if a.CorrectOption != nil || a.IsCorrect != nil || a.Explanation != nil {
					t.Fatalf("unknown fields %+v", a)
				}
			}
			if toTask(e).QuizResult != nil {
				t.Fatal("fabricated history")
			}
		})
	}
}

func TestGradingStoredGeneratedKeysAndRounding(t *testing.T) {
	body := airouter.SampleContent("reading")
	e := Exercise{TaskType: "reading", ContentJSON: contentJSON("Generated", string(body), true)}
	_, rev, actual, err := ContentContract(e)
	if err != nil || rev == nil {
		t.Fatal(err)
	}
	var c struct {
		Questions []airouter.Question `json:"questions"`
	}
	if err := json.Unmarshal(actual, &c); err != nil {
		t.Fatal(err)
	}
	for _, correct := range []int{0, 1, 2, len(c.Questions)} {
		answers := map[string]string{}
		for i, q := range c.Questions {
			choice := q.Answer
			if i >= correct {
				for key := range q.Options {
					if key != q.Answer {
						choice = key
						break
					}
				}
			}
			answers[q.ID] = choice
		}
		raw, err := gradeSubmission(e, 3600, ProgressSubmission{Revision: rev, Answers: answers})
		if err != nil {
			t.Fatal(err)
		}
		var result QuizResult
		if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatal(err)
		}
		want := int(float64(correct)*100/float64(len(c.Questions)) + 0.5)
		if result.CorrectCount == nil || *result.CorrectCount != correct || result.ScorePercent == nil || *result.ScorePercent != want || result.DurationSeconds != 3600 {
			t.Fatal(string(raw))
		}
		for i, a := range result.Answers {
			q := c.Questions[i]
			if a.QuestionID != q.ID || a.CorrectOption == nil || *a.CorrectOption != q.Answer || a.Explanation == nil || *a.Explanation != q.Explanation {
				t.Fatal("not server ordered/keyed/explained", a)
			}
		}
	}
}

func TestRevisionlessInvalidStudyFieldsPreserveValidSelections(t *testing.T) {
	for _, field := range []string{`"passage":12`, `"passage":"too short"`, `"glossary":123`} {
		h, _ := gradingHarness(t)
		body := airouter.SampleContent("reading")
		var content map[string]json.RawMessage
		if err := json.Unmarshal(body, &content); err != nil {
			t.Fatal(err)
		}
		var bad map[string]json.RawMessage
		if err := json.Unmarshal([]byte("{"+field+"}"), &bad); err != nil {
			t.Fatal(err)
		}
		for key, value := range bad {
			content[key] = value
		}
		body, _ = json.Marshal(content)
		e := gradingExercise()
		e.ContentJSON = contentJSON("Invalid study", string(body), true)
		h.quests.exercises[2] = []Exercise{e}
		var questions struct {
			Questions []airouter.Question `json:"questions"`
		}
		if err := json.Unmarshal(body, &questions); err != nil {
			t.Fatal(err)
		}
		id := questions.Questions[0].ID
		choice := questions.Questions[0].Answer
		out, err := h.svc.RecordProgress(context.Background(), "u1", e.ID, 29, ProgressSubmission{Answers: map[string]string{id: choice}})
		if err != nil {
			t.Fatal(err)
		}
		var result QuizResult
		if err := json.Unmarshal(out.QuizResult, &result); err != nil {
			t.Fatal(err)
		}
		if result.Status != "ungraded" || result.AnsweredCount != 1 || result.QuestionCount != len(questions.Questions) || result.CorrectCount != nil || result.ScorePercent != nil || result.Answers[0].CorrectOption != nil || result.Answers[0].IsCorrect != nil || result.Answers[0].Explanation != nil {
			t.Fatal(string(out.QuizResult))
		}
		h.log.calls = nil
		revision := "client-revision"
		_, err = h.svc.RecordProgress(context.Background(), "u1", e.ID, 29, ProgressSubmission{Revision: &revision, Answers: map[string]string{id: choice}})
		if !strings.Contains(fmt.Sprint(err), ErrExerciseContentInvalid.Error()) || len(h.log.calls) != 0 {
			t.Fatal(err, h.log.calls)
		}
	}
	for _, questions := range []string{`[{"id":"q","options":{"A":"same","B":"Same"}}]`, `[{"id":"q","options":{"A":"one","B":"two"}},{"id":"q","options":{"A":"one","B":"two"}}]`, `[{"id":"q","options":{"A":null,"B":"two"}}]`, `[{"id":"q","options":{"A":"one"}}]`} {
		h, _ := gradingHarness(t)
		h.quests.exercises[2][0].ContentJSON = json.RawMessage(`{"content_schema":1,"content":{"questions":` + questions + `}}`)
		_, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 29, ProgressSubmission{Answers: map[string]string{"q": "A"}})
		if err != ErrInvalidAnswers || len(h.log.calls) != 0 {
			t.Fatal(err, h.log.calls)
		}
	}
}
