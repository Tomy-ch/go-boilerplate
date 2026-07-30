// ワークフローが自前で組む Markdown フェンスの退行を検査する lint スクリプト。
//
// `upsert-pr-comment` が本文中の最長バッククォート連 + 1 をフェンス長とするのは `details-summary`
// 経路だけで、渡さない呼び出しでは本文を素通しする。そこでフェンスを自前に組むワークフローが
// フェンスの責任を負い、固定 3 連は PR 提出者が書いたソース行を引用する本文に閉じられる。
//
// 機械的に判定できる 2 点のみを見る。「その本文が攻撃者制御か」は判定しない。
//
//   1. `run:` ブロックが固定長のフェンスを出力していないこと
//   2. 複数のワークフローが持つ `fence_for` の実装が互いに完全一致していること
//
// 2 が要るのは、実装を composite action へ集約できないため。フェンス文字列を output 経由で受け取ると
// バッククォートがシェルの二重引用符文脈でコマンド置換になる。複製は意図的な選択であり、そのぶん
// 片方だけが直る事故を機械で止める。
import fs from "node:fs"
import path from "node:path"

const REPO_ROOT = process.cwd()
const WORKFLOWS_DIR = ".github/workflows"

const FIXED_FENCE = /(?:^|[\s'"])(`{3,})(?:[A-Za-z0-9_-]*)['"]?\s*$/
const FENCE_FOR_BLOCK = /^\s*fence_for\(\)\s*\{$/

function listWorkflows() {
  const dir = path.join(REPO_ROOT, WORKFLOWS_DIR)
  return fs
    .readdirSync(dir)
    .filter((name) => name.endsWith(".yaml") || name.endsWith(".yml"))
    .sort()
    .map((name) => path.join(WORKFLOWS_DIR, name))
}

// インデント差で不一致にならないよう、各行を trim して比較単位にする。
function extractFenceFor(lines) {
  const start = lines.findIndex((line) => FENCE_FOR_BLOCK.test(line))
  if (start < 0) return null

  const body = []
  for (let i = start; i < lines.length; i++) {
    const trimmed = lines[i].trim()
    body.push(trimmed)
    if (i > start && trimmed === "}") return body.join("\n")
  }
  return null
}

function findFixedFences(lines) {
  const hits = []
  lines.forEach((line, index) => {
    const trimmed = line.trim()
    // 変数でフェンスを組む行（`echo "${fence}text"`）は対象外。リテラルだけを見る。
    if (!trimmed.startsWith("echo ")) return
    if (trimmed.includes("${")) return
    if (FIXED_FENCE.test(trimmed)) hits.push({ line: index + 1, text: trimmed })
  })
  return hits
}

const violations = []
const implementations = new Map()

for (const file of listWorkflows()) {
  const lines = fs.readFileSync(path.join(REPO_ROOT, file), "utf8").split("\n")

  for (const hit of findFixedFences(lines)) {
    violations.push(
      `${file}:${hit.line}: 固定長のフェンスを出力しています。本文がこのフェンスを閉じられます: ${hit.text}`,
    )
  }

  const impl = extractFenceFor(lines)
  if (impl !== null) implementations.set(file, impl)
}

const impls = [...implementations.entries()]
if (impls.length > 1) {
  const [refFile, refImpl] = impls[0]
  for (const [file, impl] of impls.slice(1)) {
    if (impl !== refImpl) {
      violations.push(
        `${file}: fence_for の実装が ${refFile} と一致しません。フェンス長の計算は全複製で同一である必要があります`,
      )
    }
  }
}

if (violations.length > 0) {
  for (const violation of violations) console.error(`✗ ${violation}`)
  console.error(`\n✗ pr-comment-fence-lint: ${violations.length} 件の違反があります`)
  process.exit(1)
}

console.log(
  `✓ pr-comment-fence-lint: ${listWorkflows().length} ワークフロー / fence_for 実装 ${impls.length} 件すべて OK`,
)
