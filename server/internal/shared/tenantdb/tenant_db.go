package tenantdb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leo/iop/server/internal/shared/errors"
)

// TenantDB wraps the same pool as PlatformDB but enforces SET LOCAL search_path
// inside every transaction it opens.
type TenantDB struct {
	pool *pgxpool.Pool
}

func NewTenantDB(pool *pgxpool.Pool) *TenantDB {
	return &TenantDB{pool: pool}
}

// Transaction starts a tx, sets search_path to the tenant's schema, then runs fn.
// SET LOCAL is automatically rolled back on COMMIT or ROLLBACK — no stale state
// can leak to the next user of this connection.
func (t *TenantDB) Transaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tc, ok := FromContext(ctx)
	if !ok {
		return errors.New(errors.KindInternal, "tenantdb.context_missing",
			"TenantDB.Transaction called without tenant context")
	}
	if err := validateSchemaIdent(tc.SchemaName); err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, t.pool, func(tx pgx.Tx) error {
		sql := fmt.Sprintf("SET LOCAL search_path TO %q, public", tc.SchemaName)
		if _, err := tx.Exec(ctx, sql); err != nil {
			return errors.Wrap(errors.KindDatabase, "tenantdb.set_search_path",
				"failed to set search_path", err)
		}
		return fn(tx)
	})
}
