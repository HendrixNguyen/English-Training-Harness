package google

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrReauthRequired means Google no longer honours this user's refresh token
// (invalid_grant), or rejects the request as unauthorized (401), or as a
// scope/permission problem (403 whose reason is in reauthReasons, or a 403
// with no parseable reason at all — the legacy shape). The handler answers
// 409 reauth_required so the client sends the user through /login again
// (auth.AuthCodeURL asks for calendar.events + tasks with prompt=consent).
//
// Every other parsed 403 reason — the quota family, accessNotConfigured /
// SERVICE_DISABLED (the Calendar or Tasks API is off in the GCP project),
// domainPolicy, … — is *not* reauth: re-consenting cannot fix any of them.
// Those, and 429, are UpstreamError (→ 502, "try again later").
var ErrReauthRequired = errors.New("google: re-authentication required")

// reauthReasons are the only reasons re-consent can fix: error.errors[].reason
// on Calendar v3 / Tasks v1, and the google.rpc.ErrorInfo detail reason Google
// uses for the same problem in the newer status+details error shape.
var reauthReasons = map[string]bool{
	"insufficientPermissions":         true,
	"forbidden":                       true,
	"ACCESS_TOKEN_SCOPE_INSUFFICIENT": true,
}

// googleErrorReasons returns every non-empty reason from Google's error
// envelope — error.errors[].reason first, then error.details[].reason (the
// google.rpc.ErrorInfo shape) — in order, or nil when the body is not that
// shape (including the legacy HTML/plain-text 403).
func googleErrorReasons(raw []byte) []string {
	var env struct {
		Error struct {
			Errors []struct {
				Reason string `json:"reason"`
			} `json:"errors"`
			Details []struct {
				Reason string `json:"reason"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil
	}
	var reasons []string
	for _, e := range env.Error.Errors {
		if e.Reason != "" {
			reasons = append(reasons, e.Reason)
		}
	}
	for _, d := range env.Error.Details {
		if d.Reason != "" {
			reasons = append(reasons, d.Reason)
		}
	}
	return reasons
}

// isReauth403 decides whether a 403 body should send the user through
// re-consent: no parseable reason at all (empty body, HTML 403 — the legacy
// shape), or any parsed reason is a scope/permission reason.
func isReauth403(raw []byte) bool {
	reasons := googleErrorReasons(raw)
	if len(reasons) == 0 {
		return true
	}
	for _, r := range reasons {
		if reauthReasons[r] {
			return true
		}
	}
	return false
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
// maps the status code as documented on the errors above (401 and a
// scope/permission (or unparseable) 403 → ErrReauthRequired; 404/410 →
// ErrNotFound; 409 → ErrAlreadyExists; everything else non-2xx, including a
// non-auth-reason 403 (quota, API disabled, domain policy, …) and 429 →
// *UpstreamError), and decodes a 2xx body into out when out is non-nil.
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
	case resp.StatusCode == http.StatusForbidden && isReauth403(raw):
		return fmt.Errorf("%w: %s returned 403 %s", ErrReauthRequired, service, strings.Join(googleErrorReasons(raw), ","))
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return fmt.Errorf("%w: %s returned %d", ErrNotFound, service, resp.StatusCode)
	case resp.StatusCode == http.StatusConflict:
		return fmt.Errorf("%w: %s returned 409", ErrAlreadyExists, service)
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		// Includes every 403 with a known non-auth reason — quota, API
		// disabled, domain policy — and 429: the handler answers 502.
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
