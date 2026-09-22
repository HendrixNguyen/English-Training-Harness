package auth

import (
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

func TestIssueThenVerifyRoundTrips(t *testing.T) {
	iss := NewTokenIssuer("secret", func() time.Time { return time.Unix(1_800_000_000, 0) })

	tok, err := iss.Issue("user-1")
	if err != nil {
		t.Fatalf("Issue() = %v", err)
	}
	userID, err := iss.Verify(tok)
	if err != nil {
		t.Fatalf("Verify() = %v", err)
	}
	if userID != "user-1" {
		t.Errorf("userID = %q, want %q", userID, "user-1")
	}
}

func TestTokenExpiryMatchesTheRedisSessionTTL(t *testing.T) {
	if TokenTTL != store.SessionTTL {
		t.Fatalf("TokenTTL = %v, store.SessionTTL = %v — they must be one value", TokenTTL, store.SessionTTL)
	}
	if TokenTTL != 24*time.Hour {
		t.Errorf("TokenTTL = %v, want 24h (spec §4)", TokenTTL)
	}
}

func TestVerifyRejectsAnExpiredToken(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	iss := NewTokenIssuer("secret", func() time.Time { return now })
	tok, err := iss.Issue("user-1")
	if err != nil {
		t.Fatalf("Issue() = %v", err)
	}

	later := NewTokenIssuer("secret", func() time.Time { return now.Add(TokenTTL + time.Minute) })
	if _, err := later.Verify(tok); err == nil {
		t.Fatal("expected an error for an expired token, got nil")
	}
}

func TestVerifyRejectsAnotherSecret(t *testing.T) {
	tok, err := NewTokenIssuer("secret", time.Now).Issue("user-1")
	if err != nil {
		t.Fatalf("Issue() = %v", err)
	}
	if _, err := NewTokenIssuer("other-secret", time.Now).Verify(tok); err == nil {
		t.Fatal("expected an error for a token signed with another secret, got nil")
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	iss := NewTokenIssuer("secret", time.Now)

	for _, bad := range []string{"", "not-a-jwt", "a.b.c"} {
		if _, err := iss.Verify(bad); err == nil {
			t.Errorf("Verify(%q) = nil error, want an error", bad)
		}
	}
}

func TestVerifyRejectsTheNoneAlgorithm(t *testing.T) {
	// alg=none with {"sub":"user-1"} and an empty signature.
	const none = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJ1c2VyLTEifQ."
	if _, err := NewTokenIssuer("secret", time.Now).Verify(none); err == nil {
		t.Fatal("expected an error for alg=none, got nil")
	}
}
