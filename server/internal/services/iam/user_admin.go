package iam

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/services/tenancy"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// CreateUserByAdminCmd is the admin shortcut: skip the approval flow,
// create platform_user + join target tenant + grant role in one transaction-ish step.
type CreateUserByAdminCmd struct {
	Username       string
	RealName       string
	Phone          string
	Password       string
	OrganizationID kernel.ID // tenant the user belongs to
	Role           string    // tenant_member (default) or tenant_admin
}

// AdminCreateUser is platform-admin only — bypasses registration_application.
// Like ApproveApplication it is retry-safe: an orphan from a prior partial failure
// (user created, membership/role not yet granted) is reused instead of dead-ending
// on "username_taken".
func (s *Service) AdminCreateUser(ctx context.Context, pool *pgxpool.Pool, cmd CreateUserByAdminCmd) (*PlatformUser, error) {
	username := strings.TrimSpace(cmd.Username)
	if !usernameRe.MatchString(username) {
		return nil, errors.New(errors.KindParam, "iam.invalid_username",
			"用户名 3-32 位，仅含小写字母/数字/-/_，须以字母开头、字母或数字结尾")
	}
	realName := strings.TrimSpace(cmd.RealName)
	if realName == "" {
		return nil, errors.New(errors.KindParam, "iam.real_name_required", "请填写真实姓名")
	}
	if cmd.OrganizationID == "" {
		return nil, errors.New(errors.KindParam, "iam.organization_required", "请选择所在单位")
	}
	t, err := s.tenants.GetTenant(ctx, cmd.OrganizationID)
	if err != nil {
		return nil, err
	}
	if t == nil || t.Status != tenancy.StatusActive {
		return nil, errors.New(errors.KindParam, "iam.organization_not_found", "所选单位不存在或已停用")
	}
	role := strings.TrimSpace(cmd.Role)
	if role == "" {
		role = "tenant_member"
	}
	if role != "tenant_member" && role != "tenant_admin" {
		return nil, errors.New(errors.KindParam, "iam.invalid_role", "角色无效")
	}

	reuse, ok, err := s.reusableUser(ctx, username, t.ID)
	if err != nil {
		return nil, err
	}
	var u *PlatformUser
	if ok {
		u = reuse // orphan / half-provisioned from a prior attempt — continue idempotently
	} else {
		u, err = s.RegisterUser(ctx, RegisterCmd{
			Username: username, Phone: cmd.Phone, Password: cmd.Password,
		})
		if err != nil {
			return nil, err
		}
		// The admin chose the initial password, so force a change on first login.
		if err := s.repo.SetPasswordMustChange(ctx, u.ID, true); err != nil {
			_ = s.bus.Publish(ctx, "iam.set_must_change_failed", map[string]any{"platform_user_id": u.ID})
		}
	}
	mem, err := s.tenants.JoinMember(ctx, pool, tenancy.JoinMemberCmd{
		PlatformUserID: u.ID, TenantID: t.ID,
		DisplayName: realName, Phone: u.Phone,
	})
	if err != nil {
		return nil, err
	}
	if err := s.GrantRoleByCode(ctx, mem.MemberID, t.ID, role); err != nil {
		return nil, err
	}
	_ = s.bus.Publish(ctx, "iam.user_created_by_admin", map[string]any{
		"platform_user_id": u.ID, "tenant_id": t.ID, "role": role,
	})
	return u, nil
}

// ListUsers returns platform_users (platform admin only) matching search across username/email/phone.
func (s *Service) ListUsers(ctx context.Context, search string, limit int) ([]*PlatformUser, error) {
	return s.repo.ListUsers(ctx, search, limit)
}

// ListUsersPage returns one page of platform_users (platform admin only) matching
// search, plus the total count of matching rows, for server-side pagination.
func (s *Service) ListUsersPage(ctx context.Context, search string, page kernel.Pagination) (*kernel.Page[*PlatformUser], error) {
	p := page.Normalize()
	users, total, err := s.repo.ListUsersPage(ctx, search, p.PageSize, p.Offset())
	if err != nil {
		return nil, err
	}
	return &kernel.Page[*PlatformUser]{Data: users, Total: total, Page: p.Page, PageSize: p.PageSize}, nil
}

// GetUser returns the user by ID, or nil if not found.
func (s *Service) GetUser(ctx context.Context, id kernel.ID) (*PlatformUser, error) {
	return s.repo.GetUserByID(ctx, id)
}

// SetUserStatus toggles active/disabled on a platform_user. Disabling also revokes
// all of the user's active sessions immediately (DB flag + Redis blacklist), so
// access is cut right away rather than lingering until token expiry.
func (s *Service) SetUserStatus(ctx context.Context, id kernel.ID, status string) error {
	if status != "active" && status != "disabled" {
		return errors.New(errors.KindParam, "iam.invalid_status", "状态无效")
	}
	if err := s.repo.UpdateUserStatus(ctx, id, status); err != nil {
		return errors.Wrap(errors.KindDatabase, "iam.update_user_status_failed", "更新失败", err)
	}
	if status == "disabled" {
		if err := s.RevokeAllSessions(ctx, id); err != nil {
			// Status is already flipped; log-and-continue (future logins blocked anyway).
			_ = s.bus.Publish(ctx, "iam.session_revoke_failed", map[string]any{"platform_user_id": id})
		}
	}
	_ = s.bus.Publish(ctx, "iam.user_status_changed", map[string]any{
		"platform_user_id": id, "status": status,
	})
	return nil
}

// AdminResetPassword sets a new password chosen by the admin. Subject to the same strength rules.
func (s *Service) AdminResetPassword(ctx context.Context, id kernel.ID, newPassword string) error {
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateUserPassword(ctx, id, hash); err != nil {
		return errors.Wrap(errors.KindDatabase, "iam.update_password_failed", "更新失败", err)
	}
	// Admin now knows this password — force the user to change it on next login.
	if err := s.repo.SetPasswordMustChange(ctx, id, true); err != nil {
		_ = s.bus.Publish(ctx, "iam.set_must_change_failed", map[string]any{"platform_user_id": id})
	}
	_ = s.bus.Publish(ctx, "iam.user_password_reset_by_admin", map[string]any{"platform_user_id": id})
	return nil
}
