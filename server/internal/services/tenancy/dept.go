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

// DeptRow is a flat department record.
type DeptRow struct {
	ID        kernel.ID  `json:"id"`
	Name      string     `json:"name"`
	ParentID  *kernel.ID `json:"parent_id,omitempty"`
	OrderNum  int        `json:"order_num"`
	Leader    string     `json:"leader,omitempty"`
	Phone     string     `json:"phone,omitempty"`
	Email     string     `json:"email,omitempty"`
	Status    string     `json:"status"`
	CreatedAt string     `json:"created_at"`
}

// DeptTreeNode embeds DeptRow with nested children.
type DeptTreeNode struct {
	DeptRow
	Children []*DeptTreeNode `json:"children,omitempty"`
}

// ListDepts returns a flat list of departments ordered by order_num, name.
func (s *Service) ListDepts(ctx context.Context, pool *pgxpool.Pool, t *Tenant) ([]DeptRow, error) {
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
		`SELECT id, name, parent_id, order_num,
		        COALESCE(leader,''), COALESCE(phone,''), COALESCE(email,''),
		        status, to_char(created_at, 'YYYY-MM-DD HH24:MI:SS')
		 FROM department
		 ORDER BY order_num, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DeptRow{}
	for rows.Next() {
		var d DeptRow
		if err := rows.Scan(&d.ID, &d.Name, &d.ParentID, &d.OrderNum,
			&d.Leader, &d.Phone, &d.Email, &d.Status, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}

// DeptTree builds and returns the department tree.
func (s *Service) DeptTree(ctx context.Context, pool *pgxpool.Pool, t *Tenant) ([]*DeptTreeNode, error) {
	flat, err := s.ListDepts(ctx, pool, t)
	if err != nil {
		return nil, err
	}
	return buildDeptTree(flat), nil
}

// buildDeptTree converts a flat list into a nested tree.
func buildDeptTree(flat []DeptRow) []*DeptTreeNode {
	byID := make(map[kernel.ID]*DeptTreeNode, len(flat))
	for i := range flat {
		n := &DeptTreeNode{DeptRow: flat[i]}
		byID[flat[i].ID] = n
	}
	roots := []*DeptTreeNode{}
	for _, d := range flat {
		node := byID[d.ID]
		if d.ParentID == nil {
			roots = append(roots, node)
		} else if parent, ok := byID[*d.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// orphaned node (parent missing/deleted) — surface as root
			roots = append(roots, node)
		}
	}
	return roots
}

// CreateDeptCmd is input for department creation.
type CreateDeptCmd struct {
	Name     string
	ParentID *kernel.ID
	OrderNum int
	Leader   string
	Phone    string
	Email    string
}

// CreateDept inserts a new department.
func (s *Service) CreateDept(ctx context.Context, pool *pgxpool.Pool, t *Tenant, cmd CreateDeptCmd) (*DeptRow, error) {
	if cmd.Name == "" {
		return nil, errors.New(errors.KindParam, "tenancy.dept_name_required", "部门名称不能为空")
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
	// Validate parent exists if provided.
	if cmd.ParentID != nil {
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM department WHERE id = $1`, *cmd.ParentID).Scan(&count); err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, errors.New(errors.KindParam, "tenancy.dept_parent_not_found", "父部门不存在")
		}
	}
	id := kernel.NewID()
	var createdAt string
	if err := tx.QueryRow(ctx,
		`INSERT INTO department (id, name, parent_id, order_num, leader, phone, email, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'active')
		 RETURNING to_char(created_at, 'YYYY-MM-DD HH24:MI:SS')`,
		id, cmd.Name, idPtrOrNil(cmd.ParentID), cmd.OrderNum,
		nullStr(cmd.Leader), nullStr(cmd.Phone), nullStr(cmd.Email),
	).Scan(&createdAt); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "tenancy.create_dept_failed", "创建部门失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &DeptRow{
		ID: id, Name: cmd.Name, ParentID: cmd.ParentID, OrderNum: cmd.OrderNum,
		Leader: cmd.Leader, Phone: cmd.Phone, Email: cmd.Email,
		Status: "active", CreatedAt: createdAt,
	}, nil
}

