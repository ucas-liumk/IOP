package iam

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

// ChangePasswordCmd applies for the authenticated user.
type ChangePasswordCmd struct {
	PlatformUserID kernel.ID
	Old            string
	New            string
}

func (s *Service) ChangePassword(ctx context.Context, cmd ChangePasswordCmd) error {
	u, err := s.repo.GetUserByID(ctx, cmd.PlatformUserID)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New(errors.KindNotFound, "iam.user_not_found", "用户不存在")
	}
	if err := CheckPassword(cmd.Old, u.PasswordHash); err != nil {
		return errors.New(errors.KindAuth, "iam.wrong_old_password", "原密码错误")
	}
	newHash, err := HashPassword(cmd.New)
	if err != nil {
		return err
	}
	pool := s.repo.(*pgRepo).pool
	_, err = pool.Exec(ctx,
		`UPDATE public.platform_user SET password_hash = $1, password_must_change = FALSE WHERE id = $2`,
		newHash, cmd.PlatformUserID)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "iam.change_password_failed", "改密码失败", err)
	}
	_ = s.bus.Publish(ctx, "iam.password_changed", map[string]any{
		"platform_user_id": cmd.PlatformUserID,
	})
	return nil
}

// Valid data_scope values. Business modules can resolve and enforce these scopes
// through module.DataScopeFunc.
const (
	DataScopeAll        = "all"
	DataScopeTenant     = "tenant"
	DataScopeDept       = "dept"
	DataScopeDeptAndSub = "dept_and_sub"
	DataScopeSelf       = "self"
	DataScopeCustom     = "custom"
)

func validDataScope(s string) bool {
	switch s {
	case DataScopeAll, DataScopeTenant, DataScopeDept, DataScopeDeptAndSub, DataScopeSelf, DataScopeCustom:
		return true
	}
	return false
}

const (
	RoleStatusActive   = "active"
	RoleStatusDisabled = "disabled"
)

var roleCodeRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,63}$`)

func validRoleStatus(s string) bool {
	return s == RoleStatusActive || s == RoleStatusDisabled
}

func normalizeRoleCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func roleTypeOf(tenantID *kernel.ID, code string) string {
	if tenantID == nil && code != "tenant_admin" && code != "tenant_member" {
		return "platform"
	}
	return "tenant"
}

// RoleListFilter is intentionally server-side; tenant-scoped routes still derive
// tenant_id from the token/context and never trust a caller-provided tenant_id.
type RoleListFilter struct {
	Search   string
	Status   string
	RoleType string
	TenantID *kernel.ID
}

// RoleSummary is what the role list returns.
type RoleSummary struct {
	ID          kernel.ID    `json:"id"`
	TenantID    *kernel.ID   `json:"tenant_id,omitempty"`
	RoleType    string       `json:"role_type"`
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	OrderNum    int          `json:"order_num"`
	Remark      string       `json:"remark,omitempty"`
	BuiltIn     bool         `json:"built_in"`
	DataScope   string       `json:"data_scope"`
	DeptIDs     []kernel.ID  `json:"dept_ids,omitempty"`
	MemberCount int          `json:"member_count"`
	Policies    []PolicyRule `json:"policies"`
}

// ListRoles returns roles visible/grantable inside the given tenant. Platform
// RBAC roles stay in /platform/rbac; tenant member grants may only use tenant
// roles plus the tenant built-ins.
func (s *Service) ListRoles(ctx context.Context, tenantID kernel.ID, filters ...RoleListFilter) ([]RoleSummary, error) {
	filter := RoleListFilter{}
	if len(filters) > 0 {
		filter = filters[0]
	}
	if filter.RoleType == "platform" {
		return []RoleSummary{}, nil
	}
	if filter.Status != "" && filter.Status != "all" && !validRoleStatus(filter.Status) {
		return nil, errors.New(errors.KindParam, "iam.invalid_role_status", "角色状态非法")
	}
	pool := s.repo.(*pgRepo).pool
	where := []string{"(tenant_id = $1 OR (tenant_id IS NULL AND code IN ('tenant_admin','tenant_member')))", "deleted_at IS NULL"}
	args := []any{tenantID}
	idx := 2
	if q := strings.TrimSpace(filter.Search); q != "" {
		where = append(where, fmt.Sprintf("(code ILIKE $%d OR name ILIKE $%d)", idx, idx))
		args = append(args, "%"+q+"%")
		idx++
	}
	if filter.Status != "" && filter.Status != "all" {
		where = append(where, fmt.Sprintf("COALESCE(status,'active') = $%d", idx))
		args = append(args, filter.Status)
		idx++
	}
	sql := `SELECT id, tenant_id, code, name, data_scope, is_builtin,
	               COALESCE(status,'active'), COALESCE(order_num,0), COALESCE(remark,'')
	        FROM public.role
	        WHERE ` + strings.Join(where, " AND ") + `
	        ORDER BY tenant_id NULLS FIRST, COALESCE(order_num,0), code`
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RoleSummary{}
	roleIDs := []kernel.ID{}
	for rows.Next() {
		var r RoleSummary
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Code, &r.Name, &r.DataScope, &r.BuiltIn, &r.Status, &r.OrderNum, &r.Remark); err != nil {
			return nil, err
		}
		r.RoleType = roleTypeOf(r.TenantID, r.Code)
		roleIDs = append(roleIDs, r.ID)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Custom data-scope dept bindings per role (scoped to this tenant).
	deptIDsByRole := map[kernel.ID][]kernel.ID{}
	if len(roleIDs) > 0 {
		drows, err := pool.Query(ctx,
			`SELECT role_id, dept_id FROM public.role_dept WHERE tenant_id = $1`, tenantID)
		if err != nil {
			return nil, err
		}
		defer drows.Close()
		for drows.Next() {
			var rid, did kernel.ID
			if err := drows.Scan(&rid, &did); err != nil {
				return nil, err
			}
			deptIDsByRole[rid] = append(deptIDsByRole[rid], did)
		}
		if err := drows.Err(); err != nil {
			return nil, err
		}
	}

	// Member counts per role (only counts within this tenant).
	mcRows, err := pool.Query(ctx,
		`SELECT role_id, count(*) FROM public.role_grant WHERE tenant_id = $1 GROUP BY role_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer mcRows.Close()
	counts := map[kernel.ID]int{}
	for mcRows.Next() {
		var rid kernel.ID
		var c int
		if err := mcRows.Scan(&rid, &c); err != nil {
			return nil, err
		}
		counts[rid] = c
	}

	// Policies per role
	policies, err := s.repo.ListPolicyForRoles(ctx, roleIDs)
	if err != nil {
		return nil, err
	}
	polByRole := map[kernel.ID][]PolicyRule{}
	for _, p := range policies {
		polByRole[p.RoleID] = append(polByRole[p.RoleID], *p)
	}
	for i := range out {
		out[i].MemberCount = counts[out[i].ID]
		out[i].Policies = polByRole[out[i].ID]
		if out[i].DataScope == DataScopeCustom {
			out[i].DeptIDs = deptIDsByRole[out[i].ID]
		}
	}
	return out, nil
}

// CreateRoleCmd creates a tenant-scoped role. DataScope defaults to "tenant"; DeptIDs
// is only persisted (into public.role_dept) when DataScope == "custom".
type CreateRoleCmd struct {
	TenantID  kernel.ID
	Code      string
	Name      string
	DataScope string
	DeptIDs   []kernel.ID
	Status    string
	OrderNum  int
	Remark    string
}

