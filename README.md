# blueberry363/skills

个人 Agent Skills 集合，遵循 Agent Skills 开放规范，使用官方 `npx skills` CLI 可一键安装到 Claude Code、Codex、Cursor、OpenCode 等 75+ 编码代理。

## 可用技能

| 技能 | 描述 |
| --- | --- |
| [spec-mode](skills/spec-mode/SKILL.md) | 规格驱动开发工作流 —— 从需求澄清、验收标准、技术设计、任务规划到实现、独立评审与修复，端到端管理复杂仓库变更，产出可追溯的规格制品。 |

## 安装

需要 Node.js ≥ 22.20。在目标项目根目录执行：

```bash
# 安装单个技能（完整 URL）
npx skills add https://github.com/blueberry363/skills.git --skill spec-mode

# 简写形式（等效）
npx skills add blueberry363/skills --skill spec-mode

# 交互式选择安装全部技能
npx skills add blueberry363/skills

# 指定目标 agent（可多个 -a）与全局安装（-g）
npx skills add blueberry363/skills --skill spec-mode -a claude-code -g
```

补充说明：

- 默认安装到当前项目的技能目录；`-g` 全局安装后跨项目可用
- `npx skills find` 浏览技能；`npx skills list` 查看已安装技能

## 更新与卸载

```bash
npx skills update spec-mode
npx skills remove spec-mode
```

## 新增技能

1. 复制 `templates/skill/` 到 `skills/<你的技能名>/`；目录名用小写 kebab-case，且必须与 frontmatter 的 `name` 完全一致
2. 填写 SKILL.md：`description` 写清"做什么 + 何时触发 + 触发词"（决定技能触发质量）；核心流程写正文；长文档放 `references/`、脚本放 `scripts/`、静态资源放 `assets/`
3. 本地校验：在仓库根目录运行 `npx skills@latest add . --list`，确认新技能被发现
4. 提交并推送到 GitHub 后，即可被任何人通过 `npx skills add blueberry363/skills --skill <名称>` 安装

## License

Apache-2.0
