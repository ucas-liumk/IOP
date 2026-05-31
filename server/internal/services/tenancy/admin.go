package tenancy

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

var memberPhoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

// MemberPost is a post assigned to a member (id + code + name).
type MemberPost struct {
	PostID kernel.ID `json:"post_id"`
	Code   string    `json:"code"`
	Name   string    `json:"name"`
}

// MemberRole is a role granted to a member inside one tenant.
type MemberRole struct {
	RoleID kernel.ID `json:"role_id"`
	Code   string    `json:"code"`
	Name   string    `json:"name"`
}

// MemberRow joins tenant_<slug>.member with public.platform_user for admin listing.
// Username (the primary login identity) is joined in from platform_user; Email is an
// optional secondary field carried from the member projection.
type MemberRow struct {
	MemberID       kernel.ID    `json:"member_id"`
	PlatformUserID kernel.ID    `json:"platform_user_id"`
	Username       string       `json:"username"`
	DisplayName    string       `json:"display_name"`
	Email          string       `json:"email,omitempty"`
	Department     string       `json:"department"`
	DeptID         *kernel.ID   `json:"dept_id,omitempty"`
	DeptCode       string       `json:"dept_code,omitempty"`
	DeptPath       string       `json:"dept_path,omitempty"`
	Gender         string       `json:"gender,omitempty"`
	Title          string       `json:"title"`
	Phone          string       `json:"phone"`
	Status         string       `json:"status"`
	Remark         string       `json:"remark,omitempty"`
	JoinedAt       string       `json:"joined_at"`
	Posts          []MemberPost `json:"posts"`
	Roles          []MemberRole `json:"roles"`
}

// ListMembersCmd parameters the member listing: pagination + free-text search +
// optional department filter (with optional subtree expansion).
type ListMembersCmd struct {
	Page    kernel.Pagination
	Search  string
	Status  string
	DeptID  *kernel.ID // nil = no dept filter
	Subtree bool       // when DeptID set, also include descendant departments
	IDs     []kernel.ID
	All     bool
}

