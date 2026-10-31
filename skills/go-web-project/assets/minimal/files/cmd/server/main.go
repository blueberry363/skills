// Command server 是 HTTP API 服务的进程入口。
//
// 只做装配：读配置 → 建依赖 → 注册路由 → 启动 → 优雅退出。
// 新增资源时在 router 注册一行，并在这里补上 service/controller 的构造。
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"{{MODULE}}/internal/config"
	"{{MODULE}}/internal/controller"
	"{{MODULE}}/internal/database"
	"{{MODULE}}/internal/router"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(cfg)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("main: 获取底层连接失败: %w", err)
	}
	defer sqlDB.Close()

	// 依赖装配：model（数据）→ service（业务）→ controller（HTTP）
	// 骨架阶段只有探针；新增资源时在此追加三行装配。
	healthController := controller.NewHealthController(sqlDB)

	engine := router.New(router.Options{
		Config:       cfg,
		Logger:       logger,
		Health:       healthController,
		TrustProxy:   cfg.TrustProxyHeaders,
		MaxBodyBytes: cfg.MaxBodyBytes,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("server starting", "addr", cfg.HTTPAddr, "env", cfg.Env)
	return serve(ctx, srv, logger, cfg.ShutdownTimeout)
}

// serve 启动服务并阻塞直到退出信号，随后在超时内完成优雅关闭。
func serve(ctx context.Context, srv *http.Server, logger *slog.Logger, shutdownTimeout time.Duration) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve: 监听失败: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("serve: 优雅关闭失败: %w", err)
		}
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: 关闭期间监听异常: %w", err)
		}
		return nil
	}
}

// newLogger 构造进程唯一的 slog.Logger：开发环境用文本，其余用 JSON。
func newLogger(cfg config.Config) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if cfg.Env == "development" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(handler).With("env", cfg.Env)
}
