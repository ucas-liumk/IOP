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

// NoticeRow is a 通知公告 (notice / announcement) record in a tenant schema.
type NoticeRow struct {
	ID        kernel.ID  `json:"id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Type      string     `json:"type"`
	Status    string     `json:"status"`
	CreatedBy *kernel.ID `json:"created_by,omitempty"`
	CreatedAt string     `json:"created_at"`
}

// ListNotices returns notices (newest first), paginated, with an optional status filter.
func (s *Service) ListNotices(ctx context.Context, pool *pgxpool.Pool, t *Tenant, p kernel.Pagination, status string) ([]NoticeRow, error) {
	p = p.Normalize()
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
	args := []any{p.PageSize, p.Offset()}
	where := " WHERE deleted_at IS NULL"
	if status != "" {
		where += " AND status = $3"
		args = append(args, status)
	}
	rows, err := tx.Query(ctx,
		`SELECT id, title, COALESCE(content,''), type, status, created_by,
		        to_char(created_at, 'YYYY-MM-DD HH24:MI:SS')
		 FROM notice`+where+`
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []NoticeRow{}
	for rows.Next() {
		var n NoticeRow
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.Type, &n.Status, &n.CreatedBy, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}

// CreateNoticeCmd is input for notice creation.
type CreateNoticeCmd struct {
	Title     string
	Content   string
	Type      string
	CreatedBy *kernel.ID
}

// CreateNotice inserts a new notice (starts in 'draft' status).
func (s *Service) CreateNotice(ctx context.Context, pool *pgxpool.Pool, t *Tenant, cmd CreateNoticeCmd) (*NoticeRow, error) {
	if cmd.Title == "" {
		return nil, errors.New(errors.KindParam, "tenancy.notice_title_required", "公告标题不能为空")
	}
	typ := cmd.Type
	if typ == "" {
		typ = "notice"
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
	id := kernel.NewID()
	var createdAt string
	if err := tx.QueryRow(ctx,
		`INSERT INTO notice (id, title, content, type, status, created_by)
		 VALUES ($1, $2, $3, $4, 'draft', $5)
		 RETURNING to_char(created_at, 'YYYY-MM-DD HH24:MI:SS')`,
		id, cmd.Title, cmd.Content, typ, idPtrOrNil(cmd.CreatedBy)).Scan(&createdAt); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "tenancy.create_notice_failed", "创建公告失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &NoticeRow{
		ID: id, Title: cmd.Title, Content: cmd.Content, Type: typ,
		Status: "draft", CreatedBy: cmd.CreatedBy, CreatedAt: createdAt,
	}, nil
}

// UpdateNoticeCmd holds optional fields for patching a notice.
type UpdateNoticeCmd struct {
	NoticeID kernel.ID
	Title    *string
	Content  *string
	Type     *string
}

// UpdateNotice updates the given fields on a notice.
func (s *Service) UpdateNotice(ctx context.Context, pool *pgxpool.Pool, t *Tenant, cmd UpdateNoticeCmd) error {
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
	args := []any{cmd.NoticeID}
	idx := 2
	addField := func(col string, val any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, idx))
		args = append(args, val)
		idx++
	}
	if cmd.Title != nil {
		addField("title", *cmd.Title)
	}
	if cmd.Content != nil {
		addField("content", *cmd.Content)
	}
	if cmd.Type != nil {
		addField("type", *cmd.Type)
	}
	if len(sets) == 0 {
		return nil
	}
	sql := "UPDATE notice SET "
	for i, ss := range sets {
		if i > 0 {
			sql += ", "
		}
		sql += ss
	}
	sql += " WHERE id = $1 AND deleted_at IS NULL"
	res, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.update_notice_failed", "更新公告失败", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "tenancy.notice_not_found", "公告不存在")
	}
	return tx.Commit(ctx)
}

// DeleteNotice removes a notice.
func (s *Service) DeleteNotice(ctx context.Context, pool *pgxpool.Pool, t *Tenant, id kernel.ID) error {
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
	res, err := tx.Exec(ctx,
		`UPDATE notice
		 SET deleted_at = now(), status = 'draft'
		 WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "tenancy.notice_not_found", "公告不存在")
	}
	return tx.Commit(ctx)
}

// setNoticeStatus is the shared helper behind PublishNotice / WithdrawNotice.
func (s *Service) setNoticeStatus(ctx context.Context, pool *pgxpool.Pool, t *Tenant, id kernel.ID, status string) error {
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
	res, err := tx.Exec(ctx,
		`UPDATE notice
		 SET status = $1
		 WHERE id = $2 AND deleted_at IS NULL`, status, id)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.notice_status_failed", "更新公告状态失败", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "tenancy.notice_not_found", "公告不存在")
	}
	return tx.Commit(ctx)
}

// PublishNotice sets a notice to 'published'.
func (s *Service) PublishNotice(ctx context.Context, pool *pgxpool.Pool, t *Tenant, id kernel.ID) error {
	return s.setNoticeStatus(ctx, pool, t, id, "published")
}

// WithdrawNotice sets a notice back to 'draft'.
func (s *Service) WithdrawNotice(ctx context.Context, pool *pgxpool.Pool, t *Tenant, id kernel.ID) error {
	return s.setNoticeStatus(ctx, pool, t, id, "draft")
}

// currentMemberID extracts the acting member id from the request context
// (set by the IAM JWTAuth middleware), if present.
func currentMemberID(c *gin.Context) *kernel.ID {
	id, ok := kernel.MemberIDFromContext(c.Request.Context())
	if !ok || id == "" {
		return nil
	}
	return &id
}

// registerNoticeRoutes mounts /admin/notices/* (tenant_admin gated by the caller).
func registerNoticeRoutes(r *gin.RouterGroup, svc *Service, pool *pgxpool.Pool, authz AuthzFunc) {
	r.GET("/admin/notices", authz("notice", "read"), func(c *gin.Context) {
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		notices, err := svc.ListNotices(c.Request.Context(), pool, tenantFromCtx(c), p, c.Query("status"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"notices": notices})
	})

	r.POST("/admin/notices", authz("notice", "manage"), func(c *gin.Context) {
		var req struct {
			Title   string `json:"title" binding:"required"`
			Content string `json:"content"`
			Type    string `json:"type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		notice, err := svc.CreateNotice(c.Request.Context(), pool, tenantFromCtx(c), CreateNoticeCmd{
			Title: req.Title, Content: req.Content, Type: req.Type, CreatedBy: currentMemberID(c),
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, notice)
	})

	r.PATCH("/admin/notices/:id", authz("notice", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "公告 ID 无效", err))
			return
		}
		var req struct {
			Title   *string `json:"title"`
			Content *string `json:"content"`
			Type    *string `json:"type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.UpdateNotice(c.Request.Context(), pool, tenantFromCtx(c), UpdateNoticeCmd{
			NoticeID: id, Title: req.Title, Content: req.Content, Type: req.Type,
		}); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/admin/notices/:id", authz("notice", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "公告 ID 无效", err))
			return
		}
		if err := svc.DeleteNotice(c.Request.Context(), pool, tenantFromCtx(c), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/admin/notices/:id/publish", authz("notice", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "公告 ID 无效", err))
			return
		}
		if err := svc.PublishNotice(c.Request.Context(), pool, tenantFromCtx(c), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/admin/notices/:id/withdraw", authz("notice", "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "公告 ID 无效", err))
			return
		}
		if err := svc.WithdrawNotice(c.Request.Context(), pool, tenantFromCtx(c), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}
