package iface

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/okr/application"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// RegisterRoutes wires OKR REST routes. Caller is expected to mount under
// authenticated + tenant-loaded group.
func RegisterRoutes(r *gin.RouterGroup, svc *application.Service) {
	// Plans -----------------------------------------------------------------
	r.GET("/plans", func(c *gin.Context) {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		level := c.Query("level")
		plans, err := svc.ListMyPlans(c.Request.Context(), claims.MemberID, level, p)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"plans": plans})
	})

	r.POST("/plans", func(c *gin.Context) {
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
			if err == nil {
				parent = &pid
			}
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

	r.GET("/plans/:id", func(c *gin.Context) {
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
		apiresp.OK(c, p)
	})

	r.POST("/plans/:id/items", func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
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

	r.PATCH("/plans/:id/items/:itemId/complete", func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		itemID, _ := kernel.ParseID(c.Param("itemId"))
		var req struct{ Note string `json:"note"` }
		_ = c.ShouldBindJSON(&req)
		if err := svc.CompleteItem(c.Request.Context(), application.CompleteItemCmd{
			PlanID: id, ItemID: itemID, Note: req.Note,
		}); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.POST("/plans/:id/close", func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		if err := svc.ClosePlan(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// Reports ---------------------------------------------------------------
	r.POST("/reports/daily", func(c *gin.Context) {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		var req struct {
			Day     string                `json:"day" binding:"required"` // yyyy-mm-dd
			Summary string                `json:"summary"`
			Entries []application.EntryInput `json:"entries"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "okr.invalid_request", "请求格式错误", err))
			return
		}
		day, _ := time.Parse("2006-01-02", req.Day)
		r2, err := svc.SubmitDaily(c.Request.Context(), application.SubmitDailyCmd{
			Owner: claims.MemberID, Day: day, Summary: req.Summary, Entries: req.Entries,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, r2)
	})

	r.POST("/reports/weekly", func(c *gin.Context) {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		var req struct {
			WeekContains string                `json:"week_contains" binding:"required"`
			Summary      string                `json:"summary"`
			Entries      []application.EntryInput `json:"entries"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "okr.invalid_request", "请求格式错误", err))
			return
		}
		when, _ := time.Parse("2006-01-02", req.WeekContains)
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

	r.GET("/reports", func(c *gin.Context) {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		var p kernel.Pagination
		_ = c.ShouldBindQuery(&p)
		typ := c.Query("type")
		rs, err := svc.ListReports(c.Request.Context(), claims.MemberID, typ, p)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"reports": rs})
	})

	r.GET("/reports/:id", func(c *gin.Context) {
		id, _ := kernel.ParseID(c.Param("id"))
		r2, err := svc.GetReport(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if r2 == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "okr.report.not_found", "报告不存在"))
			return
		}
		comments, _ := svc.ListReportComments(c.Request.Context(), id)
		apiresp.OK(c, gin.H{"report": r2, "comments": comments})
	})

	r.POST("/reports/:id/comments", func(c *gin.Context) {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		id, _ := kernel.ParseID(c.Param("id"))
		var req struct{ Body string `json:"body" binding:"required"` }
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
	r.GET("/rollups/weekly", func(c *gin.Context) {
		week := c.Query("week")
		var when time.Time
		if week == "" {
			when = time.Now().UTC()
		} else {
			when, _ = time.Parse("2006-01-02", week)
		}
		rows, err := svc.RollupWeekly(c.Request.Context(), when)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"rows": rows})
	})
}
