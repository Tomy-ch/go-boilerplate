import fs from "node:fs"
import { ROOT_DIR, newSetupCommand } from "./lib/runtime.mjs"
import { toAbsolutePath, updateFile } from "./lib/file-utils.mjs"
import {
  SAMPLE_DOMAINS,
  MARKER_FILES,
  BUILD_STEPS,
  stripSampleMarkers,
} from "./lib/sample-api.mjs"

// 再生成・整形・検証（make gen-api / gen-query / fix / lint）は Go ツールチェーンが要るため
// ここでは行わず、ホスト側の make ターゲット（setup-remove-sample-api）が担当する。

// 既に存在しないパスはスキップする（再実行や部分実装でも安全に動かすため）。
function deletePaths(dryRun) {
  const deleted = []
  const missing = []

  for (const [domain, def] of Object.entries(SAMPLE_DOMAINS)) {
    for (const relativePath of def.paths) {
      const absolutePath = toAbsolutePath(relativePath)

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
  console.log("🧹 サンプルAPI（user / product / order）の削除を開始します。")
  console.log(`   ルート: ${ROOT_DIR}`)
  if (dryRun) {
    console.log("   モード: dry-run（ファイルは変更しません）")
  }

  const deletion = deletePaths(dryRun)
  const stripped = stripMarkerFiles(dryRun)
  report(deletion, stripped, dryRun)

  const buildHint = `make ${BUILD_STEPS.join(" ")}`
  if (dryRun) {
    console.log(
      `\n次の手順:\n  実削除  : DRY_RUN を外して再実行\n  再生成等: 削除後に \`${buildHint}\`（make setup-remove-sample-api 経由なら自動実行）`
    )
    return
  }

  console.log(
    `\n✅ 削除とマーカー除去が完了しました。\n   続けて再生成・整形・検証を行ってください: \`${buildHint}\`\n   （make setup-remove-sample-api 経由で実行した場合は自動で続行されます）`
  )
}

const program = newSetupCommand("remove-sample-api")
program
  .description("ボイラープレートのサンプルAPI（user / product / order）を一括削除する")
  .addHelpText(
    "after",
    `
動作:
  1. manifest（scripts/setup/lib/sample-api.mjs）の宣言パスを丸ごと削除
  2. 共有ファイル（DI 4 ファイル + openapi.yaml）の sample-api マーカー行を除去

削除後は ${BUILD_STEPS.map(s => `make ${s}`).join(" → ")} で再生成・整形・検証してください
（このスクリプトは Go ツールチェーンを持たない node_tool_runner で動くため再生成は行いません。
 make setup-remove-sample-api 経由ならホスト側で自動的に続行されます）。

基盤データ prefecture（migration 000001 等）は削除しません。
共有生成物（*.gen.go / openapi.gen.yaml 等）は再生成に任せます。
拡張時は sample-api.mjs の各ドメイン paths への追記とマーカー付与だけで対象に含まれます。`
  )
  .action(options => {
    try {
      run({ dryRun: options.dryRun })
    } catch (error) {
      program.error(`エラー: ${error.message}`)
    }
  })
  .parse()
