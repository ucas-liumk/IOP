package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/okr/domain"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

// PGRollupQuery is the CQRS-style direct query: joins member + okr_report.
type PGRollupQuery struct{ db *tenantdb.TenantDB }

func NewPGRollupQuery(db *tenantdb.TenantDB) *PGRollupQuery { return &PGRollupQuery{db: db} }

func (q *PGRollupQuery) WeeklyByDept(ctx context.Context, periodStart, periodEnd string) ([]domain.RollupRow, error) {
	return q.WeeklyByDeptScoped(ctx, periodStart, periodEnd, nil, true)
}

func (q *PGRollupQuery) WeeklyByDeptScoped(ctx context.Context, periodStart, periodEnd string, owners []kernel.ID, allOwners bool) ([]domain.RollupRow, error) {
	var out []domain.RollupRow
	err := q.db.Transaction(ctx, func(tx pgx.Tx) error {
		args := []any{periodStart, periodEnd}
		where := ""
		if !allOwners {
			if len(owners) == 0 {
				return nil
			}
			where = "WHERE m.id = ANY($3::uuid[])"
			args = append(args, idStrings(owners))
		}
		rows, err := tx.Query(ctx,
			`SELECT m.id, m.display_name, COALESCE(m.department,''),
			        (r.id IS NOT NULL) AS submitted,
			        COALESCE(r.summary,'')
			 FROM member m
			 LEFT JOIN okr_report r ON r.owner = m.id AND r.type = 'weekly'
			   AND r.period_start = $1 AND r.period_end = $2
			 `+where+`
			 ORDER BY m.department NULLS LAST, m.display_name`, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var row domain.RollupRow
			if err := rows.Scan(&row.MemberID, &row.OwnerName, &row.Department, &row.Submitted, &row.Summary); err != nil {
				return err
			}
			out = append(out, row)
		}
		return rows.Err()
	})
	return out, err
}
