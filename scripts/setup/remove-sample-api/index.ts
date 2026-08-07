#!/usr/bin/env -S tsx
// ボイラープレートのサンプルAPIを一括削除する。削除対象の宣言は lib/sample-manifest.ts、
// マーカー除去規則は lib/sample-api.ts が持ち、ここは削除・書き込み・出力だけを担う。
//
// 再生成・整形・検証（make gen-api / gen-query / fix / lint）は Go ツールチェーンが要るため
// ここでは行わず、ホスト側の make ターゲット（setup-remove-sample-api）が担当する。

import fs from "node:fs";
import path from "node:path";

import { toAbsolutePath, updateFile } from "../lib/file-utils";
import { ROOT_DIR, type SetupOptions, newSetupCommand } from "../lib/runtime";
import {
  SETUP_VERIFIER_DIR,
  isWithinRoot,
  sharedModuleTargets,
  stripSampleMarkers,
} from "./sample-api";
import { BUILD_STEPS, MARKER_FILES, SAMPLE_DOMAINS } from "./sample-manifest";

// 削除確認スクリプト（verify-sample-removal.ts）が git status と突き合わせる「登録済み削除対象」の
// スナップショット出力先。manifest（sample-manifest.ts）自身が削除対象で削除後は読めないため、
// 削除時にここへ書き出す。削除確認スクリプトは manifest に依存せずこの JSON だけで照合できる。
const SNAPSHOT_PATH = path.join(ROOT_DIR, "scripts/setup/.sample-removal-snapshot.json");

type DeletedPath = {
  domain: string;
  relativePath: string;
};

type DeletionResult = {
  deleted: DeletedPath[];
  missing: string[];
};

type StrippedFile = {
  relativePath: string;
  removedLines: number;
};

// 全ドメインの登録パスを列挙して照合用スナップショットへ書き出す。
function writeSnapshot(): void {
  const registeredPaths = Object.values(SAMPLE_DOMAINS).flatMap((def) => def.paths);
  fs.writeFileSync(SNAPSHOT_PATH, `${JSON.stringify({ registeredPaths }, null, 2)}\n`);
}

// 判定は lib/sample-api.ts が持つ。ここは違反を例外へ変えるだけ。
function assertWithinRoot(absolutePath: string, relativePath: string): void {
  if (!isWithinRoot(absolutePath, ROOT_DIR, path.sep)) {
    throw new Error(
      `削除対象が ROOT_DIR の外（または ROOT_DIR 自体）を指しています: "${relativePath}"`,
    );
  }
}

// 既に存在しないパスはスキップする（再実行や部分実装でも安全に動かすため）。
function deletePaths(dryRun: boolean): DeletionResult {
  const deleted: DeletedPath[] = [];
  const missing: string[] = [];

  for (const [domain, def] of Object.entries(SAMPLE_DOMAINS)) {
    for (const relativePath of def.paths) {
      const absolutePath = toAbsolutePath(relativePath);
      assertWithinRoot(absolutePath, relativePath);

      if (!fs.existsSync(absolutePath)) {
        missing.push(relativePath);
        continue;
      }

      if (!dryRun) {
        fs.rmSync(absolutePath, { recursive: true, force: true });
      }

      deleted.push({ domain, relativePath });
    }
  }

  return { deleted, missing };
}

function stripMarkerFiles(dryRun: boolean): StrippedFile[] {
  const changed: StrippedFile[] = [];

  for (const relativePath of MARKER_FILES) {
    let removedLines = 0;

    const updated = updateFile(
      relativePath,
      (original) => {
        const result = stripSampleMarkers(original);
        removedLines = result.removed;

        return result.content;
      },
      dryRun,
    );

    if (updated) {
      changed.push({ relativePath, removedLines });
    }
  }

  return changed;
}

function report({ deleted, missing }: DeletionResult, stripped: StrippedFile[], dryRun: boolean): void {
  const label = dryRun ? "【Dry Run】削除対象" : "削除完了";

  console.log(`\n${label}: ${deleted.length} パス`);
  for (const domain of Object.keys(SAMPLE_DOMAINS)) {
    const items = deleted.filter((d) => d.domain === domain);
    if (items.length === 0) {
      continue;
    }
    console.log(`\n[${domain}] ${SAMPLE_DOMAINS[domain].description}`);
    for (const item of items) {
      console.log(`  - ${item.relativePath}`);
    }
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

/**
 * 共有モジュール（setup/lib）を、使う側が全て消えたときだけ道連れにする。
 *
 * @remarks
 * このツール自身のディレクトリは manifest の宣言に従って削除されるため、ここでは扱いません。
 */
function removeSharedModules(): void {
  const setupDir = path.join(ROOT_DIR, "scripts/setup");
  const verifierExists = fs.existsSync(path.join(setupDir, SETUP_VERIFIER_DIR));

  for (const target of sharedModuleTargets(verifierExists)) {
    fs.rmSync(path.join(setupDir, target), { force: true, recursive: true });
    console.log(`🧹 初期化ツールは既に消えているため、setup/${target} も撤去しました。`);
  }
}

function run({ dryRun }: SetupOptions): void {
  console.log("🧹 サンプルAPI（user / prefecture / product / order）の削除を開始します。");
  console.log(`   ルート: ${ROOT_DIR}`);
  if (dryRun) {
    console.log("   モード: dry-run（ファイルは変更しません）");
  }

  // マーカー除去を先に行う。マーカー不整合があればここで throw し、
  // ファイル削除前に中断できる（削除済み・マーカー未除去の半端な状態を避ける）。
  const stripped = stripMarkerFiles(dryRun);
  const deletion = deletePaths(dryRun);
  report(deletion, stripped, dryRun);

  const buildHint = `make ${BUILD_STEPS.join(" ")}`;
  if (dryRun) {
    console.log(
      `\n次の手順:\n  実削除  : DRY_RUN を外して再実行\n  再生成等: 削除後に \`${buildHint}\`（make setup-remove-sample-api 経由なら自動実行）`,
    );
    return;
  }

  writeSnapshot();
  removeSharedModules();

  console.log(
    `\n✅ 削除とマーカー除去が完了しました。\n   続けて再生成・整形・検証を行ってください: \`${buildHint}\`\n   （make setup-remove-sample-api 経由で実行した場合は自動で続行されます）`,
  );
}

const program = newSetupCommand("remove-sample-api");
program
  .description("ボイラープレートのサンプルAPI（user / prefecture / product / order）を一括削除する")
  .addHelpText(
    "after",
    `
動作:
  1. manifest（scripts/setup/remove-sample-api/sample-manifest.ts）の宣言パスを丸ごと削除
  2. 共有ファイル（DI 4 ファイル + openapi.yaml）の sample-api マーカー行を除去

削除後は ${BUILD_STEPS.map((s) => `make ${s}`).join(" → ")} で再生成・整形・検証してください
（このスクリプトは Go ツールチェーンを持たない node_tool_runner で動くため再生成は行いません。
 make setup-remove-sample-api 経由ならホスト側で自動的に続行されます）。

core 基盤の idempotency_keys（migration 000001）は削除しません（prefecture は user サンプルの依存ドメインとして削除対象）。
共有生成物（*.gen.go / openapi.gen.yaml 等）は再生成に任せます。
拡張時は sample-manifest.ts の各ドメイン paths への追記とマーカー付与だけで対象に含まれます。`,
  )
  .action((options: SetupOptions) => {
    try {
      run({ dryRun: options.dryRun });
    } catch (error) {
      program.error(`エラー: ${(error as Error).message}`);
    }
  })
  .parse();
