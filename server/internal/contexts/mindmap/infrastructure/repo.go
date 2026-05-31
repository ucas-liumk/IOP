// Package infrastructure is the PG persistence for the mindmap module. All
// queries run through tenantdb.TenantDB, which sets search_path to the caller's
// tenant schema — so the same SQL is automatically isolated per tenant.
package infrastructure

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/mindmap/domain"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

type Repo struct{ db *tenantdb.TenantDB }

func NewRepo(db *tenantdb.TenantDB) *Repo { return &Repo{db: db} }

const cols = `id, created_by, title, data, created_at, updated_at`

func scan(row pgx.Row) (*domain.Mindmap, error) {
	m := &domain.Mindmap{}
	err := row.Scan(&m.ID, &m.Owner, &m.Title, &m.Data, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *Repo) Create(ctx context.Context, m *domain.Mindmap) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO mindmap (`+cols+`) VALUES ($1,$2,$3,$4,$5,$6)`,
			m.ID, m.Owner, m.Title, m.Data, m.CreatedAt, m.UpdatedAt)
		return err
	})
}

// List returns the caller's maps without the (potentially large) data blob —
// the editor fetches the full document via Get.
func (r *Repo) List(ctx context.Context, owner kernel.ID) ([]*domain.Mindmap, error) {
	var out []*domain.Mindmap
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, created_by, title, created_at, updated_at
			 FROM mindmap WHERE created_by=$1 ORDER BY updated_at DESC`, owner)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.Mindmap{}
		for rows.Next() {
			m := &domain.Mindmap{}
			if err := rows.Scan(&m.ID, &m.Owner, &m.Title, &m.CreatedAt, &m.UpdatedAt); err != nil {
				return err
			}
			out = append(out, m)
		}
		return rows.Err()
	})
	return out, err
}

func (r *Repo) Get(ctx context.Context, owner, id kernel.ID) (*domain.Mindmap, error) {
	var m *domain.Mindmap
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `SELECT `+cols+` FROM mindmap WHERE id=$1 AND created_by=$2`, id, owner)
		got, err := scan(row)
		if err != nil {
			return err
		}
		m = got
		return nil
	})
	return m, err
}

// Update persists title and/or data. Both are always written; the service loads
// the current row first and applies partial changes before calling this.
func (r *Repo) Update(ctx context.Context, m *domain.Mindmap) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE mindmap SET title=$1, data=$2, updated_at=now()
			 WHERE id=$3 AND created_by=$4`,
			m.Title, m.Data, m.ID, m.Owner)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) Delete(ctx context.Context, owner, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, `DELETE FROM mindmap WHERE id=$1 AND created_by=$2`, id, owner)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}
