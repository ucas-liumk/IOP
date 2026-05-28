package infrastructure

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/okr/domain"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

type PGPlanRepo struct{ db *tenantdb.TenantDB }

func NewPGPlanRepo(db *tenantdb.TenantDB) *PGPlanRepo { return &PGPlanRepo{db: db} }

func (r *PGPlanRepo) Save(ctx context.Context, p *domain.Plan) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO okr_plan (id, level, owner, period_start, period_end, title, parent_id, status, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			 ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, status=EXCLUDED.status, updated_at=EXCLUDED.updated_at`,
			p.ID, string(p.Level), p.Owner, p.Period.Start, p.Period.End, p.Title,
			nullableID(p.ParentID), string(p.Status), p.CreatedAt, p.UpdatedAt)
		if err != nil {
			return err
		}
		// Wipe & rewrite items (simplest correct strategy for v1; v1.5+ can do delta).
		if _, err := tx.Exec(ctx, `DELETE FROM okr_plan_item WHERE plan_id = $1`, p.ID); err != nil {
			return err
		}
		for _, it := range p.Items {
			if _, err := tx.Exec(ctx,
				`INSERT INTO okr_plan_item (id, plan_id, title, weight, progress_pct, progress_note, status, sort_order, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
				it.ID, p.ID, it.Title, it.Weight, it.ProgressPct, it.ProgressNote, string(it.Status), it.SortOrder, it.CreatedAt, it.UpdatedAt); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PGPlanRepo) Get(ctx context.Context, id kernel.ID) (*domain.Plan, error) {
	var p *domain.Plan
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`SELECT id, level, owner, period_start, period_end, title, parent_id, status, created_at, updated_at
			 FROM okr_plan WHERE id = $1`, id)
		var raw domain.Plan
		var level, status string
		var parentID *kernel.ID
		err := row.Scan(&raw.ID, &level, &raw.Owner, &raw.Period.Start, &raw.Period.End, &raw.Title, &parentID, &status, &raw.CreatedAt, &raw.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		raw.Level = domain.PlanLevel(level)
		raw.Status = domain.PlanStatus(status)
		raw.ParentID = parentID
		raw.Items = []*domain.PlanItem{}
		// items
		rows, err := tx.Query(ctx,
			`SELECT id, title, weight, progress_pct, COALESCE(progress_note,''), status, sort_order, created_at, updated_at
			 FROM okr_plan_item WHERE plan_id = $1 ORDER BY sort_order`, id)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			it := &domain.PlanItem{}
			var st string
			if err := rows.Scan(&it.ID, &it.Title, &it.Weight, &it.ProgressPct, &it.ProgressNote, &st, &it.SortOrder, &it.CreatedAt, &it.UpdatedAt); err != nil {
				return err
			}
			it.Status = domain.ItemStatus(st)
			raw.Items = append(raw.Items, it)
		}
		p = &raw
		return rows.Err()
	})
	return p, err
}

func (r *PGPlanRepo) List(ctx context.Context, f domain.PlanFilter) ([]*domain.Plan, error) {
	page := f.Pagination.Normalize()
	var out []*domain.Plan
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		args := []any{}
		where := "WHERE TRUE"
		idx := 1
		if f.Owner != "" {
			where += ` AND owner = $` + itoa(idx)
			args = append(args, f.Owner)
			idx++
		}
		if f.Level != "" {
			where += ` AND level = $` + itoa(idx)
			args = append(args, string(f.Level))
			idx++
		}
		args = append(args, page.PageSize, page.Offset())
		rows, err := tx.Query(ctx,
			`SELECT id, level, owner, period_start, period_end, title, parent_id, status, created_at, updated_at
			 FROM okr_plan `+where+` ORDER BY period_start DESC LIMIT $`+itoa(idx)+` OFFSET $`+itoa(idx+1), args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var raw domain.Plan
			var level, status string
			var parentID *kernel.ID
			if err := rows.Scan(&raw.ID, &level, &raw.Owner, &raw.Period.Start, &raw.Period.End, &raw.Title, &parentID, &status, &raw.CreatedAt, &raw.UpdatedAt); err != nil {
				return err
			}
			raw.Level = domain.PlanLevel(level)
			raw.Status = domain.PlanStatus(status)
			raw.ParentID = parentID
			raw.Items = []*domain.PlanItem{}
			out = append(out, &raw)
		}
		return rows.Err()
	})
	return out, err
}

func (r *PGPlanRepo) ListChildren(ctx context.Context, parentID kernel.ID) ([]*domain.Plan, error) {
	var out []*domain.Plan
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, level, owner, period_start, period_end, title, parent_id, status, created_at, updated_at
			 FROM okr_plan WHERE parent_id = $1`, parentID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var raw domain.Plan
			var level, status string
			var pid *kernel.ID
			if err := rows.Scan(&raw.ID, &level, &raw.Owner, &raw.Period.Start, &raw.Period.End, &raw.Title, &pid, &status, &raw.CreatedAt, &raw.UpdatedAt); err != nil {
				return err
			}
			raw.Level = domain.PlanLevel(level)
			raw.Status = domain.PlanStatus(status)
			raw.ParentID = pid
			out = append(out, &raw)
		}
		return rows.Err()
	})
	return out, err
}

func nullableID(id *kernel.ID) any {
	if id == nil {
		return nil
	}
	return *id
}

// small helper to avoid pulling in strconv
func itoa(i int) string {
	if i < 10 {
		return string('0' + rune(i))
	}
	return string('0'+rune(i/10)) + string('0'+rune(i%10))
}
