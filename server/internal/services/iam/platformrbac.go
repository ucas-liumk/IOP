package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// HasAnyPlatformRole reports whether the user can enter the platform console at all.
// Back-compat: the legacy is_platform_admin flag also counts.
func (s *Service) HasAnyPlatformRole(ctx context.Context, platformUserID kernel.ID) bool {
	roles, err := s.repo.ListPlatformRolesForUser(ctx, platformUserID)
	if err == nil && len(roles) > 0 {
		return true
	}
	return s.IsPlatformAdminUser(ctx, platformUserID)
}

// EnforcePlatform allows the action iff the user holds a platform role whose policy
// permits (resource, action). It mirrors Enforce: no governance read, no audit/purge
// lock, no code-level super_admin short-circuit. The built-in super_admin role carries
// an all-access '*'/'*' policy (seeded in migration 000010), so it passes the generic
// match below like any other role.
//
// Authority model: is_platform_admin is a bootstrap signal only. SeedPlatformRBAC keeps
// every flagged user granted super_admin. Runtime authority is the role grant — to fully
// remove a super admin, clear the flag (so the seed won't re-grant) AND revoke the grant.
func (s *Service) EnforcePlatform(ctx context.Context, platformUserID kernel.ID, resource, action string) error {
	roles, err := s.repo.ListPlatformRolesForUser(ctx, platformUserID)
	if err != nil {
		return err
	}
	if len(roles) == 0 {
		return errors.New(errors.KindForbidden, "iam.no_platform_role", "无平台权限")
	}
	roleIDs := make([]kernel.ID, 0, len(roles))
	for _, r := range roles {
		roleIDs = append(roleIDs, r.ID)
	}
	policies, err := s.repo.ListPolicyForRoles(ctx, roleIDs)
	if err != nil {
		return err
	}
	for _, p := range policies {
		if p.Effect == "allow" && matchPolicy(p.Resource, resource) && matchPolicy(p.Action, action) {
			return nil
		}
	}
	return errors.New(errors.KindForbidden, "iam.platform_forbidden", "无权访问")
}

// GrantPlatformRoleByCode is a convenience used by seeding and tests.
func (s *Service) GrantPlatformRoleByCode(ctx context.Context, userID kernel.ID, code string, by kernel.ID) error {
	role, err := s.repo.GetPlatformRoleByCode(ctx, code)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New(errors.KindNotFound, "iam.role_not_found", "平台角色不存在")
	}
	return s.repo.GrantPlatformRole(ctx, role.ID, userID, by)
}

// RoleWithPolicies is a platform role plus its policy rules (for the admin UI).
type RoleWithPolicies struct {
	*Role
	BuiltIn  bool          `json:"built_in"`
	Policies []*PolicyRule `json:"policies"`
	Members  []kernel.ID   `json:"members"`
}

var builtinPlatformRoleCodes = map[string]bool{
	"super_admin": true, "platform_admin": true, "sys_admin": true, "sec_admin": true, "audit_admin": true,
}

type CreatePlatformRoleCmd struct {
	Code     string
	Name     string
	Status   string
	OrderNum int
	Remark   string
}

type UpdatePlatformRoleCmd struct {
	Code     *string
	Name     *string
	Status   *string
	OrderNum *int
	Remark   *string
}

// CreatePlatformRole creates a new custom platform role.
func (s *Service) CreatePlatformRole(ctx context.Context, cmd CreatePlatformRoleCmd) (*Role, error) {
	code := normalizeRoleCode(cmd.Code)
	name := strings.TrimSpace(cmd.Name)
	if code == "" || name == "" {
		return nil, errors.New(errors.KindParam, "iam.invalid_role", "code/name 必填")
	}
	if !roleCodeRe.MatchString(code) {
		return nil, errors.New(errors.KindParam, "iam.invalid_role_code", "角色编码需小写字母开头，可包含数字、-、_")
	}
	status := strings.TrimSpace(cmd.Status)
	if status == "" {
		status = RoleStatusActive
	}
	if !validRoleStatus(status) {
		return nil, errors.New(errors.KindParam, "iam.invalid_role_status", "角色状态只能是 active 或 disabled")
	}
	if exists, err := s.roleCodeExists(ctx, nil, code, ""); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New(errors.KindConflict, "iam.platform_role_code_taken", "平台角色编码已存在")
	}
	role := &Role{
		ID: kernel.NewID(), Code: code, Name: name, Status: status,
		OrderNum: cmd.OrderNum, Remark: strings.TrimSpace(cmd.Remark), CreatedAt: s.clock.Now(),
	}
	pool := s.repo.(*pgRepo).pool
	if _, err := pool.Exec(ctx,
		`INSERT INTO public.role (id, tenant_id, code, name, status, order_num, remark, is_builtin, created_at)
		 VALUES ($1, NULL, $2, $3, $4, $5, $6, FALSE, $7)`,
		role.ID, role.Code, role.Name, role.Status, role.OrderNum, role.Remark, role.CreatedAt); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "iam.create_role_failed", "创建平台角色失败", err)
	}
	return role, nil
}

