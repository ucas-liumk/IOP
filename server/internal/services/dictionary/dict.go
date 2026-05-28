package dictionary

import "context"

// Item is one entry in a dictionary type.
type Item struct {
	TypeCode  string `json:"type_code"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	Active    bool   `json:"active"`
}

// Repository is the persistence contract. M1: in-memory MemoryRepo.
// M3: replaced by pgRepo with Redis cache layer.
type Repository interface {
	List(ctx context.Context, typeCode string) ([]Item, error)
}

// MemoryRepo is an in-memory Repository used by M1 tests and dev seeding.
func MemoryRepo(seed map[string][]Item) Repository {
	return memRepo{data: seed}
}

type memRepo struct{ data map[string][]Item }

func (m memRepo) List(_ context.Context, typeCode string) ([]Item, error) {
	if items, ok := m.data[typeCode]; ok {
		return items, nil
	}
	return nil, ErrTypeNotFound
}
