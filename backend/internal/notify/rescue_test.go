package notify

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// fakeRescueRepo adds the candidates query to the shared fakeRepo without
// touching fakes_test.go (another plan is editing it today).
type fakeRescueRepo struct{ *fakeRepo }

func (f *fakeRescueRepo) RescueCandidates(_ context.Context) ([]RescueCandidate, error) {
	f.log.add("repo.RescueCandidates()")
	var out []RescueCandidate
	for id, p := range f.prefs {
		if len(f.subs[id]) > 0 { // the SQL joins push_subscriptions
			out = append(out, RescueCandidate{UserID: id, Timezone: p.Timezone})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UserID < out[j].UserID })
	return out, nil
}

// fixedCandidatesRepo answers a fixed candidate list whatever the
// subscriptions say — the race where a user unsubscribes between the
// candidates query and the send.
type fixedCandidatesRepo struct {
	*fakeRepo
	candidates []RescueCandidate
}

func (f *fixedCandidatesRepo) RescueCandidates(_ context.Context) ([]RescueCandidate, error) {
	f.log.add("repo.RescueCandidates()")
	return f.candidates, nil
}

type fakeRescueFlags struct {
	log     *callLog
	claimed map[string]bool // key → already set
	err     error
}

func (f *fakeRescueFlags) Claim(_ context.Context, userID, localDate string) (bool, error) {
	f.log.add("flags.Claim(%s,%s)", userID, localDate)
	if f.err != nil {
		return false, f.err
	}
	k := store.RescueKey(userID, localDate)
	if f.claimed[k] {
		return false, nil
	}
	f.claimed[k] = true
	return true, nil
}

// recordingSender delegates to fakeSender and keeps every payload it saw.
type recordingSender struct {
	*fakeSender
	pmu      sync.Mutex
	payloads []Payload
}

func (s *recordingSender) Send(ctx context.Context, sub Subscription, p Payload) error {
	s.pmu.Lock()
	s.payloads = append(s.payloads, p)
	s.pmu.Unlock()
	return s.fakeSender.Send(ctx, sub, p)
}

type rescueHarness struct {
	*harness
	flags  *fakeRescueFlags
	sender *recordingSender
	r      *Rescue
}

// newRescueHarness: u1 in Ho Chi Minh City with one subscription.
func newRescueHarness() *rescueHarness {
	h := newHarness()
	h.repo.subs["u1"] = []Subscription{{ID: "sub-1", Endpoint: "https://push.example/1", P256dh: "p", Auth: "a"}}
	rh := &rescueHarness{
		harness: h,
		flags:   &fakeRescueFlags{log: h.log, claimed: map[string]bool{}},
		sender:  &recordingSender{fakeSender: h.sender},
	}
	rh.r = NewRescue(&fakeRescueRepo{h.repo}, h.counter, rh.flags, rh.sender, func() time.Time { return h.now })
	return rh
}

func utc(day, hour int) time.Time {
	return time.Date(2026, time.September, day, hour, 0, 0, 0, time.UTC)
}

