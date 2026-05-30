// Package infrastructure is the PG persistence for the books module. All queries
// run through tenantdb.TenantDB, which sets search_path to the caller's tenant
// schema — so the same SQL is automatically isolated per tenant.
package infrastructure

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/books/domain"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

type Repo struct{ db *tenantdb.TenantDB }

func NewRepo(db *tenantdb.TenantDB) *Repo { return &Repo{db: db} }

const bookCols = `id, isbn, title, author, publisher, category, total, available, cover_url, location, created_at, updated_at`

func scanBook(row pgx.Row) (*domain.Book, error) {
	b := &domain.Book{}
	err := row.Scan(&b.ID, &b.ISBN, &b.Title, &b.Author, &b.Publisher, &b.Category,
		&b.Total, &b.Available, &b.CoverURL, &b.Location, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

// ---- Books ----

func (r *Repo) CreateBook(ctx context.Context, b *domain.Book) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO book (`+bookCols+`)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			b.ID, b.ISBN, b.Title, b.Author, b.Publisher, b.Category,
			b.Total, b.Available, b.CoverURL, b.Location, b.CreatedAt, b.UpdatedAt)
		return err
	})
}

// UpdateBook persists the editable metadata plus total/available copy counts.
func (r *Repo) UpdateBook(ctx context.Context, b *domain.Book) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE book SET isbn=$1, title=$2, author=$3, publisher=$4, category=$5,
			        total=$6, available=$7, cover_url=$8, location=$9, updated_at=now()
			 WHERE id=$10`,
			b.ISBN, b.Title, b.Author, b.Publisher, b.Category,
			b.Total, b.Available, b.CoverURL, b.Location, b.ID)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) DeleteBook(ctx context.Context, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, `DELETE FROM book WHERE id=$1`, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) GetBook(ctx context.Context, id kernel.ID) (*domain.Book, error) {
	var b *domain.Book
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `SELECT `+bookCols+` FROM book WHERE id=$1`, id)
		got, err := scanBook(row)
		if err != nil {
			return err
		}
		b = got
		return nil
	})
	return b, err
}

// ListBooks runs the paged + search catalog query and returns the total count.
func (r *Repo) ListBooks(ctx context.Context, f domain.BookFilter) ([]*domain.Book, int, error) {
	where := []string{"1=1"}
	args := []any{}
	if s := strings.TrimSpace(f.Search); s != "" {
		args = append(args, "%"+s+"%")
		i := strconv.Itoa(len(args))
		where = append(where, "(title ILIKE $"+i+" OR author ILIKE $"+i+" OR isbn ILIKE $"+i+" OR publisher ILIKE $"+i+")")
	}
	if c := strings.TrimSpace(f.Category); c != "" {
		args = append(args, c)
		where = append(where, "category = $"+strconv.Itoa(len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var out []*domain.Book
	var total int
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM book WHERE `+whereSQL, args...).Scan(&total); err != nil {
			return err
		}
		pargs := append(append([]any{}, args...), f.PageSize, (f.Page-1)*f.PageSize)
		q := `SELECT ` + bookCols + ` FROM book WHERE ` + whereSQL +
			` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(pargs)-1) + ` OFFSET $` + strconv.Itoa(len(pargs))
		rows, err := tx.Query(ctx, q, pargs...)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.Book{}
		for rows.Next() {
			b, err := scanBook(rows)
			if err != nil {
				return err
			}
			out = append(out, b)
		}
		return rows.Err()
	})
	return out, total, err
}

// ---- Borrows ----

// Borrow atomically decrements availability and inserts a borrow record. If no
// copy is available the UPDATE affects 0 rows and the whole tx aborts — this is
// the over-borrow guard (concurrent borrows can't drive available below 0).
func (r *Repo) Borrow(ctx context.Context, b *domain.Borrow) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE book SET available = available - 1, updated_at = now()
			 WHERE id = $1 AND available > 0`, b.BookID)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return errNoStock
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO book_borrow (id, book_id, member_id, borrowed_at, due_at, returned_at, status)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			b.ID, b.BookID, b.MemberID, b.BorrowedAt, b.DueAt, b.ReturnedAt, b.Status)
		return err
	})
}

// errNoStock is sentinel for "no available copy". The service maps it to a 400.
var errNoStock = errors.New("books: no available copy")

// IsNoStock reports whether err is the no-available-copy sentinel.
func IsNoStock(err error) bool { return errors.Is(err, errNoStock) }

func (r *Repo) GetBorrow(ctx context.Context, id kernel.ID) (*domain.Borrow, error) {
	var b *domain.Borrow
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`SELECT id, book_id, member_id, borrowed_at, due_at, returned_at, status
			 FROM book_borrow WHERE id=$1`, id)
		var x domain.Borrow
		err := row.Scan(&x.ID, &x.BookID, &x.MemberID, &x.BorrowedAt, &x.DueAt, &x.ReturnedAt, &x.Status)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		b = &x
		return nil
	})
	return b, err
}

// Return atomically marks a borrow returned and increments availability. Only an
// active (not-yet-returned) record matches, so a double-return is a no-op (0 rows).
func (r *Repo) Return(ctx context.Context, id kernel.ID, returnedAt time.Time) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		var bookID kernel.ID
		err := tx.QueryRow(ctx,
			`UPDATE book_borrow SET returned_at=$1, status=$2, updated_at=now()
			 WHERE id=$3 AND returned_at IS NULL
			 RETURNING book_id`,
			returnedAt, domain.StatusReturned, id).Scan(&bookID)
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		if err != nil {
			return err
		}
		// Increment availability but never exceed total (defense-in-depth).
		_, err = tx.Exec(ctx,
			`UPDATE book SET available = LEAST(available + 1, total), updated_at = now() WHERE id = $1`, bookID)
		return err
	})
}

// ListBorrows returns borrow records (optionally scoped to a member) joined with
// book metadata, newest first.
func (r *Repo) ListBorrows(ctx context.Context, f domain.BorrowFilter) ([]*domain.Borrow, error) {
	where := []string{"1=1"}
	args := []any{}
	if f.MemberID != "" {
		args = append(args, f.MemberID)
		where = append(where, "bb.member_id = $"+strconv.Itoa(len(args)))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where = append(where, "bb.status = $"+strconv.Itoa(len(args)))
	}
	q := `SELECT bb.id, bb.book_id, bb.member_id, bb.borrowed_at, bb.due_at, bb.returned_at, bb.status,
	             b.title, b.author, b.isbn
	      FROM book_borrow bb
	      JOIN book b ON b.id = bb.book_id
	      WHERE ` + strings.Join(where, " AND ") + `
	      ORDER BY bb.borrowed_at DESC LIMIT 500`

	var out []*domain.Borrow
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.Borrow{}
		for rows.Next() {
			b := &domain.Borrow{}
			if err := rows.Scan(&b.ID, &b.BookID, &b.MemberID, &b.BorrowedAt, &b.DueAt,
				&b.ReturnedAt, &b.Status, &b.BookTitle, &b.BookAuthor, &b.BookISBN); err != nil {
				return err
			}
			out = append(out, b)
		}
		return rows.Err()
	})
	return out, err
}
