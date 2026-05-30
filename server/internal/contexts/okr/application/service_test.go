package application

import (
	"context"
	"testing"
	"time"

	"github.com/leo/iop/server/internal/contexts/okr/domain"
	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/kernel"
)

type fakePlanRepo struct {
	plans map[kernel.ID]*domain.Plan
}

func (r *fakePlanRepo) Save(context.Context, *domain.Plan) error { return nil }
func (r *fakePlanRepo) Get(_ context.Context, id kernel.ID) (*domain.Plan, error) {
	return r.plans[id], nil
}
func (r *fakePlanRepo) List(context.Context, domain.PlanFilter) ([]*domain.Plan, error) {
	return nil, nil
}
func (r *fakePlanRepo) ListChildren(context.Context, kernel.ID) ([]*domain.Plan, error) {
	return nil, nil
}

func TestCreatePlanRejectsCrossOwnerParent(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	parentOwner := kernel.NewID()
	childOwner := kernel.NewID()
	parent, err := domain.NewPlan(domain.LevelMonth, parentOwner, domain.Period{
		Start: now,
		End:   now.AddDate(0, 1, 0),
	}, "parent", nil, now)
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	svc := NewService(&fakePlanRepo{plans: map[kernel.ID]*domain.Plan{parent.ID: parent}},
		nil, nil, eventbus.NewInprocBus(1), kernel.NewFakeClock(now))
	_, err = svc.CreatePlan(context.Background(), CreatePlanCmd{
		Level:    string(domain.LevelWeek),
		Owner:    childOwner,
		From:     now.AddDate(0, 0, 1),
		To:       now.AddDate(0, 0, 7),
		Title:    "child",
		ParentID: &parent.ID,
	})
	if err == nil {
		t.Fatal("expected cross-owner parent to be rejected")
	}
}
