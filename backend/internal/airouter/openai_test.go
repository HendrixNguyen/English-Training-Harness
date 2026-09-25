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

func TestOpenAICompatibleSendsTheSpec62RequestAndReturnsTheFirstChoice(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"cefr_level\":\"B1\"}"}}],"usage":{"prompt_tokens":7,"completion_tokens":9,"total_tokens":16}}`))
	}))
	defer srv.Close()

	logs := captureLog(t)
	p := NewOpenAICompatibleProvider(srv.URL+"/v1/", "sk-1", "deepseek-chat", srv.Client())
	out, err := p.GenerateContent(withTask(context.Background(), TaskEssayGrading), "SYS", "USR")
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	if out != `{"cefr_level":"B1"}` {
		t.Errorf("out = %q", out)
	}
	if gotPath != "/v1/chat/completions" {
		t.Errorf("path = %q (trailing slash on the base URL must not double up)", gotPath)
	}
	if gotAuth != "Bearer sk-1" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotBody["model"] != "deepseek-chat" || gotBody["temperature"] != 0.2 {
		t.Errorf("model/temperature = %v/%v", gotBody["model"], gotBody["temperature"])
	}
	if gotBody["response_format"].(map[string]any)["type"] != "json_object" {
		t.Errorf("response_format = %v", gotBody["response_format"])
	}
	msgs := gotBody["messages"].([]any)
	if len(msgs) != 2 || msgs[0].(map[string]any)["role"] != "system" || msgs[0].(map[string]any)["content"] != "SYS" ||
		msgs[1].(map[string]any)["role"] != "user" || msgs[1].(map[string]any)["content"] != "USR" {
		t.Errorf("messages = %v", msgs)
	}
	got := logs.String()
	if !strings.Contains(got, "openai-compat(deepseek-chat) task=essay_grading ok ") || !strings.Contains(got, "tokens prompt=7 completion=9") {
		t.Errorf("log missing usage line:\n%s", got)
	}
	if strings.Contains(got, "sk-1") {
		t.Errorf("the API key reached the log:\n%s", got)
	}
}

func TestOpenAICompatibleRejectsNon2xxEmptyChoicesAndBadJSON(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
		want   string
	}{
		"401":           {401, `{"error":{"message":"bad key"}}`, "status 401"},
		"empty choices": {200, `{"choices":[]}`, "empty"},
		"not json":      {200, `oops`, "decoding"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			p := NewOpenAICompatibleProvider(srv.URL, "k", "m", srv.Client())
			_, err := p.GenerateContent(context.Background(), "s", "u")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestOpenAICompatibleDefaults(t *testing.T) {
	p := NewOpenAICompatibleProvider("", "k", "", nil)
	if p.baseURL != DefaultOpenAIBaseURL || p.model != DefaultOpenAIModel || p.client.Timeout != 0 {
		t.Errorf("defaults = %q %q %v", p.baseURL, p.model, p.client.Timeout)
	}
	if DefaultOpenAIBaseURL != "https://api.openai.com/v1" || DefaultDeepSeekBaseURL != "https://api.deepseek.com/v1" {
		t.Errorf("base URLs = %q %q", DefaultOpenAIBaseURL, DefaultDeepSeekBaseURL)
	}
	if DefaultOpenAIModel != "gpt-4o-mini" || DefaultDeepSeekModel != "deepseek-chat" {
		t.Errorf("models = %q %q (§6.2)", DefaultOpenAIModel, DefaultDeepSeekModel)
	}
}
