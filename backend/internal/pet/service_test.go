package pet

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	h := &harness{repo: newFakeRepo(fixedClock(now)), challenges: newFakeChallenges(), study: newFakeStudy()}
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
	if got.LastTargetMetDate == nil || *got.LastTargetMetDate != "2026-09-22" {
		t.Errorf("LastTargetMetDate = %v, want 2026-09-22", got.LastTargetMetDate)
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

func TestOnTargetMetTwiceForTheSameLocalDateBumpsOnce(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{PlantName: "Fern", HealthPoints: 80, CurrentStreak: 4, Stage: StageSapling}

	for i := 0; i < 2; i++ {
		if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-22"); err != nil {
			t.Fatalf("call %d: %v — a repeat must be a silent no-op, not an error", i+1, err)
		}
	}
	got := h.repo.states["u1"]
	if got.HealthPoints != 100 || got.CurrentStreak != 5 {
		t.Errorf("state = %+v, want 100/5 — the second call for the same day double-bumped", got)
	}
	if h.repo.saved != 1 {
		t.Errorf("saved %d times, want 1", h.repo.saved)
	}
}

func TestOnTargetMetForTheNextLocalDateBumpsAgain(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 40, CurrentStreak: 4, Stage: StageSapling}

	if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-22"); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-23"); err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["u1"]; got.HealthPoints != 80 || got.CurrentStreak != 6 || *got.LastTargetMetDate != "2026-09-23" {
		t.Errorf("state = %+v, want 80/6 marked 2026-09-23 — the guard must not simply never bump", got)
	}
}

func TestOnTargetMetForAnEarlierLocalDateIsIgnored(t *testing.T) {
	// A user who moves their timezone west can make "today" an earlier date
	// than the one already counted; the marker is monotonic, so no re-earn.
	h := newHarness(sept22)
	d := "2026-09-23"
	h.repo.states["u1"] = State{HealthPoints: 40, CurrentStreak: 1, Stage: StageSprout, LastTargetMetDate: &d}

	if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-22"); err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["u1"]; got.HealthPoints != 40 || h.repo.saved != 0 {
		t.Errorf("state = %+v saved=%d, want untouched", got, h.repo.saved)
	}
}

func TestRevivePassResolvesTheLocalDayItWasPassedOn(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}
	h.study.set("u1", "2026-09-22", 0)
	if _, err := h.svc.Revive(ctx, "u1"); err != nil {
		t.Fatalf("start: %v", err)
	}
	h.study.set("u1", "2026-09-22", 900)
	out, err := h.svc.Revive(ctx, "u1")
	if err != nil || !out.Passed {
		t.Fatalf("pass: out=%+v err=%v", out, err)
	}
	got := h.repo.states["u1"]
	if got.JudgedThrough == nil || *got.JudgedThrough != "2026-09-22" {
		t.Errorf("JudgedThrough = %v, want 2026-09-22 (plan decision 4: a passed revival resolves its day)", got.JudgedThrough)
	}
	if got.LastTargetMetDate != nil || got.LastPracticedAt != nil {
		t.Error("a revival is not a met target")
	}
	// The +20 for a full 30 minutes the same day is still available.
	if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-22"); err != nil {
		t.Fatal(err)
	}
	if got = h.repo.states["u1"]; got.HealthPoints != 70 || got.CurrentStreak != 1 {
		t.Errorf("after revive then target met = %+v, want 70/1", got)
	}
}

func TestReviveSurfacesATimezoneReadFailure(t *testing.T) {
	h := newHarness(sept22)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted}
	h.repo.timezoneErr = errBoom
	if _, err := h.svc.Revive(ctx, "u1"); !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want errBoom", err)
	}
	if h.challenges.started != 0 {
		t.Error("a challenge was started without knowing the user's local day")
	}
}

// judgedThrough seeds a pet the sweep has already seen, so tests are not
// exercising the first-contact rule unless they mean to.
func judgedThrough(d string) *string { return &d }

