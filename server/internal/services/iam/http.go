package iam

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/tenancy"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// RegisterRoutes wires /auth/* and /me/* routes.
func RegisterRoutes(r *gin.RouterGroup, svc *Service, tenants *tenancy.Service, pool *pgxpool.Pool) {
	r.POST("/auth/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Email    string `json:"email"` // back-compat — older clients still send "email"
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		login := req.Username
		if login == "" {
			login = req.Email
		}
		if login == "" {
			apiresp.Fail(c, errors.New(errors.KindParam, "iam.invalid_request", "请输入用户名"))
			return
		}
		tok, u, err := svc.Login(c.Request.Context(), LoginCmd{
			Login: login, Password: req.Password,
			IP: c.ClientIP(), UA: c.Request.UserAgent(),
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"token": tok, "user": u})
	})

	r.POST("/auth/register", func(c *gin.Context) {
		var req struct {
			Username       string `json:"username" binding:"required"`
			RealName       string `json:"real_name" binding:"required"`
			OrganizationID string `json:"organization_id" binding:"required"`
			Phone          string `json:"phone"`
			Password       string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_request", "请求格式错误", err))
			return
		}
		orgID, err := kernel.ParseID(req.OrganizationID)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "iam.invalid_organization_id", "organization_id 无效", err))
			return
		}
		app, err := svc.SubmitApplication(c.Request.Context(), SubmitApplicationCmd{
			Username:       req.Username,
			RealName:       req.RealName,
			OrganizationID: orgID,
			Phone:          req.Phone,
			Password:       req.Password,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{
			"application_id": app.ID,
			"status":         app.Status,
			"organization":   app.Organization,
			"message":        "申请已提交，等待管理员审批",
		})
	})

	r.POST("/auth/refresh", func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}
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
		var req struct {
			TenantID string `json:"tenant_id" binding:"required"`
		}
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
