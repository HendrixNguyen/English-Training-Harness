package airouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiSendsTheSpec62RequestAndReturnsTheFirstPart(t *testing.T) {
	var gotPath, gotKey, gotCT string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotKey, gotCT = r.URL.Path, r.Header.Get("x-goog-api-key"), r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"ok\":true}"}]}}]}`))
	}))
	defer srv.Close()

	p := NewGeminiProvider("k-123", srv.URL, "gemini-2.5-flash", srv.Client())
	out, err := p.GenerateContent(context.Background(), "SYS", "USR")
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	if out != `{"ok":true}` {
		t.Errorf("out = %q", out)
	}
	if gotPath != "/v1beta/models/gemini-2.5-flash:generateContent" {
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
}

func TestGeminiRejectsNon2xxEmptyCandidatesAndBadJSON(t *testing.T) {
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
	if p.client.Timeout != ProviderTimeout {
		t.Errorf("timeout = %v, want %v (§6.2: 30s)", p.client.Timeout, ProviderTimeout)
	}
}
