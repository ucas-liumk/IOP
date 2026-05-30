// Package infrastructure is the PG persistence for the news module. All queries
// run through tenantdb.TenantDB, which sets search_path to the caller's tenant
// schema — so the same SQL is automatically isolated per tenant.
package infrastructure

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/news/domain"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

type Repo struct{ db *tenantdb.TenantDB }

func NewRepo(db *tenantdb.TenantDB) *Repo { return &Repo{db: db} }

// ---- Categories ----

func (r *Repo) CreateCategory(ctx context.Context, c *domain.Category) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO news_category (id, name, order_num) VALUES ($1,$2,$3)`,
			c.ID, c.Name, c.OrderNum)
		return err
	})
}

func (r *Repo) UpdateCategory(ctx context.Context, id kernel.ID, name string, orderNum int) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE news_category SET name=$1, order_num=$2 WHERE id=$3`,
			name, orderNum, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) DeleteCategory(ctx context.Context, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, `DELETE FROM news_category WHERE id=$1`, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) ListCategories(ctx context.Context) ([]*domain.Category, error) {
	var out []*domain.Category
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT c.id, c.name, c.order_num, COALESCE(a.cnt, 0)
			 FROM news_category c
			 LEFT JOIN (
			   SELECT category_id, count(*) AS cnt FROM news_article GROUP BY category_id
			 ) a ON a.category_id = c.id
			 ORDER BY c.order_num, c.name`)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.Category{}
		for rows.Next() {
			c := &domain.Category{}
			if err := rows.Scan(&c.ID, &c.Name, &c.OrderNum, &c.ArticleCount); err != nil {
				return err
			}
			out = append(out, c)
		}
		return rows.Err()
	})
	return out, err
}

func (r *Repo) GetCategory(ctx context.Context, id kernel.ID) (*domain.Category, error) {
	var c *domain.Category
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `SELECT id, name, order_num FROM news_category WHERE id=$1`, id)
		var x domain.Category
		err := row.Scan(&x.ID, &x.Name, &x.OrderNum)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		c = &x
		return nil
	})
	return c, err
}

// ---- Articles ----

const articleCols = `a.id, a.category_id, a.title, a.summary, a.content, a.cover_url,
	a.author, a.status, a.published_at, a.views, a.created_by, a.created_at, a.updated_at,
	COALESCE(c.name, '')`

func scanArticle(row pgx.Row) (*domain.Article, error) {
	a := &domain.Article{}
	err := row.Scan(&a.ID, &a.CategoryID, &a.Title, &a.Summary, &a.Content, &a.CoverURL,
		&a.Author, &a.Status, &a.PublishedAt, &a.Views, &a.CreatedBy, &a.CreatedAt,
		&a.UpdatedAt, &a.CategoryName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *Repo) CreateArticle(ctx context.Context, a *domain.Article) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO news_article
			 (id, category_id, title, summary, content, cover_url, author, status,
			  published_at, views, created_by, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			a.ID, a.CategoryID, a.Title, a.Summary, a.Content, a.CoverURL, a.Author,
			a.Status, a.PublishedAt, a.Views, a.CreatedBy, a.CreatedAt, a.UpdatedAt)
		return err
	})
}

// UpdateArticle persists the EDITABLE content fields only. It deliberately does
// NOT touch status/published_at/views — those are owned by SetStatus / IncrViews —
// so a concurrent edit and a concurrent publish/view-bump operate on disjoint
// columns and cannot clobber each other.
func (r *Repo) UpdateArticle(ctx context.Context, a *domain.Article) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE news_article SET category_id=$1, title=$2, summary=$3, content=$4,
			        cover_url=$5, author=$6, updated_at=now()
			 WHERE id=$7`,
			a.CategoryID, a.Title, a.Summary, a.Content, a.CoverURL, a.Author, a.ID)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// SetStatus does a targeted status/published_at update in a single statement,
// independent of the editable-field overwrite in UpdateArticle.
func (r *Repo) SetStatus(ctx context.Context, id kernel.ID, status string, publishedAt interface{}) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE news_article SET status=$1, published_at=$2, updated_at=now() WHERE id=$3`,
			status, publishedAt, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) DeleteArticle(ctx context.Context, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, `DELETE FROM news_article WHERE id=$1`, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) GetArticle(ctx context.Context, id kernel.ID) (*domain.Article, error) {
	var a *domain.Article
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`SELECT `+articleCols+` FROM news_article a
			 LEFT JOIN news_category c ON c.id = a.category_id
			 WHERE a.id=$1`, id)
		got, err := scanArticle(row)
		if err != nil {
			return err
		}
		a = got
		return nil
	})
	return a, err
}

// IncrViews bumps the view counter atomically (single UPDATE, no read-modify-write).
func (r *Repo) IncrViews(ctx context.Context, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE news_article SET views = views + 1 WHERE id=$1`, id)
		return err
	})
}

// ListArticles runs the management / reader query with pagination. Returns the
// page rows plus the total matching count.
func (r *Repo) ListArticles(ctx context.Context, f domain.Filter, p kernel.Pagination) ([]*domain.Article, int, error) {
	p = p.Normalize()
	where := []string{"1=1"}
	args := []any{}
	add := func(clause string, val any) {
		args = append(args, val)
		where = append(where, clause+" $"+strconv.Itoa(len(args)))
	}
	if f.PublishedOnly {
		where = append(where, "a.status = 'published'")
	} else if f.Status != "" {
		add("a.status =", f.Status)
	}
	if f.CategoryID != nil {
		add("a.category_id =", *f.CategoryID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		args = append(args, "%"+kw+"%")
		where = append(where, "a.title ILIKE $"+strconv.Itoa(len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	// Published feed sorts by published_at desc; management sorts by created_at desc.
	order := "a.created_at DESC"
	if f.PublishedOnly {
		order = "a.published_at DESC NULLS LAST, a.created_at DESC"
	}

	var out []*domain.Article
	var total int
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM news_article a WHERE `+whereSQL, args...).Scan(&total); err != nil {
			return err
		}
		pageArgs := append(append([]any{}, args...), p.PageSize, p.Offset())
		q := `SELECT ` + articleCols + ` FROM news_article a
		      LEFT JOIN news_category c ON c.id = a.category_id
		      WHERE ` + whereSQL +
			` ORDER BY ` + order +
			` LIMIT $` + strconv.Itoa(len(pageArgs)-1) + ` OFFSET $` + strconv.Itoa(len(pageArgs))
		rows, err := tx.Query(ctx, q, pageArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.Article{}
		for rows.Next() {
			a, err := scanArticle(rows)
			if err != nil {
				return err
			}
			out = append(out, a)
		}
		return rows.Err()
	})
	return out, total, err
}
