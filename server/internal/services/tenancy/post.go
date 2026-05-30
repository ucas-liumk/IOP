package tenancy

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// PostRow is a 岗位 (post / position) record in a tenant schema.
type PostRow struct {
	ID        kernel.ID `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	OrderNum  int       `json:"order_num"`
	Status    string    `json:"status"`
	CreatedAt string    `json:"created_at"`
}

// ListPosts returns all posts ordered by order_num, code.
func (s *Service) ListPosts(ctx context.Context, pool *pgxpool.Pool, t *Tenant) ([]PostRow, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx,
		`SELECT id, code, name, order_num, status,
		        to_char(created_at, 'YYYY-MM-DD HH24:MI:SS')
		 FROM post
		 ORDER BY order_num, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PostRow{}
	for rows.Next() {
		var p PostRow
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.OrderNum, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}

// CreatePostCmd is input for post creation.
type CreatePostCmd struct {
	Code     string
	Name     string
	OrderNum int
}

// CreatePost inserts a new post (code must be unique within the tenant).
func (s *Service) CreatePost(ctx context.Context, pool *pgxpool.Pool, t *Tenant, cmd CreatePostCmd) (*PostRow, error) {
	if cmd.Code == "" || cmd.Name == "" {
		return nil, errors.New(errors.KindParam, "tenancy.post_invalid", "岗位编码/名称不能为空")
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return nil, err
	}
	var dup int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM post WHERE code = $1`, cmd.Code).Scan(&dup); err != nil {
		return nil, err
	}
	if dup > 0 {
		return nil, errors.New(errors.KindConflict, "tenancy.post_code_exists", "岗位编码已存在")
	}
	id := kernel.NewID()
	var createdAt string
	if err := tx.QueryRow(ctx,
		`INSERT INTO post (id, code, name, order_num, status)
		 VALUES ($1, $2, $3, $4, 'active')
		 RETURNING to_char(created_at, 'YYYY-MM-DD HH24:MI:SS')`,
		id, cmd.Code, cmd.Name, cmd.OrderNum).Scan(&createdAt); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "tenancy.create_post_failed", "创建岗位失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &PostRow{
		ID: id, Code: cmd.Code, Name: cmd.Name, OrderNum: cmd.OrderNum,
		Status: "active", CreatedAt: createdAt,
	}, nil
}

// UpdatePostCmd holds optional fields for patching a post.
type UpdatePostCmd struct {
	PostID   kernel.ID
	Name     *string
	OrderNum *int
	Status   *string
}

// UpdatePost updates the given fields on a post.
func (s *Service) UpdatePost(ctx context.Context, pool *pgxpool.Pool, t *Tenant, cmd UpdatePostCmd) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return err
	}
	sets := []string{}
	args := []any{cmd.PostID}
	idx := 2
	addField := func(col string, val any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, idx))
		args = append(args, val)
		idx++
	}
	if cmd.Name != nil {
		addField("name", *cmd.Name)
	}
	if cmd.OrderNum != nil {
		addField("order_num", *cmd.OrderNum)
	}
	if cmd.Status != nil {
		addField("status", *cmd.Status)
	}
	if len(sets) == 0 {
		return nil
	}
	sql := "UPDATE post SET "
	for i, ss := range sets {
		if i > 0 {
			sql += ", "
		}
		sql += ss
	}
	sql += " WHERE id = $1"
	res, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.update_post_failed", "更新岗位失败", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "tenancy.post_not_found", "岗位不存在")
	}
	return tx.Commit(ctx)
}

// DeletePost removes a post, rejecting if it is assigned to any member.
func (s *Service) DeletePost(ctx context.Context, pool *pgxpool.Pool, t *Tenant, id kernel.ID) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return err
	}
	var assigned int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM member_post WHERE post_id = $1`, id).Scan(&assigned); err != nil {
		return err
	}
	if assigned > 0 {
		return errors.New(errors.KindParam, "tenancy.post_in_use", "岗位已分配给成员,无法删除")
	}
	res, err := tx.Exec(ctx, `DELETE FROM post WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "tenancy.post_not_found", "岗位不存在")
	}
	return tx.Commit(ctx)
}

// registerPostRoutes mounts /admin/posts/* (tenant_admin gated by the caller).
func registerPostRoutes(r *gin.RouterGroup, svc *Service, pool *pgxpool.Pool) {
	r.GET("/admin/posts", func(c *gin.Context) {
		posts, err := svc.ListPosts(c.Request.Context(), pool, tenantFromCtx(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"posts": posts})
	})

	r.POST("/admin/posts", func(c *gin.Context) {
		var req struct {
			Code     string `json:"code" binding:"required"`
			Name     string `json:"name" binding:"required"`
			OrderNum int    `json:"order_num"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		post, err := svc.CreatePost(c.Request.Context(), pool, tenantFromCtx(c), CreatePostCmd{
			Code: req.Code, Name: req.Name, OrderNum: req.OrderNum,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, post)
	})

	r.PATCH("/admin/posts/:id", func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "岗位 ID 无效", err))
			return
		}
		var req struct {
			Name     *string `json:"name"`
			OrderNum *int    `json:"order_num"`
			Status   *string `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.UpdatePost(c.Request.Context(), pool, tenantFromCtx(c), UpdatePostCmd{
			PostID: id, Name: req.Name, OrderNum: req.OrderNum, Status: req.Status,
		}); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/admin/posts/:id", func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "岗位 ID 无效", err))
			return
		}
		if err := svc.DeletePost(c.Request.Context(), pool, tenantFromCtx(c), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}
