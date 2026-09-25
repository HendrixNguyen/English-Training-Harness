package google

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestTasksInsertTaskListReturnsTheID(t *testing.T) {
	srv, calls := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{"POST /users/@me/lists": {200, `{"id":"list_1","title":"English daily quests"}`}})

	c := NewHTTPTasksClient()
	c.BaseURL = srv.URL
	id, err := c.InsertTaskList(context.Background(), "ya29.tok", TasklistTitle)
	if err != nil {
		t.Fatalf("InsertTaskList: %v", err)
	}
	if id != "list_1" || (*calls)[0].Body["title"] != TasklistTitle {
		t.Errorf("id = %q, body = %v", id, (*calls)[0].Body)
	}
}

func TestTasksDeleteTaskListTreatsMissingAsDone(t *testing.T) {
	srv, calls := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{
		"DELETE /users/@me/lists/list_1": {204, ``},
		"DELETE /users/@me/lists/gone":   {404, `{"error":{"code":404}}`},
	})
	c := NewHTTPTasksClient()
	c.BaseURL = srv.URL
	if err := c.DeleteTaskList(context.Background(), "t", "list_1"); err != nil {
		t.Errorf("204: %v", err)
	}
	if err := c.DeleteTaskList(context.Background(), "t", "gone"); err != nil {
		t.Errorf("404 on delete must be nil (already gone), got %v", err)
	}
	if len(*calls) != 2 || (*calls)[0].Method != http.MethodDelete {
		t.Errorf("calls = %+v", *calls)
	}
}

func TestTasksInsertTaskPostsTitleNotesAndDue(t *testing.T) {
	srv, calls := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{"POST /lists/list_1/tasks": {200, `{"id":"task_1"}`}})
	c := NewHTTPTasksClient()
	c.BaseURL = srv.URL

	due := time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC)
	err := c.InsertTask(context.Background(), "t", "list_1", Task{Title: "Day 1: Greetings", Notes: "3 quests · 30 minutes", Due: due})
	if err != nil {
		t.Fatalf("InsertTask: %v", err)
	}
	b := (*calls)[0].Body
	if b["title"] != "Day 1: Greetings" || b["notes"] != "3 quests · 30 minutes" || b["due"] != "2026-09-02T00:00:00Z" {
		t.Errorf("body = %v", b)
	}
}

func TestTasksInsertTaskMapsReauth(t *testing.T) {
	srv, _ := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{"POST /lists/list_1/tasks": {401, `{"error":{"code":401}}`}})
	c := NewHTTPTasksClient()
	c.BaseURL = srv.URL
	if err := c.InsertTask(context.Background(), "t", "list_1", Task{Title: "x"}); !errors.Is(err, ErrReauthRequired) {
		t.Errorf("err = %v, want ErrReauthRequired", err)
	}
}

func TestTasksMapsAQuota403ToUpstreamNotReauth(t *testing.T) {
	srv, _ := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{"POST /lists/l1/tasks": fakeAnswer(403, `{"error":{"code":403,"errors":[{"domain":"usageLimits","reason":"userRateLimitExceeded"}]}}`)})
	c := NewHTTPTasksClient()
	c.BaseURL = srv.URL
	err := c.InsertTask(context.Background(), "tok", "l1", Task{Title: "Day 1"})
	var up *UpstreamError
	if !errors.As(err, &up) || up.Service != "tasks" || up.Status != 403 {
		t.Fatalf("err = %v, want *UpstreamError{tasks, 403}", err)
	}
	if errors.Is(err, ErrReauthRequired) {
		t.Fatal("a Tasks quota 403 must not force re-consent")
	}
}