// UpdateDeptCmd holds optional fields for patching a department.
type UpdateDeptCmd struct {
	DeptID   kernel.ID
	Name     *string
	ParentID *kernel.ID // nil = don't change; use UpdateDeptClearParent to clear
	OrderNum *int
	Leader   *string
	Phone    *string
	Email    *string
	Status   *string
}

// UpdateDept updates the given fields on a department.
func (s *Service) UpdateDept(ctx context.Context, pool *pgxpool.Pool, t *Tenant, cmd UpdateDeptCmd) error {
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
	args := []any{cmd.DeptID}
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
	if cmd.Leader != nil {
		addField("leader", nullStr(*cmd.Leader))
	}
	if cmd.Phone != nil {
		addField("phone", nullStr(*cmd.Phone))
	}
	if cmd.Email != nil {
		addField("email", nullStr(*cmd.Email))
	}
	if cmd.Status != nil {
		addField("status", *cmd.Status)
	}
	if len(sets) == 0 {
		return nil
	}
	sql := "UPDATE department SET "
	for i, ss := range sets {
		if i > 0 {
			sql += ", "
		}
		sql += ss
	}
	sql += " WHERE id = $1"
	res, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.update_dept_failed", "更新部门失败", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "tenancy.dept_not_found", "部门不存在")
	}
	return tx.Commit(ctx)
}

