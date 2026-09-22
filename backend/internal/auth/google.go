package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Default Google endpoints. They are fields on GoogleClient so tests can point
// at an httptest.Server — this slice never calls Google in `go test ./...`.
const (
	DefaultTokenURL    = "https://oauth2.googleapis.com/token"
	DefaultUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
)

// GoogleToken is the subset of Google's token response this product uses.
// RefreshToken is empty on re-consent unless prompt=consent was requested — see
// AuthCodeURL in scopes.go.
type GoogleToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// GoogleProfile is the OpenID userinfo payload (spec §5.1 step 2).
type GoogleProfile struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Exchanger is the Google side of sign-in, so Service can be tested with a fake.
type Exchanger interface {
	Exchange(ctx context.Context, code, redirectURI string) (GoogleToken, error)
	UserInfo(ctx context.Context, accessToken string) (GoogleProfile, error)
}

// GoogleClient is the real Exchanger.
type GoogleClient struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	UserInfoURL  string
	HTTPClient   *http.Client
}

// NewGoogleClient builds a client against the production endpoints.
func NewGoogleClient(clientID, clientSecret string) *GoogleClient {
	return &GoogleClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     DefaultTokenURL,
		UserInfoURL:  DefaultUserInfoURL,
		HTTPClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *GoogleClient) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// Exchange swaps an authorization code for tokens (spec §5.1 step 2).
func (c *GoogleClient) Exchange(ctx context.Context, code, redirectURI string) (GoogleToken, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return GoogleToken{}, fmt.Errorf("auth: building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var tok GoogleToken
	if err := c.doJSON(req, &tok); err != nil {
		return GoogleToken{}, err
	}
	if tok.AccessToken == "" {
		return GoogleToken{}, fmt.Errorf("auth: Google returned no access token")
	}
	return tok, nil
}

// UserInfo fetches sub/email/name for an access token.
func (c *GoogleClient) UserInfo(ctx context.Context, accessToken string) (GoogleProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.UserInfoURL, nil)
	if err != nil {
		return GoogleProfile{}, fmt.Errorf("auth: building userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	var p GoogleProfile
	if err := c.doJSON(req, &p); err != nil {
		return GoogleProfile{}, err
	}
	if p.Sub == "" || p.Email == "" {
		return GoogleProfile{}, fmt.Errorf("auth: Google profile is missing sub or email")
	}
	return p, nil
}

func (c *GoogleClient) doJSON(req *http.Request, out any) error {
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("auth: calling %s: %w", req.URL.Host, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("auth: reading %s response: %w", req.URL.Host, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("auth: %s returned %d: %s", req.URL.Host, resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("auth: decoding %s response: %w", req.URL.Host, err)
	}
	return nil
}

var _ Exchanger = (*GoogleClient)(nil)
