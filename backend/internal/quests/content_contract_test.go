package quests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTaskContentContract(t *testing.T) {
	quiz := `{"questions":[{"id":"q1","prompt":"Choose the verb.","options":{"A":"go","B":"went"}}]}`
	cases := []struct {
		name, body, origin string
		revision           bool
	}{
		{"nested passage draft", `{"content":{"passage":"Read about meetings."}}`, "outline", false},
		{"nested word draft", `{"content":{"words":[{"term":"book","definition":"a thing to read"}]}}`, "outline", false},
		{"one option draft", `{"content":{"questions":[{"id":"q1","prompt":"Choose.","options":{"A":"one"}}]}}`, "outline", false},
		{"root passage", `{"passage":"A historical passage.","questions":[]}`, "legacy", true},
		{"root words", `{"words":[{"term":"book","definition":"a thing to read"}]}`, "legacy", true},
		{"explicit word list", `{"content_origin":"legacy","content":{"words":[{"term":"book","definition":"a thing to read","example":"Read a book."}]}}`, "legacy", true},
		{"explicit passage", `{"content_origin":"legacy","content":{"passage":"A historical passage."}}`, "legacy", true},
		{"complete nested keyless quiz", `{"content":` + quiz + `}`, "legacy", true},
		{"root quiz", quiz, "legacy", true},
		{"one option root", `{"questions":[{"id":"q1","prompt":"Choose.","options":{"A":"one"}}]}`, "outline", false},
		{"bad generated", `{"content_schema":1,"content":{"words":[]}}`, "generated", false},
		{"valid generated", string(contentJSON("title", string(airouter.SampleContent("vocabulary")), true)), "generated", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			task := toTask(Exercise{TaskType: "vocabulary", ContentJSON: json.RawMessage(c.body)})
			if task.ContentOrigin != c.origin || (task.ContentRevision != nil) != c.revision {
				t.Fatalf("origin/revision=%s/%v want %s/%t", task.ContentOrigin, task.ContentRevision, c.origin, c.revision)
			}
		})
	}
}
func TestContentRevisionCanonical(t *testing.T) {
	a := toTask(Exercise{TaskType: "reading", ContentJSON: json.RawMessage(`{"title":"First","duration_minutes":10,"passage":"Actual passage","questions":[],"quiz_result":{"score":1}}`)})
	b := toTask(Exercise{TaskType: "reading", IsCompleted: true, ContentJSON: json.RawMessage(`{"questions":[],"passage":"Actual passage","title":"Second","duration_minutes":20,"quiz_result":{"score":2}}`)})
	if a.ContentRevision == nil || b.ContentRevision == nil || *a.ContentRevision != *b.ContentRevision {
		t.Fatal("revision changed for metadata/key order/completion")
	}
	for _, e := range []Exercise{{TaskType: "vocabulary", ContentJSON: b.ContentJSON}, {TaskType: "reading", ContentJSON: json.RawMessage(`{"passage":"Changed passage","questions":[]}`)}} {
		c := toTask(e)
		if c.ContentRevision == nil || *a.ContentRevision == *c.ContentRevision {
			t.Fatal("revision did not change with content/type")
		}
	}
}
func TestInvalidGeneratedRequiresExplicitRepair(t *testing.T) {
	now := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	for _, mode := range []string{"ordinary", "success", "bad output", "completed", "future", "missing", "limited", "lost valid", "lost invalid"} {
		t.Run(mode, func(t *testing.T) {
			target := baseTarget(now)
			target.Exercise.ContentJSON = contentJSON("Old", `{"words":[]}`, true)
			original := string(target.Exercise.ContentJSON)
			repo := &fakeContentRepo{target: target}
			ai := &fakeContentAI{answers: []string{string(airouter.SampleContent("vocabulary"))}}
			lim := &fakeContentLimiter{}
			path := "/api/v1/quests/exercises/ex-1?repair=true"
			want := 200
			wantAI := 1
			switch mode {
			case "ordinary":
				path = "/api/v1/quests/exercises/ex-1"
				want = 409
				wantAI = 0
			case "bad output":
				ai.answers = []string{"bad", "bad", string(airouter.SampleContent("vocabulary"))}
				want = 502
				wantAI = 2
			case "completed":
				repo.target.Exercise.IsCompleted = true
				want = 409
				wantAI = 0
			case "future":
				repo.target.Exercise.DayNumber = 2
				want = 404
				wantAI = 0
			case "missing":
				repo.targetErr = ErrExerciseNotFound
				want = 404
				wantAI = 0
			case "limited":
				lim.err = airouter.ErrRateLimited
				want = 429
				wantAI = 0
			case "lost valid", "lost invalid":
				repo.loseCAS = true
				if mode == "lost valid" {
					repo.winner = contentJSON("Winner", string(airouter.SampleContent("vocabulary")), true)
				} else {
					repo.winner = repo.target.Exercise.ContentJSON
					want = 409
				}
			}
			svc := NewContentService(repo, ai, lim, func() time.Time { return now })
			w := httptest.NewRecorder()
			newContentRouter(svc, "u1").ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
			if w.Code != want || ai.calls != wantAI {
				t.Fatalf("status=%d AI=%d want %d/%d: %s", w.Code, ai.calls, want, wantAI, w.Body)
			}
			if want == 409 && !strings.Contains(w.Body.String(), `"error":"exercise_content_invalid"`) {
				t.Fatal(w.Body.String())
			}
			if mode == "bad output" && string(repo.target.Exercise.ContentJSON) != original {
				t.Fatal("failed repair mutated original")
			}
			if strings.HasPrefix(mode, "lost") && repo.targetCalls != 2 {
				t.Fatalf("loser rereads=%d want2 total", repo.targetCalls)
			}
			if mode == "success" && repo.savedLevel != "B1" {
				t.Fatalf("saved level=%s", repo.savedLevel)
			}
		})
	}
}
func TestSavedGenerationUsesOriginalLevel(t *testing.T) {
	now := time.Now()
	target := baseTarget(now)
	target.Level = "C2"
	target.Exercise.TaskType = "reading"
	body := string(airouter.SampleContent("reading"))
	for _, level := range []string{"B1", ""} {
		t.Run("saved="+level, func(t *testing.T) {
			var m map[string]any
			_ = json.Unmarshal(contentJSON("Reading", body, true), &m)
			if level != "" {
				m["content_level"] = level
			}
			target.Exercise.ContentJSON, _ = json.Marshal(m)
			repo := &fakeContentRepo{target: target}
			ai := &fakeContentAI{}
			lim := &fakeContentLimiter{}
			svc := NewContentService(repo, ai, lim, func() time.Time { return now })
			task, err := svc.Exercise(context.Background(), "u1", "ex-1")
			if err != nil || task.ContentOrigin != "generated" || ai.calls != 0 || lim.calls != 0 {
				t.Fatalf("cached task=%+v err=%v AI=%d", task, err, ai.calls)
			}
		})
	}
	target.Exercise.ContentJSON = contentJSON("Reading", `{"passage":"tiny","questions":[]}`, true)
	_, err := NewContentService(&fakeContentRepo{target: target}, &fakeContentAI{}, &fakeContentLimiter{}, func() time.Time { return now }).Exercise(context.Background(), "u1", "ex-1")
	if !errors.Is(err, ErrExerciseContentInvalid) {
		t.Fatalf("err=%v", err)
	}
}

