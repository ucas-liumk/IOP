// Package domain holds the low-code online-form entities. Kept deliberately light
// (anemic structs + service-layer logic) — this is a CRUD/data-collection app, not
// a complex domain. A FormDef carries a JSONB field schema; an Entry is one
// submitted record whose Data is keyed by field key.
package domain

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Field types supported by the form designer.
const (
	FieldText     = "text"
	FieldTextarea = "textarea"
	FieldNumber   = "number"
	FieldDate     = "date"
	FieldSelect   = "select"
	FieldCheckbox = "checkbox"
	FieldMoney    = "money"
	FieldPhone    = "phone"
)

// FormDef status values.
const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

// validFieldTypes is the allow-list used when validating a definition's schema.
var validFieldTypes = map[string]bool{
	FieldText: true, FieldTextarea: true, FieldNumber: true, FieldDate: true,
	FieldSelect: true, FieldCheckbox: true, FieldMoney: true, FieldPhone: true,
}

// ValidFieldType reports whether t is a supported field type.
func ValidFieldType(t string) bool { return validFieldTypes[t] }

// Field is one column in a form's schema. Stored inside FormDef.Fields (JSONB).
type Field struct {
	Key      string   `json:"key"`               // stable identifier used as Entry.Data key
	Label    string   `json:"label"`             // 中文展示名
	Type     string   `json:"type"`              // text|textarea|number|date|select|checkbox|money|phone
	Required bool     `json:"required"`          // enforced on submit
	Options  []string `json:"options,omitempty"` // for select (choices)
}

// FormDef is a form definition (the schema + metadata).
type FormDef struct {
	ID        kernel.ID `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	Fields    []Field   `json:"fields"`
	Status    string    `json:"status"`
	CreatedBy kernel.ID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// EntryCount is a query-time projection (not persisted on the row).
	EntryCount int `json:"entry_count"`
}

// Entry is one submitted record. Data is keyed by Field.Key.
type Entry struct {
	ID          kernel.ID      `json:"id"`
	FormID      kernel.ID      `json:"form_id"`
	Data        map[string]any `json:"data"`
	SubmittedBy kernel.ID      `json:"submitted_by"`
	CreatedAt   time.Time      `json:"created_at"`
}

// EntryFilter selects entries for the paged list query.
type EntryFilter struct {
	FormID kernel.ID
	Search string // free-text match against the JSONB data payload
}
