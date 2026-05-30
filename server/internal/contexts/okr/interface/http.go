package iface

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/okr/application"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

// RegisterRoutes wires OKR REST routes. Caller is expected to mount under
// authenticated + tenant-loaded group. authz gates each route by the module's
// declared RBAC permissions (resource×action from the Manifest); when nil
// (e.g. in unit tests) routes are mounted ungated.
func RegisterRoutes(r *gin.RouterGroup, svc *application.Service, authz module.AuthzFunc, tdb *tenantdb.TenantDB, dataScope module.DataScopeFunc) {
	// gate returns the RBAC middleware for (resource, action), or a no-op when
	// authz is not wired (tests). Used as a per-route guard.
	gate := func(resource, action string) gin.HandlerFunc {
		if authz == nil {
			return func(c *gin.Context) { c.Next() }
		}
		return authz(resource, action)
	}
	const (
		planRes   = "okr.plan"
		reportRes = "okr.report"
		rollupRes = "okr.rollup"
	)

	type ownerScope struct {
		all    bool
		owners []kernel.ID
		set    map[kernel.ID]struct{}
	}
	newOwnerScope := func(owners []kernel.ID, all bool) ownerScope {
		set := make(map[kernel.ID]struct{}, len(owners))
		unique := make([]kernel.ID, 0, len(owners))
		for _, id := range owners {
			if id == "" {
				continue
			}
			if _, ok := set[id]; ok {
				continue
			}
			set[id] = struct{}{}
			unique = append(unique, id)
		}
		return ownerScope{all: all, owners: unique, set: set}
	}
	canAccess := func(scope ownerScope, owner kernel.ID) bool {
		if scope.all {
			return true
		}
		_, ok := scope.set[owner]
		return ok
	}
	scopeForbidden := func() error {
		return errors.New(errors.KindForbidden, "okr.scope_forbidden", "无权访问该 OKR 数据")
	}
	resolveScope := func(c *gin.Context) (ownerScope, error) {
		claims, ok := iam.ClaimsFromContext(c.Request.Context())
		if !ok || claims.MemberID == "" || claims.TenantID == "" {
			return ownerScope{}, errors.New(errors.KindAuth, "okr.no_session", "未登录")
		}
		if dataScope == nil || tdb == nil {
			return newOwnerScope([]kernel.ID{claims.MemberID}, false), nil
		}
		spec, err := dataScope(c.Request.Context(), claims.MemberID, claims.TenantID)
		if err != nil {
			return ownerScope{}, err
		}
		switch spec.Kind {
		case iam.DataScopeAll:
			return newOwnerScope(nil, true), nil
		case iam.DataScopeSelf:
			return newOwnerScope([]kernel.ID{claims.MemberID}, false), nil
		case iam.DataScopeDept:
			deptID, err := currentMemberDept(c.Request.Context(), tdb, claims.MemberID)
			if err != nil {
				return ownerScope{}, err
			}
			if deptID == nil {
				return newOwnerScope([]kernel.ID{claims.MemberID}, false), nil
			}
			owners, err := ownersInDepartments(c.Request.Context(), tdb, []kernel.ID{*deptID})
			if err != nil {
				return ownerScope{}, err
			}
			return newOwnerScope(owners, false), nil
		case iam.DataScopeDeptAndSub:
			deptID, err := currentMemberDept(c.Request.Context(), tdb, claims.MemberID)
			if err != nil {
				return ownerScope{}, err
			}
			if deptID == nil {
				return newOwnerScope([]kernel.ID{claims.MemberID}, false), nil
			}
			deptIDs, err := expandDepartments(c.Request.Context(), tdb, claims.TenantID, []kernel.ID{*deptID})
			if err != nil {
				return ownerScope{}, err
			}
			owners, err := ownersInDepartments(c.Request.Context(), tdb, deptIDs)
			if err != nil {
				return ownerScope{}, err
			}
			return newOwnerScope(owners, false), nil
		case iam.DataScopeCustom:
			owners, err := ownersInDepartments(c.Request.Context(), tdb, spec.DeptIDs)
			if err != nil {
				return ownerScope{}, err
			}
			return newOwnerScope(owners, false), nil
		default:
			return newOwnerScope([]kernel.ID{claims.MemberID}, false), nil
		}
	}

	// Plans -----------------------------------------------------------------
	r.GET("/plans", gate(planRes, "read"), func(c *gin.Context) {
		scope, err := resolveScope(c)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		level := c.Query("level")
		plans, err := svc.ListPlansScoped(c.Request.Context(), scope.owners, scope.all, level, p)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"plans": plans})
	})

	r.POST("/plans", gate(planRes, "write"), func(c *gin.Context) {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		var req struct {
			Level    string  `json:"level" binding:"required"`
			From     string  `json:"from"  binding:"required"`
			To       string  `json:"to"    binding:"required"`
			Title    string  `json:"title" binding:"required"`
			ParentID *string `json:"parent_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "okr.invalid_request", "请求格式错误", err))
			return
		}
		from, err := time.Parse("2006-01-02", req.From)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "okr.invalid_date", "日期格式错误", err))
			return
		}
		to, err := time.Parse("2006-01-02", req.To)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "okr.invalid_date", "日期格式错误", err))
			return
		}
		var parent *kernel.ID
		if req.ParentID != nil && *req.ParentID != "" {
			pid, err := kernel.ParseID(*req.ParentID)
			if err != nil {
				apiresp.Fail(c, errors.Wrap(errors.KindParam, "okr.invalid_parent_id", "parent_id 格式错误", err))
				return
			}
			parent = &pid
		}
		p, err := svc.CreatePlan(c.Request.Context(), application.CreatePlanCmd{
			Level: req.Level, Owner: claims.MemberID,
			From: from, To: to, Title: req.Title, ParentID: parent,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, p)
	})

	r.GET("/plans/:id", gate(planRes, "read"), func(c *gin.Context) {
		scope, err := resolveScope(c)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		p, err := svc.GetPlan(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if p == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "okr.plan.not_found", "计划不存在"))
			return
		}
		if !canAccess(scope, p.Owner) {
			apiresp.Fail(c, scopeForbidden())
			return
		}
		apiresp.OK(c, p)
	})

	r.POST("/plans/:id/items", gate(planRes, "write"), func(c *gin.Context) {
		scope, err := resolveScope(c)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		p, err := svc.GetPlan(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if p == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "okr.plan.not_found", "计划不存在"))
			return
		}
		if !canAccess(scope, p.Owner) {
			apiresp.Fail(c, scopeForbidden())
			return
		}
		var req struct {
			Title  string `json:"title" binding:"required"`
			Weight int    `json:"weight"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "okr.invalid_request", "请求格式错误", err))
			return
		}
		it, err := svc.AddPlanItem(c.Request.Context(), application.AddItemCmd{
			PlanID: id, Title: req.Title, Weight: req.Weight,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, it)
	})

	r.PATCH("/plans/:id/items/:itemId/complete", gate(planRes, "write"), func(c *gin.Context) {
		scope, err := resolveScope(c)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		itemID, err := kernel.ParseID(c.Param("itemId"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		p, err := svc.GetPlan(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if p == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "okr.plan.not_found", "计划不存在"))
			return
		}
		if !canAccess(scope, p.Owner) {
			apiresp.Fail(c, scopeForbidden())
			return
		}
		var req struct {
			Note string `json:"note"`
		}
		_ = c.ShouldBindJSON(&req)
		if err := svc.CompleteItem(c.Request.Context(), application.CompleteItemCmd{
			PlanID: id, ItemID: itemID, Note: req.Note,
		}); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/plans/:id/close", gate(planRes, "write"), func(c *gin.Context) {
		scope, err := resolveScope(c)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		p, err := svc.GetPlan(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if p == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "okr.plan.not_found", "计划不存在"))
			return
		}
		if !canAccess(scope, p.Owner) {
			apiresp.Fail(c, scopeForbidden())
			return
		}
		if err := svc.ClosePlan(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// Reports ---------------------------------------------------------------
	r.POST("/reports/daily", gate(reportRes, "write"), func(c *gin.Context) {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		var req struct {
			Day     string                   `json:"day" binding:"required"` // yyyy-mm-dd
			Summary string                   `json:"summary"`
			Entries []application.EntryInput `json:"entries"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "okr.invalid_request", "请求格式错误", err))
			return
		}
		day, err := time.Parse("2006-01-02", req.Day)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "okr.invalid_date", "日期格式错误", err))
			return
		}
		r2, err := svc.SubmitDaily(c.Request.Context(), application.SubmitDailyCmd{
			Owner: claims.MemberID, Day: day, Summary: req.Summary, Entries: req.Entries,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, r2)
	})

	r.POST("/reports/weekly", gate(reportRes, "write"), func(c *gin.Context) {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		var req struct {
			WeekContains string                   `json:"week_contains" binding:"required"`
			Summary      string                   `json:"summary"`
			Entries      []application.EntryInput `json:"entries"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "okr.invalid_request", "请求格式错误", err))
			return
		}
		when, err := time.Parse("2006-01-02", req.WeekContains)
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "okr.invalid_date", "日期格式错误", err))
			return
		}
		r2, err := svc.SubmitWeekly(c.Request.Context(), application.SubmitWeeklyCmd{
			Owner: claims.MemberID, WeekContains: when,
			Summary: req.Summary, Entries: req.Entries,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, r2)
	})

	r.GET("/reports", gate(reportRes, "read"), func(c *gin.Context) {
		scope, err := resolveScope(c)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		typ := c.Query("type")
		rs, err := svc.ListReportsScoped(c.Request.Context(), scope.owners, scope.all, typ, p)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"reports": rs})
	})

	r.GET("/reports/:id", gate(reportRes, "read"), func(c *gin.Context) {
		scope, err := resolveScope(c)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		r2, err := svc.GetReport(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if r2 == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "okr.report.not_found", "报告不存在"))
			return
		}
		if !canAccess(scope, r2.Owner) {
			apiresp.Fail(c, scopeForbidden())
			return
		}
		comments, _ := svc.ListReportComments(c.Request.Context(), id)
		apiresp.OK(c, gin.H{"report": r2, "comments": comments})
	})

	r.POST("/reports/:id/comments", gate(reportRes, "write"), func(c *gin.Context) {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		scope, err := resolveScope(c)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		r2, err := svc.GetReport(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if r2 == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "okr.report.not_found", "报告不存在"))
			return
		}
		if !canAccess(scope, r2.Owner) {
			apiresp.Fail(c, scopeForbidden())
			return
		}
		var req struct {
			Body string `json:"body" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "okr.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.CommentReport(c.Request.Context(), id, claims.MemberID, req.Body); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// Rollup ----------------------------------------------------------------
	r.GET("/rollups/weekly", gate(rollupRes, "read"), func(c *gin.Context) {
		scope, err := resolveScope(c)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		week := c.Query("week")
		var when time.Time
		if week == "" {
			when = time.Now().UTC()
		} else {
			var err error
			when, err = time.Parse("2006-01-02", week)
			if err != nil {
				apiresp.Fail(c, errors.Wrap(errors.KindParam, "okr.invalid_date", "日期格式错误", err))
				return
			}
		}
		rows, err := svc.RollupWeeklyScoped(c.Request.Context(), when, scope.owners, scope.all)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"rows": rows})
	})
}

func currentMemberDept(ctx context.Context, tdb *tenantdb.TenantDB, memberID kernel.ID) (*kernel.ID, error) {
	var out *kernel.ID
	err := tdb.Transaction(ctx, func(tx pgx.Tx) error {
		var deptID *kernel.ID
		err := tx.QueryRow(ctx, `SELECT dept_id FROM member WHERE id = $1`, memberID).Scan(&deptID)
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		out = deptID
		return nil
	})
	return out, err
}

func expandDepartments(ctx context.Context, tdb *tenantdb.TenantDB, tenantID kernel.ID, roots []kernel.ID) ([]kernel.ID, error) {
	if len(roots) == 0 {
		return nil, nil
	}
	out := []kernel.ID{}
	err := tdb.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`WITH RECURSIVE dept_tree AS (
			   SELECT id
			   FROM department
			   WHERE id = ANY($1::uuid[]) AND tenant_id = $2 AND deleted_at IS NULL AND status = 'active'
			   UNION ALL
			   SELECT d.id
			   FROM department d
			   JOIN dept_tree p ON d.parent_id = p.id
			   WHERE d.tenant_id = $2 AND d.deleted_at IS NULL AND d.status = 'active'
			 )
			 SELECT DISTINCT id FROM dept_tree`,
			idStrings(roots), tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id kernel.ID
			if err := rows.Scan(&id); err != nil {
				return err
			}
			out = append(out, id)
		}
		return rows.Err()
	})
	return out, err
}

func ownersInDepartments(ctx context.Context, tdb *tenantdb.TenantDB, deptIDs []kernel.ID) ([]kernel.ID, error) {
	if len(deptIDs) == 0 {
		return nil, nil
	}
	out := []kernel.ID{}
	err := tdb.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id FROM member WHERE dept_id = ANY($1::uuid[])`,
			idStrings(deptIDs))
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id kernel.ID
			if err := rows.Scan(&id); err != nil {
				return err
			}
			out = append(out, id)
		}
		return rows.Err()
	})
	return out, err
}

func idStrings(ids []kernel.ID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}
