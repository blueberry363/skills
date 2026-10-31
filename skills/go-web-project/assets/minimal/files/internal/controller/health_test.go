package controller_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"{{MODULE}}/internal/config"
	"{{MODULE}}/internal/controller"
	"{{MODULE}}/internal/router"
)

// stubPinger 让探针测试不依赖数据库。
type stubPinger struct{ err error }

func (s stubPinger) PingContext(context.Context) error { return s.err }

// newEngine 走真实装配路径（含中间件与严格解码），只替换数据库探针。
func newEngine(t *testing.T, p controller.Pinger) *gin.Engine {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return router.New(router.Options{
		Config:       config.Config{Env: "test", GinMode: gin.TestMode},
		Logger:       logger,
		Health:       controller.NewHealthController(p),
		MaxBodyBytes: 1 << 20,
	})
}

func TestLiveness(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	newEngine(t, stubPinger{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200", rec.Code)
	}
}

func TestReadiness(t *testing.T) {
	cases := []struct {
		name       string
		pingErr    error
		wantStatus int
	}{
		{"数据库可用", nil, http.StatusOK},
		{"数据库不可用", errors.New("connection refused"), http.StatusServiceUnavailable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

			newEngine(t, stubPinger{err: tc.pingErr}).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("状态码 = %d, 期望 %d, body=%s", rec.Code, tc.wantStatus, rec.Body)
			}
		})
	}
}
