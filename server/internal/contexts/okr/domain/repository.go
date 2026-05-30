package domain

import (
	"context"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// PlanFilter is a value object describing a Plan query.
type PlanFilter struct {
	Owner      kernel.ID
	Owners     []kernel.ID
	AllOwners  bool
	Level      PlanLevel
	Pagination kernel.Pagination
}

// PlanRepository abstracts persistence.  Implemented in infrastructure/pg_plan_repo.go.
type PlanRepository interface {
	Save(ctx context.Context, p *Plan) error
	Get(ctx context.Context, id kernel.ID) (*Plan, error)
	List(ctx context.Context, f PlanFilter) ([]*Plan, error)
	ListChildren(ctx context.Context, parentID kernel.ID) ([]*Plan, error)
}

// ReportRepository.
type ReportFilter struct {
	Owner      kernel.ID
	Owners     []kernel.ID
	AllOwners  bool
	Type       ReportType
	From, To   *Period
	Pagination kernel.Pagination
}

type ReportRepository interface {
	Save(ctx context.Context, r *Report) error
	Get(ctx context.Context, id kernel.ID) (*Report, error)
	GetByOwnerAndPeriod(ctx context.Context, owner kernel.ID, typ ReportType, period Period) (*Report, error)
	List(ctx context.Context, f ReportFilter) ([]*Report, error)
	Comment(ctx context.Context, reportID kernel.ID, author kernel.ID, body string) error
	ListComments(ctx context.Context, reportID kernel.ID) ([]Comment, error)
}

// Comment is a read-side projection.
type Comment struct {
	ID        kernel.ID
	ReportID  kernel.ID
	Author    kernel.ID
	Body      string
	CreatedAt int64 // unix
}

// RollupQuery is a Query-side projection (CQRS local optimization).
type RollupQuery interface {
	WeeklyByDept(ctx context.Context, periodStart, periodEnd string) ([]RollupRow, error)
	WeeklyByDeptScoped(ctx context.Context, periodStart, periodEnd string, owners []kernel.ID, allOwners bool) ([]RollupRow, error)
}

// RollupRow is what /rollups/weekly returns: one line per member.
type RollupRow struct {
	MemberID   kernel.ID `json:"member_id"`
	OwnerName  string    `json:"owner_name"`
	Department string    `json:"department"`
	Submitted  bool      `json:"submitted"`
	Summary    string    `json:"summary,omitempty"`
}
