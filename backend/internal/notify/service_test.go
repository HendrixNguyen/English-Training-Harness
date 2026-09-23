package notify

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

var validSub = &Subscription{Endpoint: "https://push.example/ep1", P256dh: "BNc5T", Auth: "aX8v"}

func TestUpdateSettingsStoresSubscriptionPreferencesAndSchedulesTonight(t *testing.T) {
	h := newHarness()
	res, err := h.svc.UpdateSettings(context.Background(), "u1", SettingsRequest{
		NotificationTime: "20:00", Timezone: "Asia/Ho_Chi_Minh", Subscription: validSub,
	})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if res.Status != "updated" || res.NotificationTime != "20:00:00" {
		t.Errorf("result = %+v (§6.4 body)", res)
	}
	want := h.hcm(22, 20, 0)
	if res.NextReminderAt != want.UTC().Format(time.RFC3339) {
		t.Errorf("next_reminder_at = %s, want %s", res.NextReminderAt, want.UTC().Format(time.RFC3339))
	}
	if h.queue.scores["u1"] != want.Unix() {
		t.Errorf("ZSET score = %d, want %d (tonight 20:00 HCM)", h.queue.scores["u1"], want.Unix())
	}
	if subs := h.repo.subs["u1"]; len(subs) != 1 || subs[0].Endpoint != validSub.Endpoint {
		t.Errorf("subscriptions = %+v", subs)
	}
	if p := h.repo.prefs["u1"]; p.NotificationTime != "20:00:00" || p.Timezone != "Asia/Ho_Chi_Minh" {
		t.Errorf("prefs = %+v", p)
	}
}

func TestUpdateSettingsWithoutASubscriptionOnlyMovesTheTime(t *testing.T) {
	h := newHarness()
	_, err := h.svc.UpdateSettings(context.Background(), "u1", SettingsRequest{NotificationTime: "06:30:00"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range h.log.calls {
		if strings.HasPrefix(c, "repo.SaveSubscription") {
			t.Errorf("no subscription in the request, but %s", c)
		}
	}
	// Timezone untouched (empty in request), time moved, tomorrow 06:30 HCM.
	if p := h.repo.prefs["u1"]; p.Timezone != "Asia/Ho_Chi_Minh" || p.NotificationTime != "06:30:00" {
		t.Errorf("prefs = %+v", p)
	}
	if want := h.hcm(23, 6, 30); h.queue.scores["u1"] != want.Unix() {
		t.Errorf("score = %d, want tomorrow 06:30 HCM %d", h.queue.scores["u1"], want.Unix())
	}
}

func TestUpdateSettingsRejectsBadInputBeforeWriting(t *testing.T) {
	cases := map[string]SettingsRequest{
		"bad clock":                {NotificationTime: "25:00"},
		"missing clock":            {NotificationTime: ""},
		"bad timezone":             {NotificationTime: "20:00", Timezone: "Mars/Olympus"},
		"subscription no endpoint": {NotificationTime: "20:00", Subscription: &Subscription{P256dh: "x", Auth: "y"}},
		"subscription no keys":     {NotificationTime: "20:00", Subscription: &Subscription{Endpoint: "https://e"}},
	}
	for name, req := range cases {
		h := newHarness()
		_, err := h.svc.UpdateSettings(context.Background(), "u1", req)
		if !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: err = %v, want ErrInvalidRequest", name, err)
		}
		if len(h.log.calls) != 0 {
			t.Errorf("%s: rejected request still made calls %v", name, h.log.calls)
		}
	}
}

func TestUpdateSettingsForAMissingUser(t *testing.T) {
	h := newHarness()
	_, err := h.svc.UpdateSettings(context.Background(), "ghost", SettingsRequest{NotificationTime: "20:00"})
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("err = %v, want ErrUserNotFound", err)
	}
}

