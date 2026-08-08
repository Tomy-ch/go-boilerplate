#!/usr/bin/env -S tsx
// 初期化（`docs/get-started/setup-repository.md` の Phase 5）が過不足なく終わったことを検証し、
// 通ったら初期化ツール一式と自身を撤去する。
//
// 撤去するのは、`replace-*` が一度きりのインスタンス化ツールで、残しても「もう当ててはいけない」
// ものにしかならないため。特に `replace-codeowners` は全ルールの所有者を一括で同じ値へ書き換えるので、
// パスごとに所有者が分かれた後の CODEOWNERS に当てると壊す。
//
// 撤去を実行直後ではなく検証成功後に置くのは、Phase 5 が 5 本の連続実行だから。1 本目の引数を
// 打ち間違えた利用者が、直すためのツールを既に失っている状態を作らない。
//
// この入口はファイル入出力と終了コードだけを担う。判定は verify.ts にある。

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { listFilesRecursive } from "../lib/file-utils";
import { stripMarkers } from "../lib/markers";
import { EXCLUDED_DIRECTORIES, isReplacementTarget } from "../replace-module/module-replace";
import {
  BOILERPLATE_MODULE,
  type ExpectedIdentity,
  LOCALIZATION_MARKER,
  LOCALIZATION_MARKER_FILES,
  SAMPLE_REMOVER_DIR,
  collectFailures,
  selfDestructTargets,
} from "./verify";

const SELF_DIR = path.dirname(fileURLToPath(import.meta.url));
const SETUP_DIR = path.resolve(SELF_DIR, "..");
const ROOT_DIR = path.resolve(SETUP_DIR, "../..");

function read(relativePath: string): string {
  const absolute = path.join(ROOT_DIR, relativePath);

  return fs.existsSync(absolute) ? fs.readFileSync(absolute, "utf8") : "";
}

/**
 * ボイラープレート名が残っている置換対象ファイルを列挙する。
 *
 * @remarks
 * 走査範囲と対象判定は `replace-module` のものをそのまま使います。検証側が独自の除外を持つと、
 * 置換器が対象を変えたときに片方だけ古い規則で見続け、取りこぼしを見逃します。
 */
function boilerplateReferences(): string[] {
  const files = listFilesRecursive(ROOT_DIR, {
    excludedDirectories: EXCLUDED_DIRECTORIES,
    shouldIncludeFile: (entryPath) => isReplacementTarget(path.relative(ROOT_DIR, entryPath)),
  });

  return files
    .filter((file) => fs.readFileSync(file, "utf8").includes(BOILERPLATE_MODULE))
    .map((file) => path.relative(ROOT_DIR, file));
}

function requireEnv(name: string): string {
  const value = process.env[name];

  if (value === undefined || value === "") {
    console.error(`❌ 環境変数 ${name} が必要です`);
    process.exit(2);
  }

  return value;
}

function expectedIdentity(): ExpectedIdentity {
  return {
    module: requireEnv("MODULE"),
    repository: requireEnv("REPOSITORY"),
    copyrightHolder: requireEnv("COPYRIGHT_HOLDER"),
    copyrightYear: requireEnv("COPYRIGHT_YEAR"),
    codeOwners: requireEnv("CODE_OWNERS"),
  };
}

/**
 * 初期化ツールを指す宣言（make ターゲットとその説明）を落とす。
 *
 * @remarks
 * ディレクトリの削除より先に行います。make は起動時に makefile を全読込するため、実行中の
 * レシピはこの書き換えの後も走り切ります。
 */
function stripDeclarations(): void {
  for (const relativePath of LOCALIZATION_MARKER_FILES) {
    const absolute = path.join(ROOT_DIR, relativePath);

    if (!fs.existsSync(absolute)) continue;

    const result = stripMarkers(fs.readFileSync(absolute, "utf8"), LOCALIZATION_MARKER);

    if (result.removed === 0) continue;

    fs.writeFileSync(absolute, result.content);
    console.log(`  ${relativePath} (${result.removed} 行)`);
  }
}

function selfDestruct(): void {
  const sampleRemoverExists = fs.existsSync(path.join(SETUP_DIR, SAMPLE_REMOVER_DIR));

  for (const target of selfDestructTargets(path.basename(SELF_DIR), sampleRemoverExists)) {
    fs.rmSync(path.join(SETUP_DIR, target), { force: true, recursive: true });
  }

  if (!sampleRemoverExists) {
    console.log("🧹 サンプル削除ツールは既に消えているため、setup/lib も撤去しました。");
  }
}

function main(): void {
  console.log("🔍 初期化の検証を開始します（置換の過不足・ボイラープレート名の残留）。");

  const failures = collectFailures({
    expected: expectedIdentity(),
    goMod: read("go.mod"),
    license: read("LICENSE"),
    codeowners: read(".github/CODEOWNERS"),
    readme: read("README.md"),
    openapi: read("openapi/openapi.yaml"),
    boilerplateReferences: boilerplateReferences(),
  });

  if (failures.length > 0) {
    console.error("❌ 初期化の検証に失敗しました:");
    for (const failure of failures) {
      console.error(`  - ${failure}`);
    }
    process.exit(1);
  }

  console.log("🧹 初期化ツールの宣言を除去します...");
  stripDeclarations();
  selfDestruct();
  console.log("✅ 初期化の完了を確認し、初期化ツールも撤去しました。");
}

try {
  main();
} catch (error) {
  console.error(`❌ 検証エラー: ${(error as Error).message}`);
  process.exit(1);
}
