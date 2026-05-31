// Package application is the use-case layer for the project module.
package application

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/project/domain"
	"github.com/leo/iop/server/internal/contexts/project/infrastructure"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/kernel"
)

type Service struct {
	repo  *infrastructure.Repo
	bus   eventbus.Bus
	clock kernel.Clock
}

func NewService(repo *infrastructure.Repo, bus eventbus.Bus, clk kernel.Clock) *Service {
	return &Service{repo: repo, bus: bus, clock: clk}
}

// ---- Projects ----

type CreateProjectCmd struct {
	CreatedBy   kernel.ID
	Name        string
	Description string
}

func (s *Service) CreateProject(ctx context.Context, cmd CreateProjectCmd) (*domain.Project, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, errors.New(errors.KindParam, "project.name_required", "项目名称不能为空")
	}
	now := s.clock.Now()
	p := &domain.Project{
		ID: kernel.NewID(), Name: name, Description: cmd.Description,
		Status: domain.StatusActive, CreatedBy: cmd.CreatedBy, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreateProject(ctx, p); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.create_failed", "创建项目失败", err)
	}
	// Seed default Kanban columns so a new board is immediately usable.
	for i, cn := range []string{"待办", "进行中", "已完成"} {
		col := &domain.Column{
			ID: kernel.NewID(), ProjectID: p.ID, Name: cn, OrderNum: i, CreatedAt: now, UpdatedAt: now,
		}
		if err := s.repo.CreateColumn(ctx, col); err != nil {
			return nil, errors.Wrap(errors.KindDatabase, "project.seed_columns_failed", "初始化看板列失败", err)
		}
		p.Columns = append(p.Columns, col)
	}
	_ = s.bus.Publish(ctx, "project.created", map[string]any{"project_id": p.ID, "created_by": cmd.CreatedBy})
	return p, nil
}

func (s *Service) ListProjects(ctx context.Context) ([]*domain.Project, error) {
	return s.repo.ListProjects(ctx)
}

func (s *Service) GetBoard(ctx context.Context, id kernel.ID) (*domain.Project, error) {
	p, err := s.repo.GetBoard(ctx, id)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.db_error", "加载看板失败", err)
	}
	if p == nil {
		return nil, errors.New(errors.KindNotFound, "project.not_found", "项目不存在")
	}
	return p, nil
}

type UpdateProjectCmd struct {
	ID          kernel.ID
	Name        *string
	Description *string
	Status      *string
}

func (s *Service) UpdateProject(ctx context.Context, cmd UpdateProjectCmd) (*domain.Project, error) {
	p, err := s.repo.GetProject(ctx, cmd.ID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.db_error", "操作失败", err)
	}
	if p == nil {
		return nil, errors.New(errors.KindNotFound, "project.not_found", "项目不存在")
	}
	if cmd.Name != nil {
		n := strings.TrimSpace(*cmd.Name)
		if n == "" {
			return nil, errors.New(errors.KindParam, "project.name_required", "项目名称不能为空")
		}
		p.Name = n
	}
	if cmd.Description != nil {
		p.Description = *cmd.Description
	}
	if cmd.Status != nil {
		switch *cmd.Status {
		case domain.StatusActive, domain.StatusArchived:
			p.Status = *cmd.Status
		default:
			return nil, errors.New(errors.KindParam, "project.invalid_status", "项目状态无效")
		}
	}
	if err := s.repo.UpdateProject(ctx, p.ID, p.Name, p.Description, p.Status); err != nil {
		return nil, notFoundOr(err, "project.not_found", "项目不存在")
	}
	return p, nil
}

func (s *Service) DeleteProject(ctx context.Context, id kernel.ID) error {
	return notFoundOr(s.repo.DeleteProject(ctx, id), "project.not_found", "项目不存在")
}

// ---- Columns ----

type CreateColumnCmd struct {
	ProjectID kernel.ID
	Name      string
}

func (s *Service) CreateColumn(ctx context.Context, cmd CreateColumnCmd) (*domain.Column, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, errors.New(errors.KindParam, "project.column_name_required", "列名称不能为空")
	}
	if p, err := s.repo.GetProject(ctx, cmd.ProjectID); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.db_error", "操作失败", err)
	} else if p == nil {
		return nil, errors.New(errors.KindNotFound, "project.not_found", "项目不存在")
	}
	order, err := s.repo.NextColumnOrder(ctx, cmd.ProjectID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.db_error", "操作失败", err)
	}
	now := s.clock.Now()
	col := &domain.Column{
		ID: kernel.NewID(), ProjectID: cmd.ProjectID, Name: name, OrderNum: order, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreateColumn(ctx, col); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.create_column_failed", "创建列失败", err)
	}
	return col, nil
}

type UpdateColumnCmd struct {
	ID       kernel.ID
	Name     *string
	OrderNum *int
}

func (s *Service) UpdateColumn(ctx context.Context, cmd UpdateColumnCmd) (*domain.Column, error) {
	col, err := s.repo.GetColumn(ctx, cmd.ID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.db_error", "操作失败", err)
	}
	if col == nil {
		return nil, errors.New(errors.KindNotFound, "project.column_not_found", "列不存在")
	}
	if cmd.Name != nil {
		n := strings.TrimSpace(*cmd.Name)
		if n == "" {
			return nil, errors.New(errors.KindParam, "project.column_name_required", "列名称不能为空")
		}
		col.Name = n
	}
	if cmd.OrderNum != nil {
		col.OrderNum = *cmd.OrderNum
	}
	if err := s.repo.UpdateColumn(ctx, col.ID, col.Name, col.OrderNum); err != nil {
		return nil, notFoundOr(err, "project.column_not_found", "列不存在")
	}
	return col, nil
}

func (s *Service) DeleteColumn(ctx context.Context, id kernel.ID) error {
	return notFoundOr(s.repo.DeleteColumn(ctx, id), "project.column_not_found", "列不存在")
}

// ---- Cards ----

type CreateCardCmd struct {
	ProjectID   kernel.ID
	ColumnID    kernel.ID
	Title       string
	Description string
	AssigneeID  *kernel.ID
	DueDate     *time.Time
	Priority    int
}

func (s *Service) CreateCard(ctx context.Context, cmd CreateCardCmd) (*domain.Card, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, errors.New(errors.KindParam, "project.card_title_required", "卡片标题不能为空")
	}
	if cmd.Priority < 0 || cmd.Priority > 3 {
		cmd.Priority = 0
	}
	// Verify the target column exists and belongs to the project.
	col, err := s.repo.GetColumn(ctx, cmd.ColumnID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.db_error", "操作失败", err)
	}
	if col == nil || col.ProjectID != cmd.ProjectID {
		return nil, errors.New(errors.KindParam, "project.column_not_found", "列不存在")
	}
	order, err := s.repo.NextCardOrder(ctx, cmd.ColumnID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.db_error", "操作失败", err)
	}
	now := s.clock.Now()
	card := &domain.Card{
		ID: kernel.NewID(), ProjectID: cmd.ProjectID, ColumnID: cmd.ColumnID,
		Title: title, Description: cmd.Description, AssigneeID: cmd.AssigneeID,
		DueDate: cmd.DueDate, Priority: cmd.Priority, OrderNum: order, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreateCard(ctx, card); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.create_card_failed", "创建卡片失败", err)
	}
	_ = s.bus.Publish(ctx, "project.card_created", map[string]any{"card_id": card.ID, "project_id": cmd.ProjectID})
	return card, nil
}

func (s *Service) GetCard(ctx context.Context, id kernel.ID) (*domain.Card, error) {
	c, err := s.repo.GetCard(ctx, id)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.db_error", "操作失败", err)
	}
	if c == nil {
		return nil, errors.New(errors.KindNotFound, "project.card_not_found", "卡片不存在")
	}
	return c, nil
}

// UpdateCardCmd carries the editable fields. Pointer fields left nil are unchanged.
type UpdateCardCmd struct {
	ID            kernel.ID
	Title         *string
	Description   *string
	Priority      *int
	AssigneeID    *kernel.ID
	ClearAssignee bool
	DueDate       *time.Time
	ClearDue      bool
}

func (s *Service) UpdateCard(ctx context.Context, cmd UpdateCardCmd) (*domain.Card, error) {
	c, err := s.repo.GetCard(ctx, cmd.ID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.db_error", "操作失败", err)
	}
	if c == nil {
		return nil, errors.New(errors.KindNotFound, "project.card_not_found", "卡片不存在")
	}
	if cmd.Title != nil {
		t := strings.TrimSpace(*cmd.Title)
		if t == "" {
			return nil, errors.New(errors.KindParam, "project.card_title_required", "卡片标题不能为空")
		}
		c.Title = t
	}
	if cmd.Description != nil {
		c.Description = *cmd.Description
	}
	if cmd.Priority != nil && *cmd.Priority >= 0 && *cmd.Priority <= 3 {
		c.Priority = *cmd.Priority
	}
	if cmd.ClearAssignee {
		c.AssigneeID = nil
	} else if cmd.AssigneeID != nil {
		c.AssigneeID = cmd.AssigneeID
	}
	if cmd.ClearDue {
		c.DueDate = nil
	} else if cmd.DueDate != nil {
		c.DueDate = cmd.DueDate
	}
	if err := s.repo.UpdateCard(ctx, c); err != nil {
		return nil, notFoundOr(err, "project.card_not_found", "卡片不存在")
	}
	return c, nil
}

// MoveCmd moves a card to a target column at a target order index.
type MoveCmd struct {
	CardID   kernel.ID
	ColumnID kernel.ID
	OrderNum int
}

func (s *Service) MoveCard(ctx context.Context, cmd MoveCmd) (*domain.Card, error) {
	c, err := s.repo.GetCard(ctx, cmd.CardID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.db_error", "操作失败", err)
	}
	if c == nil {
		return nil, errors.New(errors.KindNotFound, "project.card_not_found", "卡片不存在")
	}
	// Target column must exist within the SAME project (no cross-board moves).
	col, err := s.repo.GetColumn(ctx, cmd.ColumnID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "project.db_error", "操作失败", err)
	}
	if col == nil || col.ProjectID != c.ProjectID {
		return nil, errors.New(errors.KindParam, "project.column_not_found", "列不存在")
	}
	order := cmd.OrderNum
	if order < 0 {
		order = 0
	}
	if err := s.repo.MoveCard(ctx, cmd.CardID, cmd.ColumnID, order); err != nil {
		return nil, notFoundOr(err, "project.card_not_found", "卡片不存在")
	}
	_ = s.bus.Publish(ctx, "project.card_moved", map[string]any{"card_id": cmd.CardID, "column_id": cmd.ColumnID})
	return s.repo.GetCard(ctx, cmd.CardID)
}

func (s *Service) DeleteCard(ctx context.Context, id kernel.ID) error {
	return notFoundOr(s.repo.DeleteCard(ctx, id), "project.card_not_found", "卡片不存在")
}

// ---- helpers ----

func notFoundOr(err error, code, msg string) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, pgx.ErrNoRows) {
		return errors.New(errors.KindNotFound, code, msg)
	}
	return errors.Wrap(errors.KindDatabase, "project.db_error", "操作失败", err)
}
