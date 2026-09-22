package auth

import (
	"strings"
	"testing"
)

func TestUpsertSQLMatchesTheContract(t *testing.T) {
	sql := upsertUserSQL

	for _, want := range []string{
		"INSERT INTO users",
		"ON CONFLICT (google_id) DO UPDATE",
		"email = EXCLUDED.email",
		"full_name = EXCLUDED.full_name",
		"RETURNING id, email, full_name, cefr_current",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("upsertUserSQL is missing %q:\n%s", want, sql)
		}
	}

	// cefr_current and target_goal must never appear on the UPDATE side:
	// re-login must not reset a user's level or goal.
	update := sql[strings.Index(sql, "DO UPDATE"):]
	for _, forbidden := range []string{"cefr_current =", "target_goal ="} {
		if strings.Contains(update, forbidden) {
			t.Errorf("the DO UPDATE clause must not set %q:\n%s", forbidden, update)
		}
	}

	// An empty refresh token from Google must not wipe the stored one.
	if !strings.Contains(update, "COALESCE(NULLIF(EXCLUDED.google_refresh_token, ''), users.google_refresh_token)") {
		t.Errorf("re-login with no refresh_token must keep the stored one:\n%s", update)
	}
}

func TestNewUserGetsAnEmptyTargetGoal(t *testing.T) {
	// users.target_goal is NOT NULL with no default in spec §3.2, and the goal
	// is only known after onboarding.
	if !strings.Contains(upsertUserSQL, "target_goal") {
		t.Fatal("the INSERT must name target_goal explicitly")
	}
	if defaultTargetGoal != "" {
		t.Errorf("defaultTargetGoal = %q, want the empty string", defaultTargetGoal)
	}
}
