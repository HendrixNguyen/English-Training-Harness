package quests

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

type harness struct {
	svc      *Service
	log      *callLog
	counter  *fakeCounter
	quests   *fakeQuestRepo
	progress *fakeProgressRepo
	pet      *fakePet
}

func newHarness(t *testing.T, now time.Time) *harness {
	t.Helper()
	log := &callLog{}
	h := &harness{
		log:      log,
		counter:  newFakeCounter(log),
		quests:   newFakeQuestRepo(log),
		progress: newFakeProgressRepo(log),
		pet:      newFakePet(log),
	}
	h.quests.roadmap = &Roadmap{ID: "rm-1", CreatedAt: now.Add(-24 * time.Hour)}
	h.quests.exercises[1] = demoExercises(1)
	h.quests.exercises[2] = demoExercises(2)
	h.svc = NewService(h.counter, h.quests, h.progress, h.pet, fixedClock(now))
	return h
}

func TestRecordProgressIncrementsRedisBeforeWritingPostgres(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 600)
	if err != nil {
		t.Fatalf("RecordProgress() = %v", err)
	}
	if out.DailySecondsSpent != 600 || out.DailyMinutesSpent != 10 || out.IsTargetMet || out.NewlyMet {
		t.Errorf("out = %+v, want seconds=600 minutes=10 isTargetMet=false newlyMet=false", out)
	}

	want := []string{
		"INCRBY u1|2026-09-22 600",
		"EXPIRE u1|2026-09-22",
		"UPSERT daily_progress u1|2026-09-22 minutes=10 target=false",
		"MARK COMPLETE ex-2-reading",
	}
	if !reflect.DeepEqual(h.log.calls, want) {
		t.Errorf("call order =\n  %v\nwant\n  %v", h.log.calls, want)
	}
}

func TestProgressBelowTheTargetStillReportsThePetState(t *testing.T) {
	// §6.2: pet_health and streak_count are on every progress response, not
	// only the one that crosses the target.
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 600)
	if err != nil {
		t.Fatalf("RecordProgress() = %v", err)
	}
	if out.PetHealth != 80 || out.StreakCount != 4 {
		t.Errorf("pet = (%d, %d), want the unbumped (80, 4)", out.PetHealth, out.StreakCount)
	}
	if h.pet.fired != 0 {
		t.Errorf("hook fired %d times below the target, want 0", h.pet.fired)
	}
}

func TestCrossingExactly1800SecondsMeetsTheTargetAndFiresOnce(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	ctx := context.Background()

	for i, seconds := range []int64{600, 600} {
		out, err := h.svc.RecordProgress(ctx, "u1", "ex-2-reading", seconds)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if out.IsTargetMet {
			t.Fatalf("call %d: IsTargetMet true at %ds, want false", i, out.DailySecondsSpent)
		}
	}

	out, err := h.svc.RecordProgress(ctx, "u1", "ex-2-practice", 600)
	if err != nil {
		t.Fatalf("third call: %v", err)
	}
	if out.DailySecondsSpent != 1800 || out.DailyMinutesSpent != 30 {
		t.Errorf("spent = %ds/%dm, want 1800/30", out.DailySecondsSpent, out.DailyMinutesSpent)
	}
	if !out.IsTargetMet || !out.NewlyMet {
		t.Errorf("out = %+v, want isTargetMet and newlyMet both true at exactly 1800s", out)
	}
	if h.pet.fired != 1 {
		t.Errorf("hook fired %d times, want 1", h.pet.fired)
	}
	// State is read AFTER the hook: 80+20 = 100, 4+1 = 5 (§8 success logic).
	if out.PetHealth != 100 || out.StreakCount != 5 {
		t.Errorf("pet = (%d, %d), want the post-hook (100, 5)", out.PetHealth, out.StreakCount)
	}
	if row := h.progress.rows["u1|2026-09-22"]; row.minutes != 30 || !row.targetMet {
		t.Errorf("daily_progress row = %+v, want minutes=30 target=true", row)
	}
}

