// Package config 从环境变量加载配置。
//
// 约束：只读环境变量，不读配置文件；缺失必填项启动即失败；
// 默认值写在代码里，说明写在 .env.example 里。
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 是进程级配置，构造后只读。
type Config struct {
	Env               string
	HTTPAddr          string
	DatabaseURL       string
	DBMaxConns        int32
	LogLevel          string
	GinMode           string
	ShutdownTimeout   time.Duration
	MaxBodyBytes      int64
	TrustProxyHeaders bool
}

// Load 读取环境变量并校验。
func Load() (Config, error) {
	cfg := Config{
		Env:             envOr("APP_ENV", "development"),
		HTTPAddr:        envOr("HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		DBMaxConns:      10,
		LogLevel:        envOr("LOG_LEVEL", "info"),
		ShutdownTimeout: 15 * time.Second,
		MaxBodyBytes:    1 << 20,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: DATABASE_URL 未设置")
	}

	// Gin 的 debug 模式会打印路由与告警，线上必须 release。
	if cfg.Env == "development" {
		cfg.GinMode = "debug"
	} else {
		cfg.GinMode = "release"
	}
	if v := os.Getenv("GIN_MODE"); v != "" {
		cfg.GinMode = v
	}

	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("config: DB_MAX_CONNS 非法: %q", v)
		}
		cfg.DBMaxConns = int32(n)
	}

	for _, item := range []struct {
		key string
		dst *time.Duration
	}{
		{"SHUTDOWN_TIMEOUT", &cfg.ShutdownTimeout},
	} {
		v := os.Getenv(item.key)
		if v == "" {
			continue
		}
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("config: %s 非法: %q", item.key, v)
		}
		*item.dst = d
	}

	cfg.TrustProxyHeaders = envOr("TRUST_PROXY_HEADERS", "false") == "true"

	return cfg, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
