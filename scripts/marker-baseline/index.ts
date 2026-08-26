#!/usr/bin/env -S tsx
// マーカー行の分布をベースラインへ書き出す。
//
// 検査そのものは `rules.test.ts` が行うので、この入口は再生成だけを担う。ゲートを人が更新
// できないと、赤を消すために検査のほうを外す圧力が生まれる。
//
// 使い方:
//   tsx scripts/marker-baseline           # 差分を表示するだけ
//   tsx scripts/marker-baseline --write   # ベースラインを現状で上書きする

import fs from "node:fs";

import { diffBaseline } from "./rules";
import { BASELINE_PATH, readBaseline, scanRepository } from "./scan";

const actual = scanRepository();
const failures = diffBaseline(actual, readBaseline());

if (process.argv.includes("--write")) {
  fs.writeFileSync(BASELINE_PATH, `${JSON.stringify(actual, null, 2)}\n`);
  console.log(`✓ marker-baseline: ${Object.keys(actual).length} ファイルで更新しました`);
  process.exit(0);
}

if (failures.length === 0) {
  console.log(`✓ marker-baseline: ${Object.keys(actual).length} ファイル、差分なし`);
  process.exit(0);
}

console.error(`✗ marker-baseline: ${failures.length} 件\n`);
for (const failure of failures) console.error(`  ${failure}`);
console.error("\n意図した変更なら: tsx scripts/marker-baseline --write");
process.exit(1);
