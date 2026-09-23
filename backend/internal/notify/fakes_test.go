package notify

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type callLog struct{ calls []string }

func (l *callLog) add(format string, args ...any) {
	l.calls = append(l.calls, fmt.Sprintf(format, args...))
}

type fakeRepo struct {
	log   *callLog
	prefs map[string]Preferences    // by user id; missing → ErrUserNotFound
	subs  map[string][]Subscription // by user id
	seq   int
}

func newFakeRepo(log *callLog) *fakeRepo {
	return &fakeRepo{log: log, prefs: map[string]Preferences{}, subs: map[string][]Subscription{}}
}

func (f *fakeRepo) UpdatePreferences(_ context.Context, userID, clock, tz string) error {
	f.log.add("repo.UpdatePreferences(%s,%s,%s)", userID, clock, tz)
	p, ok := f.prefs[userID]
	if !ok {
		return ErrUserNotFound
	}
	p.NotificationTime = clock
	if tz != "" {
		p.Timezone = tz
	}
	f.prefs[userID] = p
	return nil
}

func (f *fakeRepo) Preferences(_ context.Context, userID string) (Preferences, error) {
	f.log.add("repo.Preferences(%s)", userID)
	p, ok := f.prefs[userID]
	if !ok {
		return Preferences{}, ErrUserNotFound
	}
	return p, nil
}

func (f *fakeRepo) SaveSubscription(_ context.Context, userID string, s Subscription) error {
	f.log.add("repo.SaveSubscription(%s,%s)", userID, s.Endpoint)
	for _, existing := range f.subs[userID] {
		if existing.Endpoint == s.Endpoint {
			return nil
		}
	}
	f.seq++
	s.ID = fmt.Sprintf("sub-%d", f.seq)
	f.subs[userID] = append(f.subs[userID], s)
	return nil
}

func (f *fakeRepo) Subscriptions(_ context.Context, userID string) ([]Subscription, error) {
	f.log.add("repo.Subscriptions(%s)", userID)
	return f.subs[userID], nil
}

func (f *fakeRepo) DeleteSubscription(_ context.Context, id string) error {
	f.log.add("repo.DeleteSubscription(%s)", id)
	for uid, list := range f.subs {
		kept := list[:0]
		for _, s := range list {
			if s.ID != id {
				kept = append(kept, s)
			}
		}
		f.subs[uid] = kept
	}
	return nil
}

type fakeQueue struct {
	log    *callLog
	scores map[string]int64 // member → unix score
}

func newFakeQueue(log *callLog) *fakeQueue { return &fakeQueue{log: log, scores: map[string]int64{}} }

func (f *fakeQueue) Schedule(_ context.Context, userID string, at time.Time) error {
	f.log.add("queue.Schedule(%s,%s)", userID, at.UTC().Format(time.RFC3339))
	f.scores[userID] = at.Unix()
	return nil
}

func (f *fakeQueue) Due(_ context.Context, now time.Time, limit int64) ([]string, error) {
	f.log.add("queue.Due(%s)", now.UTC().Format(time.RFC3339))
	var out []string
	for m, s := range f.scores {
		if s <= now.Unix() {
			out = append(out, m)
		}
	}
	sort.Strings(out)
	if int64(len(out)) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeQueue) Remove(_ context.Context, userID string) error {
	f.log.add("queue.Remove(%s)", userID)
	delete(f.scores, userID)
	return nil
}

type fakeSender struct {
	log       *callLog
	gone      map[string]bool // endpoint → answer ErrSubscriptionGone
	fail      map[string]bool // endpoint → answer a generic error
	forbidden map[string]bool // endpoint → answer ErrForbiddenEndpoint

	// mu guards sent: RunWorker's tests read it from the test goroutine while
	// Tick runs in a background one (see worker_test.go). Every other test
	// calls Tick synchronously, but the mutex costs nothing there either.
	mu   sync.Mutex
	sent []string // endpoints in order
}

func newFakeSender(log *callLog) *fakeSender {
	return &fakeSender{log: log, gone: map[string]bool{}, fail: map[string]bool{}, forbidden: map[string]bool{}}
}

func (f *fakeSender) Send(_ context.Context, sub Subscription, _ Payload) error {
	f.log.add("sender.Send(%s)", sub.Endpoint)
	if f.gone[sub.Endpoint] {
		return ErrSubscriptionGone
	}
	if f.forbidden[sub.Endpoint] {
		return fmt.Errorf("%w: refusing to dial 10.0.0.5", ErrForbiddenEndpoint)
	}
	if f.fail[sub.Endpoint] {
		return fmt.Errorf("push service returned 429")
	}
	f.mu.Lock()
	f.sent = append(f.sent, sub.Endpoint)
	f.mu.Unlock()
	return nil
}

// Sent returns a snapshot of the endpoints sent so far. Tests that call Tick
// synchronously may also read f.sent directly; Sent() is for the one test
// (worker_test.go) that reads it while a worker goroutine is still writing.
func (f *fakeSender) Sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.sent...)
}

type fakeCounter struct {
	log    *callLog
	totals map[string]int64 // userID+"|"+localDate → seconds
	err    error
}

func newFakeCounter(log *callLog) *fakeCounter {
	return &fakeCounter{log: log, totals: map[string]int64{}}
}

func (f *fakeCounter) Total(_ context.Context, userID, localDate string) (int64, error) {
	f.log.add("counter.Total(%s,%s)", userID, localDate)
	if f.err != nil {
		return 0, f.err
	}
	return f.totals[userID+"|"+localDate], nil
}

// harness: one user in Ho Chi Minh City with a 20:00 reminder.
type harness struct {
	log     *callLog
	repo    *fakeRepo
	queue   *fakeQueue
	sender  *fakeSender
	counter *fakeCounter
	now     time.Time
	svc     *Service
}

func newHarness() *harness {
	log := &callLog{}
	h := &harness{
		log:     log,
		repo:    newFakeRepo(log),
		queue:   newFakeQueue(log),
		sender:  newFakeSender(log),
		counter: newFakeCounter(log),
		// 2026-09-22T10:00Z = 17:00 in Ho Chi Minh City.
		now: time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC),
	}
	h.repo.prefs["u1"] = Preferences{NotificationTime: "20:00:00", Timezone: "Asia/Ho_Chi_Minh"}
	h.svc = NewService(h.repo, h.queue, h.sender, h.counter, func() time.Time { return h.now })
	return h
}

func (h *harness) hcm(day, hour, min int) time.Time {
	return time.Date(2026, time.September, day, hour, min, 0, 0, Location("Asia/Ho_Chi_Minh"))
}
