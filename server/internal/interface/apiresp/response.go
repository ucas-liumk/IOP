package apiresp

import (
	stderrors "errors"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// envelope is the uniform response shape:
// success:  {"code":0, "data": ..., "trace_id": "..."}
// failure:  {"code":-1, "error": {"code","message","kind"}, "trace_id":"..."}
type envelope struct {
	Code    int     `json:"code"`
	Data    any     `json:"data,omitempty"`
	Error   *errOut `json:"error,omitempty"`
	TraceID string  `json:"trace_id"`
}

type errOut struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Kind    string `json:"kind"`
}

// OK writes 200 with the wrapped data.
func OK(c *gin.Context, data any) {
	c.JSON(200, envelope{
		Code:    0,
		Data:    data,
		TraceID: kernel.TraceIDFromContext(c.Request.Context()),
	})
}

// Created writes 201 with the wrapped data.
func Created(c *gin.Context, data any) {
	c.JSON(201, envelope{
		Code:    0,
		Data:    data,
		TraceID: kernel.TraceIDFromContext(c.Request.Context()),
	})
}

// Fail writes the error using Kind → HTTP status mapping.
// If err is not *errors.Error, treats as KindInternal.
func Fail(c *gin.Context, err error) {
	var e *errors.Error
	if !stderrors.As(err, &e) {
		e = errors.Wrap(errors.KindInternal, "internal.unwrapped", "未分类错误", err)
	}
	c.AbortWithStatusJSON(e.Kind.HTTPStatus(), envelope{
		Code: -1,
		Error: &errOut{
			Code:    e.Code,
			Message: e.Message,
			Kind:    e.Kind.String(),
		},
		TraceID: kernel.TraceIDFromContext(c.Request.Context()),
	})
}
