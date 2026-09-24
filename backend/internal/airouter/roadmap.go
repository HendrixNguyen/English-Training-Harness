package airouter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// The §6.1 shape.
const (
	Modules            = 4
	DaysPerModule      = 7
	TasksPerDay        = 3
	DefaultTaskMinutes = 10
)

// §6.1 constraint 4: "Each Daily Quest MUST be calculated to take
// approximately 30 minutes to complete, split into 3 distinct tasks (10 mins
// each)". "Approximately" is read as ±10 minutes on the day and ±5 on a task;
// quests hard-codes total_minutes_required: 30 and the pet needs 1800 s, so
// a day outside this band is a promise the learner cannot keep.
const (
	minTaskMinutes = 5
	maxTaskMinutes = 15
	minDayMinutes  = 20
	maxDayMinutes  = 40
)

// TaskTypes are the §3.2 task_category values in the §6.1 order
// (Vocabulary/Grammar, Reading/Listening, Practice/Interactive).
var TaskTypes = []string{"vocabulary", "reading", "practice"}

var cefrLevels = map[string]bool{"A1": true, "A2": true, "B1": true, "B2": true, "C1": true, "C2": true}

// Roadmap is the validated §6.1 output; it is also what roadmaps.roadmap_json stores.
type Roadmap struct {
	Title     string   `json:"title"`
	CEFRLevel string   `json:"cefr_level"`
	Modules   []Module `json:"modules"`
}

// Module is one week.
type Module struct {
	Week  int    `json:"week"`
	Title string `json:"title"`
	Focus string `json:"focus"`
	Days  []Day  `json:"days"`
}

// Day is one daily quest.
type Day struct {
	Title string `json:"title"`
	Tasks []Task `json:"tasks"`
}

// Task is one ~10-minute exercise. Marshalled whole into exercises.content_json,
// so its title and duration_minutes are what GET /quests/daily exposes.
type Task struct {
	Type            string          `json:"type"`
	Title           string          `json:"title"`
	DurationMinutes int             `json:"duration_minutes"`
	Content         json.RawMessage `json:"content,omitempty"`
}

// Exercise is one §3.2 exercises row, ready to insert.
type Exercise struct {
	DayNumber   int
	TaskType    string
	ContentJSON json.RawMessage
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidRoadmap, fmt.Sprintf(format, args...))
}

// ParseRoadmap decodes a model response strictly: no markdown fences or
// preamble (§6.1 constraint 1), no trailing tokens, exactly 4 modules × 7 days
// × 3 tasks with the three task types each present once, non-empty task
// titles, durations within (0, 30] (a missing duration becomes 10), and a
// valid cefr_level. Unknown extra fields are tolerated. Nothing is stripped or
// repaired — a non-conforming answer is the caller's cue to retry.
func ParseRoadmap(raw string) (Roadmap, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Roadmap{}, invalid("empty response")
	}
	if strings.HasPrefix(trimmed, "```") {
		return Roadmap{}, invalid("response is wrapped in a markdown fence")
	}
	if !strings.HasPrefix(trimmed, "{") {
		return Roadmap{}, invalid("response does not start with a JSON object")
	}

	dec := json.NewDecoder(strings.NewReader(trimmed))
	var r Roadmap
	if err := dec.Decode(&r); err != nil {
		return Roadmap{}, invalid("decoding: %v", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return Roadmap{}, invalid("trailing content after the JSON object")
	}

	if !cefrLevels[r.CEFRLevel] {
		return Roadmap{}, invalid("cefr_level %q is not a CEFR level", r.CEFRLevel)
	}
	if len(r.Modules) != Modules {
		return Roadmap{}, invalid("%d modules, want %d", len(r.Modules), Modules)
	}
	for mi := range r.Modules {
		m := &r.Modules[mi]
		if len(m.Days) != DaysPerModule {
			return Roadmap{}, invalid("module %d has %d days, want %d", mi+1, len(m.Days), DaysPerModule)
		}
		for di := range m.Days {
			d := &m.Days[di]
			if len(d.Tasks) != TasksPerDay {
				return Roadmap{}, invalid("module %d day %d has %d tasks, want %d", mi+1, di+1, len(d.Tasks), TasksPerDay)
			}
			seen := map[string]bool{}
			dayMinutes := 0
			for ti := range d.Tasks {
				task := &d.Tasks[ti]
				if !isTaskType(task.Type) {
					return Roadmap{}, invalid("module %d day %d task %d has type %q", mi+1, di+1, ti+1, task.Type)
				}
				if seen[task.Type] {
					return Roadmap{}, invalid("module %d day %d repeats task type %q", mi+1, di+1, task.Type)
				}
				seen[task.Type] = true
				if strings.TrimSpace(task.Title) == "" {
					return Roadmap{}, invalid("module %d day %d task %d has no title", mi+1, di+1, ti+1)
				}
				if task.DurationMinutes == 0 {
					task.DurationMinutes = DefaultTaskMinutes
				}
				if task.DurationMinutes < minTaskMinutes || task.DurationMinutes > maxTaskMinutes {
					return Roadmap{}, invalid("module %d day %d task %d duration %d is outside %d..%d", mi+1, di+1, ti+1, task.DurationMinutes, minTaskMinutes, maxTaskMinutes)
				}
				dayMinutes += task.DurationMinutes
			}
			if dayMinutes < minDayMinutes || dayMinutes > maxDayMinutes {
				return Roadmap{}, invalid("module %d day %d adds up to %d minutes, want %d..%d (§6.1: approximately 30)", mi+1, di+1, dayMinutes, minDayMinutes, maxDayMinutes)
			}
		}
	}
	return r, nil
}

func isTaskType(s string) bool {
	for _, t := range TaskTypes {
		if s == t {
			return true
		}
	}
	return false
}

// Exercises flattens a validated roadmap into its 84 exercises rows.
// day_number = (module-1)*7 + day; content_json is the whole task object.
func (r Roadmap) Exercises() []Exercise {
	out := make([]Exercise, 0, Modules*DaysPerModule*TasksPerDay)
	for mi, m := range r.Modules {
		for di, d := range m.Days {
			for _, t := range d.Tasks {
				content, _ := json.Marshal(t) // a validated Task always marshals
				out = append(out, Exercise{
					DayNumber:   mi*DaysPerModule + di + 1,
					TaskType:    t.Type,
					ContentJSON: content,
				})
			}
		}
	}
	return out
}
