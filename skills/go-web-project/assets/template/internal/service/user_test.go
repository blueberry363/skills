package service_test

import (
	"context"
	"errors"
	"testing"

	"{{MODULE}}/internal/apperr"
	"{{MODULE}}/internal/model"
	"{{MODULE}}/internal/service"
)

// fakeUserStore 让 service 测试不依赖数据库。
type fakeUserStore struct {
	create func(ctx context.Context, user *model.User) error
	get    func(ctx context.Context, id int64) (*model.User, error)
	list   func(ctx context.Context, limit, offset int) ([]model.User, error)
}

func (f fakeUserStore) Create(ctx context.Context, user *model.User) error {
	return f.create(ctx, user)
}

func (f fakeUserStore) GetByID(ctx context.Context, id int64) (*model.User, error) {
	return f.get(ctx, id)
}

func (f fakeUserStore) List(ctx context.Context, limit, offset int) ([]model.User, error) {
	return f.list(ctx, limit, offset)
}

func TestCreateUserValidation(t *testing.T) {
	cases := []struct {
		name    string
		in      service.CreateUserInput
		wantErr error
	}{
		{"邮箱无 @", service.CreateUserInput{Email: "abc", Name: "A"}, apperr.ErrInvalid},
		{"名字为空", service.CreateUserInput{Email: "a@b.com", Name: "  "}, apperr.ErrInvalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := fakeUserStore{
				create: func(context.Context, *model.User) error { return nil },
				get:    func(context.Context, int64) (*model.User, error) { return nil, nil },
				list:   func(context.Context, int, int) ([]model.User, error) { return nil, nil },
			}

			_, err := service.NewUserService(store).Create(context.Background(), tc.in)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("错误 = %v, 期望 %v", err, tc.wantErr)
			}
		})
	}
}

func TestCreateUserNormalizesEmail(t *testing.T) {
	var saved *model.User
	store := fakeUserStore{
		create: func(_ context.Context, user *model.User) error {
			saved = user
			return nil
		},
		get:  func(context.Context, int64) (*model.User, error) { return nil, nil },
		list: func(context.Context, int, int) ([]model.User, error) { return nil, nil },
	}

	_, err := service.NewUserService(store).Create(context.Background(), service.CreateUserInput{
		Email: "  Alice@Example.COM ",
		Name:  " Alice ",
	})
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if saved == nil || saved.Email != "alice@example.com" || saved.Name != "Alice" {
		t.Fatalf("归一化结果 = %+v", saved)
	}
}

// 唯一约束冲突必须原样向上传递，由 response 层映射为 409。
func TestCreateUserConflictPropagates(t *testing.T) {
	store := fakeUserStore{
		create: func(context.Context, *model.User) error {
			return apperr.ErrConflict
		},
		get:  func(context.Context, int64) (*model.User, error) { return nil, nil },
		list: func(context.Context, int, int) ([]model.User, error) { return nil, nil },
	}

	_, err := service.NewUserService(store).Create(context.Background(), service.CreateUserInput{
		Email: "a@b.com",
		Name:  "Alice",
	})
	if !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("错误 = %v, 期望 ErrConflict", err)
	}
}

func TestListRejectsOutOfRange(t *testing.T) {
	cases := []struct {
		name          string
		limit, offset int
	}{
		{"limit 为 0", 0, 0},
		{"limit 超过 100", 101, 0},
		{"offset 为负", 20, -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := fakeUserStore{
				create: func(context.Context, *model.User) error { return nil },
				get:    func(context.Context, int64) (*model.User, error) { return nil, nil },
				list: func(context.Context, int, int) ([]model.User, error) {
					t.Errorf("越界参数不应到达 store")
					return nil, nil
				},
			}

			_, err := service.NewUserService(store).List(context.Background(), tc.limit, tc.offset)
			if !errors.Is(err, apperr.ErrInvalid) {
				t.Fatalf("错误 = %v, 期望 ErrInvalid", err)
			}
		})
	}
}

func TestGetUserRejectsNonPositiveID(t *testing.T) {
	store := fakeUserStore{
		create: func(context.Context, *model.User) error { return nil },
		get:    func(context.Context, int64) (*model.User, error) { return &model.User{}, nil },
		list:   func(context.Context, int, int) ([]model.User, error) { return nil, nil },
	}

	_, err := service.NewUserService(store).Get(context.Background(), 0)
	if !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("错误 = %v, 期望 ErrInvalid", err)
	}
}
