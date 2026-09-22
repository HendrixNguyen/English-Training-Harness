package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// versionTableDDL is the migration runner's own bookkeeping table.
const versionTableDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
)`

// Postgres is the process-wide connection pool built from DATABASE_URL.
type Postgres struct {
	Pool *pgxpool.Pool
}

// NewPostgres parses url and creates a lazy pool; it does not dial. Call Ping
// to confirm the database is reachable.
func NewPostgres(ctx context.Context, url string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("store: parsing DATABASE_URL: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: creating pool: %w", err)
	}
	return &Postgres{Pool: pool}, nil
}

// Ping reports whether the database is reachable (used by GET /healthz).
func (p *Postgres) Ping(ctx context.Context) error { return p.Pool.Ping(ctx) }

// Close releases the pool.
func (p *Postgres) Close() { p.Pool.Close() }

// Migrator returns the Migrator backed by this pool.
func (p *Postgres) Migrator() Migrator { return &PgMigrator{pool: p.Pool} }

// PgMigrator applies migrations to Postgres, one transaction per version.
type PgMigrator struct{ pool *pgxpool.Pool }

// migrationLockKey is an arbitrary constant used with pg_advisory_lock to
// serialize Migrate across concurrent processes. It has no meaning beyond
// being unique to this migrator among the advisory locks this codebase takes.
const migrationLockKey = 727100001

// Lock implements store.Locker with a session-level Postgres advisory lock
// held on a single pinned connection for the whole migration run. A second
// caller blocks in pg_advisory_lock until the first's unlock runs, so by the
// time it proceeds, AppliedVersions correctly reports the first caller's work
// as already done instead of racing to apply it a second time.
func (m *PgMigrator) Lock(ctx context.Context) (func(), error) {
	conn, err := m.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: acquiring a connection for the migration lock: %w", err)
	}
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, int64(migrationLockKey)); err != nil {
		conn.Release()
		return nil, fmt.Errorf("store: acquiring the migration lock: %w", err)
	}
	return func() {
		_, _ = conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, int64(migrationLockKey))
		conn.Release()
	}, nil
}

var _ Locker = (*PgMigrator)(nil)

func (m *PgMigrator) EnsureVersionTable(ctx context.Context) error {
	_, err := m.pool.Exec(ctx, versionTableDDL)
	return err
}

func (m *PgMigrator) AppliedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := m.pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

// Apply runs the migration body and records the version in one transaction, so
// a failed migration leaves no half-applied schema and no version row.
func (m *PgMigrator) Apply(ctx context.Context, version, sql string) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, sql); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// compile-time check
var _ Migrator = (*PgMigrator)(nil)
