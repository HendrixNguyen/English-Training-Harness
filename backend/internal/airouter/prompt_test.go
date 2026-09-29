package airouter

import (
	"strings"
	"testing"
)

func TestRoadmapSystemPromptIsSpec61Verbatim(t *testing.T) {
	for _, want := range []string{
		"You are an elite AI Language Curriculum Architect.",
		"CRITICAL CONSTRAINTS:",
		"1. Output ONLY valid JSON matching the requested schema. No markdown backticks, no code blocks, no conversational preamble.",
		"2. Structure the output into 4 distinct Modules (Weeks).",
		"3. Each Module must contain 7 Daily Quests (Total 28 Days).",
		"4. Each Daily Quest MUST be calculated to take approximately 30 minutes to complete, split into 3 distinct tasks (10 mins each): Vocabulary/Grammar, Reading/Listening, and Practice/Interactive.",
		"5. Difficulty must scale progressively across the modules.",
	} {
		if !strings.Contains(RoadmapSystemPrompt, want) {
			t.Errorf("RoadmapSystemPrompt missing %q", want)
		}
	}
}

func TestRoadmapUserPromptCarriesTheLearnerAndTheSchema(t *testing.T) {
	p := RoadmapUserPrompt("B1", "IELTS 7.0 Preparation", 30)
	for _, want := range []string{"B1", "IELTS 7.0 Preparation", "30 minutes", RoadmapSchema, `"vocabulary"`, `"reading"`, `"practice"`} {
		if !strings.Contains(p, want) {
			t.Errorf("RoadmapUserPrompt missing %q", want)
		}
	}
}

func TestRoadmapUserPromptCarriesLevelAndGoalGuidance(t *testing.T) {
	p := RoadmapUserPrompt("B1", "Business English", 30)
	for _, want := range []string{
		LevelGuidance("B1"),
		`Business English`,
		`never generic greetings, numbers or classroom basics unless the level is A1`,
		RoadmapSchema,
		"Daily study time: 30 minutes",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("RoadmapUserPrompt missing %q\ngot: %s", want, p)
		}
	}
}
