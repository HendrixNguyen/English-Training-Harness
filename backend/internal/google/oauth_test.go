package google

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOAuthClientRefreshesAnAccessToken(t *testing.T) {
	var gotForm map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		gotForm = map[string]string{}
		for k := range r.PostForm {
			gotForm[k] = r.PostForm.Get(k)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ya29.new","expires_in":3599,"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	c := NewOAuthClient("cid", "secret")
	c.TokenURL = srv.URL

	tok, err := c.AccessToken(context.Background(), "1//refresh")
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if tok != "ya29.new" {
		t.Errorf("token = %q", tok)
	}
	want := map[string]string{"grant_type": "refresh_token", "refresh_token": "1//refresh", "client_id": "cid", "client_secret": "secret"}
	for k, v := range want {
		if gotForm[k] != v {
			t.Errorf("form[%s] = %q, want %q", k, gotForm[k], v)
		}
	}
}

func TestOAuthClientMapsInvalidGrantToReauthRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Token has been expired or revoked."}`))
	}))
	defer srv.Close()

	c := NewOAuthClient("cid", "secret")
	c.TokenURL = srv.URL
	_, err := c.AccessToken(context.Background(), "1//dead")
	if !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("err = %v, want ErrReauthRequired", err)
	}
}

func TestOAuthClientReportsOtherFailuresAsUpstream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal_failure"}`))
	}))
	defer srv.Close()

	c := NewOAuthClient("cid", "secret")
	c.TokenURL = srv.URL
	_, err := c.AccessToken(context.Background(), "1//x")
	var up *UpstreamError
	if !errors.As(err, &up) || up.Status != 500 || up.Service != "oauth" {
		t.Fatalf("err = %v, want *UpstreamError{Service: oauth, Status: 500}", err)
	}
	if errors.Is(err, ErrReauthRequired) {
		t.Error("a 500 is not a re-auth condition")
	}
}

func TestOAuthClientRejectsAnEmptyAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	c := NewOAuthClient("cid", "secret")
	c.TokenURL = srv.URL
	if _, err := c.AccessToken(context.Background(), "1//x"); err == nil {
		t.Fatal("expected an error when access_token is missing")
	}
}
