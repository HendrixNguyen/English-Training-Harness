package auth

import (
	"strings"
	"testing"
)

func TestScopesCoverOpenIDCalendarAndTasks(t *testing.T) {
	want := []string{
		"openid",
		"email",
		"profile",
		"https://www.googleapis.com/auth/calendar.events",
		"https://www.googleapis.com/auth/tasks",
	}
	if len(Scopes) != len(want) {
		t.Fatalf("Scopes = %v, want %d entries", Scopes, len(want))
	}
	for i, w := range want {
		if Scopes[i] != w {
			t.Errorf("Scopes[%d] = %q, want %q", i, Scopes[i], w)
		}
	}
}

func TestScopeStringIsSpaceSeparated(t *testing.T) {
	got := ScopeString()
	if strings.Count(got, " ") != len(Scopes)-1 {
		t.Errorf("ScopeString() = %q, want space-separated", got)
	}
	if !strings.Contains(got, "auth/tasks") {
		t.Errorf("ScopeString() = %q, missing the tasks scope", got)
	}
}

func TestAuthCodeURLRequestsOfflineAccess(t *testing.T) {
	u := AuthCodeURL("cid", "https://app.example.com/callback", "state-123")

	for _, want := range []string{
		"access_type=offline",
		"prompt=consent",
		"response_type=code",
		"client_id=cid",
		"state=state-123",
	} {
		if !strings.Contains(u, want) {
			t.Errorf("AuthCodeURL() = %q, missing %q", u, want)
		}
	}
	if !strings.HasPrefix(u, "https://accounts.google.com/o/oauth2/v2/auth?") {
		t.Errorf("AuthCodeURL() = %q, wrong endpoint", u)
	}
}
