package kernel

import (
	"fmt"

	"github.com/google/uuid"
)

// ID is the canonical 128-bit identifier used across all aggregates and entities.
// Format: UUID v7 (time-ordered, sortable). String form: canonical 8-4-4-4-12 hex.
type ID string

// NewID returns a fresh UUID v7. Time-ordered: lexicographic sort == chronological.
func NewID() ID {
	u, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("kernel: cannot generate UUID v7: %v", err))
	}
	return ID(u.String())
}

// ParseID validates and normalizes an external ID string.
func ParseID(s string) (ID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return "", fmt.Errorf("invalid id %q: %w", s, err)
	}
	return ID(u.String()), nil
}

func (id ID) String() string { return string(id) }
