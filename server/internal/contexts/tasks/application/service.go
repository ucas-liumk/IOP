// Package application is the use-case layer for the tasks module.
package application

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/tasks/domain"
	"github.com/leo/iop/server/internal/contexts/tasks/infrastructure"
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

// ---- Lists ----

type CreateListCmd struct {
	Owner kernel.ID
	Name  string
	Color string
}

func (s *Service) CreateList(ctx context.Context, cmd CreateListCmd) (*domain.TaskList, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, errors.New(errors.KindParam, "tasks.list_name_required", "清单名称不能为空")
	}
	now := s.clock.Now()
	l := &domain.TaskList{
		ID: kernel.NewID(), Owner: cmd.Owner, Name: name, Color: cmd.Color,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreateList(ctx, l); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "tasks.create_list_failed", "创建清单失败", err)
	}
	_ = s.bus.Publish(ctx, "tasks.list_created", map[string]any{"list_id": l.ID, "owner": cmd.Owner})
	return l, nil
}

func (s *Service) ListLists(ctx context.Context, owner kernel.ID) ([]*domain.TaskList, error) {
	return s.repo.ListLists(ctx, owner)
}

type UpdateListCmd struct {
	Owner     kernel.ID
	ID        kernel.ID
	Name      string
	Color     string
	SortOrder int
	Archived  bool
}

func (s *Service) UpdateList(ctx context.Context, cmd UpdateListCmd) error {
	if strings.TrimSpace(cmd.Name) == "" {
		return errors.New(errors.KindParam, "tasks.list_name_required", "清单名称不能为空")
	}
	if err := s.repo.UpdateList(ctx, cmd.Owner, cmd.ID, cmd.Name, cmd.Color, cmd.SortOrder, cmd.Archived); err != nil {
		return notFoundOr(err, "tasks.list_not_found", "清单不存在")
	}
	return nil
}

func (s *Service) DeleteList(ctx context.Context, owner, id kernel.ID) error {
	return notFoundOr(s.repo.DeleteList(ctx, owner, id), "tasks.list_not_found", "清单不存在")
}

// ---- Tasks ----

type CreateTaskCmd struct {
	Owner    kernel.ID
	ListID   *kernel.ID
	ParentID *kernel.ID
	Title    string
	Note     string
	Priority int
	DueDate  *time.Time
	Tags     []string
}

func (s *Service) CreateTask(ctx context.Context, cmd CreateTaskCmd) (*domain.Task, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, errors.New(errors.KindParam, "tasks.title_required", "任务标题不能为空")
	}
	if cmd.Priority < 0 || cmd.Priority > 3 {
		cmd.Priority = 0
	}
	// Ownership checks: a member may only attach a task to their OWN list / parent
	// task. GetList/GetTask are owner-scoped, so a non-owned or missing id yields nil.
	if cmd.ListID != nil {
		if l, err := s.repo.GetList(ctx, cmd.Owner, *cmd.ListID); err != nil {
			return nil, err
		} else if l == nil {
			return nil, errors.New(errors.KindParam, "tasks.list_not_found", "清单不存在")
		}
	}
	if cmd.ParentID != nil {
		if p, err := s.repo.GetTask(ctx, cmd.Owner, *cmd.ParentID); err != nil {
			return nil, err
		} else if p == nil {
			return nil, errors.New(errors.KindParam, "tasks.parent_not_found", "父任务不存在")
		}
	}
	now := s.clock.Now()
	t := &domain.Task{
		ID: kernel.NewID(), Owner: cmd.Owner, ListID: cmd.ListID, ParentID: cmd.ParentID,
		Title: title, Note: cmd.Note, Priority: cmd.Priority, Status: domain.StatusTodo,
		DueDate: cmd.DueDate, Tags: normTags(cmd.Tags), CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreateTask(ctx, t); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "tasks.create_task_failed", "创建任务失败", err)
	}
	_ = s.bus.Publish(ctx, "tasks.task_created", map[string]any{"task_id": t.ID, "owner": cmd.Owner})
	return t, nil
}

