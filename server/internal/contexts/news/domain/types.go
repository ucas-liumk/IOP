// Package domain holds the 时政资讯 (gov news CMS) entities. Kept light
// (anemic structs + service-layer logic) — this is a CRUD-style content module
// modeled on a government news portal / 人民网频道.
package domain

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Article status values.
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

// Category groups articles (e.g. 时政要闻 / 政策解读 / 通知公告).
type Category struct {
	ID       kernel.ID `json:"id"`
	Name     string    `json:"name"`
	OrderNum int       `json:"order_num"`
	// ArticleCount is a query-time projection (not persisted on the row).
	ArticleCount int `json:"article_count"`
}

// Article is a single news item.
type Article struct {
	ID          kernel.ID  `json:"id"`
	CategoryID  *kernel.ID `json:"category_id,omitempty"`
	Title       string     `json:"title"`
	Summary     string     `json:"summary"`
	Content     string     `json:"content"`
	CoverURL    string     `json:"cover_url"`
	Author      string     `json:"author"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Views       int        `json:"views"`
	CreatedBy   kernel.ID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	// CategoryName is a query-time projection (joined from category).
	CategoryName string `json:"category_name,omitempty"`
}

// Filter selects articles for the management / reader queries.
type Filter struct {
	CategoryID    *kernel.ID // nil = any category
	Status        string     // "", "draft", "published"
	PublishedOnly bool       // reader feed: only published rows
	Keyword       string     // title contains (management search)
}
