package quests

import "context"

// PetState is the slice of pet_states (§3.2) that the §6.2 progress response
// reports back as pet_health / streak_count.
type PetState struct {
	Health int // pet_states.health_points
	Streak int // pet_states.current_streak
}

// Pet is quests' view of the pet slice. Packages talk via interfaces, never
// each other's tables (CODEMAP), so quests never reads pet_states itself.
//
// OnTargetMet is the in-process hook fired the first time a user crosses the
// 30-minute target on a given local day — the trigger backend spec §6.2
// describes for POST /quests/progress ("increases plant health (+20%), and
// increments streak"; the arithmetic is §8's success logic). It is called
// AFTER the daily_progress upsert, and its error is logged rather than
// returned: a failing pet update must never roll back a recorded study session.
//
// State is read after the hook so the response carries the post-bump values.
type Pet interface {
	OnTargetMet(ctx context.Context, userID, localDate string) error
	State(ctx context.Context, userID string) (PetState, error)
}

// NopPet is the default until the pet slice registers the real implementation.
// It reports the §3.2 pet_states column defaults (health_points 100,
// current_streak 0) — the state a freshly onboarded pet has — so the §6.2
// response shape is complete from day one. See the plan's Reconciliation notes.
type NopPet struct{}

func (NopPet) OnTargetMet(context.Context, string, string) error { return nil }

func (NopPet) State(context.Context, string) (PetState, error) {
	return PetState{Health: 100, Streak: 0}, nil
}

var _ Pet = NopPet{}
