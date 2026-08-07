#!/usr/bin/env -S tsx
// release/vX.Y.Z のブランチ名から OpenAPI の info.version を X.Y.Z（SemVer のみ・SHA 等は付けない）へ書き換える。
//
// 使い方:
//   tsx scripts/stamp-openapi-version [<ref>]
//     <ref> 省略時は環境変数 GITHUB_REF_NAME を使用する。
//   release/vX.Y.Z 以外の ref は no-op（スキップして正常終了）。

import { readFileSync, writeFileSync } from "node:fs";

import { planStamp } from "./version";

const OPENAPI_PATH = new URL("../../openapi/openapi.yaml", import.meta.url);

function main(): void {
  const ref = process.argv[2] ?? process.env.GITHUB_REF_NAME ?? "";
  const plan = planStamp(ref, () => readFileSync(OPENAPI_PATH, "utf8"));

  switch (plan.kind) {
    case "skip":
      console.log(`ref '${plan.ref}' は release/vX.Y.Z 形式ではないため version stamp をスキップします`);
      return;
    case "missing":
      console.error("openapi/openapi.yaml に info.version 行が見つかりません");
      process.exit(1);
      return;
    case "unchanged":
      console.log(`info.version は既に ${plan.version}（変更なし）`);
      return;
    case "write":
      writeFileSync(OPENAPI_PATH, plan.content);
      console.log(`info.version: ${plan.from} → ${plan.to}`);
  }
}

main();
