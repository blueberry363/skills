# 换栈对照表

默认基线是 `Gin + GORM + MySQL 8.4`，MVC 分层。团队已有技术栈时，按本表只改对应段落，其余段落（错误与日志、测试、安全、提交规范、禁止清单）基本不动。

## 1. Web 框架

基线是 Gin。换框架时**必须同步**第 1、3、4 段与中间件顺序，并重跑全部 controller 测试。

| 基线 | 替换为 | 需要改的段落 | 常见陷阱 |
|---|---|---|---|
| Gin | Echo | 第 1、3、4 段；`*gin.Context` → `echo.Context`；`ShouldBindJSON` → `Bind`；必须接管 `HTTPErrorHandler` 才能保住统一响应体 | Echo 默认错误处理返回纯文本 |
| Gin | 标准库 `net/http` + chi | 第 1、3、4 段；controller 签名变成 `func(w, r)`；绑定与校验要自己写 | 失去 binding 标签，校验代码量上升，需明确放置位置 |
| Gin | Fiber | 不推荐：`fasthttp` 与标准 `context`/`net/http` 生态不兼容 | 无法复用 `http.Handler` 中间件，探针与优雅关闭都要重写 |
| Gin | 继续 Gin，但按模块拆路由树 | 第 3、4 段 | 路由分散后"路由表在哪"会消失，需保留 `router` 包作为唯一注册处 |

## 2. 数据访问

| 基线 | 替换为 | 需要改的段落 | 常见陷阱 |
|---|---|---|---|
| GORM | sqlc + pgx | 第 1、3、6 段；新增 `db/queries` 与生成目录；model 层换成生成代码 | 需额外安装 sqlc，团队要接受"改 SQL 要跑生成" |
| GORM | sqlx | 第 1、6 段；查询写在 store 内，用 `db` tag 映射 | 手写扫描代码，`SELECT *` 更容易蔓延 |
| GORM | ent | 第 1、3、6 段 | 生成 schema 与迁移耦合，必须指定唯一来源（推荐仍用 goose） |
| 任意 ORM | 原生 `database/sql` | 第 1、6 段 | 事务与扫描样板代码激增，需统一封装 helper |

换掉 GORM 时，第 6 段关于 `TranslateError` 与 `gorm.ErrDuplicatedKey` 的说明必须一并重写，否则"唯一冲突 → 409"的链路会静默失效。

## 3. 数据库

| 基线 | 替换为 | 需要改的段落 | 常见陷阱 |
|---|---|---|---|
| MySQL 8.4 | PostgreSQL 17 | 第 1、2、6 段；驱动换 `gorm.io/driver/postgres`；DSN 从 `user:pass@tcp(host:port)/db` 换成 `postgres://user:pass@host:5432/db?sslmode=disable`；Makefile 里 goose 的方言参数由 `mysql` 改 `postgres` | `DATETIME(3)`→`timestamptz`、`AUTO_INCREMENT`→`GENERATED ALWAYS AS IDENTITY`、`parseTime=True` 不再需要改用 `sslmode`；`db.Raw` 的 `?` 占位符要改成 `$1`（GORM 会代劳，手写 Raw 不会） |
| MySQL 8.4 | SQLite（测试） | 第 7 段；可用 `gorm.io/driver/sqlite` 内存库跑集成测试 | 并发与类型亲和性有差异，文中写明"仅测试用" |
| MySQL 8.0/5.7 | MySQL 8.4 | 第 1、6 段 | 5.7 无 `utf8mb4_0900_ai_ci` 默认排序规则；`DATETIME(3)` 与 `ON UPDATE CURRENT_TIMESTAMP(3)` 需显式写 |

## 4. 分层风格

| 基线 | 替换为 | 需要改的段落 | 常见陷阱 |
|---|---|---|---|
| MVC（controller/service/model） | 三段式 + repository | 第 3 段；从 model 拆出 repository，model 只留结构体 | 拆完容易在 service 与 repository 之间重复校验 |
| MVC | 整洁架构 / DDD 分层 | 第 3 段整体重写 | 小项目引入后样板代码翻倍，需在 PR 说明收益 |
| MVC | 单文件 handler | 第 3 段；只适合极小服务 | 超过 3 个资源后难以定位，测试也无法分层 |

## 5. 换栈后的自检

- 第 2 段每条命令实测通过（换栈后命令几乎全变，最容易留错）；
- 第 4 段中间件顺序与新框架真实链路一致，且 panic 仍被兜住；
- 第 5 段的状态码映射在新响应层里仍然唯一；
- 第 6 段"冲突判定"在当前 ORM 下仍可 `errors.Is`；
- 第 10 段禁止清单与新栈不冲突（例如换了 sqlc 就不能再写"禁止原生 SQL"）。
