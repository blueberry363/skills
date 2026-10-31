// Package service 是 MVC 的 Service 层：业务规则与事务边界。
//
// 约束：不 import gin 与 net/http；只返回 apperr 中的错误；
// 参数校验先在这里做语义校验（格式由 controller 的 binding 负责）。
package service

import (
	"context"
	"fmt"
	"strings"

	"{{MODULE}}/internal/apperr"
	"{{MODULE}}/internal/model"
)

// CreateUserInput 是业务入参，不含 HTTP 细节。
type CreateUserInput struct {
	Email string
	Name  string
}

// UserService 是用户领域的用例集合。
type UserService interface {
	Create(ctx context.Context, in CreateUserInput) (*model.User, error)
	Get(ctx context.Context, id int64) (*model.User, error)
	List(ctx context.Context, limit, offset int) ([]model.User, error)
}

type userService struct {
	store model.UserStore
}

// NewUserService 构造业务服务。
func NewUserService(store model.UserStore) UserService {
	return &userService{store: store}
}

func (s *userService) Create(ctx context.Context, in CreateUserInput) (*model.User, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if !strings.Contains(in.Email, "@") {
		return nil, apperr.Invalid("email", "格式不合法")
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, apperr.Invalid("name", "不能为空")
	}

	user := &model.User{Email: in.Email, Name: in.Name}
	if err := s.store.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("service.CreateUser: %w", err)
	}
	return user, nil
}

func (s *userService) Get(ctx context.Context, id int64) (*model.User, error) {
	if id <= 0 {
		return nil, apperr.Invalid("id", "必须为正整数")
	}

	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service.GetUser: %w", err)
	}
	return user, nil
}

func (s *userService) List(ctx context.Context, limit, offset int) ([]model.User, error) {
	if limit < 1 || limit > 100 {
		return nil, apperr.Invalid("limit", "必须在 1–100 之间")
	}
	if offset < 0 {
		return nil, apperr.Invalid("offset", "不能为负")
	}

	users, err := s.store.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("service.ListUsers: %w", err)
	}
	return users, nil
}
