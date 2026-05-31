// Package application is the use-case layer for the mindmap module.
package application

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/mindmap/domain"
	"github.com/leo/iop/server/internal/contexts/mindmap/infrastructure"
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

type CreateCmd struct {
	Owner kernel.ID
	Title string
	Data  json.RawMessage // optional; defaults to a single root node
}

func (s *Service) Create(ctx context.Context, cmd CreateCmd) (*domain.Mindmap, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, errors.New(errors.KindParam, "mindmap.title_required", "标题不能为空")
	}
	data := cmd.Data
	if !validJSON(data) {
		data = domain.DefaultData(title)
	}
	now := s.clock.Now()
	m := &domain.Mindmap{
		ID: kernel.NewID(), Owner: cmd.Owner, Title: title, Data: data,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "mindmap.create_failed", "创建思维导图失败", err)
	}
	_ = s.bus.Publish(ctx, "mindmap.created", map[string]any{"mindmap_id": m.ID, "owner": cmd.Owner})
	return m, nil
}

func (s *Service) List(ctx context.Context, owner kernel.ID) ([]*domain.Mindmap, error) {
	return s.repo.List(ctx, owner)
}

func (s *Service) Get(ctx context.Context, owner, id kernel.ID) (*domain.Mindmap, error) {
	m, err := s.repo.Get(ctx, owner, id)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "mindmap.db_error", "操作失败", err)
	}
	if m == nil {
		return nil, errors.New(errors.KindNotFound, "mindmap.not_found", "思维导图不存在")
	}
	return m, nil
}

// UpdateCmd carries the editable fields. Pointer fields left nil are unchanged.
type UpdateCmd struct {
	Owner kernel.ID
	ID    kernel.ID
	Title *string
	Data  *json.RawMessage
}

func (s *Service) Update(ctx context.Context, cmd UpdateCmd) (*domain.Mindmap, error) {
	m, err := s.repo.Get(ctx, cmd.Owner, cmd.ID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "mindmap.db_error", "操作失败", err)
	}
	if m == nil {
		return nil, errors.New(errors.KindNotFound, "mindmap.not_found", "思维导图不存在")
	}
	if cmd.Title != nil {
		t := strings.TrimSpace(*cmd.Title)
		if t == "" {
			return nil, errors.New(errors.KindParam, "mindmap.title_required", "标题不能为空")
		}
		m.Title = t
	}
	if cmd.Data != nil {
		if !validJSON(*cmd.Data) {
			return nil, errors.New(errors.KindParam, "mindmap.invalid_data", "导图数据格式错误")
		}
		m.Data = *cmd.Data
	}
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, notFoundOr(err, "mindmap.not_found", "思维导图不存在")
	}
	_ = s.bus.Publish(ctx, "mindmap.updated", map[string]any{"mindmap_id": m.ID, "owner": cmd.Owner})
	return m, nil
}

func (s *Service) Delete(ctx context.Context, owner, id kernel.ID) error {
	return notFoundOr(s.repo.Delete(ctx, owner, id), "mindmap.not_found", "思维导图不存在")
}

// ---- helpers ----

func validJSON(b json.RawMessage) bool {
	return len(b) > 0 && json.Valid(b)
}

func notFoundOr(err error, code, msg string) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, pgx.ErrNoRows) {
		return errors.New(errors.KindNotFound, code, msg)
	}
	return errors.Wrap(errors.KindDatabase, "mindmap.db_error", "操作失败", err)
}
