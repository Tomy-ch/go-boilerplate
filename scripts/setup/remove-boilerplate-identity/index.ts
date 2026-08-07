#!/usr/bin/env -S tsx
// テンプレート自身を語る散文を落とし、ボイラープレートの顔を消す。
//
// 落とすのは「これはボイラープレートである」と述べている行だけで、リポジトリ名・モジュール名は
// replace-module / replace-repository-reference が既に置換している。運用中も読み返す内容
// （clamp される設定値のレビュー・除外 ADR への入口）は残す。
//
// 使い方:
//   tsx scripts/setup/remove-boilerplate-identity [--dry-run]
//
// この入口はファイル入出力と終了コードだけを担う。対象の宣言は targets.ts、マーカー除去の機構は
// ../lib/markers.ts にある。

import fs from "node:fs";
import path from "node:path";

import { toAbsolutePath } from "../lib/file-utils";
import { stripMarkers } from "../lib/markers";
import { type SetupOptions, newSetupCommand } from "../lib/runtime";
import { BOILERPLATE_MARKER, BOILERPLATE_MARKER_FILES } from "./targets";

type StrippedFile = {
  relativePath: string;
  removedLines: number;
};

/** マーカー行を落とす。dryRun では書き込みだけを飛ばし、戻り値は実行時と一致させる。 */
function stripFiles(dryRun: boolean): StrippedFile[] {
  const stripped: StrippedFile[] = [];

  for (const relativePath of BOILERPLATE_MARKER_FILES) {
    const absolute = toAbsolutePath(relativePath);

    if (!fs.existsSync(absolute)) {
      continue;
    }

    const result = stripMarkers(fs.readFileSync(absolute, "utf8"), BOILERPLATE_MARKER);

    if (result.removed === 0) {
      continue;
    }

    if (!dryRun) {
      fs.writeFileSync(absolute, result.content);
    }
    stripped.push({ relativePath, removedLines: result.removed });
  }

  return stripped;
}

/** 撤去の成功後に自身のディレクトリごと消す。ファイル列挙は判定モジュールを足したとき漏れる。 */
function selfDestruct(): void {
  fs.rmSync(path.dirname(new URL(import.meta.url).pathname), { force: true, recursive: true });
}

function run({ dryRun }: SetupOptions): void {
  console.log("🧹 ボイラープレートを名乗る記述を除去します。");
  if (dryRun) {
    console.log("   モード: dry-run（ファイルは変更しません）");
  }

  const stripped = stripFiles(dryRun);

  if (stripped.length === 0) {
    console.log("除去対象は見つかりませんでした（既に実行済みの可能性があります）。");
    return;
  }

  console.log(`\n${dryRun ? "【Dry Run】除去対象" : "除去完了"}: ${stripped.length} ファイル`);
  for (const item of stripped) {
    console.log(`  - ${item.relativePath} (${item.removedLines} 行)`);
  }

  if (dryRun) {
    console.log("\n次の手順: DRY_RUN を外して再実行");
    return;
  }

  selfDestruct();
  console.log("\n✅ 除去が完了し、このツールも撤去しました。");
}

const program = newSetupCommand("remove-boilerplate-identity");
program
  .description("テンプレート自身を語る散文を落とし、ボイラープレートの顔を消す")
  .action((options: SetupOptions) => {
    try {
      run({ dryRun: options.dryRun });
    } catch (error) {
      program.error(`エラー: ${(error as Error).message}`);
    }
  })
  .parse();
