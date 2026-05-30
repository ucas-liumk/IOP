package iam

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/tenancy"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

// JWTAuth verifies the Authorization: Bearer <jwt> header and stores claims in ctx.
func JWTAuth(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			apiresp.Fail(c, errors.New(errors.KindAuth, "iam.missing_token", "缺少 Authorization 头"))
			return
		}
		tok := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		claims, err := svc.VerifyAccessToken(c.Request.Context(), tok)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		ctx := kernel.WithMemberID(c.Request.Context(), claims.MemberID)
		if claims.TenantID != "" {
			ctx = kernel.WithTenantID(ctx, claims.TenantID)
		}
		// Stash full claims for downstream handlers (SwitchTenant, /me).
		ctx = context.WithValue(ctx, claimsKey{}, claims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

type claimsKey struct{}

// ClaimsFromContext returns the verified Claims if JWTAuth ran.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey{}).(*Claims)
	return c, ok
}

// TenantLoader loads tenant metadata from the JWT claim and validates status.
func TenantLoader(tenants *tenancy.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		tid, ok := kernel.TenantIDFromContext(c.Request.Context())
		if !ok {
			apiresp.Fail(c, errors.New(errors.KindAuth, "iam.tenant_required", "请先切换租户"))
			return
		}
		t, err := tenants.GetTenant(c.Request.Context(), tid)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if t == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "iam.tenant_not_found", "租户不存在"))
			return
		}
		if t.Status != tenancy.StatusActive {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":     -1,
				"error":    gin.H{"code": "iam.tenant_inactive", "message": "租户不可用", "kind": "forbidden"},
				"trace_id": kernel.TraceIDFromContext(c.Request.Context()),
			})
			return
		}
		ctx := tenantdb.WithTenant(c.Request.Context(), &tenantdb.TenantContext{
			ID: string(t.ID), Slug: t.Slug, SchemaName: t.SchemaName, Status: t.Status,
		})
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// RBAC enforces (resource, action) on the current member+tenant.
// PasswordChangeGate blocks a user whose password_must_change flag is set from
// reaching any privileged/business endpoint until they change their password. It
// allowlists the personal (/api/me*) and auth (/api/auth/*) surfaces so the shell
// can load and POST /me/password / switch-tenant / logout still work. This is the
// authoritative server-side enforcement — the SPA guard is only UX.
func PasswordChangeGate(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		// Allowlist the auth + personal subtrees only. Delimiter-safe so a future
		// route like /api/members can't accidentally bypass the gate.
		if strings.HasPrefix(p, "/api/auth/") || p == "/api/me" || strings.HasPrefix(p, "/api/me/") {
			c.Next()
			return
		}
		claims, ok := ClaimsFromContext(c.Request.Context())
		if !ok || claims.PlatformUserID == "" {
			c.Next()
			return
		}
		u, err := svc.repo.GetUserByID(c.Request.Context(), claims.PlatformUserID)
		if err == nil && u != nil && u.PasswordMustChange {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.password_change_required",
				"请先修改初始密码后再使用"))
			return
		}
		c.Next()
	}
}

func RBAC(svc *Service, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c.Request.Context())
		if !ok {
			apiresp.Fail(c, errors.New(errors.KindAuth, "iam.no_session", "未登录"))
			return
		}
		if claims.MemberID == "" || claims.TenantID == "" {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.no_tenant", "请先选择租户"))
			return
		}
		if err := svc.Enforce(c.Request.Context(), claims.MemberID, claims.TenantID, resource, action); err != nil {
			apiresp.Fail(c, err)
			return
		}
		c.Next()
	}
}