func TestFurtherProgressTheSameDayDoesNotRefire(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	ctx := context.Background()

	if _, err := h.svc.RecordProgress(ctx, "u1", "ex-2-reading", 1800); err != nil {
		t.Fatalf("first call: %v", err)
	}
	out, err := h.svc.RecordProgress(ctx, "u1", "ex-2-practice", 600)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if !out.IsTargetMet {
		t.Error("IsTargetMet = false after the target was already met")
	}
	if out.NewlyMet {
		t.Error("NewlyMet = true on a second call the same day")
	}
	if h.pet.fired != 1 {
		t.Errorf("hook fired %d times, want 1", h.pet.fired)
	}
	if out.PetHealth != 100 || out.StreakCount != 5 {
		t.Errorf("pet = (%d, %d), want (100, 5) — no second bump", out.PetHealth, out.StreakCount)
	}
	if row := h.progress.rows["u1|2026-09-22"]; row.minutes != 40 {
		t.Errorf("minutes_spent = %d, want 40 (2400s / 60)", row.minutes)
	}
}

func TestMinutesUseIntegerDivision(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 1799)
	if err != nil {
		t.Fatalf("RecordProgress: %v", err)
	}
	row := h.progress.rows["u1|2026-09-22"]
	if row.minutes != 29 || row.targetMet || out.DailyMinutesSpent != 29 {
		t.Errorf("row = %+v, out.minutes = %d; want minutes=29 target=false at 1799s", row, out.DailyMinutesSpent)
	}
}

func TestProgressUsesTheUsersTimezoneForTheDate(t *testing.T) {
	// 18:30Z is already the 23rd in Ho Chi Minh City.
	now := time.Date(2026, time.September, 22, 18, 30, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.quests.timezone = "Asia/Ho_Chi_Minh"

	if _, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 600); err != nil {
		t.Fatalf("RecordProgress: %v", err)
	}
	if _, ok := h.progress.rows["u1|2026-09-23"]; !ok {
		t.Errorf("rows = %v, want a row for the user's local date 2026-09-23", h.progress.rows)
	}
}

func TestAPetHookFailureDoesNotFailTheRequest(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.pet.hookErr = errors.New("pet is on fire")

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 1800)
	if err != nil {
		t.Fatalf("RecordProgress() = %v, want nil: a hook failure must not fail the write", err)
	}
	if !out.NewlyMet {
		t.Error("NewlyMet = false despite crossing the target")
	}
}

func TestAPetStateFailureDoesNotFailTheRequest(t *testing.T) {
	// The write is already committed; a 500 here would make the client retry
	// and double-count. Pet fields fall back to zero and are logged.
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.pet.stateErr = errors.New("pet_states unreachable")

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 600)
	if err != nil {
		t.Fatalf("RecordProgress() = %v, want nil", err)
	}
	if out.DailySecondsSpent != 600 || out.PetHealth != 0 || out.StreakCount != 0 {
		t.Errorf("out = %+v, want the progress recorded and zero pet fields", out)
	}
}

func TestRecordProgressRejectsAnExerciseOutsideTheActiveRoadmap(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	_, err := h.svc.RecordProgress(context.Background(), "u1", "someone-elses-exercise", 600)
	if !errors.Is(err, ErrExerciseNotFound) {
		t.Fatalf("err = %v, want ErrExerciseNotFound", err)
	}
	// A 404 must leave Redis and daily_progress byte for byte as they were:
	// this line was missing when the write-before-validate defect shipped.
	if len(h.log.calls) != 0 {
		t.Errorf("a rejected call touched Redis/Postgres: %v", h.log.calls)
	}
}

func TestRecordProgressRejectsAnExerciseFromAnotherDay(t *testing.T) {
	// The roadmap is on day 2; ex-1-reading is a real, owned exercise from
	// day 1. Progress is only recordable against today's tasks.
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	_, err := h.svc.RecordProgress(context.Background(), "u1", "ex-1-reading", 600)
	if !errors.Is(err, ErrExerciseNotFound) {
		t.Fatalf("err = %v, want ErrExerciseNotFound", err)
	}
	if len(h.log.calls) != 0 {
		t.Errorf("a rejected call touched Redis/Postgres: %v", h.log.calls)
	}
}

