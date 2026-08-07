#!/usr/bin/env -S tsx
// release/vX.Y.Z のブランチ名から OpenAPI の info.version を X.Y.Z（SemVer のみ・SHA 等は付けない）へ書き換える。
//
// 使い方:
//   tsx scripts/stamp-openapi-version [<ref>]
//     <ref> 省略時は環境変数 GITHUB_REF_NAME を使用する。
//   release/vX.Y.Z 以外の ref は no-op（スキップして正常終了）。

import { readFileSync, writeFileSync } from "node:fs";

import { deriveVersion, readVersion, replaceVersion } from "./version";

const OPENAPI_PATH = new URL("../openapi/openapi.yaml", import.meta.url);

function main(): void {
  const ref = process.argv[2] ?? process.env.GITHUB_REF_NAME ?? "";
  const version = deriveVersion(ref);

  if (version === null) {
    console.log(`ref '${ref}' は release/vX.Y.Z 形式ではないため version stamp をスキップします`);
    return;
  }

  const content = readFileSync(OPENAPI_PATH, "utf8");
  const current = readVersion(content);

  if (current === null) {
    console.error("openapi/openapi.yaml に info.version 行が見つかりません");
    process.exit(1);
  }

  if (current === version) {
    console.log(`info.version は既に ${version}（変更なし）`);
    return;
  }

  writeFileSync(OPENAPI_PATH, replaceVersion(content, version));
  console.log(`info.version: ${current} → ${version}`);
}

main();
