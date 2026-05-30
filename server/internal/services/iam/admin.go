package iam

import (
	"context"
	"fmt"

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

// Valid data_scope values (reserved; not yet enforced on business queries — see spec §9).
const (
	DataScopeAll        = "all"
	DataScopeDept       = "dept"
	DataScopeDeptAndSub = "dept_and_sub"
	DataScopeSelf       = "self"
	DataScopeCustom     = "custom"
)

func validDataScope(s string) bool {
	switch s {
	case DataScopeAll, DataScopeDept, DataScopeDeptAndSub, DataScopeSelf, DataScopeCustom:
		return true
	}
	return false
}

// RoleSummary is what the role list returns.
type RoleSummary struct {
	ID          kernel.ID    `json:"id"`
	TenantID    *kernel.ID   `json:"tenant_id,omitempty"`
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	BuiltIn     bool         `json:"built_in"`
	DataScope   string       `json:"data_scope"`
	DeptIDs     []kernel.ID  `json:"dept_ids,omitempty"`
	MemberCount int          `json:"member_count"`
	Policies    []PolicyRule `json:"policies"`
}

// ListRoles returns roles visible in the given tenant (NULL tenant_id = platform-wide).
func (s *Service) ListRoles(ctx context.Context, tenantID kernel.ID) ([]RoleSummary, error) {
	pool := s.repo.(*pgRepo).pool
	rows, err := pool.Query(ctx,
		`SELECT id, tenant_id, code, name, data_scope, is_builtin FROM public.role
		 WHERE tenant_id IS NULL OR tenant_id = $1
		 ORDER BY tenant_id NULLS FIRST, code`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RoleSummary{}
	roleIDs := []kernel.ID{}
	for rows.Next() {
		var r RoleSummary
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Code, &r.Name, &r.DataScope, &r.BuiltIn); err != nil {
			return nil, err
		}
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

// CreateRoleCmd creates a tenant-scoped role. DataScope defaults to "all"; DeptIDs
// is only persisted (into public.role_dept) when DataScope == "custom".
type CreateRoleCmd struct {
	TenantID  kernel.ID
	Code      string
	Name      string
	DataScope string
	DeptIDs   []kernel.ID
}

func (s *Service) CreateRole(ctx context.Context, cmd CreateRoleCmd) (*Role, error) {
	if cmd.Code == "" || cmd.Name == "" {
		return nil, errors.New(errors.KindParam, "iam.invalid_role", "code/name 必填")
	}
	scope := cmd.DataScope
	if scope == "" {
		scope = DataScopeAll
	}
	if !validDataScope(scope) {
		return nil, errors.New(errors.KindParam, "iam.invalid_data_scope", "数据范围非法")
	}
	r := &Role{
		ID: kernel.NewID(), TenantID: &cmd.TenantID,
		Code: cmd.Code, Name: cmd.Name, CreatedAt: s.clock.Now(),
	}
	pool := s.repo.(*pgRepo).pool
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx,
		`INSERT INTO public.role (id, tenant_id, code, name, data_scope, is_builtin, created_at)
		 VALUES ($1,$2,$3,$4,$5,FALSE,$6)`,
		r.ID, *r.TenantID, r.Code, r.Name, scope, r.CreatedAt); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "iam.create_role_failed", "创建角色失败", err)
	}
	if scope == DataScopeCustom {
		if err := replaceRoleDepts(ctx, tx, r.ID, cmd.TenantID, cmd.DeptIDs); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "iam.create_role_failed", "创建角色失败", err)
	}
	return r, nil
}

// UpdateRoleCmd patches a role. nil fields are left unchanged. Built-in roles may
// have name/data_scope/dept_ids changed but NOT their code.
type UpdateRoleCmd struct {
	TenantID  kernel.ID
	RoleID    kernel.ID
	Code      *string
	Name      *string
	DataScope *string
	DeptIDs   *[]kernel.ID // nil = leave bindings unchanged
}

func (s *Service) UpdateRole(ctx context.Context, cmd UpdateRoleCmd) error {
	pool := s.repo.(*pgRepo).pool

	// Load the role to learn whether it's built-in and its (possibly platform-wide) scope.
	var isBuiltin bool
	var roleTenant *kernel.ID
	var curScope string
	if err := pool.QueryRow(ctx,
		`SELECT is_builtin, tenant_id, data_scope FROM public.role
		 WHERE id = $1 AND (tenant_id IS NULL OR tenant_id = $2)`,
		cmd.RoleID, cmd.TenantID).Scan(&isBuiltin, &roleTenant, &curScope); err != nil {
		return errors.New(errors.KindNotFound, "iam.role_not_found", "角色不存在")
	}
	if cmd.Code != nil && isBuiltin {
		return errors.New(errors.KindForbidden, "iam.builtin_code_locked", "内置角色编码不可修改")
	}
	if cmd.DataScope != nil && !validDataScope(*cmd.DataScope) {
		return errors.New(errors.KindParam, "iam.invalid_data_scope", "数据范围非法")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	sets := []string{}
	args := []any{cmd.RoleID}
	idx := 2
	if cmd.Code != nil {
		sets = append(sets, fmt.Sprintf("code = $%d", idx))
		args = append(args, *cmd.Code)
		idx++
	}
	if cmd.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", idx))
		args = append(args, *cmd.Name)
		idx++
	}
	if cmd.DataScope != nil {
		sets = append(sets, fmt.Sprintf("data_scope = $%d", idx))
		args = append(args, *cmd.DataScope)
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
		sql += " WHERE id = $1"
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
	return tx.Commit(ctx)
}

// replaceRoleDepts rewrites the custom data-scope dept bindings for a role+tenant.
func replaceRoleDepts(ctx context.Context, tx pgx.Tx, roleID, tenantID kernel.ID, deptIDs []kernel.ID) error {
	if _, err := tx.Exec(ctx,
		`DELETE FROM public.role_dept WHERE role_id = $1 AND tenant_id = $2`, roleID, tenantID); err != nil {
		return err
	}
	for _, did := range deptIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO public.role_dept (role_id, tenant_id, dept_id) VALUES ($1,$2,$3)
			 ON CONFLICT (role_id, dept_id) DO NOTHING`, roleID, tenantID, did); err != nil {
			return errors.Wrap(errors.KindDatabase, "iam.role_dept_failed", "保存数据范围部门失败", err)
		}
	}
	return nil
}

// DeleteRole removes a custom role (built-in + cross-tenant rejected).
func (s *Service) DeleteRole(ctx context.Context, tenantID, roleID kernel.ID) error {
	pool := s.repo.(*pgRepo).pool
	var isBuiltin bool
	if err := pool.QueryRow(ctx,
		`SELECT is_builtin FROM public.role WHERE id = $1 AND tenant_id = $2`,
		roleID, tenantID).Scan(&isBuiltin); err != nil {
		// No such tenant-owned role (could be a built-in template with NULL tenant_id).
		return errors.New(errors.KindForbidden, "iam.cannot_delete_role", "内置角色不可删除")
	}
	if isBuiltin {
		return errors.New(errors.KindForbidden, "iam.cannot_delete_role", "内置角色不可删除")
	}
	res, err := pool.Exec(ctx,
		`DELETE FROM public.role WHERE id = $1 AND tenant_id = $2`, roleID, tenantID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New(errors.KindForbidden, "iam.cannot_delete_role", "内置角色不可删除")
	}
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
