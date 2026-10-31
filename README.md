# skills

个人 Agent Skills 集合，遵循 [Agent Skills](https://agentskills.io) 开放规范，用官方 `npx skills` CLI 一条命令装进 Claude Code、Codex、Cursor、OpenCode 等 75+ 编码代理。

每个技能就是一个 `SKILL.md`：frontmatter 声明"做什么 + 何时触发"，正文写可执行流程，长文档、脚本与模板按需放进 `references/`、`scripts/`、`assets/`。

## 技能一览

| 技能 | 作用 | 何时触发 |
|---|---|---|
| [spec-mode](skills/spec-mode/SKILL.md) | 规格驱动开发：大改动按需求 → 设计 → 任务 → 实现 → 独立评审推进，全程留痕 | 跨模块 / 跨会话的复杂改动；"用 spec-mode 做…" |
| [go-web-project](skills/go-web-project/SKILL.md) | Go Web 脚手架（Gin + GORM + MySQL 8.4，MVC）+ 可执行的 AGENTS.md 工程约束 | 新项目初始化；给 Go 服务补约束；"初始化 Go Web 项目" |

## 安装

需要 Node.js ≥ 22.20，在目标项目根目录执行：

```bash
npx skills add blueberry363/skills                                       # 交互式选装
npx skills add blueberry363/skills --skill spec-mode                     # 只装一个
npx skills add blueberry363/skills --skill spec-mode -a claude-code -g   # 指定 agent + 全局
```

- 默认装到当前项目；`-g` 全局可用，`-a` 可重复指定多个 agent
- `npx skills find` 搜索技能，`npx skills list` 查看已装
- 不安装也能试用：`npx skills use blueberry363/skills@spec-mode --agent claude-code`

## spec-mode

**作用**：把一次大改动从需求带到经过独立评审的实现。六个阶段顺序推进，每阶段固定落盘，随时可中断续跑。

产物都在 `.doc/<特性名>/`：

| 文件 | 内容 |
|---|---|
| `spec.md` | 问题、目标与非目标、功能/非功能需求、验收标准 |
| `design.md` | 技术方案、ADR、接口与数据契约、兼容性与风险 |
| `tasks.md` | 原子任务队列、状态、完成证据、评审整改项 |
| `review.md` | 独立评审检查点与评审历史 |

两条硬机制：验收标准（AC）与测试要求（TR）只允许 `rule`（客观可判定，通过 / 不通过）或 `rubric`（维度 + 分档 + 阈值）两种类型；Review 是唯一成功出口，`fail` 必须把 findings 落成 `tasks.md` 的待办再回去修。

### 使用教程

1. **触发**：说"用 spec-mode 实现 <需求>"，或描述一个跨多文件、有歧义风险的改动并要求走 spec-mode。
2. **Specify / Design / Plan**：agent 先勘察代码再动笔，产出三份文档；关键歧义会当场问你，不会自行假设。
3. **Approve**：你审阅三份文档后批准或提修改意见。改需求回到 Specify、改架构回到 Design，相关下游产物重生成后才再次请求批准。
4. **Implement**：agent 按优先级逐条实现，每条写完成证据，队列清空才进入评审；实现期间不改 `design.md`。
5. **Review**：换新上下文做独立评审。`pass` 收工；`fail` 则生成整改任务，回到 Implement 再走一轮评审。

会话中断后重新打开，agent 从 `.doc/` 现有产物继续，不必重述需求。

**不适用**：单文件小修、纯调研或代码评审、只写文档不出实现、单点 debug。

## go-web-project

**作用**：两件事。一是把工程约束写成 agent 会真的执行的 `AGENTS.md`（根文件 80–180 行 + `docs/` 六篇专题：stack / api / data / testing / security / coding-standards）；二是按同一套约定生成可编译的脚手架。

基线 `Gin + GORM + MySQL 8.4`、MVC 分层（controller / service / model），含路由表、中间件、统一响应体、探针、goose 迁移、OpenAPI、Makefile、Dockerfile、compose、CI 与示例领域 `user`。

### 使用教程

**1. 新项目初始化**：让 agent 执行，或直接跑技能目录下的 `scripts/init.mjs`（安装位置随 agent 而异）。

```bash
node scripts/init.mjs ./api --module=example.com/acme/api            # 完整模板
node scripts/init.mjs ./api --module=example.com/acme/api --minimal  # 只留分层骨架
```

`--minimal` 删掉 user 示例，保留探针、统一响应与示例测试，仍是可编译可迁移的项目；`--dry-run` 只列出将写入的文件。

**2. 验证**：按顺序执行，任一步失败先修再交付。

```bash
go build ./...
gofmt -l .              # 必须无输出
go vet ./...
go test -shuffle=on ./...
go mod tidy             # 应当无 diff
```

**3. 已有仓库**：只产出约束。agent 先读 `go.mod`、目录树、Makefile、CI 与 `db/migrations/`，再增量增补 `AGENTS.md` 和 `docs/`，命令与 CI 对齐，不整篇重写。

**4. 换栈**：换 Web 框架、ORM、数据库或分层风格时读 `references/stack-swap.md`，按对照表只改对应段落。

**5. 自检**：改完模板或约束后跑 `node scripts/verify.mjs`——生成两种模式的脚手架、跑真实命令、逐条比对文档断言。

**不适用**：纯库 / SDK、CLI 工具、gRPC-only 服务，以及只要 README 或需求文档的场景。

## 更新与卸载

```bash
npx skills check            # 检查可用更新
npx skills update           # 更新已装技能
npx skills remove spec-mode # 卸载
```

## 新增技能

```
skills/<技能名>/SKILL.md     技能定义：frontmatter + 流程
           references/       长文档，按需读取
           scripts/          可执行脚本
           assets/           模板与静态资源
templates/skill/SKILL.md     新技能模板
```

1. 从 `templates/skill/SKILL.md` 复制到 `skills/<技能名>/`，目录名与 frontmatter `name` 必须同为小写 kebab-case
2. `description` 写清"做什么 + 何时触发 + 触发词"，它决定技能能否被准确加载；正文写流程，长内容放 `references/`
3. 仓库根目录运行 `npx skills add . --list`，确认新技能可被发现
4. 推送后任何人可用 `npx skills add blueberry363/skills --skill <名称>` 安装

## License

Apache-2.0
