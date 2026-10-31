//go:build integration

// 集成测试：连真实 MySQL 验证 store 与迁移产物的一致性。
// 运行前：make db-up && make migrate-up，然后 make test-integration。
package model_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"{{MODULE}}/internal/apperr"
	"{{MODULE}}/internal/model"
)

// TestUserStoreCRUD 在单个事务内走完创建、查询、冲突与列表，结束时回滚，不留数据。
func TestUserStoreCRUD(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL 未设置：先 make db-up 并 cp .env.example .env")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("开启事务失败: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	store := model.NewUserStore(tx)
	ctx := context.Background()
	const email = "integration@example.com"

	user := &model.User{Email: email, Name: "集成测试"}
	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("创建后 ID 未回填")
	}

	got, err := store.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("按 ID 查询失败: %v", err)
	}
	if got.Email != email || got.Name != "集成测试" {
		t.Fatalf("查询结果不符: %+v", got)
	}

	dup := &model.User{Email: email, Name: "重复邮箱"}
	if err := store.Create(ctx, dup); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("唯一冲突应返回 ErrConflict，实际: %v", err)
	}

	users, err := store.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}
	if len(users) == 0 {
		t.Fatal("列表为空，但事务内刚创建了记录")
	}
}
