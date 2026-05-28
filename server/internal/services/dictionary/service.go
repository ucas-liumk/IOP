package dictionary

import (
	"context"

	"github.com/leo/iop/server/internal/shared/errors"
)

// ErrTypeNotFound is returned when a type_code has no items.
var ErrTypeNotFound = errors.New(errors.KindNotFound, "dictionary.type.not_found", "字典类型不存在")

// Service is the public API of the dictionary service.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

// Lookup returns active items for a type_code.
// M3 will add Redis caching + tenant override merging.
func (s *Service) Lookup(ctx context.Context, typeCode string) ([]Item, error) {
	items, err := s.repo.List(ctx, typeCode)
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(items))
	for _, it := range items {
		if it.Active {
			out = append(out, it)
		}
	}
	return out, nil
}
