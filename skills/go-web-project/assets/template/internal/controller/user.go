// Package controller 是 MVC 的 Controller 层（也承担 View 职责）。
//
// 约束：只做 HTTP 适配——参数绑定与格式校验、调用 service、
// 把 model 转成 view 输出。业务规则写在 service，SQL 写在 model。
package controller

import (
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"{{MODULE}}/internal/apperr"
	"{{MODULE}}/internal/model"
	"{{MODULE}}/internal/response"
	"{{MODULE}}/internal/service"
)

// UserController 处理 /api/v1/users 下的请求。
type UserController struct {
	svc    service.UserService
	logger *slog.Logger
}

// NewUserController 构造控制器。
func NewUserController(svc service.UserService, logger *slog.Logger) *UserController {
	return &UserController{svc: svc, logger: logger}
}

// Register 注册本控制器的路由。
func (ctl *UserController) Register(r gin.IRouter) {
	group := r.Group("/users")
	group.POST("", ctl.Create)
	group.GET("", ctl.List)
	group.GET("/:id", ctl.Get)
}

// createUserRequest 是请求体契约，binding 标签由 Gin 内置 validator 执行。
type createUserRequest struct {
	Email string `json:"email" binding:"required,email,max=255"`
	Name  string `json:"name" binding:"required,min=1,max=64"`
}

// userView 是响应视图（MVC 的 V）。不直接序列化 model，避免泄露字段。
type userView struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type listView struct {
	Items  []userView `json:"items"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

func newUserView(u *model.User) userView {
	return userView{ID: u.ID, Email: u.Email, Name: u.Name, CreatedAt: u.CreatedAt}
}

// Create 处理 POST /api/v1/users。
func (ctl *UserController) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, ctl.logger, bindError(err))
		return
	}

	user, err := ctl.svc.Create(c.Request.Context(), service.CreateUserInput{
		Email: req.Email,
		Name:  req.Name,
	})
	if err != nil {
		response.Fail(c, ctl.logger, err)
		return
	}

	response.Created(c, newUserView(user))
}

// Get 处理 GET /api/v1/users/:id。
func (ctl *UserController) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, ctl.logger, apperr.Invalid("id", "必须为正整数"))
		return
	}

	user, err := ctl.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, ctl.logger, err)
		return
	}

	response.OK(c, newUserView(user))
}

// List 处理 GET /api/v1/users。
func (ctl *UserController) List(c *gin.Context) {
	limit, err := queryInt(c, "limit", 20)
	if err != nil {
		response.Fail(c, ctl.logger, err)
		return
	}
	offset, err := queryInt(c, "offset", 0)
	if err != nil {
		response.Fail(c, ctl.logger, err)
		return
	}

	users, err := ctl.svc.List(c.Request.Context(), limit, offset)
	if err != nil {
		response.Fail(c, ctl.logger, err)
		return
	}

	items := make([]userView, 0, len(users))
	for i := range users {
		items = append(items, newUserView(&users[i]))
	}

	response.OK(c, listView{Items: items, Limit: limit, Offset: offset})
}

// bindError 把绑定错误转为字段级 422：只输出 json 字段名与原因，不回显 Go 类型名或 validator 原文。
func bindError(err error) error {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) && len(verrs) > 0 {
		return apperr.Invalid(verrs[0].Field(), tagMessage(verrs[0].Tag()))
	}
	return apperr.Invalid("body", "请求体格式不正确")
}

func tagMessage(tag string) string {
	switch tag {
	case "required":
		return "不能为空"
	case "email":
		return "格式不合法"
	case "max":
		return "长度超出上限"
	case "min":
		return "长度不足"
	default:
		return "不合法"
	}
}

// queryInt 解析查询参数，缺失用默认值，非法值返回 422。
func queryInt(c *gin.Context, key string, def int) (int, error) {
	raw := c.Query(key)
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, apperr.Invalid(key, "必须为整数")
	}
	return n, nil
}
