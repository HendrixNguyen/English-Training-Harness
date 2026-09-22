package airouter

import "net/http"

// Config is everything NewRouter needs. Keys are the backend spec §9 / §8
// variables; base URLs and models are optional overrides (tests, proxies,
// model churn). It lives here rather than in internal/config so the package
// is testable with a plain lookup func and config.Load's "required variables"
// contract is unchanged — AI keys are optional.
type Config struct {
	GeminiAPIKey  string
	GeminiBaseURL string
	GeminiModel   string

	OpenAIAPIKey  string
	OpenAIBaseURL string
	OpenAIModel   string

	DeepSeekAPIKey  string
	DeepSeekBaseURL string
	DeepSeekModel   string

	// HTTPClient, when set, is shared by every provider (tests).
	HTTPClient *http.Client
}

// ConfigFromEnv reads the provider variables through lookup (os.Getenv in
// main) and fills defaults for every optional value.
func ConfigFromEnv(lookup func(string) string) Config {
	or := func(v, def string) string {
		if v == "" {
			return def
		}
		return v
	}
	return Config{
		GeminiAPIKey:  lookup("GEMINI_API_KEY"),
		GeminiBaseURL: or(lookup("GEMINI_BASE_URL"), DefaultGeminiBaseURL),
		GeminiModel:   or(lookup("GEMINI_MODEL"), DefaultGeminiModel),

		OpenAIAPIKey:  lookup("OPENAI_API_KEY"),
		OpenAIBaseURL: or(lookup("OPENAI_BASE_URL"), DefaultOpenAIBaseURL),
		OpenAIModel:   or(lookup("OPENAI_MODEL"), DefaultOpenAIModel),

		DeepSeekAPIKey:  lookup("DEEPSEEK_API_KEY"),
		DeepSeekBaseURL: or(lookup("DEEPSEEK_BASE_URL"), DefaultDeepSeekBaseURL),
		DeepSeekModel:   or(lookup("DEEPSEEK_MODEL"), DefaultDeepSeekModel),
	}
}

// NewRouter registers one provider per configured API key (§6.2 NewRouter).
// Unlike the pseudocode it never fails: with no keys the router boots and
// Route returns ErrNoProviders, so a developer without AI credentials can
// still run every non-AI route.
func NewRouter(cfg Config) *Router {
	providers := map[ProviderType]LLMProvider{}
	if cfg.GeminiAPIKey != "" {
		providers[ProviderGemini] = NewGeminiProvider(cfg.GeminiAPIKey, cfg.GeminiBaseURL, cfg.GeminiModel, cfg.HTTPClient)
	}
	if cfg.OpenAIAPIKey != "" {
		providers[ProviderOpenAI] = NewOpenAICompatibleProvider(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.HTTPClient)
	}
	if cfg.DeepSeekAPIKey != "" {
		base, model := cfg.DeepSeekBaseURL, cfg.DeepSeekModel
		if base == "" {
			base = DefaultDeepSeekBaseURL
		}
		if model == "" {
			model = DefaultDeepSeekModel
		}
		providers[ProviderDeepSeek] = NewOpenAICompatibleProvider(base, cfg.DeepSeekAPIKey, model, cfg.HTTPClient)
	}
	return NewRouterWithProviders(providers)
}
