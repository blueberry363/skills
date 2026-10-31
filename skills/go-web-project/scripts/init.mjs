#!/usr/bin/env node
// 脚手架生成器：把 assets/template 渲染成目标目录，并生成项目级 AGENTS.md。
//
// 用法：
//   node scripts/init.mjs <目标目录> --module=<go module 路径> [--service=api] [--go=1.27.0] [--minimal] [--force] [--dry-run]
//
// 也可作为库使用：import { generate } from "./init.mjs"
// 只做文件渲染，不联网、不执行 go 命令（除读取本地 go 版本）。

import { execFileSync } from "node:child_process";
import { mkdir, readdir, readFile, stat, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const skillRoot = path.resolve(__dirname, "..");
const templateRoot = path.join(skillRoot, "assets", "template");
const minimalRoot = path.join(skillRoot, "assets", "minimal");
const agentsTemplatePath = path.join(skillRoot, "references", "agents-template.md");

const SERVICE_RE = /^[a-z][a-z0-9-]*$/;
const MODULE_RE = /^[a-zA-Z0-9._~/-]+$/;

function parseArgs(argv) {
  const opts = {
    force: false,
    dryRun: false,
    minimal: false,
    target: "",
    module: "",
    service: "",
    goVersion: "",
  };
  const positional = [];

  for (const arg of argv) {
    if (arg === "--force") opts.force = true;
    else if (arg === "--dry-run") opts.dryRun = true;
    else if (arg === "--minimal") opts.minimal = true;
    else if (arg === "--help" || arg === "-h") opts.help = true;
    else if (arg.startsWith("--module=")) opts.module = arg.slice("--module=".length);
    else if (arg.startsWith("--service=")) opts.service = arg.slice("--service=".length);
    else if (arg.startsWith("--go=")) opts.goVersion = arg.slice("--go=".length);
    else if (arg.startsWith("-")) throw new Error(`未知参数 ${arg}`);
    else positional.push(arg);
  }

  if (opts.help) return opts;
  if (positional.length > 1) throw new Error("只接受一个目标目录");
  opts.target = positional[0] || "";
  return opts;
}

function usage() {
  return [
    "用法：node scripts/init.mjs <目标目录> --module=<module 路径> [选项]",
    "",
    "  --module=<path>   go.mod 的 module 路径，必填，例如 example.com/acme/api",
    "  --service=<name>  二进制名，默认取目录名；只能含小写字母、数字、连字符",
    "  --go=<version>    go 指令版本，默认读取本机 go 版本",
    "  --minimal         只留分层骨架：删除 user 示例，保留探针与统一响应",
    "  --force           目标目录非空时仍然写入（覆盖同名文件）",
    "  --dry-run         只打印将要写入的文件，不落盘",
  ].join("\n");
}

export function detectGoVersion() {
  try {
    const out = execFileSync("go", ["env", "GOVERSION"], {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"],
    }).trim();
    const v = out.replace(/^go/, "");
    return /^\d+\.\d+(\.\d+)?$/.test(v) ? v : "1.27.0";
  } catch {
    return "1.27.0";
  }
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

function render(text, vars) {
  return text
    .replaceAll("{{MODULE}}", vars.module)
    .replaceAll("{{SERVICE}}", vars.service)
    .replaceAll("{{GO_VERSION}}", vars.goVersion)
    .replaceAll("{{GO_MINOR}}", vars.goMinor);
}

// 去掉模板顶部的用法注释块，避免生成物里留下"复制本文件到…"的说明。
function stripUsageComment(md) {
  return md.replace(/^<!--[\s\S]*?-->\s*\n?/, "");
}

// applyMinimal 应用「只留分层骨架」覆盖层：
// 先按 remove.txt 删掉业务示例，再用 assets/minimal/files 覆盖受影响文件。
// 覆盖层只放差异文件，完整模板保持唯一事实源，避免两份模板漂移。
async function applyMinimal(plan, vars) {
  const removals = new Set(
    (await readFile(path.join(minimalRoot, "remove.txt"), "utf8"))
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter((line) => line.length > 0 && !line.startsWith("#")),
  );

  const filesRoot = path.join(minimalRoot, "files");
  const overlays = [];
  for (const abs of await walk(filesRoot)) {
    const rel = path.relative(filesRoot, abs).split(path.sep).join("/");
    overlays.push({ rel: render(rel, vars), body: render(await readFile(abs, "utf8"), vars) });
  }

  const overlaid = new Set(overlays.map((o) => o.rel));
  const kept = plan.filter((p) => !removals.has(p.rel) && !overlaid.has(p.rel));
  return [...kept, ...overlays];
}

/**
 * 渲染脚手架到目标目录（dryRun 时只返回计划）。
 *
 * @param {object} options
 * @param {string} options.target       目标目录
 * @param {string} options.module       go module 路径（必填）
 * @param {string} [options.service]    二进制名，默认取目录名
 * @param {string} [options.goVersion]  go 指令版本，默认读本机
 * @param {boolean} [options.minimal]   只留分层骨架
 * @param {boolean} [options.force]     目标目录非空时仍写入
 * @param {boolean} [options.dryRun]    只计算不落盘
 * @returns {Promise<{plan: {rel: string, body: string}[], target: string, vars: object}>}
 */
export async function generate(options = {}) {
  if (!options.target) throw new Error(`缺少目标目录\n\n${usage()}`);
  if (!options.module) throw new Error(`缺少 --module\n\n${usage()}`);

  const target = path.resolve(options.target);
  const service = options.service || path.basename(target).toLowerCase();
  const goVersion = options.goVersion || detectGoVersion();

  if (!SERVICE_RE.test(service)) throw new Error(`--service 只能含小写字母、数字、连字符：${service}`);
  if (!MODULE_RE.test(options.module)) throw new Error(`--module 含非法字符：${options.module}`);

  const vars = {
    module: options.module,
    service,
    goVersion,
    // 容器镜像只保证 major.minor 标签存在，patch 标签未必发布
    goMinor: goVersion.split(".").slice(0, 2).join("."),
  };

  // rel 统一用 / 分隔：跨平台比较与输出都需要同一种形式
  const toRel = (root, abs) => path.relative(root, abs).split(path.sep).join("/");

  let plan = [];
  for (const abs of await walk(templateRoot)) {
    plan.push({
      rel: render(toRel(templateRoot, abs), vars),
      body: render(await readFile(abs, "utf8"), vars),
    });
  }

  if (options.minimal) {
    plan = await applyMinimal(plan, vars);
  }

  // 项目级 AGENTS.md 与 references 中的模板共用同一份事实源。
  if (!plan.some((p) => p.rel === "AGENTS.md")) {
    const md = stripUsageComment(await readFile(agentsTemplatePath, "utf8"));
    plan.push({ rel: "AGENTS.md", body: render(md, vars) });
  }

  if (!options.dryRun) {
    const exists = await stat(target).then(() => true).catch(() => false);
    if (exists) {
      const existing = await readdir(target);
      if (existing.length > 0 && !options.force) {
        throw new Error(`目标目录非空：${target}\n加 --force 覆盖同名文件，或换一个空目录。`);
      }
    }

    for (const item of plan) {
      const dest = path.join(target, item.rel);
      await mkdir(path.dirname(dest), { recursive: true });
      await writeFile(dest, item.body, "utf8");
    }
  }

  return { plan, target, vars };
}

async function main() {
  const opts = parseArgs(process.argv.slice(2));
  if (opts.help) {
    console.log(usage());
    return;
  }

  const { plan, target, vars } = await generate(opts);

  if (opts.dryRun) {
    console.log(`将写入 ${plan.length} 个文件 → ${target}${opts.minimal ? "（minimal）" : ""}`);
    for (const p of plan) console.log(`  ${p.rel}`);
    return;
  }

  console.log(`✔ 已生成 ${plan.length} 个文件 → ${target}`);
  console.log(`  模块：${vars.module}`);
  console.log(`  服务：${vars.service}`);
  console.log(`  Go ：${vars.goVersion}`);
  console.log(`  模式：${opts.minimal ? "minimal（仅分层骨架）" : "完整（含 user 示例）"}`);
  console.log("");
  console.log("下一步：");
  console.log(`  cd ${path.relative(process.cwd(), target) || "."}`);
  console.log("  go build ./...          # 依赖已锁定，无需先 tidy");
  console.log("  cp .env.example .env    # 填写 DATABASE_URL");
  console.log("  docker compose up -d db && make migrate-up");
  console.log("  make run");
  console.log("");
  console.log("交付前运行 node scripts/verify.mjs 做一致性自检。");
}

// 仅在被直接执行时跑 CLI；被 verify.mjs import 时不触发。
const invokedDirectly =
  process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href;

if (invokedDirectly) {
  main().catch((err) => {
    console.error(`错误：${err.message}`);
    process.exit(1);
  });
}
