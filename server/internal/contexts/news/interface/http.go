// Package iface wires the news (时政资讯) module's REST routes. Mounted under
// /api/apps/news/* by the module Registry; every route is RBAC-gated via the
// module.AuthzFunc passed in (resource×action from the Manifest).
package iface

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/news/application"
	"github.com/leo/iop/server/internal/contexts/news/domain"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
)

// Permission resources (namespaced by module code). Readers need read; editors
// (content management) need manage.
const (
	resRead   = "news.read"
	resManage = "news.manage"
)

func RegisterRoutes(r *gin.RouterGroup, svc *application.Service, authz module.AuthzFunc) {
	gate := func(resource, action string) gin.HandlerFunc {
		if authz == nil {
			return func(c *gin.Context) { c.Next() }
		}
		return authz(resource, action)
	}
	caller := func(c *gin.Context) kernel.ID {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		if claims == nil {
			return ""
		}
		return claims.MemberID
	}

	// ---- Categories (manage) ----
	r.GET("/categories", gate(resRead, "read"), func(c *gin.Context) {
		cats, err := svc.ListCategories(c.Request.Context())
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"categories": cats})
	})

	r.POST("/categories", gate(resManage, "write"), func(c *gin.Context) {
		var req struct {
			Name     string `json:"name" binding:"required"`
			OrderNum int    `json:"order_num"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "news.invalid_request", "请求格式错误", err))
			return
		}
		cat, err := svc.CreateCategory(c.Request.Context(), application.CreateCategoryCmd{
			Name: req.Name, OrderNum: req.OrderNum,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, cat)
	})

	r.PATCH("/categories/:id", gate(resManage, "write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "news.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			Name     string `json:"name" binding:"required"`
			OrderNum int    `json:"order_num"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "news.invalid_request", "请求格式错误", err))
			return
		}
		if err := svc.UpdateCategory(c.Request.Context(), application.UpdateCategoryCmd{
			ID: id, Name: req.Name, OrderNum: req.OrderNum,
		}); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/categories/:id", gate(resManage, "delete"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "news.invalid_id", "id 无效", err))
			return
		}
		if err := svc.DeleteCategory(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// ---- Articles (manage list / CRUD) ----
	r.GET("/articles", gate(resManage, "read"), func(c *gin.Context) {
		f := domain.Filter{
			Status:  c.Query("status"),
			Keyword: c.Query("keyword"),
		}
		f.CategoryID = optID(c.Query("category_id"))
		page, _ := svc.ListArticles(c.Request.Context(), f, pagination(c))
		apiresp.OK(c, page)
	})

	r.POST("/articles", gate(resManage, "write"), func(c *gin.Context) {
		var req struct {
			CategoryID string `json:"category_id"`
			Title      string `json:"title" binding:"required"`
			Summary    string `json:"summary"`
			Content    string `json:"content"`
			CoverURL   string `json:"cover_url"`
			Author     string `json:"author"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "news.invalid_request", "请求格式错误", err))
			return
		}
		a, err := svc.CreateArticle(c.Request.Context(), application.CreateArticleCmd{
			CategoryID: optID(req.CategoryID), Title: req.Title, Summary: req.Summary,
			Content: req.Content, CoverURL: req.CoverURL, Author: req.Author, CreatedBy: caller(c),
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, a)
	})

	// GET /articles/:id — readable with read perm; bumps view counter for readers.
	r.GET("/articles/:id", gate(resRead, "read"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "news.invalid_id", "id 无效", err))
			return
		}
		a, err := svc.GetArticle(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		if a == nil {
			apiresp.Fail(c, errors.New(errors.KindNotFound, "news.article_not_found", "文章不存在"))
			return
		}
		// Count a read only when the caller is a reader viewing a published article.
		if a.Status == domain.StatusPublished && c.Query("track") == "1" {
			_ = svc.IncrViews(c.Request.Context(), id)
			a.Views++
		}
		apiresp.OK(c, a)
	})

	r.PATCH("/articles/:id", gate(resManage, "write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "news.invalid_id", "id 无效", err))
			return
		}
		var req struct {
			Title      *string `json:"title"`
			Summary    *string `json:"summary"`
			Content    *string `json:"content"`
			CoverURL   *string `json:"cover_url"`
			Author     *string `json:"author"`
			CategoryID *string `json:"category_id"` // "" clears
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "news.invalid_request", "请求格式错误", err))
			return
		}
		cmd := application.UpdateArticleCmd{
			ID: id, Title: req.Title, Summary: req.Summary, Content: req.Content,
			CoverURL: req.CoverURL, Author: req.Author,
		}
		if req.CategoryID != nil {
			if *req.CategoryID == "" {
				cmd.ClearCat = true
			} else {
				cmd.CategoryID = optID(*req.CategoryID)
			}
		}
		a, err := svc.UpdateArticle(c.Request.Context(), cmd)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, a)
	})

	r.POST("/articles/:id/publish", gate(resManage, "write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "news.invalid_id", "id 无效", err))
			return
		}
		a, err := svc.SetPublished(c.Request.Context(), id, true)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, a)
	})

	r.POST("/articles/:id/unpublish", gate(resManage, "write"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "news.invalid_id", "id 无效", err))
			return
		}
		a, err := svc.SetPublished(c.Request.Context(), id, false)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, a)
	})

	r.DELETE("/articles/:id", gate(resManage, "delete"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "news.invalid_id", "id 无效", err))
			return
		}
		if err := svc.DeleteArticle(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// ---- Reader feed (published, paged, by category) ----
	r.GET("/feed", gate(resRead, "read"), func(c *gin.Context) {
		page, err := svc.ListPublished(c.Request.Context(), optID(c.Query("category")), pagination(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, page)
	})
}

func optID(s string) *kernel.ID {
	if s == "" {
		return nil
	}
	id, err := kernel.ParseID(s)
	if err != nil {
		return nil
	}
	return &id
}

func pagination(c *gin.Context) kernel.Pagination {
	var p kernel.Pagination
	_ = c.ShouldBindQuery(&p)
	return p.Normalize()
}
