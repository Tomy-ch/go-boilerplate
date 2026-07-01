// Mermaid のコードフェンス（```mermaid）を実際の mermaid パーサで構文検証する lint スクリプト。
// markdownlint-cli2 は Markdown の体裁しか見ず mermaid 図の文法は素通りするため、その穴を塞ぐ。
// node_tool_runner コンテナ内で `make md-lint-ci` から呼ばれる前提（mermaid / linkedom は scripts/node_modules）。
//
// mermaid.parse は DOMPurify サニタイズで DOM を要求するため、mermaid のロードには DOM 環境が要る。
// 1 つでも壊れた図があれば非 0 で終了する。
import fs from "node:fs"
import path from "node:path"
import { createRequire } from "node:module"

// 依存ロード（環境セットアップ）。mermaid / linkedom は node_tool_runner コンテナ内の
// scripts/node_modules に入る前提。ここで失敗した場合は mermaid 図の文法問題ではなく
// 「環境未整備」なので、生の ERR_MODULE_NOT_FOUND スタックトレースではなく、原因と対処を
// 明示して exit 2（lint 失敗の exit 1 と区別）で落とす。
let mermaid
try {
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
  mermaid = mermaidModule.default ?? mermaidModule
  // logLevel:5(fatal) でパース失敗時の冗長な内部ログを抑止し、本スクリプトの整形済み出力に一本化する。
  mermaid.initialize({ startOnLoad: false, securityLevel: "loose", logLevel: 5 })
} catch (e) {
  const detail = (e && e.message ? e.message : String(e)).trim()
  const depMissing = Boolean(e) && (e.code === "ERR_MODULE_NOT_FOUND" || /cannot find (package|module)/i.test(detail))
  console.error("✘ mermaid-lint: セットアップエラー（mermaid 図の文法問題ではありません）")
  if (depMissing) {
    console.error("    原因: mermaid / linkedom を解決できません（scripts/node_modules が未整備）。")
    console.error("    対処: この lint は node_tool_runner コンテナ内で動く前提です。")
    console.error("          - 通常は `make md-lint`（コンテナ経由）で実行する（host での直接実行は不可）。")
    console.error("          - イメージが古い場合は `make tool-runners-build`（または `-clean`）で再ビルドする。")
  } else {
    console.error("    原因: 依存ロード中に想定外のエラーが発生しました。")
  }
  console.error(`    詳細: ${detail}`)
  process.exit(2)
}

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
      // close は `\s*$` 終端のため、フェンス文字のみの行だけが閉じになる（```mermaid 等は閉じ扱いにならない）。
      if (close.test(lines[j])) break
      body.push(lines[j])
    }
    blocks.push({ startLine: i + 1, code: body.join("\n") })
    i = j
  }
  return blocks
}

const repoRoot = process.cwd()
const files = collectMarkdown(repoRoot)
const suffix = (n) => (n > 0 ? `（読めず skip: ${n} 件）` : "")

let blockCount = 0
let fileWithBlocks = 0
const failures = []
const skipped = []

for (const rel of files) {
  let content
  try {
    content = fs.readFileSync(path.join(repoRoot, rel), "utf8")
  } catch {
    // 読めないファイル（壊れた symlink や権限エラー）は検証対象外としてスキップする。
    // markdownlint も壊れた symlink を黙って飛ばすため挙動を揃える。
    skipped.push(rel)
    continue
  }
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

console.log(`✓ mermaid-lint: ${blockCount} ブロック / ${fileWithBlocks} ファイル すべて OK${suffix(skipped.length)}`)
