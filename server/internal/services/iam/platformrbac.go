package iam

import (
	"context"

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
