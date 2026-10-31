package model

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"{{MODULE}}/internal/apperr"
)

// UserStore 是 User 的持久化接口。service 依赖该接口，测试时用假实现替换。
type UserStore interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	List(ctx context.Context, limit, offset int) ([]User, error)
}

type userStore struct {
	db *gorm.DB
}

// NewUserStore 构造仓储实现。
func NewUserStore(db *gorm.DB) UserStore {
	return &userStore{db: db}
}

func (s *userStore) Create(ctx context.Context, user *User) error {
	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("model.CreateUser: %w", apperr.ErrConflict)
		}
		return fmt.Errorf("model.CreateUser: %w", err)
	}
	return nil
}

func (s *userStore) GetByID(ctx context.Context, id int64) (*User, error) {
	var user User
	// Select 显式列出字段：避免 SELECT *，也避免后续加了大字段被动加载。
	err := s.db.WithContext(ctx).
		Select("id", "email", "name", "created_at", "updated_at").
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("model.GetUser: %w", apperr.ErrNotFound)
		}
		return nil, fmt.Errorf("model.GetUser: %w", err)
	}
	return &user, nil
}

func (s *userStore) List(ctx context.Context, limit, offset int) ([]User, error) {
	var users []User
	err := s.db.WithContext(ctx).
		Select("id", "email", "name", "created_at", "updated_at").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("model.ListUsers: %w", err)
	}
	return users, nil
}
