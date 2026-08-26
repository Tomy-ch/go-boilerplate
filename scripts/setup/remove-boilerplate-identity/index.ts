#!/usr/bin/env -S tsx
// 上流のボイラープレートである間だけ成り立つ記述を落とす。
//
// 落とすのはマーカーで囲われた記述だけで、リポジトリ名・モジュール名は replace-module /
// replace-repository-reference が既に置換している。運用中も読み返す内容（clamp される設定値の
// レビュー・除外 ADR への入口）はマーカーの外に居るため残る。
//
// 使い方:
//   tsx scripts/setup/remove-boilerplate-identity [--dry-run]
//
// この入口はファイル入出力と終了コードだけを担う。対象の宣言は targets.ts、マーカー除去の機構は
// ../lib/markers.ts にある。

import fs from "node:fs";
import path from "node:path";

import { listFilesRecursive, toAbsolutePath, toRelativePath } from "../lib/file-utils";
import { stripMarkers } from "../lib/markers";
import { ROOT_DIR, type SetupOptions, newSetupCommand } from "../lib/runtime";
import {
  BOILERPLATE_DELETE_PATHS,
  BOILERPLATE_MARKER,
  EXCLUDED_DIRECTORIES,
  isScanTarget,
} from "./targets";

type StrippedFile = {
  relativePath: string;
  removedLines: number;
};

/**
 * マーカー行を落とす。dryRun では書き込みだけを飛ばし、戻り値は実行時と一致させる。
 *
 * @remarks
 * 対象を列挙せずリポジトリを走査するのは、列挙が「マーカーを書いたのに一覧へ足し忘れる」を
 * 静かに通すためです。列挙side の取りこぼしは何も出力せず成功するので、作成先で前提が
 * 残っていることに誰も気づけません。
 */
function stripFiles(dryRun: boolean): StrippedFile[] {
  const stripped: StrippedFile[] = [];
  const files = listFilesRecursive(ROOT_DIR, {
    excludedDirectories: EXCLUDED_DIRECTORIES,
    shouldIncludeFile: (entryPath) => isScanTarget(toRelativePath(entryPath)),
  });

  for (const absolute of files) {
    const result = stripMarkers(fs.readFileSync(absolute, "utf8"), BOILERPLATE_MARKER);

    if (result.removed === 0) {
      continue;
    }

    if (!dryRun) {
      fs.writeFileSync(absolute, result.content);
    }
    stripped.push({ relativePath: toRelativePath(absolute), removedLines: result.removed });
  }

  return stripped;
}

/** 全体が上流限定であるファイル / ディレクトリを消す。dryRun では消した体で相対パスだけ返す。 */
function deletePaths(dryRun: boolean): string[] {
  const deleted: string[] = [];

  for (const relativePath of BOILERPLATE_DELETE_PATHS) {
    const absolute = toAbsolutePath(relativePath);

    if (!fs.existsSync(absolute)) {
      continue;
    }

    if (!dryRun) {
      fs.rmSync(absolute, { force: true, recursive: true });
    }
    deleted.push(relativePath);
  }

  return deleted;
}

/** 撤去の成功後に自身のディレクトリごと消す。ファイル列挙は判定モジュールを足したとき漏れる。 */
function selfDestruct(): void {
  fs.rmSync(path.dirname(new URL(import.meta.url).pathname), { force: true, recursive: true });
}

function run({ dryRun }: SetupOptions): void {
  console.log("🧹 ボイラープレート限定の記述を除去します。");
  if (dryRun) {
    console.log("   モード: dry-run（ファイルは変更しません）");
  }

  const stripped = stripFiles(dryRun);
  const deleted = deletePaths(dryRun);

  if (stripped.length === 0 && deleted.length === 0) {
    console.log("除去対象は見つかりませんでした（既に実行済みの可能性があります）。");
    return;
  }

  console.log(`\n${dryRun ? "【Dry Run】除去対象" : "除去完了"}: ${stripped.length} ファイル`);
  for (const item of stripped) {
    console.log(`  - ${item.relativePath} (${item.removedLines} 行)`);
  }

  if (deleted.length > 0) {
    console.log(`\n${dryRun ? "【Dry Run】削除対象" : "削除完了"}: ${deleted.length} ファイル`);
    for (const relativePath of deleted) {
      console.log(`  - ${relativePath}`);
    }
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
  .description("ボイラープレートである間だけ成り立つ記述を落とす")
  .action((options: SetupOptions) => {
    try {
      run({ dryRun: options.dryRun });
    } catch (error) {
      program.error(`エラー: ${(error as Error).message}`);
    }
  })
  .parse();