func countCalls(calls []string, prefix string) int {
	n := 0
	for _, c := range calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

func TestRescueMinutes(t *testing.T) {
	for _, tt := range []struct {
		total int64
		want  int
	}{{0, 30}, {1, 30}, {60, 29}, {1740, 1}, {1799, 1}} {
		if got := RescueMinutes(tt.total); got != tt.want {
			t.Errorf("RescueMinutes(%d) = %d, want %d", tt.total, got, tt.want)
		}
	}
}

func TestRescuePayloadCopy(t *testing.T) {
	want := Payload{Title: "Tớ cần cậu thêm 12 phút nữa", Body: "Cây của cậu cần thêm 12 phút hôm nay", URL: "/"}
	if got := RescuePayload(12); got != want {
		t.Errorf("RescuePayload(12) = %+v, want %+v", got, want)
	}
}

func TestRescueSweepSendsOnlyInLocalHour22(t *testing.T) {
	rh := newRescueHarness()
	rh.counter.totals["u1|2026-09-22"] = 720 // 12 minutes done

	for _, hour := range []int{14, 15, 16} {
		stats, err := rh.r.Sweep(context.Background(), utc(22, hour))
		if err != nil {
			t.Fatalf("sweep at %02d:00 UTC: %v", hour, err)
		}
		if hour == 15 {
			if stats.Candidates != 1 || stats.InWindow != 1 || stats.Sent != 1 {
				t.Errorf("15:00 UTC stats = %+v, want 1 candidate in window, 1 sent", stats)
			}
		} else if stats.InWindow != 0 || stats.Sent != 0 {
			t.Errorf("%02d:00 UTC stats = %+v, want nobody in window", hour, stats)
		}
	}
	if got := rh.sender.Sent(); len(got) != 1 || got[0] != "https://push.example/1" {
		t.Errorf("sent = %v, want one push to https://push.example/1", got)
	}
	if n := countCalls(rh.log.calls, "flags.Claim(u1,2026-09-22)"); n != 1 {
		t.Errorf("flags.Claim(u1,2026-09-22) called %d times, want 1; calls: %v", n, rh.log.calls)
	}
	want := Payload{Title: "Tớ cần cậu thêm 18 phút nữa", Body: "Cây của cậu cần thêm 18 phút hôm nay", URL: "/"}
	if len(rh.sender.payloads) != 1 || rh.sender.payloads[0] != want {
		t.Errorf("payloads = %+v, want [%+v]", rh.sender.payloads, want)
	}
}

func TestRescueSweepUsesTheUsersLocalDateAcrossMidnightUTC(t *testing.T) {
	rh := newRescueHarness()
	rh.repo.prefs["u2"] = Preferences{NotificationTime: "20:00:00", Timezone: "America/New_York"}
	rh.repo.subs["u2"] = []Subscription{{ID: "sub-2", Endpoint: "https://push.example/2", P256dh: "p", Auth: "a"}}

	// 2026-09-23T02:00Z = 22:00 EDT on 2026-09-22.
	stats, err := rh.r.Sweep(context.Background(), utc(23, 2))
	if err != nil {
		t.Fatal(err)
	}
	if stats.InWindow != 1 || stats.Sent != 1 {
		t.Errorf("stats = %+v, want u2 in window and sent", stats)
	}
	if got := rh.sender.Sent(); len(got) != 1 || got[0] != "https://push.example/2" {
		t.Errorf("sent = %v, want one push to u2", got)
	}
	joined := strings.Join(rh.log.calls, ";")
	for _, want := range []string{"counter.Total(u2,2026-09-22)", "flags.Claim(u2,2026-09-22)"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %s; calls: %v", want, rh.log.calls)
		}
	}
	if strings.Contains(joined, "2026-09-23") {
		t.Errorf("the UTC date leaked into a key; calls: %v", rh.log.calls)
	}
}

