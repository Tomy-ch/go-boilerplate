#!/usr/bin/env node
// release/vX.Y.Z のブランチ名から OpenAPI の info.version を契約版(X.Y.Z)へ同期する。
//
// production マージ前（release ブランチ）で確定させることで、production 到達時点でタグと spec が一致する。
// build metadata / commit SHA は付けない（毎 push のチャーンを避けるため。commit 単位の追跡は runtime の /version 側の責務）。
//
// 使い方:
//   node scripts/stamp-openapi-version.mjs [<ref>]
//     <ref> 省略時は環境変数 GITHUB_REF_NAME を使用する。
//   release/vX.Y.Z 以外の ref は no-op（スキップして正常終了）。

import { readFileSync, writeFileSync } from "node:fs"

const OPENAPI_PATH = new URL("../openapi/openapi.yaml", import.meta.url)
// info ブロック直下の最初の version 行（2スペースインデント）。
// /m で行頭一致し、g を付けないことで最初の1件（= info.version）のみを置換する。
const VERSION_LINE = /^  version: .*$/m
const SEMVER = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/

// release/vX.Y.Z → X.Y.Z（それ以外は null）。
function deriveVersion(ref) {
  const matched = /^release\/v(.+)$/.exec(ref ?? "")
  return matched ? matched[1] : null
}

function main() {
  const ref = process.argv[2] ?? process.env.GITHUB_REF_NAME ?? ""
  const version = deriveVersion(ref)

  if (!version || !SEMVER.test(version)) {
    console.log(`ref '${ref}' は release/vX.Y.Z 形式ではないため version stamp をスキップします`)
    return
  }

  const content = readFileSync(OPENAPI_PATH, "utf8")
  const line = VERSION_LINE.exec(content)
  if (!line) {
    console.error("openapi/openapi.yaml に info.version 行が見つかりません")
    process.exit(1)
  }

  const current = line[0].replace(/^  version: /, "")
  if (current === version) {
    console.log(`info.version は既に ${version}（変更なし）`)
    return
  }

  const updated = content.replace(VERSION_LINE, () => `  version: ${version}`)
  writeFileSync(OPENAPI_PATH, updated)
  console.log(`info.version: ${current} → ${version}`)
}

main()
