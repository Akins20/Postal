// Package db wires the PostgreSQL connection pool (pgx) used across Postal.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Akins20/postal/internal/platform/db/sqlc"
)

// pgUniqueViolation is the PostgreSQL SQLSTATE for a unique-constraint breach.
const pgUniqueViolation = "23505"

// IsUniqueViolation reports whether err is (or wraps) a Postgres unique-
// constraint violation. Shared so each domain store need not re-derive it.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}

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

// Queries returns a sqlc query set bound to the pool for non-transactional use.
func (p *Pool) Queries() *sqlc.Queries {
	return sqlc.New(p.Pool)
}

// WithTx runs fn inside a database transaction, committing on success and
// rolling back on error or panic. The sqlc query set passed to fn is bound to
// the transaction, so all statements share it atomically.
func (p *Pool) WithTx(ctx context.Context, fn func(q *sqlc.Queries) error) error {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	// Rollback is a no-op once the tx is committed; the error is intentionally
	// ignored in that case.
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(sqlc.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