func TestSweepJudgesEachPetsOwnLocalYesterday(t *testing.T) {
	h := newHarness(midnite) // 00:00Z on the 23rd: UTC just ended the 22nd; Ho Chi Minh (UTC+7) ended it seven hours ago
	seed := func(user, tz string, health, streak int, through string) {
		h.repo.states[user] = State{HealthPoints: health, CurrentStreak: streak, Stage: StageFor(health, streak), UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough(through)}
		h.repo.timezones[user] = tz
	}
	seed("utc-missed", "UTC", 100, 9, "2026-09-21")
	seed("utc-met-marker", "UTC", 100, 9, "2026-09-21")
	seed("utc-met-counter", "UTC", 100, 9, "2026-09-21")
	seed("utc-short", "UTC", 40, 1, "2026-09-21")
	seed("hcm-done", "Asia/Ho_Chi_Minh", 100, 9, "2026-09-22") // the 17:00Z tick already judged its 22nd
	seed("hcm-late", "Asia/Ho_Chi_Minh", 100, 9, "2026-09-21") // that tick was missed (a restart): catch up now
	d := "2026-09-22"
	st := h.repo.states["utc-met-marker"]
	st.LastTargetMetDate = &d
	h.repo.states["utc-met-marker"] = st
	h.study.set("utc-met-counter", "2026-09-22", 1800)
	h.study.set("utc-short", "2026-09-22", 1799)

	n, err := h.svc.Sweep(ctx, midnite)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if n != 3 {
		t.Errorf("penalised %d, want 3 (utc-missed, utc-short, hcm-late)", n)
	}
	want := map[string]struct {
		health, streak int
		through        string
	}{
		"utc-missed":      {70, 0, "2026-09-22"},
		"utc-met-marker":  {100, 9, "2026-09-22"}, // spared by its own marker, no counter needed
		"utc-met-counter": {100, 9, "2026-09-22"}, // spared by the counter fallback, day recorded
		"utc-short":       {10, 0, "2026-09-22"},  // 1799s is a miss
		"hcm-done":        {100, 9, "2026-09-22"}, // nothing to do
		"hcm-late":        {70, 0, "2026-09-22"},  // judged at 07:00 local, one tick is never lost
	}
	for user, w := range want {
		got := h.repo.states[user]
		if got.HealthPoints != w.health || got.CurrentStreak != w.streak || got.JudgedThrough == nil || *got.JudgedThrough != w.through {
			t.Errorf("%s = %+v, want %d/%d judged through %s", user, got, w.health, w.streak, w.through)
		}
	}
	if !h.repo.states["utc-missed"].UpdatedAt.Equal(midnite) {
		t.Error("a penalised pet must be stamped with the sweep's clock")
	}
}

func TestSweepIsIdempotentAcrossTicksAndHours(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}

	total := 0
	for _, at := range []time.Time{midnite, midnite.Add(20 * time.Minute), midnite.Add(time.Hour), midnite.Add(13 * time.Hour)} {
		n, err := h.svc.Sweep(ctx, at)
		if err != nil {
			t.Fatalf("sweep at %v: %v", at, err)
		}
		total += n
	}
	if total != 1 || h.repo.states["u1"].HealthPoints != 70 {
		t.Errorf("four sweeps in one local day penalised %d times, health %d; want 1 and 70", total, h.repo.states["u1"].HealthPoints)
	}
}

func TestTwoConcurrentSweepsPenaliseOnce(t *testing.T) {
	h := newHarness(midnite)
	for i := 0; i < 20; i++ {
		h.repo.states[fmt.Sprintf("u%02d", i)] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	}
	// Neither sweep may write until both have read: with identical stale
	// candidate lists, only the conditional write can keep the count at 20.
	var ready sync.WaitGroup
	ready.Add(2)
	h.repo.afterCandidates = func() { ready.Done(); ready.Wait() }

	var wg sync.WaitGroup
	counts := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := h.svc.Sweep(ctx, midnite)
			if err != nil {
				t.Errorf("Sweep: %v", err)
			}
			counts <- n
		}()
	}
	wg.Wait()
	close(counts)
	sum := 0
	for n := range counts {
		sum += n
	}
	if sum != 20 {
		t.Errorf("two concurrent sweeps reported %d penalties over 20 pets, want 20 — the count must follow the conditional write", sum)
	}
	for user, st := range h.repo.states {
		if st.HealthPoints != 70 {
			t.Errorf("%s health = %d, want 70 (penalised exactly once)", user, st.HealthPoints)
		}
	}
}

