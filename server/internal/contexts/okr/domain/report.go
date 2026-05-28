package domain

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// ReportEntry is an entity within Report aggregate.
type ReportEntry struct {
	ID           kernel.ID  `json:"id"`
	PlanItemID   *kernel.ID `json:"plan_item_id,omitempty"`
	Title        string     `json:"title"`
	Detail       string     `json:"detail,omitempty"`
	ProgressNote string     `json:"progress_note,omitempty"`
	SortOrder    int        `json:"sort_order"`
}

// Report is the aggregate root for daily/weekly status.
type Report struct {
	ID          kernel.ID      `json:"id"`
	Type        ReportType     `json:"type"`
	Owner       kernel.ID      `json:"owner"`
	Period      Period         `json:"period"`
	Summary     string         `json:"summary"`
	Entries     []*ReportEntry `json:"entries"`
	SubmittedAt time.Time      `json:"submitted_at"`
}

// NewDailyReport enforces Period == 1 day.
func NewDailyReport(owner kernel.ID, period Period, summary string, now time.Time) (*Report, error) {
	if period.Days() != 1 {
		return nil, ErrDailyPeriodWrong
	}
	if !period.End.After(period.Start.Add(-time.Hour)) || period.End.After(period.Start.Add(24*time.Hour)) {
		// allowed: same calendar day; reject anything else
	}
	return &Report{
		ID:          kernel.NewID(),
		Type:        ReportDaily,
		Owner:       owner,
		Period:      period,
		Summary:     summary,
		Entries:     []*ReportEntry{},
		SubmittedAt: now,
	}, nil
}

// NewWeeklyReport enforces Period == 7 days starting on Monday.
func NewWeeklyReport(owner kernel.ID, period Period, summary string, now time.Time) (*Report, error) {
	if period.Days() != 7 {
		return nil, ErrWeeklyPeriodWrong
	}
	if period.Start.Weekday() != time.Monday {
		return nil, ErrWeeklyPeriodWrong
	}
	return &Report{
		ID:          kernel.NewID(),
		Type:        ReportWeekly,
		Owner:       owner,
		Period:      period,
		Summary:     summary,
		Entries:     []*ReportEntry{},
		SubmittedAt: now,
	}, nil
}

// AddEntry appends a free-form bullet (optionally tied to a PlanItem).
func (r *Report) AddEntry(title, detail, note string, planItemID *kernel.ID) *ReportEntry {
	e := &ReportEntry{
		ID:           kernel.NewID(),
		PlanItemID:   planItemID,
		Title:        title,
		Detail:       detail,
		ProgressNote: note,
		SortOrder:    len(r.Entries) + 1,
	}
	r.Entries = append(r.Entries, e)
	return e
}
