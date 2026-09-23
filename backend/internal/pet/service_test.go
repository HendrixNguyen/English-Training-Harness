package pet

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/quests"
)

type harness struct {
	svc        *Service
	repo       *fakeRepo
	challenges *fakeChallenges
	study      *fakeStudy
}

func newHarness(now time.Time) *harness {
	h := &harness{repo: newFakeRepo(), challenges: newFakeChallenges(), study: newFakeStudy()}
	h.svc = NewService(h.repo, h.challenges, h.study, fixedClock(now))
	return h
}

var (
	ctx     = context.Background()
	sept22  = time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	midnite = time.Date(2026, time.September, 23, 0, 0, 0, 0, time.UTC) // 00:00 UTC = a :00 sweep
)

func TestEnsureCreatesTheRowOnceAndReturnsTheDefaults(t *testing.T) {
	h := newHarness(sept22)

	first, err := h.svc.Ensure(ctx, "u1")
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if first.PlantName != "My Green Buddy" || first.HealthPoints != 100 || first.Stage != StageSprout || first.CurrentStreak != 0 || first.LastPracticedAt != nil {
		t.Errorf("fresh pet = %+v, want the §3.2 defaults", first)
	}
	if _, err := h.svc.Ensure(ctx, "u1"); err != nil {
		t.Fatalf("second Ensure: %v", err)
	}
	if h.repo.ensured != 2 || len(h.repo.states) != 1 {
		t.Errorf("ensured %d times into %d rows, want 2 calls, 1 row", h.repo.ensured, len(h.repo.states))
	}
}

func TestOnTargetMetAppliesSpec8SuccessOnceAndPersists(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{PlantName: "Fern", HealthPoints: 80, CurrentStreak: 4, Stage: StageSapling}

	if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-22"); err != nil {
		t.Fatalf("OnTargetMet: %v", err)
	}
	got := h.repo.states["u1"]
	if got.HealthPoints != 100 || got.CurrentStreak != 5 || got.Stage != StageSapling {
		t.Errorf("state = %+v, want 100/5/sapling", got)
	}
	if got.LastPracticedAt == nil || !got.LastPracticedAt.Equal(sept22) {
		t.Errorf("LastPracticedAt = %v, want %v", got.LastPracticedAt, sept22)
	}
	if got.PlantName != "Fern" {
		t.Errorf("PlantName = %q, want untouched", got.PlantName)
	}
	if h.repo.saved != 1 {
		t.Errorf("saved %d times, want 1", h.repo.saved)
	}
}

func TestOnTargetMetForAUserWithoutARowCreatesIt(t *testing.T) {
	h := newHarness(sept22)
	if err := h.svc.OnTargetMet(ctx, "new", "2026-09-22"); err != nil {
		t.Fatalf("OnTargetMet: %v", err)
	}
	if got := h.repo.states["new"]; got.HealthPoints != 100 || got.CurrentStreak != 1 {
		t.Errorf("state = %+v, want the defaults bumped: 100 (capped) / streak 1", got)
	}
}

func TestQuestHookSatisfiesQuestsPet(t *testing.T) {
	var _ quests.Pet = (*QuestHook)(nil)

	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 40, CurrentStreak: 2}
	hook := NewQuestHook(h.svc)

	st, err := hook.State(ctx, "u1")
	if err != nil {
		t.Fatalf("State: %v", err)
	}
	if st != (quests.PetState{Health: 40, Streak: 2}) {
		t.Errorf("State = %+v, want {40 2}", st)
	}
	if err := hook.OnTargetMet(ctx, "u1", "2026-09-22"); err != nil {
		t.Fatalf("OnTargetMet: %v", err)
	}
	st, _ = hook.State(ctx, "u1")
	if st != (quests.PetState{Health: 60, Streak: 3}) {
		t.Errorf("State after hook = %+v, want {60 3}", st)
	}
}

func TestReviveIsRejectedWhileHealthIsAboveZero(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 10, Stage: StageSprout}

	if _, err := h.svc.Revive(ctx, "u1"); !errors.Is(err, ErrNotWilted) {
		t.Fatalf("err = %v, want ErrNotWilted", err)
	}
	if h.challenges.started != 0 {
		t.Error("a rejected revive started a challenge")
	}
}

