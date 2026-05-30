// Package infrastructure is the PG persistence for the project module. All
// queries run through tenantdb.TenantDB, which sets search_path to the caller's
// tenant schema — so the same SQL is automatically isolated per tenant.
package infrastructure

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/project/domain"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

type Repo struct{ db *tenantdb.TenantDB }

func NewRepo(db *tenantdb.TenantDB) *Repo { return &Repo{db: db} }

// ---- Projects ----

func (r *Repo) CreateProject(ctx context.Context, p *domain.Project) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO project (id, name, description, status, created_by, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			p.ID, p.Name, p.Description, p.Status, p.CreatedBy, p.CreatedAt, p.UpdatedAt)
		return err
	})
}

func (r *Repo) ListProjects(ctx context.Context) ([]*domain.Project, error) {
	var out []*domain.Project
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT p.id, p.name, p.description, p.status, p.created_by, p.created_at, p.updated_at,
			        COALESCE(c.total,0)
			 FROM project p
			 LEFT JOIN (SELECT project_id, count(*) AS total FROM project_card GROUP BY project_id) c
			   ON c.project_id = p.id
			 ORDER BY (p.status='archived'), p.created_at DESC`)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.Project{}
		for rows.Next() {
			p := &domain.Project{}
			if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.CreatedBy,
				&p.CreatedAt, &p.UpdatedAt, &p.CardCount); err != nil {
				return err
			}
			out = append(out, p)
		}
		return rows.Err()
	})
	return out, err
}

func (r *Repo) GetProject(ctx context.Context, id kernel.ID) (*domain.Project, error) {
	var p *domain.Project
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`SELECT id, name, description, status, created_by, created_at, updated_at
			 FROM project WHERE id=$1`, id)
		var x domain.Project
		err := row.Scan(&x.ID, &x.Name, &x.Description, &x.Status, &x.CreatedBy, &x.CreatedAt, &x.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		p = &x
		return nil
	})
	return p, err
}

// GetBoard loads a project with its columns and cards in one transaction.
func (r *Repo) GetBoard(ctx context.Context, id kernel.ID) (*domain.Project, error) {
	var p *domain.Project
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`SELECT id, name, description, status, created_by, created_at, updated_at
			 FROM project WHERE id=$1`, id)
		var x domain.Project
		err := row.Scan(&x.ID, &x.Name, &x.Description, &x.Status, &x.CreatedBy, &x.CreatedAt, &x.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}

		// columns
		colRows, err := tx.Query(ctx,
			`SELECT id, project_id, name, order_num, created_at, updated_at
			 FROM project_column WHERE project_id=$1 ORDER BY order_num, created_at`, id)
		if err != nil {
			return err
		}
		colIndex := map[kernel.ID]*domain.Column{}
		x.Columns = []*domain.Column{}
		func() {
			defer colRows.Close()
			for colRows.Next() {
				col := &domain.Column{}
				if err = colRows.Scan(&col.ID, &col.ProjectID, &col.Name, &col.OrderNum, &col.CreatedAt, &col.UpdatedAt); err != nil {
					return
				}
				col.Cards = []*domain.Card{}
				x.Columns = append(x.Columns, col)
				colIndex[col.ID] = col
			}
			err = colRows.Err()
		}()
		if err != nil {
			return err
		}

		// cards
		cardRows, err := tx.Query(ctx,
			`SELECT `+cardCols+` FROM project_card WHERE project_id=$1 ORDER BY order_num, created_at`, id)
		if err != nil {
			return err
		}
		defer cardRows.Close()
		for cardRows.Next() {
			card, scanErr := scanCard(cardRows)
			if scanErr != nil {
				return scanErr
			}
			if col, ok := colIndex[card.ColumnID]; ok {
				col.Cards = append(col.Cards, card)
			}
		}
		if err := cardRows.Err(); err != nil {
			return err
		}
		p = &x
		return nil
	})
	return p, err
}

func (r *Repo) UpdateProject(ctx context.Context, id kernel.ID, name, description, status string) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE project SET name=$1, description=$2, status=$3, updated_at=now() WHERE id=$4`,
			name, description, status, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) DeleteProject(ctx context.Context, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, `DELETE FROM project WHERE id=$1`, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// ---- Columns ----

func (r *Repo) CreateColumn(ctx context.Context, col *domain.Column) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO project_column (id, project_id, name, order_num, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6)`,
			col.ID, col.ProjectID, col.Name, col.OrderNum, col.CreatedAt, col.UpdatedAt)
		return err
	})
}

// NextColumnOrder returns max(order_num)+1 for a project's columns.
func (r *Repo) NextColumnOrder(ctx context.Context, projectID kernel.ID) (int, error) {
	var next int
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `SELECT COALESCE(MAX(order_num),-1)+1 FROM project_column WHERE project_id=$1`, projectID)
		return row.Scan(&next)
	})
	return next, err
}

func (r *Repo) GetColumn(ctx context.Context, id kernel.ID) (*domain.Column, error) {
	var col *domain.Column
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`SELECT id, project_id, name, order_num, created_at, updated_at FROM project_column WHERE id=$1`, id)
		var x domain.Column
		err := row.Scan(&x.ID, &x.ProjectID, &x.Name, &x.OrderNum, &x.CreatedAt, &x.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		col = &x
		return nil
	})
	return col, err
}

func (r *Repo) UpdateColumn(ctx context.Context, id kernel.ID, name string, orderNum int) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE project_column SET name=$1, order_num=$2, updated_at=now() WHERE id=$3`,
			name, orderNum, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) DeleteColumn(ctx context.Context, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, `DELETE FROM project_column WHERE id=$1`, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// ---- Cards ----

const cardCols = `id, project_id, column_id, title, description, assignee_id, due_date, priority, order_num, created_at, updated_at`

func scanCard(row pgx.Row) (*domain.Card, error) {
	c := &domain.Card{}
	err := row.Scan(&c.ID, &c.ProjectID, &c.ColumnID, &c.Title, &c.Description,
		&c.AssigneeID, &c.DueDate, &c.Priority, &c.OrderNum, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *Repo) CreateCard(ctx context.Context, c *domain.Card) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO project_card (`+cardCols+`)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			c.ID, c.ProjectID, c.ColumnID, c.Title, c.Description, c.AssigneeID,
			c.DueDate, c.Priority, c.OrderNum, c.CreatedAt, c.UpdatedAt)
		return err
	})
}