// ListMembers returns one page of members of the current tenant (via
// TenantContext) + their posts, plus the total count of rows matching the same
// filters (search / dept_id / subtree). Supports free-text search and a
// department filter (optionally subtree).
func (s *Service) ListMembers(ctx context.Context, pool *pgxpool.Pool, t *Tenant, cmd ListMembersCmd) (*kernel.Page[MemberRow], error) {
	p := cmd.Page.Normalize()

	// Resolve dept filter ids (subtree expansion is computed from the in-Go tree).
	var deptFilter []kernel.ID
	if cmd.DeptID != nil {
		deptFilter = []kernel.ID{*cmd.DeptID}
		if cmd.Subtree {
			desc, err := s.DescendantDeptIDs(ctx, pool, t, *cmd.DeptID)
			if err != nil {
				return nil, err
			}
			deptFilter = append(deptFilter, desc...)
		}
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
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return nil, err
	}

	// Build the shared WHERE clause once (with its own arg list) so the page query
	// and the COUNT(*) total apply identical filters.
	filterArgs := []any{}
	fidx := 1
	where := ""
	if strings.TrimSpace(cmd.Search) != "" {
		// Primary search is over display_name / username / phone (identity is
		// username/phone-first). email/status/department are kept as convenience
		// matches for the admin search box.
		where += fmt.Sprintf(" AND (m.display_name ILIKE $%d OR COALESCE(u.username,'') ILIKE $%d OR COALESCE(u.phone,m.phone,'') ILIKE $%d OR COALESCE(u.email,m.email,'') ILIKE $%d OR COALESCE(m.department,'') ILIKE $%d OR m.status ILIKE $%d)", fidx, fidx, fidx, fidx, fidx, fidx)
		filterArgs = append(filterArgs, "%"+strings.TrimSpace(cmd.Search)+"%")
		fidx++
	}
	if status := strings.TrimSpace(cmd.Status); status != "" {
		where += fmt.Sprintf(" AND m.status = $%d", fidx)
		filterArgs = append(filterArgs, status)
		fidx++
	}
	if cmd.DeptID != nil {
		where += fmt.Sprintf(" AND m.dept_id = ANY($%d)", fidx)
		filterArgs = append(filterArgs, deptFilter)
		fidx++
	}
	if len(cmd.IDs) > 0 {
		where += fmt.Sprintf(" AND m.id = ANY($%d)", fidx)
		filterArgs = append(filterArgs, cmd.IDs)
		fidx++
	}

	// Total matching rows (same filters, no LIMIT/OFFSET).
	var total int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM member m
		 LEFT JOIN public.platform_user u ON u.id = m.platform_user_id
		 WHERE 1=1`+where, filterArgs...).Scan(&total); err != nil {
		return nil, err
	}

	args := append([]any{}, filterArgs...)
	sql := `SELECT m.id, m.platform_user_id, COALESCE(u.username,''), m.display_name, COALESCE(u.email,m.email,''),
		        COALESCE(d.name, m.department, ''), m.dept_id, COALESCE(d.org_code,''),
		        COALESCE(m.gender,''), COALESCE(m.title,''), COALESCE(u.phone,m.phone,''),
		        m.status, COALESCE(m.remark,''), to_char(m.joined_at, 'YYYY-MM-DD HH24:MI:SS')
		 FROM member m
		 LEFT JOIN public.platform_user u ON u.id = m.platform_user_id
		 LEFT JOIN department d ON d.id = m.dept_id AND d.deleted_at IS NULL AND d.tenant_id = $` + fmt.Sprint(fidx) + `
		 WHERE 1=1` + where + `
		 ORDER BY m.joined_at DESC`
	args = append(args, t.ID)
	if !cmd.All {
		sql += fmt.Sprintf(`
		 LIMIT $%d OFFSET $%d`, fidx+1, fidx+2)
		args = append(args, p.PageSize, p.Offset())
	}
	rows, err := tx.Query(ctx,
		sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MemberRow{}
	idsList := []kernel.ID{}
	for rows.Next() {
		var r MemberRow
		if err := rows.Scan(&r.MemberID, &r.PlatformUserID, &r.Username, &r.DisplayName, &r.Email,
			&r.Department, &r.DeptID, &r.DeptCode, &r.Gender, &r.Title, &r.Phone, &r.Status, &r.Remark, &r.JoinedAt); err != nil {
			return nil, err
		}
		r.Posts = []MemberPost{}
		r.Roles = []MemberRole{}
		out = append(out, r)
		idsList = append(idsList, r.MemberID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Attach posts per member in one query.
	if len(idsList) > 0 {
		prows, err := tx.Query(ctx,
			`SELECT mp.member_id, p.id, p.code, p.name
			 FROM member_post mp JOIN post p ON p.id = mp.post_id
			 WHERE mp.member_id = ANY($1) AND p.deleted_at IS NULL
			 ORDER BY p.order_num, p.code`, idsList)
		if err != nil {
			return nil, err
		}
		defer prows.Close()
		byMember := map[kernel.ID][]MemberPost{}
		for prows.Next() {
			var mid kernel.ID
			var mp MemberPost
			if err := prows.Scan(&mid, &mp.PostID, &mp.Code, &mp.Name); err != nil {
				return nil, err
			}
			byMember[mid] = append(byMember[mid], mp)
		}
		if err := prows.Err(); err != nil {
			return nil, err
		}
		for i := range out {
			if ps := byMember[out[i].MemberID]; ps != nil {
				out[i].Posts = ps
			}
		}
	}
	if len(idsList) > 0 {
		rrows, err := tx.Query(ctx,
			`SELECT g.member_id, r.id, r.code, r.name
			 FROM public.role_grant g JOIN public.role r ON r.id = g.role_id
			 WHERE g.tenant_id = $1 AND g.member_id = ANY($2) AND r.deleted_at IS NULL
			 ORDER BY r.tenant_id NULLS FIRST, r.code`, t.ID, idsList)
		if err != nil {
			return nil, err
		}
		defer rrows.Close()
		byMember := map[kernel.ID][]MemberRole{}
		for rrows.Next() {
			var mid kernel.ID
			var mr MemberRole
			if err := rrows.Scan(&mid, &mr.RoleID, &mr.Code, &mr.Name); err != nil {
				return nil, err
			}
			byMember[mid] = append(byMember[mid], mr)
		}
		if err := rrows.Err(); err != nil {
			return nil, err
		}
		for i := range out {
			if rs := byMember[out[i].MemberID]; rs != nil {
				out[i].Roles = rs
			}
		}
	}
	if len(out) > 0 {
		depts, err := listDeptRowsInTx(ctx, tx, t.ID)
		if err != nil {
			return nil, err
		}
		byDept := map[kernel.ID]DeptRow{}
		for _, d := range depts {
			byDept[d.ID] = d
		}
		for i := range out {
			if out[i].DeptID != nil {
				if d, ok := byDept[*out[i].DeptID]; ok {
					out[i].Department = d.Name
					out[i].DeptCode = d.OrgCode
					out[i].DeptPath = d.Path
				}
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &kernel.Page[MemberRow]{Data: out, Total: total, Page: p.Page, PageSize: p.PageSize}, nil
}

func listDeptRowsInTx(ctx context.Context, tx pgx.Tx, tenantID kernel.ID) ([]DeptRow, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, name, COALESCE(org_code,''), parent_id, COALESCE(org_type,'department'), order_num,
		        COALESCE(leader,''), COALESCE(leader_account,''), COALESCE(phone,''), COALESCE(email,''),
		        status, COALESCE(remark,''), COALESCE(is_root, false), to_char(created_at, 'YYYY-MM-DD HH24:MI:SS')
		 FROM department
		 WHERE tenant_id = $1 AND deleted_at IS NULL
		 ORDER BY order_num, name, org_code`, tenantID)
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
	return attachDeptPaths(out), nil
}

