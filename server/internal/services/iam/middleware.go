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