func TestReviveStartsAChallengeAtTheCurrentCounterValue(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}
	h.study.set("u1", "2026-09-22", 300) // already studied 5 min today

	out, err := h.svc.Revive(ctx, "u1")
	if err != nil {
		t.Fatalf("Revive: %v", err)
	}
	if out.Passed {
		t.Error("Passed = true on the call that starts the challenge")
	}
	if out.State.HealthPoints != 0 || out.State.Stage != StageWilted {
		t.Errorf("state = %+v, want still wilted", out.State)
	}
	c, ok := h.challenges.items["u1"]
	if !ok || c.LocalDate != "2026-09-22" || c.StartSeconds != 300 || !c.StartedAt.Equal(sept22) {
		t.Errorf("challenge = %+v ok=%t, want {2026-09-22, 300, %v}", c, ok, sept22)
	}
}

func TestReviveUsesTheUsersTimezoneForTheLocalDate(t *testing.T) {
	// 18:30Z is already the 23rd in Ho Chi Minh City.
	h := newHarness(time.Date(2026, time.September, 22, 18, 30, 0, 0, time.UTC))
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}
	h.repo.timezones["u1"] = "Asia/Ho_Chi_Minh"

	if _, err := h.svc.Revive(ctx, "u1"); err != nil {
		t.Fatalf("Revive: %v", err)
	}
	if c := h.challenges.items["u1"]; c.LocalDate != "2026-09-23" {
		t.Errorf("LocalDate = %q, want 2026-09-23", c.LocalDate)
	}
}

func TestRevivePassesOnceNineHundredSecondsWereStudiedSinceTheStart(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted, CurrentStreak: 0}
	h.study.set("u1", "2026-09-22", 300)
	if _, err := h.svc.Revive(ctx, "u1"); err != nil {
		t.Fatalf("start: %v", err)
	}

	h.study.set("u1", "2026-09-22", 300+899)
	out, err := h.svc.Revive(ctx, "u1")
	if err != nil {
		t.Fatalf("check at 899s: %v", err)
	}
	if out.Passed || out.State.HealthPoints != 0 {
		t.Errorf("out = %+v at 899s, want not passed", out)
	}
	if h.challenges.started != 1 {
		t.Errorf("challenge restarted (%d starts) while still active on the same day", h.challenges.started)
	}

	h.study.set("u1", "2026-09-22", 300+900)
	out, err = h.svc.Revive(ctx, "u1")
	if err != nil {
		t.Fatalf("check at 900s: %v", err)
	}
	if !out.Passed || out.State.HealthPoints != 50 || out.State.Stage != StageSprout || out.State.CurrentStreak != 0 {
		t.Errorf("out = %+v at 900s, want passed with 50/sprout/0 (§6.3)", out)
	}
	if h.challenges.cleared != 1 || len(h.challenges.items) != 0 {
		t.Error("passing did not clear the challenge")
	}
	if got := h.repo.states["u1"]; got.HealthPoints != 50 {
		t.Errorf("persisted health = %d, want 50", got.HealthPoints)
	}
}

func TestReviveOnANewLocalDayRestartsTheChallenge(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}
	h.challenges.items["u1"] = Challenge{StartedAt: sept22.Add(-20 * time.Hour), LocalDate: "2026-09-21", StartSeconds: 0}
	h.study.set("u1", "2026-09-21", 5000) // yesterday's total is irrelevant now
	h.study.set("u1", "2026-09-22", 120)

	out, err := h.svc.Revive(ctx, "u1")
	if err != nil {
		t.Fatalf("Revive: %v", err)
	}
	if out.Passed {
		t.Error("a stale challenge from yesterday must not pass on today's call")
	}
	if c := h.challenges.items["u1"]; c.LocalDate != "2026-09-22" || c.StartSeconds != 120 {
		t.Errorf("challenge = %+v, want restarted for today at 120s", c)
	}
}