func (s *Service) GetMember(ctx context.Context, pool *pgxpool.Pool, t *Tenant, id kernel.ID) (*MemberRow, error) {
	page, err := s.ListMembers(ctx, pool, t, ListMembersCmd{
		Page: kernel.Pagination{Page: 1, PageSize: 1},
		IDs:  []kernel.ID{id},
	})
	if err != nil {
		return nil, err
	}
	if len(page.Data) == 0 {
		return nil, errors.New(errors.KindNotFound, "tenancy.member_not_found", "成员不存在")
	}
	return &page.Data[0], nil
}

// MemberGroupNode is the organization tree decorated with direct members.
type MemberGroupNode struct {
	Dept      DeptRow            `json:"dept"`
	UserCount int                `json:"user_count"`
	Users     []MemberRow        `json:"users"`
	Children  []*MemberGroupNode `json:"children,omitempty"`
}

func (s *Service) GroupMembers(ctx context.Context, pool *pgxpool.Pool, t *Tenant, cmd ListMembersCmd) ([]*MemberGroupNode, error) {
	depts, err := s.ListDeptsFiltered(ctx, pool, t, DeptListFilter{Status: "active"})
	if err != nil {
		return nil, err
	}
	cmd.All = true
	cmd.Page = kernel.Pagination{Page: 1, PageSize: 200}
	page, err := s.ListMembers(ctx, pool, t, cmd)
	if err != nil {
		return nil, err
	}
	byID := make(map[kernel.ID]*MemberGroupNode, len(depts))
	for i := range depts {
		d := depts[i]
		byID[d.ID] = &MemberGroupNode{Dept: d, Users: []MemberRow{}}
	}
	for _, m := range page.Data {
		if m.DeptID == nil {
			continue
		}
		if n := byID[*m.DeptID]; n != nil {
			n.Users = append(n.Users, m)
			n.UserCount++
		}
	}
	roots := []*MemberGroupNode{}
	for _, d := range depts {
		n := byID[d.ID]
		if d.ParentID == nil {
			roots = append(roots, n)
			continue
		}
		if p := byID[*d.ParentID]; p != nil {
			p.Children = append(p.Children, n)
		} else {
			roots = append(roots, n)
		}
	}
	return roots, nil
}

// UpdateMember updates editable fields on a member row in the tenant schema.
// DeptID is a tri-state: nil = leave unchanged; *DeptID = set/clear (set ClearDept
// to clear to NULL, otherwise the pointed-to value is validated + applied).
type UpdateMemberCmd struct {
	MemberID    kernel.ID
	DisplayName *string
	Department  *string
	Title       *string
	Phone       *string
	Email       *string
	Gender      *string
	Remark      *string
	DeptID      *kernel.ID // when SetDept is true: nil clears, non-nil assigns
	SetDept     bool       // whether dept_id was present in the request
}

