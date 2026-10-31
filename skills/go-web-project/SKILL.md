---
name: go-web-project
description: 为 Go Web 服务初始化可编译的脚手架并建立可执行的 AGENTS.md 工程约束。按 Gin + GORM + MySQL 基线、MVC 分层（controller/service/model）生成完整骨架：路由表、中间件、controller/service/model 示例、统一响应体、探针、迁移、Makefile、Dockerfile、compose 与 CI，并产出约束分层边界、命令、错误与日志、迁移、测试、安全规则的 AGENTS.md。当用户要求初始化 Go Web 项目脚手架、为项目选定 Web 框架并落地 MVC 分层、为新项目写 AGENTS.md，或在该类服务仓库中新增接口、查询、迁移、测试时使用。不适用：纯 Go 库/SDK、CLI 工具、gRPC-only 服务、非 Go 后端，以及只需 README 或纯需求文档的场景。
---

# Go Web Project

用一份**可执行**的 AGENTS.md 把 Go Web 服务的工程约束固化下来，然后按它写代码。

本技能的价值不在文采，而在于：agent 会真的执行文件里的命令。所以每条命令必须真实、可复制、幂等，每条约束必须可判定。

## 何时使用

- 用户要求为 Go Web 项目初始化脚手架、写、改、评审 AGENTS.md；
- 在没有 AGENTS.md 的 Go 服务仓库里开始开发，需要先建立约束；
- 在既有 Go 服务中新增接口、查询、迁移、测试，需要知道"新代码放哪、怎么验"；
- 用户问 Go 项目怎么分层（MVC 还是三段式）、选 Gin 还是 Echo、用 GORM 还是 sqlc。

## 不适用

- 纯库/SDK/CLI 项目：不涉及 HTTP 服务生命周期与 DB 迁移，用通用 AGENTS.md 即可；
- gRPC-only 或消息驱动服务：把 `references/agents-template.md` 第 4 段换成传输层约定，其余可复用；
- 只需要 README、需求文档、架构散文：不加载本技能。

## 输入

- 目标仓库根目录与现有代码状态（`go.mod`、目录树、`Makefile`、CI 配置、`db/migrations`）；
- 用户的技术栈确认。默认基线为 `Gin + GORM + MySQL 8.4`、MVC 分层；偏离开此基线时先读 `references/stack-swap.md`。

## 输出

1. 目标仓库根目录的 `AGENTS.md`（根文件，80–180 行）与 `docs/` 专题文档（6 篇，每篇 20–120 行）；
2. 全新项目额外产出：可编译的脚手架（完整 33 个文件 / minimal 30 个文件，含锁定的 `go.mod`/`go.sum`，见下节）；
3. 必要的配套：`Makefile` 中的任务入口、CI 命令对齐说明。

脚手架可选，约束是必产出。用户只要约束时不要生成代码。

## 脚手架初始化（新项目）

只初始化文件，不联网、不执行 go 命令（除读取本机 Go 版本），由你随后验证。

```bash
node scripts/init.mjs <目标目录> --module=<module 路径> [--service=api] [--go=1.27.0] [--minimal] [--force] [--dry-run]
```

| 参数 | 说明 |
|---|---|
| `<目标目录>` | 必填；非空目录需加 `--force` |
| `--module` | 必填，`go.mod` 的 module 路径，如 `example.com/acme/api` |
| `--service` | 可选，二进制名，默认取目录名；小写字母/数字/连字符 |
| `--go` | 可选，默认读本机 `go env GOVERSION`，避免写出本机编译不了的版本 |
| `--minimal` | 只留分层骨架：删除 user 示例，保留探针、统一响应与示例测试 |
| `--dry-run` | 只列表将写入的文件 |

两种模式（同一份模板，minimal 用覆盖层实现，避免两份模板漂移）：

| 模式 | 文件数 | 内容 |
|---|---|---|
| 完整（默认） | 33 | 分层骨架 + `user` 增删查改示例、controller/service 测试、`users` 迁移与 OpenAPI 端点 |
| `--minimal` | 30 | 分层骨架 + `/healthz` `/readyz` 与探针测试；`service/`、`model/` 只留 `doc.go` 说明扩展顺序；占位迁移 `SELECT 1` |

