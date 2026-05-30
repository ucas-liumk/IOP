package tenancy

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// DeptRow is a flat department record.
type DeptRow struct {
	ID            kernel.ID  `json:"id"`
	TenantID      kernel.ID  `json:"tenant_id"`
	Name          string     `json:"name"`
	OrgCode       string     `json:"org_code"`
	ParentID      *kernel.ID `json:"parent_id,omitempty"`
	OrgType       string     `json:"org_type"`
	OrderNum      int        `json:"order_num"`
	Leader        string     `json:"leader,omitempty"`
	LeaderAccount string     `json:"leader_account,omitempty"`
	Phone         string     `json:"phone,omitempty"`
	Email         string     `json:"email,omitempty"`
	Status        string     `json:"status"`
	Remark        string     `json:"remark,omitempty"`
	Path          string     `json:"path,omitempty"`
	IsRoot        bool       `json:"is_root"`
	CreatedAt     string     `json:"created_at"`
}

// DeptTreeNode embeds DeptRow with nested children.
type DeptTreeNode struct {
	DeptRow
	Children []*DeptTreeNode `json:"children,omitempty"`
}

// DeptListFilter narrows organization listings. Tenant scope is deliberately not
// caller-supplied; it always comes from the resolved Tenant.
type DeptListFilter struct {
	Search string
	Status string
}

// ListDepts returns a flat list of departments ordered by order_num, name.
func (s *Service) ListDepts(ctx context.Context, pool *pgxpool.Pool, t *Tenant) ([]DeptRow, error) {
	return s.ListDeptsFiltered(ctx, pool, t, DeptListFilter{})
}