// tick-time harness: it is 20:00:05 in HCM and u1 is due.
func dueHarness() *harness {
	h := newHarness()
	h.now = h.hcm(22, 20, 0).Add(5 * time.Second)
	h.queue.scores["u1"] = h.hcm(22, 20, 0).Unix()
	h.repo.subs["u1"] = []Subscription{
		{ID: "s1", Endpoint: "https://push.example/ep1", P256dh: "a", Auth: "b"},
		{ID: "s2", Endpoint: "https://push.example/ep2", P256dh: "c", Auth: "d"},
	}
	return h
}

func TestTickSendsToEverySubscriptionAndReslotsTomorrowBeforeSending(t *testing.T) {
	h := dueHarness()
	stats, err := h.svc.Tick(context.Background(), h.now)
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if stats.Due != 1 || stats.Sent != 2 || stats.Skipped != 0 || stats.Pruned != 0 {
		t.Errorf("stats = %+v", stats)
	}
	if want := h.hcm(23, 20, 0); h.queue.scores["u1"] != want.Unix() {
		t.Errorf("re-slot = %d, want tomorrow 20:00 HCM %d", h.queue.scores["u1"], want.Unix())
	}
	idx := func(prefix string) int {
		for i, c := range h.log.calls {
			if strings.HasPrefix(c, prefix) {
				return i
			}
		}
		return -1
	}
	if !(idx("queue.Schedule(u1") < idx("sender.Send(")) {
		t.Errorf("must re-slot before sending (crash safety): %v", h.log.calls)
	}
	if len(h.sender.sent) != 2 {
		t.Errorf("sent = %v", h.sender.sent)
	}
}

func TestTickIgnoresUsersNotYetDue(t *testing.T) {
	h := dueHarness()
	h.repo.prefs["u2"] = Preferences{NotificationTime: "21:00:00", Timezone: "Asia/Ho_Chi_Minh"}
	h.queue.scores["u2"] = h.hcm(22, 21, 0).Unix()
	h.repo.subs["u2"] = []Subscription{{ID: "s3", Endpoint: "https://push.example/ep3"}}

	stats, _ := h.svc.Tick(context.Background(), h.now)
	if stats.Due != 1 {
		t.Errorf("due = %d, want 1", stats.Due)
	}
	for _, e := range h.sender.sent {
		if e == "https://push.example/ep3" {
			t.Error("u2 (21:00) was sent at 20:00")
		}
	}
	if h.queue.scores["u2"] != h.hcm(22, 21, 0).Unix() {
		t.Error("u2's slot must not move")
	}
}

func TestTickSkipsAUserWhoAlreadyMetTodaysTarget(t *testing.T) {
	h := dueHarness()
	h.counter.totals["u1|2026-09-22"] = TargetSeconds // exactly 1800 counts as met (quests: total >= 1800)

	stats, err := h.svc.Tick(context.Background(), h.now)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Skipped != 1 || stats.Sent != 0 || len(h.sender.sent) != 0 {
		t.Errorf("stats = %+v, sent = %v", stats, h.sender.sent)
	}
	if want := h.hcm(23, 20, 0); h.queue.scores["u1"] != want.Unix() {
		t.Error("a skipped user is still re-slotted for tomorrow")
	}
	if !strings.Contains(strings.Join(h.log.calls, ";"), "counter.Total(u1,2026-09-22)") {
		t.Errorf("the counter must be read for the user's LOCAL date: %v", h.log.calls)
	}
}

func TestTickSendsWhenTheCounterIsUnavailable(t *testing.T) {
	h := dueHarness()
	h.counter.err = errors.New("redis down")
	stats, err := h.svc.Tick(context.Background(), h.now)
	if err == nil {
		t.Error("counter failure must be reported")
	}
	if stats.Sent != 2 {
		t.Errorf("sent = %d, want 2 — a missing counter means 'not met', not 'skip'", stats.Sent)
	}
}