minimal 模式仍然可编译、可迁移、有测试，不是空壳。

生成内容：`cmd/server/main.go`（只做装配）、`internal/{router,middleware,controller,service,model,response,apperr,config,database}`、`db/migrations`、`api/openapi.yaml`、`docs/`（stack / api / data / testing / security / coding-standards 六篇）、`Makefile`、`Dockerfile`、`compose.yaml`、`.golangci.yml`、`.env.example`、`.gitignore`、`.github/workflows/ci.yml`、`README.md`，以及由 `references/agents-template.md` 渲染出的根 `AGENTS.md`（单一事实源，不在 assets 里重复一份）。

生成后必须依次执行，任一步失败先修再交付：

```bash
go build ./...        # 依赖已锁定，无需先 tidy
gofmt -l .            # 必须无输出
go vet ./...
go test -shuffle=on ./...
go mod tidy           # 应当无 diff
```

`go test -race` 需要 cgo 与 C 工具链：本机具备才作为必跑命令，否则只在 CI 执行，并在 AGENTS.md 第 2 段标注前提。

### 一键自检

改了模板、脚手架或 AGENTS.md 之后，不要靠眼看，跑一次自检：

```bash
node scripts/verify.mjs          # 生成两种模式 → 跑全部命令 → 比对 AGENTS.md 断言
node scripts/verify.mjs --race   # 本机有 C 工具链时连竞态一起验
node scripts/verify.mjs --keep   # 失败时保留临时目录便于排查
```

它覆盖 59 项断言：技能仓库结构、两种模式的文件数与占位符、`go build`/`gofmt`/`go vet`/`go test`/`tidy` 漂移、
根文件行数与表格结构、`docs/` 索引完整性与每篇篇幅、`make` 目标是否存在、引用路径是否存在、6 项版本一致、
分层边界、统一响应体规则、编码规范（禁用包名 / 导出注释 / 无直接退出进程）、minimal 无业务残留。任一项失败即退出码 1。

## 工作流

### 1. 勘察（不改任何文件）

全新仓库先按上节生成脚手架，再跳到第 2 步；已有仓库先读这五样再动笔：

| 读什么 | 确认什么 |
|---|---|
| `go.mod` | `go` 指令版本、module 路径、直接依赖 |
| 目录树 | `cmd/` `internal/` `db/` 是否已存在，真实分层与命名 |
| `Makefile` / `Taskfile` | 已有任务入口，避免另造一套命令 |
| `.github/workflows/*` | CI 实际执行的命令 = AGENTS.md 要写的命令 |
| `db/migrations/` | 迁移工具（goose / migrate / atlas）与命名习惯 |

已有 `AGENTS.md` 时，只做增量修订，不要整篇重写。

### 2. 选段

以 `references/agents-template.md` 为基线（根文件承载高频约束，专题细节在 `docs/`），逐段判断：

- 仓库里不存在的环节（如没有 OpenAPI、没有集成测试）→ **删除整段**，不要留空标题；
- 仓库有特殊机制的环节 → 就地改写，保留"可判定"的语气。

### 3. 落地

- 复制 `references/agents-template.md` 到仓库根目录 `AGENTS.md`；
- 替换全部占位符：`{{MODULE}}`（go.mod module 路径）、`{{SERVICE}}`（二进制名）、`{{GO_VERSION}}`（go 指令）、`{{GO_MINOR}}`（镜像标签）；
- 目录树、命令、字段名按第 1 步勘察结果改为真实值；
- 删除模板顶部的注释块，不留 `{{` 残留。

### 4. 验证（不可跳过）

逐条执行 AGENTS.md 中列出的命令。跑不通的命令只有两种处理：改成真实命令，或删除。

```bash
go build ./...                      # 先证能编译
gofmt -l .                          # 必须无输出
go vet ./...
go test -shuffle=on ./...
CGO_ENABLED=1 go test -race ./...   # 有 C 工具链时才作为必跑
```

同时核对：`go.mod` 的版本号与文中一致；目录树与 `ls` 结果一致；`make check` 若存在则与 CI 命令一致。

