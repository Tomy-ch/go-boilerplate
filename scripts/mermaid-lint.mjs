// Mermaid のコードフェンス（```mermaid）を実際の mermaid パーサで構文検証する lint スクリプト。
// markdownlint-cli2 は Markdown の体裁しか見ず mermaid 図の文法は素通りするため、その穴を塞ぐ。
// node_tool_runner コンテナ内で `make md-lint-ci` から呼ばれる前提（mermaid / linkedom は scripts/node_modules）。
//
// mermaid.parse は flowchart の DOMPurify サニタイズで DOM を要求するため、import 前に linkedom で
// 最小の window/document を用意してから mermaid を動的 import する。1 つでも壊れた図があれば非 0 で終了する。
import fs from "node:fs"
import path from "node:path"
import { createRequire } from "node:module"

// import 順の都合で mermaid より先に DOM を globalThis へ載せる必要があるため require で先行ロードする。
const require = createRequire(import.meta.url)
const { parseHTML } = require("linkedom")

const { window, document } = parseHTML("<!doctype html><html><head></head><body></body></html>")
globalThis.window = window
globalThis.document = document
Object.defineProperty(globalThis, "navigator", { value: window.navigator, configurable: true })
globalThis.location = window.location
globalThis.requestAnimationFrame = (fn) => setTimeout(fn, 0)
globalThis.MutationObserver = window.MutationObserver

const mermaidModule = await import("mermaid")
const mermaid = mermaidModule.default ?? mermaidModule
// logLevel:5(fatal) でパース失敗時の冗長な内部ログを抑止し、本スクリプトの整形済み出力に一本化する。
mermaid.initialize({ startOnLoad: false, securityLevel: "loose", logLevel: 5 })

// markdownlint-cli2 の MD_GLOBS と対象範囲を揃える（生成物・vendor・AGENTS.md を除外）。
const EXCLUDE_DIRS = new Set(["vendor", "node_modules", ".git"])
const EXCLUDE_PREFIXES = [
  "docs/portal/guides/",
  "docs/coverage/",
  "docs/db-schema/",
]
const EXCLUDE_FILES = new Set(["AGENTS.md"])

// repoRoot 配下の *.md を再帰収集する（除外ディレクトリは降りない）。
function collectMarkdown(repoRoot) {
  const out = []
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const abs = path.join(dir, entry.name)
      if (entry.isDirectory()) {
        if (EXCLUDE_DIRS.has(entry.name)) continue
        walk(abs)
        continue
      }
      if (!entry.name.endsWith(".md")) continue
      const rel = path.relative(repoRoot, abs)
      if (EXCLUDE_FILES.has(rel)) continue
      if (EXCLUDE_PREFIXES.some((p) => rel.startsWith(p))) continue
      out.push(rel)
    }
  }
  walk(repoRoot)
  return out.sort()
}

// Markdown から ```mermaid フェンスを開始行付きで抜き出す。
function extractMermaidBlocks(content) {
  const lines = content.split("\n")
  const blocks = []
  const fence = /^(\s*)(`{3,}|~{3,})\s*mermaid\s*$/
  for (let i = 0; i < lines.length; i++) {
    const m = fence.exec(lines[i])
    if (!m) continue
    const marker = m[2][0].repeat(m[2].length)
    const close = new RegExp(`^\\s*${marker[0] === "`" ? "`" : "~"}{${m[2].length},}\\s*$`)
    const body = []
    let j = i + 1
    for (; j < lines.length; j++) {
      if (close.test(lines[j]) && !lines[j].trim().endsWith("mermaid")) break
      body.push(lines[j])
    }
    blocks.push({ startLine: i + 1, code: body.join("\n") })
    i = j
  }
  return blocks
}

const repoRoot = process.cwd()
const files = collectMarkdown(repoRoot)

let blockCount = 0
let fileWithBlocks = 0
const failures = []

for (const rel of files) {
  const content = fs.readFileSync(path.join(repoRoot, rel), "utf8")
  const blocks = extractMermaidBlocks(content)
  if (blocks.length > 0) fileWithBlocks++
  for (let b = 0; b < blocks.length; b++) {
    blockCount++
    try {
      await mermaid.parse(blocks[b].code)
    } catch (e) {
      const msg = (e && e.message ? e.message : String(e)).trim()
      failures.push({ rel, startLine: blocks[b].startLine, index: b + 1, msg })
    }
  }
}

if (failures.length > 0) {
  console.error(`✘ mermaid-lint: ${failures.length} 件の壊れた mermaid ブロック\n`)
  for (const f of failures) {
    console.error(`  ${f.rel}:${f.startLine}  (block #${f.index})`)
    for (const line of f.msg.split("\n")) console.error(`    ${line}`)
    console.error("")
  }
  console.error(`検証 ${blockCount} ブロック / ${fileWithBlocks} ファイル中 ${failures.length} 件 NG`)
  process.exit(1)
}

console.log(`✓ mermaid-lint: ${blockCount} ブロック / ${fileWithBlocks} ファイル すべて OK`)