// ListDeptsFiltered returns tenant-scoped organizations. Search/status matches
// keep ancestors in the returned set so tree UIs can preserve the full path.
func (s *Service) ListDeptsFiltered(ctx context.Context, pool *pgxpool.Pool, t *Tenant, filter DeptListFilter) ([]DeptRow, error) {
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
		`SELECT id, tenant_id, name, COALESCE(org_code,''), parent_id, COALESCE(org_type,'department'), order_num,
		        COALESCE(leader,''), COALESCE(leader_account,''), COALESCE(phone,''), COALESCE(email,''),
		        status, COALESCE(remark,''), COALESCE(is_root, false), to_char(created_at, 'YYYY-MM-DD HH24:MI:SS')
		 FROM department
		 WHERE tenant_id = $1 AND deleted_at IS NULL
		 ORDER BY order_num, name, org_code`, t.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DeptRow{}
	for rows.Next() {
		var d DeptRow
		if err := rows.Scan(&d.ID, &d.TenantID, &d.Name, &d.OrgCode, &d.ParentID, &d.OrgType, &d.OrderNum,
			&d.Leader, &d.LeaderAccount, &d.Phone, &d.Email, &d.Status, &d.Remark, &d.IsRoot, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out = attachDeptPaths(out)
	out = filterDeptRows(out, filter)
	return out, tx.Commit(ctx)
}

// DeptTree builds and returns the department tree.
func (s *Service) DeptTree(ctx context.Context, pool *pgxpool.Pool, t *Tenant) ([]*DeptTreeNode, error) {
	return s.DeptTreeFiltered(ctx, pool, t, DeptListFilter{})
}

// DeptTreeFiltered builds and returns a filtered department tree.
func (s *Service) DeptTreeFiltered(ctx context.Context, pool *pgxpool.Pool, t *Tenant, filter DeptListFilter) ([]*DeptTreeNode, error) {
	flat, err := s.ListDeptsFiltered(ctx, pool, t, filter)
	if err != nil {
		return nil, err
	}
	return buildDeptTree(flat), nil
}

// GetDept returns one tenant-scoped organization by id.
func (s *Service) GetDept(ctx context.Context, pool *pgxpool.Pool, t *Tenant, id kernel.ID) (*DeptRow, error) {
	flat, err := s.ListDepts(ctx, pool, t)
	if err != nil {
		return nil, err
	}
	for i := range flat {
		if flat[i].ID == id {
			return &flat[i], nil
		}
	}
	return nil, errors.New(errors.KindNotFound, "tenancy.dept_not_found", "部门不存在")
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

func attachDeptPaths(flat []DeptRow) []DeptRow {
	byID := make(map[kernel.ID]*DeptRow, len(flat))
	for i := range flat {
		byID[flat[i].ID] = &flat[i]
	}
	var pathOf func(id kernel.ID, seen map[kernel.ID]bool) string
	pathOf = func(id kernel.ID, seen map[kernel.ID]bool) string {
		d := byID[id]
		if d == nil {
			return ""
		}
		if seen[id] {
			return d.Name
		}
		seen[id] = true
		if d.ParentID == nil {
			return d.Name
		}
		pp := pathOf(*d.ParentID, seen)
		if pp == "" {
			return d.Name
		}
		return pp + "/" + d.Name
	}
	for i := range flat {
		flat[i].Path = pathOf(flat[i].ID, map[kernel.ID]bool{})
	}
	return flat
}

func filterDeptRows(flat []DeptRow, filter DeptListFilter) []DeptRow {
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	status := strings.TrimSpace(filter.Status)
	if search == "" && status == "" {
		return flat
	}
	byID := make(map[kernel.ID]DeptRow, len(flat))
	for _, d := range flat {
		byID[d.ID] = d
	}
	include := map[kernel.ID]bool{}
	for _, d := range flat {
		if status != "" && d.Status != status {
			continue
		}
		if search != "" {
			hay := strings.ToLower(d.Name + " " + d.OrgCode)
			if !strings.Contains(hay, search) {
				continue
			}
		}
		cur := d
		for {
			include[cur.ID] = true
			if cur.ParentID == nil {
				break
			}
			parent, ok := byID[*cur.ParentID]
			if !ok {
				break
			}
			cur = parent
		}
	}
	out := make([]DeptRow, 0, len(include))
	for _, d := range flat {
		if include[d.ID] {
			out = append(out, d)
		}
	}
	return out
}

// CreateDeptCmd is input for department creation.
type CreateDeptCmd struct {
	Name          string
	OrgCode       string
	ParentID      *kernel.ID
	OrgType       string
	OrderNum      int
	Leader        string
	LeaderAccount string
	Phone         string
	Email         string
	Status        string
	Remark        string
}

// CreateDept inserts a new department.
func (s *Service) CreateDept(ctx context.Context, pool *pgxpool.Pool, t *Tenant, cmd CreateDeptCmd) (*DeptRow, error) {
	cmd.Name = strings.TrimSpace(cmd.Name)
	cmd.OrgCode = strings.TrimSpace(cmd.OrgCode)
	cmd.OrgType = normalizeOrgType(cmd.OrgType)
	cmd.Status = normalizeDeptStatus(cmd.Status)
	if cmd.Name == "" {
		return nil, errors.New(errors.KindParam, "tenancy.dept_name_required", "部门名称不能为空")
	}
	if cmd.OrgCode == "" {
		return nil, errors.New(errors.KindParam, "tenancy.dept_code_required", "组织编码不能为空")
	}
	if err := validateDeptStatus(cmd.Status); err != nil {
		return nil, err
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
	parentID := cmd.ParentID
	if parentID == nil {
		rid, err := rootDeptID(ctx, tx, t)
		if err != nil {
			return nil, err
		}
		parentID = &rid
	}
	// Validate parent exists if provided.
	if parentID != nil {
		var count int
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM department
			 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
			*parentID, t.ID).Scan(&count); err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, errors.New(errors.KindParam, "tenancy.dept_parent_not_found", "父部门不存在")
		}
	}
	if err := ensureOrgCodeUnique(ctx, tx, t, cmd.OrgCode, nil); err != nil {
		return nil, err
	}
	id := kernel.NewID()
	var createdAt string
	if err := tx.QueryRow(ctx,
		`INSERT INTO department (
		    id, tenant_id, name, org_code, parent_id, org_type, order_num,
		    leader, leader_account, phone, email, status, remark, is_root
		 )
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, false)
		 RETURNING to_char(created_at, 'YYYY-MM-DD HH24:MI:SS')`,
		id, t.ID, cmd.Name, cmd.OrgCode, idPtrOrNil(parentID), cmd.OrgType, cmd.OrderNum,
		nullStr(cmd.Leader), nullStr(cmd.LeaderAccount), nullStr(cmd.Phone), nullStr(cmd.Email),
		cmd.Status, nullStr(cmd.Remark),
	).Scan(&createdAt); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "tenancy.create_dept_failed", "创建部门失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	_ = s.bus.Publish(ctx, "tenancy.dept_created", map[string]any{
		"tenant_id": t.ID, "dept_id": id, "org_code": cmd.OrgCode,
	})
	return &DeptRow{
		ID: id, TenantID: t.ID, Name: cmd.Name, OrgCode: cmd.OrgCode, ParentID: parentID,
		OrgType: cmd.OrgType, OrderNum: cmd.OrderNum,
		Leader: cmd.Leader, LeaderAccount: cmd.LeaderAccount, Phone: cmd.Phone, Email: cmd.Email,
		Status: cmd.Status, Remark: cmd.Remark, CreatedAt: createdAt,
	}, nil
}