func TestRecordProgressRejectsAnOutOfRangeDuration(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	for _, seconds := range []int64{0, -1, -600, MaxDurationSeconds + 1, 1_000_000_000, 100_000_000_000_000} {
		_, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", seconds)
		if !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("seconds = %d: err = %v, want ErrInvalidDuration", seconds, err)
		}
	}
	if len(h.log.calls) != 0 {
		t.Errorf("a rejected call touched Redis/Postgres: %v", h.log.calls)
	}
}

func TestRecordProgressAcceptsTheMaximumPerCallDuration(t *testing.T) {
	// The bound is inclusive: one full hour in a single report is allowed.
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))

	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", MaxDurationSeconds)
	if err != nil {
		t.Fatalf("RecordProgress(%d) = %v, want nil", MaxDurationSeconds, err)
	}
	if out.DailySecondsSpent != MaxDurationSeconds || out.DailyMinutesSpent != MaxDurationSeconds/60 {
		t.Errorf("out = %+v, want %ds/%dm", out, MaxDurationSeconds, MaxDurationSeconds/60)
	}
}

func TestRecordProgressRejectsCrossingTheDailyCeiling(t *testing.T) {
	// minutes_spent is INT (§3.2); the ceiling keeps the counter far from it
	// and is checked against the running total BEFORE the INCRBY.
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.counter.totals["u1|2026-09-22"] = MaxDailySeconds - 100

	_, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 101)
	if !errors.Is(err, ErrInvalidDuration) {
		t.Fatalf("101s over the ceiling: err = %v, want ErrInvalidDuration", err)
	}
	if len(h.log.calls) != 0 {
		t.Fatalf("a rejected call touched Redis/Postgres: %v", h.log.calls)
	}

	// Landing exactly on the ceiling is allowed …
	out, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 100)
	if err != nil {
		t.Fatalf("100s to reach the ceiling exactly: %v", err)
	}
	if out.DailySecondsSpent != MaxDailySeconds {
		t.Errorf("DailySecondsSpent = %d, want %d", out.DailySecondsSpent, MaxDailySeconds)
	}
	// … and one more second is not.
	if _, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-practice", 1); !errors.Is(err, ErrInvalidDuration) {
		t.Errorf("1s past the ceiling: err = %v, want ErrInvalidDuration", err)
	}
	if got := h.counter.totals["u1|2026-09-22"]; got != MaxDailySeconds {
		t.Errorf("counter = %d after the rejection, want it untouched at %d", got, MaxDailySeconds)
	}
}

func TestRecordProgressWithoutAnActiveRoadmap(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.quests.roadmap = nil

	if _, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 600); !errors.Is(err, ErrNoActiveRoadmap) {
		t.Fatalf("err = %v, want ErrNoActiveRoadmap", err)
	}
}

func TestDailyReturnsTodaysThreeTasksAndTheRunningTotal(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	// Roadmap created 24h ago → day 2.
	if _, err := h.svc.RecordProgress(context.Background(), "u1", "ex-2-reading", 900); err != nil {
		t.Fatalf("seeding progress: %v", err)
	}

	got, err := h.svc.Daily(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Daily() = %v", err)
	}
	if got.Date != "2026-09-22" {
		t.Errorf("Date = %q, want 2026-09-22", got.Date)
	}
	if got.DayNumber != 2 {
		t.Errorf("DayNumber = %d, want 2", got.DayNumber)
	}
	if got.TotalMinutesRequired != 30 {
		t.Errorf("TotalMinutesRequired = %d, want 30 (§6.2)", got.TotalMinutesRequired)
	}
	if len(got.Tasks) != 3 {
		t.Fatalf("len(Tasks) = %d, want 3", len(got.Tasks))
	}
	if got.AccumulatedSeconds != 900 || got.IsTargetMet {
		t.Errorf("accumulated = %d isTargetMet = %t, want 900/false", got.AccumulatedSeconds, got.IsTargetMet)
	}
	seen := map[string]bool{}
	for _, task := range got.Tasks {
		seen[task.TaskType] = true
		if task.DurationMinutes != DefaultTaskMinutes {
			t.Errorf("task %s DurationMinutes = %d, want the %d-minute default", task.ID, task.DurationMinutes, DefaultTaskMinutes)
		}
	}
	for _, want := range []string{"vocabulary", "reading", "practice"} {
		if !seen[want] {
			t.Errorf("missing the %s task", want)
		}
	}
}