func (s *Service) UpdatePlatformRole(ctx context.Context, id kernel.ID, cmd UpdatePlatformRoleCmd) error {
	role, err := s.repo.GetPlatformRoleByID(ctx, id)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New(errors.KindNotFound, "iam.role_not_found", "平台角色不存在")
	}
	if builtinPlatformRoleCodes[role.Code] {
		if cmd.Code != nil {
			return errors.New(errors.KindForbidden, "iam.builtin_code_locked", "内置角色编码不可修改")
		}
	}
	pool := s.repo.(*pgRepo).pool
	sets := []string{}
	args := []any{id}
	idx := 2
	if cmd.Code != nil {
		code := normalizeRoleCode(*cmd.Code)
		if !roleCodeRe.MatchString(code) {
			return errors.New(errors.KindParam, "iam.invalid_role_code", "角色编码需小写字母开头，可包含数字、-、_")
		}
		if exists, err := s.roleCodeExists(ctx, nil, code, id); err != nil {
			return err
		} else if exists {
			return errors.New(errors.KindConflict, "iam.platform_role_code_taken", "平台角色编码已存在")
		}
		sets = append(sets, fmt.Sprintf("code = $%d", idx))
		args = append(args, code)
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
	if cmd.Status != nil {
		if !validRoleStatus(*cmd.Status) {
			return errors.New(errors.KindParam, "iam.invalid_role_status", "角色状态只能是 active 或 disabled")
		}
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
		sets = append(sets, fmt.Sprintf("remark = $%d", idx))
		args = append(args, strings.TrimSpace(*cmd.Remark))
		idx++
	}
	if len(sets) == 0 {
		return nil
	}
	sql := "UPDATE public.role SET " + strings.Join(sets, ", ") +
		" WHERE id = $1 AND tenant_id IS NULL AND deleted_at IS NULL AND code NOT IN ('tenant_admin','tenant_member')"
	if _, err := pool.Exec(ctx, sql, args...); err != nil {
		return errors.Wrap(errors.KindDatabase, "iam.update_role_failed", "更新平台角色失败", err)
	}
	return nil
}

// DeletePlatformRole deletes a platform role by ID, rejecting built-in roles.
func (s *Service) DeletePlatformRole(ctx context.Context, id kernel.ID) error {
	role, err := s.repo.GetPlatformRoleByID(ctx, id)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New(errors.KindNotFound, "iam.role_not_found", "平台角色不存在")
	}
	if builtinPlatformRoleCodes[role.Code] {
		return errors.New(errors.KindForbidden, "iam.builtin_role_undeletable", "内置角色不可删除")
	}
	pool := s.repo.(*pgRepo).pool
	var grants int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM public.platform_role_grant WHERE role_id = $1`, id).Scan(&grants); err != nil {
		return err
	}
	if grants > 0 {
		return errors.New(errors.KindConflict, "iam.role_in_use", "角色已分配给用户，不能删除")
	}
	return s.repo.DeletePlatformRole(ctx, id)
}

// ListPlatformRolesWithPolicies returns all platform roles with their policies and members.
func (s *Service) ListPlatformRolesWithPolicies(ctx context.Context, filters ...RoleListFilter) ([]*RoleWithPolicies, error) {
	filter := RoleListFilter{}
	if len(filters) > 0 {
		filter = filters[0]
	}
	roles, err := s.repo.ListPlatformRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*RoleWithPolicies, 0, len(roles))
	for _, role := range roles {
		if !roleMatchesFilter(role, filter) {
			continue
		}
		pols, _ := s.repo.ListPolicyForRoles(ctx, []kernel.ID{role.ID})
		members, _ := s.repo.ListPlatformRoleMembers(ctx, role.ID)
		out = append(out, &RoleWithPolicies{
			Role: role, BuiltIn: builtinPlatformRoleCodes[role.Code], Policies: pols, Members: members,
		})
	}
	return out, nil
}

func roleMatchesFilter(role *Role, filter RoleListFilter) bool {
	if role == nil {
		return false
	}
	if filter.RoleType != "" && filter.RoleType != "all" && filter.RoleType != roleTypeOf(role.TenantID, role.Code) {
		return false
	}
	if filter.Status != "" && filter.Status != "all" && role.Status != filter.Status {
		return false
	}
	if q := strings.ToLower(strings.TrimSpace(filter.Search)); q != "" {
		return strings.Contains(strings.ToLower(role.Code), q) || strings.Contains(strings.ToLower(role.Name), q)
	}
	return true
}

func (s *Service) ListAllRolesForPlatform(ctx context.Context, filter RoleListFilter) ([]RoleSummary, error) {
	if filter.Status != "" && filter.Status != "all" && !validRoleStatus(filter.Status) {
		return nil, errors.New(errors.KindParam, "iam.invalid_role_status", "角色状态非法")
	}
	if filter.RoleType != "" && filter.RoleType != "all" && filter.RoleType != "platform" && filter.RoleType != "tenant" {
		return nil, errors.New(errors.KindParam, "iam.invalid_role_type", "角色类型非法")
	}
	pool := s.repo.(*pgRepo).pool
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	idx := 1
	switch filter.RoleType {
	case "platform":
		where = append(where, "tenant_id IS NULL AND code NOT IN ('tenant_admin','tenant_member')")
	case "tenant":
		where = append(where, "(tenant_id IS NOT NULL OR code IN ('tenant_admin','tenant_member'))")
	}
	if filter.TenantID != nil {
		where = append(where, fmt.Sprintf("tenant_id = $%d", idx))
		args = append(args, *filter.TenantID)
		idx++
	}
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
	sql := `SELECT r.id, r.tenant_id, r.code, r.name, r.data_scope, r.is_builtin,
	               COALESCE(status,'active'), COALESCE(order_num,0), COALESCE(remark,''),
	               CASE WHEN tenant_id IS NULL AND code NOT IN ('tenant_admin','tenant_member')
	                    THEN (SELECT count(*) FROM public.platform_role_grant pg WHERE pg.role_id = r.id)
	                    ELSE (SELECT count(*) FROM public.role_grant rg WHERE rg.role_id = r.id)
	               END AS member_count
	        FROM public.role r
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
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Code, &r.Name, &r.DataScope, &r.BuiltIn, &r.Status, &r.OrderNum, &r.Remark, &r.MemberCount); err != nil {
			return nil, err
		}
		r.RoleType = roleTypeOf(r.TenantID, r.Code)
		out = append(out, r)
		roleIDs = append(roleIDs, r.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	policies, err := s.repo.ListPolicyForRoles(ctx, roleIDs)
	if err != nil {
		return nil, err
	}
	byRole := map[kernel.ID][]PolicyRule{}
	for _, p := range policies {
		byRole[p.RoleID] = append(byRole[p.RoleID], *p)
	}
	for i := range out {
		out[i].Policies = byRole[out[i].ID]
	}
	return out, nil
}

