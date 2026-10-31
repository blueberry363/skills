// Package router 是唯一的路由注册处：中间件装配 + 路由表。
//
// 新增资源时只在这里加一行 Register，不要在其他包里建子路由树。
package router

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"{{MODULE}}/internal/apperr"
	"{{MODULE}}/internal/config"
	"{{MODULE}}/internal/controller"
	"{{MODULE}}/internal/middleware"
	"{{MODULE}}/internal/response"
)

// Options 是构造引擎所需的依赖，全部由 main 注入。
type Options struct {
	Config       config.Config
	Logger       *slog.Logger
	User         *controller.UserController
	Health       *controller.HealthController
	TrustProxy   bool
	MaxBodyBytes int64
}

// New 组装 gin.Engine。
func New(o Options) *gin.Engine {
	gin.SetMode(o.Config.GinMode)

	// 未知字段直接报错：字段名拼错时快速失败，而不是被静默忽略。
	gin.EnableJsonDecoderDisallowUnknownFields()
	// 字段错误用 json 名输出（如 email），而不是 Go 结构体字段名。
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(jsonTagName)
	}

	engine := gin.New()
	engine.Use(
		middleware.RequestID(),
		middleware.AccessLog(o.Logger),
		middleware.Recovery(o.Logger),
		middleware.BodyLimit(o.MaxBodyBytes),
	)

	// 只信任显式配置的代理，否则 c.ClientIP() 可被伪造。
	if err := engine.SetTrustedProxies(middleware.TrustedProxies(o.TrustProxy)); err != nil {
		o.Logger.Warn("设置可信代理失败", "err", err)
	}

	// 未匹配路由也走统一响应体，避免 Gin 返回纯文本 404。
	engine.NoRoute(func(c *gin.Context) {
		response.Fail(c, o.Logger, fmt.Errorf("route %s: %w", c.Request.URL.Path, apperr.ErrNotFound))
	})

	o.Health.Register(engine)

	api := engine.Group("/api/v1")
	o.User.Register(api)

	return engine
}

// jsonTagName 取 json 标签名，无标签或为 "-" 时回退到字段名。
func jsonTagName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	if name == "" || name == "-" {
		return f.Name
	}
	return name
}
