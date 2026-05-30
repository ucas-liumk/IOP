// Package application is the use-case layer for the books (图书馆) module.
package application

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/books/domain"
	"github.com/leo/iop/server/internal/contexts/books/infrastructure"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// defaultLoanDays is the borrow period when the caller doesn't specify one.
const defaultLoanDays = 30

type Service struct {
	repo  *infrastructure.Repo
	bus   eventbus.Bus
	clock kernel.Clock
}

func NewService(repo *infrastructure.Repo, bus eventbus.Bus, clk kernel.Clock) *Service {
	return &Service{repo: repo, bus: bus, clock: clk}
}

// ---- Books ----

type CreateBookCmd struct {
	ISBN      string
	Title     string
	Author    string
	Publisher string
	Category  string
	Total     int
	CoverURL  string
	Location  string
}

func (s *Service) CreateBook(ctx context.Context, cmd CreateBookCmd) (*domain.Book, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, errors.New(errors.KindParam, "books.title_required", "书名不能为空")
	}
	if cmd.Total < 0 {
		cmd.Total = 0
	}
	now := s.clock.Now()
	b := &domain.Book{
		ID:        kernel.NewID(),
		ISBN:      strings.TrimSpace(cmd.ISBN),
		Title:     title,
		Author:    strings.TrimSpace(cmd.Author),
		Publisher: strings.TrimSpace(cmd.Publisher),
		Category:  strings.TrimSpace(cmd.Category),
		Total:     cmd.Total,
		Available: cmd.Total,
		CoverURL:  strings.TrimSpace(cmd.CoverURL),
		Location:  strings.TrimSpace(cmd.Location),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateBook(ctx, b); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "books.create_failed", "新增图书失败", err)
	}
	_ = s.bus.Publish(ctx, "books.book_created", map[string]any{"book_id": b.ID})
	return b, nil
}

type UpdateBookCmd struct {
	ID        kernel.ID
	ISBN      string
	Title     string
	Author    string
	Publisher string
	Category  string
	Total     int
	CoverURL  string
	Location  string
}

func (s *Service) UpdateBook(ctx context.Context, cmd UpdateBookCmd) (*domain.Book, error) {
	b, err := s.repo.GetBook(ctx, cmd.ID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "books.db_error", "操作失败", err)
	}
	if b == nil {
		return nil, errors.New(errors.KindNotFound, "books.not_found", "图书不存在")
	}
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, errors.New(errors.KindParam, "books.title_required", "书名不能为空")
	}
	if cmd.Total < 0 {
		return nil, errors.New(errors.KindParam, "books.total_invalid", "馆藏总量不能为负")
	}
	// Number of copies currently out on loan must remain consistent: total can't
	// drop below what's already borrowed.
	onLoan := b.Total - b.Available
	if cmd.Total < onLoan {
		return nil, errors.New(errors.KindBusiness, "books.total_below_loaned", "馆藏总量不能小于已借出数量")
	}
	b.ISBN = strings.TrimSpace(cmd.ISBN)
	b.Title = title
	b.Author = strings.TrimSpace(cmd.Author)
	b.Publisher = strings.TrimSpace(cmd.Publisher)
	b.Category = strings.TrimSpace(cmd.Category)
	b.CoverURL = strings.TrimSpace(cmd.CoverURL)
	b.Location = strings.TrimSpace(cmd.Location)
	b.Total = cmd.Total
	b.Available = cmd.Total - onLoan
	if err := s.repo.UpdateBook(ctx, b); err != nil {
		return nil, notFoundOr(err, "books.not_found", "图书不存在")
	}
	return b, nil
}

func (s *Service) DeleteBook(ctx context.Context, id kernel.ID) error {
	return notFoundOr(s.repo.DeleteBook(ctx, id), "books.not_found", "图书不存在")
}

func (s *Service) GetBook(ctx context.Context, id kernel.ID) (*domain.Book, error) {
	b, err := s.repo.GetBook(ctx, id)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "books.db_error", "操作失败", err)
	}
	if b == nil {
		return nil, errors.New(errors.KindNotFound, "books.not_found", "图书不存在")
	}
	return b, nil
}

func (s *Service) ListBooks(ctx context.Context, f domain.BookFilter) ([]*domain.Book, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}
	books, total, err := s.repo.ListBooks(ctx, f)
	if err != nil {
		return nil, 0, errors.Wrap(errors.KindDatabase, "books.list_failed", "查询图书失败", err)
	}
	return books, total, nil
}

// ---- Borrows ----

// Borrow records a loan for the given member, decrementing availability. Returns
// a 400 if no copy is available.
func (s *Service) Borrow(ctx context.Context, bookID, memberID kernel.ID) (*domain.Borrow, error) {
	if memberID == "" {
		return nil, errors.New(errors.KindAuth, "books.no_member", "无法确定借阅人")
	}
	b, err := s.repo.GetBook(ctx, bookID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "books.db_error", "操作失败", err)
	}
	if b == nil {
		return nil, errors.New(errors.KindNotFound, "books.not_found", "图书不存在")
	}
	now := s.clock.Now()
	rec := &domain.Borrow{
		ID:         kernel.NewID(),
		BookID:     bookID,
		MemberID:   memberID,
		BorrowedAt: now,
		DueAt:      now.Add(defaultLoanDays * 24 * time.Hour),
		Status:     domain.StatusBorrowed,
	}
	if err := s.repo.Borrow(ctx, rec); err != nil {
		if infrastructure.IsNoStock(err) {
			return nil, errors.New(errors.KindBusiness, "books.no_stock", "该图书暂无可借副本")
		}
		return nil, errors.Wrap(errors.KindDatabase, "books.borrow_failed", "借阅失败", err)
	}
	_ = s.bus.Publish(ctx, "books.borrowed", map[string]any{"book_id": bookID, "member_id": memberID, "borrow_id": rec.ID})
	return rec, nil
}

// Return marks a borrow returned and restocks the copy.
func (s *Service) Return(ctx context.Context, borrowID kernel.ID) error {
	if err := s.repo.Return(ctx, borrowID, s.clock.Now()); err != nil {
		return notFoundOr(err, "books.borrow_not_found", "借阅记录不存在或已归还")
	}
	_ = s.bus.Publish(ctx, "books.returned", map[string]any{"borrow_id": borrowID})
	return nil
}

func (s *Service) ListBorrows(ctx context.Context, f domain.BorrowFilter) ([]*domain.Borrow, error) {
	out, err := s.repo.ListBorrows(ctx, f)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "books.borrows_failed", "查询借阅记录失败", err)
	}
	return out, nil
}

// ---- helpers ----

func notFoundOr(err error, code, msg string) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, pgx.ErrNoRows) {
		return errors.New(errors.KindNotFound, code, msg)
	}
	return errors.Wrap(errors.KindDatabase, "books.db_error", "操作失败", err)
}
