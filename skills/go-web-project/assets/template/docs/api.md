# HTTP 约定、错误与响应

改端点、改响应体、加错误码或调日志字段前读本文件。

## 路由与绑定

- 路由写成 `group := r.Group("/<资源名>")` + `group.POST("")`，统一挂在 `router.New` 的 `/api/v1` 下；不要在 controller 里建新的路由树根。
- 请求体用 `c.ShouldBindJSON(&req)`，校验规则写在 `binding` 标签（`required,email,max=255`）；绑定失败统一返回 422，经 `bindError` 转换，message 为 `<json 字段名>: <原因>`（如 `email: 格式不合法`）。禁止把 `err.Error()` 原样回给客户端，它会暴露 Go 类型名与 validator 原文。
- 严格解码已全局开启（`gin.EnableJsonDecoderDisallowUnknownFields()`）：未知字段直接报 422，不要关掉。
- 路径参数用 `c.Param("id")` 并显式 `strconv` 转换；查询参数缺失用默认值、非法返回 422。
- 列表接口必须分页：`limit` 缺省为 20，取值范围 1–100；`limit` 越界或 `offset` 为负返回 422，不做静默修正；禁止无界查询。响应中的 `limit`/`offset` 为生效值。

## 响应与中间件

- 响应只经 `response.OK` / `response.Created` / `response.Fail` 写出；禁止在 controller 里直接 `c.JSON` 或 `c.AbortWithStatus`。
- 唯一例外：`/healthz` `/readyz` 由 `controller/health.go` 直接 `c.JSON` 返回裸 JSON（探针不解析响应信封），其余接口不得套用该例外。
- 中间件顺序固定：`RequestID → AccessLog → Recovery → BodyLimit`。不要使用 `gin.Default()`（它自带非结构化日志与精简版 Recovery）。
- `http.Server` 必须显式设置 `ReadHeaderTimeout`(5s)、`ReadTimeout`(15s)、`WriteTimeout`(30s)、`IdleTimeout`(60s)。
- 需要数据库或出站调用时传 `c.Request.Context()`，不要用 `context.Background()`。
- 新增端点必须同步 `api/openapi.yaml`，并在 controller 测试覆盖成功与至少一个失败分支。

## 错误模型

- 业务错误定义在 `internal/apperr`：`ErrUnauthorized/ErrForbidden/ErrNotFound/ErrConflict/ErrInvalid`，字段级用 `apperr.Invalid(field, msg)`。
- 包装格式：`fmt.Errorf("service.<方法名>: %w", err)`；判定用 `errors.Is` / `errors.As`；禁止比较错误字符串。
- 中间层不记日志；只在 handler 边界记录一次。4xx 不记（客户端问题），5xx 记录并带 `request_id`。

## 业务码与响应形状

状态码与业务码映射只发生在 `response.Fail`。业务码规则为 `HTTP 状态码 * 100 + 序号`：

| 业务码 | 触发条件 | 响应示例 |
|---|---|---|
| `0` | 成功 | `{"code":0,"message":"ok","data":{...}}` |
| `42201` | 校验失败 | `{"code":42201,"message":"email: 格式不合法"}` |
| `40101` / `40301` | 未认证 / 无权 | `{"code":40101,"message":"未认证"}` |
| `40401` | 资源不存在 | `{"code":40401,"message":"资源不存在"}` |
| `40901` | 唯一冲突 | `{"code":40901,"message":"资源已存在"}` |
| `50001` | 未预期错误 | `{"code":50001,"message":"服务器内部错误"}` |

## 日志字段

固定字段：`request_id`、`method`、`path`、`status`、`duration_ms`、`client_ip`、`err`。

禁止记录：token、密码、连接串、完整请求体、PII 明文。
