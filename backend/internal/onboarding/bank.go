// Package onboarding runs the placement test and kicks off the roadmap
// (1st-thinking §5.1 steps 4-5; backend spec §6.1 for the wire DTOs; §4 for
// the quiz:placement hash).
package onboarding

// Question is one placement item. Correct and Level never leave the server;
// they are what the grader prompt is built from.
type Question struct {
	ID      string
	Prompt  string
	Options map[string]string // "A".."D"
	Correct string
	Level   string // the CEFR level the item probes
}

// Bank is the fixed placement set: two items per level A1-C1, ordered by
// difficulty. Ten items keep the grader prompt small and the quiz under five
// minutes; the LLM grader (not a score table) turns the pattern of answers
// into a level, which is what §6.1 asks for.
var Bank = []Question{
	{ID: "q1", Level: "A1", Prompt: "She ___ a teacher.", Options: map[string]string{"A": "am", "B": "is", "C": "are", "D": "be"}, Correct: "B"},
	{ID: "q2", Level: "A1", Prompt: "I ___ to the cinema yesterday.", Options: map[string]string{"A": "go", "B": "goes", "C": "went", "D": "gone"}, Correct: "C"},
	{ID: "q3", Level: "A2", Prompt: "There isn't ___ milk left in the fridge.", Options: map[string]string{"A": "some", "B": "any", "C": "many", "D": "a few"}, Correct: "B"},
	{ID: "q4", Level: "A2", Prompt: "If it rains tomorrow, we ___ at home.", Options: map[string]string{"A": "stay", "B": "will stay", "C": "stayed", "D": "would stay"}, Correct: "B"},
	{ID: "q5", Level: "B1", Prompt: "I've lived here ___ 2019.", Options: map[string]string{"A": "for", "B": "since", "C": "during", "D": "from"}, Correct: "B"},
	{ID: "q6", Level: "B1", Prompt: "The report ___ by the team last week.", Options: map[string]string{"A": "wrote", "B": "was written", "C": "has written", "D": "is writing"}, Correct: "B"},
	{ID: "q7", Level: "B2", Prompt: "Choose the word closest in meaning to 'reluctant'.", Options: map[string]string{"A": "eager", "B": "unwilling", "C": "careless", "D": "confident"}, Correct: "B"},
	{ID: "q8", Level: "B2", Prompt: "Hardly ___ the meeting started when the fire alarm went off.", Options: map[string]string{"A": "had", "B": "has", "C": "did", "D": "was"}, Correct: "A"},
	{ID: "q9", Level: "C1", Prompt: "Her argument was so ___ that even her critics conceded the point.", Options: map[string]string{"A": "tenuous", "B": "cogent", "C": "verbose", "D": "tentative"}, Correct: "B"},
	{ID: "q10", Level: "C1", Prompt: "Had it not been for the delay, we ___ the deadline.", Options: map[string]string{"A": "would meet", "B": "had met", "C": "would have met", "D": "met"}, Correct: "C"},
}

// Lookup finds a bank item by id.
func Lookup(id string) (Question, bool) {
	for _, q := range Bank {
		if q.ID == id {
			return q, true
		}
	}
	return Question{}, false
}

// PublicQuestion is what GET /api/v1/onboarding/quiz shows: no answer, no level.
type PublicQuestion struct {
	ID      string            `json:"id"`
	Prompt  string            `json:"prompt"`
	Options map[string]string `json:"options"`
}

// QuizResponse is the GET /api/v1/onboarding/quiz 200 body. This endpoint is
// not in either spec; see CODEMAP.
type QuizResponse struct {
	Questions []PublicQuestion `json:"questions"`
}

// PublicBank strips the server-only fields.
func PublicBank() QuizResponse {
	out := QuizResponse{Questions: make([]PublicQuestion, 0, len(Bank))}
	for _, q := range Bank {
		out.Questions = append(out.Questions, PublicQuestion{ID: q.ID, Prompt: q.Prompt, Options: q.Options})
	}
	return out
}