func TestLegacyQuizMustBeComplete(t *testing.T) {
	base := airouter.PracticeContent{Questions: []airouter.Question{{ID: "q1", Prompt: "Which form fits?", Options: map[string]string{"A": "go", "B": "went"}, Answer: "B", Explanation: "The past tense fits yesterday."}, {ID: "q2", Prompt: "Which place fits?", Options: map[string]string{"A": "town", "B": "school"}}}}
	for _, name := range []string{"keyed and keyless", "duplicate ids", "duplicate prompts", "duplicate options", "unknown option key", "blank prompt", "placeholder prompt", "placeholder explanation", "wrong answer", "unkeyed explanation"} {
		t.Run(name, func(t *testing.T) {
			raw, _ := json.Marshal(base)
			var c airouter.PracticeContent
			_ = json.Unmarshal(raw, &c)
			switch name {
			case "duplicate ids":
				c.Questions[1].ID = c.Questions[0].ID
			case "duplicate prompts":
				c.Questions[1].Prompt = " WHICH  form fits? "
			case "duplicate options":
				c.Questions[0].Options["B"] = " GO "
			case "unknown option key":
				c.Questions[0].Options["E"] = "new"
			case "blank prompt":
				c.Questions[0].Prompt = " "
			case "placeholder prompt":
				c.Questions[0].Prompt = "…"
			case "placeholder explanation":
				c.Questions[0].Explanation = "TODO"
			case "wrong answer":
				c.Questions[0].Answer = "D"
			case "unkeyed explanation":
				c.Questions[1].Explanation = "Some context."
			}
			body, _ := json.Marshal(c)
			origin, revision, _, err := ContentContract(Exercise{TaskType: "practice", ContentJSON: contentJSON("Quiz", string(body), false)})
			want := "outline"
			if name == "keyed and keyless" {
				want = "legacy"
			}
			if err != nil || origin != want || (revision != nil) != (want == "legacy") {
				t.Fatalf("origin=%s revision=%v err=%v want%s", origin, revision, err, want)
			}
		})
	}
}

