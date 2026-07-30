// graphify の `affected`（変更影響の逆引き）をシンボル名で叩けるようにするラッパー。
//
// `graphify affected` は一意なノード id しか受け付けず、`NormalizeError()` のような
// シンボル名では "No unique node match" で止まる。id は graph.json の中にしか無いため、
// 素の CLI では利用者が 14MB の JSON を自力で漁ることになる。ここで名前→id を解決し、
// 曖昧なときは候補を出して選ばせる。
//
// 使い方:
//   node .claude/scripts/graph-affected.mjs <symbol> [--depth N] [--graph PATH] [-- <graphify の追加引数>]
//
// 前提: `graphify update .` 済み（graphify-out/graph.json が存在する）。
import fs from "node:fs"
import { spawnSync } from "node:child_process"

const DEFAULT_GRAPH = "graphify-out/graph.json"

const argv = process.argv.slice(2)
const passThroughAt = argv.indexOf("--")
const passThrough = passThroughAt === -1 ? [] : argv.slice(passThroughAt + 1)
const args = passThroughAt === -1 ? argv : argv.slice(0, passThroughAt)

// オプションを剥がした残りをシンボル名として扱う。
const options = new Map()
const positional = []
for (let i = 0; i < args.length; i += 1) {
  if (args[i] === "--depth" || args[i] === "--graph") {
    options.set(args[i], args[i + 1])
    i += 1
    continue
  }
  positional.push(args[i])
}

const symbol = positional[0]
if (!symbol) {
  console.error("使い方: node .claude/scripts/graph-affected.mjs <symbol> [--depth N] [--graph PATH] [-- <graphify の追加引数>]")
  process.exit(2)
}

const graphPath = options.get("--graph") ?? DEFAULT_GRAPH
if (!fs.existsSync(graphPath)) {
  console.error(`✘ グラフがありません: ${graphPath}`)
  console.error('    対処: `mise exec "pipx:graphifyy[sql]" -- graphify update . --no-cluster` で生成する。')
  process.exit(2)
}

const graph = JSON.parse(fs.readFileSync(graphPath, "utf8"))
const nodes = graph.nodes ?? []

// 絞り込みは 3 段。Go はシンボルをケースで区別する（`NormalizeError` と `normalizeError` は
// 別物）ため、ケースを保った完全一致を最優先する。
const strip = (value) => (value ?? "").replace(/\(\)$/, "")
const fold = (value) => strip(value).toLowerCase()
const wanted = strip(symbol)
const passes = [
  (n) => strip(n.label) === wanted,
  (n) => fold(n.label) === fold(wanted) || fold(n.norm_label) === fold(wanted),
  (n) => fold(n.label).includes(fold(wanted)),
]

let candidates = []
for (const matches of passes) {
  candidates = nodes.filter(matches)
  if (candidates.length > 0) {
    break
  }
}

if (candidates.length === 0) {
  console.error(`✘ '${symbol}' に一致するノードがありません（${nodes.length} ノードを検索）`)
  console.error("    グラフが古い可能性があります。`graphify update .` を実行してから再試行してください。")
  process.exit(1)
}

if (candidates.length > 1) {
  console.error(`✘ '${symbol}' が ${candidates.length} 件に一致します。id を指定して再実行してください:`)
  for (const n of candidates.slice(0, 20)) {
    console.error(`    ${n.id}  (${n.label} @ ${n.source_file}:${n.source_location})`)
  }
  if (candidates.length > 20) {
    console.error(`    ... 他 ${candidates.length - 20} 件`)
  }
  process.exit(1)
}

const target = candidates[0]
console.error(`→ ${target.label} @ ${target.source_file}:${target.source_location} (${target.id})`)

const graphifyArgs = ["exec", "pipx:graphifyy[sql]", "--", "graphify", "affected", target.id]
if (options.has("--depth")) {
  graphifyArgs.push("--depth", options.get("--depth"))
}
if (options.get("--graph")) {
  graphifyArgs.push("--graph", options.get("--graph"))
}
graphifyArgs.push(...passThrough)

const run = spawnSync("mise", graphifyArgs, { stdio: "inherit" })
process.exit(run.status ?? 1)
