#!/usr/bin/env -S tsx
// DAST（OWASP ZAP）のサンプル設定一式を撤去する。
//
// 撤去するのはワークフロー本体・ZAP のルールファイル・一覧やセットアップ手順の該当行・
// それらを呼ぶ make ターゲット、そしてこのツール自身。有効/無効を切り替えるオプションは
// 持たない（撤去するかしないかの二択でよく、戻したくなれば git の履歴から辿れる）。
//
// 使い方:
//   tsx scripts/setup/remove-dast-setting [--dry-run]
//
// この入口はファイル入出力と終了コードだけを担う。対象の宣言は dast-targets.ts、
// マーカー除去の機構は ../lib/markers.ts にある。

import fs from "node:fs";

import { toAbsolutePath, updateFile } from "../lib/file-utils";
import { stripMarkers } from "../lib/markers";
import { type SetupOptions, newSetupCommand } from "../lib/runtime";
import {
  ACTION_PIN_LOCKFILE,
  DAST_ACTION_PIN_KEY,
  DAST_EGRESS_SECTION_PREFIX,
  DAST_MARKER,
  DAST_MARKER_FILES,
  DAST_PATHS,
  EGRESS_SSOT,
  isRemovablePath,
  stripActionPin,
  stripEgressJob,
} from "./dast-targets";

type StrippedFile = {
  relativePath: string;
  removedLines: number;
};

type DeletionResult = {
  deleted: string[];
  missing: string[];
};

/** 既に存在しないパスはスキップする（再実行や部分適用でも安全に動かすため）。 */
function deletePaths(dryRun: boolean): DeletionResult {
  const deleted: string[] = [];
  const missing: string[] = [];

  for (const relativePath of DAST_PATHS) {
    if (!isRemovablePath(relativePath)) {
      throw new Error(`削除対象として受け付けられないパスです: "${relativePath}"`);
    }

    const absolutePath = toAbsolutePath(relativePath);

    if (!fs.existsSync(absolutePath)) {
      missing.push(relativePath);
      continue;
    }

    if (!dryRun) {
      fs.rmSync(absolutePath, { force: true, recursive: true });
    }

    deleted.push(relativePath);
  }

  return { deleted, missing };
}

/** マーカー行を落とす。dryRun では書き込みだけを飛ばし、戻り値は実行時と一致させる。 */
function stripMarkerFiles(dryRun: boolean): StrippedFile[] {
  const stripped: StrippedFile[] = [];

  for (const relativePath of DAST_MARKER_FILES) {
    let removedLines = 0;

    const updated = updateFile(
      relativePath,
      (original) => {
        const result = stripMarkers(original, DAST_MARKER);
        removedLines = result.removed;

        return result.content;
      },
      dryRun,
    );

    if (updated !== null) {
      stripped.push({ relativePath, removedLines });
    }
  }

  return stripped;
}

/** pin lockfile から DAST の action エントリを落とす。書き換えたパスを返す（不要なら null）。 */
function stripActionPinEntry(dryRun: boolean): string | null {
  return updateFile(
    ACTION_PIN_LOCKFILE,
    (original) => stripActionPin(original, DAST_ACTION_PIN_KEY),
    dryRun,
  );
}

/** egress の SSOT から DAST のジョブセクションを落とす。書き換えたパスを返す（不要なら null）。 */
function stripEgressEntry(dryRun: boolean): string | null {
  return updateFile(
    EGRESS_SSOT,
    (original) => stripEgressJob(original, DAST_EGRESS_SECTION_PREFIX),
    dryRun,
  );
}

function report({ deleted, missing }: DeletionResult, stripped: StrippedFile[], dryRun: boolean): void {
  console.log(`\n${dryRun ? "【Dry Run】削除対象" : "削除完了"}: ${deleted.length} パス`);
  for (const relativePath of deleted) {
    console.log(`  - ${relativePath}`);
  }

  if (missing.length > 0) {
    console.log(`\n🟡 既に存在しない（スキップ）: ${missing.length} パス`);
    for (const relativePath of missing) {
      console.log(`  - ${relativePath}`);
    }
  }

  console.log(
    `\n${dryRun ? "【Dry Run】マーカー除去対象" : "マーカー除去完了"}: ${stripped.length} ファイル`,
  );
  for (const item of stripped) {
    console.log(`  - ${item.relativePath} (${item.removedLines} 行)`);
  }
}

function run({ dryRun }: SetupOptions): void {
  console.log("🧹 DAST（OWASP ZAP）のサンプル設定を撤去します。");
  if (dryRun) {
    console.log("   モード: dry-run（ファイルは変更しません）");
  }

  // マーカー除去を先に行う。マーカー不整合があればここで throw し、ファイル削除前に中断できる
  // （削除済み・マーカー未除去の半端な状態を避ける）。
  const stripped = stripMarkerFiles(dryRun);
  const pinEntry = stripActionPinEntry(dryRun);
  const egressEntry = stripEgressEntry(dryRun);
  const deletion = deletePaths(dryRun);
  report(deletion, stripped, dryRun);

  console.log(
    `\n${dryRun ? "【Dry Run】pin lockfile" : "pin lockfile"}: ${
      pinEntry === null ? "該当エントリなし" : `${pinEntry} から ${DAST_ACTION_PIN_KEY} を除去`
    }`,
  );

  console.log(
    `${dryRun ? "【Dry Run】egress SSOT" : "egress SSOT"}: ${
      egressEntry === null
        ? "該当エントリなし"
        : `${egressEntry} から ${DAST_EGRESS_SECTION_PREFIX} のジョブを除去`
    }`,
  );

  if (dryRun) {
    console.log("\n次の手順: DRY_RUN を外して再実行");
    return;
  }

  console.log("\n✅ DAST の撤去が完了し、このツールも一緒に消えました。");
}

const program = newSetupCommand("remove-dast-setting");
program
  .description("DAST（OWASP ZAP）のサンプル設定一式を撤去する")
  .addHelpText(
    "after",
    `
動作:
  1. dast-targets.ts の宣言パス（ワークフロー本体 / ZAP ルールファイル / このツール自身）を削除
  2. ワークフロー一覧・セットアップ手順・CI 定義の dast マーカー行を除去
  3. .github/actions-pin.toml から、参照が消える action のエントリを除去
  4. .github/egress.toml から、消えるワークフローのジョブセクションを除去

有効/無効を切り替えるオプションはありません。撤去するかしないかの二択で足り、無効のまま
残る設定は誰にも読まれないまま腐ります。撤去後に中身を参照したくなったら git の履歴から辿れます。`,
  )
  .action((options: SetupOptions) => {
    try {
      run({ dryRun: options.dryRun });
    } catch (error) {
      program.error(`エラー: ${(error as Error).message}`);
    }
  })
  .parse();
