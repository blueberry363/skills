# 编码规范

基线是 [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) 与 [Google Go Style Guide](https://google.github.io/styleguide/go/) 的共识部分。这里只列**能被 review 逐条判定**的条目；纯风格细节交给 `gofmt` 与 `golangci-lint`。

## 命名

- 包名用小写单词、不用下划线、不用复数：`user` 而非 `users` / `user_util`；目录名与包名保持一致。
- 禁止 `util`、`utils`、`common`、`helper` 包；放不下的代码说明职责没拆清。
- 缩写统一大写：`userID`、`httpClient`、`apiURL`；导出名写 `UserID`，不写 `UserId`。
- 变量名长度随作用域收缩：3 行内可用 `i` / `n` / `err`；包级与导出名必须自解释。
- 同一类型所有方法用同一个单字母 receiver；禁止 `this` / `self`。
- 单方法接口用 `-er` 命名（`Reader`、`Closer`）；不要 `IUserService` 这类前缀。

## 错误与返回值

- 错误串小写、结尾不带标点：`errors.New("user not found")`，不是 `"User Not Found."`。
- 包装用 `%w`，只在需要判定处用 `errors.Is` / `errors.As`；不要用 `%v` 丢掉错误链。
- 不要"记完日志再返回同一个错误"：日志只在边界记一次（见 `docs/api.md`）。
- 库代码禁止 `panic`、`log.Fatal`、`os.Exit`；进程级致命错误只在 `main` 处理。
- 哨兵错误放包级 `var ErrX = errors.New(...)`，不要每次调用都 `New` 一个新的。
- 不要返回"接口为 nil 但具体值非 nil"的 typed-nil；宁可返回具体类型。

## 并发与资源

- `context.Context` 永远是第一个参数、命名为 `ctx`；禁止存进结构体字段。
- 每个 goroutine 都必须有退出路径（`ctx.Done()` / 超时 / 关闭信号），禁止发出后不管。
- `defer` 在拿到资源后立即写；循环里打开的资源要么抽成函数，要么显式关闭（`defer` 会累积）。
- 循环里用 `time.After` 会泄漏定时器；改用 `time.NewTimer` 并 `Stop`。
- 共享状态必须显式保护；`sync.Mutex` 零值可用，`sync.Map` 仅在压测证实热点后使用。
- 不在循环里 `go func` 捕获循环变量，显式传参（Go 1.22 语义已修，可读性才是理由）。

## 写法与性能

- 已知大小时预分配：`make([]T, 0, n)`；循环拼接字符串用 `strings.Builder`。
- 单值转换用 `strconv`，不要 `fmt.Sprintf`（热路径差距显著）。
- 时间相等用 `Equal` 而非 `==`；浮点比较用误差范围，禁止 `==`。
- 导出标识符必须有注释且以名字开头；注释解释"为什么"，不复述代码。
- `TODO` 必须带可追踪编号：`// TODO(#123): 补限流`。
- 基准测试用 `b.Loop()`（Go 1.24+）；不要用 `time.Sleep` 做同步。

## 明确禁止

- `math/rand` 用于任何安全用途（token、密码、盐）——一律 `crypto/rand`。
- 超过 10 行的函数里出现裸 `return`（naked return）。
- 导出可变全局变量；包级 `var` 只允许不可变哨兵与注入点。
- `unsafe`、`reflect`、`interface{}`（请写 `any`）出现在业务主路径，除非注释说明理由。
