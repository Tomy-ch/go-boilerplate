// ワークフローが自前で組む Markdown フェンスの退行を検査する lint スクリプト。
//
// `upsert-pr-comment` が本文中の最長バッククォート連 + 1 をフェンス長とするのは `details-summary`
// 経路だけで、渡さない呼び出しでは本文を素通しする。そこでフェンスを自前に組むワークフローが
// フェンスの責任を負い、固定 3 連は PR 提出者が書いたソース行を引用する本文に閉じられる。
//
// inline code span も長さ 1 のフェンスでしかないので、同じ話が span にも及ぶ。値に含まれる 1 個の
// バッククォートが span を閉じ、以降が生 Markdown に戻る。git のパスに使えない文字は NUL と `/` だけ
// なので、ファイル名を span へ入れる素通し経路は同じ穴を持つ。
//
// 機械的に判定できる 3 点のみを見る。「その本文が攻撃者制御か」は判定しない。
//
//   1. `run:` ブロックが固定長のフェンスを出力していないこと
//   2. 複数のワークフローが持つ `fence_for` の実装が互いに完全一致していること
//   3. 素通し呼び出しを持つワークフローが、inline code span の内側へ値を補間していないこと
//
// 2 が要るのは、実装を composite action へ集約できないため。フェンス文字列を output 経由で受け取ると
// バッククォートがシェルの二重引用符文脈でコマンド置換になる。複製は意図的な選択であり、そのぶん
// 片方だけが直る事故を機械で止める。
//
// 3 の粒度はワークフローファイル単位で、ステップから本文ファイルへのデータフローは追わない
// （`out=/tmp/...` の間接参照解決は脆く、既存の行ベース走査から逸脱する）。素通し呼び出しと
// `details-summary` 付き呼び出しが同居するファイルでは過剰検出になり得るが、意図した単純化として
// 除外リストで受ける。span を変数経由や jq の文字列連結で組む形は検出できない（偽陰性）。
import fs from "node:fs"
import path from "node:path"

const REPO_ROOT = process.cwd()
const WORKFLOWS_DIR = ".github/workflows"
const COMMENT_ACTION = "./.github/actions/upsert-pr-comment"

const FIXED_FENCE = /(?:^|[\s'"])(`{3,})(?:[A-Za-z0-9_-]*)['"]?\s*$/
const FENCE_FOR_BLOCK = /^\s*fence_for\(\)\s*\{$/
const COMMENT_ACTION_USE = new RegExp(
  `uses:\\s*["']?${COMMENT_ACTION.replace(/[.\/]/g, "\\$&")}["']?\\s*(#.*)?$`,
)
const DETAILS_SUMMARY = /^\s*details-summary:\s*(.*)$/
const STEP_BULLET = /^\s*-\s/
// バッククォート 1 個で開いて 1 個で閉じる span の中に、シェル変数展開か printf の変換指定がある形。
const INTERPOLATED_SPAN = /`[^`\n]*(?:\$\{|\$[A-Za-z_]|%[-0-9.*]*[sb])[^`\n]*`/

// 解決までフェンス検査から外すワークフロー。エントリは根拠の issue を持ち、直したら消す。
const PASS_THROUGH_EXCLUSIONS = new Map([
  [
    "image-scan.yaml",
    "#871: SBOM summary が素通し経路で、span どころか生 Markdown のまま値を埋めている（データ源が SBOM で本文が見出しをレンダリングさせる設計のため #835 と同じ解決が採れない）",
  ],
])

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

// アクションがフェンスするのは `details-summary` が空でない値を持つときだけ（action.yaml の
// `detailsSummary ? ... : content`）。キーの有無で判定すると `details-summary: ''` が「フェンス済み」に
// 化けて検査が黙る。式は静的に空か判定できないので、フェンスされない側へ倒す。
function fencesBody(line) {
  const matched = line.match(DETAILS_SUMMARY)
  if (matched === null) return false

  const value = matched[1].trim()
  if (value === "" || value === "''" || value === '""') return false
  return !value.includes("${{")
}

// `details-summary` を渡さない呼び出しがあるか。アクションが本文をフェンスするのはこの入力がある
// ときだけで、無ければ本文はそのまま PR コメントになる。
function hasPassThroughCall(lines) {
  for (let i = 0; i < lines.length; i++) {
    if (!COMMENT_ACTION_USE.test(lines[i])) continue

    // ステップの範囲は `uses:` の深さではなく、そのステップを開いた `-` の桁で決める。深さで見ると
    // `- uses:` から書き始めたステップが自分の `-` の桁を `uses:` の桁として測り、次のステップの `-`
    // で止まれない。素通しの呼び出しが後続ステップの `details-summary` を拾って隠れる向きに外れる。
    let bullet = i
    while (bullet >= 0 && !STEP_BULLET.test(lines[bullet])) bullet--
    const stepIndent =
      bullet >= 0 ? lines[bullet].length - lines[bullet].trimStart().length : 0

    let fenced = false
    for (let j = i + 1; j < lines.length; j++) {
      const line = lines[j]
      if (line.trim() === "") continue
      const indent = line.length - line.trimStart().length
      // 次のステップを開く `-`（同じ桁）か、ステップより浅いキーに出たらこの呼び出しは終わり。
      if (indent < stepIndent || (indent === stepIndent && STEP_BULLET.test(line))) break
      if (fencesBody(line)) {
        fenced = true
        break
      }
    }
    if (!fenced) return true
  }
  return false
}

function findInterpolatedSpans(lines) {
  const hits = []
  lines.forEach((line, index) => {
    const trimmed = line.trim()
    // 規約そのものを説明するコメント行に反応させない。
    if (trimmed.startsWith("#")) return
    if (INTERPOLATED_SPAN.test(trimmed)) hits.push({ line: index + 1, text: trimmed })
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

  if (hasPassThroughCall(lines) && !PASS_THROUGH_EXCLUSIONS.has(path.basename(file))) {
    for (const hit of findInterpolatedSpans(lines)) {
      violations.push(
        `${file}:${hit.line}: 本文素通しの呼び出しがあるワークフローで、inline code span へ値を補間しています。値に含まれるバッククォート 1 個が span を閉じ、以降が生 Markdown になります: ${hit.text}`,
      )
    }
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

// 除外を黙ったまま緑にすると「検査が通った」と「見ていない」の区別が付かなくなる。
for (const [name, reason] of PASS_THROUGH_EXCLUSIONS) {
  console.log(`- span 検査を除外: ${name} — ${reason}`)
}

console.log(
  `✓ pr-comment-fence-lint: ${listWorkflows().length} ワークフロー / fence_for 実装 ${impls.length} 件すべて OK`,
)
