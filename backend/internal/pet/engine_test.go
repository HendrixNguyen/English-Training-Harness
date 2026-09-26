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

	s := ApplyTargetMet(State{HealthPoints: 80, CurrentStreak: 4, Stage: StageSapling}, now, "2026-09-22")
	if s.HealthPoints != 100 || s.CurrentStreak != 5 {
		t.Errorf("state = %+v, want health 100 streak 5", s)
	}
	if s.LastPracticedAt == nil || !s.LastPracticedAt.Equal(now) || !s.UpdatedAt.Equal(now) {
		t.Errorf("timestamps = %v / %v, want both %v", s.LastPracticedAt, s.UpdatedAt, now)
	}

	capped := ApplyTargetMet(State{HealthPoints: 95, CurrentStreak: 13}, now, "2026-09-22")
	if capped.HealthPoints != 100 {
		t.Errorf("health = %d, want capped at 100", capped.HealthPoints)
	}
	if capped.Stage != StageFruitful {
		t.Errorf("stage = %q, want fruitful at streak 14", capped.Stage)
	}

	revived := ApplyTargetMet(State{HealthPoints: 0, Stage: StageWilted}, now, "2026-09-22")
	if revived.HealthPoints != 20 || revived.Stage != StageSprout || revived.CurrentStreak != 1 {
		t.Errorf("a wilted plant that meets the target = %+v, want 20/sprout/1", revived)
	}
}

func TestApplyMissSubtractsThirtyFloorsAtZeroAndWilts(t *testing.T) {
	now := time.Date(2026, time.September, 23, 0, 0, 0, 0, time.UTC)

	s := State{HealthPoints: 100, CurrentStreak: 9, Stage: StageFlowering}
	for i, want := range []int{70, 40, 10, 0} {
		s = ApplyMiss(s, now, "2026-09-22")
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

	again := ApplyMiss(s, now, "2026-09-22")
	if again.HealthPoints != 0 {
		t.Errorf("health below zero: %d", again.HealthPoints)
	}
}

func TestApplyReviveResetsToFiftySproutZeroStreak(t *testing.T) {
	now := time.Date(2026, time.September, 23, 9, 0, 0, 0, time.UTC)
	s := ApplyRevive(State{HealthPoints: 0, Stage: StageWilted, CurrentStreak: 0}, now, "2026-09-22")
	if s.HealthPoints != ReviveHealth || s.Stage != StageSprout || s.CurrentStreak != 0 {
		t.Errorf("revived = %+v, want 50/sprout/0 (backend spec §6.3)", s)
	}
	if !s.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", s.UpdatedAt, now)
	}
	if r := ApplyRevive(State{HealthPoints: 0, Stage: StageWilted, Shields: 1}, now, "2026-09-22"); r.Shields != 1 {
		t.Error("revive must not touch shields")
	}
}

func TestApplyTargetMetAwardsAShieldEverySeventhDayCappedAtTwo(t *testing.T) {
	for _, tt := range []struct{ streak, shields, wantShields int }{
		{6, 0, 1},  // day 7
		{7, 1, 1},  // day 8: nothing
		{13, 1, 2}, // day 14
		{20, 2, 2}, // day 21 with a full rack: capped
		{13, 0, 1}, // day 14 after a spend
	} {
		got := ApplyTargetMet(State{HealthPoints: 100, CurrentStreak: tt.streak, Shields: tt.shields}, sept22, "2026-09-22")
		if got.Shields != tt.wantShields || got.CurrentStreak != tt.streak+1 {
			t.Errorf("streak %d, shields %d → shields %d (streak %d), want %d", tt.streak, tt.shields, got.Shields, got.CurrentStreak, tt.wantShields)
		}
		if got.LastShieldUsedOn != nil {
			t.Error("a success must not touch LastShieldUsedOn")
		}
	}
}