func (s *Service) CreateRole(ctx context.Context, cmd CreateRoleCmd) (*Role, error) {
	code := normalizeRoleCode(cmd.Code)
	name := strings.TrimSpace(cmd.Name)
	if code == "" || name == "" {
		return nil, errors.New(errors.KindParam, "iam.invalid_role", "code/name 必填")
	}
	if !roleCodeRe.MatchString(code) {
		return nil, errors.New(errors.KindParam, "iam.invalid_role_code", "角色编码需小写字母开头，可包含数字、-、_")
	}
	scope := cmd.DataScope
	if scope == "" {
		scope = DataScopeTenant
	}
	if !validDataScope(scope) {
		return nil, errors.New(errors.KindParam, "iam.invalid_data_scope", "数据范围非法")
	}
	status := strings.TrimSpace(cmd.Status)
	if status == "" {
		status = RoleStatusActive
	}
	if !validRoleStatus(status) {
		return nil, errors.New(errors.KindParam, "iam.invalid_role_status", "角色状态只能是 active 或 disabled")
	}
	if exists, err := s.roleCodeExists(ctx, &cmd.TenantID, code, ""); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New(errors.KindConflict, "iam.role_code_taken", "同一租户内角色编码已存在")
	}
	r := &Role{
		ID: kernel.NewID(), TenantID: &cmd.TenantID,
		Code: code, Name: name, Status: status, OrderNum: cmd.OrderNum,
		Remark: strings.TrimSpace(cmd.Remark), CreatedAt: s.clock.Now(),
	}
	pool := s.repo.(*pgRepo).pool
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx,
		`INSERT INTO public.role (id, tenant_id, code, name, data_scope, status, order_num, remark, is_builtin, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,FALSE,$9)`,
		r.ID, *r.TenantID, r.Code, r.Name, scope, status, r.OrderNum, r.Remark, r.CreatedAt); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "iam.create_role_failed", "创建角色失败", err)
	}
	if scope == DataScopeCustom {
		if err := s.validateTenantDeptIDsTx(ctx, tx, cmd.TenantID, cmd.DeptIDs); err != nil {
			return nil, err
		}
		if err := replaceRoleDepts(ctx, tx, r.ID, cmd.TenantID, cmd.DeptIDs); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "iam.create_role_failed", "创建角色失败", err)
	}
	_ = s.bus.Publish(ctx, "iam.role_created", map[string]any{
		"tenant_id": cmd.TenantID, "role_id": r.ID, "code": r.Code,
	})
	return r, nil
}

// UpdateRoleCmd patches a tenant-owned custom role. nil fields are left unchanged.
type UpdateRoleCmd struct {
	TenantID  kernel.ID
	RoleID    kernel.ID
	Code      *string
	Name      *string
	DataScope *string
	DeptIDs   *[]kernel.ID // nil = leave bindings unchanged
	Status    *string
	OrderNum  *int
	Remark    *string
}

