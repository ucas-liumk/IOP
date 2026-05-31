// Package application is the use-case layer for the docs (知识库) module.
package application

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/docs/domain"
	"github.com/leo/iop/server/internal/contexts/docs/infrastructure"
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

// Tree returns the full knowledge-base tree (metadata only — no content).
// Folders and docs share one tree; orphaned nodes (whose parent was deleted out
// from under them, which CASCADE normally prevents) are surfaced at the root so
// they never silently vanish.
func (s *Service) Tree(ctx context.Context) ([]*domain.Node, error) {
	flat, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "docs.tree_failed", "加载知识库失败", err)
	}
	byID := make(map[kernel.ID]*domain.Node, len(flat))
	for _, n := range flat {
		n.Children = []*domain.Node{}
		byID[n.ID] = n
	}
	roots := []*domain.Node{}
	for _, n := range flat {
		if n.ParentID != nil {
			if p, ok := byID[*n.ParentID]; ok {
				p.Children = append(p.Children, n)
				continue
			}
		}
		roots = append(roots, n)
	}
	return roots, nil
}

func (s *Service) Get(ctx context.Context, id kernel.ID) (*domain.Node, error) {
	n, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "docs.get_failed", "加载文档失败", err)
	}
	if n == nil {
		return nil, errors.New(errors.KindNotFound, "docs.not_found", "文档不存在")
	}
	return n, nil
}

type CreateCmd struct {
	Actor    kernel.ID
	ParentID *kernel.ID
	Title    string
	Type     string // folder / doc
	Content  string
}

func (s *Service) Create(ctx context.Context, cmd CreateCmd) (*domain.Node, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, errors.New(errors.KindParam, "docs.title_required", "标题不能为空")
	}
	typ := cmd.Type
	if typ != domain.TypeFolder && typ != domain.TypeDoc {
		typ = domain.TypeDoc
	}
	// A non-root node must hang under an existing FOLDER (docs can't contain children).
	if cmd.ParentID != nil {
		p, err := s.repo.Get(ctx, *cmd.ParentID)
		if err != nil {
			return nil, errors.Wrap(errors.KindDatabase, "docs.get_failed", "加载父节点失败", err)
		}
		if p == nil {
			return nil, errors.New(errors.KindParam, "docs.parent_not_found", "父目录不存在")
		}
		if p.Type != domain.TypeFolder {
			return nil, errors.New(errors.KindParam, "docs.parent_not_folder", "只能在目录下创建")
		}
	}
	content := cmd.Content
	if typ == domain.TypeFolder {
		content = ""
	}
	now := s.clock.Now()
	n := &domain.Node{
		ID: kernel.NewID(), ParentID: cmd.ParentID, Title: title, Type: typ,
		Content: content, CreatedBy: cmd.Actor, UpdatedBy: cmd.Actor,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "docs.create_failed", "创建失败", err)
	}
	_ = s.bus.Publish(ctx, "docs.node_created", map[string]any{"id": n.ID, "type": n.Type, "actor": cmd.Actor})
	return n, nil
}

// Rename updates a node's title.
func (s *Service) Rename(ctx context.Context, actor, id kernel.ID, title string) error {
	t := strings.TrimSpace(title)
	if t == "" {
		return errors.New(errors.KindParam, "docs.title_required", "标题不能为空")
	}
	return notFoundOr(s.repo.UpdateMeta(ctx, id, actor, t), "docs.not_found", "节点不存在")
}

// SaveDoc persists a doc's content (and optionally renames it in the same call).
func (s *Service) SaveDoc(ctx context.Context, actor, id kernel.ID, title *string, content string) (*domain.Node, error) {
	if title != nil {
		if err := s.Rename(ctx, actor, id, *title); err != nil {
			return nil, err
		}
	}
	if err := s.repo.SaveContent(ctx, id, actor, content); err != nil {
		return nil, notFoundOr(err, "docs.not_found", "文档不存在")
	}
	_ = s.bus.Publish(ctx, "docs.doc_saved", map[string]any{"id": id, "actor": actor})
	return s.Get(ctx, id)
}

type MoveCmd struct {
	Actor    kernel.ID
	ID       kernel.ID
	ParentID *kernel.ID
	ToRoot   bool
	OrderNum int
}

func (s *Service) Move(ctx context.Context, cmd MoveCmd) error {
	if cmd.ParentID != nil {
		if *cmd.ParentID == cmd.ID {
			return errors.New(errors.KindParam, "docs.move_into_self", "不能移动到自身")
		}
		p, err := s.repo.Get(ctx, *cmd.ParentID)
		if err != nil {
			return errors.Wrap(errors.KindDatabase, "docs.get_failed", "加载父节点失败", err)
		}
		if p == nil {
			return errors.New(errors.KindParam, "docs.parent_not_found", "目标目录不存在")
		}
		if p.Type != domain.TypeFolder {
			return errors.New(errors.KindParam, "docs.parent_not_folder", "只能移动到目录下")
		}
		// Prevent cycles: target must not be a descendant of the moving node.
		if cyclic, err := s.isDescendant(ctx, *cmd.ParentID, cmd.ID); err != nil {
			return err
		} else if cyclic {
			return errors.New(errors.KindParam, "docs.move_cycle", "不能移动到自己的子节点下")
		}
	}
	var parent *kernel.ID
	if !cmd.ToRoot {
		parent = cmd.ParentID
	}
	return notFoundOr(s.repo.Move(ctx, cmd.ID, cmd.Actor, parent, cmd.OrderNum), "docs.not_found", "节点不存在")
}

// isDescendant reports whether candidate is the same as, or a descendant of, ancestor.
func (s *Service) isDescendant(ctx context.Context, candidate, ancestor kernel.ID) (bool, error) {
	cur := candidate
	for i := 0; i < 1000; i++ { // bound the walk defensively
		if cur == ancestor {
			return true, nil
		}
		n, err := s.repo.Get(ctx, cur)
		if err != nil {
			return false, errors.Wrap(errors.KindDatabase, "docs.get_failed", "加载节点失败", err)
		}
		if n == nil || n.ParentID == nil {
			return false, nil
		}
		cur = *n.ParentID
	}
	return false, nil
}

func (s *Service) Delete(ctx context.Context, id kernel.ID) error {
	return notFoundOr(s.repo.Delete(ctx, id), "docs.not_found", "节点不存在")
}

func notFoundOr(err error, code, msg string) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, pgx.ErrNoRows) {
		return errors.New(errors.KindNotFound, code, msg)
	}
	return errors.Wrap(errors.KindDatabase, "docs.db_error", "操作失败", err)
}