func TestTickPrunesGoneSubscriptionsAndKeepsTheRest(t *testing.T) {
	h := dueHarness()
	h.sender.gone["https://push.example/ep1"] = true

	stats, err := h.svc.Tick(context.Background(), h.now)
	if err != nil {
		t.Fatalf("a gone subscription is normal, not an error: %v", err)
	}
	if stats.Pruned != 1 || stats.Sent != 1 {
		t.Errorf("stats = %+v", stats)
	}
	if subs := h.repo.subs["u1"]; len(subs) != 1 || subs[0].ID != "s2" {
		t.Errorf("subscriptions after prune = %+v, want only s2", subs)
	}
}

func TestTickReportsSendFailuresButContinues(t *testing.T) {
	h := dueHarness()
	h.sender.fail["https://push.example/ep1"] = true
	stats, err := h.svc.Tick(context.Background(), h.now)
	if err == nil || stats.Failed != 1 || stats.Sent != 1 {
		t.Errorf("err = %v, stats = %+v", err, stats)
	}
	if len(h.repo.subs["u1"]) != 2 {
		t.Error("a transient failure must not delete the subscription")
	}
}

func TestTickDropsUsersWithNoSubscriptionsOrNoRow(t *testing.T) {
	h := dueHarness()
	h.repo.subs["u1"] = nil
	h.queue.scores["ghost"] = h.now.Unix() - 10 // due, but has no users row

	stats, err := h.svc.Tick(context.Background(), h.now)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Due != 2 || stats.Sent != 0 {
		t.Errorf("stats = %+v", stats)
	}
	if _, ok := h.queue.scores["u1"]; ok {
		t.Error("u1 has no subscriptions and must leave the queue (UpdateSettings re-adds)")
	}
	if _, ok := h.queue.scores["ghost"]; ok {
		t.Error("a user with no row must leave the queue")
	}
}

func TestUpdateSettingsRefusesHostileEndpointsBeforeWriting(t *testing.T) {
	for name, raw := range hostileEndpoints {
		h := newHarness()
		_, err := h.svc.UpdateSettings(context.Background(), "u1", SettingsRequest{
			NotificationTime: "20:00",
			Subscription:     &Subscription{Endpoint: raw, P256dh: "BNc5T", Auth: "aX8v"},
		})
		if !errors.Is(err, ErrInvalidRequest) || !errors.Is(err, ErrForbiddenEndpoint) {
			t.Errorf("%s: err = %v, want ErrInvalidRequest wrapping ErrForbiddenEndpoint", name, err)
		}
		if len(h.log.calls) != 0 {
			t.Errorf("%s: rejected endpoint still made calls %v", name, h.log.calls)
		}
		if len(h.queue.scores) != 0 || len(h.repo.subs["u1"]) != 0 {
			t.Errorf("%s: rejected endpoint was stored or scheduled", name)
		}
	}
}

func TestTickPrunesAForbiddenEndpointAndReportsItOnce(t *testing.T) {
	// A row stored before validation existed (or via a DNS name that now
	// resolves privately) must be deleted on the first tick, not retried daily.
	h := dueHarness()
	h.sender.forbidden["https://push.example/ep1"] = true

	stats, err := h.svc.Tick(context.Background(), h.now)
	if !errors.Is(err, ErrForbiddenEndpoint) {
		t.Errorf("err = %v, want the forbidden endpoint reported so the worker logs it", err)
	}
	if stats.Pruned != 1 || stats.Sent != 1 || stats.Failed != 0 {
		t.Errorf("stats = %+v, want Pruned 1 Sent 1 Failed 0", stats)
	}
	if subs := h.repo.subs["u1"]; len(subs) != 1 || subs[0].ID != "s2" {
		t.Errorf("subscriptions after prune = %+v, want only s2", subs)
	}
	var deleted bool
	for _, c := range h.log.calls {
		if c == "repo.DeleteSubscription(s1)" {
			deleted = true
		}
	}
	if !deleted {
		t.Errorf("s1 was not deleted; calls = %v", h.log.calls)
	}
}
