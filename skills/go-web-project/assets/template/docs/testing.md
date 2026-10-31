# 测试

写测试或判断"测到什么程度"时读本文件。

## 层次与写法

- controller 测试必须通过 `router.New(...)` 装配（含中间件与严格解码），只替换 `service` 的假实现，这样测到的行为与线上一致。
- service 测试用假 store（实现 model 层定义的 store 接口），不连数据库。
- `gin.SetMode(gin.TestMode)`；用 `httptest.NewRequest` / `httptest.NewRecorder`，不启动真实端口。
- 表驱动 + `t.Run` 子测试；断言用标准库，失败信息带上输入与期望值。
- 断言统一响应体时校验 `code` 与 `message`，不要只断言 HTTP 状态码。
- 共享的测试辅助函数调用 `t.Helper()`，让失败行号指向真正的用例。

## 覆盖到什么程度

- 新端点：成功 1 例 + 至少 1 个失败分支（校验失败或资源不存在）。
- bug 修复：先写能失败的复现测试，再改实现。
- 依赖数据库的测试加 `//go:build integration` 标签，用 `t.Cleanup` 清理数据；运行用 `make test-integration`（需 `make db-up` 与 `.env`，不进 `make check`）。
- 并发相关改动必须跑 `make test-race`（需 C 工具链）。
- 基准测试用 `b.Loop()`（Go 1.24+）。

## 禁止

- 禁止用 `time.Sleep` 等待异步结果，改用轮询加超时或同步原语。
- 禁止为了让测试通过而放宽断言或删除用例；修实现，不修标准。
- 禁止在测试里连线上数据库或使用生产凭证。
