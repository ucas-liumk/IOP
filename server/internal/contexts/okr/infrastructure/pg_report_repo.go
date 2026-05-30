package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/okr/domain"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

type PGReportRepo struct{ db *tenantdb.TenantDB }

func NewPGReportRepo(db *tenantdb.TenantDB) *PGReportRepo { return &PGReportRepo{db: db} }

func (r *PGReportRepo) Save(ctx context.Context, rep *domain.Report) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO okr_report (id, type, owner, period_start, period_end, summary, submitted_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (id) DO UPDATE SET summary=EXCLUDED.summary`,
			rep.ID, string(rep.Type), rep.Owner, rep.Period.Start, rep.Period.End, rep.Summary, rep.SubmittedAt)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM okr_report_entry WHERE report_id = $1`, rep.ID); err != nil {
			return err
		}
		for _, e := range rep.Entries {
			if _, err := tx.Exec(ctx,
				`INSERT INTO okr_report_entry (id, report_id, plan_item_id, title, detail, progress_note, sort_order)
				 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				e.ID, rep.ID, idOrNil(e.PlanItemID), e.Title, e.Detail, e.ProgressNote, e.SortOrder); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PGReportRepo) Get(ctx context.Context, id kernel.ID) (*domain.Report, error) {
	var out *domain.Report
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		var rep domain.Report
		var typ string
		err := tx.QueryRow(ctx,
			`SELECT id, type, owner, period_start, period_end, COALESCE(summary,''), submitted_at
			 FROM okr_report WHERE id = $1`, id).
			Scan(&rep.ID, &typ, &rep.Owner, &rep.Period.Start, &rep.Period.End, &rep.Summary, &rep.SubmittedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		rep.Type = domain.ReportType(typ)
		rep.Entries = []*domain.ReportEntry{}
		rows, err := tx.Query(ctx,
			`SELECT id, plan_item_id, title, COALESCE(detail,''), COALESCE(progress_note,''), sort_order
			 FROM okr_report_entry WHERE report_id = $1 ORDER BY sort_order`, id)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			e := &domain.ReportEntry{}
			var pid *kernel.ID
			if err := rows.Scan(&e.ID, &pid, &e.Title, &e.Detail, &e.ProgressNote, &e.SortOrder); err != nil {
				return err
			}
			e.PlanItemID = pid
			rep.Entries = append(rep.Entries, e)
		}
		out = &rep
		return rows.Err()
	})
	return out, err
}

func (r *PGReportRepo) GetByOwnerAndPeriod(ctx context.Context, owner kernel.ID, typ domain.ReportType, period domain.Period) (*domain.Report, error) {
	var out *domain.Report
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		var id kernel.ID
		err := tx.QueryRow(ctx,
			`SELECT id FROM okr_report WHERE owner = $1 AND type = $2 AND period_start = $3`,
			owner, string(typ), period.Start).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		r2, err := r.Get(ctx, id)
		if err != nil {
			return err
		}
		out = r2
		return nil
	})
	return out, err
}

func (r *PGReportRepo) List(ctx context.Context, f domain.ReportFilter) ([]*domain.Report, error) {
	page := f.Pagination.Normalize()
	var out []*domain.Report
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		args := []any{}
		where := "WHERE TRUE"
		idx := 1
		if f.Owner != "" {
			where += ` AND owner = $` + itoa(idx)
			args = append(args, f.Owner)
			idx++
		} else if !f.AllOwners {
			if len(f.Owners) == 0 {
				return nil
			}
			where += ` AND owner = ANY($` + itoa(idx) + `::uuid[])`
			args = append(args, idStrings(f.Owners))
			idx++
		}
		if f.Type != "" {
			where += ` AND type = $` + itoa(idx)
			args = append(args, string(f.Type))
			idx++
		}
		args = append(args, page.PageSize, page.Offset())
		rows, err := tx.Query(ctx,
			`SELECT id, type, owner, period_start, period_end, COALESCE(summary,''), submitted_at
			 FROM okr_report `+where+` ORDER BY period_end DESC LIMIT $`+itoa(idx)+` OFFSET $`+itoa(idx+1), args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var rep domain.Report
			var typ string
			if err := rows.Scan(&rep.ID, &typ, &rep.Owner, &rep.Period.Start, &rep.Period.End, &rep.Summary, &rep.SubmittedAt); err != nil {
				return err
			}
			rep.Type = domain.ReportType(typ)
			rep.Entries = []*domain.ReportEntry{}
			out = append(out, &rep)
		}
		return rows.Err()
	})
	return out, err
}

func (r *PGReportRepo) Comment(ctx context.Context, reportID, author kernel.ID, body string) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO okr_report_comment (id, report_id, author, body, created_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			kernel.NewID(), reportID, author, body, time.Now().UTC())
		return err
	})
}

func (r *PGReportRepo) ListComments(ctx context.Context, reportID kernel.ID) ([]domain.Comment, error) {
	var out []domain.Comment
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, report_id, author, body, EXTRACT(EPOCH FROM created_at)::bigint
			 FROM okr_report_comment WHERE report_id = $1 ORDER BY created_at`, reportID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c domain.Comment
			if err := rows.Scan(&c.ID, &c.ReportID, &c.Author, &c.Body, &c.CreatedAt); err != nil {
				return err
			}
			out = append(out, c)
		}
		return rows.Err()
	})
	return out, err
}

func idOrNil(id *kernel.ID) any {
	if id == nil {
		return nil
	}
	return *id
}
