package google

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// DefaultCalendarBaseURL is Google Calendar API v3.
const DefaultCalendarBaseURL = "https://www.googleapis.com/calendar/v3"

// CalendarClient is the two Calendar calls this package makes, on the user's
// primary calendar. InsertEvent returns Google's event id; PatchEvent updates
// the event with that id and returns ErrNotFound when Google no longer has it.
type CalendarClient interface {
	InsertEvent(ctx context.Context, accessToken string, ev Event) (string, error)
	PatchEvent(ctx context.Context, accessToken, eventID string, ev Event) error
}

// HTTPCalendarClient is the real CalendarClient. BaseURL is a field so tests
// point it at an httptest.Server.
type HTTPCalendarClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewHTTPCalendarClient builds a client against the production endpoint.
func NewHTTPCalendarClient() *HTTPCalendarClient {
	return &HTTPCalendarClient{BaseURL: DefaultCalendarBaseURL, HTTPClient: defaultHTTPClient()}
}

func (c *HTTPCalendarClient) InsertEvent(ctx context.Context, accessToken string, ev Event) (string, error) {
	var out struct {
		ID string `json:"id"`
	}
	if err := doJSON(ctx, c.HTTPClient, "calendar", http.MethodPost, c.BaseURL+"/calendars/primary/events", accessToken, ev.payload(), &out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("google: calendar insert returned no id")
	}
	return out.ID, nil
}

func (c *HTTPCalendarClient) PatchEvent(ctx context.Context, accessToken, eventID string, ev Event) error {
	return doJSON(ctx, c.HTTPClient, "calendar", http.MethodPatch, c.BaseURL+"/calendars/primary/events/"+url.PathEscape(eventID), accessToken, ev.payload(), nil)
}

var _ CalendarClient = (*HTTPCalendarClient)(nil)
