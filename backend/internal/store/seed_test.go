package store

import "testing"

func TestDemoRoadmapShapeMatchesSpec61(t *testing.T) {
	if DemoRoadmapDays != 28 {
		t.Errorf("DemoRoadmapDays = %d, want 28 (4 modules x 7 days)", DemoRoadmapDays)
	}
	want := []string{"vocabulary", "reading", "practice"}
	if len(DemoTaskTypes) != len(want) {
		t.Fatalf("DemoTaskTypes = %v, want %v", DemoTaskTypes, want)
	}
	for i, w := range want {
		if DemoTaskTypes[i] != w {
			t.Errorf("DemoTaskTypes[%d] = %q, want %q", i, DemoTaskTypes[i], w)
		}
	}
}
