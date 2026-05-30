package application

import (
	"context"

	"github.com/leo/iop/server/internal/contexts/crm/domain"
	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

type Service struct {
	tenant *tenantdb.TenantDB
	bus    eventbus.Bus
	clock  kernel.Clock
}

func NewService(tenant *tenantdb.TenantDB, bus eventbus.Bus, clk kernel.Clock) *Service {
	return &Service{tenant: tenant, bus: bus, clock: clk}
}

// CreateItem inserts a row into crm_item and publishes crm.item_created.
func (s *Service) CreateItem(ctx context.Context, title, body string) (*domain.Item, error) {
	// TODO: replace with real DB-backed implementation.
	item := &domain.Item{
		ID:        kernel.NewID(),
		Title:     title,
		Body:      body,
		CreatedAt: s.clock.Now(),
	}
	_ = s.bus.Publish(ctx, "crm.item_created", map[string]any{
		"item_id": item.ID, "title": item.Title,
	})
	return item, nil
}
