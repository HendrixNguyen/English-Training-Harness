package pet

import (
	"context"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/quests"
)

// QuestHook adapts *Service to quests.Pet — the hook quests fires once per
// user per local day when the 30-minute target is crossed, and the state read
// that fills pet_health / streak_count on the §6.2 progress response.
// cmd/api/main.go registers it in place of quests.NopPet{}.
type QuestHook struct{ svc *Service }

// NewQuestHook wraps a Service.
func NewQuestHook(svc *Service) *QuestHook { return &QuestHook{svc: svc} }

func (h *QuestHook) OnTargetMet(ctx context.Context, userID, localDate string) error {
	return h.svc.OnTargetMet(ctx, userID, localDate)
}

func (h *QuestHook) State(ctx context.Context, userID string) (quests.PetState, error) {
	st, err := h.svc.Ensure(ctx, userID)
	if err != nil {
		return quests.PetState{}, err
	}
	return quests.PetState{Health: st.HealthPoints, Streak: st.CurrentStreak}, nil
}

var _ quests.Pet = (*QuestHook)(nil)
