package domain

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Topic names follow <bounded-context>.<event_name> convention.
const (
	TopicPlanCreated       = "okr.plan_created"
	TopicPlanItemAdded     = "okr.plan_item_added"
	TopicPlanItemCompleted = "okr.plan_item_completed"
	TopicPlanClosed        = "okr.plan_closed"
	TopicDailySubmitted    = "okr.daily_submitted"
	TopicWeeklySubmitted   = "okr.weekly_submitted"
	TopicWeeklyOverdue     = "okr.weekly_overdue"
)

type PlanCreated struct {
	PlanID      kernel.ID `json:"plan_id"`
	Level       string    `json:"level"`
	Owner       kernel.ID `json:"owner"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	Title       string    `json:"title"`
}

type PlanItemAdded struct {
	PlanID kernel.ID `json:"plan_id"`
	ItemID kernel.ID `json:"item_id"`
	Title  string    `json:"title"`
}

type PlanItemCompleted struct {
	PlanID   kernel.ID `json:"plan_id"`
	ItemID   kernel.ID `json:"item_id"`
	Progress int       `json:"progress"`
}

type PlanClosedEvent struct {
	PlanID   kernel.ID `json:"plan_id"`
	ClosedAt time.Time `json:"closed_at"`
}

type ReportSubmitted struct {
	ReportID  kernel.ID `json:"report_id"`
	Type      string    `json:"type"`
	Owner     kernel.ID `json:"owner"`
	PeriodEnd time.Time `json:"period_end"`
}

type WeeklyOverdue struct {
	Owner     kernel.ID `json:"owner"`
	PeriodEnd time.Time `json:"period_end"`
}
