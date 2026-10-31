# 技术栈与配置

本仓库的选型是**有意为之**的，改动前先确认收益大于迁移成本。

## 运行时与库

| 关注点 | 选型 | 约束 |
|---|---|---|
| 语言 | Go {{GO_VERSION}}（以 `go.mod` 为准） | 不降级；升级单独 PR |
| Web 框架 | `github.com/gin-gonic/gin` v1.12 | 唯一 HTTP 框架，不再叠第二套 |
| 参数校验 | Gin 内置 `binding` 标签（validator v10） | 格式校验写标签，语义校验写 service |
| ORM | `gorm.io/gorm` v1.31 + `gorm.io/driver/mysql` | 开启 `TranslateError`，用哨兵错误判定冲突 |
| 数据库 | MySQL 8.4（InnoDB / utf8mb4） | schema 由迁移文件定义 |
| 迁移 | `goose` v3，经 `make` 以固定版本 `go run` | 禁止 `AutoMigrate` |
| 日志 | 标准库 `log/slog` | 不引 zap/logrus |
| 配置 | 环境变量 + `internal/config` | 不引 viper，不读 YAML |
| 契约 | OpenAPI 3.1（`api/openapi.yaml`） | 契约先行，改动同步实现与测试 |
| Lint | `golangci-lint` v2 | 配置在 `.golangci.yml`，与 CI 同一份 |

## 显式不引入

| 不引入 | 替代做法 |
|---|---|
| 第二个 Web 框架（Echo/Fiber/chi） | 用 Gin 的 Group 与中间件 |
| 原生 SQL / sqlc 混用 | GORM 查询构造器；确需原生 SQL 用 `db.Raw` 并写注释说明 |
| DI 容器（wire/fx） | `main.go` 手工装配，构造函数注入 |
| 断言库（testify 等） | 标准库 `if` + `t.Fatalf` |
| 配置框架（viper/envconfig） | `os.Getenv` + 默认值 + 启动即校验 |
| 日志库（zap/logrus/zerolog） | `log/slog` |
| GORM `AutoMigrate` | goose 迁移文件 |
| 重量级 mock 框架 | 接口 + 手写假实现 |

需要跨越任何一条时，先开 PR 说明理由，再动代码。

## 依赖引入规则

- 新增依赖前确认标准库与现有依赖无法解决，并在 PR 描述写明理由；
- 引入后立即 `go mod tidy`，提交 `go.mod` 与 `go.sum`；
- 不引入与现有库职责重叠的第二套实现（日志、错误、配置、ORM 各只有一套）；
- 新增依赖必须跑 `make vuln`（`govulncheck`）无高危告警。

## 配置项

配置只来自环境变量。本地把 `.env.example` 复制为 `.env` 后用 `make run`、`make migrate-up` 启动：Makefile 通过 `-include .env` 加载它，`.env` 的值覆盖同名环境变量；临时覆盖用 `make run DATABASE_URL=...`。直接 `go run` 不读取 `.env`。

| 变量 | 默认值 | 说明 |
|---|---|---|
| `DATABASE_URL` | 无（必填） | MySQL DSN（不是 URL），必须含 `parseTime=True`；缺失即启动失败 |
| `APP_ENV` | `development` | 决定日志格式与 Gin 模式 |
| `HTTP_ADDR` | `:8080` | 监听地址 |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `GIN_MODE` | 按 `APP_ENV` 派生 | 生产必须 `release` |
| `DB_MAX_CONNS` | `10` | 连接池上限 |
| `SHUTDOWN_TIMEOUT` | `15s` | 优雅关闭超时 |
| `TRUST_PROXY_HEADERS` | `false` | 仅可信代理后为 `true` |

新增配置项必须同时改三处：`internal/config`、`.env.example`、本表。