func (s *Service) UpdateMember(ctx context.Context, pool *pgxpool.Pool, t *Tenant, cmd UpdateMemberCmd) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return err
	}

	var platformUserID kernel.ID
	var isPlatformAdmin bool
	if err := tx.QueryRow(ctx,
		`SELECT m.platform_user_id, COALESCE(u.is_platform_admin, false)
		 FROM member m JOIN public.platform_user u ON u.id = m.platform_user_id
		 WHERE m.id = $1`,
		cmd.MemberID).Scan(&platformUserID, &isPlatformAdmin); err != nil {
		if pgx.ErrNoRows == err {
			return errors.New(errors.KindNotFound, "tenancy.member_not_found", "成员不存在")
		}
		return err
	}
	if isPlatformAdmin {
		return errors.New(errors.KindForbidden, "tenancy.platform_admin_locked", "不能操作平台管理员账号")
	}

	// Build dynamic update — only set fields that were provided
	sets := []string{}
	args := []any{cmd.MemberID}
	idx := 2
	if cmd.DisplayName != nil {
		sets = append(sets, fmt.Sprintf("display_name = $%d", idx))
		args = append(args, *cmd.DisplayName)
		idx++
	}
	if cmd.Department != nil {
		sets = append(sets, fmt.Sprintf("department = $%d", idx))
		args = append(args, *cmd.Department)
		idx++
	}
	if cmd.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", idx))
		args = append(args, *cmd.Title)
		idx++
	}
	if cmd.Phone != nil {
		phone := strings.TrimSpace(*cmd.Phone)
		if phone != "" && !memberPhoneRe.MatchString(phone) {
			return errors.New(errors.KindParam, "tenancy.invalid_phone", "手机号格式错误")
		}
		if phone != "" {
			var dup int
			if err := tx.QueryRow(ctx,
				`SELECT count(*) FROM public.platform_user WHERE phone = $1 AND id <> $2`,
				phone, platformUserID).Scan(&dup); err != nil {
				return err
			}
			if dup > 0 {
				return errors.New(errors.KindConflict, "tenancy.phone_taken", "手机号已被注册")
			}
		}
		sets = append(sets, fmt.Sprintf("phone = $%d", idx))
		args = append(args, nullStr(phone))
		if _, err := tx.Exec(ctx,
			`UPDATE public.platform_user SET phone = $1 WHERE id = $2`,
			nullStr(phone), platformUserID); err != nil {
			return errors.Wrap(errors.KindDatabase, "tenancy.update_user_phone_failed", "更新手机号失败", err)
		}
		idx++
	}
	if cmd.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*cmd.Email))
		if email != "" && !strings.Contains(email, "@") {
			return errors.New(errors.KindParam, "tenancy.invalid_email", "邮箱格式错误")
		}
		if email != "" {
			var dup int
			if err := tx.QueryRow(ctx,
				`SELECT count(*) FROM public.platform_user WHERE email = $1 AND id <> $2`,
				email, platformUserID).Scan(&dup); err != nil {
				return err
			}
			if dup > 0 {
				return errors.New(errors.KindConflict, "tenancy.email_taken", "邮箱已被注册")
			}
		}
		sets = append(sets, fmt.Sprintf("email = $%d", idx))
		args = append(args, nullStr(email))
		if _, err := tx.Exec(ctx,
			`UPDATE public.platform_user SET email = $1 WHERE id = $2`,
			nullStr(email), platformUserID); err != nil {
			return errors.Wrap(errors.KindDatabase, "tenancy.update_user_email_failed", "更新邮箱失败", err)
		}
		idx++
	}
	if cmd.Gender != nil {
		sets = append(sets, fmt.Sprintf("gender = $%d", idx))
		args = append(args, strings.TrimSpace(*cmd.Gender))
		idx++
	}
	if cmd.Remark != nil {
		sets = append(sets, fmt.Sprintf("remark = $%d", idx))
		args = append(args, strings.TrimSpace(*cmd.Remark))
		idx++
	}
	if cmd.SetDept {
		if cmd.DeptID != nil {
			// Validate the target department exists and is assignable in this tenant schema.
			var count int
			if err := tx.QueryRow(ctx,
				`SELECT count(*) FROM department
				 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL AND status = 'active'`,
				*cmd.DeptID, t.ID).Scan(&count); err != nil {
				return err
			}
			if count == 0 {
				return errors.New(errors.KindParam, "tenancy.dept_not_assignable", "部门不存在或已停用")
			}
		}
		sets = append(sets, fmt.Sprintf("dept_id = $%d", idx))
		args = append(args, idPtrOrNil(cmd.DeptID))
		idx++
	}
	if len(sets) == 0 {
		return nil
	}
	sql := "UPDATE member SET "
	for i, ss := range sets {
		if i > 0 {
			sql += ", "
		}
		sql += ss
	}
	sql += " WHERE id = $1"
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.update_member_failed", "更新成员失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "tenancy.member_updated", map[string]any{
		"tenant_id": t.ID, "member_id": cmd.MemberID,
	})
	return nil
}

