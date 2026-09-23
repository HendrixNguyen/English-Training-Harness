package quests

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// callLog records the order of side effects across all fakes in one test, so
// the §5.2 ordering contract (counter first, then Postgres) is asserted rather
// than assumed.
type callLog struct{ calls []string }

func (l *callLog) add(format string, args ...any) {
	l.calls = append(l.calls, fmt.Sprintf(format, args...))
}

type fakeCounter struct {
	log    *callLog
	totals map[string]int64 // keyed by userID|localDate
	err    error
}

func newFakeCounter(l *callLog) *fakeCounter {
	return &fakeCounter{log: l, totals: map[string]int64{}}
}

func (f *fakeCounter) key(userID, date string) string { return userID + "|" + date }

func (f *fakeCounter) Add(_ context.Context, userID, localDate string, seconds int64) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.log.add("INCRBY %s %d", f.key(userID, localDate), seconds)
	f.totals[f.key(userID, localDate)] += seconds
	f.log.add("EXPIRE %s", f.key(userID, localDate))
	return f.totals[f.key(userID, localDate)], nil
}

func (f *fakeCounter) Total(_ context.Context, userID, localDate string) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.totals[f.key(userID, localDate)], nil
}

type fakeQuestRepo struct {
	log             *callLog
	timezone        string
	roadmap         *Roadmap
	exercises       map[int][]Exercise // by day_number
	completed       map[string]bool
	markCompleteErr error
}

func newFakeQuestRepo(l *callLog) *fakeQuestRepo {
	return &fakeQuestRepo{
		log:       l,
		timezone:  "UTC",
		exercises: map[int][]Exercise{},
		completed: map[string]bool{},
	}
}

func (f *fakeQuestRepo) Profile(context.Context, string) (Profile, error) {
	return Profile{Timezone: f.timezone}, nil
}

func (f *fakeQuestRepo) ActiveRoadmap(context.Context, string) (Roadmap, error) {
	if f.roadmap == nil {
		return Roadmap{}, ErrNoActiveRoadmap
	}
	return *f.roadmap, nil
}

func (f *fakeQuestRepo) ExercisesForDay(_ context.Context, _ string, day int) ([]Exercise, error) {
	return f.exercises[day], nil
}

func (f *fakeQuestRepo) CheckExercise(_ context.Context, _, exerciseID string, day int) error {
	for _, e := range f.exercises[day] {
		if e.ID == exerciseID {
			return nil
		}
	}
	return ErrExerciseNotFound
}

func (f *fakeQuestRepo) MarkComplete(_ context.Context, _, exerciseID string) error {
	if f.markCompleteErr != nil {
		return f.markCompleteErr
	}
	f.log.add("MARK COMPLETE %s", exerciseID)
	f.completed[exerciseID] = true
	return nil
}

type fakeProgressRepo struct {
	log  *callLog
	rows map[string]struct {
		minutes   int
		targetMet bool
	}
	err     error // Upsert fails
	markErr error // MarkTargetMet fails
}

func newFakeProgressRepo(l *callLog) *fakeProgressRepo {
	return &fakeProgressRepo{log: l, rows: map[string]struct {
		minutes   int
		targetMet bool
	}{}}
}

// Upsert mirrors upsertProgressSQL: minutes never lower, and the row's
// is_target_met is reported, never written here.
func (f *fakeProgressRepo) Upsert(_ context.Context, userID, localDate string, minutes int) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	key := userID + "|" + localDate
	row := f.rows[key]
	row.minutes = max(row.minutes, minutes)
	f.rows[key] = row
	f.log.add("UPSERT daily_progress %s minutes=%d", key, row.minutes)
	return row.targetMet, nil
}

func (f *fakeProgressRepo) MarkTargetMet(_ context.Context, userID, localDate string) error {
	if f.markErr != nil {
		return f.markErr
	}
	key := userID + "|" + localDate
	row := f.rows[key]
	row.targetMet = true
	f.rows[key] = row
	f.log.add("MARK TARGET MET %s", key)
	return nil
}

type fakePet struct {
	log      *callLog
	fired    int
	hookErr  error
	state    PetState
	stateErr error
}

// newFakePet starts mid-course (health 80, streak 4) so a +20/+1 bump is visible.
func newFakePet(l *callLog) *fakePet {
	return &fakePet{log: l, state: PetState{Health: 80, Streak: 4}}
}

// OnTargetMet applies §8's success arithmetic to the fake state so a test can
// prove State() is read after the hook, not before.
func (p *fakePet) OnTargetMet(_ context.Context, userID, localDate string) error {
	p.fired++
	p.log.add("ON TARGET MET %s|%s", userID, localDate)
	if p.hookErr != nil {
		return p.hookErr
	}
	p.state.Health = min(100, p.state.Health+20)
	p.state.Streak++
	return nil
}

func (p *fakePet) State(context.Context, string) (PetState, error) {
	if p.stateErr != nil {
		return PetState{}, p.stateErr
	}
	return p.state, nil
}

// demoExercises builds a day's three tasks in the §6.1 categories.
func demoExercises(day int) []Exercise {
	return []Exercise{
		{ID: fmt.Sprintf("ex-%d-practice", day), DayNumber: day, TaskType: "practice", ContentJSON: json.RawMessage(`{"n":1}`)},
		{ID: fmt.Sprintf("ex-%d-reading", day), DayNumber: day, TaskType: "reading", ContentJSON: json.RawMessage(`{"n":2}`)},
		{ID: fmt.Sprintf("ex-%d-vocabulary", day), DayNumber: day, TaskType: "vocabulary", ContentJSON: json.RawMessage(`{"n":3}`)},
	}
}

// fixedClock returns a clock pinned to t.
func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }
