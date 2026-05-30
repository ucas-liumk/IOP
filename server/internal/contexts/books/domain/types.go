// Package domain holds the library (图书馆) entities. Kept deliberately light
// (anemic structs + service-layer logic) — this module is a CRUD-style app.
package domain

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Borrow status values.
const (
	StatusBorrowed = "borrowed"
	StatusReturned = "returned"
	StatusOverdue  = "overdue"
)

// Book is a catalog title. Total copies vs. currently-available copies are
// tracked so the catalog can show availability and block over-borrowing.
type Book struct {
	ID        kernel.ID `json:"id"`
	ISBN      string    `json:"isbn"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Publisher string    `json:"publisher"`
	Category  string    `json:"category"`
	Total     int       `json:"total"`
	Available int       `json:"available"`
	CoverURL  string    `json:"cover_url"`
	Location  string    `json:"location"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Borrow is a single borrowing record linking a book to a member.
type Borrow struct {
	ID         kernel.ID  `json:"id"`
	BookID     kernel.ID  `json:"book_id"`
	MemberID   kernel.ID  `json:"member_id"`
	BorrowedAt time.Time  `json:"borrowed_at"`
	DueAt      time.Time  `json:"due_at"`
	ReturnedAt *time.Time `json:"returned_at,omitempty"`
	Status     string     `json:"status"`
	// Query-time projections (joined from book), not persisted on the row.
	BookTitle  string `json:"book_title,omitempty"`
	BookAuthor string `json:"book_author,omitempty"`
	BookISBN   string `json:"book_isbn,omitempty"`
}

// BookFilter selects books for the paged catalog query.
type BookFilter struct {
	Search   string // matches title / author / isbn / publisher
	Category string
	Page     int
	PageSize int
}

// BorrowFilter selects borrow records. MemberID set => mine; empty => all.
type BorrowFilter struct {
	MemberID kernel.ID // "" = all borrows
	Status   string
}
