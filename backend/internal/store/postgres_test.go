package store

import (
	"context"
	"strings"
	"testing"
)

func TestNewPostgresRejectsAnUnparseableURL(t *testing.T) {
	if _, err := NewPostgres(context.Background(), "://not a url"); err == nil {
		t.Fatal("expected an error for a malformed DATABASE_URL, got nil")
	}
}

func TestVersionTableDDLIsIdempotent(t *testing.T) {
	if !strings.Contains(versionTableDDL, "CREATE TABLE IF NOT EXISTS schema_migrations") {
		t.Errorf("versionTableDDL must be IF NOT EXISTS, got:\n%s", versionTableDDL)
	}
	if !strings.Contains(versionTableDDL, "version TEXT PRIMARY KEY") {
		t.Errorf("schema_migrations needs a primary key on version, got:\n%s", versionTableDDL)
	}
}
