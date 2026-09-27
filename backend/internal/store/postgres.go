package store

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// pgQuerier is what PgMigrator needs from either the pool or one pinned
// connection: *pgxpool.Pool and *pgxpool.Conn both satisfy it.
type pgQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PgMigrator applies migrations to Postgres, one transaction per version. One
// value serves one Migrate call at a time — Postgres.Migrator() returns a
// fresh one per call, which every caller uses.
type PgMigrator struct {
	pool *pgxpool.Pool
	conn *pgxpool.Conn // set while the migration lock is held; nil otherwise
}

// db returns the connection statements should run on: the pinned connection
// while the migration lock is held (so EnsureVersionTable/AppliedVersions/
// Apply never need a second connection from the pool, which would deadlock a
// pool_max_conns=1 configuration), otherwise the pool itself.
func (m *PgMigrator) db() pgQuerier {
	if m.conn != nil {
		return m.conn
	}
	return m.pool
}

// migrationLockKey is an arbitrary constant used with pg_advisory_lock to
// serialize Migrate across concurrent processes. It has no meaning beyond
// being unique to this migrator among the advisory locks this codebase takes.
const migrationLockKey = 727100001

// MigrationLockWait bounds how long Lock waits for another process's
// migration to finish when the caller's context has no deadline of its own.
// Tests shorten it to keep the contention path fast.
var MigrationLockWait = 30 * time.Second

// migrationUnlockTimeout bounds the unlock call itself; it never inherits the
// caller's context cancellation (see the closure Lock returns).
var migrationUnlockTimeout = 5 * time.Second

// Lock implements store.Locker with a session-level Postgres advisory lock
// held on a single pinned connection for the whole migration run. It first
// tries a non-blocking pg_try_advisory_lock; if another process holds it, it
// logs the contention and blocks in pg_advisory_lock, bounded by
// MigrationLockWait when ctx carries no deadline of its own, so a caller
// waiting behind another process's migration cannot hang forever with no
// diagnostic. The returned unlock function runs pg_advisory_unlock on an
// uncancellable, timeout-bounded context — never the caller's own context —
// because a cancelled caller (e.g. a SIGTERM during boot migration) must not
// leave the session, and the lock it holds, sitting back in the pool; if the
// unlock cannot be confirmed, the connection is hijacked and closed instead
// of released, so the lock never leaks into the pool.
func (m *PgMigrator) Lock(ctx context.Context) (func(), error) {
	conn, err := m.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: acquiring a connection for the migration lock: %w", err)
	}

	var got bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, int64(migrationLockKey)).Scan(&got); err != nil {
		conn.Release()
		return nil, fmt.Errorf("store: acquiring the migration lock: %w", err)
	}
	if !got {
		wait := MigrationLockWait
		log.Printf("store: migration lock %d is held by another process; waiting up to %s", migrationLockKey, wait)

		wctx := ctx
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			wctx, cancel = context.WithTimeout(ctx, wait)
			defer cancel()
		}
		start := time.Now()
		if _, err := conn.Exec(wctx, `SELECT pg_advisory_lock($1)`, int64(migrationLockKey)); err != nil {
			// The session holds nothing yet: a cancelled/timed-out
			// pg_advisory_lock acquires nothing, so it is safe to return
			// the connection to the pool.
			conn.Release()
			return nil, fmt.Errorf("store: acquiring the migration lock: another process held it for longer than %s: %w", wait, err)
		}
		log.Printf("store: migration lock acquired after %s", time.Since(start).Round(time.Millisecond))
	}

	m.conn = conn
	return func() {
		m.conn = nil
		uctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), migrationUnlockTimeout)
		defer cancel()
		var released bool
		err := conn.QueryRow(uctx, `SELECT pg_advisory_unlock($1)`, int64(migrationLockKey)).Scan(&released)
		if err == nil && released {
			conn.Release()
			return
		}
		// The session may still hold the lock: end the session instead of
		// handing it back to the pool with the lock leaked into it.
		log.Printf("store: releasing the migration lock failed (released=%v, err=%v); closing the connection", released, err)
		pc := conn.Hijack()
		_ = pc.Close(uctx)
	}, nil
}

var _ Locker = (*PgMigrator)(nil)

func (m *PgMigrator) EnsureVersionTable(ctx context.Context) error {
	_, err := m.db().Exec(ctx, versionTableDDL)
	return err
}

func (m *PgMigrator) AppliedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := m.db().Query(ctx, `SELECT version FROM schema_migrations`)
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
	tx, err := m.db().Begin(ctx)
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
