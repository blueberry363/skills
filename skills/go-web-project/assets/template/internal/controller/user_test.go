package controller_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"{{MODULE}}/internal/apperr"
	"{{MODULE}}/internal/config"
	"{{MODULE}}/internal/controller"
	"{{MODULE}}/internal/model"
	"{{MODULE}}/internal/router"
	"{{MODULE}}/internal/service"
)

// fakeUserService 让 controller 测试不依赖数据库。
type fakeUserService struct {
	create func(ctx context.Context, in service.CreateUserInput) (*model.User, error)
	get    func(ctx context.Context, id int64) (*model.User, error)
	list   func(ctx context.Context, limit, offset int) ([]model.User, error)
}

func (f fakeUserService) Create(ctx context.Context, in service.CreateUserInput) (*model.User, error) {
	return f.create(ctx, in)
}

func (f fakeUserService) Get(ctx context.Context, id int64) (*model.User, error) {
	return f.get(ctx, id)
}

func (f fakeUserService) List(ctx context.Context, limit, offset int) ([]model.User, error) {
	return f.list(ctx, limit, offset)
}

// stubPinger 让健康检查在测试中不连数据库。
type stubPinger struct{ err error }

func (s stubPinger) PingContext(context.Context) error { return s.err }

// newEngine 走真实装配路径（含中间件与严格解码），只替换 service。
func newEngine(t *testing.T, svc service.UserService) *gin.Engine {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return router.New(router.Options{
		Config:       config.Config{Env: "test", GinMode: gin.TestMode},
		Logger:       logger,
		User:         controller.NewUserController(svc, logger),
		Health:       controller.NewHealthController(stubPinger{}),
		MaxBodyBytes: 1 << 20,
	})
}

// envelope 是统一响应体的测试视图。
type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func decode(t *testing.T, body []byte) envelope {
	t.Helper()
	var e envelope
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("响应不是合法 JSON: %v, body=%s", err, body)
	}
	return e
}

func TestCreateUser(t *testing.T) {
	okUser := &model.User{ID: 1, Email: "a@b.com", Name: "Alice", CreatedAt: time.Now()}

	cases := []struct {
		name       string
		body       string
		svcErr     error
		wantStatus int
		wantCode   int
	}{
		{"成功", `{"email":"a@b.com","name":"Alice"}`, nil, http.StatusCreated, 0},
		{"邮箱格式错误", `{"email":"not-an-email","name":"Alice"}`, nil, http.StatusUnprocessableEntity, 42201},
		{"缺少必填字段", `{"email":"a@b.com"}`, nil, http.StatusUnprocessableEntity, 42201},
		{"未知字段", `{"email":"a@b.com","name":"A","nope":1}`, nil, http.StatusUnprocessableEntity, 42201},
		{"业务拒绝", `{"email":"a@b.com","name":"Alice"}`, apperr.Invalid("email", "域名不被允许"), http.StatusUnprocessableEntity, 42201},
		{"邮箱已存在", `{"email":"a@b.com","name":"Alice"}`, apperr.ErrConflict, http.StatusConflict, 40901},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := fakeUserService{
				create: func(context.Context, service.CreateUserInput) (*model.User, error) {
					return okUser, tc.svcErr
				},
				get:  func(context.Context, int64) (*model.User, error) { return okUser, nil },
				list: func(context.Context, int, int) ([]model.User, error) { return nil, nil },
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			newEngine(t, svc).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("状态码 = %d, 期望 %d, body=%s", rec.Code, tc.wantStatus, rec.Body)
			}
			if got := decode(t, rec.Body.Bytes()).Code; got != tc.wantCode {
				t.Fatalf("业务码 = %d, 期望 %d", got, tc.wantCode)
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	cases := []struct {
		name       string
		path       string
		svcErr     error
		wantStatus int
	}{
		{"成功", "/api/v1/users/1", nil, http.StatusOK},
		{"不存在", "/api/v1/users/1", apperr.ErrNotFound, http.StatusNotFound},
		{"id 非法", "/api/v1/users/abc", nil, http.StatusUnprocessableEntity},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := fakeUserService{
				create: func(context.Context, service.CreateUserInput) (*model.User, error) {
					return &model.User{}, nil
				},
				get: func(context.Context, int64) (*model.User, error) {
					return &model.User{ID: 1, Email: "a@b.com", Name: "Alice"}, tc.svcErr
				},
				list: func(context.Context, int, int) ([]model.User, error) { return nil, nil },
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)

			newEngine(t, svc).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("状态码 = %d, 期望 %d, body=%s", rec.Code, tc.wantStatus, rec.Body)
			}
		})
	}
}

