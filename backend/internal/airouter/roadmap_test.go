package airouter

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// validRoadmapJSON builds a 4x7x3 roadmap; mutate lets a test break one thing.
func validRoadmapJSON(t *testing.T, mutate func(r *Roadmap)) string {
	t.Helper()
	r := Roadmap{Title: "Road to IELTS 7", CEFRLevel: "B1"}
	for m := 1; m <= Modules; m++ {
		mod := Module{Week: m, Title: fmt.Sprintf("Week %d", m), Focus: "focus"}
		for d := 1; d <= DaysPerModule; d++ {
			day := Day{Title: fmt.Sprintf("Day %d", (m-1)*DaysPerModule+d)}
			for _, tt := range TaskTypes {
				day.Tasks = append(day.Tasks, Task{Type: tt, Title: tt + " task", DurationMinutes: 10, Content: SampleContent(tt)})
			}
			mod.Days = append(mod.Days, day)
		}
		r.Modules = append(r.Modules, mod)
	}
	if mutate != nil {
		mutate(&r)
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func TestParseRoadmapAcceptsTheExactShape(t *testing.T) {
	r, err := ParseRoadmap(validRoadmapJSON(t, nil))
	if err != nil {
		t.Fatalf("ParseRoadmap: %v", err)
	}
	if len(r.Modules) != 4 || len(r.Modules[3].Days) != 7 || len(r.Modules[3].Days[6].Tasks) != 3 {
		t.Errorf("shape = %d modules, %d days, %d tasks", len(r.Modules), len(r.Modules[3].Days), len(r.Modules[3].Days[6].Tasks))
	}
	if r.CEFRLevel != "B1" {
		t.Errorf("CEFRLevel = %q", r.CEFRLevel)
	}
}

func TestParseRoadmapToleratesSurroundingWhitespaceAndUnknownFields(t *testing.T) {
	raw := "\n  " + strings.Replace(validRoadmapJSON(t, nil), `{"title"`, `{"model_note":"hi","title"`, 1) + "\n"
	if _, err := ParseRoadmap(raw); err != nil {
		t.Fatalf("ParseRoadmap: %v", err)
	}
}

func TestParseRoadmapDefaultsAMissingDurationToTen(t *testing.T) {
	r, err := ParseRoadmap(validRoadmapJSON(t, func(r *Roadmap) { r.Modules[0].Days[0].Tasks[0].DurationMinutes = 0 }))
	if err != nil {
		t.Fatalf("ParseRoadmap: %v", err)
	}
	if got := r.Modules[0].Days[0].Tasks[0].DurationMinutes; got != DefaultTaskMinutes {
		t.Errorf("DurationMinutes = %d, want %d", got, DefaultTaskMinutes)
	}
}

func TestParseRoadmapRejects(t *testing.T) {
	cases := map[string]string{
		"markdown fence":      "```json\n" + validRoadmapJSON(t, nil) + "\n```",
		"preamble":            "Here is your roadmap: " + validRoadmapJSON(t, nil),
		"trailing garbage":    validRoadmapJSON(t, nil) + " {}",
		"empty":               "",
		"not an object":       `[1,2,3]`,
		"three modules":       validRoadmapJSON(t, func(r *Roadmap) { r.Modules = r.Modules[:3] }),
		"five modules":        validRoadmapJSON(t, func(r *Roadmap) { r.Modules = append(r.Modules, r.Modules[0]) }),
		"six days":            validRoadmapJSON(t, func(r *Roadmap) { r.Modules[1].Days = r.Modules[1].Days[:6] }),
		"two tasks":           validRoadmapJSON(t, func(r *Roadmap) { r.Modules[2].Days[3].Tasks = r.Modules[2].Days[3].Tasks[:2] }),
		"four tasks":          validRoadmapJSON(t, func(r *Roadmap) { d := &r.Modules[2].Days[3]; d.Tasks = append(d.Tasks, d.Tasks[0]) }),
		"duplicate task type": validRoadmapJSON(t, func(r *Roadmap) { r.Modules[0].Days[0].Tasks[1].Type = "vocabulary" }),
		"unknown task type":   validRoadmapJSON(t, func(r *Roadmap) { r.Modules[0].Days[0].Tasks[2].Type = "listening" }),
		"empty task title":    validRoadmapJSON(t, func(r *Roadmap) { r.Modules[3].Days[6].Tasks[0].Title = "" }),
		"absurd duration":     validRoadmapJSON(t, func(r *Roadmap) { r.Modules[3].Days[6].Tasks[0].DurationMinutes = 120 }),
		"bad cefr":            validRoadmapJSON(t, func(r *Roadmap) { r.CEFRLevel = "B7" }),
		"free-form content":   validRoadmapJSON(t, func(r *Roadmap) { r.Modules[0].Days[0].Tasks[2].Content = json.RawMessage(`{"items":[]}`) }),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseRoadmap(raw)
			if err == nil {
				t.Fatal("ParseRoadmap accepted it")
			}
			if !errors.Is(err, ErrInvalidRoadmap) {
				t.Errorf("err = %v, want it to wrap ErrInvalidRoadmap", err)
			}
		})
	}
}

func TestExercisesFlattensTo84RowsCarryingTitleAndDuration(t *testing.T) {
	r, err := ParseRoadmap(validRoadmapJSON(t, nil))
	if err != nil {
		t.Fatalf("ParseRoadmap: %v", err)
	}
	ex := r.Exercises()
	if len(ex) != 84 {
		t.Fatalf("len = %d, want 84", len(ex))
	}
	if ex[0].DayNumber != 1 || ex[83].DayNumber != 28 || ex[21].DayNumber != 8 {
		t.Errorf("day numbers: first %d, [21] %d, last %d; want 1, 8, 28", ex[0].DayNumber, ex[21].DayNumber, ex[83].DayNumber)
	}
	seen := map[string]int{}
	for _, e := range ex {
		seen[e.TaskType]++
		var meta struct {
			Type            string `json:"type"`
			Title           string `json:"title"`
			DurationMinutes int    `json:"duration_minutes"`
		}
		if err := json.Unmarshal(e.ContentJSON, &meta); err != nil {
			t.Fatalf("content_json is not JSON: %v", err)
		}
		if meta.Title == "" || meta.DurationMinutes != 10 || meta.Type != e.TaskType {
			t.Errorf("content_json = %s, want title, duration_minutes=10 and type=%s (what quests' toTask reads)", e.ContentJSON, e.TaskType)
		}
	}
	if seen["vocabulary"] != 28 || seen["reading"] != 28 || seen["practice"] != 28 {
		t.Errorf("task type counts = %v, want 28 each", seen)
	}
}
