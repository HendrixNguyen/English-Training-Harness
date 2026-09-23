package pet

import (
	"context"
	"errors"
	"sort"
	"time"
)

type fakeRepo struct {
	states    map[string]State  // by userID; present == row exists
	timezones map[string]string // by userID; missing == "UTC"
	ensured   int
	saved     int
	ensureErr error
	saveErr   error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{states: map[string]State{}, timezones: map[string]string{}}
}

// defaultState mirrors the §3.2 column defaults a fresh INSERT produces.
func defaultState(now time.Time) State {
	return State{PlantName: "My Green Buddy", HealthPoints: 100, Stage: StageSprout, UpdatedAt: now}
}

func (f *fakeRepo) Ensure(_ context.Context, userID string) error {
	if f.ensureErr != nil {
		return f.ensureErr
	}
	f.ensured++
	if _, ok := f.states[userID]; !ok {
		f.states[userID] = defaultState(time.Time{})
	}
	return nil
}

func (f *fakeRepo) Get(_ context.Context, userID string) (State, error) {
	s, ok := f.states[userID]
	if !ok {
		return State{}, ErrNoPet
	}
	return s, nil
}

func (f *fakeRepo) Save(_ context.Context, userID string, s State) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	if _, ok := f.states[userID]; !ok {
		return ErrNoPet
	}
	f.saved++
	s.PlantName = f.states[userID].PlantName
	f.states[userID] = s
	return nil
}

func (f *fakeRepo) Timezone(_ context.Context, userID string) (string, error) {
	if tz, ok := f.timezones[userID]; ok {
		return tz, nil
	}
	return "UTC", nil
}

func (f *fakeRepo) Timezones(context.Context) ([]string, error) {
	seen := map[string]bool{}
	for userID := range f.states {
		tz, _ := f.Timezone(context.Background(), userID)
		seen[tz] = true
	}
	out := make([]string, 0, len(seen))
	for tz := range seen {
		out = append(out, tz)
	}
	sort.Strings(out)
	return out, nil
}

func (f *fakeRepo) SweepCandidates(_ context.Context, timezones []string) ([]Candidate, error) {
	want := map[string]bool{}
	for _, tz := range timezones {
		want[tz] = true
	}
	var out []Candidate
	for userID, s := range f.states {
		tz, _ := f.Timezone(context.Background(), userID)
		if want[tz] {
			out = append(out, Candidate{UserID: userID, Timezone: tz, State: s})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UserID < out[j].UserID })
	return out, nil
}

type fakeChallenges struct {
	items   map[string]Challenge
	started int
	cleared int
}

func newFakeChallenges() *fakeChallenges { return &fakeChallenges{items: map[string]Challenge{}} }

func (f *fakeChallenges) Start(_ context.Context, userID string, c Challenge) error {
	f.started++
	f.items[userID] = c
	return nil
}

func (f *fakeChallenges) Get(_ context.Context, userID string) (Challenge, bool, error) {
	c, ok := f.items[userID]
	return c, ok, nil
}

func (f *fakeChallenges) Clear(_ context.Context, userID string) error {
	f.cleared++
	delete(f.items, userID)
	return nil
}

type fakeStudy struct {
	totals map[string]int64 // keyed by userID|localDate
	err    error
}

func newFakeStudy() *fakeStudy { return &fakeStudy{totals: map[string]int64{}} }

func (f *fakeStudy) set(userID, localDate string, seconds int64) {
	f.totals[userID+"|"+localDate] = seconds
}

func (f *fakeStudy) Total(_ context.Context, userID, localDate string) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.totals[userID+"|"+localDate], nil
}

var errBoom = errors.New("boom")

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }
