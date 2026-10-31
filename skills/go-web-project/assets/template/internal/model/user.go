// Package model 是 MVC 的 Model 层：数据模型 + 数据访问。
//
// 约定：只做持久化，不含业务规则；所有查询必须带 context；
// 返回 *gorm.ErrRecordNotFound 时统一转成 apperr.ErrNotFound。
package model

import "time"

// User 对应 users 表。字段与迁移文件保持一致，改动需同时加迁移。
type User struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Email     string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Name      string    `gorm:"size:64;not null;default:''" json:"name"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 固定表名，避免 GORM 复数化规则变化导致漂移。
func (User) TableName() string {
	return "users"
}
