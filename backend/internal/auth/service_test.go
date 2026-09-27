package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeExchanger struct {
	token   GoogleToken
	profile GoogleProfile
	err     error

	gotCode, gotRedirect, gotAccessToken string
}

func (f *fakeExchanger) Exchange(_ context.Context, code, redirectURI string) (GoogleToken, error) {
	f.gotCode, f.gotRedirect = code, redirectURI
	return f.token, f.err
}

func (f *fakeExchanger) UserInfo(_ context.Context, accessToken string) (GoogleProfile, error) {
	f.gotAccessToken = accessToken
	return f.profile, f.err
}

type fakeRepo struct {
	users map[string]User // keyed by google_id
	last  struct{ googleID, email, fullName, refreshToken string }
	err   error
}

func newFakeRepo() *fakeRepo { return &fakeRepo{users: map[string]User{}} }

func (f *fakeRepo) UpsertByGoogleID(_ context.Context, googleID, email, fullName, refreshToken string) (User, error) {
	if f.err != nil {
		return User{}, f.err
	}
	f.last.googleID, f.last.email, f.last.fullName, f.last.refreshToken = googleID, email, fullName, refreshToken

	u, ok := f.users[googleID]
	if !ok {
		u = User{ID: "id-" + googleID, CEFRCurrent: "A1"} // a fresh row's §3.2 default
	}
	u.Email, u.FullName = email, fullName
	f.users[googleID] = u
	return u, nil
}

func newTestService(ex *fakeExchanger, repo *fakeRepo, sess *fakeSessions) *Service {
	return NewService(ex, repo, sess, NewTokenIssuer("secret", func() time.Time { return time.Unix(1_800_000_000, 0) }))
}

func TestSignInCreatesANewUserAndASession(t *testing.T) {
	ex := &fakeExchanger{
		token:   GoogleToken{AccessToken: "at", RefreshToken: "rt"},
		profile: GoogleProfile{Sub: "google-1", Email: "a@example.com", Name: "A Person"},
	}
	repo, sess := newFakeRepo(), newFakeSessions()
	svc := newTestService(ex, repo, sess)

	out, err := svc.SignIn(context.Background(), "the-code", "https://app/cb")
	if err != nil {
		t.Fatalf("SignIn() = %v", err)
	}

	if ex.gotCode != "the-code" || ex.gotRedirect != "https://app/cb" {
		t.Errorf("Exchange got (%q, %q)", ex.gotCode, ex.gotRedirect)
	}
	if ex.gotAccessToken != "at" {
		t.Errorf("UserInfo got access token %q, want %q", ex.gotAccessToken, "at")
	}
	if repo.last.googleID != "google-1" || repo.last.refreshToken != "rt" || repo.last.email != "a@example.com" {
		t.Errorf("upsert got %+v", repo.last)
	}
	if out.User.ID != "id-google-1" || out.User.CEFRCurrent != "A1" {
		t.Errorf("user = %+v", out.User)
	}

	stored, err := sess.Get(context.Background(), out.User.ID)
	if err != nil {
		t.Fatalf("no session stored: %v", err)
	}
	if stored != out.Token {
		t.Errorf("stored session = %q, want the issued token %q", stored, out.Token)
	}
	if sess.ttls[out.User.ID] != TokenTTL {
		t.Errorf("session TTL = %v, want %v", sess.ttls[out.User.ID], TokenTTL)
	}
}

func TestSignInKeepsAReturningUsersLevel(t *testing.T) {
	repo := newFakeRepo()
	repo.users["google-1"] = User{ID: "id-google-1", CEFRCurrent: "B2"}

	ex := &fakeExchanger{
		token:   GoogleToken{AccessToken: "at", RefreshToken: "rt2"},
		profile: GoogleProfile{Sub: "google-1", Email: "a@example.com", Name: "A Person"},
	}
	svc := newTestService(ex, repo, newFakeSessions())

	out, err := svc.SignIn(context.Background(), "code", "uri")
	if err != nil {
		t.Fatalf("SignIn() = %v", err)
	}
	if out.User.CEFRCurrent != "B2" {
		t.Errorf("CEFRCurrent = %q, want B2 preserved on re-login", out.User.CEFRCurrent)
	}
}

func TestSignInIssuesAVerifiableToken(t *testing.T) {
	ex := &fakeExchanger{
		token:   GoogleToken{AccessToken: "at"},
		profile: GoogleProfile{Sub: "google-1", Email: "a@example.com"},
	}
	iss := NewTokenIssuer("secret", func() time.Time { return time.Unix(1_800_000_000, 0) })
	svc := NewService(ex, newFakeRepo(), newFakeSessions(), iss)

	out, err := svc.SignIn(context.Background(), "code", "uri")
	if err != nil {
		t.Fatalf("SignIn() = %v", err)
	}
	sub, err := iss.Verify(out.Token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if sub != out.User.ID {
		t.Errorf("token subject = %q, want %q", sub, out.User.ID)
	}
}

func TestSignInPropagatesAnExchangeFailure(t *testing.T) {
	ex := &fakeExchanger{err: errors.New("invalid_grant")}
	svc := newTestService(ex, newFakeRepo(), newFakeSessions())

	if _, err := svc.SignIn(context.Background(), "bad", "uri"); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestSignInFailsAndWritesNoSessionWhenTheRepoFails(t *testing.T) {
	ex := &fakeExchanger{
		token:   GoogleToken{AccessToken: "at"},
		profile: GoogleProfile{Sub: "google-1", Email: "a@example.com"},
	}
	repo := &fakeRepo{err: errors.New("pg down")}
	sess := newFakeSessions()
	svc := newTestService(ex, repo, sess)

	if _, err := svc.SignIn(context.Background(), "code", "uri"); err == nil {
		t.Fatal("expected an error, got nil")
	}

	if _, err := sess.Get(context.Background(), "google-1"); err == nil {
		t.Error("expected no session written after a repo failure")
	}
}

func TestSignInFailsWhenTheSessionWriteFails(t *testing.T) {
	ex := &fakeExchanger{
		token:   GoogleToken{AccessToken: "at"},
		profile: GoogleProfile{Sub: "google-1", Email: "a@example.com"},
	}
	fakeErr := errors.New("dial tcp: i/o timeout")
	sess := &fakeSessions{vals: map[string]string{}, ttls: map[string]time.Duration{}, err: fakeErr}
	svc := newTestService(ex, newFakeRepo(), sess)

	_, err := svc.SignIn(context.Background(), "code", "uri")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, fakeErr) {
		t.Errorf("err = %v, want it to wrap %v", err, fakeErr)
	}
}
