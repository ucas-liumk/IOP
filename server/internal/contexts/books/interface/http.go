// Package iface wires the books module's REST routes. Mounted under
// /api/apps/books/* by the module Registry; every route is RBAC-gated via the
// module.AuthzFunc passed in (resource×action from the Manifest).
package iface

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/books/application"
	"github.com/leo/iop/server/internal/contexts/books/domain"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/module"
)

// Resource name (namespaced by module code so catalogs don't collide).
const bookRes = "books.book"

func RegisterRoutes(r *gin.RouterGroup, svc *application.Service, authz module.AuthzFunc) {
	gate := func(resource, action string) gin.HandlerFunc {
		if authz == nil {
			return func(c *gin.Context) { c.Next() }
		}
		return authz(resource, action)
	}
	memberID := func(c *gin.Context) kernel.ID {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		if claims == nil {
			return ""
		}
		return claims.MemberID
	}

	// ---- Books (catalog) ----
	r.GET("/books", gate(bookRes, "read"), func(c *gin.Context) {
		f := domain.BookFilter{
			Search:   c.Query("search"),
			Category: c.Query("category"),
			Page:     atoiDefault(c.Query("page"), 1),
			PageSize: atoiDefault(c.Query("page_size"), 20),
		}
		books, total, err := svc.ListBooks(c.Request.Context(), f)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"books": books, "total": total, "page": f.Page, "page_size": f.PageSize})
	})

	r.GET("/books/:id", gate(bookRes, "read"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "books.invalid_id", "id 无效", err))
			return
		}
		b, err := svc.GetBook(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, b)
	})

	r.POST("/books", gate(bookRes, "manage"), func(c *gin.Context) {
		var req bookReq
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "books.invalid_request", "请求格式错误", err))
			return
		}
		b, err := svc.CreateBook(c.Request.Context(), application.CreateBookCmd{
			ISBN: req.ISBN, Title: req.Title, Author: req.Author, Publisher: req.Publisher,
			Category: req.Category, Total: req.Total, CoverURL: req.CoverURL, Location: req.Location,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, b)
	})

	r.PATCH("/books/:id", gate(bookRes, "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "books.invalid_id", "id 无效", err))
			return
		}
		var req bookReq
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "books.invalid_request", "请求格式错误", err))
			return
		}
		b, err := svc.UpdateBook(c.Request.Context(), application.UpdateBookCmd{
			ID: id, ISBN: req.ISBN, Title: req.Title, Author: req.Author, Publisher: req.Publisher,
			Category: req.Category, Total: req.Total, CoverURL: req.CoverURL, Location: req.Location,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, b)
	})

	r.DELETE("/books/:id", gate(bookRes, "manage"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "books.invalid_id", "id 无效", err))
			return
		}
		if err := svc.DeleteBook(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// ---- Borrow / Return ----
	r.POST("/books/:id/borrow", gate(bookRes, "borrow"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "books.invalid_id", "id 无效", err))
			return
		}
		rec, err := svc.Borrow(c.Request.Context(), id, memberID(c))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, rec)
	})

	r.POST("/borrows/:id/return", gate(bookRes, "borrow"), func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "books.invalid_id", "id 无效", err))
			return
		}
		if err := svc.Return(c.Request.Context(), id); err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	// ---- Borrow records ----
	// scope=mine (default) → caller's own; scope=all → every record (needs manage).
	r.GET("/borrows", gate(bookRes, "read"), func(c *gin.Context) {
		f := domain.BorrowFilter{Status: c.Query("status")}
		if c.Query("scope") == "all" {
			// All-records view requires the manage permission.
			gate(bookRes, "manage")(c)
			if c.IsAborted() {
				return
			}
		} else {
			f.MemberID = memberID(c)
		}
		out, err := svc.ListBorrows(c.Request.Context(), f)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"borrows": out})
	})
}

type bookReq struct {
	ISBN      string `json:"isbn"`
	Title     string `json:"title" binding:"required"`
	Author    string `json:"author"`
	Publisher string `json:"publisher"`
	Category  string `json:"category"`
	Total     int    `json:"total"`
	CoverURL  string `json:"cover_url"`
	Location  string `json:"location"`
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return def
		}
		n = n*10 + int(r-'0')
	}
	if n == 0 {
		return def
	}
	return n
}
