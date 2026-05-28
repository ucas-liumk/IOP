package tenantdb

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlatformDB wraps the pgx pool for public-schema access.
// All queries run on the `public` schema (no SET LOCAL).
type PlatformDB struct {
	pool *pgxpool.Pool
}

func NewPlatformDB(pool *pgxpool.Pool) *PlatformDB {
	return &PlatformDB{pool: pool}
}

func (p *PlatformDB) Pool() *pgxpool.Pool { return p.pool }

// Transaction runs fn in a transaction on the public schema.
func (p *PlatformDB) Transaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, p.pool, fn)
}

func (p *PlatformDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

func (p *PlatformDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

func (p *PlatformDB) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := p.pool.Exec(ctx, sql, args...)
	return err
}
