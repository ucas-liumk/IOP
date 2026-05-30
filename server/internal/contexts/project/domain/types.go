// Package domain holds the project-management (Kanban / Teambition-style)
// entities. Kept deliberately light (anemic structs + service-layer logic) —
// this is a CRUD-style board app, not a complex domain.
package domain

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Project status values.
const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

// Card priority levels (stored as smallint).
const (
	PriorityNone   = 0
	PriorityLow    = 1
	PriorityMedium = 2
	PriorityHigh   = 3
)

// Project is a Kanban board. Tenant-scoped; created_by is the member that owns it.
type Project struct {
	ID          kernel.ID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedBy   kernel.ID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Columns/CardCount are query-time projections (not persisted on the row).
	Columns   []*Column `json:"columns,omitempty"`
	CardCount int       `json:"card_count"`
}

// Column is a vertical lane on a board (e.g. 待办 / 进行中 / 已完成).
type Column struct {
	ID        kernel.ID `json:"id"`
	ProjectID kernel.ID `json:"project_id"`
	Name      string    `json:"name"`
	OrderNum  int       `json:"order_num"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Cards is a query-time projection (loaded for board detail).
	Cards []*Card `json:"cards,omitempty"`
}

// Card is a task/ticket that lives in a column.
type Card struct {
	ID          kernel.ID  `json:"id"`
	ProjectID   kernel.ID  `json:"project_id"`
	ColumnID    kernel.ID  `json:"column_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	AssigneeID  *kernel.ID `json:"assignee_id,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Priority    int        `json:"priority"`
	OrderNum    int        `json:"order_num"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
