package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// fakeSessions is the in-memory SessionStore used across this package's tests.
type fakeSessions struct {
	vals map[string]string
	ttls map[string]time.Duration
	err  error
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{vals: map[string]string{}, ttls: map[string]time.Duration{}}
}

func (f *fakeSessions) Put(_ context.Context, userID, token string, ttl time.Duration) error {
	if f.err != nil {
		return f.err
	}
	f.vals[userID] = token
	f.ttls[userID] = ttl
	return nil
}

func (f *fakeSessions) Get(_ context.Context, userID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	v, ok := f.vals[userID]
	if !ok {
		return "", ErrNoSession
	}
	return v, nil
}

func (f *fakeSessions) Delete(_ context.Context, userID string) error {
	if f.err != nil {
		return f.err
	}
	delete(f.vals, userID)
	return nil
}

func TestSessionKeyComesFromStore(t *testing.T) {
	// The store package owns the §4 key topology; auth must not build key
	// strings of its own.
	if got, want := store.SessionKey("u1"), "sess:u1:token"; got != want {
		t.Fatalf("store.SessionKey = %q, want %q", got, want)
	}
}

func TestFakeSessionsSatisfiesSessionStore(t *testing.T) {
	var s SessionStore = newFakeSessions()

	ctx := context.Background()
	if err := s.Put(ctx, "u1", "tok", store.SessionTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Get(ctx, "u1")
	if err != nil || got != "tok" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	if err := s.Delete(ctx, "u1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, "u1"); err == nil {
		t.Fatal("Get after Delete = nil error, want ErrNoSession")
	}
}

func TestGetOnAnUnreachableRedisIsUnavailableNotNoSession(t *testing.T) {
	// Nothing listens on port 1, so the dial fails immediately instead of
	// timing out — but bound it anyway in case a firewall drops instead of
	// refusing.
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { _ = client.Close() })
	s := &RedisSessionStore{Client: client}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := s.Get(ctx, "u1")
	if err == nil {
		t.Fatal("Get against an unreachable Redis = nil error, want ErrSessionStoreUnavailable")
	}
	if !errors.Is(err, ErrSessionStoreUnavailable) {
		t.Errorf("err = %v, want it to wrap ErrSessionStoreUnavailable", err)
	}
	if errors.Is(err, ErrNoSession) {
		t.Errorf("err = %v, want it NOT to wrap ErrNoSession (the store is down, not the key absent)", err)
	}
}
