package google

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type recordedCall struct {
	Method, Path, Auth string
	Body               map[string]any
}

// fakeGoogleAPI records every request and answers from a per-path script.
func fakeGoogleAPI(t *testing.T, answers map[string]struct {
	Status int
	Body   string
}) (*httptest.Server, *[]recordedCall) {
	t.Helper()
	var calls []recordedCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := recordedCall{Method: r.Method, Path: r.URL.Path, Auth: r.Header.Get("Authorization")}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&rc.Body)
		}
		calls = append(calls, rc)
		a, ok := answers[r.Method+" "+r.URL.Path]
		if !ok {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(a.Status)
		_, _ = w.Write([]byte(a.Body))
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func sampleEvent() Event {
	start := time.Date(2026, time.September, 22, 20, 0, 0, 0, Location("Asia/Ho_Chi_Minh"))
	return Event{Summary: EventSummary, Description: EventDescription, Start: start, End: start.Add(EventDuration), TimeZone: "Asia/Ho_Chi_Minh"}
}

func TestCalendarInsertEventPostsToPrimaryAndReturnsTheID(t *testing.T) {
	srv, calls := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{"POST /calendars/primary/events": {200, `{"id":"evt_1","status":"confirmed"}`}})

	c := NewHTTPCalendarClient()
	c.BaseURL = srv.URL
	id, err := c.InsertEvent(context.Background(), "ya29.tok", sampleEvent())
	if err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}
	if id != "evt_1" {
		t.Errorf("id = %q", id)
	}
	call := (*calls)[0]
	if call.Auth != "Bearer ya29.tok" {
		t.Errorf("Authorization = %q", call.Auth)
	}
	rec, _ := call.Body["recurrence"].([]any)
	if len(rec) != 1 || rec[0] != Recurrence {
		t.Errorf("recurrence = %v, want [%s]", call.Body["recurrence"], Recurrence)
	}
	start, _ := call.Body["start"].(map[string]any)
	if start["timeZone"] != "Asia/Ho_Chi_Minh" || start["dateTime"] != "2026-09-22T20:00:00+07:00" {
		t.Errorf("start = %v", start)
	}
}

func TestCalendarPatchEventUsesTheStoredID(t *testing.T) {
	srv, calls := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{"PATCH /calendars/primary/events/evt_1": {200, `{"id":"evt_1"}`}})

	c := NewHTTPCalendarClient()
	c.BaseURL = srv.URL
	if err := c.PatchEvent(context.Background(), "ya29.tok", "evt_1", sampleEvent()); err != nil {
		t.Fatalf("PatchEvent: %v", err)
	}
	if got := (*calls)[0]; got.Method != http.MethodPatch || got.Path != "/calendars/primary/events/evt_1" {
		t.Errorf("call = %+v", got)
	}
}

func TestCalendarMapsStatusesToSentinelErrors(t *testing.T) {
	srv, _ := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{
		"PATCH /calendars/primary/events/gone":      {404, `{"error":{"code":404}}`},
		"PATCH /calendars/primary/events/forbidden": {403, `{"error":{"code":403,"message":"insufficient scopes"}}`},
		"PATCH /calendars/primary/events/broken":    {500, `{"error":{"code":500}}`},
	})
	c := NewHTTPCalendarClient()
	c.BaseURL = srv.URL
	ctx := context.Background()

	if err := c.PatchEvent(ctx, "t", "gone", sampleEvent()); !errors.Is(err, ErrNotFound) {
		t.Errorf("404: err = %v, want ErrNotFound", err)
	}
	if err := c.PatchEvent(ctx, "t", "forbidden", sampleEvent()); !errors.Is(err, ErrReauthRequired) {
		t.Errorf("403: err = %v, want ErrReauthRequired", err)
	}
	var up *UpstreamError
	if err := c.PatchEvent(ctx, "t", "broken", sampleEvent()); !errors.As(err, &up) || up.Service != "calendar" {
		t.Errorf("500: err = %v, want *UpstreamError{calendar}", err)
	}
}
