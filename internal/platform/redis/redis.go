// Package redis wires the Redis client used as the asynq broker, rate-limit
// counter store, and cache across Postal.
package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Client wraps a go-redis client. Domains receive it via constructors; there is
// no package-level global.
type Client struct {
	*redis.Client
}

// Options configures the Redis connection.
type Options struct {
	// Addr is the host:port of the Redis server.
	Addr string
	// Password authenticates to Redis (empty when auth is disabled).
	Password string
	// DB is the logical database number.
	DB int
}

// Connect opens a Redis client and verifies it with a ping bounded by ctx. The
// caller owns the returned Client and must Close it.
func Connect(ctx context.Context, opts Options) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("pinging redis: %w", err)
	}
	return &Client{Client: rdb}, nil
}

// Ping verifies Redis is reachable within ctx's deadline. Used by /readyz.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}
