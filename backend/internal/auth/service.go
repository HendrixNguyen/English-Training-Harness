package auth

import (
	"context"
	"fmt"
)

// SignInResult is what the handler serialises.
type SignInResult struct {
	Token string
	User  User
}

// Service performs spec §5.1 steps 1-3: exchange the code, store the refresh
// token, issue a JWT.
type Service struct {
	google   Exchanger
	users    UserRepo
	sessions SessionStore
	tokens   *TokenIssuer
}

// NewService wires the four collaborators.
func NewService(google Exchanger, users UserRepo, sessions SessionStore, tokens *TokenIssuer) *Service {
	return &Service{google: google, users: users, sessions: sessions, tokens: tokens}
}

// SignIn exchanges code for tokens, upserts the user and opens a session.
func (s *Service) SignIn(ctx context.Context, code, redirectURI string) (SignInResult, error) {
	tok, err := s.google.Exchange(ctx, code, redirectURI)
	if err != nil {
		return SignInResult{}, fmt.Errorf("auth: exchanging code: %w", err)
	}
	profile, err := s.google.UserInfo(ctx, tok.AccessToken)
	if err != nil {
		return SignInResult{}, fmt.Errorf("auth: fetching profile: %w", err)
	}

	user, err := s.users.UpsertByGoogleID(ctx, profile.Sub, profile.Email, profile.Name, tok.RefreshToken)
	if err != nil {
		return SignInResult{}, fmt.Errorf("auth: upserting google_id %s: %w", profile.Sub, err)
	}

	jwtToken, err := s.tokens.Issue(user.ID)
	if err != nil {
		return SignInResult{}, err
	}
	// The Redis key is written last: a failure here must not leave a user
	// holding a token with no session behind it.
	if err := s.sessions.Put(ctx, user.ID, jwtToken, TokenTTL); err != nil {
		return SignInResult{}, err
	}
	return SignInResult{Token: jwtToken, User: user}, nil
}