func (s *Service) UpdateRole(ctx context.Context, cmd UpdateRoleCmd) error {
	pool := s.repo.(*pgRepo).pool

	// Tenant admins can only manage roles owned by their tenant. The shared
	// tenant_admin/tenant_member templates have tenant_id NULL and stay locked.
	var isBuiltin bool
	var curScope string
	if err := pool.QueryRow(ctx,
		`SELECT is_builtin, data_scope FROM public.role
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		cmd.RoleID, cmd.TenantID).Scan(&isBuiltin, &curScope); err != nil {
		return errors.New(errors.KindForbidden, "iam.role_scope_forbidden", "只能管理本租户自定义角色")
	}
	if isBuiltin {
		return errors.New(errors.KindForbidden, "iam.builtin_role_locked", "内置角色不可修改")
	}
	if cmd.Code != nil {
		code := normalizeRoleCode(*cmd.Code)
		if !roleCodeRe.MatchString(code) {
			return errors.New(errors.KindParam, "iam.invalid_role_code", "角色编码需小写字母开头，可包含数字、-、_")
		}
		if exists, err := s.roleCodeExists(ctx, &cmd.TenantID, code, cmd.RoleID); err != nil {
			return err
		} else if exists {
			return errors.New(errors.KindConflict, "iam.role_code_taken", "同一租户内角色编码已存在")
		}
		cmd.Code = &code
	}
	if cmd.DataScope != nil && !validDataScope(*cmd.DataScope) {
		return errors.New(errors.KindParam, "iam.invalid_data_scope", "数据范围非法")
	}
	if cmd.Status != nil && !validRoleStatus(*cmd.Status) {
		return errors.New(errors.KindParam, "iam.invalid_role_status", "角色状态只能是 active 或 disabled")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	sets := []string{}
	args := []any{cmd.RoleID, cmd.TenantID}
	idx := 3
	if cmd.Code != nil {
		sets = append(sets, fmt.Sprintf("code = $%d", idx))
		args = append(args, *cmd.Code)
		idx++
	}
	if cmd.Name != nil {
		name := strings.TrimSpace(*cmd.Name)
		if name == "" {
			return errors.New(errors.KindParam, "iam.invalid_role", "角色名称不能为空")
		}
		sets = append(sets, fmt.Sprintf("name = $%d", idx))
		args = append(args, name)
		idx++
	}
	if cmd.DataScope != nil {
		sets = append(sets, fmt.Sprintf("data_scope = $%d", idx))
		args = append(args, *cmd.DataScope)
		idx++
	}
	if cmd.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", idx))
		args = append(args, *cmd.Status)
		idx++
	}
	if cmd.OrderNum != nil {
		sets = append(sets, fmt.Sprintf("order_num = $%d", idx))
		args = append(args, *cmd.OrderNum)
		idx++
	}
	if cmd.Remark != nil {
		remark := strings.TrimSpace(*cmd.Remark)
		sets = append(sets, fmt.Sprintf("remark = $%d", idx))
		args = append(args, remark)
		idx++
	}
	if len(sets) > 0 {
		sql := "UPDATE public.role SET "
		for i, ss := range sets {
			if i > 0 {
				sql += ", "
			}
			sql += ss
		}
		sql += " WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL"
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return errors.Wrap(errors.KindDatabase, "iam.update_role_failed", "更新角色失败", err)
		}
	}

	// Resolve the effective scope after this update, to decide custom-dept handling.
	effScope := curScope
	if cmd.DataScope != nil {
		effScope = *cmd.DataScope
	}
	if effScope == DataScopeCustom {
		if cmd.DeptIDs != nil {
			if err := s.validateTenantDeptIDsTx(ctx, tx, cmd.TenantID, *cmd.DeptIDs); err != nil {
				return err
			}
			if err := replaceRoleDepts(ctx, tx, cmd.RoleID, cmd.TenantID, *cmd.DeptIDs); err != nil {
				return err
			}
		}
	} else if cmd.DataScope != nil {
		// Switched away from custom — drop any stale dept bindings for this tenant.
		if _, err := tx.Exec(ctx,
			`DELETE FROM public.role_dept WHERE role_id = $1 AND tenant_id = $2`,
			cmd.RoleID, cmd.TenantID); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "iam.role_updated", map[string]any{
		"tenant_id": cmd.TenantID, "role_id": cmd.RoleID,
	})
	return nil
}

// replaceRoleDepts rewrites the custom data-scope dept bindings for a role+tenant.
func replaceRoleDepts(ctx context.Context, tx pgx.Tx, roleID, tenantID kernel.ID, deptIDs []kernel.ID) error {
	if _, err := tx.Exec(ctx,
		`DELETE FROM public.role_dept WHERE role_id = $1 AND tenant_id = $2`, roleID, tenantID); err != nil {
		return err
	}
	seen := map[kernel.ID]bool{}
	for _, did := range deptIDs {
		if did == "" || seen[did] {
			continue
		}
		seen[did] = true
		if _, err := tx.Exec(ctx,
			`INSERT INTO public.role_dept (role_id, tenant_id, dept_id) VALUES ($1,$2,$3)
			 ON CONFLICT (role_id, dept_id) DO NOTHING`, roleID, tenantID, did); err != nil {
			return errors.Wrap(errors.KindDatabase, "iam.role_dept_failed", "保存数据范围部门失败", err)
		}
	}
	return nil
}

func (s *Service) validateTenantDeptIDsTx(ctx context.Context, tx pgx.Tx, tenantID kernel.ID, deptIDs []kernel.ID) error {
	unique := make([]string, 0, len(deptIDs))
	seen := map[kernel.ID]bool{}
	for _, id := range deptIDs {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, string(id))
	}
	if len(unique) == 0 {
		return errors.New(errors.KindParam, "iam.custom_scope_dept_required", "自定义数据范围至少选择一个组织")
	}
	t, err := s.tenants.GetTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	if t == nil {
		return errors.New(errors.KindNotFound, "iam.tenant_not_found", "租户不存在")
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return errors.Wrap(errors.KindDatabase, "iam.set_search_path_failed", "切换租户组织库失败", err)
	}
	var n int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM department
		 WHERE id = ANY($1::uuid[]) AND tenant_id = $2 AND deleted_at IS NULL`,
		unique, tenantID).Scan(&n); err != nil {
		return err
	}
	if n != len(unique) {
		return errors.New(errors.KindForbidden, "iam.cross_tenant_dept_scope", "自定义数据范围只能选择本租户组织")
	}
	return nil
}

