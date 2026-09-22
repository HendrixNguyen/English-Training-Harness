package store

import (
	"context"
	"errors"
	"io/fs"
	"reflect"
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

type fakeMigrator struct {
	ensured  int
	applied  []string
	appliedS []string // the SQL passed to Apply, in order
	failOn   string
}

func (f *fakeMigrator) EnsureVersionTable(_ context.Context) error {
	f.ensured++
	return nil
}

func (f *fakeMigrator) AppliedVersions(_ context.Context) (map[string]bool, error) {
	out := map[string]bool{}
	for _, v := range f.applied {
		out[v] = true
	}
	return out, nil
}

func (f *fakeMigrator) Apply(_ context.Context, version, sql string) error {
	if version == f.failOn {
		return errors.New("boom")
	}
	f.applied = append(f.applied, version)
	f.appliedS = append(f.appliedS, sql)
	return nil
}

func TestMigrateAppliesPendingVersions(t *testing.T) {
	m := &fakeMigrator{}

	got, err := Migrate(context.Background(), m, MigrationsFS)
	if err != nil {
		t.Fatalf("Migrate() = %v, want nil error", err)
	}
	if want := []string{"0001_init"}; !reflect.DeepEqual(got, want) {
		t.Errorf("applied = %v, want %v", got, want)
	}
	if m.ensured != 1 {
		t.Errorf("EnsureVersionTable called %d times, want 1", m.ensured)
	}
	if !strings.Contains(m.appliedS[0], "CREATE TABLE users (") {
		t.Error("Apply did not receive the up SQL")
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	m := &fakeMigrator{}

	if _, err := Migrate(context.Background(), m, MigrationsFS); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	got, err := Migrate(context.Background(), m, MigrationsFS)
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("second run applied %v, want nothing", got)
	}
}

func TestMigrateIgnoresDownFiles(t *testing.T) {
	m := &fakeMigrator{}

	if _, err := Migrate(context.Background(), m, MigrationsFS); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	for _, sql := range m.appliedS {
		if strings.Contains(sql, "DROP TABLE") {
			t.Fatal("Migrate applied a .down.sql file")
		}
	}
}

func TestMigrateReportsApplyFailure(t *testing.T) {
	m := &fakeMigrator{failOn: "0001_init"}

	if _, err := Migrate(context.Background(), m, MigrationsFS); err == nil {
		t.Fatal("expected an error when Apply fails, got nil")
	}
}