func TestSweepWiltsAfterFourMissedDays(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, CurrentStreak: 20, Stage: StageFruitful, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}

	for day := 0; day < 4; day++ {
		if _, err := h.svc.Sweep(ctx, midnite.AddDate(0, 0, day)); err != nil {
			t.Fatalf("day %d: %v", day, err)
		}
	}
	if got := h.repo.states["u1"]; got.HealthPoints != 0 || got.Stage != StageWilted || got.CurrentStreak != 0 || *got.JudgedThrough != "2026-09-25" {
		t.Errorf("after four misses = %+v, want 0/wilted/0 judged through 2026-09-25", got)
	}
}

func TestSweepDoesNotPenaliseADayResolvedByARevive(t *testing.T) {
	eight := time.Date(2026, time.September, 22, 20, 0, 0, 0, time.UTC) // 20:00 local (UTC) on the 22nd
	h := newHarness(eight)
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted, UpdatedAt: eight.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	if _, err := h.svc.Revive(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	h.study.set("u1", "2026-09-22", 900) // the challenge, and nothing more: 900 < 1800
	if out, err := h.svc.Revive(ctx, "u1"); err != nil || !out.Passed {
		t.Fatalf("pass: %+v %v", out, err)
	}

	n, err := h.svc.Sweep(ctx, midnite) // 00:00 on the 23rd
	if err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["u1"]; n != 0 || got.HealthPoints != 50 || got.Stage != StageSprout {
		t.Errorf("after revive at 20:00 and the midnight sweep: n=%d state=%+v; want 0 and 50/sprout — the 15-minute challenge bought the day", n, got)
	}
	// The mirror case: the tick for the 22nd was missed (restart), the user
	// revives at 00:10 on the 23rd, the sweep runs late at 01:00. The plant
	// sat at 0 all through the 22nd, so that day's miss has nothing left to
	// take — it must not be taken from the 50 that came after (plan decision 4).
	h2 := newHarness(midnite.Add(10 * time.Minute))
	h2.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	if _, err := h2.svc.Revive(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	h2.study.set("u1", "2026-09-23", 900)
	if out, _ := h2.svc.Revive(ctx, "u1"); !out.Passed {
		t.Fatal("expected the pass")
	}
	if n, _ := h2.svc.Sweep(ctx, midnite.Add(time.Hour)); n != 0 || h2.repo.states["u1"].HealthPoints != 50 {
		t.Errorf("late sweep after a next-morning revival: n=%d health=%d, want 0 and 50", n, h2.repo.states["u1"].HealthPoints)
	}
}

func TestSweepStillJudgesYesterdayWhenTodaysTargetWasMetFirst(t *testing.T) {
	// The tick for the 22nd was missed; the user meets the 23rd's target at
	// 00:40 (a 1800s call at 00:00:30 counts toward the 23rd); the sweep runs
	// at 01:00. OnTargetMet moves last_target_met_date, not judged_through,
	// so the 22nd is still judged — and penalised.
	h := newHarness(midnite.Add(40 * time.Minute))
	h.repo.states["u1"] = State{HealthPoints: 100, CurrentStreak: 3, Stage: StageSapling, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	if err := h.svc.OnTargetMet(ctx, "u1", "2026-09-23"); err != nil {
		t.Fatal(err)
	}
	n, err := h.svc.Sweep(ctx, midnite.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["u1"]; n != 1 || got.HealthPoints != 70 || got.CurrentStreak != 0 || *got.LastTargetMetDate != "2026-09-23" || *got.JudgedThrough != "2026-09-22" {
		t.Errorf("n=%d %+v; want 1 and 70/0, last_target_met_date 2026-09-23, judged through 2026-09-22", n, got)
	}
}

func TestSweepFirstContactJudgesOnlyDaysThePetExisted(t *testing.T) {
	// Created at 00:10 on the 23rd (after the midnight tick), swept at 01:00:
	// there is no 22nd to judge; the marker is initialised instead.
	h := newHarness(midnite.Add(10 * time.Minute))
	if _, err := h.svc.Ensure(ctx, "late"); err != nil {
		t.Fatal(err)
	}
	n, err := h.svc.Sweep(ctx, midnite.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["late"]; n != 0 || got.HealthPoints != 100 || got.JudgedThrough == nil || *got.JudgedThrough != "2026-09-22" {
		t.Errorf("fresh pet after its first sweep: n=%d %+v; want untouched and judged through 2026-09-22", n, got)
	}
	// Created at 23:50 on the 22nd, swept at 00:00: it existed on the 22nd
	// (for ten minutes) and studied nothing — that is a miss, as today.
	h = newHarness(midnite.Add(-10 * time.Minute))
	if _, err := h.svc.Ensure(ctx, "early"); err != nil {
		t.Fatal(err)
	}
	if n, _ := h.svc.Sweep(ctx, midnite); n != 1 || h.repo.states["early"].HealthPoints != 70 {
		t.Errorf("pet created at 23:50: n=%d health=%d, want 1 and 70", n, h.repo.states["early"].HealthPoints)
	}
}

func TestSweepJudgesEveryLocalDayExactlyOnceInEveryZone(t *testing.T) {
	zones := []struct {
		tz   string
		from time.Time
	}{
		{"UTC", time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)},
		{"Asia/Ho_Chi_Minh", time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)},
		{"Asia/Kolkata", time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)},    // +05:30: midnight at :30
		{"Asia/Kathmandu", time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)},  // +05:45
		{"Pacific/Chatham", time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)}, // +12:45 / +13:45, DST 2026-09-27
		{"America/Havana", time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)}, // falls back 2026-11-01 01:00→00:00: local hour 0 twice
		{"America/Santiago", time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)}, // springs forward 2026-09-06 00:00→01:00: no local hour 0
	}
	for _, z := range zones {
		t.Run(z.tz, func(t *testing.T) {
			loc := quests.Location(z.tz)
			h := newHarness(z.from)
			h.repo.timezones["u"] = z.tz
			// Already judged through the day before the first tick's local
			// date, so every penalty below is one local-date change.
			first := quests.LocalDate(z.from, loc)
			h.repo.states["u"] = State{HealthPoints: 100, CurrentStreak: 5, Stage: StageSapling, UpdatedAt: z.from.Add(-96 * time.Hour), JudgedThrough: judgedThrough(PreviousDate(first))}

			seen := map[string]bool{first: true}
			penalised := 0
			for tick := z.from; tick.Before(z.from.Add(72 * time.Hour)); tick = tick.Add(time.Hour) {
				n, err := h.svc.Sweep(ctx, tick)
				if err != nil {
					t.Fatalf("sweep at %v: %v", tick, err)
				}
				if n > 1 {
					t.Errorf("tick %v penalised %d pets, there is one", tick, n)
				}
				penalised += n
				seen[quests.LocalDate(tick, loc)] = true
			}
			// Independent oracle: the number of local dates the ticks entered
			// after the first one — 2 for UTC (its first tick is already a
			// midnight), 3 for every other zone here. The oracle, not a
			// constant, decides, so a DST rule change cannot rot the test.
			want := len(seen) - 1
			if penalised != want {
				t.Errorf("penalised %d times over 72 hourly ticks, want %d (one per local day that ended)", penalised, want)
			}
			if got := h.repo.states["u"].HealthPoints; got != max(0, 100-30*want) {
				t.Errorf("health = %d, want %d", got, max(0, 100-30*want))
			}
		})
	}
}

