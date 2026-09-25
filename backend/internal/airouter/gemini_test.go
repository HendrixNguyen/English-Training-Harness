package airouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGeminiSendsTheSpec62RequestAndReturnsTheText(t *testing.T) {
	var gotPath, gotKey, gotCT string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotKey, gotCT = r.URL.Path, r.Header.Get("x-goog-api-key"), r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"ok\":true}"}]}}]}`))
	}))
	defer srv.Close()

	p := NewGeminiProvider("k-123", srv.URL, "gemini-3.8-flash", srv.Client())
	out, err := p.GenerateContent(context.Background(), "SYS", "USR")
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	if out != `{"ok":true}` {
		t.Errorf("out = %q", out)
	}
	if gotPath != "/v1beta/models/gemini-3.8-flash:generateContent" {
		t.Errorf("path = %q", gotPath)
	}
	if gotKey != "k-123" || gotCT != "application/json" {
		t.Errorf("headers: key=%q content-type=%q", gotKey, gotCT)
	}
	if strings.Contains(srv.URL+gotPath, "key=") {
		t.Error("API key must travel in a header, not the query string")
	}

	sys := gotBody["system_instruction"].(map[string]any)["parts"].([]any)[0].(map[string]any)["text"]
	if sys != "SYS" {
		t.Errorf("system_instruction text = %v", sys)
	}
	content := gotBody["contents"].([]any)[0].(map[string]any)
	if content["role"] != "user" || content["parts"].([]any)[0].(map[string]any)["text"] != "USR" {
		t.Errorf("contents[0] = %v", content)
	}
	gen := gotBody["generationConfig"].(map[string]any)
	if gen["response_mime_type"] != "application/json" || gen["temperature"] != 0.2 {
		t.Errorf("generationConfig = %v, want application/json and 0.2", gen)
	}
	if gen["maxOutputTokens"] != float64(GeminiMaxOutputTokens) || GeminiMaxOutputTokens < 16384 {
		t.Errorf("maxOutputTokens = %v, want %d (≥ 16384: 84 tasks with content)", gen["maxOutputTokens"], GeminiMaxOutputTokens)
	}
}

func TestGeminiJoinsEveryPartOfTheFirstCandidate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// The Generative Language API may split one candidate's answer across
		// parts; a roadmap split mid-object is not JSON unless re-joined.
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"a\":1,"},{"text":"\"b\":2}"}]},"finishReason":"STOP"}]}`))
	}))
	defer srv.Close()
	p := NewGeminiProvider("k", srv.URL, "m", srv.Client())
	out, err := p.GenerateContent(context.Background(), "s", "u")
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	if out != `{"a":1,"b":2}` {
		t.Fatalf("out = %q, want the two parts concatenated in order", out)
	}
}

func TestGeminiNamesANonStopFinishReason(t *testing.T) {
	cases := map[string]struct {
		body    string
		wantErr string // "" = success
	}{
		"no finishReason (older responses)": {`{"candidates":[{"content":{"parts":[{"text":"{}"}]}}]}`, ""},
		"STOP":                              {`{"candidates":[{"content":{"parts":[{"text":"{}"}]},"finishReason":"STOP"}]}`, ""},
		"MAX_TOKENS with partial text":      {`{"candidates":[{"content":{"parts":[{"text":"{\"title\":\"Road"}]},"finishReason":"MAX_TOKENS"}]}`, "MAX_TOKENS"},
		"SAFETY":                            {`{"candidates":[{"content":{"parts":[{"text":""}]},"finishReason":"SAFETY"}]}`, "SAFETY"},
		"RECITATION":                        {`{"candidates":[{"content":{"parts":[{"text":"x"}]},"finishReason":"RECITATION"}]}`, "RECITATION"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(tc.body)) }))
			defer srv.Close()
			p := NewGeminiProvider("k", srv.URL, "m", srv.Client())
			out, err := p.GenerateContent(context.Background(), "s", "u")
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("err = %v, want success", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) || !strings.Contains(err.Error(), "finishReason") {
				t.Fatalf("err = %v (out %q), want an error naming finishReason %s", err, out, tc.wantErr)
			}
		})
	}
}

