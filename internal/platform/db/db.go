// Package db wires the PostgreSQL connection pool (pgx) used across Postal.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps a pgx connection pool. Domains receive it via constructors; there
// is no package-level global.
type Pool struct {
	*pgxpool.Pool
}

// Connect opens a connection pool against the given DSN and verifies it with a
// ping bounded by ctx. The caller owns the returned Pool and must Close it.
func Connect(ctx context.Context, dsn string) (*Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("creating pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}
	return &Pool{Pool: pool}, nil
}

// Ping verifies the database is reachable within ctx's deadline. Used by /readyz.
func (p *Pool) Ping(ctx context.Context) error {
	if err := p.Pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}
	return nil
}