func (s *Service) roleCodeExists(ctx context.Context, tenantID *kernel.ID, code string, exclude kernel.ID) (bool, error) {
	pool := s.repo.(*pgRepo).pool
	var n int
	if tenantID == nil {
		err := pool.QueryRow(ctx,
			`SELECT count(*) FROM public.role
			 WHERE tenant_id IS NULL AND lower(code) = lower($1) AND deleted_at IS NULL
			   AND ($2::uuid IS NULL OR id <> $2)`,
			code, idOrNilV(exclude)).Scan(&n)
		return n > 0, err
	}
	err := pool.QueryRow(ctx,
		`SELECT count(*) FROM public.role
		 WHERE tenant_id = $1 AND lower(code) = lower($2) AND deleted_at IS NULL
		   AND ($3::uuid IS NULL OR id <> $3)`,
		*tenantID, code, idOrNilV(exclude)).Scan(&n)
	return n > 0, err
}

// DeleteRole removes a custom role (built-in + cross-tenant rejected).
func (s *Service) DeleteRole(ctx context.Context, tenantID, roleID kernel.ID) error {
	pool := s.repo.(*pgRepo).pool
	var isBuiltin bool
	if err := pool.QueryRow(ctx,
		`SELECT is_builtin FROM public.role WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		roleID, tenantID).Scan(&isBuiltin); err != nil {
		// No such tenant-owned role (could be a built-in template with NULL tenant_id).
		return errors.New(errors.KindForbidden, "iam.cannot_delete_role", "内置角色不可删除")
	}
	if isBuiltin {
		return errors.New(errors.KindForbidden, "iam.cannot_delete_role", "内置角色不可删除")
	}
	var grants int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM public.role_grant WHERE role_id = $1 AND tenant_id = $2`,
		roleID, tenantID).Scan(&grants); err != nil {
		return err
	}
	if grants > 0 {
		return errors.New(errors.KindConflict, "iam.role_in_use", "角色已分配给用户，不能删除")
	}
	res, err := pool.Exec(ctx,
		`UPDATE public.role
		 SET deleted_at = now(), status = 'disabled'
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`, roleID, tenantID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindForbidden, "iam.cannot_delete_role", "内置角色不可删除")
	}
	_ = s.bus.Publish(ctx, "iam.role_deleted", map[string]any{
		"tenant_id": tenantID, "role_id": roleID,
	})
	return nil
}

// SeedRolePermissions idempotently grants a set of (resource, action) rules to a
// built-in (tenant-less) role by code. Used at boot to give the default
// tenant_member role read/write access to every registered module's resources,
// so members can actually use apps their tenant enabled while RBAC stays enforced
// (admins can still tighten this with custom roles). No-op if the role is absent.
func (s *Service) SeedRolePermissions(ctx context.Context, roleCode string, resourceActions [][2]string) error {
	role, err := s.repo.GetRoleByCode(ctx, roleCode, nil)
	if err != nil {
		return err
	}
	if role == nil {
		return nil
	}
	for _, ra := range resourceActions {
		if err := s.AddPolicy(ctx, role.ID, ra[0], ra[1]); err != nil {
			return err
		}
	}
	return nil
}

// AddPolicy attaches a (resource, action) rule to a role.
func (s *Service) AddPolicy(ctx context.Context, roleID kernel.ID, resource, action string) error {
	pool := s.repo.(*pgRepo).pool
	_, err := pool.Exec(ctx,
		`INSERT INTO public.role_policy (role_id, resource, action, effect)
		 VALUES ($1,$2,$3,'allow') ON CONFLICT DO NOTHING`,
		roleID, resource, action)
	return err
}

func (s *Service) RemovePolicy(ctx context.Context, roleID kernel.ID, resource, action string) error {
	pool := s.repo.(*pgRepo).pool
	_, err := pool.Exec(ctx,
		`DELETE FROM public.role_policy WHERE role_id = $1 AND resource = $2 AND action = $3`,
		roleID, resource, action)
	return err
}