func TestGeneratedRevisionIgnoresEnvelope(t *testing.T) {
	body := airouter.SampleContent("reading")
	a := contentJSON("A", string(body), true)
	var m map[string]any
	_ = json.Unmarshal(a, &m)
	m["title"] = "B"
	m["duration_minutes"] = 25
	m["content_level"] = "B1"
	m["quiz_result"] = map[string]any{"score_percent": 100}
	m["is_completed"] = true
	b, _ := json.Marshal(m)
	at := toTask(Exercise{TaskType: "reading", ContentJSON: a})
	bt := toTask(Exercise{TaskType: "reading", ContentJSON: b, IsCompleted: true})
	if at.ContentRevision == nil || bt.ContentRevision == nil || *at.ContentRevision != *bt.ContentRevision {
		t.Fatal("envelope changed revision")
	}
	var c airouter.ReadingContent
	_ = json.Unmarshal(body, &c)
	c.Questions[0].Answer = "A"
	changed, _ := json.Marshal(c)
	m["content"] = json.RawMessage(changed)
	b, _ = json.Marshal(m)
	bt = toTask(Exercise{TaskType: "reading", ContentJSON: b})
	if bt.ContentRevision == nil || *at.ContentRevision == *bt.ContentRevision {
		t.Fatal("answer content must change revision")
	}
}

func TestHistoricalGeneratedUnionBoundsAndQuality(t *testing.T) {
	for _, n := range []int{239, 240, 519, 520, 1440, 3040, 3041} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			var c airouter.ReadingContent
			_ = json.Unmarshal(airouter.SampleContent("reading"), &c)
			c.Passage = strings.Repeat("x", n)
			body, _ := json.Marshal(c)
			_, revision, _, err := ContentContract(Exercise{TaskType: "reading", ContentJSON: contentJSON("Read", string(body), true)})
			valid := n >= 240 && n <= 3040
			if (err == nil) != valid || (revision != nil) != valid {
				t.Fatalf("runes%d err=%v revision=%v", n, err, revision)
			}
		})
	}
	for _, mode := range []string{"duplicate prompt", "placeholder", "missing answer", "count", "bad saved level"} {
		t.Run(mode, func(t *testing.T) {
			var c airouter.ReadingContent
			_ = json.Unmarshal(airouter.SampleContent("reading"), &c)
			switch mode {
			case "duplicate prompt":
				c.Questions[1].Prompt = c.Questions[0].Prompt
			case "placeholder":
				c.Questions[0].Explanation = "TBD"
			case "missing answer":
				c.Questions[0].Answer = ""
			case "count":
				c.Questions = c.Questions[:1]
			}
			body, _ := json.Marshal(c)
			snapshot := contentJSON("Read", string(body), true)
			if mode == "bad saved level" {
				var m map[string]json.RawMessage
				_ = json.Unmarshal(snapshot, &m)
				m["content_level"] = json.RawMessage(`"ZZ"`)
				snapshot, _ = json.Marshal(m)
			}
			_, _, _, err := ContentContract(Exercise{TaskType: "reading", ContentJSON: snapshot})
			if !errors.Is(err, ErrExerciseContentInvalid) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestContentServiceHydratesAmbiguousDraftsOnly(t *testing.T) {
	now := time.Now()
	for _, body := range []string{`{"content":{"passage":"Read about work."}}`, `{"content":{"questions":[{"id":"q1","prompt":"Choose.","options":{"A":"one"}}]}}`, `{"passage":"A complete historical passage."}`, `{"content_origin":"legacy","content":{"words":[{"term":"book","definition":"a text"}]}}`, `{"content":{"questions":[{"id":"q1","prompt":"Choose a verb.","options":{"A":"go","B":"went"}}]}}`} {
		t.Run(body, func(t *testing.T) {
			target := baseTarget(now)
			target.Exercise.ContentJSON = json.RawMessage(body)
			repo := &fakeContentRepo{target: target}
			ai := &fakeContentAI{answers: []string{string(airouter.SampleContent("vocabulary"))}}
			lim := &fakeContentLimiter{}
			task, err := NewContentService(repo, ai, lim, func() time.Time { return now }).Exercise(context.Background(), "u1", "ex-1")
			if err != nil {
				t.Fatal(err)
			}
			origin, _, _, _ := ContentContract(target.Exercise)
			wantCalls := 0
			wantOrigin := "legacy"
			if origin == "outline" {
				wantCalls = 1
				wantOrigin = "generated"
			}
			if ai.calls != wantCalls || lim.calls != wantCalls || task.ContentOrigin != wantOrigin {
				t.Fatalf("AI%d limiter%d origin%s", ai.calls, lim.calls, task.ContentOrigin)
			}
		})
	}
}

func TestHistoricalGeneratedRetainsQualityRejectionReason(t *testing.T) {
	var c airouter.ReadingContent
	_ = json.Unmarshal(airouter.SampleContent("reading"), &c)
	c.Questions[1].Prompt = c.Questions[0].Prompt
	body, _ := json.Marshal(c)
	_, _, _, err := ContentContract(Exercise{TaskType: "reading", ContentJSON: contentJSON("Read", string(body), true)})
	if !errors.Is(err, ErrExerciseContentInvalid) || !strings.Contains(err.Error(), "question q2 repeats prompt of question q1") {
		t.Fatalf("quality reason masked: %v", err)
	}
}
