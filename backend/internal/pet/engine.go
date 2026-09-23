// Package pet is the virtual-plant engine: health, streak and stage
// (1st-thinking §5.2 steps 4-5; backend spec §6.3 for the wire DTOs, §8 for
// the arithmetic). Success (+20, streak+1) is applied once, at progress time,
// through the quests hook; the hourly cron applies only the miss penalty.
package pet

import "time"

// Arithmetic from backend spec §8 and §6.3.
const (
	MaxHealth            = 100
	TargetMetHealthBonus = 20  // §8 success logic: Health = Min(100, Health + 20)
	MissPenalty          = 30  // §8 inactivity logic: Health = Max(0, Health - 30)
	ReviveHealth         = 50  // §6.3: "Resets health to 50% upon passing"
	ReviveSeconds        = 900 // §7: "15-minute revival challenge"
)

// Stages are the §3.2 pet_stage enum values. StageSeed is never produced —
// the DDL default is 'sprout' (see the reconcile-pet-states-stage bug).
const (
	StageSeed      = "seed"
	StageSprout    = "sprout"
	StageSapling   = "sapling"
	StageFlowering = "flowering"
	StageFruitful  = "fruitful"
	StageWilted    = "wilted"
)

// State is the pet_states row (§3.2) minus its ids.
type State struct {
	PlantName       string
	HealthPoints    int
	Stage           string
	CurrentStreak   int
	LastPracticedAt *time.Time
	UpdatedAt       time.Time
}

// StageFor derives the stage from health and streak. The spec gives no
// thresholds; these are the pet slice's decision, recorded in CODEMAP:
// wilted at 0 health, otherwise sprout (0-2), sapling (3-6), flowering (7-13),
// fruitful (14+) by streak.
func StageFor(health, streak int) string {
	switch {
	case health <= 0:
		return StageWilted
	case streak >= 14:
		return StageFruitful
	case streak >= 7:
		return StageFlowering
	case streak >= 3:
		return StageSapling
	default:
		return StageSprout
	}
}

// ApplyTargetMet is §8's success logic, run once per local day from the
// quests hook: +20 capped at 100, streak+1, last_practiced_at = now.
func ApplyTargetMet(s State, now time.Time) State {
	s.HealthPoints = min(MaxHealth, s.HealthPoints+TargetMetHealthBonus)
	s.CurrentStreak++
	s.Stage = StageFor(s.HealthPoints, s.CurrentStreak)
	t := now
	s.LastPracticedAt = &t
	s.UpdatedAt = now
	return s
}

// ApplyMiss is §8's inactivity logic, run by the hourly cron for a user whose
// previous local day stayed under 1800s: -30 floored at 0, wilted at 0. §8 is
// silent on the streak; a streak with a missed day in it is not a streak, so
// it resets.
func ApplyMiss(s State, now time.Time) State {
	s.HealthPoints = max(0, s.HealthPoints-MissPenalty)
	s.CurrentStreak = 0
	s.Stage = StageFor(s.HealthPoints, s.CurrentStreak)
	s.UpdatedAt = now
	return s
}

// ApplyRevive is the §6.3 pass: health 50, sprout, streak 0.
func ApplyRevive(s State, now time.Time) State {
	s.HealthPoints = ReviveHealth
	s.CurrentStreak = 0
	s.Stage = StageFor(s.HealthPoints, s.CurrentStreak)
	s.UpdatedAt = now
	return s
}