func TestRescueSweepSendsAtMostOncePerDay(t *testing.T) {
	rh := newRescueHarness()
	at := utc(22, 15)

	first, err := rh.r.Sweep(context.Background(), at)
	if err != nil || first.Sent != 1 {
		t.Fatalf("first sweep = %+v, %v; want 1 sent", first, err)
	}
	second, err := rh.r.Sweep(context.Background(), at.Add(20*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if second.AlreadySent != 1 || second.Sent != 0 {
		t.Errorf("second sweep = %+v, want AlreadySent 1, Sent 0", second)
	}
	if got := rh.sender.Sent(); len(got) != 1 {
		t.Errorf("sent = %v, want exactly one push", got)
	}
	if !rh.flags.claimed["rescue:u1:2026-09-22"] {
		t.Errorf("flag key rescue:u1:2026-09-22 not set; claimed: %v", rh.flags.claimed)
	}
}

func TestRescueSweepSkipsAMetDay(t *testing.T) {
	rh := newRescueHarness()
	rh.counter.totals["u1|2026-09-22"] = 1800

	stats, err := rh.r.Sweep(context.Background(), utc(22, 15))
	if err != nil {
		t.Fatal(err)
	}
	if stats.Met != 1 || stats.Sent != 0 {
		t.Errorf("stats = %+v, want Met 1, nothing sent", stats)
	}
	if len(rh.sender.Sent()) != 0 {
		t.Errorf("sent on a met day: %v", rh.sender.Sent())
	}
	if n := countCalls(rh.log.calls, "flags.Claim("); n != 0 {
		t.Errorf("a met day claimed the flag; calls: %v", rh.log.calls)
	}
}

func TestRescueSweepSkipsAUserWhenRedisFails(t *testing.T) {
	rh := newRescueHarness()
	rh.counter.err = errors.New("redis down")

	stats, err := rh.r.Sweep(context.Background(), utc(22, 15))
	if stats.Skipped != 1 || stats.Sent != 0 {
		t.Errorf("counter error: stats = %+v, want Skipped 1, nothing sent", stats)
	}
	if err == nil || !strings.Contains(err.Error(), "u1") {
		t.Errorf("counter error: err = %v, want one naming u1", err)
	}
	if n := countCalls(rh.log.calls, "flags.Claim("); n != 0 {
		t.Errorf("counter error claimed the flag; calls: %v", rh.log.calls)
	}

	rh.counter.err = nil
	rh.flags.err = errors.New("redis down")
	stats, err = rh.r.Sweep(context.Background(), utc(22, 15))
	if stats.Skipped != 1 || stats.Sent != 0 {
		t.Errorf("flag error: stats = %+v, want Skipped 1, nothing sent", stats)
	}
	if err == nil || !strings.Contains(err.Error(), "u1") {
		t.Errorf("flag error: err = %v, want one naming u1", err)
	}
	if len(rh.sender.Sent()) != 0 {
		t.Errorf("sent despite a Redis error: %v", rh.sender.Sent())
	}
}

func TestRescueSweepIsSilentWithoutASubscription(t *testing.T) {
	rh := newRescueHarness()
	rh.repo.prefs["u3"] = Preferences{NotificationTime: "20:00:00", Timezone: "Asia/Ho_Chi_Minh"}
	rh.repo.subs["u3"] = nil

	stats, err := rh.r.Sweep(context.Background(), utc(22, 15))
	if err != nil {
		t.Fatal(err)
	}
	if stats.Candidates != 1 {
		t.Errorf("candidates = %d, want 1 (u1 only)", stats.Candidates)
	}
	if strings.Contains(strings.Join(rh.log.calls, ";"), "u3") {
		t.Errorf("u3 was touched; calls: %v", rh.log.calls)
	}

	// The race: u1 is a candidate but has unsubscribed by the time we send.
	h := newHarness()
	flags := &fakeRescueFlags{log: h.log, claimed: map[string]bool{}}
	repo := &fixedCandidatesRepo{fakeRepo: h.repo, candidates: []RescueCandidate{{UserID: "u1", Timezone: "Asia/Ho_Chi_Minh"}}}
	r := NewRescue(repo, h.counter, flags, h.sender, func() time.Time { return h.now })
	stats, err = r.Sweep(context.Background(), utc(22, 15))
	if err != nil {
		t.Errorf("no subscription is not an error: %v", err)
	}
	if stats.Sent != 0 || len(h.sender.Sent()) != 0 {
		t.Errorf("sent without a subscription: stats %+v, sent %v", stats, h.sender.Sent())
	}
}

func TestRescueSweepPrunesGoneSubscriptionsAndContinues(t *testing.T) {
	rh := newRescueHarness()
	rh.repo.subs["u1"] = []Subscription{
		{ID: "sub-1", Endpoint: "https://push.example/gone", P256dh: "p", Auth: "a"},
		{ID: "sub-2", Endpoint: "https://push.example/ok", P256dh: "p", Auth: "a"},
	}
	rh.sender.gone["https://push.example/gone"] = true

	stats, err := rh.r.Sweep(context.Background(), utc(22, 15))
	if err != nil {
		t.Fatal(err)
	}
	if stats.Pruned != 1 || stats.Sent != 1 {
		t.Errorf("stats = %+v, want Pruned 1, Sent 1", stats)
	}
	if !strings.Contains(strings.Join(rh.log.calls, ";"), "repo.DeleteSubscription(sub-1)") {
		t.Errorf("gone subscription not deleted; calls: %v", rh.log.calls)
	}

	// A failing endpoint for u1 must not stop u4's rescue.
	rh = newRescueHarness()
	rh.repo.subs["u1"] = []Subscription{{ID: "sub-1", Endpoint: "https://push.example/fail", P256dh: "p", Auth: "a"}}
	rh.sender.fail["https://push.example/fail"] = true
	rh.repo.prefs["u4"] = Preferences{NotificationTime: "20:00:00", Timezone: "Asia/Ho_Chi_Minh"}
	rh.repo.subs["u4"] = []Subscription{{ID: "sub-4", Endpoint: "https://push.example/4", P256dh: "p", Auth: "a"}}

	stats, err = rh.r.Sweep(context.Background(), utc(22, 15))
	if stats.Failed != 1 || stats.Sent != 1 {
		t.Errorf("stats = %+v, want Failed 1 (u1), Sent 1 (u4)", stats)
	}
	if err == nil || !strings.Contains(err.Error(), "u1") {
		t.Errorf("err = %v, want one naming u1", err)
	}
	if got := rh.sender.Sent(); len(got) != 1 || got[0] != "https://push.example/4" {
		t.Errorf("sent = %v, want u4's push", got)
	}
}
