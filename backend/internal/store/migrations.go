package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// MigrationsFS holds the numbered DDL files. Each version is a pair
// NNNN_name.up.sql / NNNN_name.down.sql; Migrate applies the .up.sql files in
// filename order. Never edit a migration that has been applied — add a new one.
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS

// Migrator is the storage side of the migration runner. PgMigrator is the
// Postgres implementation; tests use a fake.
type Migrator interface {
	// EnsureVersionTable creates the bookkeeping table if it is missing.
	EnsureVersionTable(ctx context.Context) error
	// AppliedVersions returns the set of versions already applied.
	AppliedVersions(ctx context.Context) (map[string]bool, error)
	// Apply runs one migration and records its version atomically.
	Apply(ctx context.Context, version, sql string) error
}

// Locker is optionally implemented by a Migrator that needs to serialize
// concurrent Migrate calls against the same database. PgMigrator implements
// it: CREATE TABLE/TYPE IF NOT EXISTS is a check-then-act, not atomic, so two
// processes migrating the same database at once (e.g. two packages' own
// integration tests, each calling Migrate independently against one shared
// TEST_DATABASE_URL) can fail with "duplicate key value violates unique
// constraint pg_type_typname_nsp_index" even though every individual
// statement says IF NOT EXISTS.
type Locker interface {
	// Lock blocks until the caller has exclusive access to run migrations and
	// returns a function that releases it. Migrate always calls the returned
	// function exactly once, even when a later step fails.
	Lock(ctx context.Context) (unlock func(), err error)
}

// Migrate applies every *.up.sql in fsys that has not been applied yet, in
// filename order, and returns the versions it applied. It is safe to call on
// every boot: a second call with no new files applies nothing. It is also
// safe to call concurrently against the same database when m implements
// Locker (see PgMigrator).
func Migrate(ctx context.Context, m Migrator, fsys fs.FS) ([]string, error) {
	if lk, ok := m.(Locker); ok {
		unlock, err := lk.Lock(ctx)
		if err != nil {
			return nil, fmt.Errorf("store: locking migrator: %w", err)
		}
		defer unlock()
	}
	if err := m.EnsureVersionTable(ctx); err != nil {
		return nil, fmt.Errorf("store: ensuring version table: %w", err)
	}
	done, err := m.AppliedVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: reading applied versions: %w", err)
	}

	names, err := fs.Glob(fsys, "migrations/*.up.sql")
	if err != nil {
		return nil, fmt.Errorf("store: listing migrations: %w", err)
	}
	sort.Strings(names)

	var applied []string
	for _, name := range names {
		version := strings.TrimSuffix(strings.TrimPrefix(name, "migrations/"), ".up.sql")
		if done[version] {
			continue
		}
		body, err := fs.ReadFile(fsys, name)
		if err != nil {
			return applied, fmt.Errorf("store: reading %s: %w", name, err)
		}
		if err := m.Apply(ctx, version, string(body)); err != nil {
			return applied, fmt.Errorf("store: applying %s: %w", version, err)
		}
		applied = append(applied, version)
	}
	return applied, nil
}
