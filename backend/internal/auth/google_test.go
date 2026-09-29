package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestExchangeSendsTheAuthorizationCodeGrant(t *testing.T) {
	var gotForm url.Values

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","refresh_token":"rt","expires_in":3599,"token_type":"Bearer"}`))
	}))
	defer tokenSrv.Close()

	c := &GoogleClient{
		ClientID:     "cid",
		ClientSecret: "csecret",
		TokenURL:     tokenSrv.URL,
		UserInfoURL:  "unused",
		HTTPClient:   tokenSrv.Client(),
	}

	tok, err := c.Exchange(context.Background(), "the-code", "https://app.example.com/callback")
	if err != nil {
		t.Fatalf("Exchange() = %v", err)
	}
	if tok.AccessToken != "at" || tok.RefreshToken != "rt" {
		t.Errorf("token = %+v", tok)
	}

	want := map[string]string{
		"grant_type":    "authorization_code",
		"code":          "the-code",
		"redirect_uri":  "https://app.example.com/callback",
		"client_id":     "cid",
		"client_secret": "csecret",
	}
	for k, v := range want {
		if got := gotForm.Get(k); got != v {
			t.Errorf("form[%s] = %q, want %q", k, got, v)
		}
	}
}

func TestExchangeReportsAGoogleError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	c := &GoogleClient{TokenURL: srv.URL, HTTPClient: srv.Client()}

	if _, err := c.Exchange(context.Background(), "bad", "uri"); err == nil {
		t.Fatal("expected an error for a 400 from Google, got nil")
	}
}

func TestUserInfoSendsTheBearerTokenAndParsesTheProfile(t *testing.T) {
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"google-123","email":"a@example.com","name":"A Person"}`))
	}))
	defer srv.Close()

	c := &GoogleClient{UserInfoURL: srv.URL, HTTPClient: srv.Client()}

	p, err := c.UserInfo(context.Background(), "at")
	if err != nil {
		t.Fatalf("UserInfo() = %v", err)
	}
	if gotAuth != "Bearer at" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer at")
	}
	if p.Sub != "google-123" || p.Email != "a@example.com" || p.Name != "A Person" {
		t.Errorf("profile = %+v", p)
	}
}

func TestUserInfoRejectsAProfileWithoutSubOrEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"name":"No IDs"}`))
	}))
	defer srv.Close()

	c := &GoogleClient{UserInfoURL: srv.URL, HTTPClient: srv.Client()}

	if _, err := c.UserInfo(context.Background(), "at"); err == nil {
		t.Fatal("expected an error when sub/email are missing, got nil")
	}
}

func TestExchangeWrapsAGoogle4xxAsRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	c := &GoogleClient{TokenURL: srv.URL, HTTPClient: srv.Client()}

	_, err := c.Exchange(context.Background(), "bad", "uri")
	if err == nil {
		t.Fatal("expected an error for a 400 from Google, got nil")
	}
	if !errors.Is(err, ErrGoogleRejected) {
		t.Errorf("err = %v, want it to wrap ErrGoogleRejected", err)
	}
	if got := err.Error(); !strings.Contains(got, "invalid_grant") {
		t.Errorf("err = %q, want it to contain Google's reason %q", got, "invalid_grant")
	}
}

func TestExchangeDoesNotCallAGoogle5xxRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := &GoogleClient{TokenURL: srv.URL, HTTPClient: srv.Client()}

	_, err := c.Exchange(context.Background(), "code", "uri")
	if err == nil {
		t.Fatal("expected an error for a 502 from Google, got nil")
	}
	if errors.Is(err, ErrGoogleRejected) {
		t.Errorf("err = %v, want it NOT to wrap ErrGoogleRejected (a 502 is ours, not Google's rejection)", err)
	}
}

func TestUserInfoWithoutASubIsRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"email":"a@example.com"}`))
	}))
	defer srv.Close()

	c := &GoogleClient{UserInfoURL: srv.URL, HTTPClient: srv.Client()}

	_, err := c.UserInfo(context.Background(), "at")
	if err == nil {
		t.Fatal("expected an error when sub is missing, got nil")
	}
	if !errors.Is(err, ErrGoogleRejected) {
		t.Errorf("err = %v, want it to wrap ErrGoogleRejected", err)
	}
}
