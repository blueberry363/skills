// Package middleware 存放全部 Gin 中间件。
//
// 顺序（外 → 内）：RequestID → AccessLog → Recovery → BodyLimit。
// 调整顺序必须评估 panic 是否仍被兜住、访问日志是否仍拿到状态码。
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"

	"{{MODULE}}/internal/reqctx"
	"{{MODULE}}/internal/response"
)

// RequestID 透传或生成 X-Request-Id，并写入 gin.Context 与响应头。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-Id")
		if id == "" {
			id = newRequestID()
		}
		c.Set(reqctx.RequestIDKey, id)
		c.Header("X-Request-Id", id)
		c.Next()
	}
}

// AccessLog 在请求结束后记录一行结构化访问日志。
func AccessLog(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		logger.InfoContext(c.Request.Context(), "http_request",
			reqctx.RequestIDKey, c.GetString(reqctx.RequestIDKey),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"bytes", c.Writer.Size(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}

// Recovery 兜住 panic，返回统一错误体，避免进程退出。
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			logger.ErrorContext(c.Request.Context(), "panic recovered",
				"panic", fmt.Sprint(rec),
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				reqctx.RequestIDKey, c.GetString(reqctx.RequestIDKey),
				"stack", string(debug.Stack()),
			)
			// 响应头若已写出，再写会触发 Gin 告警，属预期行为。
			response.Fail(c, logger, fmt.Errorf("panic: %v", rec))
			c.Abort()
		}()

		c.Next()
	}
}

// BodyLimit 限制请求体字节数，防止内存被打满。
func BodyLimit(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limit > 0 && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

// TrustedProxies 返回可信反向代理地址；为空表示不信任任何代理。
func TrustedProxies(enabled bool) []string {
	if !enabled {
		return nil
	}
	return []string{"127.0.0.1", "::1"}
}

// newRequestID 生成 16 字节随机 hex，避免为一行代码引入 uuid 依赖。
func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf)
}
