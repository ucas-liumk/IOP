// Package domain holds the mind-map (思维导图 / ProcessOn-style) entities.
// Deliberately light: a mindmap is a title plus a JSON node tree blob, owned by
// a member and isolated per tenant.
package domain

import (
	"encoding/json"
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Mindmap is a single mind map. Data is the raw node tree as produced/consumed
// by the frontend editor (simple-mind-map): {data:{text},children:[...]}.
// Stored as JSONB; carried here as json.RawMessage so the service never needs to
// understand the tree shape — it just round-trips it.
type Mindmap struct {
	ID        kernel.ID       `json:"id"`
	Owner     kernel.ID       `json:"created_by"`
	Title     string          `json:"title"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// DefaultData is the seed node tree used when a map is created without one.
func DefaultData(title string) json.RawMessage {
	root, _ := json.Marshal(map[string]any{
		"data":     map[string]any{"text": title},
		"children": []any{},
	})
	return root
}