func (s *Service) ensureTenantRoleEditable(ctx context.Context, tenantID, roleID kernel.ID) error {
	pool := s.repo.(*pgRepo).pool
	var isBuiltin bool
	if err := pool.QueryRow(ctx,
		`SELECT is_builtin FROM public.role
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		roleID, tenantID).Scan(&isBuiltin); err != nil {
		return errors.New(errors.KindForbidden, "iam.role_scope_forbidden", "只能管理本租户自定义角色")
	}
	if isBuiltin {
		return errors.New(errors.KindForbidden, "iam.builtin_role_locked", "内置角色不可修改")
	}
	return nil
}

func (s *Service) ensurePlatformRoleEditable(ctx context.Context, roleID kernel.ID) error {
	role, err := s.repo.GetPlatformRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New(errors.KindNotFound, "iam.role_not_found", "平台角色不存在")
	}
	if builtinPlatformRoleCodes[role.Code] {
		return errors.New(errors.KindForbidden, "iam.builtin_role_locked", "内置角色不可修改")
	}
	return nil
}

func (s *Service) AddTenantPolicy(ctx context.Context, tenantID, roleID kernel.ID, resource, action string) error {
	if resource == "" || action == "" {
		return errors.New(errors.KindParam, "iam.invalid_policy", "resource/action 必填")
	}
	if err := s.ensureTenantRoleEditable(ctx, tenantID, roleID); err != nil {
		return err
	}
	if err := s.AddPolicy(ctx, roleID, resource, action); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "iam.role_policy_updated", map[string]any{
		"tenant_id": tenantID, "role_id": roleID, "op": "add",
	})
	return nil
}

func (s *Service) RemoveTenantPolicy(ctx context.Context, tenantID, roleID kernel.ID, resource, action string) error {
	if resource == "" || action == "" {
		return errors.New(errors.KindParam, "iam.invalid_policy", "resource/action 必填")
	}
	if err := s.ensureTenantRoleEditable(ctx, tenantID, roleID); err != nil {
		return err
	}
	if err := s.RemovePolicy(ctx, roleID, resource, action); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "iam.role_policy_updated", map[string]any{
		"tenant_id": tenantID, "role_id": roleID, "op": "remove",
	})
	return nil
}

// PolicyChange is one (resource, action) entry in a batch policy update.
type PolicyChange struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// BatchPolicy applies a set of policy additions and removals to a tenant role in a
// single transaction (all-or-nothing). Adds are idempotent (ON CONFLICT DO
// NOTHING); removals are no-ops when the rule is absent. Empty resource/action
// entries are skipped.
func (s *Service) BatchPolicy(ctx context.Context, roleID kernel.ID, add, remove []PolicyChange) error {
	pool := s.repo.(*pgRepo).pool
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	for _, p := range remove {
		if p.Resource == "" || p.Action == "" {
			continue
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM public.role_policy WHERE role_id = $1 AND resource = $2 AND action = $3`,
			roleID, p.Resource, p.Action); err != nil {
			return errors.Wrap(errors.KindDatabase, "iam.batch_policy_failed", "批量更新权限失败", err)
		}
	}
	for _, p := range add {
		if p.Resource == "" || p.Action == "" {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO public.role_policy (role_id, resource, action, effect)
			 VALUES ($1,$2,$3,'allow') ON CONFLICT DO NOTHING`,
			roleID, p.Resource, p.Action); err != nil {
			return errors.Wrap(errors.KindDatabase, "iam.batch_policy_failed", "批量更新权限失败", err)
		}
	}
	return tx.Commit(ctx)
}

func (s *Service) BatchTenantPolicy(ctx context.Context, tenantID, roleID kernel.ID, add, remove []PolicyChange) error {
	if err := s.ensureTenantRoleEditable(ctx, tenantID, roleID); err != nil {
		return err
	}
	if err := s.BatchPolicy(ctx, roleID, add, remove); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "iam.role_policy_updated", map[string]any{
		"tenant_id": tenantID, "role_id": roleID,
		"add": len(add), "remove": len(remove),
	})
	return nil
}

func (s *Service) AddPlatformPolicy(ctx context.Context, roleID kernel.ID, resource, action string) error {
	if resource == "" || action == "" {
		return errors.New(errors.KindParam, "iam.invalid_policy", "resource/action 必填")
	}
	if err := s.ensurePlatformRoleEditable(ctx, roleID); err != nil {
		return err
	}
	return s.repo.AddPlatformPolicy(ctx, roleID, resource, action)
}

func (s *Service) RemovePlatformPolicy(ctx context.Context, roleID kernel.ID, resource, action string) error {
	if resource == "" || action == "" {
		return errors.New(errors.KindParam, "iam.invalid_policy", "resource/action 必填")
	}
	if err := s.ensurePlatformRoleEditable(ctx, roleID); err != nil {
		return err
	}
	return s.repo.RemovePlatformPolicy(ctx, roleID, resource, action)
}

// BatchPlatformPolicy applies a set of policy additions and removals to a
// platform role in a single transaction (all-or-nothing). Same semantics as
// BatchPolicy but against the platform role_policy rows (tenant_id IS NULL role).
func (s *Service) BatchPlatformPolicy(ctx context.Context, roleID kernel.ID, add, remove []PolicyChange) error {
	if err := s.ensurePlatformRoleEditable(ctx, roleID); err != nil {
		return err
	}
	pool := s.repo.(*pgRepo).pool
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	for _, p := range remove {
		if p.Resource == "" || p.Action == "" {
			continue
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM public.role_policy WHERE role_id = $1 AND resource = $2 AND action = $3`,
			roleID, p.Resource, p.Action); err != nil {
			return errors.Wrap(errors.KindDatabase, "iam.batch_policy_failed", "批量更新权限失败", err)
		}
	}
	for _, p := range add {
		if p.Resource == "" || p.Action == "" {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO public.role_policy (role_id, resource, action, effect)
			 VALUES ($1,$2,$3,'allow') ON CONFLICT (role_id, resource, action) DO NOTHING`,
			roleID, p.Resource, p.Action); err != nil {
			return errors.Wrap(errors.KindDatabase, "iam.batch_policy_failed", "批量更新权限失败", err)
		}
	}
	return tx.Commit(ctx)
}

// RevokeRole removes a (memberID, roleID, tenantID) grant.
func (s *Service) RevokeRole(ctx context.Context, memberID, tenantID, roleID kernel.ID) error {
	pool := s.repo.(*pgRepo).pool
	_, err := pool.Exec(ctx,
		`DELETE FROM public.role_grant WHERE role_id = $1 AND member_id = $2 AND tenant_id = $3`,
		roleID, memberID, tenantID)
	return err
}

// MemberRoles lists role grants for a member in a tenant.
func (s *Service) MemberRoles(ctx context.Context, memberID, tenantID kernel.ID) ([]*Role, error) {
	return s.repo.ListMemberRoles(ctx, memberID, tenantID)
}

// IsPlatformAdminUser is the authoritative, GLOBAL platform-admin check: it reads
// the is_platform_admin flag on the platform_user, independent of any tenant
// membership. This is what gates the platform console.
func (s *Service) IsPlatformAdminUser(ctx context.Context, platformUserID kernel.ID) bool {
	u, err := s.repo.GetUserByID(ctx, platformUserID)
	return err == nil && u != nil && u.IsPlatformAdmin
}

// IsTenantAdmin reports whether the member is an admin OF THIS TENANT (holds the
// tenant_admin role in this tenant). Global platform admins are NOT implicitly
// tenant admins — they govern at the platform layer, not inside tenant data.
func (s *Service) IsTenantAdmin(ctx context.Context, memberID, tenantID kernel.ID) bool {
	roles, _ := s.repo.ListMemberRoles(ctx, memberID, tenantID)
	for _, r := range roles {
		if r.Code == "tenant_admin" {
			return true
		}
	}
	return false
}

// ListSessions returns active sessions for a user (admin or self).
type SessionRow struct {
	ID        kernel.ID `json:"id"`
	IssuedAt  string    `json:"issued_at"`
	ExpiresAt string    `json:"expires_at"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	Current   bool      `json:"current"`
	Revoked   bool      `json:"revoked"`
}

func (s *Service) ListSessions(ctx context.Context, userID, currentSession kernel.ID) ([]SessionRow, error) {
	pool := s.repo.(*pgRepo).pool
	rows, err := pool.Query(ctx,
		`SELECT id, to_char(issued_at,'YYYY-MM-DD HH24:MI:SS'),
		        to_char(expires_at,'YYYY-MM-DD HH24:MI:SS'),
		        COALESCE(host(ip_address),''), COALESCE(user_agent,''), revoked
		 FROM public.session WHERE platform_user_id = $1
		 ORDER BY issued_at DESC LIMIT 20`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SessionRow{}
	for rows.Next() {
		var r SessionRow
		if err := rows.Scan(&r.ID, &r.IssuedAt, &r.ExpiresAt, &r.IPAddress, &r.UserAgent, &r.Revoked); err != nil {
			return nil, err
		}
		r.Current = r.ID == currentSession
		out = append(out, r)
	}
	return out, rows.Err()
}

// OnlineSessionRow is one active session for the online-users view.
type OnlineSessionRow struct {
	SessionID   kernel.ID `json:"session_id"`
	MemberID    kernel.ID `json:"member_id,omitempty"`
	DisplayName string    `json:"display_name"`
	IPAddress   string    `json:"ip_address,omitempty"`
	IssuedAt    string    `json:"issued_at"`
	ExpiresAt   string    `json:"expires_at"`
}

// ListOnlineSessions returns active (non-revoked, unexpired) sessions for the
// given tenant, with the member display name resolved from the tenant schema.
// schemaName is the tenant_<slug> schema for display-name resolution.
func (s *Service) ListOnlineSessions(ctx context.Context, tenantID kernel.ID, schemaName string) ([]OnlineSessionRow, error) {
	pool := s.repo.(*pgRepo).pool
	rows, err := pool.Query(ctx,
		`SELECT id, member_id,
		        to_char(issued_at,'YYYY-MM-DD HH24:MI:SS'),
		        to_char(expires_at,'YYYY-MM-DD HH24:MI:SS'),
		        COALESCE(host(ip_address),'')
		 FROM public.session
		 WHERE tenant_id = $1 AND revoked = FALSE AND expires_at > now()
		 ORDER BY issued_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OnlineSessionRow{}
	memberIDs := []kernel.ID{}
	for rows.Next() {
		var r OnlineSessionRow
		var mid *kernel.ID
		if err := rows.Scan(&r.SessionID, &mid, &r.IssuedAt, &r.ExpiresAt, &r.IPAddress); err != nil {
			return nil, err
		}
		if mid != nil {
			r.MemberID = *mid
			memberIDs = append(memberIDs, *mid)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Resolve member display names from the tenant schema in one query.
	if len(memberIDs) > 0 && schemaName != "" {
		conn, err := pool.Acquire(ctx)
		if err != nil {
			return nil, err
		}
		defer conn.Release()
		if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %q, public", schemaName)); err != nil {
			return nil, err
		}
		defer conn.Exec(ctx, "RESET search_path") //nolint:errcheck
		nameRows, err := conn.Query(ctx,
			`SELECT id, COALESCE(display_name,'') FROM member WHERE id = ANY($1)`, memberIDs)
		if err != nil {
			return nil, err
		}
		defer nameRows.Close()
		names := map[kernel.ID]string{}
		for nameRows.Next() {
			var id kernel.ID
			var name string
			if err := nameRows.Scan(&id, &name); err != nil {
				return nil, err
			}
			names[id] = name
		}
		if err := nameRows.Err(); err != nil {
			return nil, err
		}
		for i := range out {
			if n, ok := names[out[i].MemberID]; ok {
				out[i].DisplayName = n
			}
		}
	}
	return out, nil
}

// GetSessionTenant returns the tenant_id bound to a session (for verifying a
// kick target belongs to the acting admin's tenant). Returns ("", nil) when the
// session has no tenant binding or does not exist.
func (s *Service) GetSessionTenant(ctx context.Context, sessionID kernel.ID) (kernel.ID, error) {
	sess, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if sess == nil || sess.TenantID == nil {
		return "", nil
	}
	return *sess.TenantID, nil
}

var _ *pgxpool.Pool
var _ pgx.Tx
