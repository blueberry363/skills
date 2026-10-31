// Package response 定义统一响应体与错误映射，是唯一写 HTTP 状态码的地方。
//
// 成功：{"code":0,"message":"ok","data":...}
// 失败：{"code":40401,"message":"资源不存在","data":null}
package response

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"{{MODULE}}/internal/apperr"
	"{{MODULE}}/internal/reqctx"
)

// Body 是所有响应的统一外壳。
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// OK 写出 200。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: 0, Message: "ok", Data: data})
}

// Created 写出 201。
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Body{Code: 0, Message: "ok", Data: data})
}

// Fail 把业务错误映射为状态码与业务码。
//
// 业务码规则：HTTP 状态码 * 100 + 序号（如 40401 表示 404 的第 1 种原因）。
func Fail(c *gin.Context, logger *slog.Logger, err error) {
	status := http.StatusInternalServerError
	code := 50001
	message := "服务器内部错误"

	switch {
	case errors.Is(err, apperr.ErrUnauthorized):
		status, code, message = http.StatusUnauthorized, 40101, "未认证"
	case errors.Is(err, apperr.ErrForbidden):
		status, code, message = http.StatusForbidden, 40301, "无权访问"
	case errors.Is(err, apperr.ErrNotFound):
		status, code, message = http.StatusNotFound, 40401, "资源不存在"
	case errors.Is(err, apperr.ErrConflict):
		status, code, message = http.StatusConflict, 40901, "资源已存在"
	case errors.Is(err, apperr.ErrInvalid):
		status, code, message = http.StatusUnprocessableEntity, 42201, err.Error()
	case errors.Is(err, context.Canceled):
		status, code, message = 499, 49901, "客户端已断开"
	}

	if status >= 500 && logger != nil {
		logger.ErrorContext(c.Request.Context(), "request failed",
			"err", err.Error(),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			reqctx.RequestIDKey, c.GetString(reqctx.RequestIDKey),
		)
	}

	c.JSON(status, Body{Code: code, Message: message})
}
