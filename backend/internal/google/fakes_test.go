package google

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// patchedEvent records one PatchEvent call: the id it targeted and the body
// (Event) sent, so tests can assert PATCH does not drop the payload.
type patchedEvent struct {
	ID string
	Ev Event
}

// errSaveBoom is fakeRepo's default SaveSyncState failure — a pool hiccup or,
// most plausibly, the client disconnecting mid-sync and cancelling the
// request context that Exec runs under.
var errSaveBoom = errors.New("boom: pool hiccup / client disconnected")

// callLog is shared by every fake so tests can assert ordering.
type callLog struct{ calls []string }

func (l *callLog) add(format string, args ...any) {
	l.calls = append(l.calls, fmt.Sprintf(format, args...))
}

type fakeTokens struct {
	log   *callLog
	token string
	err   error
}

func (f *fakeTokens) RefreshToken(_ context.Context, userID string) (string, error) {
	f.log.add("tokens.RefreshToken(%s)", userID)
	return f.token, f.err
}

type fakeOAuth struct {
	log *callLog
	err error
}

func (f *fakeOAuth) AccessToken(_ context.Context, refresh string) (string, error) {
	f.log.add("oauth.AccessToken(%s)", refresh)
	if f.err != nil {
		return "", f.err
	}
	return "access-for-" + refresh, nil
}

type fakeCalendar struct {
	log      *callLog
	nextID   string
	patchErr error // returned by PatchEvent (e.g. ErrNotFound)
	errs     map[string]error
	known    map[string]bool // ids Google has seen: a repeat insert is a 409, like the real API
	inserted []Event
	patched  []patchedEvent
}

func (f *fakeCalendar) fail(method string) error { return f.errs[method] }

func (f *fakeCalendar) InsertEvent(_ context.Context, tok string, ev Event) (string, error) {
	f.log.add("calendar.InsertEvent(%s,%s)", tok, ev.ID)
	if err := f.fail("InsertEvent"); err != nil {
		return "", err
	}
	id := ev.ID
	if id == "" {
		id = f.nextID
	}
	if f.known == nil {
		f.known = map[string]bool{}
	}
	if f.known[id] {
		return "", fmt.Errorf("%w: calendar returned 409", ErrAlreadyExists)
	}
	f.known[id] = true
	f.inserted = append(f.inserted, ev)
	return id, nil
}

func (f *fakeCalendar) PatchEvent(_ context.Context, tok, id string, ev Event) error {
	f.log.add("calendar.PatchEvent(%s,%s)", tok, id)
	if err := f.fail("PatchEvent"); err != nil {
		return err
	}
	if f.patchErr != nil {
		return f.patchErr
	}
	f.patched = append(f.patched, patchedEvent{ID: id, Ev: ev})
	return nil
}

type fakeTasks struct {
	log       *callLog
	nextList  string
	insertErr error // returned by InsertTask after failAfter successes
	failAfter int
	errs      map[string]error
	deleted   []string
	tasks     map[string][]Task
}

func (f *fakeTasks) fail(method string) error { return f.errs[method] }

func (f *fakeTasks) InsertTaskList(_ context.Context, tok, title string) (string, error) {
	f.log.add("tasks.InsertTaskList(%s,%s)", tok, title)
	if err := f.fail("InsertTaskList"); err != nil {
		return "", err
	}
	if f.tasks == nil {
		f.tasks = map[string][]Task{}
	}
	return f.nextList, nil
}

func (f *fakeTasks) DeleteTaskList(_ context.Context, tok, id string) error {
	f.log.add("tasks.DeleteTaskList(%s,%s)", tok, id)
	if err := f.fail("DeleteTaskList"); err != nil {
		return err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeTasks) InsertTask(_ context.Context, tok, list string, t Task) error {
	f.log.add("tasks.InsertTask(%s,%s,%s)", tok, list, t.Title)
	if err := f.fail("InsertTask"); err != nil {
		return err
	}
	if f.insertErr != nil && len(f.tasks[list]) >= f.failAfter {
		return f.insertErr
	}
	f.tasks[list] = append(f.tasks[list], t)
	return nil
}

type fakeRepo struct {
	log     *callLog
	profile Profile
	roadmap Roadmap
	noRoad  bool
	days    []DayTitles
	state   SyncState
	noState bool
	saved   []SyncState
	errs    map[string]error

	failSaveAt int   // 1-based index of the SaveSyncState call that fails (0 = never)
	saveErr    error // what it fails with (defaults to errSaveBoom)
	saveCalls  int
}

func (f *fakeRepo) fail(method string) error { return f.errs[method] }

func (f *fakeRepo) Profile(_ context.Context, userID string) (Profile, error) {
	f.log.add("repo.Profile(%s)", userID)
	if err := f.fail("Profile"); err != nil {
		return Profile{}, err
	}
	return f.profile, nil
}

func (f *fakeRepo) ActiveRoadmap(_ context.Context, userID string) (Roadmap, error) {
	f.log.add("repo.ActiveRoadmap(%s)", userID)
	if err := f.fail("ActiveRoadmap"); err != nil {
		return Roadmap{}, err
	}
	if f.noRoad {
		return Roadmap{}, ErrNoActiveRoadmap
	}
	return f.roadmap, nil
}

func (f *fakeRepo) DayTitles(_ context.Context, roadmapID string) ([]DayTitles, error) {
	f.log.add("repo.DayTitles(%s)", roadmapID)
	if err := f.fail("DayTitles"); err != nil {
		return nil, err
	}
	return f.days, nil
}

func (f *fakeRepo) SyncState(_ context.Context, userID string) (SyncState, error) {
	f.log.add("repo.SyncState(%s)", userID)
	if err := f.fail("SyncState"); err != nil {
		return SyncState{}, err
	}
	if f.noState {
		return SyncState{}, ErrNoSyncState
	}
	return f.state, nil
}

func (f *fakeRepo) SaveSyncState(_ context.Context, s SyncState) error {
	f.log.add("repo.SaveSyncState(evt=%s,list=%s,roadmap=%s,n=%d)", s.CalendarEventID, s.TasklistID, s.RoadmapID, s.TasksCreatedCount)
	if err := f.fail("SaveSyncState"); err != nil {
		return err
	}
	f.saveCalls++
	if f.failSaveAt != 0 && f.saveCalls == f.failSaveAt {
		if f.saveErr == nil {
			return errSaveBoom
		}
		return f.saveErr
	}
	f.saved = append(f.saved, s)
	f.state, f.noState = s, false
	return nil
}

// harness wires fakes for a user in Ho Chi Minh City with a 28-day roadmap.
type harness struct {
	log    *callLog
	tokens *fakeTokens
	oauth  *fakeOAuth
	cal    *fakeCalendar
	tasks  *fakeTasks
	repo   *fakeRepo
	now    time.Time
	svc    *Service
}

func newHarness() *harness {
	log := &callLog{}
	days := make([]DayTitles, 0, RoadmapDays)
	for d := 1; d <= RoadmapDays; d++ {
		days = append(days, DayTitles{Day: d, Titles: []string{fmt.Sprintf("Vocab %d", d), fmt.Sprintf("Read %d", d), fmt.Sprintf("Practice %d", d)}})
	}
	h := &harness{
		log:    log,
		tokens: &fakeTokens{log: log, token: "1//refresh"},
		oauth:  &fakeOAuth{log: log},
		cal:    &fakeCalendar{log: log, nextID: "evt_new"},
		tasks:  &fakeTasks{log: log, nextList: "list_new"},
		repo: &fakeRepo{
			log:     log,
			profile: Profile{NotificationTime: "20:00:00", Timezone: "Asia/Ho_Chi_Minh"},
			roadmap: Roadmap{ID: "roadmap-A", CreatedAt: time.Date(2026, time.September, 1, 18, 0, 0, 0, time.UTC)},
			days:    days,
			noState: true,
		},
		now: time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC),
	}
	h.svc = NewService(h.tokens, h.oauth, h.cal, h.tasks, h.repo, func() time.Time { return h.now })
	return h
}
