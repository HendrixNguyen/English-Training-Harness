package store

import (
	"io/fs"
	"strings"
	"testing"
)

func readMigration(t *testing.T, name string) string {
	t.Helper()
	b, err := fs.ReadFile(MigrationsFS, "migrations/"+name)
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(b)
}

func TestMigration0001MatchesSpec32(t *testing.T) {
	up := readMigration(t, "0001_init.up.sql")

	want := []string{
		"CREATE TYPE cefr_level AS ENUM ('A1', 'A2', 'B1', 'B2', 'C1', 'C2');",
		"CREATE TYPE pet_stage AS ENUM ('seed', 'sprout', 'sapling', 'flowering', 'fruitful', 'wilted');",
		"CREATE TYPE task_category AS ENUM ('vocabulary', 'reading', 'practice');",
		"CREATE TABLE users (",
		"CREATE TABLE push_subscriptions (",
		"CREATE TABLE pet_states (",
		"CREATE TABLE daily_progress (",
		"CREATE TABLE roadmaps (",
		"CREATE TABLE exercises (",
		"email VARCHAR(255) UNIQUE NOT NULL",
		"google_id VARCHAR(255) UNIQUE NOT NULL",
		"target_goal VARCHAR(255) NOT NULL",
		// The 1:1 constraint merged by plan add-unique-user-id-to-pet-states-ddl.
		"user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"health_points INT DEFAULT 100 CHECK (health_points BETWEEN 0 AND 100)",
		"stage pet_stage DEFAULT 'sprout'",
		"UNIQUE(user_id, date)",
		"task_type task_category NOT NULL",
		"content_json JSONB NOT NULL",
	}
	for _, w := range want {
		if !strings.Contains(up, w) {
			t.Errorf("0001_init.up.sql is missing %q", w)
		}
	}
}

func TestOnlyPetStatesHasUniqueUserID(t *testing.T) {
	up := readMigration(t, "0001_init.up.sql")

	if got := strings.Count(up, "user_id UUID UNIQUE NOT NULL"); got != 1 {
		t.Errorf("UNIQUE user_id appears %d times, want exactly 1 (pet_states only)", got)
	}
	// push_subscriptions, daily_progress and roadmaps stay 1:N.
	if got := strings.Count(up, "user_id UUID REFERENCES users(id) ON DELETE CASCADE"); got != 3 {
		t.Errorf("plain 1:N user_id appears %d times, want 3", got)
	}
}

func TestMigration0001DownDropsEverything(t *testing.T) {
	down := readMigration(t, "0001_init.down.sql")

	for _, w := range []string{
		"DROP TABLE IF EXISTS exercises;",
		"DROP TABLE IF EXISTS roadmaps;",
		"DROP TABLE IF EXISTS daily_progress;",
		"DROP TABLE IF EXISTS pet_states;",
		"DROP TABLE IF EXISTS push_subscriptions;",
		"DROP TABLE IF EXISTS users;",
		"DROP TYPE IF EXISTS task_category;",
		"DROP TYPE IF EXISTS pet_stage;",
		"DROP TYPE IF EXISTS cefr_level;",
	} {
		if !strings.Contains(down, w) {
			t.Errorf("0001_init.down.sql is missing %q", w)
		}
	}
	// exercises references roadmaps, so it must be dropped first.
	if strings.Index(down, "DROP TABLE IF EXISTS exercises;") > strings.Index(down, "DROP TABLE IF EXISTS roadmaps;") {
		t.Error("down migration must drop exercises before roadmaps")
	}
}
