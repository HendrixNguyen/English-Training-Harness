package store

import (
	"context"
	"testing"
)

func TestNewRedisRejectsAnUnparseableURL(t *testing.T) {
	if _, err := NewRedis(context.Background(), "http://localhost:6379"); err == nil {
		t.Fatal("expected an error for a non-redis REDIS_URL, got nil")
	}
}

func TestNewRedisAcceptsAValidURLWithoutDialling(t *testing.T) {
	// No server is running; NewRedis must not connect.
	r, err := NewRedis(context.Background(), "redis://localhost:6379/0")
	if err != nil {
		t.Fatalf("NewRedis() = %v, want nil error", err)
	}
	defer r.Close()

	if r.Client == nil {
		t.Fatal("Client is nil")
	}
}
