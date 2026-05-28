package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/shared/kernel"
)

func TestRequestID_GeneratesIfMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		tid := kernel.TraceIDFromContext(c.Request.Context())
		if tid == "" {
			t.Errorf("expected non-empty trace id in ctx")
		}
		c.String(200, tid)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Request-Id") == "" {
		t.Fatalf("expected X-Request-Id response header")
	}
	if w.Body.String() == "" {
		t.Fatalf("expected handler to see trace id")
	}
}

func TestRequestID_PreservesIncoming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(200) })
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "client-trace-xyz")
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Request-Id") != "client-trace-xyz" {
		t.Fatalf("expected echo of client trace id")
	}
}