func (s *Service) GetTask(ctx context.Context, owner, id kernel.ID) (*domain.Task, error) {
	return s.repo.GetTask(ctx, owner, id)
}

func (s *Service) ListTasks(ctx context.Context, f domain.Filter) ([]*domain.Task, error) {
	return s.repo.ListTasks(ctx, f)
}

func (s *Service) Counts(ctx context.Context, owner kernel.ID) (map[string]int, error) {
	return s.repo.CountByView(ctx, owner)
}

// UpdateTaskCmd carries the editable fields. Pointer fields left nil are unchanged.
type UpdateTaskCmd struct {
	Owner     kernel.ID
	ID        kernel.ID
	Title     *string
	Note      *string
	Priority  *int
	ListID    *kernel.ID
	ClearList bool
	DueDate   *time.Time
	ClearDue  bool
	Tags      *[]string
}

func (s *Service) UpdateTask(ctx context.Context, cmd UpdateTaskCmd) (*domain.Task, error) {
	t, err := s.repo.GetTask(ctx, cmd.Owner, cmd.ID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New(errors.KindNotFound, "tasks.task_not_found", "任务不存在")
	}
	if cmd.Title != nil {
		nt := strings.TrimSpace(*cmd.Title)
		if nt == "" {
			return nil, errors.New(errors.KindParam, "tasks.title_required", "任务标题不能为空")
		}
		t.Title = nt
	}
	if cmd.Note != nil {
		t.Note = *cmd.Note
	}
	if cmd.Priority != nil && *cmd.Priority >= 0 && *cmd.Priority <= 3 {
		t.Priority = *cmd.Priority
	}
	if cmd.ClearList {
		t.ListID = nil
	} else if cmd.ListID != nil {
		// Verify the target list belongs to this owner before reassigning.
		if l, err := s.repo.GetList(ctx, cmd.Owner, *cmd.ListID); err != nil {
			return nil, err
		} else if l == nil {
			return nil, errors.New(errors.KindParam, "tasks.list_not_found", "清单不存在")
		}
		t.ListID = cmd.ListID
	}
	if cmd.ClearDue {
		t.DueDate = nil
	} else if cmd.DueDate != nil {
		t.DueDate = cmd.DueDate
	}
	if cmd.Tags != nil {
		t.Tags = normTags(*cmd.Tags)
	}
	if err := s.repo.UpdateTask(ctx, t); err != nil {
		return nil, notFoundOr(err, "tasks.task_not_found", "任务不存在")
	}
	return t, nil
}

// SetStatus completes/uncompletes a task via a targeted status update (no
// full-row overwrite), so it never clobbers a concurrent edit.
func (s *Service) SetStatus(ctx context.Context, owner, id kernel.ID, done bool) (*domain.Task, error) {
	status := domain.StatusTodo
	var completedAt *time.Time
	if done {
		status = domain.StatusDone
		now := s.clock.Now()
		completedAt = &now
	}
	if err := s.repo.SetTaskStatus(ctx, owner, id, status, completedAt); err != nil {
		return nil, notFoundOr(err, "tasks.task_not_found", "任务不存在")
	}
	evt := "tasks.task_reopened"
	if done {
		evt = "tasks.task_completed"
	}
	_ = s.bus.Publish(ctx, evt, map[string]any{"task_id": id, "owner": owner})
	return s.repo.GetTask(ctx, owner, id)
}

func (s *Service) DeleteTask(ctx context.Context, owner, id kernel.ID) error {
	return notFoundOr(s.repo.DeleteTask(ctx, owner, id), "tasks.task_not_found", "任务不存在")
}

// ---- helpers ----

func normTags(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func notFoundOr(err error, code, msg string) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, pgx.ErrNoRows) {
		return errors.New(errors.KindNotFound, code, msg)
	}
	return errors.Wrap(errors.KindDatabase, "tasks.db_error", "操作失败", err)
}
