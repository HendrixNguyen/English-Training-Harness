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
// §3.2 task_category values.
const RoadmapSchema = `{
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
            {"type": "vocabulary", "title": "string", "duration_minutes": 10, "content": {}},
            {"type": "reading",    "title": "string", "duration_minutes": 10, "content": {}},
            {"type": "practice",   "title": "string", "duration_minutes": 10, "content": {}}
          ]
        }
      ]
    }
  ]
}
"modules" has exactly 4 entries, each "days" exactly 7, each "tasks" exactly 3 with the three types in that order. "content" is free-form JSON for the task material (word lists, passages, prompts).`

// RoadmapUserPrompt is the user turn for TaskRoadmapGen. It carries a CEFR
// descriptor (LevelGuidance) and the goal's register so a B1 Business
// English learner gets workplace tasks, not "Good morning / one, two,
// three" — the guidance lives here, not in RoadmapSystemPrompt, which stays
// the §6.1 text verbatim (prompt_test.go pins it).
func RoadmapUserPrompt(cefrLevel, targetGoal string, dailyMinutes int) string {
	return fmt.Sprintf(`Learner profile:
- Current CEFR level: %s
- Target goal: %s
- Daily study time: %d minutes

Write every task for a %s learner. %s
Every task must use the language of the learner's goal ("%s"): its situations, vocabulary and register. Day 1 starts inside that goal — never generic greetings, numbers or classroom basics unless the level is A1.

Produce the 28-day roadmap as JSON with exactly this schema:
%s`, cefrLevel, targetGoal, dailyMinutes, cefrLevel, LevelGuidance(cefrLevel), targetGoal, RoadmapSchema)
}
