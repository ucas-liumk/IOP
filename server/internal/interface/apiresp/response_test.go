package apiresp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/shared/errors"
)

func TestOK_WrapsData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	OK(c, map[string]string{"hello": "world"})

	var body struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != 0 {
		t.Fatalf("expected code=0, got %d", body.Code)
	}
	if body.Data["hello"] != "world" {
		t.Fatalf("data not wrapped")
	}
}

func TestFail_MapsKindToStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	Fail(c, errors.New(errors.KindNotFound, "iam.user.not_found", "用户不存在"))
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
