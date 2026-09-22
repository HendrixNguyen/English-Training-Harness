package store

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Redis is the process-wide client built from REDIS_URL. Key names and TTLs
// live in keys.go (spec §4).
type Redis struct {
	Client *redis.Client
}

// NewRedis parses url and creates a client; it does not dial. Call Ping to
// confirm the server is reachable.
func NewRedis(_ context.Context, url string) (*Redis, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("store: parsing REDIS_URL: %w", err)
	}
	return &Redis{Client: redis.NewClient(opt)}, nil
}

// Ping reports whether Redis is reachable (used by GET /healthz).
func (r *Redis) Ping(ctx context.Context) error { return r.Client.Ping(ctx).Err() }

// Close releases the client.
func (r *Redis) Close() error { return r.Client.Close() }
