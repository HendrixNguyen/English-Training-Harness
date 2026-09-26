package onboarding

import (
	"context"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
)

// AssessmentRequest is the backend spec §6.1 POST /onboarding/assessment body.
type AssessmentRequest struct {
	TargetGoal       string `json:"target_goal"`
	NotificationTime string `json:"notification_time"`
	Timezone         string `json:"timezone"`
	// PlantName is optional; trimmed, 1..MaxPlantNameRunes runes, blank → DefaultPlantName.
	PlantName string   `json:"plant_name"`
	Answers   []Answer `json:"answers"`
}

// PetState is the §6.1 pet_state object. onboarding defines its own type and
// interface so it never imports pet (see the plan's architecture note).
type PetState struct {
	PlantName    string `json:"plant_name"`
	HealthPoints int    `json:"health_points"`
	Stage        string `json:"stage"`
}

// Pet creates the pet_states row idempotently and reports it. plantName is
// the name to create the row with (or rename it to); "" creates the row if
// missing and never touches an existing name — the re-submit path passes "".
// Satisfied in cmd/api/main.go by an adapter over *pet.Service.EnsureNamed.
type Pet interface {
	Ensure(ctx context.Context, userID, plantName string) (PetState, error)
}

// Generator is the one airouter method onboarding needs; *airouter.Router
// satisfies it.
type Generator interface {
	Route(ctx context.Context, task airouter.TaskType, systemPrompt, userPrompt string) (string, error)
}

// AssessmentResult is the §6.1 response plus Created (201 vs 200), which
// never reaches the wire.
type AssessmentResult struct {
	Status        string   `json:"status"`
	AssessedLevel string   `json:"assessed_level"`
	RoadmapID     string   `json:"roadmap_id"`
	PetState      PetState `json:"pet_state"`
	Created       bool     `json:"-"`
}
