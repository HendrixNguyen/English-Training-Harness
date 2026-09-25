package airouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Gemini defaults. §6.2's "gemini.api.internal" was a placeholder; this is the
// real Generative Language API. The model is configurable (GEMINI_MODEL)
// because names churn: gemini-2.5-flash answers 404 "no longer available to
// new users" since 2026-09; Google points at gemini-3.8-flash.
const (
	DefaultGeminiBaseURL = "https://generativelanguage.googleapis.com"
	DefaultGeminiModel   = "gemini-3.8-flash"
)

// GeminiProvider calls models/{model}:generateContent in JSON mode.
type GeminiProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewGeminiProvider builds a provider. Empty baseURL/model/client take the
// defaults; tests pass an httptest server URL.
func NewGeminiProvider(apiKey, baseURL, model string, client *http.Client) *GeminiProvider {
	if baseURL == "" {
		baseURL = DefaultGeminiBaseURL
	}
	if model == "" {
		model = DefaultGeminiModel
	}
	if client == nil {
		client = &http.Client{} // no Timeout: the per-task deadline is in the context (timeouts.go)
	}
	return &GeminiProvider{apiKey: apiKey, baseURL: strings.TrimRight(baseURL, "/"), model: model, client: client}
}

func (g *GeminiProvider) GenerateContent(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", g.baseURL, g.model)
	reqBody := map[string]any{
		"system_instruction": map[string]any{"parts": []map[string]string{{"text": systemPrompt}}},
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]string{{"text": userPrompt}}},
		},
		"generationConfig": map[string]any{
			"response_mime_type": "application/json",
			"temperature":        0.2,
		},
	}
	label := fmt.Sprintf("gemini(%s)", g.model)
	started := time.Now()
	body, err := postJSON(ctx, g.client, label, url, reqBody, map[string]string{"x-goog-api-key": g.apiKey})
	if err != nil {
		return "", fmt.Errorf("gemini: %w", err)
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("gemini: decoding response: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}
	logCall(ctx, label, started, parsed.UsageMetadata.PromptTokenCount, parsed.UsageMetadata.CandidatesTokenCount)
	return parsed.Candidates[0].Content.Parts[0].Text, nil
}

// errorBodyChars bounds how much of an upstream error body reaches errors
// and logs: enough for Google's "model retired" sentence, never a flood.
const errorBodyChars = 200

// retryBackoff is the pause before the single retry of a 429/502/503/504
// answer ("Spikes in demand are usually temporary"). A var so tests need not wait.
var retryBackoff = 2 * time.Second

func retryable(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// postJSON is shared by both providers: marshal, POST, require 2xx, return the
// body. A retryable status is tried once more after retryBackoff, still under
// ctx; any other failure is returned with the status and a truncated body.
// label names the provider and model in log lines and never includes the key.
func postJSON(ctx context.Context, client *http.Client, label, url string, reqBody any, headers map[string]string) ([]byte, error) {
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	for attempt := 0; ; attempt++ {
		body, status, err := doJSON(ctx, client, url, raw, headers)
		if err != nil {
			return nil, err
		}
		if status >= 200 && status <= 299 {
			return body, nil
		}
		upstream := fmt.Errorf("status %d: %s", status, truncate(body, errorBodyChars))
		if attempt > 0 || !retryable(status) {
			return nil, upstream
		}
		log.Printf("airouter: %s: %v; retrying once in %s", label, upstream, retryBackoff)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("%w while waiting to retry: %v", ctx.Err(), upstream)
		case <-time.After(retryBackoff):
		}
	}
}

// doJSON is one POST: transport errors are returned, any status is reported.
func doJSON(ctx context.Context, client *http.Client, url string, raw []byte, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, 0, fmt.Errorf("read body: %w", err)
	}
	return body, resp.StatusCode, nil
}

// logCall is the one line an operator reads per successful call.
func logCall(ctx context.Context, label string, started time.Time, promptTokens, completionTokens int) {
	log.Printf("airouter: %s task=%s ok %.1fs tokens prompt=%d completion=%d", label, taskFrom(ctx), time.Since(started).Seconds(), promptTokens, completionTokens)
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}

var _ LLMProvider = (*GeminiProvider)(nil)
