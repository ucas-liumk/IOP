// Package domain holds the task-management entities. Kept deliberately light
// (anemic structs + service-layer logic) — this module is a CRUD-style app, not
// a complex domain like OKR.
package domain

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Priority levels (stored as smallint).
const (
	PriorityNone   = 0
	PriorityLow    = 1
	PriorityMedium = 2
	PriorityHigh   = 3
)

// Task status values.
const (
	StatusTodo = "todo"
	StatusDone = "done"
)

// TaskList is a list/project that groups tasks.
type TaskList struct {
	ID        kernel.ID `json:"id"`
	Owner     kernel.ID `json:"owner"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	SortOrder int       `json:"sort_order"`
	Archived  bool      `json:"archived"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// TaskCount/DoneCount are query-time projections (not persisted on the row).
	TaskCount int `json:"task_count"`
	DoneCount int `json:"done_count"`
}

// Task is a single to-do. A task with ParentID set is a subtask.
type Task struct {
	ID          kernel.ID  `json:"id"`
	Owner       kernel.ID  `json:"owner"`
	ListID      *kernel.ID `json:"list_id,omitempty"`
	ParentID    *kernel.ID `json:"parent_id,omitempty"`
	Title       string     `json:"title"`
	Note        string     `json:"note"`
	Priority    int        `json:"priority"`
	Status      string     `json:"status"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Tags        []string   `json:"tags"`
	SortOrder   int        `json:"sort_order"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	// Subtasks is a query-time projection (loaded for a single-task fetch).
	Subtasks []*Task `json:"subtasks,omitempty"`
}

// Filter selects tasks for the list/smart-view queries.
type Filter struct {
	Owner       kernel.ID
	ListID      *kernel.ID // nil = any list
	Status      string     // "", "todo", "done"
	View        string     // "", "today", "next7", "overdue", "completed"
	IncludeSubs bool       // include subtasks in the flat result
	Tag         string
}
