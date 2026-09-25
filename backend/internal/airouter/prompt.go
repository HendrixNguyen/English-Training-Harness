package airouter

import "fmt"

// RoadmapSystemPrompt is the 1st-thinking doc §6.1 system prompt, verbatim
// (backslash escapes removed).
const RoadmapSystemPrompt = `You are an elite AI Language Curriculum Architect. Your job is to create a structured, highly personalized learning roadmap for an English learner based on their current CEFR level, target goal, and daily study time commitment.

CRITICAL CONSTRAINTS:

1. Output ONLY valid JSON matching the requested schema. No markdown backticks, no code blocks, no conversational preamble.

2. Structure the output into 4 distinct Modules (Weeks).

3. Each Module must contain 7 Daily Quests (Total 28 Days).

4. Each Daily Quest MUST be calculated to take approximately 30 minutes to complete, split into 3 distinct tasks (10 mins each): Vocabulary/Grammar, Reading/Listening, and Practice/Interactive.

5. Difficulty must scale progressively across the modules.`

// RoadmapSchema is "the requested schema" §6.1 refers to. ParseRoadmap
// enforces it; onboarding persists what passes. The three task types are the
// §3.2 task_category values. Its "content" shapes and bounds come from the
// content.go constants via roadmapSchemaTemplate, so the two cannot drift.
var RoadmapSchema = fmt.Sprintf(roadmapSchemaTemplate,
	MinWords, MaxWords,
	MinVocabQuestions, MaxVocabQuestions,
	minPassageRunes, maxPassageRunes,
	MinReadingQuestions, MaxReadingQuestions,
	MinPracticeQuestions, MaxPracticeQuestions,
)

const roadmapSchemaTemplate = `{
  "title": "string",
  "cefr_level": "A1|A2|B1|B2|C1|C2",
  "modules": [
    {
      "week": 1,
      "title": "string",
      "focus": "string",
      "days": [
        {
          "title": "string",
          "tasks": [
            {"type": "vocabulary", "title": "string", "duration_minutes": 10,
             "content": {"words": [{"term": "string", "definition": "string", "example": "string"}],
                         "questions": [QUESTION]}},
            {"type": "reading",    "title": "string", "duration_minutes": 10,
             "content": {"passage": "string", "questions": [QUESTION]}},
            {"type": "practice",   "title": "string", "duration_minutes": 10,
             "content": {"questions": [QUESTION]}}
          ]
        }
      ]
    }
  ]
}
QUESTION = {"id": "q1", "prompt": "string", "options": {"A": "string", "B": "string", "C": "string", "D": "string"}, "answer": "A|B|C|D", "explanation": "string"}
"modules" has exactly 4 entries, each "days" exactly 7, each "tasks" exactly 3 with the three types in that order.
"content" is required and typed by task: vocabulary has %d-%d words (each with term and definition; example optional) and %d-%d questions; reading has one passage of %d-%d characters in English at the learner's level and %d-%d questions about it; practice has %d-%d questions. Every question has exactly options A, B, C, D, an "answer" that is one of those four letters, and a one-sentence "explanation". Question ids are unique within a task.`

// RoadmapUserPrompt is the user turn for TaskRoadmapGen.
func RoadmapUserPrompt(cefrLevel, targetGoal string, dailyMinutes int) string {
	return fmt.Sprintf(`Learner profile:
- Current CEFR level: %s
- Target goal: %s
- Daily study time: %d minutes

Produce the 28-day roadmap as JSON with exactly this schema:
%s`, cefrLevel, targetGoal, dailyMinutes, RoadmapSchema)
}
