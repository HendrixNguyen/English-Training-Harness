package store

import (
	"bytes"
	"context"
	"io/fs"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

// TestIntegrationUnlockReleasesTheLockOnACancelledContext reproduces the bug:
// unlock ran pg_advisory_unlock on the caller's own context, so a cancelled
// context (e.g. a SIGTERM during boot migration) made the unlock call fail
// silently and the session went back to the pool still holding the lock.
func TestIntegrationUnlockReleasesTheLockOnACancelledContext(t *testing.T) {
	pg := requirePostgres(t)

	m := pg.Migrator().(*PgMigrator)
	ctx, cancel := context.WithCancel(context.Background())
	unlock, err := m.Lock(ctx)
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	cancel()
	unlock()

	var count int64
	err = pg.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM pg_locks WHERE locktype = 'advisory' AND classid = 0 AND objid = $1 AND granted`,
		int64(migrationLockKey)).Scan(&count)
	if err != nil {
		t.Fatalf("querying pg_locks: %v", err)
	}
	if count != 0 {
		t.Errorf("pg_locks still holds the migration advisory lock after unlock() on a cancelled context (count=%d)", count)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	pg2, err := NewPostgres(context.Background(), os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	t.Cleanup(pg2.Close)
	unlock2, err := pg2.Migrator().(*PgMigrator).Lock(ctx2)
	if err != nil {
		t.Fatalf("a second Postgres could not acquire the migration lock within 3s: %v (the lock leaked into the pool)", err)
	}
	unlock2()
}

// TestIntegrationLockWaitIsBoundedAndLogged reproduces the folded finding:
// Lock's blocking pg_advisory_lock call had no bound and logged nothing, so a
// caller waiting behind another process's migration hung forever with no
// diagnostic.
func TestIntegrationLockWaitIsBoundedAndLogged(t *testing.T) {
	pg := requirePostgres(t)

	origWait := MigrationLockWait
	MigrationLockWait = 1 * time.Second
	t.Cleanup(func() { MigrationLockWait = origWait })

	origOutput := log.Writer()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(origOutput) })

	holder := pg.Migrator().(*PgMigrator)
	holderUnlock, err := holder.Lock(context.Background())
	if err != nil {
		t.Fatalf("holder Lock: %v", err)
	}

	start := time.Now()
	pg2, err := NewPostgres(context.Background(), os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	t.Cleanup(pg2.Close)
	_, err = pg2.Migrator().(*PgMigrator).Lock(context.Background())
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Lock on a contended lock returned nil error, want a bounded timeout error")
	}
	if elapsed >= 3*time.Second {
		t.Errorf("Lock took %s to give up, want well under 3s (MigrationLockWait=1s)", elapsed)
	}
	if !strings.Contains(err.Error(), "migration lock") || !strings.Contains(err.Error(), "1s") {
		t.Errorf("error %q does not mention the migration lock and the 1s wait", err.Error())
	}
	if !strings.Contains(buf.String(), "waiting") {
		t.Errorf("log output %q does not mention waiting for the contended lock", buf.String())
	}

	holderUnlock()

	buf.Reset()
	ctx3, cancel3 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel3()
	unlock3, err := pg2.Migrator().(*PgMigrator).Lock(ctx3)
	if err != nil {
		t.Fatalf("Lock after the holder released: %v", err)
	}
	unlock3()
	if strings.Contains(buf.String(), "waiting") {
		t.Errorf("an uncontended Lock logged contention: %q", buf.String())
	}
}

// TestIntegrationMigrateSucceedsWithPoolMaxConnsOne reproduces the deadlock:
// Lock pins one connection out of the pool for the whole run, and
// EnsureVersionTable/AppliedVersions/Apply used to ask the pool for a second
// one, which never arrives when pool_max_conns=1.
func TestIntegrationMigrateSucceedsWithPoolMaxConnsOne(t *testing.T) {
	pg := requirePostgres(t)
	reset(t, pg)
	t.Cleanup(func() { reset(t, pg) })

	url := os.Getenv("TEST_DATABASE_URL")
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	pg1, err := NewPostgres(context.Background(), url+sep+"pool_max_conns=1")
	if err != nil {
		t.Fatalf("NewPostgres(pool_max_conns=1): %v", err)
	}
	t.Cleanup(pg1.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	applied, err := Migrate(ctx, pg1.Migrator(), MigrationsFS)
	if err != nil {
		t.Fatalf("Migrate with pool_max_conns=1: %v", err)
	}
	wantNames, err := fs.Glob(MigrationsFS, "migrations/*.up.sql")
	if err != nil {
		t.Fatalf("listing migrations: %v", err)
	}
	if len(applied) != len(wantNames) {
		t.Errorf("applied %v (%d), want %d versions (%v)", applied, len(applied), len(wantNames), wantNames)
	}
}
