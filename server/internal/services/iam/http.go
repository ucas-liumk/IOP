package iam

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/tenancy"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// RegisterRoutes wires /auth/* and /me/* routes.
func RegisterRoutes(r *gin.RouterGroup, svc *Service, tenants *tenancy.Service) {
	r.POST("/auth/login", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		tok, u, err := svc.Login(c.Request.Context(), LoginCmd{
			Email: req.Email, Password: req.Password,
			IP: c.ClientIP(), UA: c.Request.UserAgent(),
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"token": tok, "user": u})
	})

	r.POST("/auth/refresh", func(c *gin.Context) {
		var req struct{ RefreshToken string `json:"refresh_token" binding:"required"` }
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		tok, err := svc.Refresh(c.Request.Context(), req.RefreshToken)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"token": tok})
	})

	// Authenticated endpoints
	auth := r.Group("")
	auth.Use(JWTAuth(svc))

	auth.POST("/auth/logout", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		if err := svc.Logout(c.Request.Context(), claims.SessionID); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	auth.POST("/auth/switch-tenant", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		var req struct{ TenantID string `json:"tenant_id" binding:"required"` }
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		tid, err := kernel.ParseID(req.TenantID)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_tenant_id", "tenant_id 格式错误", err))
			return
		}
		tok, err := svc.SwitchTenant(c.Request.Context(), claims.SessionID, tid)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"token": tok})
	})

	auth.GET("/me", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		u, err := svc.repo.GetUserByID(c.Request.Context(), claims.PlatformUserID)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{
			"user":      u,
			"tenant_id": claims.TenantID,
			"member_id": claims.MemberID,
		})
	})

	auth.GET("/me/tenants", func(c *gin.Context) {
		claims, _ := ClaimsFromContext(c.Request.Context())
		ts, err := tenants.ListMyTenants(c.Request.Context(), claims.PlatformUserID)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"tenants": ts})
	})
}