func TestDailyDateIsTheUsersLocalDate(t *testing.T) {
	// 18:30Z is already the 23rd in Ho Chi Minh City; §6.2's `date` must agree
	// with the day the counter and daily_progress are keyed on.
	h := newHarness(t, time.Date(2026, time.September, 22, 18, 30, 0, 0, time.UTC))
	h.quests.timezone = "Asia/Ho_Chi_Minh"

	got, err := h.svc.Daily(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Daily() = %v", err)
	}
	if got.Date != "2026-09-23" {
		t.Errorf("Date = %q, want 2026-09-23", got.Date)
	}
}

func TestDailyWithoutAnActiveRoadmap(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	h.quests.roadmap = nil

	if _, err := h.svc.Daily(context.Background(), "u1"); !errors.Is(err, ErrNoActiveRoadmap) {
		t.Fatalf("err = %v, want ErrNoActiveRoadmap", err)
	}
}

func TestDailyClampsPastDay28(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)
	h.quests.roadmap = &Roadmap{ID: "rm-1", CreatedAt: now.Add(-90 * 24 * time.Hour)}
	h.quests.exercises[28] = demoExercises(28)

	got, err := h.svc.Daily(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Daily() = %v", err)
	}
	if got.DayNumber != 28 {
		t.Errorf("DayNumber = %d, want 28 (clamped)", got.DayNumber)
	}
	if len(got.Tasks) != 3 {
		t.Errorf("len(Tasks) = %d, want 3", len(got.Tasks))
	}
}

func TestDailyWithNoRowsForTheDayReturnsAnEmptyList(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	delete(h.quests.exercises, 2)

	got, err := h.svc.Daily(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Daily() = %v", err)
	}
	if got.Tasks == nil || len(got.Tasks) != 0 {
		t.Errorf("Tasks = %#v, want an empty non-nil slice (serialises as [])", got.Tasks)
	}
}

func TestTaskTitleAndDurationComeFromContentJSON(t *testing.T) {
	// §6.2 puts title and duration_minutes on each task; §3.2 has no such
	// columns, so they ride inside content_json.
	rich := toTask(Exercise{
		ID: "ex-1", TaskType: "vocabulary", IsCompleted: true,
		ContentJSON: json.RawMessage(`{"title":"10 Key Business Email Phrasings","duration_minutes":15,"words":[]}`),
	})
	if rich.ID != "ex-1" || rich.TaskType != "vocabulary" || !rich.IsCompleted {
		t.Errorf("identity fields not copied: %+v", rich)
	}
	if rich.Title != "10 Key Business Email Phrasings" || rich.DurationMinutes != 15 {
		t.Errorf("title/duration = %q/%d, want the content_json values", rich.Title, rich.DurationMinutes)
	}
	if string(rich.ContentJSON) != `{"title":"10 Key Business Email Phrasings","duration_minutes":15,"words":[]}` {
		t.Errorf("ContentJSON was altered: %s", rich.ContentJSON)
	}

	bare := toTask(Exercise{ID: "ex-2", TaskType: "reading", ContentJSON: json.RawMessage(`{"passage":"..."}`)})
	if bare.Title != "" || bare.DurationMinutes != DefaultTaskMinutes {
		t.Errorf("bare task = %q/%d, want \"\"/%d", bare.Title, bare.DurationMinutes, DefaultTaskMinutes)
	}

	broken := toTask(Exercise{ID: "ex-3", TaskType: "practice", ContentJSON: json.RawMessage(`not json`)})
	if broken.DurationMinutes != DefaultTaskMinutes {
		t.Errorf("malformed content_json must still yield the default duration, got %d", broken.DurationMinutes)
	}
}
