package airouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Defaults for the two OpenAI-compatible vendors (§2.1, §6.2).
const (
	DefaultOpenAIBaseURL   = "https://api.openai.com/v1"
	DefaultOpenAIModel     = "gpt-4o-mini"
	DefaultDeepSeekBaseURL = "https://api.deepseek.com/v1"
	DefaultDeepSeekModel   = "deepseek-chat"
)

// OpenAICompatibleProvider speaks the chat/completions dialect shared by
// OpenAI and DeepSeek.
type OpenAICompatibleProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewOpenAICompatibleProvider builds a provider. An empty baseURL/model means
// OpenAI's; DeepSeek callers pass DefaultDeepSeekBaseURL/DefaultDeepSeekModel
// (see config.go).
func NewOpenAICompatibleProvider(baseURL, apiKey, model string, client *http.Client) *OpenAICompatibleProvider {
	if baseURL == "" {
		baseURL = DefaultOpenAIBaseURL
	}
	if model == "" {
		model = DefaultOpenAIModel
	}
	if client == nil {
		client = &http.Client{} // no Timeout: the per-task deadline is in the context (timeouts.go)
	}
	return &OpenAICompatibleProvider{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, model: model, client: client}
}

func (o *OpenAICompatibleProvider) GenerateContent(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := map[string]any{
		"model": o.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.2,
	}
	label := fmt.Sprintf("openai-compat(%s)", o.model)
	started := time.Now()
	body, err := postJSON(ctx, o.client, label, o.baseURL+"/chat/completions", reqBody,
		map[string]string{"Authorization": "Bearer " + o.apiKey})
	if err != nil {
		return "", fmt.Errorf("openai-compat(%s): %w", o.model, err)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("openai-compat(%s): decoding response: %w", o.model, err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("openai-compat(%s): empty choices", o.model)
	}
	logCall(ctx, label, started, parsed.Usage.PromptTokens, parsed.Usage.CompletionTokens)
	return parsed.Choices[0].Message.Content, nil
}

var _ LLMProvider = (*OpenAICompatibleProvider)(nil)