func TestApplyMissSpendsAShieldBeforeThePenalty(t *testing.T) {
	pre := State{HealthPoints: 100, CurrentStreak: 9, Stage: StageFlowering, Shields: 1}
	got := ApplyMiss(pre, sept22, "2026-09-22")
	if got.HealthPoints != 100 || got.CurrentStreak != 9 || got.Stage != StageFlowering {
		t.Errorf("shielded miss changed the plant: %+v", got)
	}
	if got.Shields != 0 || got.LastShieldUsedOn == nil || *got.LastShieldUsedOn != "2026-09-22" {
		t.Errorf("shields/last used = %d/%v, want 0/2026-09-22", got.Shields, got.LastShieldUsedOn)
	}
	if got.JudgedThrough == nil || *got.JudgedThrough != "2026-09-22" || !got.UpdatedAt.Equal(sept22) {
		t.Errorf("a shielded miss must still resolve the day and stamp the clock: %+v", got)
	}
	// No shield left: the ordinary §8 penalty, and the spend date is kept.
	again := ApplyMiss(got, sept22, "2026-09-23")
	if again.HealthPoints != 70 || again.CurrentStreak != 0 || again.Stage != StageSprout || again.Shields != 0 || *again.LastShieldUsedOn != "2026-09-22" {
		t.Errorf("unshielded miss = %+v, want 70/0/sprout, shields 0, last used kept", again)
	}
}

func TestSpec8Constants(t *testing.T) {
	if TargetMetHealthBonus != 20 || MissPenalty != 30 || MaxHealth != 100 || ReviveSeconds != 900 {
		t.Errorf("constants = (+%d, -%d, max %d, revive %ds), want (+20, -30, 100, 900)", TargetMetHealthBonus, MissPenalty, MaxHealth, ReviveSeconds)
	}
}

func TestApplyTargetMetStampsTheLocalDateItWasAppliedFor(t *testing.T) {
	got := ApplyTargetMet(State{HealthPoints: 80, CurrentStreak: 4}, sept22, "2026-09-22")
	if got.LastTargetMetDate == nil || *got.LastTargetMetDate != "2026-09-22" {
		t.Errorf("LastTargetMetDate = %v, want 2026-09-22", got.LastTargetMetDate)
	}
	if got.JudgedThrough != nil {
		t.Errorf("JudgedThrough = %v, want untouched (success does not judge the day)", *got.JudgedThrough)
	}
}

func TestApplyMissRecordsTheJudgedDayAndNeverMovesItBackwards(t *testing.T) {
	later := "2026-09-25"
	got := ApplyMiss(State{HealthPoints: 100, JudgedThrough: &later}, sept22, "2026-09-22")
	if got.HealthPoints != 70 || *got.JudgedThrough != "2026-09-25" {
		t.Errorf("got %d / %s, want 70 and judged_through kept at 2026-09-25", got.HealthPoints, *got.JudgedThrough)
	}
	got = ApplyMiss(State{HealthPoints: 100}, sept22, "2026-09-22")
	if got.JudgedThrough == nil || *got.JudgedThrough != "2026-09-22" {
		t.Errorf("JudgedThrough = %v, want 2026-09-22", got.JudgedThrough)
	}
}

func TestApplyReviveResolvesTheDayWithoutCountingItAsASuccess(t *testing.T) {
	got := ApplyRevive(State{HealthPoints: 0, Stage: StageWilted}, sept22, "2026-09-22")
	if got.HealthPoints != 50 || got.Stage != StageSprout || got.CurrentStreak != 0 {
		t.Errorf("got %+v, want 50/sprout/0", got)
	}
	if got.JudgedThrough == nil || *got.JudgedThrough != "2026-09-22" {
		t.Errorf("JudgedThrough = %v, want 2026-09-22 — a passed revival resolves its day", got.JudgedThrough)
	}
	if got.LastTargetMetDate != nil || got.LastPracticedAt != nil {
		t.Error("a revival must not look like a met target: LastTargetMetDate and LastPracticedAt stay nil")
	}
}

func TestPreviousDateIsCivilArithmetic(t *testing.T) {
	for in, want := range map[string]string{
		"2026-09-23": "2026-09-22",
		"2026-09-01": "2026-08-31",
		"2026-03-01": "2026-02-28",
		"2028-03-01": "2028-02-29",
		"2027-01-01": "2026-12-31",
	} {
		if got := PreviousDate(in); got != want {
			t.Errorf("PreviousDate(%s) = %s, want %s", in, got, want)
		}
	}
}
