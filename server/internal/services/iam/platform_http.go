package iam

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/services/audit"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// PlatformAccess gates the platform console: any platform role (or legacy
// is_platform_admin) may enter. Replaces PlatformAdminRequired on the platform group.
func PlatformAccess(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c.Request.Context())
		if !ok || claims.PlatformUserID == "" {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.platform_access_required", "请使用平台账号"))
			return
		}
		if !svc.HasAnyPlatformRole(c.Request.Context(), claims.PlatformUserID) {
			apiresp.Fail(c, errors.New(errors.KindForbidden, "iam.platform_access_required", "无平台访问权限"))
			return
		}
		c.Next()
	}
}

// PlatformAuthz enforces a single (resource, action) on the platform side and, on
// successful non-GET requests, records a platform audit entry. Under three_member
// mode a high-risk point additionally requires a non-empty X-Reason header.
func PlatformAuthz(svc *Service, aud *audit.Service, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c.Request.Context())
		if !ok || claims.PlatformUserID == "" {
			apiresp.Fail(c, errors.New(errors.KindAuth, "iam.no_session", "未登录"))
			return
		}
		ctx := c.Request.Context()
		if err := svc.EnforcePlatform(ctx, claims.PlatformUserID, resource, action); err != nil {
			apiresp.Fail(c, err)
			return
		}

		mode := svc.GovernanceMode(ctx)
		reason := c.GetHeader("X-Reason")
		if mode == ModeThreeMember && svc.IsHighRiskPermission(resource, action) && reason == "" {
			apiresp.Fail(c, errors.New(errors.KindParam, "iam.reason_required", "高危操作需在 X-Reason 头中填写原因"))
			return
		}

		c.Next()

		// Record successful writes only: apiresp.Fail aborts the context, so a
		// handler error (4xx/5xx) leaves IsAborted() true and is not audited here.
		if c.Request.Method != "GET" && !c.IsAborted() {
			roles, _ := svc.repo.ListPlatformRolesForUser(ctx, claims.PlatformUserID)
			codes := make([]string, 0, len(roles))
			for _, r := range roles {
				codes = append(codes, r.Code)
			}
			detail, _ := json.Marshal(gin.H{"path": c.Request.URL.Path, "status": c.Writer.Status()})
			aud.RecordPlatform(ctx, audit.PlatformEntry{
				Actor:          string(claims.PlatformUserID),
				ActorRole:      strings.Join(codes, ","),
				Action:         resource + "/" + action,
				Resource:       resource,
				ResourceID:     c.Param("id"),
				Reason:         reason,
				GovernanceMode: mode,
				TraceID:        kernel.TraceIDFromContext(ctx),
				Detail:         detail,
			})
		}
	}
}
