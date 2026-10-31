# 数据与迁移

加表、改字段、写事务、调查询前读本文件。

## 连接

- 连接池在 `internal/database.Open` 构造：`TranslateError: true` 让唯一冲突变成 `gorm.ErrDuplicatedKey`，`MaxConns` 来自环境变量，`Ping` 失败即退出。
- 连接串是 MySQL DSN（`user:pass@tcp(host:port)/db?charset=utf8mb4&parseTime=True&loc=Local`），不是 PostgreSQL 那样的 URL；缺 `parseTime=True` 会在启动时被拦下。

## 查询

- 所有查询必须带 context：`s.db.WithContext(ctx)`。
- 查询显式列出字段（`Select("id", "email", ...)`），禁止 `SELECT *`；列表必须 `Limit`。
- 避免 N+1：关联数据用 `Preload` 或 `Joins`，不要在循环里查询。
- 禁止字符串拼接 SQL；需要拼接标识符时用白名单，不要相信入参。

## 事务

- 事务只在 service 层开启：`db.Transaction(func(tx *gorm.DB) error { ... })`。
- 事务内用 `model.NewXxxStore(tx)` 重新构造绑定该事务的 store 再调用，不要复用绑定连接池的 store。
- 事务函数内返回错误即回滚；不要在里面吞掉错误继续执行。

## 迁移

- 迁移只增不改，命名 `0001_init.sql`、`0002_add_user_status.sql`；合并前验证 `make migrate-up` 与 `make migrate-down`。
- 迁移文件是 schema 的唯一来源：`AutoMigrate` 一律禁止，模型字段变更必须配迁移。
- 建表必须显式 `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`，时间字段用 `DATETIME(3)`（对应 `time.Time`）；不要依赖服务器默认值，不同环境默认值不同。
- 唯一冲突的底层是 MySQL 错误码 1062；业务层只判 `gorm.ErrDuplicatedKey`，禁止匹配错误字符串。

## 资源释放

- 后台 goroutine 必须可退出：监听 `ctx.Done()` 或带超时，禁止在请求内启动无法回收的任务。
- 大结果集要分页或流式处理，禁止一次性 `Find` 全表。
