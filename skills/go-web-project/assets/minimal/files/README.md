# {{SERVICE}}

Go Web 服务骨架：**Gin + GORM + MySQL 8.4**，MVC 分层（controller / service / model）。
当前为 **minimal 模式**：只有分层骨架与探针，业务示例已移除。

- 面向人的快速上手 → 本文件
- 工程约束、命令清单、评审规则 → [AGENTS.md](AGENTS.md)（agent 会读它）

## 快速开始

```bash
cp .env.example .env          # 默认已指向 compose 里的 MySQL
docker compose up -d db
make migrate-up               # 执行 0001_init.sql（占位迁移）
make run
```

验证：

```bash
curl localhost:8080/healthz   # {"status":"ok"}
curl localhost:8080/readyz    # 数据库可用时 200
curl localhost:8080/api/v1/nope   # 统一错误体，code=40401
```

响应统一为 `{"code":0,"message":"ok","data":...}`，失败时 `code` 为业务码（如 `40401`）。

## 目录（MVC）

```
cmd/server/          进程入口，只做装配
internal/
  router/            路由表与中间件装配
  middleware/        RequestID / AccessLog / Recovery / BodyLimit
  controller/        C：绑定、校验、调用 service、model → view（当前只有探针）
  service/           S：业务规则与事务边界（空，见 doc.go）
  model/             M：GORM 模型 + 数据访问（空，见 doc.go）
  response/          统一响应体与错误映射
  apperr/            业务错误哨兵
  config/            环境变量配置
  database/          GORM 连接池
db/migrations/       goose 迁移（0001_init.sql 为占位）
api/                 OpenAPI 契约
docs/                专题文档（HTTP/数据/测试/安全/编码规范），入口见 AGENTS.md 索引
```

## 加第一个资源（按这个顺序）

1. `db/migrations/0002_create_xxx.sql`：建表；
2. `internal/model/xxx.go` + `xxx_store.go`：模型与 store；
3. `internal/service/xxx.go`：用例与事务；
4. `internal/controller/xxx.go`：绑定、校验、view 转换；
5. `internal/router/router.go`：加一行 `Register`；
6. `api/openapi.yaml`：补端点契约；
7. 写 controller 测试（走 `router.New` 装配）。

## 常用命令

| 目的 | 命令 |
|---|---|
| 构建 | `go build ./...` |
| 运行 | `make run` |
| 测试 | `go test -shuffle=on ./...` |
| 全量验收 | `make check` |
| 迁移 | `make migrate-up` |

完整清单与分层约束见 [AGENTS.md](AGENTS.md)。
