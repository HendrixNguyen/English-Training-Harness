// Package pet is the virtual-plant engine: health, streak and stage
// (1st-thinking §5.2 steps 4-5; backend spec §6.3 for the wire DTOs, §8 for
// the arithmetic). Success (+20, streak+1) is applied once, at progress time,
// through the quests hook; the hourly cron applies only the miss penalty.
package pet

import (
	"fmt"
	"time"
)

// Arithmetic from backend spec §8 and §6.3.
const (
	MaxHealth            = 100
	TargetMetHealthBonus = 20  // §8 success logic: Health = Min(100, Health + 20)
	MissPenalty          = 30  // §8 inactivity logic: Health = Max(0, Health - 30)
	ReviveHealth         = 50  // §6.3: "Resets health to 50% upon passing"
	ReviveSeconds        = 900 // §7: "15-minute revival challenge"

	MaxShields      = 2 // migration 0004 CHECK (shields BETWEEN 0 AND 2)
	ShieldEveryDays = 7 // a shield per 7th consecutive met day
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

// State is the pet_states row (§3.2 + migration 0003) minus its ids.
type State struct {
	PlantName       string
	HealthPoints    int
	Stage           string
	CurrentStreak   int
	LastPracticedAt *time.Time
	UpdatedAt       time.Time
	// LastTargetMetDate is the local YYYY-MM-DD whose §8 success was applied
	// last — the pet's own once-per-day guard. nil until the first target is met.
	LastTargetMetDate *string
	// JudgedThrough is the latest local YYYY-MM-DD that can no longer be
	// penalised: its miss was applied, it was spared, or a revival resolved it.
	// nil for a pet the sweep has never seen.
	JudgedThrough *string
	// Shields is how many streak shields the pet holds (0..MaxShields,
	// migration 0004). ApplyTargetMet awards one on every ShieldEveryDays-th
	// consecutive met day; ApplyMiss spends one instead of the §8 penalty.
	Shields int
	// LastShieldUsedOn is the local YYYY-MM-DD a shield was last spent for;
	// nil until the first spend. The client shows the spend for seven days.
	LastShieldUsedOn *string
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

// ApplyTargetMet is §8's success logic for localDate, applied once per local
// day (Repo.SaveTargetMet enforces the once): +20 capped at 100, streak+1,
// last_practiced_at = now, last_target_met_date = localDate. Every
// ShieldEveryDays-th consecutive met day also awards a streak shield, capped
// at MaxShields (migration 0004, streak shield plan).
func ApplyTargetMet(s State, now time.Time, localDate string) State {
	s.HealthPoints = min(MaxHealth, s.HealthPoints+TargetMetHealthBonus)
	s.CurrentStreak++
	if s.CurrentStreak%ShieldEveryDays == 0 {
		s.Shields = min(MaxShields, s.Shields+1)
	}
	s.Stage = StageFor(s.HealthPoints, s.CurrentStreak)
	t := now
	s.LastPracticedAt = &t
	s.LastTargetMetDate = &localDate
	s.UpdatedAt = now
	return s
}

// ApplyMiss is §8's inactivity logic for the local day judged, run by the
// hourly sweep when that day stayed under 1800s: -30 floored at 0, wilted at
// 0, streak reset (§8 is silent on the streak; a streak with a missed day in
// it is not a streak), and judged_through advanced to judged. When a streak
// shield is held (migration 0004), it is spent instead: health, streak and
// stage are untouched, and LastShieldUsedOn records the judged day. The real
// repo performs this arithmetic in SQL (PgRepo.PenaliseMiss); this Go form is
// the reference the fake repo and the integration test hold it to.
func ApplyMiss(s State, now time.Time, judged string) State {
	if s.Shields > 0 {
		// The shield takes the hit: health, streak and stage are untouched.
		s.Shields--
		s.LastShieldUsedOn = &judged
	} else {
		s.HealthPoints = max(0, s.HealthPoints-MissPenalty)
		s.CurrentStreak = 0
		s.Stage = StageFor(s.HealthPoints, s.CurrentStreak)
	}
	s.JudgedThrough = laterDate(s.JudgedThrough, judged)
	s.UpdatedAt = now
	return s
}

// ApplyRevive is the §6.3 pass: health 50, sprout, streak 0 — and the local
// day it was passed on is resolved (judged_through = localDate), so that
// night's sweep does not take the §8 penalty out of the 50 (plan decision 4).
// It is not a success: last_practiced_at and last_target_met_date are untouched,
// so reaching 1800s later the same day still earns the +20. Shields are untouched.
func ApplyRevive(s State, now time.Time, localDate string) State {
	s.HealthPoints = ReviveHealth
	s.CurrentStreak = 0
	s.Stage = StageFor(s.HealthPoints, s.CurrentStreak)
	s.JudgedThrough = laterDate(s.JudgedThrough, localDate)
	s.UpdatedAt = now
	return s
}

// laterDate keeps a verdict date monotonic. YYYY-MM-DD strings order lexically.
func laterDate(cur *string, d string) *string {
	if cur != nil && *cur > d {
		return cur
	}
	return &d
}

// PreviousDate is the civil day before a YYYY-MM-DD. It is pure calendar
// arithmetic — no timezone, no instant — so it is unaffected by a zone whose
// local midnight does not exist (spring-forward at 00:00) or exists twice.
func PreviousDate(date string) string {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		// Callers pass quests.LocalDate output; a bad value would judge the
		// wrong day silently, so fail loudly.
		panic(fmt.Sprintf("pet: PreviousDate got a malformed date %q", date))
	}
	return d.AddDate(0, 0, -1).Format("2006-01-02")
}
