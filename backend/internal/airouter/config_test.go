package airouter

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func lookup(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestConfigFromEnvReadsTheSpec9VariablesAndDefaults(t *testing.T) {
	cfg := ConfigFromEnv(lookup(map[string]string{
		"GEMINI_API_KEY":    "g",
		"OPENAI_API_KEY":    "o",
		"DEEPSEEK_API_KEY":  "d",
		"DEEPSEEK_BASE_URL": "http://ds.local/v1",
		"GEMINI_MODEL":      "gemini-2.5-pro",
	}))
	want := Config{
		GeminiAPIKey: "g", GeminiBaseURL: DefaultGeminiBaseURL, GeminiModel: "gemini-2.5-pro",
		OpenAIAPIKey: "o", OpenAIBaseURL: DefaultOpenAIBaseURL, OpenAIModel: DefaultOpenAIModel,
		DeepSeekAPIKey: "d", DeepSeekBaseURL: "http://ds.local/v1", DeepSeekModel: DefaultDeepSeekModel,
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("ConfigFromEnv =\n%+v\nwant\n%+v", cfg, want)
	}
}

func TestNewRouterRegistersOnlyProvidersWithAKey(t *testing.T) {
	cases := []struct {
		env  map[string]string
		want []ProviderType
	}{
		{map[string]string{}, nil},
		{map[string]string{"GEMINI_API_KEY": "g"}, []ProviderType{ProviderGemini}},
		{map[string]string{"DEEPSEEK_API_KEY": "d", "OPENAI_BASE_URL": "http://x"}, []ProviderType{ProviderDeepSeek}}, // a base URL alone registers nothing
		{map[string]string{"GEMINI_API_KEY": "g", "OPENAI_API_KEY": "o", "DEEPSEEK_API_KEY": "d"}, []ProviderType{ProviderGemini, ProviderOpenAI, ProviderDeepSeek}},
	}
	for _, tc := range cases {
		r := NewRouter(ConfigFromEnv(lookup(tc.env)))
		got := r.Providers()
		if len(got) == 0 && len(tc.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("env %v → Providers() = %v, want %v", tc.env, got, tc.want)
		}
	}
}

func TestNewRouterWithNoKeysBootsButRoutesToErrNoProviders(t *testing.T) {
	r := NewRouter(ConfigFromEnv(lookup(nil)))
	if r == nil {
		t.Fatal("NewRouter returned nil; the binary must boot without AI keys")
	}
	if _, err := r.Route(context.Background(), TaskRoadmapGen, "s", "u"); !errors.Is(err, ErrNoProviders) {
		t.Errorf("err = %v, want ErrNoProviders", err)
	}
}

func TestNewRouterWiresDeepSeekOntoTheOpenAIDialect(t *testing.T) {
	r := NewRouter(Config{DeepSeekAPIKey: "d", DeepSeekBaseURL: "http://ds", DeepSeekModel: "deepseek-chat"})
	p, ok := r.providers[ProviderDeepSeek].(*OpenAICompatibleProvider)
	if !ok {
		t.Fatalf("deepseek provider is %T, want *OpenAICompatibleProvider", r.providers[ProviderDeepSeek])
	}
	if p.baseURL != "http://ds" || p.model != "deepseek-chat" || p.apiKey != "d" {
		t.Errorf("deepseek = %+v", p)
	}
}