func TestSweepPenalisesOnlyUsersAtLocalMidnightWhoMissedYesterday(t *testing.T) {
	h := newHarness(midnite) // 00:00 UTC: UTC users just hit midnight; Ho Chi Minh (UTC+7) is at 07:00
	h.repo.states["utc-missed"] = State{HealthPoints: 100, CurrentStreak: 9, Stage: StageFlowering, UpdatedAt: midnite.Add(-30 * time.Hour)}
	h.repo.states["utc-met"] = State{HealthPoints: 100, CurrentStreak: 9, Stage: StageFlowering, UpdatedAt: midnite.Add(-3 * time.Hour)}
	h.repo.states["utc-short"] = State{HealthPoints: 40, CurrentStreak: 1, Stage: StageSprout, UpdatedAt: midnite.Add(-30 * time.Hour)}
	h.repo.states["hcm-missed"] = State{HealthPoints: 100, CurrentStreak: 9, Stage: StageFlowering, UpdatedAt: midnite.Add(-30 * time.Hour)}
	h.repo.timezones["hcm-missed"] = "Asia/Ho_Chi_Minh"
	h.study.set("utc-met", "2026-09-22", 1800)
	h.study.set("utc-short", "2026-09-22", 1799)

	n, err := h.svc.Sweep(ctx, midnite)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if n != 2 {
		t.Errorf("penalised %d pets, want 2 (utc-missed, utc-short)", n)
	}
	if got := h.repo.states["utc-missed"]; got.HealthPoints != 70 || got.CurrentStreak != 0 || got.Stage != StageSprout || !got.UpdatedAt.Equal(midnite) {
		t.Errorf("utc-missed = %+v, want 70/0/sprout stamped %v", got, midnite)
	}
	if got := h.repo.states["utc-short"]; got.HealthPoints != 10 || got.Stage != StageSprout {
		t.Errorf("utc-short = %+v, want 10/sprout (1799s is a miss)", got)
	}
	if got := h.repo.states["utc-met"]; got.HealthPoints != 100 || got.CurrentStreak != 9 {
		t.Errorf("utc-met = %+v, want untouched", got)
	}
	if got := h.repo.states["hcm-missed"]; got.HealthPoints != 100 {
		t.Errorf("hcm-missed = %+v, want untouched — it is 07:00 in Ho Chi Minh City", got)
	}
}

func TestSweepAtSeventeenUTCCatchesHoChiMinhMidnight(t *testing.T) {
	at := time.Date(2026, time.September, 22, 17, 0, 0, 0, time.UTC) // 00:00 on the 23rd in UTC+7
	h := newHarness(at)
	h.repo.states["hcm"] = State{HealthPoints: 100, UpdatedAt: at.Add(-30 * time.Hour)}
	h.repo.timezones["hcm"] = "Asia/Ho_Chi_Minh"
	h.repo.states["utc"] = State{HealthPoints: 100, UpdatedAt: at.Add(-30 * time.Hour)}

	if _, err := h.svc.Sweep(ctx, at); err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if got := h.repo.states["hcm"]; got.HealthPoints != 70 {
		t.Errorf("hcm = %+v, want 70 — local day 2026-09-22 ended with no study", got)
	}
	if got := h.repo.states["utc"]; got.HealthPoints != 100 {
		t.Errorf("utc = %+v, want untouched at 17:00 UTC", got)
	}
}

func TestSweepIsIdempotentWithinTheSameLocalDay(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-30 * time.Hour)}

	if _, err := h.svc.Sweep(ctx, midnite); err != nil {
		t.Fatalf("first: %v", err)
	}
	n, err := h.svc.Sweep(ctx, midnite.Add(20*time.Minute)) // a restart re-running the hour
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if n != 0 || h.repo.states["u1"].HealthPoints != 70 {
		t.Errorf("second sweep penalised %d, health = %d; want 0 and 70 — updated_at is already past local midnight", n, h.repo.states["u1"].HealthPoints)
	}
}

func TestSweepWiltsAfterFourMissedDays(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, CurrentStreak: 20, Stage: StageFruitful, UpdatedAt: midnite.Add(-30 * time.Hour)}

	for day := 0; day < 4; day++ {
		at := midnite.AddDate(0, 0, day)
		h.svc.now = fixedClock(at)
		if _, err := h.svc.Sweep(ctx, at); err != nil {
			t.Fatalf("day %d: %v", day, err)
		}
	}
	if got := h.repo.states["u1"]; got.HealthPoints != 0 || got.Stage != StageWilted || got.CurrentStreak != 0 {
		t.Errorf("after four misses = %+v, want 0/wilted/0", got)
	}
}

func TestSweepSkipsAUnreadableCounterAndContinues(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-30 * time.Hour)}
	h.study.err = errBoom

	n, err := h.svc.Sweep(ctx, midnite)
	if err == nil {
		t.Fatal("Sweep returned nil with an unreadable counter; want the error surfaced")
	}
	if n != 0 || h.repo.states["u1"].HealthPoints != 100 {
		t.Errorf("a pet was penalised on an unreadable counter: n=%d health=%d", n, h.repo.states["u1"].HealthPoints)
	}
}
