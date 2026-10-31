// Package database 建立并配置 GORM 的 MySQL 连接。
//
// 启动时 Ping 一次，失败即返回错误让进程退出，不带病启动。
package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Open 打开连接池并验证可用。
//
// dsn 形如 user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local，
// 不是 PostgreSQL 那样的 URL。
func Open(ctx context.Context, dsn string, maxConns int32) (*gorm.DB, error) {
	// GORM 依赖 parseTime=True 才能把 DATETIME 扫描进 time.Time；
	// 缺了它不会立刻报错，而是在读时间字段时才炸，所以这里提前拦下。
	if !strings.Contains(dsn, "parseTime=true") && !strings.Contains(dsn, "parseTime=True") {
		return nil, fmt.Errorf("database: DATABASE_URL 必须包含 parseTime=True")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		// 把驱动错误翻译成 gorm.ErrDuplicatedKey 等哨兵错误，
		// 业务层就不必解析 MySQL 的错误码 1062。
		TranslateError: true,
		// 慢查询交给应用日志，GORM 只保留告警级输出。
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("database: 连接失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: 获取底层连接: %w", err)
	}
	sqlDB.SetMaxOpenConns(int(maxConns))
	sqlDB.SetMaxIdleConns(int(maxConns) / 2)
	// 小于 MySQL 的 wait_timeout 默认值（8h），避免拿到被服务端断开的连接。
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("database: ping 失败: %w", err)
	}

	return db, nil
}
