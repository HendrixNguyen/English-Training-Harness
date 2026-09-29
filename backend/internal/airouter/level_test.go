package airouter

import (
	"strings"
	"testing"
)

func TestLevelGuidanceCoversEveryLevel(t *testing.T) {
	for _, level := range []string{"A1", "A2", "B1", "B2", "C1", "C2"} {
		g := LevelGuidance(level)
		if g == "" {
			t.Errorf("LevelGuidance(%q) is empty", level)
		}
		if !strings.Contains(g, level) {
			t.Errorf("LevelGuidance(%q) = %q, missing its own level code", level, g)
		}
		for _, word := range []string{"vocabulary", "grammar", "passage"} {
			if !strings.Contains(g, word) {
				t.Errorf("LevelGuidance(%q) = %q, missing %q", level, g, word)
			}
		}
	}
}

func TestLevelGuidanceUnknownLevelFallsBackToB1(t *testing.T) {
	if got, want := LevelGuidance("nope"), LevelGuidance("B1"); got != want {
		t.Errorf("LevelGuidance(unknown) = %q, want the documented B1 default %q", got, want)
	}
	if got, want := LevelGuidance(""), LevelGuidance("B1"); got != want {
		t.Errorf("LevelGuidance(\"\") = %q, want the documented B1 default %q", got, want)
	}
}