func TestListUsersDefaults(t *testing.T) {
	var gotLimit, gotOffset int
	svc := fakeUserService{
		create: func(context.Context, service.CreateUserInput) (*model.User, error) { return &model.User{}, nil },
		get:    func(context.Context, int64) (*model.User, error) { return &model.User{}, nil },
		list: func(_ context.Context, limit, offset int) ([]model.User, error) {
			gotLimit, gotOffset = limit, offset
			return []model.User{{ID: 2}, {ID: 1}}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

	newEngine(t, svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200", rec.Code)
	}
	if gotLimit != 20 || gotOffset != 0 {
		t.Fatalf("默认分页 = (%d,%d), 期望 (20,0)", gotLimit, gotOffset)
	}

	var payload struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
		Limit int `json:"limit"`
	}
	e := decode(t, rec.Body.Bytes())
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		t.Fatalf("data 解析失败: %v", err)
	}
	if len(payload.Items) != 2 || payload.Limit != 20 {
		t.Fatalf("data = %s", e.Data)
	}
}

func TestCreateUserBindErrorMessage(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantMsg string
	}{
		{"邮箱格式错误", `{"email":"not-an-email","name":"Alice"}`, "email: 格式不合法"},
		{"缺少必填字段", `{"email":"a@b.com"}`, "name: 不能为空"},
		{"未知字段", `{"email":"a@b.com","name":"A","nope":1}`, "body: 请求体格式不正确"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")

			newEngine(t, service.NewUserService(&fakeStore{})).ServeHTTP(rec, req)

			if got := decode(t, rec.Body.Bytes()).Message; got != tc.wantMsg {
				t.Fatalf("message = %q, 期望 %q", got, tc.wantMsg)
			}
		})
	}
}

// fakeStore 让 controller 测试走真实 service，只替换持久化层。
type fakeStore struct{ listCalls int }

func (s *fakeStore) Create(context.Context, *model.User) error { return nil }

func (s *fakeStore) GetByID(context.Context, int64) (*model.User, error) { return nil, nil }

func (s *fakeStore) List(context.Context, int, int) ([]model.User, error) {
	s.listCalls++
	return nil, nil
}

func TestListUsersRejectsOutOfRange(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{"limit 为 0", "/api/v1/users?limit=0"},
		{"limit 超过 100", "/api/v1/users?limit=101"},
		{"offset 为负", "/api/v1/users?offset=-1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStore{}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)

			newEngine(t, service.NewUserService(store)).ServeHTTP(rec, req)

			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("状态码 = %d, 期望 422, body=%s", rec.Code, rec.Body)
			}
			if store.listCalls != 0 {
				t.Fatalf("越界参数不应到达 store")
			}
		})
	}
}

func TestListUsersMaxLimit(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?limit=100", nil)

	newEngine(t, service.NewUserService(&fakeStore{})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200, body=%s", rec.Code, rec.Body)
	}
	var payload struct {
		Limit int `json:"limit"`
	}
	if err := json.Unmarshal(decode(t, rec.Body.Bytes()).Data, &payload); err != nil {
		t.Fatalf("data 解析失败: %v", err)
	}
	if payload.Limit != 100 {
		t.Fatalf("响应 limit = %d, 期望 100（应回显生效值）", payload.Limit)
	}
}

// 接口漂移时编译期即失败。
var _ service.UserService = fakeUserService{}
