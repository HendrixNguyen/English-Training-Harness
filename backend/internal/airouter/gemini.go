package airouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ProviderTimeout is the §6.2 HTTP client timeout.
const ProviderTimeout = 30 * time.Second

// Gemini defaults. §6.2's "gemini.api.internal" was a placeholder; this is the
// real Generative Language API. The model is configurable because names churn.
const (
	DefaultGeminiBaseURL = "https://generativelanguage.googleapis.com"
	DefaultGeminiModel   = "gemini-2.5-flash"
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
		client = &http.Client{Timeout: ProviderTimeout}
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
	body, err := postJSON(ctx, g.client, url, reqBody, map[string]string{"x-goog-api-key": g.apiKey})
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
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		PromptFeedback struct {
			BlockReason string `json:"blockReason"`
		} `json:"promptFeedback"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("gemini: decoding response: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		if br := parsed.PromptFeedback.BlockReason; br != "" {
			// The prompt itself was refused: the one fact an operator needs.
			return "", fmt.Errorf("gemini: empty response: blockReason %s", br)
		}
		return "", fmt.Errorf("gemini: empty response")
	}
	// STOP (or absent, on older responses) is the only complete answer. Anything
	// else — MAX_TOKENS, SAFETY, RECITATION, … — would otherwise surface downstream
	// as "ParseRoadmap: unexpected end of JSON input" and burn a paid retry.
	if fr := parsed.Candidates[0].FinishReason; fr != "" && fr != "STOP" {
		return "", fmt.Errorf("gemini: finishReason %s (answer incomplete or refused)", fr)
	}
	var sb strings.Builder
	for _, part := range parsed.Candidates[0].Content.Parts {
		sb.WriteString(part.Text)
	}
	return sb.String(), nil
}

// postJSON is shared by both providers: marshal, POST, require 2xx, return the
// body. Error bodies are truncated so a verbose upstream cannot flood logs.
func postJSON(ctx context.Context, client *http.Client, url string, reqBody any, headers map[string]string) ([]byte, error) {
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(body, 512))
	}
	return body, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}

var _ LLMProvider = (*GeminiProvider)(nil)
