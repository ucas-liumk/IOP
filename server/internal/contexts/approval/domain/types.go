// Package domain holds the approval-center (审批中心 / 钉钉·飞书审批-style) entities.
// Deliberately light (anemic structs + service-layer flow logic): a Form is a
// reusable template (custom field schema + an ordered approval flow); submitting a
// Form snapshots its fields+flow into an Instance and generates the first node's
// approval Tasks; approvers Act on their Task to advance or finish the Instance.
package domain

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Instance / task statuses.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
	StatusCanceled = "canceled"
	StatusRead     = "read" // cc task acknowledged
)

// Form status.
const (
	FormActive   = "active"
	FormDisabled = "disabled"
)

// Flow node types.
const (
	NodeApprove = "approve"
	NodeCC      = "cc"
)

// Assignee resolution strategy for a flow node.
const (
	AssigneeUser       = "user"        // a specific member id
	AssigneeRole       = "role"        // every member granted role_code
	AssigneeDeptLeader = "dept_leader" // the initiator's department leader
)

// Countersign mode for an approve node.
const (
	ModeOr  = "or"  // any one approver decides
	ModeAnd = "and" // every approver must approve
)

// Field is one custom field in a form's schema.
type Field struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`     // text / textarea / number / date / select / radio
	Required bool     `json:"required"` //
	Options  []string `json:"options"`  // for select / radio
}

// FlowNode is one step in the approval flow.
type FlowNode struct {
	Type         string `json:"type"`          // approve / cc
	AssigneeType string `json:"assignee_type"` // user / role / dept_leader
	AssigneeID   string `json:"assignee_id"`   // member id (when assignee_type=user)
	RoleCode     string `json:"role_code"`     // role code (when assignee_type=role)
	Mode         string `json:"mode"`          // and / or (approve nodes only)
}

// Form is a reusable approval template.
type Form struct {
	ID          kernel.ID  `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Icon        string     `json:"icon"`
	Description string     `json:"description"`
	Fields      []Field    `json:"fields"`
	Flow        []FlowNode `json:"flow"`
	Status      string     `json:"status"`
	CreatedBy   kernel.ID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Instance is one submitted approval. It snapshots the form's name/fields/flow so
// that later edits to the template never mutate a historical request.
type Instance struct {
	ID          kernel.ID      `json:"id"`
	FormID      kernel.ID      `json:"form_id"`
	FormName    string         `json:"form_name"`
	Fields      []Field        `json:"fields"`
	Data        map[string]any `json:"data"`
	Flow        []FlowNode     `json:"flow"`
	InitiatorID kernel.ID      `json:"initiator_id"`
	Status      string         `json:"status"`
	CurrentNode int            `json:"current_node"`
	CreatedAt   time.Time      `json:"created_at"`
	FinishedAt  *time.Time     `json:"finished_at,omitempty"`
	// Query-time projections (not persisted on the row).
	InitiatorName string  `json:"initiator_name,omitempty"`
	Tasks         []*Task `json:"tasks,omitempty"`
	// CanCancel is true when the viewing member is the initiator and the instance
	// is still pending (i.e. withdrawable). Viewer-relative; set by GetInstance.
	CanCancel bool `json:"can_cancel"`
}

// Task is one approver/cc node-task generated for an instance node.
type Task struct {
	ID         kernel.ID  `json:"id"`
	InstanceID kernel.ID  `json:"instance_id"`
	NodeIndex  int        `json:"node_index"`
	AssigneeID kernel.ID  `json:"assignee_id"`
	Type       string     `json:"type"` // approve / cc
	Mode       string     `json:"mode"`
	Status     string     `json:"status"`
	Comment    string     `json:"comment"`
	ActedAt    *time.Time `json:"acted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	// Query-time projections.
	AssigneeName   string `json:"assignee_name,omitempty"`
	FormName       string `json:"form_name,omitempty"`
	InstanceStatus string `json:"instance_status,omitempty"`
	// Mine is true when the viewing member is this task's assignee. Viewer-relative;
	// set by GetInstance so the UI can show action buttons without knowing the
	// caller's member id.
	Mine bool `json:"mine"`
}

// TaskQuery selects the inbox lists for a member.
type TaskQuery struct {
	Member kernel.ID
	Type   string // "todo" | "done" | "initiated" | "cc"
}
