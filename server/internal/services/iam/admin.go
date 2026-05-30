package iam

import (
	"context"

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

// RoleSummary is what the role list returns.
type RoleSummary struct {
	ID          kernel.ID    `json:"id"`
	TenantID    *kernel.ID   `json:"tenant_id,omitempty"`
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	BuiltIn     bool         `json:"built_in"`
	MemberCount int          `json:"member_count"`
	Policies    []PolicyRule `json:"policies"`
}

// ListRoles returns roles visible in the given tenant (NULL tenant_id = platform-wide).
func (s *Service) ListRoles(ctx context.Context, tenantID kernel.ID) ([]RoleSummary, error) {
	pool := s.repo.(*pgRepo).pool
	rows, err := pool.Query(ctx,
		`SELECT id, tenant_id, code, name FROM public.role
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
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Code, &r.Name); err != nil {
			return nil, err
		}
		r.BuiltIn = r.TenantID == nil
		roleIDs = append(roleIDs, r.ID)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
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
	}
	return out, nil
}

// CreateRoleCmd creates a tenant-scoped role.
type CreateRoleCmd struct {
	TenantID kernel.ID
	Code     string
	Name     string
}

func (s *Service) CreateRole(ctx context.Context, cmd CreateRoleCmd) (*Role, error) {
	if cmd.Code == "" || cmd.Name == "" {
		return nil, errors.New(errors.KindParam, "iam.invalid_role", "code/name 必填")
	}
	r := &Role{
		ID: kernel.NewID(), TenantID: &cmd.TenantID,
		Code: cmd.Code, Name: cmd.Name, CreatedAt: s.clock.Now(),
	}
	pool := s.repo.(*pgRepo).pool
	_, err := pool.Exec(ctx,
		`INSERT INTO public.role (id, tenant_id, code, name, created_at) VALUES ($1,$2,$3,$4,$5)`,
		r.ID, *r.TenantID, r.Code, r.Name, r.CreatedAt)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "iam.create_role_failed", "创建角色失败", err)
	}
	return r, nil
}

// DeleteRole removes a custom role (built-in rejected).
func (s *Service) DeleteRole(ctx context.Context, tenantID, roleID kernel.ID) error {
	pool := s.repo.(*pgRepo).pool
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

var _ *pgxpool.Pool
var _ pgx.Tx