func TestGeminiRejectsNon2xxEmptyCandidatesAndBadJSON(t *testing.T) {
	noBackoff(t)
	cases := map[string]struct {
		status int
		body   string
		want   string
	}{
		"500":              {500, `{"error":"boom"}`, "status 500"},
		"429":              {429, `rate`, "status 429"},
		"empty candidates": {200, `{"candidates":[]}`, "empty"},
		"empty parts":      {200, `{"candidates":[{"content":{"parts":[]}}]}`, "empty"},
		"not json":         {200, `<html>`, "decoding"},
		"prompt blocked":   {200, `{"promptFeedback":{"blockReason":"SAFETY","safetyRatings":[]},"candidates":[]}`, "blockReason SAFETY"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			p := NewGeminiProvider("k", srv.URL, "m", srv.Client())
			_, err := p.GenerateContent(context.Background(), "s", "u")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestGeminiDefaultsToTheRealEndpoint(t *testing.T) {
	p := NewGeminiProvider("k", "", "", nil)
	if p.baseURL != DefaultGeminiBaseURL || p.model != DefaultGeminiModel || p.client == nil {
		t.Errorf("defaults = %q %q client=%v", p.baseURL, p.model, p.client)
	}
	if DefaultGeminiBaseURL != "https://generativelanguage.googleapis.com" {
		t.Errorf("DefaultGeminiBaseURL = %q", DefaultGeminiBaseURL)
	}
	if p.client.Timeout != 0 {
		t.Errorf("client.Timeout = %v, want 0: the per-task deadline travels in the context (timeouts.go)", p.client.Timeout)
	}
	if DefaultGeminiModel != "gemini-3.8-flash" {
		t.Errorf("DefaultGeminiModel = %q; gemini-2.5-flash is retired for new accounts (404, 2026-09-25)", DefaultGeminiModel)
	}
}

// captureLog routes the standard logger into a buffer for one test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

func noBackoff(t *testing.T) {
	t.Helper()
	prev := retryBackoff
	retryBackoff = 0
	t.Cleanup(func() { retryBackoff = prev })
}

const geminiOK = `{"candidates":[{"content":{"parts":[{"text":"ok"}]}}],"usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":34,"totalTokenCount":46}}`

func TestGeminiRetriesOnceOn503AndLogsElapsedAndTokens(t *testing.T) {
	noBackoff(t)
	logs := captureLog(t)
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"code":503,"message":"This model is currently experiencing high demand. Spikes in demand are usually temporary. Please try again later.","status":"UNAVAILABLE"}}`))
			return
		}
		_, _ = w.Write([]byte(geminiOK))
	}))
	defer srv.Close()

	p := NewGeminiProvider("k-123", srv.URL, "gemini-3.8-flash", srv.Client())
	out, err := p.GenerateContent(withTask(context.Background(), TaskPlacementTest), "s", "u")
	if err != nil || out != "ok" {
		t.Fatalf("GenerateContent = %q, %v; want ok after one retry", out, err)
	}
	if calls != 2 {
		t.Errorf("upstream calls = %d, want 2", calls)
	}
	got := logs.String()
	for _, want := range []string{"retrying once", "status 503", "high demand", "gemini(gemini-3.8-flash) task=placement_test ok ", "tokens prompt=12 completion=34"} {
		if !strings.Contains(got, want) {
			t.Errorf("log missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "k-123") {
		t.Errorf("the API key reached the log:\n%s", got)
	}
}

func TestGeminiDoesNotRetryA404AndReportsGooglesMessageTruncated(t *testing.T) {
	noBackoff(t)
	var calls int32
	msg := "This model models/gemini-2.5-flash is no longer available to new users. Please update your code to use models/gemini-3.8-flash for the latest features and improvements. We recommend you to use the Interactions API " + strings.Repeat("x", 300)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":404,"message":"` + msg + `","status":"NOT_FOUND"}}`))
	}))
	defer srv.Close()

	p := NewGeminiProvider("k", srv.URL, "gemini-2.5-flash", srv.Client())
	_, err := p.GenerateContent(context.Background(), "s", "u")
	if err == nil || !strings.Contains(err.Error(), "status 404") || !strings.Contains(err.Error(), "no longer available to new users") {
		t.Fatalf("err = %v, want status 404 with Google's message", err)
	}
	if strings.Contains(err.Error(), "xxxxxxxxxx") || !strings.Contains(err.Error(), "…") {
		t.Errorf("error body not cut at %d chars: %v", errorBodyChars, err)
	}
	if calls != 1 {
		t.Errorf("upstream calls = %d, want 1 (404 is not retried)", calls)
	}
}

func TestGeminiGivesUpAfterTheSecond503(t *testing.T) {
	noBackoff(t)
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"status":"UNAVAILABLE"}}`))
	}))
	defer srv.Close()
	p := NewGeminiProvider("k", srv.URL, "m", srv.Client())
	if _, err := p.GenerateContent(context.Background(), "s", "u"); err == nil || !strings.Contains(err.Error(), "status 503") {
		t.Fatalf("err = %v, want status 503", err)
	}
	if calls != 2 {
		t.Errorf("upstream calls = %d, want exactly 2", calls)
	}
}

func TestGeminiRetryBackoffRespectsTheDeadline(t *testing.T) {
	prev := retryBackoff
	retryBackoff = time.Minute
	t.Cleanup(func() { retryBackoff = prev })
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	p := NewGeminiProvider("k", srv.URL, "m", srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := p.GenerateContent(ctx, "s", "u")
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "status 503") {
		t.Fatalf("err = %v, want DeadlineExceeded that still names the 503", err)
	}
	if time.Since(started) > time.Second || calls != 1 {
		t.Errorf("took %s with %d calls; the backoff must end with the context", time.Since(started), calls)
	}
}
