// Package domain holds the knowledge-base (知识库 / 语雀/飞书文档/Notion-style)
// entities. A wiki is a tree of nodes: a node is either a folder (容器) or a doc
// (a page with rich-text/markdown content). Kept light (anemic structs +
// service-layer logic) — this is a CRUD/tree app, not a complex domain.
package domain

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Node types.
const (
	TypeFolder = "folder"
	TypeDoc    = "doc"
)

// Node is one entry in the knowledge-base tree. A folder groups children; a doc
// carries Content. ParentID nil means a top-level (root) node.
type Node struct {
	ID        kernel.ID  `json:"id"`
	ParentID  *kernel.ID `json:"parent_id,omitempty"`
	Title     string     `json:"title"`
	Type      string     `json:"type"`              // folder / doc
	Content   string     `json:"content,omitempty"` // html/markdown; only meaningful for docs
	OrderNum  int        `json:"order_num"`
	CreatedBy kernel.ID  `json:"created_by"`
	UpdatedBy kernel.ID  `json:"updated_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	// Children is a query-time projection used when building the tree.
	Children []*Node `json:"children,omitempty"`
}