### 5. 同步

- README 保持"给人看"的定位，不重复 AGENTS.md 内容；两边冲突时以 AGENTS.md 为准并在 PR 中修正 README；
- 长文（架构、领域边界、ADR）放 `docs/`，在 AGENTS.md 末段写相对路径索引，正文不展开。

## 内容分布（根文件 + docs/）

| # | 主题 | 位置 | 一句话要求 |
|---|---|---|---|
| 1 | 技术栈与配置 | `docs/stack.md` | 选型写死 + 配置项清单，说明"不要替换"的理由 |
| 2 | 命令 | 根文件 | 唯一任务入口表 + **完成定义** |
| 3 | MVC 分层与依赖 | 根文件 | 层职责红线 + 新代码放哪 + 常见改动落点 |
| 4 | HTTP 约定 | `docs/api.md` | 路由、绑定、超时、中间件顺序 |
| 5 | 错误与响应 | `docs/api.md` | 包装格式、状态码映射、业务码表、日志字段 |
| 6 | 数据与迁移 | `docs/data.md` | 连接、事务边界、迁移为 schema 唯一来源 |
| 7 | 测试 | `docs/testing.md` | 怎么写、测到什么程度、bug 先复现 |
| 8 | 安全 | `docs/security.md` | 校验位置、密钥来源、出站超时、漏洞扫描 |
| 9 | 提交与 PR | 根文件 | 分支、commit、PR 前置命令 |
| 10 | 禁止清单 | 根文件 | 8–12 条 Do-Not，可逐条判定 |
| 11 | 文档索引 | 根文件 | "何时读哪篇" + 关键文件，替代长文 |
| 12 | 编码规范 | `docs/coding-standards.md` | 只列可判定的语言级规则 |

分工原则：**每个 PR 都要看一眼的留在根文件，只有改某类东西才需要的抽到 `docs/`**；抽离后必须在根文件留"何时读"索引，否则 agent 不会主动打开。

完整结构原理、写作规则与反模式见 `references/authoring-guide.md`；需要换框架、换 ORM、换数据库或换分层风格时见 `references/stack-swap.md`。

## 三条硬规则

1. **单一事实源**：命令的真实定义在 `Makefile`/CI，AGENTS.md 只引用、不对文；两边不一致以 Makefile 为准并立即修正 AGENTS.md。
2. **命令必须真跑通**：未经本机执行的命令不得写入。失效命令会让 agent 反复失败，比没有指令更糟。
3. **唯一路径**：一个任务只给一种做法。禁止写"也可以用 X"、"或者用 Y"，那等于把决策成本推回给 agent。

## 交付前检查清单

- [ ] 根文件 80–180 行；`docs/` 每篇 20–120 行；每段 ≤ 12 行
- [ ] 无 `{{` 占位符残留
- [ ] 每条命令实测通过
- [ ] `go` 指令版本与 `go.mod` 一致
- [ ] 目录树与真实结构一致
- [ ] 与 CI 命令一致
- [ ] `node scripts/verify.mjs` 全绿（59 项断言）
- [ ] 禁止清单中每条都能被 review 判定"违反/未违反"

## 参考

- `scripts/init.mjs`：脚手架生成器（渲染 `assets/template/` 并生成 AGENTS.md），也可作为库 `import { generate }`
- `scripts/verify.mjs`：一致性自检（两种模式 + 命令 + 文档断言，47 项）
- `assets/template/`：完整脚手架模板树，含 `{{MODULE}}`/`{{SERVICE}}`/`{{GO_VERSION}}`/`{{GO_MINOR}}` 占位符
- `assets/minimal/`：minimal 覆盖层（`remove.txt` 删除清单 + `files/` 差异文件）
- `references/agents-template.md`：AGENTS.md 模板（生成器也用它渲染项目级 AGENTS.md）
- `references/authoring-guide.md`：AGENTS.md 的生效机制、内容分层、根文件与 docs/ 的分工、写作规则与反模式
- `references/stack-swap.md`：换 Web 框架 / ORM / 数据库 / 分层风格时需改动的段落对照表
