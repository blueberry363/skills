package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Pinger 是就绪探针依赖的最小能力，*sql.DB 直接满足，测试可替换。
type Pinger interface {
	PingContext(ctx context.Context) error
}

// HealthController 提供存活与就绪探针。
type HealthController struct {
	db Pinger
}

// NewHealthController 构造探针控制器。
func NewHealthController(db Pinger) *HealthController {
	return &HealthController{db: db}
}

// Register 注册探针路由。
func (ctl *HealthController) Register(r gin.IRouter) {
	r.GET("/healthz", ctl.Liveness)
	r.GET("/readyz", ctl.Readiness)
}

// Liveness 只表示进程活着，不做任何 IO。
func (ctl *HealthController) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readiness 表示依赖可用，会 ping 数据库。
func (ctl *HealthController) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := ctl.db.PingContext(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "dependency": "mysql"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