// DeleteDept removes a department, rejecting if it has children or assigned members.
func (s *Service) DeleteDept(ctx context.Context, pool *pgxpool.Pool, t *Tenant, id kernel.ID) error {
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
	var children int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM department WHERE parent_id = $1`, id).Scan(&children); err != nil {
		return err
	}
	if children > 0 {
		return errors.New(errors.KindParam, "tenancy.dept_has_children", "请先删除子部门")
	}
	var members int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM member WHERE dept_id = $1`, id).Scan(&members); err != nil {
		return err
	}
	if members > 0 {
		return errors.New(errors.KindParam, "tenancy.dept_has_members", "请先移除部门成员")
	}
	res, err := tx.Exec(ctx, `DELETE FROM department WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "tenancy.dept_not_found", "部门不存在")
	}
	return tx.Commit(ctx)
}

// MoveDept reparents a department, rejecting cycles.
func (s *Service) MoveDept(ctx context.Context, pool *pgxpool.Pool, t *Tenant, id kernel.ID, newParentID *kernel.ID) error {
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
	// Load all depts to check for cycles.
	rows, err := tx.Query(ctx, `SELECT id, parent_id FROM department`)
	if err != nil {
		return err
	}
	defer rows.Close()
	parentOf := map[kernel.ID]*kernel.ID{}
	for rows.Next() {
		var did kernel.ID
		var pid *kernel.ID
		if err := rows.Scan(&did, &pid); err != nil {
			return err
		}
		parentOf[did] = pid
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	// Check cycle: newParentID must not be id or any descendant of id.
	if newParentID != nil {
		if *newParentID == id {
			return errors.New(errors.KindParam, "tenancy.dept_cycle", "不能将部门移动到自身")
		}
		// Walk ancestors of newParentID — if we reach id, it's a cycle.
		cur := newParentID
		for cur != nil {
			p := parentOf[*cur]
			if p != nil && *p == id {
				return errors.New(errors.KindParam, "tenancy.dept_cycle", "不能移动到子孙部门下")
			}
			cur = p
		}
	}
	res, err := tx.Exec(ctx, `UPDATE department SET parent_id = $1 WHERE id = $2`, idPtrOrNil(newParentID), id)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.move_dept_failed", "移动部门失败", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "tenancy.dept_not_found", "部门不存在")
	}
	return tx.Commit(ctx)
}

// DescendantDeptIDs returns the IDs of all descendants of the given dept (not including itself),
// using the flat list fetched in one query.
func (s *Service) DescendantDeptIDs(ctx context.Context, pool *pgxpool.Pool, t *Tenant, deptID kernel.ID) ([]kernel.ID, error) {
	flat, err := s.ListDepts(ctx, pool, t)
	if err != nil {
		return nil, err
	}
	return collectDescendants(flat, deptID), nil
}

func collectDescendants(flat []DeptRow, rootID kernel.ID) []kernel.ID {
	// Build children map.
	childrenOf := map[kernel.ID][]kernel.ID{}
	for _, d := range flat {
		if d.ParentID != nil {
			childrenOf[*d.ParentID] = append(childrenOf[*d.ParentID], d.ID)
		}
	}
	// BFS.
	queue := childrenOf[rootID]
	var out []kernel.ID
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		out = append(out, cur)
		queue = append(queue, childrenOf[cur]...)
	}
	return out
}

// helpers ----------------------------------------------------------------

func idPtrOrNil(id *kernel.ID) any {
	if id == nil {
		return nil
	}
	return *id
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// registerDeptRoutes mounts /admin/depts/* (tenant_admin gated by the caller).
func registerDeptRoutes(r *gin.RouterGroup, svc *Service, pool *pgxpool.Pool) {
	r.GET("/admin/depts", func(c *gin.Context) {
		depts, err := svc.ListDepts(c.Request.Context(), pool, tenantFromCtx(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"depts": depts})
	})

	r.GET("/admin/depts/tree", func(c *gin.Context) {
		tree, err := svc.DeptTree(c.Request.Context(), pool, tenantFromCtx(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"tree": tree})
	})

	r.POST("/admin/depts", func(c *gin.Context) {
		var req struct {
			Name     string  `json:"name" binding:"required"`
			ParentID *string `json:"parent_id"`
			OrderNum int     `json:"order_num"`
			Leader   string  `json:"leader"`
			Phone    string  `json:"phone"`
			Email    string  `json:"email"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		cmd := CreateDeptCmd{
			Name: req.Name, OrderNum: req.OrderNum,
			Leader: req.Leader, Phone: req.Phone, Email: req.Email,
		}
		if req.ParentID != nil && *req.ParentID != "" {
			pid, err := kernel.ParseID(*req.ParentID)
			if err != nil {
				apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_parent_id", "parent_id 无效", err))
				return
			}
			cmd.ParentID = &pid
		}
		dept, err := svc.CreateDept(c.Request.Context(), pool, tenantFromCtx(c), cmd)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, dept)
	})

	r.PATCH("/admin/depts/:id", func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "部门 ID 无效", err))
			return
		}
		var req struct {
			Name     *string `json:"name"`
			OrderNum *int    `json:"order_num"`
			Leader   *string `json:"leader"`
			Phone    *string `json:"phone"`
			Email    *string `json:"email"`
			Status   *string `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.UpdateDept(c.Request.Context(), pool, tenantFromCtx(c), UpdateDeptCmd{
			DeptID: id, Name: req.Name, OrderNum: req.OrderNum,
			Leader: req.Leader, Phone: req.Phone, Email: req.Email, Status: req.Status,
		}); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/admin/depts/:id", func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "部门 ID 无效", err))
			return
		}
		if err := svc.DeleteDept(c.Request.Context(), pool, tenantFromCtx(c), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/admin/depts/:id/move", func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "部门 ID 无效", err))
			return
		}
		var req struct {
			ParentID *string `json:"parent_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		var newParent *kernel.ID
		if req.ParentID != nil && *req.ParentID != "" {
			pid, err := kernel.ParseID(*req.ParentID)
			if err != nil {
				apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_parent_id", "parent_id 无效", err))
				return
			}
			newParent = &pid
		}
		if err := svc.MoveDept(c.Request.Context(), pool, tenantFromCtx(c), id, newParent); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}