// UpdateDeptCmd holds optional fields for patching a department.
type UpdateDeptCmd struct {
	DeptID        kernel.ID
	Name          *string
	OrgCode       *string
	ParentID      *kernel.ID // nil = don't change; move is handled by MoveDept
	OrgType       *string
	OrderNum      *int
	Leader        *string
	LeaderAccount *string
	Phone         *string
	Email         *string
	Status        *string
	Remark        *string
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
	var isRoot bool
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(is_root, false)
		 FROM department
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		cmd.DeptID, t.ID).Scan(&isRoot); err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return errors.New(errors.KindNotFound, "tenancy.dept_not_found", "部门不存在")
		}
		return err
	}
	if cmd.Status != nil {
		status := normalizeDeptStatus(*cmd.Status)
		if err := validateDeptStatus(status); err != nil {
			return err
		}
		*cmd.Status = status
	}
	if isRoot && cmd.Status != nil && *cmd.Status != "active" {
		return errors.New(errors.KindParam, "tenancy.dept_root_status", "根组织不能禁用")
	}
	if cmd.OrgType != nil {
		orgType := normalizeOrgType(*cmd.OrgType)
		*cmd.OrgType = orgType
	}
	if cmd.Name != nil {
		name := strings.TrimSpace(*cmd.Name)
		if name == "" {
			return errors.New(errors.KindParam, "tenancy.dept_name_required", "部门名称不能为空")
		}
		*cmd.Name = name
	}
	if cmd.OrgCode != nil {
		code := strings.TrimSpace(*cmd.OrgCode)
		if code == "" {
			return errors.New(errors.KindParam, "tenancy.dept_code_required", "组织编码不能为空")
		}
		if err := ensureOrgCodeUnique(ctx, tx, t, code, &cmd.DeptID); err != nil {
			return err
		}
		*cmd.OrgCode = code
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
	if cmd.OrgCode != nil {
		addField("org_code", *cmd.OrgCode)
	}
	if cmd.OrgType != nil {
		addField("org_type", *cmd.OrgType)
	}
	if cmd.OrderNum != nil {
		addField("order_num", *cmd.OrderNum)
	}
	if cmd.Leader != nil {
		addField("leader", nullStr(*cmd.Leader))
	}
	if cmd.LeaderAccount != nil {
		addField("leader_account", nullStr(*cmd.LeaderAccount))
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
	if cmd.Remark != nil {
		addField("remark", nullStr(*cmd.Remark))
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
	sql += fmt.Sprintf(" WHERE id = $1 AND tenant_id = $%d AND deleted_at IS NULL", idx)
	args = append(args, t.ID)
	res, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.update_dept_failed", "更新部门失败", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "tenancy.dept_not_found", "部门不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "tenancy.dept_updated", map[string]any{
		"tenant_id": t.ID, "dept_id": cmd.DeptID,
	})
	return nil
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
	var isRoot bool
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(is_root, false)
		 FROM department
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		id, t.ID).Scan(&isRoot); err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return errors.New(errors.KindNotFound, "tenancy.dept_not_found", "部门不存在")
		}
		return err
	}
	if isRoot {
		return errors.New(errors.KindParam, "tenancy.dept_root_delete_forbidden", "根组织不能删除")
	}
	var children int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM department
		 WHERE parent_id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		id, t.ID).Scan(&children); err != nil {
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
	res, err := tx.Exec(ctx,
		`UPDATE department
		 SET deleted_at = now(), status = 'disabled'
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		id, t.ID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "tenancy.dept_not_found", "部门不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "tenancy.dept_deleted", map[string]any{
		"tenant_id": t.ID, "dept_id": id,
	})
	return nil
}

// SetDeptStatus enables/disables an organization. cascade applies the same
// status to every descendant, used by the UI after the admin confirms.
func (s *Service) SetDeptStatus(ctx context.Context, pool *pgxpool.Pool, t *Tenant, id kernel.ID, status string, cascade bool) error {
	status = normalizeDeptStatus(status)
	if err := validateDeptStatus(status); err != nil {
		return err
	}
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
	var isRoot bool
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(is_root, false)
		 FROM department
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		id, t.ID).Scan(&isRoot); err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return errors.New(errors.KindNotFound, "tenancy.dept_not_found", "部门不存在")
		}
		return err
	}
	if isRoot && status != "active" {
		return errors.New(errors.KindParam, "tenancy.dept_root_status", "根组织不能禁用")
	}

	targets := []kernel.ID{id}
	if cascade {
		rows, err := tx.Query(ctx,
			`SELECT id, parent_id
			 FROM department
			 WHERE tenant_id = $1 AND deleted_at IS NULL`, t.ID)
		if err != nil {
			return err
		}
		childrenOf := map[kernel.ID][]kernel.ID{}
		for rows.Next() {
			var did kernel.ID
			var pid *kernel.ID
			if err := rows.Scan(&did, &pid); err != nil {
				rows.Close()
				return err
			}
			if pid != nil {
				childrenOf[*pid] = append(childrenOf[*pid], did)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		targets = append(targets, collectChildIDs(childrenOf, id)...)
	}
	res, err := tx.Exec(ctx,
		`UPDATE department
		 SET status = $1
		 WHERE id = ANY($2) AND tenant_id = $3 AND deleted_at IS NULL`,
		status, targets, t.ID)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.update_dept_status_failed", "更新组织状态失败", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "tenancy.dept_not_found", "部门不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "tenancy.dept_status_changed", map[string]any{
		"tenant_id": t.ID, "dept_id": id, "status": status, "cascade": cascade,
	})
	return nil
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
	rootID, err := rootDeptID(ctx, tx, t)
	if err != nil {
		return err
	}
	if id == rootID {
		return errors.New(errors.KindParam, "tenancy.dept_root_move_forbidden", "根组织不能移动")
	}
	parentID := newParentID
	if parentID == nil {
		parentID = &rootID
	}
	// Load all depts to check for cycles.
	rows, err := tx.Query(ctx,
		`SELECT id, parent_id
		 FROM department
		 WHERE tenant_id = $1 AND deleted_at IS NULL`, t.ID)
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
	if parentID != nil {
		if *parentID == id {
			return errors.New(errors.KindParam, "tenancy.dept_cycle", "不能将部门移动到自身")
		}
		if _, ok := parentOf[*parentID]; !ok {
			return errors.New(errors.KindParam, "tenancy.dept_parent_not_found", "父部门不存在")
		}
		// Walk ancestors of newParentID — if we reach id, it's a cycle.
		cur := parentID
		for cur != nil {
			p := parentOf[*cur]
			if p != nil && *p == id {
				return errors.New(errors.KindParam, "tenancy.dept_cycle", "不能移动到子孙部门下")
			}
			cur = p
		}
	}
	res, err := tx.Exec(ctx,
		`UPDATE department
		 SET parent_id = $1
		 WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`,
		idPtrOrNil(parentID), id, t.ID)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.move_dept_failed", "移动部门失败", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "tenancy.dept_not_found", "部门不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "tenancy.dept_moved", map[string]any{
		"tenant_id": t.ID, "dept_id": id, "parent_id": idPtrOrNil(parentID),
	})
	return nil
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

func collectChildIDs(childrenOf map[kernel.ID][]kernel.ID, rootID kernel.ID) []kernel.ID {
	queue := append([]kernel.ID(nil), childrenOf[rootID]...)
	out := []kernel.ID{}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		out = append(out, cur)
		queue = append(queue, childrenOf[cur]...)
	}
	return out
}

// EnsureRootDept creates the one root organization node for the tenant if it
// does not already exist. It is idempotent and safe to run after provisioning or
// schema synchronization.
func (s *Service) EnsureRootDept(ctx context.Context, pool *pgxpool.Pool, t *Tenant) (*DeptRow, error) {
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
	rootID, err := rootDeptID(ctx, tx, t)
	if err != nil {
		return nil, err
	}
	var d DeptRow
	if err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, name, COALESCE(org_code,''), parent_id, COALESCE(org_type,'unit'), order_num,
		        COALESCE(leader,''), COALESCE(leader_account,''), COALESCE(phone,''), COALESCE(email,''),
		        status, COALESCE(remark,''), COALESCE(is_root, false), to_char(created_at, 'YYYY-MM-DD HH24:MI:SS')
		 FROM department
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		rootID, t.ID).Scan(&d.ID, &d.TenantID, &d.Name, &d.OrgCode, &d.ParentID, &d.OrgType, &d.OrderNum,
		&d.Leader, &d.LeaderAccount, &d.Phone, &d.Email, &d.Status, &d.Remark, &d.IsRoot, &d.CreatedAt); err != nil {
		return nil, err
	}
	d.Path = d.Name
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &d, nil
}

// helpers ----------------------------------------------------------------

func normalizeOrgType(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "department"
	}
	return v
}

func normalizeDeptStatus(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" || v == "normal" || v == "enabled" {
		return "active"
	}
	return v
}

func validateDeptStatus(status string) error {
	if status != "active" && status != "disabled" {
		return errors.New(errors.KindParam, "tenancy.invalid_dept_status", "状态只能是 active 或 disabled")
	}
	return nil
}

func ensureOrgCodeUnique(ctx context.Context, tx pgx.Tx, t *Tenant, code string, excludeID *kernel.ID) error {
	var count int
	args := []any{t.ID, code}
	sql := `SELECT count(*) FROM department
	        WHERE tenant_id = $1 AND lower(org_code) = lower($2) AND deleted_at IS NULL`
	if excludeID != nil {
		sql += ` AND id <> $3`
		args = append(args, *excludeID)
	}
	if err := tx.QueryRow(ctx, sql, args...).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return errors.New(errors.KindConflict, "tenancy.dept_code_exists", "同一租户内组织编码已存在")
	}
	return nil
}

func rootDeptID(ctx context.Context, tx pgx.Tx, t *Tenant) (kernel.ID, error) {
	var id kernel.ID
	err := tx.QueryRow(ctx,
		`SELECT id
		 FROM department
		 WHERE tenant_id = $1 AND is_root = true AND deleted_at IS NULL
		 ORDER BY created_at ASC
		 LIMIT 1`, t.ID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !stderrors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	name := strings.TrimSpace(t.Name)
	if name == "" {
		name = strings.TrimSpace(t.Slug)
	}
	if name == "" {
		name = "根组织"
	}
	id = kernel.NewID()
	if _, err := tx.Exec(ctx,
		`INSERT INTO department (id, tenant_id, name, org_code, parent_id, org_type, order_num, status, is_root)
		 VALUES ($1, $2, $3, 'ROOT', NULL, 'unit', 0, 'active', true)`,
		id, t.ID, name); err != nil {
		return "", errors.Wrap(errors.KindDatabase, "tenancy.create_root_dept_failed", "创建根组织失败", err)
	}
	return id, nil
}

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

func deptFilterFromQuery(c *gin.Context) DeptListFilter {
	return DeptListFilter{
		Search: c.Query("search"),
		Status: c.Query("status"),
	}
}

func dryRunFromRequest(c *gin.Context) bool {
	v := strings.ToLower(strings.TrimSpace(c.Query("dry_run")))
	if v == "" {
		v = strings.ToLower(strings.TrimSpace(c.Query("dryRun")))
	}
	if v == "" {
		v = strings.ToLower(strings.TrimSpace(c.PostForm("dry_run")))
	}
	return v == "1" || v == "true" || v == "yes"
}

type deptCreateRequest struct {
	Name          string  `json:"name" binding:"required"`
	OrgCode       string  `json:"org_code" binding:"required"`
	ParentID      *string `json:"parent_id"`
	OrgType       string  `json:"org_type"`
	OrderNum      int     `json:"order_num"`
	Leader        string  `json:"leader"`
	LeaderAccount string  `json:"leader_account"`
	Phone         string  `json:"phone"`
	Email         string  `json:"email"`
	Status        string  `json:"status"`
	Remark        string  `json:"remark"`
}

type deptPatchRequest struct {
	Name          *string `json:"name"`
	OrgCode       *string `json:"org_code"`
	ParentID      *string `json:"parent_id"`
	OrgType       *string `json:"org_type"`
	OrderNum      *int    `json:"order_num"`
	Leader        *string `json:"leader"`
	LeaderAccount *string `json:"leader_account"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	Status        *string `json:"status"`
	Remark        *string `json:"remark"`
}

func parseOptionalID(v *string) (*kernel.ID, bool, error) {
	if v == nil {
		return nil, false, nil
	}
	if strings.TrimSpace(*v) == "" {
		return nil, true, nil
	}
	id, err := kernel.ParseID(*v)
	if err != nil {
		return nil, true, err
	}
	return &id, true, nil
}

func createDeptCmdFromRequest(req deptCreateRequest) (CreateDeptCmd, error) {
	pid, _, err := parseOptionalID(req.ParentID)
	if err != nil {
		return CreateDeptCmd{}, errors.Wrap(errors.KindParam, "tenancy.invalid_parent_id", "parent_id 无效", err)
	}
	return CreateDeptCmd{
		Name: req.Name, OrgCode: req.OrgCode, ParentID: pid, OrgType: req.OrgType,
		OrderNum: req.OrderNum, Leader: req.Leader, LeaderAccount: req.LeaderAccount,
		Phone: req.Phone, Email: req.Email, Status: req.Status, Remark: req.Remark,
	}, nil
}

// registerDeptRoutes mounts /admin/depts/* (tenant_admin gated by the caller).
// authz adds the per-route dept:write gate on import/export endpoints.
func registerDeptRoutes(r *gin.RouterGroup, svc *Service, pool *pgxpool.Pool, authz AuthzFunc) {
	// Export / template / import are registered before parameterized routes so the
	// literal paths take precedence. All gated with dept:write.
	r.GET("/admin/depts/export", authz("dept", "write"), func(c *gin.Context) {
		rows, err := svc.ExportDepts(c.Request.Context(), pool, tenantFromCtx(c), deptFilterFromQuery(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.XLSX(c, "departments.xlsx", rows)
	})

	r.GET("/admin/depts/template", authz("dept", "write"), func(c *gin.Context) {
		apiresp.XLSX(c, "departments_template.xlsx", DeptTemplateRows())
	})

	r.POST("/admin/depts/import", authz("dept", "write"), func(c *gin.Context) {
		records, err := apiresp.ParseTabularUpload(c, "file")
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		res, err := svc.ImportDepts(c.Request.Context(), pool, tenantFromCtx(c), records, dryRunFromRequest(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, res)
	})

	r.GET("/admin/depts", authz("dept", "read"), func(c *gin.Context) {
		depts, err := svc.ListDeptsFiltered(c.Request.Context(), pool, tenantFromCtx(c), deptFilterFromQuery(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"depts": depts})
	})

	r.GET("/admin/depts/tree", authz("dept", "read"), func(c *gin.Context) {
		tree, err := svc.DeptTreeFiltered(c.Request.Context(), pool, tenantFromCtx(c), deptFilterFromQuery(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"tree": tree})
	})

	r.GET("/admin/depts/:id", authz("dept", "read"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "部门 ID 无效", err))
			return
		}
		dept, err := svc.GetDept(c.Request.Context(), pool, tenantFromCtx(c), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, dept)
	})

	r.POST("/admin/depts", authz("dept", "write"), func(c *gin.Context) {
		var req deptCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		cmd, err := createDeptCmdFromRequest(req)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		dept, err := svc.CreateDept(c.Request.Context(), pool, tenantFromCtx(c), cmd)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, dept)
	})

	r.PATCH("/admin/depts/:id", authz("dept", "write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "部门 ID 无效", err))
			return
		}
		var req deptPatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.UpdateDept(c.Request.Context(), pool, tenantFromCtx(c), UpdateDeptCmd{
			DeptID: id, Name: req.Name, OrgCode: req.OrgCode, OrgType: req.OrgType,
			OrderNum: req.OrderNum, Leader: req.Leader, LeaderAccount: req.LeaderAccount,
			Phone: req.Phone, Email: req.Email, Status: req.Status, Remark: req.Remark,
		}); err != nil {
			apiresp.Fail(c, err)
			return
		}
		if parentID, present, err := parseOptionalID(req.ParentID); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_parent_id", "parent_id 无效", err))
			return
		} else if present {
			if err := svc.MoveDept(c.Request.Context(), pool, tenantFromCtx(c), id, parentID); err != nil {
				apiresp.Fail(c, err)
				return
			}
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/admin/depts/:id/status", authz("dept", "write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "部门 ID 无效", err))
			return
		}
		var req struct {
			Status  string `json:"status" binding:"required"`
			Cascade bool   `json:"cascade"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.SetDeptStatus(c.Request.Context(), pool, tenantFromCtx(c), id, req.Status, req.Cascade); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/admin/depts/:id", authz("dept", "write"), func(c *gin.Context) {
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

	r.POST("/admin/depts/:id/move", authz("dept", "write"), func(c *gin.Context) {
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
