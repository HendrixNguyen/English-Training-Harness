package google

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DefaultTasksBaseURL is Google Tasks API v1.
const DefaultTasksBaseURL = "https://tasks.googleapis.com/tasks/v1"

// Task is one Google Tasks item: one per roadmap day.
type Task struct {
	Title string
	Notes string
	Due   time.Time // UTC midnight of the due date; Google keeps only the date
}

func (t Task) payload() map[string]any {
	p := map[string]any{"title": t.Title, "notes": t.Notes}
	if !t.Due.IsZero() {
		p["due"] = t.Due.UTC().Format(time.RFC3339)
	}
	return p
}

// TasksClient is the three Tasks calls this package makes. DeleteTaskList
// returns nil when the list is already gone.
type TasksClient interface {
	InsertTaskList(ctx context.Context, accessToken, title string) (string, error)
	DeleteTaskList(ctx context.Context, accessToken, tasklistID string) error
	InsertTask(ctx context.Context, accessToken, tasklistID string, t Task) error
}

// HTTPTasksClient is the real TasksClient.
type HTTPTasksClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewHTTPTasksClient builds a client against the production endpoint.
func NewHTTPTasksClient() *HTTPTasksClient {
	return &HTTPTasksClient{BaseURL: DefaultTasksBaseURL, HTTPClient: defaultHTTPClient()}
}

func (c *HTTPTasksClient) InsertTaskList(ctx context.Context, accessToken, title string) (string, error) {
	var out struct {
		ID string `json:"id"`
	}
	if err := doJSON(ctx, c.HTTPClient, "tasks", http.MethodPost, c.BaseURL+"/users/@me/lists", accessToken, map[string]string{"title": title}, &out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("google: tasklist insert returned no id")
	}
	return out.ID, nil
}

func (c *HTTPTasksClient) DeleteTaskList(ctx context.Context, accessToken, tasklistID string) error {
	err := doJSON(ctx, c.HTTPClient, "tasks", http.MethodDelete, c.BaseURL+"/users/@me/lists/"+url.PathEscape(tasklistID), accessToken, nil, nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (c *HTTPTasksClient) InsertTask(ctx context.Context, accessToken, tasklistID string, t Task) error {
	return doJSON(ctx, c.HTTPClient, "tasks", http.MethodPost, c.BaseURL+"/lists/"+url.PathEscape(tasklistID)+"/tasks", accessToken, t.payload(), nil)
}

var _ TasksClient = (*HTTPTasksClient)(nil)