// AssignMemberPost adds a (member, post) mapping. Validates both rows exist; the
// mapping is idempotent (ON CONFLICT DO NOTHING).
func (s *Service) AssignMemberPost(ctx context.Context, pool *pgxpool.Pool, t *Tenant, memberID, postID kernel.ID) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return err
	}
	var mc, pc int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM member WHERE id = $1`, memberID).Scan(&mc); err != nil {
		return err
	}
	if mc == 0 {
		return errors.New(errors.KindNotFound, "tenancy.member_not_found", "成员不存在")
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM post WHERE id = $1`, postID).Scan(&pc); err != nil {
		return err
	}
	if pc == 0 {
		return errors.New(errors.KindNotFound, "tenancy.post_not_found", "岗位不存在")
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO member_post (member_id, post_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		memberID, postID); err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.assign_post_failed", "分配岗位失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "tenancy.member_post_assigned", map[string]any{
		"tenant_id": t.ID, "member_id": memberID, "post_id": postID,
	})
	return nil
}

// RemoveMemberPost removes a (member, post) mapping.
func (s *Service) RemoveMemberPost(ctx context.Context, pool *pgxpool.Pool, t *Tenant, memberID, postID kernel.ID) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM member_post WHERE member_id = $1 AND post_id = $2`, memberID, postID); err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.remove_post_failed", "移除岗位失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "tenancy.member_post_removed", map[string]any{
		"tenant_id": t.ID, "member_id": memberID, "post_id": postID,
	})
	return nil
}

// SetMemberStatus toggles a member between active / disabled.
func (s *Service) SetMemberStatus(ctx context.Context, pool *pgxpool.Pool, t *Tenant, id kernel.ID, status string) error {
	if status != "active" && status != "disabled" {
		return errors.New(errors.KindParam, "tenancy.invalid_status", "状态非法")
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
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return err
	}
	var platformUserID kernel.ID
	var isPlatformAdmin bool
	if err := tx.QueryRow(ctx,
		`SELECT m.platform_user_id, COALESCE(u.is_platform_admin, false)
		 FROM member m JOIN public.platform_user u ON u.id = m.platform_user_id
		 WHERE m.id = $1`,
		id).Scan(&platformUserID, &isPlatformAdmin); err != nil {
		if err == pgx.ErrNoRows {
			return errors.New(errors.KindNotFound, "tenancy.member_not_found", "成员不存在")
		}
		return err
	}
	if isPlatformAdmin {
		return errors.New(errors.KindForbidden, "tenancy.platform_admin_locked", "不能操作平台管理员账号")
	}
	res, err := tx.Exec(ctx, `UPDATE member SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "tenancy.member_not_found", "成员不存在")
	}
	if _, err := tx.Exec(ctx,
		`UPDATE public.tenant_membership SET status = $1
		 WHERE tenant_id = $2 AND member_id = $3 AND platform_user_id = $4`,
		status, t.ID, id, platformUserID); err != nil {
		return err
	}
	if status == "disabled" {
		if _, err := tx.Exec(ctx,
			`UPDATE public.session SET revoked = TRUE
			 WHERE tenant_id = $1 AND member_id = $2 AND revoked = FALSE`,
			t.ID, id); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "tenancy.member_status_changed", map[string]any{
		"tenant_id": t.ID, "member_id": id, "status": status,
	})
	return nil
}

// UpdateName updates tenant display name (admin only).
func (s *Service) UpdateName(ctx context.Context, id kernel.ID, name string) error {
	if name == "" {
		return errors.New(errors.KindParam, "tenancy.empty_name", "名称不能为空")
	}
	_, err := s.tenantRepo.(*pgTenantRepo).pool.Exec(ctx,
		`UPDATE public.tenant SET name = $1 WHERE id = $2`, name, id)
	return err
}

// CountMembers returns active member count for current tenant.
func (s *Service) CountMembers(ctx context.Context, pool *pgxpool.Pool, t *Tenant) (int, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %q, public", t.SchemaName)); err != nil {
		return 0, err
	}
	defer conn.Exec(ctx, "RESET search_path") //nolint:errcheck
	var n int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM member WHERE status = 'active'").Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

var _ pgx.Tx
