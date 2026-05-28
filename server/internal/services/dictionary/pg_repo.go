package dictionary

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/shared/tenantdb"
)

// pgRepo loads platform defaults from public.dict_item and merges with tenant overrides.
// For M3 keep platform side in memory (MemoryRepo seed) and apply tenant override at lookup.
type pgRepo struct {
	memory Repository
	tenant *tenantdb.TenantDB
	pool   *pgxpool.Pool
}

// NewPGRepo composes the in-memory seed with a tenant-aware override layer.
func NewPGRepo(memory Repository, pool *pgxpool.Pool, tenant *tenantdb.TenantDB) Repository {
	return &pgRepo{memory: memory, tenant: tenant, pool: pool}
}

func (r *pgRepo) List(ctx context.Context, typeCode string) ([]Item, error) {
	base, err := r.memory.List(ctx, typeCode)
	if err != nil {
		return nil, err
	}
	// If tenant ctx not set, return base as-is.
	if _, ok := tenantdb.FromContext(ctx); !ok {
		return base, nil
	}
	overrides, err := r.loadOverrides(ctx, typeCode)
	if err != nil {
		return base, nil // best-effort
	}
	out := make([]Item, 0, len(base))
	for _, it := range base {
		if o, ok := overrides[it.Code]; ok {
			if o.Name != "" {
				it.Name = o.Name
			}
			if o.SortOrder != 0 {
				it.SortOrder = o.SortOrder
			}
			it.Active = o.Active
		}
		out = append(out, it)
	}
	return out, nil
}

type overrideRow struct {
	Name      string
	SortOrder int
	Active    bool
}

func (r *pgRepo) loadOverrides(ctx context.Context, typeCode string) (map[string]overrideRow, error) {
	out := map[string]overrideRow{}
	err := r.tenant.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT item_code, COALESCE(override->>'name',''), COALESCE((override->>'sort_order')::int,0), COALESCE((override->>'active')::bool, true)
			 FROM dict_override WHERE type_code = $1`, typeCode)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var code string
			var r overrideRow
			if err := rows.Scan(&code, &r.Name, &r.SortOrder, &r.Active); err != nil {
				return err
			}
			out[code] = r
		}
		return rows.Err()
	})
	return out, err
}
