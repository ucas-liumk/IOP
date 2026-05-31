// Package application is the use-case layer for the news (时政资讯) module.
package application

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/news/domain"
	"github.com/leo/iop/server/internal/contexts/news/infrastructure"
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

// PagedArticles is the paged reader/management result.
type PagedArticles struct {
	Items    []*domain.Article `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// ---- Categories ----

type CreateCategoryCmd struct {
	Name     string
	OrderNum int
}

func (s *Service) CreateCategory(ctx context.Context, cmd CreateCategoryCmd) (*domain.Category, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, errors.New(errors.KindParam, "news.category_name_required", "栏目名称不能为空")
	}
	c := &domain.Category{ID: kernel.NewID(), Name: name, OrderNum: cmd.OrderNum}
	if err := s.repo.CreateCategory(ctx, c); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "news.create_category_failed", "创建栏目失败", err)
	}
	return c, nil
}

func (s *Service) ListCategories(ctx context.Context) ([]*domain.Category, error) {
	return s.repo.ListCategories(ctx)
}

type UpdateCategoryCmd struct {
	ID       kernel.ID
	Name     string
	OrderNum int
}

func (s *Service) UpdateCategory(ctx context.Context, cmd UpdateCategoryCmd) error {
	if strings.TrimSpace(cmd.Name) == "" {
		return errors.New(errors.KindParam, "news.category_name_required", "栏目名称不能为空")
	}
	return notFoundOr(s.repo.UpdateCategory(ctx, cmd.ID, strings.TrimSpace(cmd.Name), cmd.OrderNum),
		"news.category_not_found", "栏目不存在")
}

func (s *Service) DeleteCategory(ctx context.Context, id kernel.ID) error {
	return notFoundOr(s.repo.DeleteCategory(ctx, id), "news.category_not_found", "栏目不存在")
}

// ---- Articles ----

type CreateArticleCmd struct {
	CategoryID *kernel.ID
	Title      string
	Summary    string
	Content    string
	CoverURL   string
	Author     string
	CreatedBy  kernel.ID
}

func (s *Service) CreateArticle(ctx context.Context, cmd CreateArticleCmd) (*domain.Article, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, errors.New(errors.KindParam, "news.title_required", "文章标题不能为空")
	}
	if err := s.checkCategory(ctx, cmd.CategoryID); err != nil {
		return nil, err
	}
	now := s.clock.Now()
	a := &domain.Article{
		ID: kernel.NewID(), CategoryID: cmd.CategoryID, Title: title,
		Summary: cmd.Summary, Content: cmd.Content, CoverURL: cmd.CoverURL,
		Author: cmd.Author, Status: domain.StatusDraft, Views: 0,
		CreatedBy: cmd.CreatedBy, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreateArticle(ctx, a); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "news.create_article_failed", "创建文章失败", err)
	}
	_ = s.bus.Publish(ctx, "news.article_created", map[string]any{"article_id": a.ID})
	return a, nil
}

func (s *Service) GetArticle(ctx context.Context, id kernel.ID) (*domain.Article, error) {
	return s.repo.GetArticle(ctx, id)
}

func (s *Service) ListArticles(ctx context.Context, f domain.Filter, p kernel.Pagination) (*PagedArticles, error) {
	items, total, err := s.repo.ListArticles(ctx, f, p)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "news.list_articles_failed", "查询文章失败", err)
	}
	p = p.Normalize()
	return &PagedArticles{Items: items, Total: total, Page: p.Page, PageSize: p.PageSize}, nil
}

// ListPublished is the reader feed: published articles only, optionally by category.
func (s *Service) ListPublished(ctx context.Context, categoryID *kernel.ID, p kernel.Pagination) (*PagedArticles, error) {
	return s.ListArticles(ctx, domain.Filter{CategoryID: categoryID, PublishedOnly: true}, p)
}

// UpdateArticleCmd carries the editable fields. Pointer fields left nil are unchanged.
type UpdateArticleCmd struct {
	ID         kernel.ID
	Title      *string
	Summary    *string
	Content    *string
	CoverURL   *string
	Author     *string
	CategoryID *kernel.ID
	ClearCat   bool
}

func (s *Service) UpdateArticle(ctx context.Context, cmd UpdateArticleCmd) (*domain.Article, error) {
	a, err := s.repo.GetArticle(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, errors.New(errors.KindNotFound, "news.article_not_found", "文章不存在")
	}
	if cmd.Title != nil {
		nt := strings.TrimSpace(*cmd.Title)
		if nt == "" {
			return nil, errors.New(errors.KindParam, "news.title_required", "文章标题不能为空")
		}
		a.Title = nt
	}
	if cmd.Summary != nil {
		a.Summary = *cmd.Summary
	}
	if cmd.Content != nil {
		a.Content = *cmd.Content
	}
	if cmd.CoverURL != nil {
		a.CoverURL = *cmd.CoverURL
	}
	if cmd.Author != nil {
		a.Author = *cmd.Author
	}
	if cmd.ClearCat {
		a.CategoryID = nil
	} else if cmd.CategoryID != nil {
		if err := s.checkCategory(ctx, cmd.CategoryID); err != nil {
			return nil, err
		}
		a.CategoryID = cmd.CategoryID
	}
	if err := s.repo.UpdateArticle(ctx, a); err != nil {
		return nil, notFoundOr(err, "news.article_not_found", "文章不存在")
	}
	return s.repo.GetArticle(ctx, cmd.ID)
}

// SetPublished publishes/unpublishes via a targeted status update (no full-row
// overwrite), so it never clobbers a concurrent content edit.
func (s *Service) SetPublished(ctx context.Context, id kernel.ID, publish bool) (*domain.Article, error) {
	status := domain.StatusDraft
	var publishedAt interface{} // nil clears published_at on unpublish
	if publish {
		status = domain.StatusPublished
		publishedAt = s.clock.Now()
	}
	if err := s.repo.SetStatus(ctx, id, status, publishedAt); err != nil {
		return nil, notFoundOr(err, "news.article_not_found", "文章不存在")
	}
	evt := "news.article_unpublished"
	if publish {
		evt = "news.article_published"
	}
	_ = s.bus.Publish(ctx, evt, map[string]any{"article_id": id})
	return s.repo.GetArticle(ctx, id)
}

// IncrViews bumps the article's view counter (reader side).
func (s *Service) IncrViews(ctx context.Context, id kernel.ID) error {
	return notFoundOr(s.repo.IncrViews(ctx, id), "news.article_not_found", "文章不存在")
}

func (s *Service) DeleteArticle(ctx context.Context, id kernel.ID) error {
	return notFoundOr(s.repo.DeleteArticle(ctx, id), "news.article_not_found", "文章不存在")
}

// ---- helpers ----

func (s *Service) checkCategory(ctx context.Context, id *kernel.ID) error {
	if id == nil {
		return nil
	}
	c, err := s.repo.GetCategory(ctx, *id)
	if err != nil {
		return err
	}
	if c == nil {
		return errors.New(errors.KindParam, "news.category_not_found", "栏目不存在")
	}
	return nil
}

func notFoundOr(err error, code, msg string) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, pgx.ErrNoRows) {
		return errors.New(errors.KindNotFound, code, msg)
	}
	return errors.Wrap(errors.KindDatabase, "news.db_error", "操作失败", err)
}
