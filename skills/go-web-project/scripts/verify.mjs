#!/usr/bin/env node
// 一致性自检：生成两种模式的脚手架，跑真实命令，并把 AGENTS.md 的断言逐条比对代码。
//
// 用法：
//   node scripts/verify.mjs [--keep] [--race] [--module=example.com/verify/api] [--dir=<已有仓库>]
//
//   --keep    保留临时目录，便于失败后排查
//   --race    额外跑 go test -race（需 C 工具链）
//   --module  指定自检用的 module 路径
//   --dir     对已有仓库只跑 AGENTS.md 断言（不生成脚手架；缺失的模板文件对应断言跳过）
//
// 退出码：0 全部通过；1 存在失败项。

import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdtemp, readdir, readFile, rm } from "node:fs/promises";
import { existsSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { generate } from "./init.mjs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const skillRoot = path.resolve(__dirname, "..");
const templateRoot = path.join(skillRoot, "assets", "template");
const minimalRoot = path.join(skillRoot, "assets", "minimal");
const agentsTemplatePath = path.join(skillRoot, "references", "agents-template.md");

const argv = process.argv.slice(2);
const keep = argv.includes("--keep");
const withRace = argv.includes("--race");
const moduleArg = argv.find((a) => a.startsWith("--module="));
const modulePath = moduleArg ? moduleArg.slice("--module=".length) : "example.com/verify/api";
const dirArg = argv.find((a) => a.startsWith("--dir="));

let passes = 0;
let failures = 0;
let skips = 0;
const failureMessages = [];

function ok(message) {
  passes += 1;
  console.log(`  ✓ ${message}`);
}

function bad(message) {
  failures += 1;
  failureMessages.push(message);
  console.log(`  ✗ ${message}`);
}

function skip(message) {
  skips += 1;
  console.log(`  - ${message}`);
}

function header(title) {
  console.log(`\n▶ ${title}`);
}

function run(cmd, cmdArgs, cwd) {
  try {
    const out = execFileSync(cmd, cmdArgs, {
      cwd,
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    });
    return { ok: true, out: out.trim() };
  } catch (err) {
    const out = `${err.stdout ?? ""}${err.stderr ?? ""}`.trim();
    return { ok: false, out, code: err.status ?? 1 };
  }
}

async function sha256(file) {
  return createHash("sha256").update(await readFile(file)).digest("hex");
}

async function walk(dir) {
  const entries = await readdir(dir, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const abs = path.join(dir, entry.name);
    if (entry.isDirectory()) files.push(...(await walk(abs)));
    else if (entry.isFile()) files.push(abs);
  }
  return files;
}

function firstMatch(text, re) {
  const m = text ? text.match(re) : null;
  return m ? m[1] : null;
}

// 检查 Markdown 表格结构：连续 | 行成块，第 2 行必须是分隔行，且各行列数一致。
function inspectTables(markdown) {
  const lines = markdown.replace(/\r\n/g, "\n").split("\n");
  const blocks = [];
  let current = [];
  for (const line of lines) {
    if (line.startsWith("|")) current.push(line);
    else if (current.length > 0) {
      blocks.push(current);
      current = [];
    }
  }
  if (current.length > 0) blocks.push(current);

  const issues = [];
  blocks.forEach((block, index) => {
    const label = `表 ${index + 1}`;
    if (block.length < 2 || !/^\|[\s:|-]+\|$/.test(block[1])) {
      issues.push(`${label} 缺少分隔行`);
      return;
    }
    const columns = block[0].split("|").length - 2;
    block.forEach((row, rowIndex) => {
      if (row.split("|").length - 2 !== columns) {
        issues.push(`${label} 第 ${rowIndex + 1} 行列数不符`);
      }
    });
  });

  return { count: blocks.length, issues };
}

// ── 技能仓库自身结构 ────────────────────────────────────────────────

async function verifySkillLayout() {
  header("技能仓库结构");

  if (existsSync(agentsTemplatePath)) ok("references/agents-template.md 存在");
  else bad("references/agents-template.md 缺失");

  for (const p of [templateRoot, path.join(minimalRoot, "remove.txt"), path.join(minimalRoot, "files")]) {
    const rel = path.relative(skillRoot, p).split(path.sep).join("/");
    if (existsSync(p)) ok(`${rel} 存在`);
    else bad(`${rel} 缺失`);
  }

  // remove.txt 的每个条目都必须真的存在于完整模板里，否则是失效清单（静默不生效）。
  const removeLines = (await readFile(path.join(minimalRoot, "remove.txt"), "utf8"))
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter((l) => l.length > 0 && !l.startsWith("#"));

  const { plan: fullPlan } = await generate({
    target: path.join(os.tmpdir(), "goweb-verify-layout-probe"),
    module: modulePath,
    service: "api",
    dryRun: true,
  });
  const fullRels = new Set(fullPlan.map((p) => p.rel));

  const stale = removeLines.filter((rel) => !fullRels.has(rel));
  if (stale.length === 0) ok(`remove.txt 的 ${removeLines.length} 个条目都能在模板中找到`);
  else bad(`remove.txt 有失效条目：${stale.join(", ")}`);

  // 覆盖层文件要么替换已有文件，要么是有意新增；新增需在输出中可见。
  const overlayFiles = await walk(path.join(minimalRoot, "files"));
  const additions = overlayFiles
    .map((abs) => path.relative(path.join(minimalRoot, "files"), abs).split(path.sep).join("/"))
    .filter((rel) => !fullRels.has(rel));
  ok(`覆盖层 ${overlayFiles.length} 个文件（新增 ${additions.length} 个：${additions.join(", ") || "无"}）`);

  // SKILL.md 声称的文件数必须与实际生成一致，防止文档漂移。
  const minPlan = await generate({
    target: path.join(os.tmpdir(), "goweb-verify-layout-probe-min"),
    module: modulePath,
    service: "api",
    minimal: true,
    dryRun: true,
  });
  const skillMd = await readFile(path.join(skillRoot, "SKILL.md"), "utf8");
  const docFull = firstMatch(skillMd, /完整（默认）\s*\|\s*(\d+)/);
  const docMin = firstMatch(skillMd, /`--minimal`\s*\|\s*(\d+)/);
  if (docFull && docMin && Number(docFull) === fullPlan.length && Number(docMin) === minPlan.plan.length) {
    ok(`SKILL.md 文件数与实际一致（完整 ${fullPlan.length} / minimal ${minPlan.plan.length}）`);
  } else {
    bad(`SKILL.md 文件数漂移：文档 完整 ${docFull} / minimal ${docMin}，实际 ${fullPlan.length} / ${minPlan.plan.length}`);
  }
}

// ── 文档断言 vs 真实代码 ───────────────────────────────────────────

async function verifyDocClaims(dir, agents, { minimal, existing = false } = {}) {
  header(`文档断言（${existing ? "已有仓库" : minimal ? "minimal" : "完整"}）`);

  const tryRead = async (p) => {
    try {
      return await readFile(p, "utf8");
    } catch {
      return null;
    }
  };

  const makefile = await tryRead(path.join(dir, "Makefile"));
  const gomod = await tryRead(path.join(dir, "go.mod"));
  const ci = await tryRead(path.join(dir, ".github", "workflows", "ci.yml"));
  const readme = await tryRead(path.join(dir, "README.md"));
  const openapi = await tryRead(path.join(dir, "api", "openapi.yaml"));
  const golangci = await tryRead(path.join(dir, ".golangci.yml"));
  const compose = await tryRead(path.join(dir, "compose.yaml"));

  // 版本与约定可能写在 docs/ 里，断言要覆盖「根文件 + 全部专题文档」
  const docDir = path.join(dir, "docs");
  const docFiles = existsSync(docDir) ? (await walk(docDir)).filter((f) => f.endsWith(".md")) : [];
  const docContents = await Promise.all(docFiles.map((f) => readFile(f, "utf8")));
  const claims = [agents, ...docContents].join("\n");

  // 1) AGENTS.md 提到的 make 目标都要真实存在
  if (!makefile) {
    skip("make 目标检查（Makefile 不存在）");
  } else {
    const targets = [...new Set([...(agents + "\n" + (readme ?? "")).matchAll(/make ([a-z][a-z0-9-]+)/g)].map((m) => m[1]))];
    const missingTargets = targets.filter((t) => !new RegExp(`^${t}:`, "m").test(makefile));
    if (missingTargets.length === 0) ok(`make 目标全部存在（${targets.length} 个）`);
    else bad(`make 目标缺失：${missingTargets.join(", ")}`);
  }

  // 2) AGENTS.md 提到的路径都要存在（允许 `pkg.Func` 这类符号引用）
  const paths = [...new Set([...agents.matchAll(/`((?:cmd|internal|db|api|docs)\/[A-Za-z0-9_./-]+\.?[A-Za-z0-9_/-]*)`/g)].map((m) => m[1]))];
  const missingPaths = paths.filter((p) => {
    const trimmed = p.replace(/\/$/, "");
    if (existsSync(path.join(dir, trimmed))) return false;
    if (existsSync(path.join(dir, `${trimmed}.go`))) return false;
    return !existsSync(path.join(dir, path.dirname(trimmed))); // 目录在即视为符号引用
  });
  if (missingPaths.length === 0) ok(`引用路径全部存在（${paths.length} 个）`);
  else bad(`引用路径缺失：${missingPaths.join(", ")}`);

  // 3) 版本断言与依赖清单一致
  const versionChecks = [
    {
      name: "gin",
      doc: firstMatch(claims, /gin`? v(\d+\.\d+)/),
      mod: firstMatch(gomod, /gin-gonic\/gin v(\d+\.\d+)\./),
    },
    {
      name: "gorm",
      doc: firstMatch(claims, /gorm` v(\d+\.\d+)/),
      mod: firstMatch(gomod, /gorm\.io\/gorm v(\d+\.\d+)\./),
    },
    {
      name: "MySQL",
      doc: firstMatch(claims, /MySQL (\d+\.\d+)/),
      mod: firstMatch(compose, /mysql:(\d+\.\d+)/),
    },
    {
      name: "OpenAPI",
      doc: firstMatch(claims, /OpenAPI (\d+\.\d+)/),
      mod: firstMatch(openapi, /openapi: (\d+\.\d+)/),
    },
    {
      name: "golangci-lint",
      doc: firstMatch(claims, /golangci-lint` v(\d+)/),
      mod: firstMatch(golangci, /version: "(\d+)"/),
    },
  ];
  for (const c of versionChecks) {
    if (!c.doc || !c.mod) {
      if (existing) skip(`${c.name} 版本一致性（${c.doc ? "实现" : "文档"}侧未声明）`);
      else bad(`${c.name} 版本基线无法解析（文档=${c.doc}，实现=${c.mod}）`);
    } else if (c.doc === c.mod) ok(`${c.name} 版本一致（${c.doc}）`);
    else bad(`${c.name} 版本不一致：文档 ${c.doc} / 实现 ${c.mod}`);
  }

  const gooseDoc = firstMatch(claims, /goose` v(\d+)/);
  const gooseMake = firstMatch(makefile, /goose\/v(\d+)\/cmd/);
  if (!gooseDoc || !gooseMake) {
    if (existing) skip("goose 大版本一致性（文档或 Makefile 未声明）");
    else bad(`goose 版本不一致：文档 v${gooseDoc} / Makefile v${gooseMake}`);
  } else if (gooseDoc === gooseMake) ok(`goose 大版本一致（v${gooseDoc}）`);
  else bad(`goose 版本不一致：文档 v${gooseDoc} / Makefile v${gooseMake}`);

  // 4) 分层边界：controller 不碰 gorm，model 不碰 gin，service 不碰 http
  const internalDir = path.join(dir, "internal");
  const goFiles = existsSync(internalDir)
    ? (await walk(internalDir)).filter((f) => f.endsWith(".go") && !f.endsWith("_test.go"))
    : [];

  if (goFiles.length === 0) {
    skip("分层边界检查（internal/ 无 Go 文件）");
  } else {
    const violations = [];
    for (const file of goFiles) {
      const rel = path.relative(dir, file).split(path.sep).join("/");
      const src = await readFile(file, "utf8");
      if (rel.startsWith("internal/controller/") && src.includes("gorm.io/gorm")) violations.push(`${rel} 引入 gorm`);
      if (rel.startsWith("internal/model/") && src.includes("gin-gonic/gin")) violations.push(`${rel} 引入 gin`);
      if (rel.startsWith("internal/service/") && /gin-gonic\/gin|"net\/http"/.test(src)) {
        violations.push(`${rel} 引入 gin/net/http`);
      }
    }
    if (violations.length === 0) ok("分层边界未被破坏（controller/model/service）");
    else bad(`分层越界：${violations.join("; ")}`);
  }

  // 5) 统一响应体规则：除探针外不得直接 c.JSON
  const controllerFiles = goFiles.filter((f) => f.includes(`${path.sep}controller${path.sep}`));
  if (controllerFiles.length === 0) {
    skip("统一响应体检查（无 controller 文件）");
  } else {
    const directJSON = [];
    for (const file of controllerFiles) {
      const base = path.basename(file);
      if (base === "health.go") continue;
      if ((await readFile(file, "utf8")).includes("c.JSON")) directJSON.push(base);
    }
    if (directJSON.length === 0) ok("除探针外没有直接 c.JSON");
    else bad(`直接写响应：${directJSON.join(", ")}`);
  }

  // 6) CI 只调用 make 入口，命令定义以 Makefile 为准
  if (!ci) {
    skip("CI 检查（.github/workflows/ci.yml 不存在）");
  } else {
    if (agents.includes("CI 执行同一套命令")) {
      if (ci.includes("make check") && ci.includes("make test-race")) ok("CI 通过 make check / make test-race 执行，与 Makefile 同源");
      else bad("CI 未调用 make check 与 make test-race");
    }
    if (agents.includes("test-race") && !ci.includes("make test-race")) bad("文档提到 test-race，但 CI 未执行");
    if (agents.includes("集成测试编译检查") && !ci.includes("go vet -tags=integration")) bad("文档提到集成测试编译检查，但 CI 未执行");
    if (agents.includes("漏洞扫描") && !ci.includes("govulncheck")) bad("文档提到漏洞扫描，但 CI 未执行");
  }

  if (!makefile) skip("make check 组成检查（Makefile 不存在）");
  else if (/^check: fmt-check vet test$/m.test(makefile)) ok("make check 的组成与文档一致");
  else bad("make check 的组成与文档描述不一致");

  // 8) 文档索引指向的文件必须存在，且每篇篇幅可控
  const docLinks = [...new Set([...agents.matchAll(/`(docs\/[A-Za-z0-9_./-]+\.md)`/g)].map((m) => m[1]))];
  const missingDocs = docLinks.filter((p) => !existsSync(path.join(dir, p)));
  if (docLinks.length === 0) skip("文档索引检查（AGENTS.md 未引用 docs/ 文件）");
  else if (missingDocs.length === 0) ok(`文档索引指向的 ${docLinks.length} 个文件都存在`);
  else bad(`文档索引指向不存在的文件：${missingDocs.join(", ")}`);

  const docDirFiles = docFiles;
  const oversized = [];
  for (const file of docDirFiles) {
    const count = (await readFile(file, "utf8")).replace(/\r\n/g, "\n").replace(/\n$/, "").split("\n").length;
    if (count < 20 || count > 120) oversized.push(`${path.basename(file)} ${count} 行`);
  }
  if (docDirFiles.length === 0) {
    if (existing) skip("docs/ 篇幅检查（docs/ 不存在或为空）");
    else bad("docs/ 为空，但 AGENTS.md 依赖它承载专题细节");
  } else if (oversized.length === 0) ok(`docs/ 共 ${docDirFiles.length} 篇，均在 20–120 行`);
  else bad(`docs/ 篇幅异常：${oversized.join(", ")}`);

  // 7) minimal 模式不得残留业务示例
  if (minimal) {
    const residue = [];
    for (const file of await walk(dir)) {
      if (!/\.(go|sql|ya?ml|md)$/.test(file)) continue;
      const src = await readFile(file, "utf8");
      if (/UserStore|CreateUser|\/users/.test(src)) residue.push(path.relative(dir, file).split(path.sep).join("/"));
    }
    if (residue.length === 0) ok("minimal 无 user 业务残留");
    else bad(`minimal 残留业务示例：${residue.join(", ")}`);
  }
}

// ── OpenAPI 契约 vs 路由注册 ─────────────────────────────────────────

// 解析 2 空格缩进的 path 键与 4 空格缩进的 HTTP method，忽略 components 等其他节点。
function parseOpenAPIPaths(yaml) {
  const paths = [];
  const lines = yaml.replace(/\r\n/g, "\n").split("\n");
  let current = null;
  for (const line of lines) {
    const pm = line.match(/^  (\/[A-Za-z0-9_{}./-]+):$/);
    if (pm) {
      current = pm[1];
      continue;
    }
    const mm = line.match(/^    (get|post|put|patch|delete):$/);
    if (mm && current) paths.push([mm[1].toUpperCase(), current]);
  }
  return paths;
}

// 期望集与 internal/router 的注册一一对应；模板改动路由时必须同步这里。
async function verifyOpenAPIContract(dir, { minimal }) {
  header("OpenAPI 契约");

  const openapi = await readFile(path.join(dir, "api", "openapi.yaml"), "utf8");
  const expected = minimal
    ? [["GET", "/healthz"], ["GET", "/readyz"]]
    : [
        ["GET", "/healthz"],
        ["GET", "/readyz"],
        ["POST", "/api/v1/users"],
        ["GET", "/api/v1/users"],
        ["GET", "/api/v1/users/{id}"],
      ];

  const key = ([m, p]) => `${m} ${p}`;
  const got = parseOpenAPIPaths(openapi).map(key).sort();
  const want = expected.map(key).sort();
  if (got.join(",") === want.join(",")) ok(`paths 与路由注册一致（${got.length} 项）`);
  else bad(`paths 与路由注册不一致：OpenAPI=[${got.join(", ")}] 期望=[${want.join(", ")}]`);

  const flatError = /required:\s*\[\s*code\s*,\s*message\s*\]/.test(openapi) && /code:\n\s+type:\s+integer/.test(openapi);
  if (flatError) ok("Error schema 与统一响应体一致（平铺，code 为 integer）");
  else bad("Error schema 与统一响应体不符（需平铺 code/message，code 为 integer）");

  if (!minimal) {
    if (openapi.includes('"499"')) ok("契约包含 499 客户端断开响应");
    else bad("契约缺少 499 客户端断开响应");
  }
}

// ── 配置面与本地启动：config.go / .env.example / docs/stack.md / Makefile / README ──

async function verifyConfigSurface(dir) {
  header("配置与本地启动");

  const configSrc = await readFile(path.join(dir, "internal", "config", "config.go"), "utf8");
  const envExample = await readFile(path.join(dir, ".env.example"), "utf8");
  const stack = await readFile(path.join(dir, "docs", "stack.md"), "utf8");
  const makefile = await readFile(path.join(dir, "Makefile"), "utf8");
  const readme = await readFile(path.join(dir, "README.md"), "utf8");
  const dockerignore = await readFile(path.join(dir, ".dockerignore"), "utf8");

  const keysOf = (re, text) => new Set([...text.matchAll(re)].map((m) => m[1]));
  const fromCode = keysOf(/"([A-Z][A-Z0-9_]{2,})"/g, configSrc);
  const fromEnv = keysOf(/^([A-Z][A-Z0-9_]*)=/gm, envExample);
  const fromDoc = keysOf(/^\| `([A-Z][A-Z0-9_]+)` \|/gm, stack);

  const diff = (a, b) => [...a].filter((k) => !b.has(k));
  const problems = [
    ...diff(fromCode, fromEnv).map((k) => `${k} 缺少于 .env.example`),
    ...diff(fromEnv, fromCode).map((k) => `${k} 未被 config 读取`),
    ...diff(fromCode, fromDoc).map((k) => `${k} 缺少于 docs/stack.md 配置表`),
    ...diff(fromDoc, fromCode).map((k) => `${k} 在配置表中但 config 未读取`),
  ];
  if (problems.length === 0) ok(`配置项三处一致（${fromCode.size} 项：config / .env.example / docs/stack.md）`);
  else bad(`配置项不一致：${problems.join("; ")}`);

  if (/^-include \.env$/m.test(makefile) && /^export$/m.test(makefile)) ok("Makefile 加载并导出 .env");
  else bad("Makefile 未加载 .env（缺少 -include .env 或 export）");

  if (/^DATABASE_URL=root:/m.test(envExample)) bad(".env.example 的 DATABASE_URL 使用了 root 账号");
  else ok(".env.example 的 DATABASE_URL 未使用 root");

  if (readme.includes("make run") && !readme.includes("go run ./cmd/server")) ok("README 快速开始使用 make run（会加载 .env）");
  else bad("README 仍用 go run ./cmd/server 启动，不会加载 .env");

  if (/^\.env$/m.test(dockerignore)) ok(".dockerignore 排除 .env");
  else bad(".dockerignore 未排除 .env，构建上下文会携带本地配置");
}

// ── 编码规范（docs/coding-standards.md）中可机器判定的部分 ────────────

async function verifyCodeStyle(dir) {
  header("编码规范（docs/coding-standards.md）");

  const isVendored = (f) => f.includes(`${path.sep}vendor${path.sep}`);
  const goFiles = (await walk(dir)).filter((f) => f.endsWith(".go") && !f.endsWith("_test.go") && !isVendored(f));
  if (goFiles.length === 0) {
    skip("编码规范检查（无 Go 文件）");
    return;
  }

  // 12.1：包名不得是 util/common/helper 这类无主包
  const banned = /^(util|utils|common|helper|helpers)$/;
  const packages = [...new Set(goFiles.map((f) => path.basename(path.dirname(f))))];
  const bannedPackages = packages.filter((name) => banned.test(name));
  if (bannedPackages.length === 0) ok(`无 util/common/helper 包（${packages.length} 个包）`);
  else bad(`出现禁用包名：${bannedPackages.join(", ")}`);

  // 12.4：导出标识符必须有注释
  const undocumented = [];
  for (const file of goFiles) {
    const lines = (await readFile(file, "utf8")).split("\n");
    lines.forEach((line, index) => {
      if (!/^(func|type|var|const) [A-Z]/.test(line)) return;
      const prev = index > 0 ? lines[index - 1].trim() : "";
      const documented = prev.startsWith("//") || prev.startsWith("/*") || prev.endsWith("*/");
      if (!documented) undocumented.push(`${path.basename(file)}:${index + 1}`);
    });
  }
  if (undocumented.length === 0) ok("导出标识符均有注释");
  else bad(`未注释的导出标识符：${undocumented.join(", ")}`);

  // 12.2：库代码禁止直接退出进程
  const exits = [];
  for (const file of goFiles) {
    const rel = path.relative(dir, file).split(path.sep).join("/");
    if (rel.startsWith("cmd/")) continue; // 进程入口允许 os.Exit
    if (/\bos\.Exit\(|\blog\.Fatal/.test(await readFile(file, "utf8"))) exits.push(path.basename(file));
  }
  if (exits.length === 0) ok("库代码无 os.Exit / log.Fatal");
  else bad(`库代码里直接退出进程：${exits.join(", ")}`);

  // request_id 键名字面量只能出现在 internal/reqctx，杜绝 middleware 与 response 各写一份后漂移。
  const keyHolders = [];
  for (const file of await walk(dir)) {
    if (!file.endsWith(".go") || isVendored(file)) continue;
    if ((await readFile(file, "utf8")).includes('"request_id"')) {
      keyHolders.push(path.relative(dir, file).split(path.sep).join("/"));
    }
  }
  const stray = keyHolders.filter((f) => !f.startsWith("internal/reqctx/"));
  if (keyHolders.length === 0) skip("request_id 键名单点检查（代码未使用该键）");
  else if (stray.length === 0) ok(`request_id 键名字面量只在 internal/reqctx（${keyHolders.length} 处）`);
  else bad(`request_id 键名字面量越界：${stray.join(", ")}`);
}

// ── 单模式整体校验 ─────────────────────────────────────────────────

async function verifyMode(label, minimal) {
  header(`${label}模式`);

  const dir = await mkdtemp(path.join(os.tmpdir(), `goweb-verify-${minimal ? "min" : "full"}-`));
  try {
    const { plan } = await generate({ target: dir, module: modulePath, service: "api", minimal });
    ok(`生成 ${plan.length} 个文件`);

    const leftovers = plan.filter((p) => /\{\{[A-Z_]+\}\}/.test(p.body)).map((p) => p.rel);
    if (leftovers.length === 0) ok("无占位符残留");
    else bad(`占位符残留：${leftovers.join(", ")}`);

    const agents = await readFile(path.join(dir, "AGENTS.md"), "utf8");
    const lines = agents.replace(/\r\n/g, "\n").replace(/\n$/, "").split("\n").length;
    if (lines >= 80 && lines <= 180) ok(`AGENTS.md ${lines} 行（根文件预算 80–180）`);
    else bad(`AGENTS.md ${lines} 行，超出根文件 80–180 预算`);

    const tables = inspectTables(agents);
    if (tables.issues.length === 0) ok(`Markdown 表格结构正常（${tables.count} 张）`);
    else bad(`表格结构异常：${tables.issues.join("; ")}`);

    const modBefore = await sha256(path.join(dir, "go.mod"));
    const sumBefore = await sha256(path.join(dir, "go.sum"));

    const build = run("go", ["build", "./..."], dir);
    build.ok ? ok("go build ./...") : bad(`go build 失败：${build.out.split("\n")[0]}`);

    const fmt = run("gofmt", ["-l", "."], dir);
    if (!fmt.ok) bad(`gofmt 执行失败：${fmt.out.split("\n")[0]}`);
    else if (fmt.out === "") ok("gofmt -l 无输出");
    else bad(`未格式化：${fmt.out.replace(/\n/g, ", ")}`);

    const vet = run("go", ["vet", "./..."], dir);
    vet.ok ? ok("go vet ./...") : bad(`go vet 失败：${vet.out.split("\n")[0]}`);

    const vetIntegration = run("go", ["vet", "-tags=integration", "./..."], dir);
    vetIntegration.ok ? ok("go vet -tags=integration ./...（集成测试可编译）") : bad(`集成测试编译失败：${vetIntegration.out.split("\n")[0]}`);
    skip("集成测试执行（需 docker 与真实数据库，不在自检范围）");

    const test = run("go", ["test", "-shuffle=on", "./..."], dir);
    test.ok ? ok("go test -shuffle=on ./...") : bad(`测试失败：${test.out.split("\n")[0]}`);

    if (withRace) {
      const race = run("go", ["test", "-race", "-shuffle=on", "./..."], dir);
      race.ok ? ok("go test -race ./...") : bad(`竞态测试失败：${race.out.split("\n")[0]}`);
    } else {
      skip("go test -race（未加 --race，需 C 工具链）");
    }

    const tidy = run("go", ["mod", "tidy"], dir);
    if (!tidy.ok) bad(`go mod tidy 失败：${tidy.out.split("\n")[0]}`);
    else {
      const stable = (await sha256(path.join(dir, "go.mod"))) === modBefore && (await sha256(path.join(dir, "go.sum"))) === sumBefore;
      stable ? ok("go mod tidy 无漂移") : bad("go mod tidy 改动了 go.mod/go.sum（依赖未锁定）");
    }

    if (!minimal) {
      // 行为回归：分页越界、绑定错误文案。测试在模板里，这里确认它们真的执行且通过。
      const names = [
        "TestListUsersRejectsOutOfRange",
        "TestListUsersMaxLimit",
        "TestCreateUserBindErrorMessage",
        "TestListRejectsOutOfRange",
      ];
      const probe = run("go", ["test", "-count=1", "-v", "-run", names.join("|"), "./..."], dir);
      const missing = names.filter((n) => !probe.out.includes(`--- PASS: ${n} `));
      if (probe.ok && missing.length === 0) ok(`行为回归测试通过（${names.length} 项：分页越界、绑定错误文案）`);
      else bad(`行为回归测试失败或未运行：${probe.ok ? missing.join(", ") : probe.out.split("\n")[0]}`);
    } else {
      skip("行为回归测试（minimal 无 user 示例）");
    }

    await verifyDocClaims(dir, agents, { minimal });
    await verifyOpenAPIContract(dir, { minimal });
    await verifyConfigSurface(dir);
    await verifyCodeStyle(dir);
  } finally {
    if (keep) console.log(`  · 保留目录：${dir}`);
    else await rm(dir, { recursive: true, force: true });
  }
}

// ── 已有仓库模式：只校验 AGENTS.md 断言，不生成脚手架 ─────────────────

async function verifyExistingRepo(dir) {
  header("已有仓库 AGENTS.md 断言");

  const agentsPath = path.join(dir, "AGENTS.md");
  if (!existsSync(agentsPath)) {
    bad("AGENTS.md 不存在");
    return;
  }

  const agents = await readFile(agentsPath, "utf8");
  await verifyDocClaims(dir, agents, { minimal: false, existing: true });
  await verifyCodeStyle(dir);
  skip("OpenAPI 契约与路由一致性（期望集按脚手架模板定义，--dir 模式跳过）");
  skip("配置面与本地启动检查（模板特定，--dir 模式跳过）");
}

async function main() {
  console.log("Go Web 脚手架一致性自检");
  console.log(`  技能：${skillRoot}`);
  console.log(`  模块：${modulePath}`);

  if (!existsSync(path.join(templateRoot, "go.mod"))) {
    console.error("错误：assets/template/go.mod 缺失，无法自检");
    process.exit(1);
  }

  if (dirArg) {
    const dir = path.resolve(dirArg.slice("--dir=".length));
    console.log(`  目录：${dir}`);
    await verifyExistingRepo(dir);
  } else {
    await verifySkillLayout();
    await verifyMode("完整", false);
    await verifyMode("minimal", true);
  }

  console.log("\n────────────────────────────");
  console.log(`通过 ${passes} 项，失败 ${failures} 项，跳过 ${skips} 项`);
  if (failures > 0) {
    console.log("失败明细：");
    for (const m of failureMessages) console.log(`  - ${m}`);
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(`错误：${err.message}`);
  process.exit(1);
});
