package domain

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// PlanItem is an entity inside Plan aggregate; not a standalone aggregate root.
// Modify only via Plan methods (AddItem / UpdateItemProgress / etc.).
type PlanItem struct {
	ID           kernel.ID  `json:"id"`
	Title        string     `json:"title"`
	Weight       int        `json:"weight"`
	ProgressPct  int        `json:"progress_pct"`
	ProgressNote string     `json:"progress_note,omitempty"`
	Status       ItemStatus `json:"status"`
	SortOrder    int        `json:"sort_order"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Plan is the aggregate root.  All invariants enforced via Plan methods.
type Plan struct {
	ID        kernel.ID   `json:"id"`
	Level     PlanLevel   `json:"level"`
	Owner     kernel.ID   `json:"owner"`
	Period    Period      `json:"period"`
	Title     string      `json:"title"`
	ParentID  *kernel.ID  `json:"parent_id,omitempty"`
	Status    PlanStatus  `json:"status"`
	Items     []*PlanItem `json:"items"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// NewPlan constructs a draft plan; validates basic invariants.
func NewPlan(level PlanLevel, owner kernel.ID, period Period, title string, parentID *kernel.ID, now time.Time) (*Plan, error) {
	if !level.Valid() {
		return nil, ErrInvalidLevel
	}
	if !period.End.After(period.Start) {
		return nil, ErrInvalidPeriod
	}
	if title == "" {
		return nil, ErrEmptyTitle
	}
	return &Plan{
		ID:        kernel.NewID(),
		Level:     level,
		Owner:     owner,
		Period:    period,
		Title:     title,
		ParentID:  parentID,
		Status:    PlanDraft,
		Items:     []*PlanItem{},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// AddItem appends a new PlanItem; enforces weight-sum invariant (<=100).
func (p *Plan) AddItem(title string, weight int, now time.Time) (*PlanItem, error) {
	if p.Status == PlanClosed {
		return nil, ErrPlanClosed
	}
	if title == "" {
		return nil, ErrEmptyTitle
	}
	total := weight
	for _, it := range p.Items {
		total += it.Weight
	}
	if total > 100 {
		return nil, ErrWeightOverflow
	}
	item := &PlanItem{
		ID:    kernel.NewID(),
		Title: title, Weight: weight,
		Status:    ItemTodo,
		SortOrder: len(p.Items) + 1,
		CreatedAt: now, UpdatedAt: now,
	}
	p.Items = append(p.Items, item)
	p.UpdatedAt = now
	return item, nil
}

// CompleteItem sets an item to done. Invariant: cannot complete on a closed plan.
func (p *Plan) CompleteItem(itemID kernel.ID, note string, now time.Time) error {
	if p.Status == PlanClosed {
		return ErrPlanClosed
	}
	for _, it := range p.Items {
		if it.ID == itemID {
			it.ProgressPct = 100
			it.Status = ItemDone
			it.ProgressNote = note
			it.UpdatedAt = now
			p.UpdatedAt = now
			return nil
		}
	}
	return ErrItemNotFound
}

// UpdateItemProgress sets percent/status on an existing item.
func (p *Plan) UpdateItemProgress(itemID kernel.ID, pct int, status ItemStatus, note string, now time.Time) error {
	if p.Status == PlanClosed {
		return ErrPlanClosed
	}
	if pct < 0 || pct > 100 {
		return ErrInvalidStatus
	}
	for _, it := range p.Items {
		if it.ID == itemID {
			it.ProgressPct = pct
			it.Status = status
			it.ProgressNote = note
			it.UpdatedAt = now
			p.UpdatedAt = now
			return nil
		}
	}
	return ErrItemNotFound
}

// Activate moves the plan from draft → active.
func (p *Plan) Activate(now time.Time) error {
	if p.Status != PlanDraft {
		return ErrInvalidStatus
	}
	p.Status = PlanActive
	p.UpdatedAt = now
	return nil
}

// Close moves the plan to terminal closed status.
func (p *Plan) Close(now time.Time) error {
	if p.Status == PlanClosed {
		return ErrInvalidStatus
	}
	p.Status = PlanClosed
	p.UpdatedAt = now
	return nil
}

// OverallProgress is a derived value: weighted average of item percents.
func (p *Plan) OverallProgress() int {
	if len(p.Items) == 0 {
		return 0
	}
	totalW, sum := 0, 0
	for _, it := range p.Items {
		totalW += it.Weight
		sum += it.Weight * it.ProgressPct
	}
	if totalW == 0 {
		return 0
	}
	return sum / totalW
}
