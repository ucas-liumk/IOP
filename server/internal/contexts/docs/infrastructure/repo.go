// Package infrastructure is the PG persistence for the docs (知识库) module. All
// queries run through tenantdb.TenantDB, which sets search_path to the caller's
// tenant schema — so the same SQL is automatically isolated per tenant.
package infrastructure

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/docs/domain"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

type Repo struct{ db *tenantdb.TenantDB }

func NewRepo(db *tenantdb.TenantDB) *Repo { return &Repo{db: db} }

// nodeCols lists the doc_node columns. Content is excluded from tree/listing
// queries (it can be large) and only loaded for a single-node fetch.
const nodeMetaCols = `id, parent_id, title, type, order_num, created_by, updated_by, created_at, updated_at`

func scanMeta(row pgx.Row) (*domain.Node, error) {
	n := &domain.Node{}
	err := row.Scan(&n.ID, &n.ParentID, &n.Title, &n.Type, &n.OrderNum,
		&n.CreatedBy, &n.UpdatedBy, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return n, nil
}

// ListAll returns every node (metadata only, no content) ordered for stable
// tree assembly: by parent, then order_num, then created_at.
func (r *Repo) ListAll(ctx context.Context) ([]*domain.Node, error) {
	var out []*domain.Node
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT `+nodeMetaCols+` FROM doc_node
			 ORDER BY order_num, created_at`)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.Node{}
		for rows.Next() {
			n, err := scanMeta(rows)
			if err != nil {
				return err
			}
			out = append(out, n)
		}
		return rows.Err()
	})
	return out, err
}

// Get returns a single node WITH its content.
func (r *Repo) Get(ctx context.Context, id kernel.ID) (*domain.Node, error) {
	var n *domain.Node
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`SELECT id, parent_id, title, type, content, order_num,
			        created_by, updated_by, created_at, updated_at
			 FROM doc_node WHERE id=$1`, id)
		x := &domain.Node{}
		err := row.Scan(&x.ID, &x.ParentID, &x.Title, &x.Type, &x.Content, &x.OrderNum,
			&x.CreatedBy, &x.UpdatedBy, &x.CreatedAt, &x.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		n = x
		return nil
	})
	return n, err
}

func (r *Repo) Create(ctx context.Context, n *domain.Node) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO doc_node
			   (id, parent_id, title, type, content, order_num, created_by, updated_by, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			n.ID, n.ParentID, n.Title, n.Type, n.Content, n.OrderNum,
			n.CreatedBy, n.UpdatedBy, n.CreatedAt, n.UpdatedAt)
		return err
	})
}

// UpdateMeta renames a node (used by both folders and docs).
func (r *Repo) UpdateMeta(ctx context.Context, id, updatedBy kernel.ID, title string) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE doc_node SET title=$1, updated_by=$2, updated_at=now() WHERE id=$3`,
			title, updatedBy, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// SaveContent persists a doc's body. Targeted update so it never clobbers a
// concurrent rename.
func (r *Repo) SaveContent(ctx context.Context, id, updatedBy kernel.ID, content string) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE doc_node SET content=$1, updated_by=$2, updated_at=now()
			 WHERE id=$3 AND type='doc'`,
			content, updatedBy, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// Move reparents a node and sets its order. parent==nil moves to root.
func (r *Repo) Move(ctx context.Context, id, updatedBy kernel.ID, parent *kernel.ID, orderNum int) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE doc_node SET parent_id=$1, order_num=$2, updated_by=$3, updated_at=now()
			 WHERE id=$4`,
			parent, orderNum, updatedBy, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// Delete removes a node. ON DELETE CASCADE on parent_id removes the subtree.
func (r *Repo) Delete(ctx context.Context, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, `DELETE FROM doc_node WHERE id=$1`, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}
