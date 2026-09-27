package onboarding

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// Gated on TEST_REDIS_URL, never the production REDIS_URL (spec §9). CI exports
// TEST_* and fails on --- SKIP; run with -p 1 (make test-integration).
func TestIntegrationQuizStoreStagesAndReusesTheGradedLevel(t *testing.T) {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL is unset; run `make up` and export it to run integration tests")
	}
	ctx := context.Background()
	rdb, err := store.NewRedis(ctx, url)
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	const user = "onboarding-integration-quiz-user"
	qs := NewRedisQuizStore(rdb)
	t.Cleanup(func() { _ = qs.Clear(ctx, user) })

	answers := []Answer{{QuestionID: "q1", SelectedOption: "B"}, {QuestionID: "q2", SelectedOption: "C"}}
	if err := qs.StageAnswers(ctx, user, answers, time.Minute); err != nil {
		t.Fatal(err)
	}
	if lvl, err := qs.StagedLevel(ctx, user, answers); err != nil || lvl != "" {
		t.Fatalf("before StageLevel: %q, %v; want \"\"", lvl, err)
	}
	if err := qs.StageLevel(ctx, user, "B1", time.Minute); err != nil {
		t.Fatal(err)
	}
	if lvl, err := qs.StagedLevel(ctx, user, answers); err != nil || lvl != "B1" {
		t.Fatalf("same answers: %q, %v; want B1", lvl, err)
	}
	changed := []Answer{{QuestionID: "q1", SelectedOption: "A"}, {QuestionID: "q2", SelectedOption: "C"}}
	if lvl, _ := qs.StagedLevel(ctx, user, changed); lvl != "" {
		t.Errorf("changed answers: %q, want \"\"", lvl)
	}
	if lvl, _ := qs.StagedLevel(ctx, user, answers[:1]); lvl != "" {
		t.Errorf("fewer answers: %q, want \"\"", lvl)
	}
	if ttl := rdb.Client.TTL(ctx, store.PlacementQuizKey(user)).Val(); ttl <= 0 || ttl > time.Minute {
		t.Errorf("TTL after StageLevel = %s, want (0, 1m]", ttl)
	}
	if err := qs.StageAnswers(ctx, user, changed, time.Minute); err != nil {
		t.Fatal(err)
	}
	if lvl, _ := qs.StagedLevel(ctx, user, changed); lvl != "" {
		t.Errorf("re-staging answers must drop the old level, got %q", lvl)
	}
	if err := qs.Clear(ctx, user); err != nil {
		t.Fatal(err)
	}
	if n := rdb.Client.Exists(ctx, store.PlacementQuizKey(user)).Val(); n != 0 {
		t.Errorf("key survives Clear")
	}
}
