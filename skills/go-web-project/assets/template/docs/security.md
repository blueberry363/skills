# 安全基线

上线前与新增依赖时读本文件。

## 输入与认证

- 格式校验写在 `binding` 标签，语义与业务不变量写在 service；两层都不信任上游。
- 接入认证时：由中间件解析凭证并把用户信息写入 `gin.Context`，controller 禁止自行解析 token 或读 `Authorization` 头。
- 鉴权失败返回 401、越权返回 403，不要用 404 掩盖越权（除确认资源存在性本身是敏感信息）。

## 密钥与配置

- 密钥只从环境变量或密钥管理读取；禁止硬编码，禁止提交 `.env`（`.env.example` 是唯一提交的样例）。
- 生产环境必须 `GIN_MODE=release`；`TRUST_PROXY_HEADERS` 仅在可信代理后设为 `true`，否则 `c.ClientIP()` 可被伪造。
- 数据库账号分两类：应用账号只授予 `SELECT` / `INSERT` / `UPDATE` / `DELETE`；迁移需要的 DDL 权限只在执行迁移时临时指定，如 `make migrate-up DATABASE_URL=<DDL 账号 DSN>`。应用运行时禁止使用 root。compose 中的本地账号拥有本库全部权限，仅供开发。

## 出站与边界

- 出站 HTTP 必须设置超时并复用 `http.Client`；禁止使用 `http.DefaultClient`。
- 请求体上限由 `BodyLimit` 中间件统一施加（默认 1 MiB），不要按接口放宽。
- 对外响应不要回显内部错误细节与堆栈；5xx 只返回通用文案，细节进日志。

## 依赖与漏洞

- 新增依赖前后各跑一次 `make vuln`；有高危告警不得合并。
- 安全用途的随机数用 `crypto/rand`，禁止 `math/rand`（见 `docs/coding-standards.md`）。
- 涉及鉴权、加密、文件上传的改动，PR 必须写明威胁模型与测试方式。
