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
// (invalid_grant), or rejects the request as unauthorized (401) or as a
// permissions/scope problem (403 without a throttling reason). The handler
// answers 409 reauth_required so the client sends the user through /login
// again (auth.AuthCodeURL asks for calendar.events + tasks with prompt=consent).
//
// A 403 whose error reason is one of throttleReasons is *not* reauth: Calendar
// v3 and Tasks v1 answer quota exhaustion with 403, and re-consenting cannot
// fix a quota. Those, and 429, are UpstreamError (→ 502, "try again later").
var ErrReauthRequired = errors.New("google: re-authentication required")

// throttleReasons are the error.errors[].reason values Google uses for quota
// and rate limiting on Calendar v3 and Tasks v1 (they arrive as 403).
var throttleReasons = map[string]bool{
	"rateLimitExceeded":     true,
	"userRateLimitExceeded": true,
	"dailyLimitExceeded":    true,
	"quotaExceeded":         true,
}

// googleErrorReason returns error.errors[0].reason from Google's standard
// error envelope, or "" when the body is not that shape.
func googleErrorReason(raw []byte) string {
	var env struct {
		Error struct {
			Errors []struct {
				Reason string `json:"reason"`
			} `json:"errors"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err != nil || len(env.Error.Errors) == 0 {
		return ""
	}
	return env.Error.Errors[0].Reason
}

// ErrNotFound means the Calendar event or Tasks list we stored an id for no
// longer exists at Google (404/410). Service re-creates it.
var ErrNotFound = errors.New("google: resource not found")

// ErrAlreadyExists means Google already holds a resource with the id we sent.
// Only Calendar events.insert with a client id relies on it (Service patches
// its own event). Any other 409 is not consumed and the handler answers 502.
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
// maps the status code as documented on the errors above (401 and non-throttle
// 403 → ErrReauthRequired; 404/410 → ErrNotFound; 409 → ErrAlreadyExists;
// everything else non-2xx, including throttling 403 and 429 → *UpstreamError),
// and decodes a 2xx body into out when out is non-nil.
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
	case resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("%w: %s returned 401", ErrReauthRequired, service)
	case resp.StatusCode == http.StatusForbidden && !throttleReasons[googleErrorReason(raw)]:
		return fmt.Errorf("%w: %s returned 403 %s", ErrReauthRequired, service, googleErrorReason(raw))
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return fmt.Errorf("%w: %s returned %d", ErrNotFound, service, resp.StatusCode)
	case resp.StatusCode == http.StatusConflict:
		return fmt.Errorf("%w: %s returned 409", ErrAlreadyExists, service)
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		// Includes throttling 403s and 429: the handler answers 502.
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