func TestSweepStillJudgesAZoneWhoseMidnightDoesNotExist(t *testing.T) {
	// America/Santiago, 2026-09-06: clocks go 23:59:59 -04 → 01:00:00 -03.
	// The 04:00Z tick is 01:00 local; local hour 0 never happens that day.
	h := newHarness(time.Date(2026, 9, 6, 4, 0, 0, 0, time.UTC))
	h.repo.timezones["u"] = "America/Santiago"
	h.repo.states["u"] = State{HealthPoints: 100, UpdatedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), JudgedThrough: judgedThrough("2026-09-04")}

	n, err := h.svc.Sweep(ctx, time.Date(2026, 9, 6, 4, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["u"]; n != 1 || got.HealthPoints != 70 || *got.JudgedThrough != "2026-09-05" {
		t.Errorf("n=%d %+v; want 2026-09-05 judged at 01:00 local", n, got)
	}
}

func TestSweepContinuesPastAUserWhoseCounterIsUnreadable(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	h.repo.states["u2"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	h.study.errFor["u1"] = errBoom

	n, err := h.svc.Sweep(ctx, midnite)
	if err == nil || !strings.Contains(err.Error(), "u1") {
		t.Fatalf("err = %v, want the failing user named", err)
	}
	if n != 1 || h.repo.states["u1"].HealthPoints != 100 || h.repo.states["u2"].HealthPoints != 70 {
		t.Errorf("n=%d u1=%d u2=%d; want 1, u1 untouched, u2 penalised — the loop must continue past u1", n, h.repo.states["u1"].HealthPoints, h.repo.states["u2"].HealthPoints)
	}
}

func TestSweepContinuesPastAUserWhoseWriteFails(t *testing.T) {
	h := newHarness(midnite)
	h.repo.states["u1"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	h.repo.states["u2"] = State{HealthPoints: 100, UpdatedAt: midnite.Add(-72 * time.Hour), JudgedThrough: judgedThrough("2026-09-21")}
	h.repo.saveErrFor["u1"] = errBoom

	n, err := h.svc.Sweep(ctx, midnite)
	if err == nil || !strings.Contains(err.Error(), "u1") {
		t.Fatalf("err = %v, want the failing user named", err)
	}
	if n != 1 || h.repo.states["u1"].HealthPoints != 100 || h.repo.states["u2"].HealthPoints != 70 {
		t.Errorf("n=%d u1=%d u2=%d; want 1, u1 untouched, u2 penalised", n, h.repo.states["u1"].HealthPoints, h.repo.states["u2"].HealthPoints)
	}
}

func TestFakeSaveKeepsAMarkerItWasNotGiven(t *testing.T) {
	h := newHarness(sept22)
	marker := "2026-09-22"
	h.repo.states["u1"] = State{HealthPoints: 0, Stage: StageWilted, LastTargetMetDate: &marker}

	// Revive's Save carries the pre-image's marker — nil when the pre-image
	// predates a concurrent OnTargetMet. GREATEST(last_target_met_date, NULL)
	// keeps the stored one.
	if err := h.repo.Save(ctx, "u1", State{HealthPoints: 50, Stage: StageSprout}); err != nil {
		t.Fatal(err)
	}
	if got := h.repo.states["u1"].LastTargetMetDate; got == nil || *got != marker {
		t.Errorf("Save with a nil marker left last_target_met_date = %v, want %s kept", got, marker)
	}
	earlier := "2026-01-01"
	_ = h.repo.Save(ctx, "u1", State{HealthPoints: 50, Stage: StageSprout, LastTargetMetDate: &earlier})
	if got := h.repo.states["u1"].LastTargetMetDate; got == nil || *got != marker {
		t.Errorf("Save with an earlier marker moved last_target_met_date to %v, want %s kept", got, marker)
	}
}
