<!--
本文件由 go-web-project 技能生成。用法：
1. 替换 {{MODULE}}（go.mod 的 module 路径）与 {{SERVICE}}（二进制名）
2. 删除仓库中不存在的段落（不要留空标题）
3. 第 2 段每条命令必须实测通过；跑不通的改为真实命令或删除
4. 与代码同 PR 维护
根文件预算 80–180 行：这里只放每天都用的约束，专题细节写进 docs/，由第 6 段索引。
-->

# AGENTS.md

本仓库是一个 Go Web 服务：模块路径 `{{MODULE}}`，二进制名 `{{SERVICE}}`，技术栈 Gin + GORM + MySQL 8.4，代码按 **MVC** 分层（controller / service / model）。

本文件只放"每天都用得上"的约束。专题细节在 `docs/`，动手前按第 6 段索引读对应文档，不要凭印象改。

## 1. 速览

- 请求路径：`middleware → router → controller(binding) → service → model(GORM) → response`
- 依赖方向：`main → router → controller → service → model`；`response`、`apperr`、`reqctx` 是叶子包，任何层可用
- `controller` 禁止 import `gorm.io/gorm`，`model` 禁止 import `gin`，`service` 禁止 import `gin` 与 `net/http`

```
cmd/server/            进程入口，只做装配：读配置 → 建依赖 → 注册路由 → 启动 → 优雅退出
internal/router/       路由表与中间件装配（新增资源只加一行 Register）
internal/middleware/   RequestID / AccessLog / Recovery / BodyLimit
internal/controller/   C：绑定参数、格式校验、调用 service、model → view
internal/service/      S：业务规则与事务边界
internal/model/        M：GORM 模型 + 数据访问（store 接口与实现）
internal/response/     统一响应体与错误映射（唯一写 HTTP 状态码的地方）
internal/apperr/       业务错误哨兵
internal/reqctx/       gin.Context 键名（middleware 与 response 共用）
internal/config/       环境变量解析与校验
internal/database/     GORM 连接池构造
db/migrations/         goose 迁移，schema 的唯一来源
api/                   OpenAPI 契约
docs/                  专题文档（按第 6 段索引按需读）
```

## 2. 命令

| 目的 | 命令 | 前提 |
|---|---|---|
| 构建 | `go build ./...` | — |
| 运行 | `make run` | 需 `.env`（make 自动加载）或导出 `DATABASE_URL` |
| 单元测试 | `go test -shuffle=on ./...` | — |
| 集成测试 | `make test-integration` | 需 make、`make db-up`、`.env` |
| 竞态检测 | `CGO_ENABLED=1 go test -race ./...` | 需 C 工具链；CI 默认执行 |
| 静态检查 | `go vet ./...` | — |
| 格式检查 | `gofmt -l .`（必须无输出） | — |
| 依赖整理 | `go mod tidy` | 首次联网 |
| 全量验收 | `make check` | 需 make；等价 `gofmt -l` + `go vet` + `go test` |
| 起停数据库 | `make db-up` / `make db-down` | 需 docker |
| 迁移 | `make migrate-up` / `make migrate-down` | 需 make、`DATABASE_URL`，首次联网拉 goose |
| Lint | `make lint` | 需 golangci-lint |
| 漏洞扫描 | `make vuln` | 需 govulncheck |

`make` 目标需 GNU make；Windows 在 Git Bash 或 WSL 中执行。CI 另有 lint 与漏洞扫描任务（见第 4 段）。

**完成定义**：`make check` 全绿（无 make 时逐条执行格式、静态检查、测试三行），新增接口有对应的 controller 测试，`go mod tidy` 无 diff。

`Makefile` 是任务入口的唯一事实源。本文件与 Makefile 不一致时以 Makefile 为准，并在同一个 PR 内修正本文件。

改动前先跑一次 `make check` 建立基线：既有失败要在 PR 里说明，不要混进自己的改动。

## 3. 层职责红线

| 层 | 可以做 | 禁止 |
|---|---|---|
| controller | 绑定与格式校验、调用 service、model → view 转换 | 业务规则、SQL、直接 `c.JSON` 拼响应（走 `response`） |
| service | 业务规则、不变量校验、事务边界、编排多个 store | import `gin`/`net/http`、写 SQL、返回 HTTP 状态码 |
| model | GORM 查询、模型定义、数据映射 | 业务规则、HTTP 概念、返回 `gin` 类型 |

新增一个资源：`model` 加模型与 store → `service` 加用例 → `controller` 加绑定与 view → `router` 加一行 `Register` → `db/migrations` 加迁移 → `api/openapi.yaml` 补契约 → 写 controller 测试。

常见改动落点：

| 需求 | 改哪些文件 |
|---|---|
| 新增端点 | `controller` 绑定 → `service` 用例 → `router` 注册 → `api/openapi.yaml` → controller 测试 |
| 新增表或字段 | `db/migrations` 加迁移 → `model` 改结构体 → `make migrate-up` |
| 新增错误码 | `apperr` 加哨兵 → `response.Fail` 加映射 → 测试断言 `code` |
| 加中间件 | `middleware` 实现 → `router` 调整顺序 → 同步 `docs/api.md` 的顺序说明 |
| 升级依赖 | `go get` → `go mod tidy` → `make vuln` → 更新 `docs/stack.md` 版本号 |

## 4. 提交与 PR

- 分支：`feat/<scope>`、`fix/<scope>`、`chore/<scope>`。
- commit：Conventional Commits，例如 `feat(api): add user list endpoint`。
- PR 前本地执行 `make check`；CI 执行同一套命令外加 `test-race`、集成测试编译检查、Lint 与漏洞扫描。
- 涉及 schema 的 PR 必须同时包含迁移与回滚脚本，并在描述中写明数据影响。
- 一个 PR 只做一类改动：功能、重构、依赖升级分开提交，便于回滚与评审。
- 改动分层、命令、`docs/` 或迁移流程时，同一个 PR 内同步更新对应文档。

## 5. 禁止清单

- 不在 controller 写业务规则，不在 service 拿 `*gin.Context`。
- 不新增第二套 Web 框架、ORM、日志库、配置库。
- 不使用 `gin.Default()`、`AutoMigrate`、`http.DefaultClient`。
- 不手改迁移文件的历史部分（已执行的迁移只能追加，不能改语义）。
- 不用 `init()` 做业务初始化。
- 不把 `context.Context` 存进结构体字段。
- 不在 `recover()` 之后继续写响应。
- 不在 handler 内直接写业务审计日志；接入审计后统一走审计中间件。
- 不以 `TODO`、空实现或注释掉的代码合并功能。

## 6. 文档索引

| 何时读 | 文档 |
|---|---|
| 加/改端点、改响应体或状态码、调日志字段 | `docs/api.md` |
| 加表、改字段、写事务、改迁移、调连接池 | `docs/data.md` |
| 写测试、判断覆盖到什么程度 | `docs/testing.md` |
| 换依赖、加配置项、动认证与密钥 | `docs/stack.md`、`docs/security.md` |
| 写业务代码前（命名、错误、并发、性能） | `docs/coding-standards.md` |

其他关键文件：`api/openapi.yaml`（对外契约）、`db/migrations/`（schema 演进历史）、`.golangci.yml`（Lint 规则）、`README.md`（面向人的快速开始）。

`docs/` 与代码同 PR 维护。本文件与 `docs/` 冲突时，以本文件和 `Makefile` 为准。
