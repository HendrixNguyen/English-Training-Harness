package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DefaultTokenURL is Google's OAuth token endpoint (same as auth.DefaultTokenURL).
const DefaultTokenURL = "https://oauth2.googleapis.com/token"

// TokenRefresher swaps a refresh token for a short-lived access token.
type TokenRefresher interface {
	AccessToken(ctx context.Context, refreshToken string) (string, error)
}

// OAuthClient is the real TokenRefresher. TokenURL is a field so tests point
// it at an httptest.Server (the auth.GoogleClient pattern).
type OAuthClient struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	HTTPClient   *http.Client
}

// NewOAuthClient builds a client against the production endpoint.
func NewOAuthClient(clientID, clientSecret string) *OAuthClient {
	return &OAuthClient{ClientID: clientID, ClientSecret: clientSecret, TokenURL: DefaultTokenURL, HTTPClient: defaultHTTPClient()}
}

// AccessToken performs grant_type=refresh_token. A 400 whose body names
// invalid_grant is Google's "revoked or expired refresh token" signal and
// becomes ErrReauthRequired; other failures are *UpstreamError.
func (c *OAuthClient) AccessToken(ctx context.Context, refreshToken string) (string, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("google: building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := c.HTTPClient
	if client == nil {
		client = defaultHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("google: calling token endpoint: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("google: reading token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var body struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &body)
		if body.Error == "invalid_grant" || resp.StatusCode == http.StatusUnauthorized {
			return "", fmt.Errorf("%w: token endpoint returned %d %s", ErrReauthRequired, resp.StatusCode, body.Error)
		}
		return "", &UpstreamError{Service: "oauth", Status: resp.StatusCode, Body: string(raw)}
	}

	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &tok); err != nil {
		return "", fmt.Errorf("google: decoding token response: %w", err)
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("google: token endpoint returned no access token")
	}
	return tok.AccessToken, nil
}

var _ TokenRefresher = (*OAuthClient)(nil)
