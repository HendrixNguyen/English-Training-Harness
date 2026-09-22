package pet

import (
	"testing"
	"time"
)

func TestStageForIsWiltedAtZeroHealthOtherwiseByStreak(t *testing.T) {
	tests := []struct {
		health, streak int
		want           string
	}{
		{0, 0, StageWilted},
		{0, 30, StageWilted}, // health wins over streak
		{100, 0, StageSprout},
		{100, 2, StageSprout},
		{100, 3, StageSapling},
		{100, 6, StageSapling},
		{100, 7, StageFlowering},
		{100, 13, StageFlowering},
		{100, 14, StageFruitful},
		{10, 100, StageFruitful},
	}
	for _, tt := range tests {
		if got := StageFor(tt.health, tt.streak); got != tt.want {
			t.Errorf("StageFor(%d, %d) = %q, want %q", tt.health, tt.streak, got, tt.want)
		}
	}
}

func TestApplyTargetMetAddsTwentyCapsAtHundredAndStampsPractice(t *testing.T) {
	now := time.Date(2026, time.September, 22, 20, 15, 0, 0, time.UTC)

	s := ApplyTargetMet(State{HealthPoints: 80, CurrentStreak: 4, Stage: StageSapling}, now)
	if s.HealthPoints != 100 || s.CurrentStreak != 5 {
		t.Errorf("state = %+v, want health 100 streak 5", s)
	}
	if s.LastPracticedAt == nil || !s.LastPracticedAt.Equal(now) || !s.UpdatedAt.Equal(now) {
		t.Errorf("timestamps = %v / %v, want both %v", s.LastPracticedAt, s.UpdatedAt, now)
	}

	capped := ApplyTargetMet(State{HealthPoints: 95, CurrentStreak: 13}, now)
	if capped.HealthPoints != 100 {
		t.Errorf("health = %d, want capped at 100", capped.HealthPoints)
	}
	if capped.Stage != StageFruitful {
		t.Errorf("stage = %q, want fruitful at streak 14", capped.Stage)
	}

	revived := ApplyTargetMet(State{HealthPoints: 0, Stage: StageWilted}, now)
	if revived.HealthPoints != 20 || revived.Stage != StageSprout || revived.CurrentStreak != 1 {
		t.Errorf("a wilted plant that meets the target = %+v, want 20/sprout/1", revived)
	}
}

func TestApplyMissSubtractsThirtyFloorsAtZeroAndWilts(t *testing.T) {
	now := time.Date(2026, time.September, 23, 0, 0, 0, 0, time.UTC)

	s := State{HealthPoints: 100, CurrentStreak: 9, Stage: StageFlowering}
	for i, want := range []int{70, 40, 10, 0} {
		s = ApplyMiss(s, now)
		if s.HealthPoints != want {
			t.Fatalf("miss %d: health = %d, want %d", i+1, s.HealthPoints, want)
		}
		if s.CurrentStreak != 0 {
			t.Errorf("miss %d: streak = %d, want 0", i+1, s.CurrentStreak)
		}
	}
	if s.Stage != StageWilted {
		t.Errorf("stage after four misses = %q, want wilted", s.Stage)
	}
	if !s.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", s.UpdatedAt, now)
	}

	again := ApplyMiss(s, now)
	if again.HealthPoints != 0 {
		t.Errorf("health below zero: %d", again.HealthPoints)
	}
}

func TestApplyReviveResetsToFiftySproutZeroStreak(t *testing.T) {
	now := time.Date(2026, time.September, 23, 9, 0, 0, 0, time.UTC)
	s := ApplyRevive(State{HealthPoints: 0, Stage: StageWilted, CurrentStreak: 0}, now)
	if s.HealthPoints != ReviveHealth || s.Stage != StageSprout || s.CurrentStreak != 0 {
		t.Errorf("revived = %+v, want 50/sprout/0 (backend spec §6.3)", s)
	}
	if !s.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", s.UpdatedAt, now)
	}
}

func TestSpec8Constants(t *testing.T) {
	if TargetMetHealthBonus != 20 || MissPenalty != 30 || MaxHealth != 100 || ReviveSeconds != 900 {
		t.Errorf("constants = (+%d, -%d, max %d, revive %ds), want (+20, -30, 100, 900)", TargetMetHealthBonus, MissPenalty, MaxHealth, ReviveSeconds)
	}
}
