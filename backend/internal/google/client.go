package google

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrReauthRequired means Google no longer honours this user's refresh token
// (invalid_grant) or rejects the scopes (401/403). The handler answers 409
// reauth_required so the client sends the user through /login again
// (auth.AuthCodeURL asks for calendar.events + tasks with prompt=consent).
var ErrReauthRequired = errors.New("google: re-authentication required")

// ErrNotFound means the Calendar event or Tasks list we stored an id for no
// longer exists at Google (404/410). Service re-creates it.
var ErrNotFound = errors.New("google: resource not found")

// ErrAlreadyExists means Google already holds a resource with the id we sent
// (Calendar events.insert with a client id → 409). Sync treats it as ours.
var ErrAlreadyExists = errors.New("google: resource already exists")

// UpstreamError is any other non-2xx from Google. The handler maps it to 502.
type UpstreamError struct {
	Service string // "oauth" | "calendar" | "tasks"
	Status  int
	Body    string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("google: %s returned %d: %s", e.Service, e.Status, e.Body)
}

// defaultHTTPClient bounds every Google call; the handler's overall deadline
// (SyncTimeout) bounds the whole sync.
func defaultHTTPClient() *http.Client { return &http.Client{Timeout: 15 * time.Second} }

// doJSON sends in (JSON-encoded, or nothing when nil) with a bearer token,
// maps the status code as documented on the errors above, and decodes a 2xx
// body into out when out is non-nil.
func doJSON(ctx context.Context, client *http.Client, service, method, url, accessToken string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("google: encoding %s request: %w", service, err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("google: building %s request: %w", service, err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if client == nil {
		client = defaultHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("google: calling %s: %w", service, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("google: reading %s response: %w", service, err)
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%w: %s returned %d", ErrReauthRequired, service, resp.StatusCode)
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return fmt.Errorf("%w: %s returned %d", ErrNotFound, service, resp.StatusCode)
	case resp.StatusCode == http.StatusConflict:
		return fmt.Errorf("%w: %s returned 409", ErrAlreadyExists, service)
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return &UpstreamError{Service: service, Status: resp.StatusCode, Body: string(raw)}
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("google: decoding %s response: %w", service, err)
	}
	return nil
}