// PlatformPermissionsForUser returns the flat set of "resource/action" the user holds
// (or ["*/*"] for super_admin), for front-end gating.
func (s *Service) PlatformPermissionsForUser(ctx context.Context, userID kernel.ID) ([]string, error) {
	roles, err := s.repo.ListPlatformRolesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	roleIDs := make([]kernel.ID, 0, len(roles))
	for _, r := range roles {
		if r.Code == "super_admin" {
			return []string{"*/*"}, nil
		}
		roleIDs = append(roleIDs, r.ID)
	}
	pols, err := s.repo.ListPolicyForRoles(ctx, roleIDs)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, p := range pols {
		if p.Effect == "allow" {
			out = append(out, p.Resource+"/"+p.Action)
		}
	}
	return out, nil
}

// PlatformPolicies returns the user's effective platform policy rules
// (ListPlatformRolesForUser → ListPolicyForRoles), de-referenced to values, for
// in-memory menu filtering via PermitsRule. The built-in super_admin role carries
// an all-access '*'/'*' policy (seeded), so a super admin's rules trivially permit
// everything. Returns an empty slice (no error) when the user has no platform role.
func (s *Service) PlatformPolicies(ctx context.Context, platformUserID kernel.ID) ([]PolicyRule, error) {
	roles, err := s.repo.ListPlatformRolesForUser(ctx, platformUserID)
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return []PolicyRule{}, nil
	}
	roleIDs := make([]kernel.ID, 0, len(roles))
	for _, r := range roles {
		roleIDs = append(roleIDs, r.ID)
	}
	pols, err := s.repo.ListPolicyForRoles(ctx, roleIDs)
	if err != nil {
		return nil, err
	}
	out := make([]PolicyRule, 0, len(pols))
	for _, p := range pols {
		if p != nil {
			out = append(out, *p)
		}
	}
	return out, nil
}

// UserHasPlatformRole reports whether the user holds a specific platform role by code.
func (s *Service) UserHasPlatformRole(ctx context.Context, userID kernel.ID, code string) bool {
	roles, err := s.repo.ListPlatformRolesForUser(ctx, userID)
	if err != nil {
		return false
	}
	for _, r := range roles {
		if r.Code == code {
			return true
		}
	}
	return false
}

// SeedPlatformRBAC ensures the built-in super_admin platform role is all-access and
// every is_platform_admin-flagged user holds it. Idempotent; safe to run on every boot.
//
// The super_admin role itself is created by migration 000009 and its '*'/'*' policy by
// migration 000010; this re-asserts the policy in code so a fresh boot is self-healing.
func (s *Service) SeedPlatformRBAC(ctx context.Context) error {
	role, err := s.repo.GetPlatformRoleByCode(ctx, "super_admin")
	if err != nil {
		return err
	}
	if role != nil {
		if err := s.repo.AddPlatformPolicy(ctx, role.ID, "*", "*"); err != nil {
			return err
		}
	}
	// Idempotently ensure every is_platform_admin-flagged user has a super_admin grant.
	return s.repo.EnsureSuperAdminGrants(ctx)
}
