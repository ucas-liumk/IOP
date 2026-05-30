package iam

import (
	"context"

	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// PlatformPermission is one assignable platform permission point (resource/action).
type PlatformPermission struct {
	Resource   string `json:"resource"`
	Action     string `json:"action"`
	Domain     string `json:"domain"`
	Label      string `json:"label"`
	IsHighRisk bool   `json:"is_high_risk"`
}

const governanceModeKey = "governance_mode"

const (
	ModeSingleAdmin = "single_admin"
	ModeThreeMember = "three_member"
)

// GovernanceMode returns the configured platform governance mode, defaulting to
// single_admin when unset.
func (s *Service) GovernanceMode(ctx context.Context) string {
	m, err := s.repo.GetPlatformSetting(ctx, governanceModeKey)
	if err != nil || m == "" {
		return ModeSingleAdmin
	}
	return m
}

// SetGovernanceMode switches the governance mode. Authorization is enforced at the route layer.
func (s *Service) SetGovernanceMode(ctx context.Context, mode string, by kernel.ID) error {
	if mode != ModeSingleAdmin && mode != ModeThreeMember {
		return errors.New(errors.KindParam, "iam.invalid_mode", "治理模式无效")
	}
	return s.repo.SetPlatformSetting(ctx, governanceModeKey, mode, by)
}

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
// permits (resource, action). super_admin is all-access EXCEPT audit/purge is locked
// for everyone (including super_admin) under three_member mode.
//
// Authority model: is_platform_admin is a bootstrap signal only. SeedPlatformRBAC keeps
// every flagged user granted super_admin. Runtime authority is the role grant — to fully
// remove a super admin, clear the flag (so the seed won't re-grant) AND revoke the grant.
func (s *Service) EnforcePlatform(ctx context.Context, platformUserID kernel.ID, resource, action string) error {
	// Fix B: fail closed — if we cannot determine the governance mode, deny rather than
	// risk silently disabling the three_member separation-of-duties lock.
	mode, err := s.repo.GetPlatformSetting(ctx, governanceModeKey)
	if err != nil {
		return errors.New(errors.KindForbidden, "iam.governance_unavailable", "无法确定治理模式,已拒绝")
	}
	if mode == "" {
		mode = ModeSingleAdmin
	}
	if mode == ModeThreeMember && resource == "audit" && action == "purge" {
		return errors.New(errors.KindForbidden, "iam.audit_purge_locked", "三员模式下审计不可清除")
	}

	roles, err := s.repo.ListPlatformRolesForUser(ctx, platformUserID)
	if err != nil {
		return err
	}
	// Fix A: grants are authoritative; the is_platform_admin flag is only a bootstrap
	// signal — no flag bypass here.
	if len(roles) == 0 {
		return errors.New(errors.KindForbidden, "iam.no_platform_role", "无平台权限")
	}

	roleIDs := make([]kernel.ID, 0, len(roles))
	for _, r := range roles {
		if r.Code == "super_admin" {
			return nil
		}
		roleIDs = append(roleIDs, r.ID)
	}

	policies, err := s.repo.ListPolicyForRoles(ctx, roleIDs)
	if err != nil {
		return err
	}
	// Fix C: use matchPolicy (same package) instead of the removed permMatch.
	// platform policies use exact resource/action or "*"; matchPolicy also supports
	// "resource*" prefix wildcards which the catalog UI does not expose, so they are
	// benignly unused here.
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

// platformCatalog is the authoritative list of platform permission points.
var platformCatalog = []PlatformPermission{
	{"org", "read", "org_identity", "查看组织", false},
	{"org", "create", "org_identity", "新建组织", false},
	{"org", "update", "org_identity", "编辑组织", false},
	{"org", "suspend", "org_identity", "暂停组织", false},
	{"org", "close", "org_identity", "关闭组织", false},
	{"org", "hierarchy", "org_identity", "管理组织层级", false},
	{"user", "read", "org_identity", "查看用户", false},
	{"user", "create", "org_identity", "新建用户", false},
	{"user", "update", "org_identity", "编辑用户", false},
	{"user", "disable", "org_identity", "停用用户", false},
	{"user", "resetpwd", "org_identity", "重置密码", false},
	{"user", "impersonate", "org_identity", "模拟登录", true},
	{"membership", "assign", "org_identity", "分配组织归属", false},
	{"role", "manage", "security", "管理平台角色", false},
	{"authz", "grant", "security", "授予角色/权限", false},
	{"platform_admin", "grant", "security", "授予平台管理员资格", true},
	{"security_policy", "manage", "security", "安全策略", false},
	{"session", "read", "security", "查看会话", false},
	{"session", "revoke", "security", "强制下线", false},
	{"audit", "read", "audit", "查看审计", false},
	{"audit", "export", "audit", "导出审计", false},
	{"audit", "config", "audit", "审计配置", false},
	{"audit", "purge", "audit", "清除审计", true},
	{"login_log", "read", "audit", "登录日志", false},
	{"app", "manage", "app_config", "应用模块治理", false},
	{"param", "manage", "app_config", "全局参数", false},
	{"dict", "manage", "app_config", "数据字典", false},
	{"announce", "manage", "app_config", "公告通知", false},
	{"branding", "manage", "app_config", "品牌主题", false},
	{"monitor", "read", "monitoring", "监控查看", false},
	{"job", "manage", "monitoring", "定时任务", false},
	{"cache", "manage", "monitoring", "缓存管理", false},
	{"schema", "sync", "monitoring", "Schema 同步", false},
	{"codegen", "use", "monitoring", "代码生成器", false},
	{"backup", "manage", "monitoring", "备份导出", true},
}

// defaultRolePolicies maps each built-in 三员 role code to its default permission
// points. super_admin is intentionally absent (it is all-access via EnforcePlatform).
var defaultRolePolicies = map[string][][2]string{
	"sys_admin": {
		{"org", "read"}, {"org", "create"}, {"org", "update"}, {"org", "suspend"}, {"org", "close"}, {"org", "hierarchy"},
		{"user", "read"}, {"user", "create"}, {"user", "update"}, {"user", "disable"}, {"user", "resetpwd"}, {"user", "impersonate"},
		{"membership", "assign"},
		{"app", "manage"}, {"param", "manage"}, {"dict", "manage"}, {"announce", "manage"}, {"branding", "manage"},
		{"monitor", "read"}, {"job", "manage"}, {"cache", "manage"}, {"schema", "sync"}, {"codegen", "use"}, {"backup", "manage"},
	},
	"sec_admin": {
		{"org", "read"}, {"user", "read"},
		{"role", "manage"}, {"authz", "grant"}, {"platform_admin", "grant"},
		{"security_policy", "manage"}, {"session", "read"}, {"session", "revoke"},
	},
	"audit_admin": {
		{"org", "read"}, {"user", "read"}, {"session", "read"}, {"monitor", "read"},
		{"audit", "read"}, {"audit", "export"}, {"audit", "config"}, {"audit", "purge"}, {"login_log", "read"},
	},
}

// highRiskSet is derived from platformCatalog at package init for O(1) lookup.
var highRiskSet = func() map[[2]string]bool {
	m := map[[2]string]bool{}
	for _, p := range platformCatalog {
		if p.IsHighRisk {
			m[[2]string{p.Resource, p.Action}] = true
		}
	}
	return m
}()

// IsHighRiskPermission reports whether (resource, action) is flagged high-risk.
func (s *Service) IsHighRiskPermission(resource, action string) bool {
	return highRiskSet[[2]string{resource, action}]
}

// catalogSet provides O(1) membership tests against the platform permission catalog.
var catalogSet = func() map[[2]string]bool {
	m := map[[2]string]bool{}
	for _, p := range platformCatalog {
		m[[2]string{p.Resource, p.Action}] = true
	}
	return m
}()

// IsCatalogPermission reports whether (resource, action) is a known catalog point.
func (s *Service) IsCatalogPermission(resource, action string) bool {
	return catalogSet[[2]string{resource, action}]
}

// AddPlatformPolicy validates the point against the catalog before persisting,
// preventing injection of arbitrary/wildcard policies via the API.
func (s *Service) AddPlatformPolicy(ctx context.Context, roleID kernel.ID, resource, action string) error {
	if !s.IsCatalogPermission(resource, action) {
		return errors.New(errors.KindParam, "iam.unknown_permission", "未知权限点,不在目录内")
	}
	return s.repo.AddPlatformPolicy(ctx, roleID, resource, action)
}

// RoleWithPolicies is a platform role plus its policy rules (for the admin UI).
type RoleWithPolicies struct {
	*Role
	BuiltIn  bool          `json:"built_in"`
	Policies []*PolicyRule `json:"policies"`
	Members  []kernel.ID   `json:"members"`
}

var builtinPlatformRoleCodes = map[string]bool{
	"super_admin": true, "sys_admin": true, "sec_admin": true, "audit_admin": true,
}

// CreatePlatformRole creates a new custom platform role.
func (s *Service) CreatePlatformRole(ctx context.Context, code, name string) error {
	return s.repo.CreatePlatformRole(ctx, kernel.NewID(), code, name)
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
	return s.repo.DeletePlatformRole(ctx, id)
}

// ListPlatformRolesWithPolicies returns all platform roles with their policies and members.
func (s *Service) ListPlatformRolesWithPolicies(ctx context.Context) ([]*RoleWithPolicies, error) {
	roles, err := s.repo.ListPlatformRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*RoleWithPolicies, 0, len(roles))
	for _, role := range roles {
		pols, _ := s.repo.ListPolicyForRoles(ctx, []kernel.ID{role.ID})
		members, _ := s.repo.ListPlatformRoleMembers(ctx, role.ID)
		out = append(out, &RoleWithPolicies{
			Role: role, BuiltIn: builtinPlatformRoleCodes[role.Code], Policies: pols, Members: members,
		})
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

// SeedPlatformRBAC upserts the permission catalog and ensures the built-in 三员 roles
// carry their default policies. Idempotent; safe to run on every boot.
func (s *Service) SeedPlatformRBAC(ctx context.Context) error {
	for _, p := range platformCatalog {
		if err := s.repo.UpsertPlatformPermission(ctx, p); err != nil {
			return err
		}
	}
	for code, pols := range defaultRolePolicies {
		role, err := s.repo.GetPlatformRoleByCode(ctx, code)
		if err != nil {
			return err
		}
		if role == nil {
			continue
		}
		for _, ra := range pols {
			if err := s.repo.AddPlatformPolicy(ctx, role.ID, ra[0], ra[1]); err != nil {
				return err
			}
		}
	}
	// Fix A: idempotently ensure every is_platform_admin-flagged user has a super_admin grant.
	if err := s.repo.EnsureSuperAdminGrants(ctx); err != nil {
		return err
	}
	return nil
}
