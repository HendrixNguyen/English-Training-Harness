package store

import (
	"context"
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

// fsMigrator is a fake Migrator for exercising Migrate's own logic (ordering,
// skip-applied, error propagation) over an in-memory fs.FS, with no database.
// Distinct name from migrations_test.go's fakeMigrator, which that file still
// owns.
type fsMigrator struct {
	ensureErr  error
	appliedErr error
	done       map[string]bool
	failOn     string
	order      []string // versions actually applied, in call order
}

func (f *fsMigrator) EnsureVersionTable(_ context.Context) error { return f.ensureErr }

func (f *fsMigrator) AppliedVersions(_ context.Context) (map[string]bool, error) {
	if f.appliedErr != nil {
		return nil, f.appliedErr
	}
	return f.done, nil
}

func (f *fsMigrator) Apply(_ context.Context, version, _ string) error {
	if version == f.failOn {
		return errors.New("boom")
	}
	f.order = append(f.order, version)
	return nil
}

// mapFS builds an in-memory migrations/*.up.sql tree, one file per name.
func mapFS(names ...string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for _, name := range names {
		fsys["migrations/"+name+".up.sql"] = &fstest.MapFile{Data: []byte("-- " + name)}
	}
	return fsys
}

func TestMigrateAppliesInFilenameOrder(t *testing.T) {
	m := &fsMigrator{}
	fsys := mapFS("0010_ten", "0002_two", "0001_one")

	got, err := Migrate(context.Background(), m, fsys)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	want := []string{"0001_one", "0002_two", "0010_ten"}
	if !reflect.DeepEqual(m.order, want) {
		t.Errorf("applied in order %v, want %v", m.order, want)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Migrate returned %v, want %v", got, want)
	}
}

func TestMigrateSkipsAppliedVersions(t *testing.T) {
	m := &fsMigrator{done: map[string]bool{"0001_one": true}}
	fsys := mapFS("0001_one", "0002_two")

	got, err := Migrate(context.Background(), m, fsys)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	want := []string{"0002_two"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Migrate returned %v, want %v", got, want)
	}
}

func TestMigrateEnsureVersionTableErrorAppliesNothing(t *testing.T) {
	sentinel := errors.New("ensure boom")
	m := &fsMigrator{ensureErr: sentinel}
	fsys := mapFS("0001_one")

	got, err := Migrate(context.Background(), m, fsys)
	if err == nil {
		t.Fatal("Migrate() = nil error, want an error")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(%v, sentinel) = false, want true", err)
	}
	if !strings.HasPrefix(err.Error(), "store: ensuring version table:") {
		t.Errorf("error = %q, want prefix %q", err.Error(), "store: ensuring version table:")
	}
	if len(got) != 0 {
		t.Errorf("Migrate returned %v, want nothing applied", got)
	}
	if len(m.order) != 0 {
		t.Errorf("Apply was called %v, want none", m.order)
	}
}

func TestMigrateAppliedVersionsErrorAppliesNothing(t *testing.T) {
	sentinel := errors.New("applied boom")
	m := &fsMigrator{appliedErr: sentinel}
	fsys := mapFS("0001_one")

	got, err := Migrate(context.Background(), m, fsys)
	if err == nil {
		t.Fatal("Migrate() = nil error, want an error")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(%v, sentinel) = false, want true", err)
	}
	if !strings.HasPrefix(err.Error(), "store: reading applied versions:") {
		t.Errorf("error = %q, want prefix %q", err.Error(), "store: reading applied versions:")
	}
	if len(got) != 0 {
		t.Errorf("Migrate returned %v, want nothing applied", got)
	}
	if len(m.order) != 0 {
		t.Errorf("Apply was called %v, want none", m.order)
	}
}

func TestMigrateReturnsPartialAppliedWhenReadFails(t *testing.T) {
	m := &fsMigrator{}
	fsys := fstest.MapFS{
		"migrations/0001_one.up.sql": &fstest.MapFile{Data: []byte("-- 0001_one")},
		"migrations/0002_two.up.sql": &fstest.MapFile{Mode: fs.ModeDir},
	}

	got, err := Migrate(context.Background(), m, fsys)
	if err == nil {
		t.Fatal("Migrate() = nil error, want an error")
	}
	if !strings.Contains(err.Error(), "reading migrations/0002_two.up.sql") {
		t.Errorf("error = %q, want it to mention reading migrations/0002_two.up.sql", err.Error())
	}
	want := []string{"0001_one"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Migrate returned %v, want %v", got, want)
	}
}

func TestMigrateReturnsPartialAppliedWhenApplyFails(t *testing.T) {
	m := &fsMigrator{failOn: "0002_two"}
	fsys := mapFS("0001_one", "0002_two", "0003_three")

	got, err := Migrate(context.Background(), m, fsys)
	if err == nil {
		t.Fatal("Migrate() = nil error, want an error")
	}
	if !strings.Contains(err.Error(), "applying 0002_two") {
		t.Errorf("error = %q, want it to mention applying 0002_two", err.Error())
	}
	want := []string{"0001_one"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Migrate returned %v, want %v", got, want)
	}
	if !reflect.DeepEqual(m.order, want) {
		t.Errorf("applied %v, want %v", m.order, want)
	}
}

func TestMigrateRefusesAMalformedMigrationName(t *testing.T) {
	m := &fsMigrator{}
	fsys := fstest.MapFS{
		"migrations/0001_one.up.sql": &fstest.MapFile{Data: []byte("-- 0001_one")},
		"migrations/README.up.sql":   &fstest.MapFile{Data: []byte("-- readme")},
	}

	got, err := Migrate(context.Background(), m, fsys)
	if err == nil {
		t.Fatal("Migrate() = nil error, want an error")
	}
	want := `store: malformed migration name "migrations/README.up.sql"`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
	if len(got) != 0 {
		t.Errorf("Migrate returned %v, want nothing applied", got)
	}
	if len(m.order) != 0 {
		t.Errorf("Apply was called %v, want none (malformed name must be caught before applying anything)", m.order)
	}
}

func TestMigration0001DownDropsTypesAfterTables(t *testing.T) {
	down := readMigration(t, "0001_init.down.sql")

	tableIdx := strings.Index(down, "DROP TABLE IF EXISTS users")
	if tableIdx == -1 {
		t.Fatal("0001_init.down.sql does not drop users")
	}
	for _, typeStmt := range []string{
		"DROP TYPE IF EXISTS cefr_level",
		"DROP TYPE IF EXISTS pet_stage",
		"DROP TYPE IF EXISTS task_category",
	} {
		idx := strings.Index(down, typeStmt)
		if idx == -1 {
			t.Errorf("0001_init.down.sql is missing %q", typeStmt)
			continue
		}
		if idx < tableIdx {
			t.Errorf("%q appears before DROP TABLE IF EXISTS users; types must drop after tables", typeStmt)
		}
	}
}
