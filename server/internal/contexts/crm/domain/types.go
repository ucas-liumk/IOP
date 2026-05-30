package domain

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Item is the seed aggregate for the crm module. Rename/extend as needed.
type Item struct {
	ID        kernel.ID `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}
