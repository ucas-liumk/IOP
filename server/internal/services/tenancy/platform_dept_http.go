package tenancy

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// orgTenantFromParam resolves the organization (tenant) named by the :tid path
// param. It writes a 4xx response and returns (nil,false) when the id is malformed
// or the org does not exist — callers should `return` on false.
//
// This is the bridge that lets the PLATFORM console (which carries NO tenant
// context of its own) operate on ANY organization: the caller passes the org's
// tenant id, we load the *Tenant (which holds the isolated SchemaName), and hand
// it straight to the existing dept service funcs — they SET LOCAL search_path to
// that schema, so all reads/writes stay scoped to the target org. No SQL is
// duplicated.
func orgTenantFromParam(c *gin.Context, svc *Service) (*Tenant, bool) {
	id, err := kernel.ParseID(c.Param("tid"))
	if err != nil {
		apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_org_id", "组织 ID 无效", err))
		return nil, false
	}
	t, err := svc.GetTenant(c.Request.Context(), id)
	if err != nil {
		apiresp.Fail(c, err)
		return nil, false
	}
	if t == nil {
		apiresp.Fail(c, errors.New(errors.KindNotFound, "tenancy.org_not_found", "组织不存在"))
		return nil, false
	}
	return t, true
}

// RegisterPlatformOrgRoutes mounts /platform/orgs/:tid/depts/* on the platform
// group. The platform group has JWTAuth + PasswordChangeGate + PlatformAccess but
// NO TenantLoader, so each handler resolves the org from :tid and injects its
// schema via the resolved *Tenant.
//
// authz is the platform RBAC gate (resource, action) supplied by the caller
// (app wiring). It is injected rather than imported so tenancy needn't depend on
// iam (iam already imports tenancy — importing back would create a cycle).
//
// Routes reuse the SAME dept service functions used by the tenant console
// (ListDepts/DeptTree/CreateDept/UpdateDept/DeleteDept/MoveDept/ExportDepts/
// DeptTemplateRows/ImportDepts) — only the tenant resolution differs.
func RegisterPlatformOrgRoutes(r *gin.RouterGroup, svc *Service, pool *pgxpool.Pool, authz AuthzFunc) {
	// CSV export / template / import. Registered before the parameterized :id
	// routes so the literal sub-paths take precedence (gin's tree disallows a
	// param segment colliding with a static one at the same position).
	r.GET("/platform/orgs/:tid/depts/export", authz("org", "read"), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		rows, err := svc.ExportDepts(c.Request.Context(), pool, t)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.CSV(c, "departments.csv", rows)
	})

	r.GET("/platform/orgs/:tid/depts/template", authz("org", "read"), func(c *gin.Context) {
		// Resolve the org so a bad/missing :tid still 404s consistently, even
		// though the template content itself is org-independent.
		if _, ok := orgTenantFromParam(c, svc); !ok {
			return
		}
		apiresp.CSV(c, "departments_template.csv", DeptTemplateRows())
	})

	r.POST("/platform/orgs/:tid/depts/import", authz("org", "write"), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		records, err := apiresp.ParseCSVUpload(c, "file")
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		res, err := svc.ImportDepts(c.Request.Context(), pool, t, records)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, res)
	})

	r.GET("/platform/orgs/:tid/depts", authz("org", "read"), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		depts, err := svc.ListDepts(c.Request.Context(), pool, t)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"depts": depts})
	})

	r.GET("/platform/orgs/:tid/depts/tree", authz("org", "read"), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		tree, err := svc.DeptTree(c.Request.Context(), pool, t)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"tree": tree})
	})

	r.POST("/platform/orgs/:tid/depts", authz("org", "write"), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		var req struct {
			Name     string  `json:"name" binding:"required"`
			ParentID *string `json:"parent_id"`
			OrderNum int     `json:"order_num"`
			Leader   string  `json:"leader"`
			Phone    string  `json:"phone"`
			Email    string  `json:"email"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		cmd := CreateDeptCmd{
			Name: req.Name, OrderNum: req.OrderNum,
			Leader: req.Leader, Phone: req.Phone, Email: req.Email,
		}
		if req.ParentID != nil && *req.ParentID != "" {
			pid, err := kernel.ParseID(*req.ParentID)
			if err != nil {
				apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_parent_id", "parent_id 无效", err))
				return
			}
			cmd.ParentID = &pid
		}
		dept, err := svc.CreateDept(c.Request.Context(), pool, t, cmd)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, dept)
	})

	r.PATCH("/platform/orgs/:tid/depts/:id", authz("org", "write"), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "部门 ID 无效", err))
			return
		}
		var req struct {
			Name     *string `json:"name"`
			OrderNum *int    `json:"order_num"`
			Leader   *string `json:"leader"`
			Phone    *string `json:"phone"`
			Email    *string `json:"email"`
			Status   *string `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.UpdateDept(c.Request.Context(), pool, t, UpdateDeptCmd{
			DeptID: id, Name: req.Name, OrderNum: req.OrderNum,
			Leader: req.Leader, Phone: req.Phone, Email: req.Email, Status: req.Status,
		}); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/platform/orgs/:tid/depts/:id", authz("org", "write"), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "部门 ID 无效", err))
			return
		}
		if err := svc.DeleteDept(c.Request.Context(), pool, t, id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/platform/orgs/:tid/depts/:id/move", authz("org", "write"), func(c *gin.Context) {
		t, ok := orgTenantFromParam(c, svc)
		if !ok {
			return
		}
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_id", "部门 ID 无效", err))
			return
		}
		var req struct {
			ParentID *string `json:"parent_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_request", "请求格式错误", err))
			return
		}
		var newParent *kernel.ID
		if req.ParentID != nil && *req.ParentID != "" {
			pid, err := kernel.ParseID(*req.ParentID)
			if err != nil {
				apiresp.Fail(c, errors.Wrap(errors.KindParam, "tenancy.invalid_parent_id", "parent_id 无效", err))
				return
			}
			newParent = &pid
		}
		if err := svc.MoveDept(c.Request.Context(), pool, t, id, newParent); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}
