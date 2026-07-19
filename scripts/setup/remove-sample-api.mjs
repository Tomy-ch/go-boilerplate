import fs from "node:fs"
import path from "node:path"
import { ROOT_DIR, newSetupCommand } from "./lib/runtime.mjs"
import { toAbsolutePath, updateFile } from "./lib/file-utils.mjs"
import { SAMPLE_DOMAINS, MARKER_FILES, BUILD_STEPS } from "./lib/sample-manifest.mjs"
import { stripSampleMarkers } from "./lib/sample-api.mjs"

// 再生成・整形・検証（make gen-api / gen-query / fix / lint）は Go ツールチェーンが要るため
// ここでは行わず、ホスト側の make ターゲット（setup-remove-sample-api）が担当する。

const ROOT_WITH_SEP = ROOT_DIR.endsWith(path.sep) ? ROOT_DIR : ROOT_DIR + path.sep

// 削除確認 mjs（verify-sample-removal.mjs）が git status と突き合わせる「登録済み削除対象」の
// スナップショット出力先。manifest（sample-manifest.mjs）自身が削除対象で削除後は読めないため、
// 削除時にここへ書き出す。削除確認 mjs は manifest に依存せずこの JSON だけで照合できる。
const SNAPSHOT_PATH = path.join(ROOT_DIR, "scripts/setup/.sample-removal-snapshot.json")

// 全ドメインの登録パスを列挙して照合用スナップショットへ書き出す。
function writeSnapshot() {
  const registeredPaths = Object.values(SAMPLE_DOMAINS).flatMap(def => def.paths)
  fs.writeFileSync(SNAPSHOT_PATH, `${JSON.stringify({ registeredPaths }, null, 2)}\n`)
}

// manifest の追記ミス（`..`・空文字・絶対パス）で ROOT_DIR 外や ROOT_DIR 自体を
// rmSync しないための安全策。dry-run でも検証されるよう削除前に必ず通す。
function assertWithinRoot(absolutePath, relativePath) {
  if (absolutePath === ROOT_DIR || !absolutePath.startsWith(ROOT_WITH_SEP)) {
    throw new Error(`削除対象が ROOT_DIR の外（または ROOT_DIR 自体）を指しています: "${relativePath}"`)
  }
}

// 既に存在しないパスはスキップする（再実行や部分実装でも安全に動かすため）。
function deletePaths(dryRun) {
  const deleted = []
  const missing = []

  for (const [domain, def] of Object.entries(SAMPLE_DOMAINS)) {
    for (const relativePath of def.paths) {
      const absolutePath = toAbsolutePath(relativePath)
      assertWithinRoot(absolutePath, relativePath)

      if (!fs.existsSync(absolutePath)) {
        missing.push(relativePath)
        continue
      }

      if (!dryRun) {
        fs.rmSync(absolutePath, { recursive: true, force: true })
      }

      deleted.push({ domain, relativePath })
    }
  }

  return { deleted, missing }
}

function stripMarkerFiles(dryRun) {
  const changed = []

  for (const relativePath of MARKER_FILES) {
    let removedLines = 0

    const updated = updateFile(relativePath, original => {
      const result = stripSampleMarkers(original)
      removedLines = result.removed
      return result.content
    }, dryRun)

    if (updated) {
      changed.push({ relativePath, removedLines })
    }
  }

  return changed
}

function report({ deleted, missing }, stripped, dryRun) {
  const label = dryRun ? "【Dry Run】削除対象" : "削除完了"

  console.log(`\n${label}: ${deleted.length} パス`)
  for (const domain of Object.keys(SAMPLE_DOMAINS)) {
    const items = deleted.filter(d => d.domain === domain)
    if (items.length === 0) {
      continue
    }
    console.log(`\n[${domain}] ${SAMPLE_DOMAINS[domain].description}`)
    for (const item of items) {
      console.log(`  - ${item.relativePath}`)
    }
  }

  if (missing.length > 0) {
    console.log(`\n🟡 既に存在しない（スキップ）: ${missing.length} パス`)
    for (const relativePath of missing) {
      console.log(`  - ${relativePath}`)
    }
  }

  console.log(
    `\n${dryRun ? "【Dry Run】マーカー除去対象" : "マーカー除去完了"}: ${stripped.length} ファイル`
  )
  for (const item of stripped) {
    console.log(`  - ${item.relativePath} (${item.removedLines} 行)`)
  }
}

function run({ dryRun }) {
  console.log("🧹 サンプルAPI（user / prefecture / product / order）の削除を開始します。")
  console.log(`   ルート: ${ROOT_DIR}`)
  if (dryRun) {
    console.log("   モード: dry-run（ファイルは変更しません）")
  }

  // マーカー除去を先に行う。マーカー不整合があればここで throw し、
  // ファイル削除前に中断できる（削除済み・マーカー未除去の半端な状態を避ける）。
  const stripped = stripMarkerFiles(dryRun)
  const deletion = deletePaths(dryRun)
  report(deletion, stripped, dryRun)

  const buildHint = `make ${BUILD_STEPS.join(" ")}`
  if (dryRun) {
    console.log(
      `\n次の手順:\n  実削除  : DRY_RUN を外して再実行\n  再生成等: 削除後に \`${buildHint}\`（make setup-remove-sample-api 経由なら自動実行）`
    )
    return
  }

  writeSnapshot()

  console.log(
    `\n✅ 削除とマーカー除去が完了しました。\n   続けて再生成・整形・検証を行ってください: \`${buildHint}\`\n   （make setup-remove-sample-api 経由で実行した場合は自動で続行されます）`
  )
}

const program = newSetupCommand("remove-sample-api")
program
  .description("ボイラープレートのサンプルAPI（user / prefecture / product / order）を一括削除する")
  .addHelpText(
    "after",
    `
動作:
  1. manifest（scripts/setup/lib/sample-manifest.mjs）の宣言パスを丸ごと削除
  2. 共有ファイル（DI 4 ファイル + openapi.yaml）の sample-api マーカー行を除去

削除後は ${BUILD_STEPS.map(s => `make ${s}`).join(" → ")} で再生成・整形・検証してください
（このスクリプトは Go ツールチェーンを持たない node_tool_runner で動くため再生成は行いません。
 make setup-remove-sample-api 経由ならホスト側で自動的に続行されます）。

core 基盤の idempotency_keys（migration 000001）は削除しません（prefecture は user サンプルの依存ドメインとして削除対象）。
共有生成物（*.gen.go / openapi.gen.yaml 等）は再生成に任せます。
拡張時は sample-manifest.mjs の各ドメイン paths への追記とマーカー付与だけで対象に含まれます。`
  )
  .action(options => {
    try {
      run({ dryRun: options.dryRun })
    } catch (error) {
      program.error(`エラー: ${error.message}`)
    }
  })
  .parse()
