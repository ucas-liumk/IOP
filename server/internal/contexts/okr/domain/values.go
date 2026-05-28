// Package domain holds OKR business invariants. Pure Go - no third-party imports.
package domain

import "time"

// PlanLevel — 4 hierarchical cadences. Year > HalfYear > Month > Week.
type PlanLevel string

const (
	LevelYear     PlanLevel = "year"
	LevelHalfYear PlanLevel = "half_year"
	LevelMonth    PlanLevel = "month"
	LevelWeek     PlanLevel = "week"
)

// Valid returns true if l is one of the canonical levels.
func (l PlanLevel) Valid() bool {
	switch l {
	case LevelYear, LevelHalfYear, LevelMonth, LevelWeek:
		return true
	}
	return false
}

// IsBroaderThan reports whether l covers more time than child.
// Used by PlanDecomposer to reject illegal parent→child mappings.
func (l PlanLevel) IsBroaderThan(child PlanLevel) bool {
	order := map[PlanLevel]int{LevelYear: 4, LevelHalfYear: 3, LevelMonth: 2, LevelWeek: 1}
	return order[l] > order[child]
}

// PlanStatus — lifecycle states for a Plan aggregate.
type PlanStatus string

const (
	PlanDraft  PlanStatus = "draft"
	PlanActive PlanStatus = "active"
	PlanClosed PlanStatus = "closed"
)

// ReportType — kinds of work reports.
type ReportType string

const (
	ReportDaily  ReportType = "daily"
	ReportWeekly ReportType = "weekly"
)

func (t ReportType) Valid() bool { return t == ReportDaily || t == ReportWeekly }

// ItemStatus — micro-state for individual PlanItem.
type ItemStatus string

const (
	ItemTodo    ItemStatus = "todo"
	ItemDoing   ItemStatus = "doing"
	ItemDone    ItemStatus = "done"
	ItemBlocked ItemStatus = "blocked"
)

// Period is the [start, end] interval a Plan or Report covers.
// Both bounds are inclusive in business semantics; in DB queries we use < end+1day
// to keep half-open ranges performant.
type Period struct {
	Start time.Time
	End   time.Time
}

// Days returns the number of inclusive days.
func (p Period) Days() int {
	return int(p.End.Sub(p.Start).Hours()/24) + 1
}

// Contains reports whether t falls within [Start, End].
func (p Period) Contains(t time.Time) bool {
	return !t.Before(p.Start) && !t.After(p.End)
}

// Overlaps reports whether two periods share at least one day.
func (p Period) Overlaps(other Period) bool {
	return !(p.End.Before(other.Start) || p.Start.After(other.End))
}

// Progress is a value object capturing percent (0-100) + optional note.
type Progress struct {
	Percent int
	Summary string
}
