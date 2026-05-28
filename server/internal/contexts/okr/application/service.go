// Package application contains OKR use-cases.  Each use-case is a method on Service.
// Application services orchestrate domain objects but never embed business rules.
package application

import (
	"context"
	"time"

	"github.com/leo/iop/server/internal/contexts/okr/domain"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// Service is the OKR application layer; instantiated once in app.go.
type Service struct {
	plans   domain.PlanRepository
	reports domain.ReportRepository
	rollup  domain.RollupQuery
	bus     eventbus.Bus
	clock   kernel.Clock
	decomp  domain.PlanDecomposer
	cadence domain.Cadence
}

func NewService(plans domain.PlanRepository, reports domain.ReportRepository, rollup domain.RollupQuery, bus eventbus.Bus, clk kernel.Clock) *Service {
	return &Service{plans: plans, reports: reports, rollup: rollup, bus: bus, clock: clk}
}

// =============================================================================
// Plan use cases
// =============================================================================

type CreatePlanCmd struct {
	Level    string
	Owner    kernel.ID
	From, To time.Time
	Title    string
	ParentID *kernel.ID
}

func (s *Service) CreatePlan(ctx context.Context, cmd CreatePlanCmd) (*domain.Plan, error) {
	period := domain.Period{Start: cmd.From, End: cmd.To}
	p, err := domain.NewPlan(domain.PlanLevel(cmd.Level), cmd.Owner, period, cmd.Title, cmd.ParentID, s.clock.Now())
	if err != nil {
		return nil, err
	}
	if cmd.ParentID != nil {
		parent, err := s.plans.Get(ctx, *cmd.ParentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, domain.ErrPlanNotFound
		}
		if err := s.decomp.Validate(parent, p.Level, p.Period); err != nil {
			return nil, err
		}
	}
	if err := s.plans.Save(ctx, p); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "okr.plan.save_failed", "保存计划失败", err)
	}
	_ = s.bus.Publish(ctx, domain.TopicPlanCreated, domain.PlanCreated{
		PlanID: p.ID, Level: string(p.Level), Owner: p.Owner,
		PeriodStart: p.Period.Start, PeriodEnd: p.Period.End, Title: p.Title,
	})
	return p, nil
}

type AddItemCmd struct {
	PlanID kernel.ID
	Title  string
	Weight int
}

func (s *Service) AddPlanItem(ctx context.Context, cmd AddItemCmd) (*domain.PlanItem, error) {
	p, err := s.plans.Get(ctx, cmd.PlanID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrPlanNotFound
	}
	item, err := p.AddItem(cmd.Title, cmd.Weight, s.clock.Now())
	if err != nil {
		return nil, err
	}
	if err := s.plans.Save(ctx, p); err != nil {
		return nil, err
	}
	_ = s.bus.Publish(ctx, domain.TopicPlanItemAdded, domain.PlanItemAdded{
		PlanID: p.ID, ItemID: item.ID, Title: item.Title,
	})
	return item, nil
}

type CompleteItemCmd struct {
	PlanID kernel.ID
	ItemID kernel.ID
	Note   string
}

func (s *Service) CompleteItem(ctx context.Context, cmd CompleteItemCmd) error {
	p, err := s.plans.Get(ctx, cmd.PlanID)
	if err != nil {
		return err
	}
	if p == nil {
		return domain.ErrPlanNotFound
	}
	if err := p.CompleteItem(cmd.ItemID, cmd.Note, s.clock.Now()); err != nil {
		return err
	}
	if err := s.plans.Save(ctx, p); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, domain.TopicPlanItemCompleted, domain.PlanItemCompleted{
		PlanID: p.ID, ItemID: cmd.ItemID, Progress: 100,
	})
	return nil
}

func (s *Service) ClosePlan(ctx context.Context, id kernel.ID) error {
	p, err := s.plans.Get(ctx, id)
	if err != nil {
		return err
	}
	if p == nil {
		return domain.ErrPlanNotFound
	}
	if err := p.Close(s.clock.Now()); err != nil {
		return err
	}
	if err := s.plans.Save(ctx, p); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, domain.TopicPlanClosed, domain.PlanClosedEvent{
		PlanID: p.ID, ClosedAt: s.clock.Now(),
	})
	return nil
}

func (s *Service) ListMyPlans(ctx context.Context, owner kernel.ID, level string, p kernel.Pagination) ([]*domain.Plan, error) {
	return s.plans.List(ctx, domain.PlanFilter{Owner: owner, Level: domain.PlanLevel(level), Pagination: p})
}

func (s *Service) GetPlan(ctx context.Context, id kernel.ID) (*domain.Plan, error) {
	return s.plans.Get(ctx, id)
}

// =============================================================================
// Report use cases
// =============================================================================

type SubmitDailyCmd struct {
	Owner   kernel.ID
	Day     time.Time
	Summary string
	Entries []EntryInput
}

type EntryInput struct {
	Title        string
	Detail       string
	ProgressNote string
	PlanItemID   *kernel.ID
}

func (s *Service) SubmitDaily(ctx context.Context, cmd SubmitDailyCmd) (*domain.Report, error) {
	day := truncateToDay(cmd.Day)
	period := domain.Period{Start: day, End: day}
	if existing, _ := s.reports.GetByOwnerAndPeriod(ctx, cmd.Owner, domain.ReportDaily, period); existing != nil {
		return nil, domain.ErrReportAlreadySent
	}
	r, err := domain.NewDailyReport(cmd.Owner, period, cmd.Summary, s.clock.Now())
	if err != nil {
		return nil, err
	}
	for _, e := range cmd.Entries {
		r.AddEntry(e.Title, e.Detail, e.ProgressNote, e.PlanItemID)
	}
	if err := s.reports.Save(ctx, r); err != nil {
		return nil, err
	}
	_ = s.bus.Publish(ctx, domain.TopicDailySubmitted, domain.ReportSubmitted{
		ReportID: r.ID, Type: "daily", Owner: r.Owner, PeriodEnd: r.Period.End,
	})
	return r, nil
}

type SubmitWeeklyCmd struct {
	Owner       kernel.ID
	WeekContains time.Time // any day in the target week
	Summary     string
	Entries     []EntryInput
}

func (s *Service) SubmitWeekly(ctx context.Context, cmd SubmitWeeklyCmd) (*domain.Report, error) {
	week := s.cadence.CurrentWeek(cmd.WeekContains)
	if existing, _ := s.reports.GetByOwnerAndPeriod(ctx, cmd.Owner, domain.ReportWeekly, week); existing != nil {
		return nil, domain.ErrReportAlreadySent
	}
	r, err := domain.NewWeeklyReport(cmd.Owner, week, cmd.Summary, s.clock.Now())
	if err != nil {
		return nil, err
	}
	for _, e := range cmd.Entries {
		r.AddEntry(e.Title, e.Detail, e.ProgressNote, e.PlanItemID)
	}
	if err := s.reports.Save(ctx, r); err != nil {
		return nil, err
	}
	_ = s.bus.Publish(ctx, domain.TopicWeeklySubmitted, domain.ReportSubmitted{
		ReportID: r.ID, Type: "weekly", Owner: r.Owner, PeriodEnd: r.Period.End,
	})
	return r, nil
}

func (s *Service) ListReports(ctx context.Context, owner kernel.ID, typ string, p kernel.Pagination) ([]*domain.Report, error) {
	return s.reports.List(ctx, domain.ReportFilter{Owner: owner, Type: domain.ReportType(typ), Pagination: p})
}

func (s *Service) GetReport(ctx context.Context, id kernel.ID) (*domain.Report, error) {
	return s.reports.Get(ctx, id)
}

func (s *Service) CommentReport(ctx context.Context, reportID, author kernel.ID, body string) error {
	if body == "" {
		return errors.New(errors.KindParam, "okr.report.empty_comment", "评论不能为空")
	}
	return s.reports.Comment(ctx, reportID, author, body)
}

func (s *Service) ListReportComments(ctx context.Context, reportID kernel.ID) ([]domain.Comment, error) {
	return s.reports.ListComments(ctx, reportID)
}

func (s *Service) RollupWeekly(ctx context.Context, week time.Time) ([]domain.RollupRow, error) {
	period := s.cadence.CurrentWeek(week)
	return s.rollup.WeeklyByDept(ctx, period.Start.Format("2006-01-02"), period.End.Format("2006-01-02"))
}

// =============================================================================
// Periodic reminders
// =============================================================================

// RemindOverdue publishes WeeklyOverdue for each member without a weekly report
// for the prior week. Triggered by cron.
func (s *Service) RemindOverdue(ctx context.Context) error {
	week := s.cadence.CurrentWeek(s.clock.Now().AddDate(0, 0, -7))
	rows, err := s.rollup.WeeklyByDept(ctx, week.Start.Format("2006-01-02"), week.End.Format("2006-01-02"))
	if err != nil {
		return err
	}
	for _, r := range rows {
		if r.Submitted {
			continue
		}
		_ = s.bus.Publish(ctx, domain.TopicWeeklyOverdue, domain.WeeklyOverdue{
			Owner: r.MemberID, PeriodEnd: week.End,
		})
	}
	return nil
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
