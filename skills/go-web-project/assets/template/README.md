# {{SERVICE}}

Go Web 服务骨架：**Gin + GORM + MySQL 8.4**，MVC 分层（controller / service / model）。

- 面向人的快速上手 → 本文件
- 工程约束、命令清单、评审规则 → [AGENTS.md](AGENTS.md)（agent 会读它）

## 快速开始

```bash
cp .env.example .env          # 默认已指向 compose 里的 MySQL
docker compose up -d db
make migrate-up               # 建表
make run
```

验证：

```bash
curl localhost:8080/healthz

curl -X POST localhost:8080/api/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"email":"a@b.com","name":"Alice"}'

curl 'localhost:8080/api/v1/users?limit=10&offset=0'
curl localhost:8080/api/v1/users/1
```

响应统一为 `{"code":0,"message":"ok","data":...}`，失败时 `code` 为业务码（如 `40401`）。

## 目录（MVC）

```
cmd/server/          进程入口，只做装配
internal/
  router/            路由表与中间件装配
  middleware/        RequestID / AccessLog / Recovery / BodyLimit
  controller/        C：绑定、校验、调用 service、model → view
  service/           业务规则与事务边界
  model/             M：GORM 模型 + 数据访问
  response/          统一响应体与错误映射
  apperr/            业务错误哨兵
  config/            环境变量配置
  database/          GORM 连接池
db/migrations/       goose 迁移
api/                 OpenAPI 契约
docs/                专题文档（HTTP/数据/测试/安全/编码规范），入口见 AGENTS.md 索引
```

新增资源：`model` → `service` → `controller` → `router` 加一行 → 加迁移。

## 常用命令

| 目的 | 命令 |
|---|---|
| 构建 | `go build ./...` |
| 运行 | `make run`（会加载 `.env`） |
| 测试 | `go test -shuffle=on ./...` |
| 全量验收 | `make check` |
| 迁移 | `make migrate-up` |

完整清单与分层约束见 [AGENTS.md](AGENTS.md)。
