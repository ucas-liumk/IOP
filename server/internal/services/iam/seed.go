package iam

import (
	"context"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/leo/iop/server/internal/services/tenancy"
)

// SeedDefaults provisions the built-in platform administrator on boot.
//
// platform_admin is a GLOBAL identity (the is_platform_admin flag on
// platform_user) — the admin is NOT a member of any tenant and there is no
// special "system" tenant. The admin governs the platform layer (organizations,
// global users, cross-tenant approvals); tenant internals are delegated to each
// tenant's own tenant_admin.
//
//	username: admin
//	password: Admin12345!   (override with IOP_SEED_ADMIN_PASSWORD; the default
//	                          forces a password change on first login)
//
// Idempotent across restarts and self-healing across partial failures.
// tenants/pool are accepted for call-site stability but no longer used.
func SeedDefaults(ctx context.Context, s *Service, tenants *tenancy.Service, pool *pgxpool.Pool, logger *zap.Logger) error {
	_ = tenants
	_ = pool
	const (
		username        = "admin"
		defaultPassword = "Admin12345!"
	)
	password := strings.TrimSpace(os.Getenv("IOP_SEED_ADMIN_PASSWORD"))
	usingDefaultPw := false
	if password == "" {
		password = defaultPassword
		usingDefaultPw = true
	}

	u, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return err
	}
	if u == nil {
		u, err = s.RegisterUser(ctx, RegisterCmd{Username: username, Password: password})
		if err != nil {
			return err
		}
		if usingDefaultPw {
			if err := s.repo.SetPasswordMustChange(ctx, u.ID, true); err != nil {
				logger.Warn("could not set password_must_change on seeded admin", zap.Error(err))
			}
		}
	}
	// Ensure the global platform-admin flag (idempotent; self-heals if a prior boot
	// created the user but crashed before flagging it).
	if !u.IsPlatformAdmin {
		if err := s.repo.SetPlatformAdmin(ctx, u.ID, true); err != nil {
			return err
		}
	}
	logger.Info("ensured default platform admin",
		zap.String("username", username),
		zap.String("user_id", string(u.ID)),
	)
	return nil
}