// NextCardOrder returns max(order_num)+1 for a column's cards.
func (r *Repo) NextCardOrder(ctx context.Context, columnID kernel.ID) (int, error) {
	var next int
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `SELECT COALESCE(MAX(order_num),-1)+1 FROM project_card WHERE column_id=$1`, columnID)
		return row.Scan(&next)
	})
	return next, err
}

func (r *Repo) GetCard(ctx context.Context, id kernel.ID) (*domain.Card, error) {
	var card *domain.Card
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `SELECT `+cardCols+` FROM project_card WHERE id=$1`, id)
		got, err := scanCard(row)
		if err != nil {
			return err
		}
		card = got
		return nil
	})
	return card, err
}

// UpdateCard persists the EDITABLE fields only. It deliberately does NOT touch
// column_id/order_num — those are owned by MoveCard — so a concurrent edit and a
// concurrent move operate on disjoint columns and cannot clobber each other.
func (r *Repo) UpdateCard(ctx context.Context, c *domain.Card) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE project_card SET title=$1, description=$2, assignee_id=$3,
			        due_date=$4, priority=$5, updated_at=now()
			 WHERE id=$6`,
			c.Title, c.Description, c.AssigneeID, c.DueDate, c.Priority, c.ID)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// MoveCard sets the card's column and order in a single targeted statement,
// independent of the editable-field overwrite in UpdateCard.
func (r *Repo) MoveCard(ctx context.Context, id, columnID kernel.ID, orderNum int) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE project_card SET column_id=$1, order_num=$2, updated_at=now() WHERE id=$3`,
			columnID, orderNum, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) DeleteCard(ctx context.Context, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, `DELETE FROM project_card WHERE id=$1`, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}
