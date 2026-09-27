package pet

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

type fakeRepo struct {
	mu          sync.Mutex
	states      map[string]State  // by userID; present == row exists
	timezones   map[string]string // by userID; missing == "UTC"
	now         func() time.Time  // stamps a fresh row's updated_at like the DDL's CURRENT_TIMESTAMP
	ensured     int
	saved       int // applied writes across Save, SaveTargetMet, PenaliseMiss, MarkJudged
	ensureErr   error
	saveErr     error
	saveErrFor  map[string]error // per-user failure of any writer; nil == fine
	timezoneErr error

	// afterCandidates, when set, runs once per SweepCandidates call after the
	// list is built and the mutex released — a test barrier so two sweeps can
	// be made to hold the same stale list before either writes.
	afterCandidates func()
}

func newFakeRepo(now func() time.Time) *fakeRepo {
	return &fakeRepo{states: map[string]State{}, timezones: map[string]string{}, now: now, saveErrFor: map[string]error{}}
}

// defaultState mirrors the §3.2 column defaults a fresh INSERT produces —
// including updated_at = CURRENT_TIMESTAMP, which the sweep's first-contact
// rule reads.
func defaultState(now time.Time) State {
	return State{PlantName: "My Green Buddy", HealthPoints: 100, Stage: StageSprout, UpdatedAt: now}
}

func (f *fakeRepo) writeErr(userID string) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	return f.saveErrFor[userID]
}

func (f *fakeRepo) Ensure(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ensureErr != nil {
		return f.ensureErr
	}
	f.ensured++
	if _, ok := f.states[userID]; !ok {
		f.states[userID] = defaultState(f.now())
	}
	return nil
}

func (f *fakeRepo) EnsureNamed(_ context.Context, userID, plantName string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ensureErr != nil {
		return f.ensureErr
	}
	f.ensured++
	if _, ok := f.states[userID]; !ok {
		f.states[userID] = defaultState(f.now())
	}
	if plantName != "" {
		st := f.states[userID]
		st.PlantName = plantName
		f.states[userID] = st
	}
	return nil
}

func (f *fakeRepo) Get(_ context.Context, userID string) (State, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.states[userID]
	if !ok {
		return State{}, ErrNoPet
	}
	return s, nil
}

func (f *fakeRepo) Save(_ context.Context, userID string, s State) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.writeErr(userID); err != nil {
		return err
	}
	cur, ok := f.states[userID]
	if !ok {
		return ErrNoPet
	}
	f.saved++
	s.PlantName = cur.PlantName
	if s.JudgedThrough != nil {
		s.JudgedThrough = laterDate(cur.JudgedThrough, *s.JudgedThrough) // GREATEST(judged_through, $8)
	} else {
		s.JudgedThrough = cur.JudgedThrough
	}
	if s.LastTargetMetDate != nil {
		s.LastTargetMetDate = laterDate(cur.LastTargetMetDate, *s.LastTargetMetDate) // GREATEST(last_target_met_date, $7)
	} else {
		s.LastTargetMetDate = cur.LastTargetMetDate
	}
	f.states[userID] = s
	return nil
}

// dateBefore is the SQL predicate `col IS NULL OR col < $d`.
func dateBefore(col *string, d string) bool { return col == nil || *col < d }

// SaveTargetMet mirrors saveTargetMetSQL: read and write under one lock is
// the fake's "one statement"; ApplyTargetMet is the shared reference.
func (f *fakeRepo) SaveTargetMet(_ context.Context, userID string, now time.Time, localDate string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.writeErr(userID); err != nil {
		return false, err
	}
	cur, ok := f.states[userID]
	if !ok || !dateBefore(cur.LastTargetMetDate, localDate) {
		return false, nil
	}
	f.saved++
	f.states[userID] = ApplyTargetMet(cur, now, localDate)
	return true, nil
}

func (f *fakeRepo) PenaliseMiss(_ context.Context, userID, judged string, now time.Time) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.writeErr(userID); err != nil {
		return false, err
	}
	cur, ok := f.states[userID]
	if !ok || !dateBefore(cur.JudgedThrough, judged) {
		return false, nil
	}
	f.saved++
	f.states[userID] = ApplyMiss(cur, now, judged)
	return true, nil
}

func (f *fakeRepo) MarkJudged(_ context.Context, userID, judged string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.writeErr(userID); err != nil {
		return false, err
	}
	cur, ok := f.states[userID]
	if !ok || !dateBefore(cur.JudgedThrough, judged) {
		return false, nil
	}
	f.saved++
	cur.JudgedThrough = &judged
	f.states[userID] = cur
	return true, nil
}

func (f *fakeRepo) Timezone(_ context.Context, userID string) (string, error) {
	if f.timezoneErr != nil {
		return "", f.timezoneErr
	}
	if tz, ok := f.timezones[userID]; ok {
		return tz, nil
	}
	return "UTC", nil
}

func (f *fakeRepo) SweepCandidates(context.Context) ([]Candidate, error) {
	f.mu.Lock()
	var out []Candidate
	for userID, s := range f.states {
		tz, _ := f.Timezone(context.Background(), userID)
		out = append(out, Candidate{UserID: userID, Timezone: tz, State: s})
	}
	f.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].UserID < out[j].UserID })
	if f.afterCandidates != nil {
		f.afterCandidates()
	}
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
	err    error            // every read fails
	errFor map[string]error // one user's reads fail; the rest are served
}

func newFakeStudy() *fakeStudy {
	return &fakeStudy{totals: map[string]int64{}, errFor: map[string]error{}}
}

func (f *fakeStudy) set(userID, localDate string, seconds int64) {
	f.totals[userID+"|"+localDate] = seconds
}

func (f *fakeStudy) Total(_ context.Context, userID, localDate string) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	if err := f.errFor[userID]; err != nil {
		return 0, err
	}
	return f.totals[userID+"|"+localDate], nil
}

var errBoom = errors.New("boom")

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }
