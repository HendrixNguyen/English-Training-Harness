package google

import (
	"context"
	"fmt"
	"time"
)

// callLog is shared by every fake so tests can assert ordering.
type callLog struct{ calls []string }

func (l *callLog) add(format string, args ...any) { l.calls = append(l.calls, fmt.Sprintf(format, args...)) }

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
	inserted []Event
	patched  []string
}

func (f *fakeCalendar) InsertEvent(_ context.Context, tok string, ev Event) (string, error) {
	f.log.add("calendar.InsertEvent(%s)", tok)
	f.inserted = append(f.inserted, ev)
	return f.nextID, nil
}

func (f *fakeCalendar) PatchEvent(_ context.Context, tok, id string, _ Event) error {
	f.log.add("calendar.PatchEvent(%s,%s)", tok, id)
	if f.patchErr != nil {
		return f.patchErr
	}
	f.patched = append(f.patched, id)
	return nil
}

type fakeTasks struct {
	log       *callLog
	nextList  string
	insertErr error // returned by InsertTask after failAfter successes
	failAfter int
	deleted   []string
	tasks     map[string][]Task
}

func (f *fakeTasks) InsertTaskList(_ context.Context, tok, title string) (string, error) {
	f.log.add("tasks.InsertTaskList(%s,%s)", tok, title)
	if f.tasks == nil {
		f.tasks = map[string][]Task{}
	}
	return f.nextList, nil
}

func (f *fakeTasks) DeleteTaskList(_ context.Context, tok, id string) error {
	f.log.add("tasks.DeleteTaskList(%s,%s)", tok, id)
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeTasks) InsertTask(_ context.Context, tok, list string, t Task) error {
	f.log.add("tasks.InsertTask(%s,%s,%s)", tok, list, t.Title)
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
}

func (f *fakeRepo) Profile(_ context.Context, userID string) (Profile, error) {
	f.log.add("repo.Profile(%s)", userID)
	return f.profile, nil
}

func (f *fakeRepo) ActiveRoadmap(_ context.Context, userID string) (Roadmap, error) {
	f.log.add("repo.ActiveRoadmap(%s)", userID)
	if f.noRoad {
		return Roadmap{}, ErrNoActiveRoadmap
	}
	return f.roadmap, nil
}

func (f *fakeRepo) DayTitles(_ context.Context, roadmapID string) ([]DayTitles, error) {
	f.log.add("repo.DayTitles(%s)", roadmapID)
	return f.days, nil
}

func (f *fakeRepo) SyncState(_ context.Context, userID string) (SyncState, error) {
	f.log.add("repo.SyncState(%s)", userID)
	if f.noState {
		return SyncState{}, ErrNoSyncState
	}
	return f.state, nil
}

func (f *fakeRepo) SaveSyncState(_ context.Context, s SyncState) error {
	f.log.add("repo.SaveSyncState(evt=%s,list=%s,roadmap=%s,n=%d)", s.CalendarEventID, s.TasklistID, s.RoadmapID, s.TasksCreatedCount)
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
