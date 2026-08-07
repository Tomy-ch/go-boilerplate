#!/usr/bin/env -S tsx
// サンプル削除が「過不足なく」完了したことを検証し、最後に自身と付随ファイルを自爆削除する。
// remove-sample-api.ts は manifest ごと自消滅するため、削除確認は manifest ではなく
// remove-sample-api.ts が書き出した .sample-removal-snapshot.json を唯一の入力として行う（bootstrap 対策）。
// このスクリプト自体はサンプル削除ツールの最終地点なので、検証成功後に自消滅してコアのみの状態を残す。
//
// 判定は lib/sample-removal-verify.ts が持ち、ここは git / make / grep の起動と終了コードだけを担う。

import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import {
  buildDanglingCommand,
  collectFailures,
  parseSnapshot,
} from "./lib/sample-removal-verify";

const SELF_PATH = fileURLToPath(import.meta.url);
const SETUP_DIR = path.dirname(SELF_PATH);
const ROOT_DIR = path.resolve(SETUP_DIR, "../..");
const SNAPSHOT_PATH = path.join(SETUP_DIR, ".sample-removal-snapshot.json");

// 自消滅の対象。判定モジュールとそのテストもこのスクリプト専用なので道連れにする
// （残すと、消えたはずの検証ツールの一部だけが利用者のリポジトリに居座る）。
const SELF_DESTRUCT_PATHS = [
  SNAPSHOT_PATH,
  path.join(SETUP_DIR, "lib/sample-removal-verify.ts"),
  path.join(SETUP_DIR, "lib/sample-removal-verify.test.ts"),
  SELF_PATH,
];

function readRegisteredPaths(): string[] {
  if (!fs.existsSync(SNAPSHOT_PATH)) {
    throw new Error(
      `スナップショットが見つかりません: ${SNAPSHOT_PATH}（remove-sample-api.ts 未実行の可能性）`,
    );
  }

  return parseSnapshot(fs.readFileSync(SNAPSHOT_PATH, "utf8"));
}

function readGitStatus(): string {
  return execFileSync("git", ["status", "--porcelain"], { cwd: ROOT_DIR, encoding: "utf8" });
}

// make help はターゲットが消えていれば非 0 で終わることがあるため、stdout を拾って続行する。
function readMakeHelp(): string {
  try {
    return execFileSync("make", ["help"], {
      cwd: ROOT_DIR,
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"],
    });
  } catch (error) {
    return (error as { stdout?: string }).stdout ?? "";
  }
}

function readDanglingHits(): string {
  return execFileSync("bash", ["-c", buildDanglingCommand()], { cwd: ROOT_DIR, encoding: "utf8" });
}

function selfDestruct(): void {
  for (const target of SELF_DESTRUCT_PATHS) {
    fs.rmSync(target, { force: true });
  }
}

function main(): void {
  console.log("🔍 サンプル削除の検証を開始します（過不足・残留参照・ツール自消滅）。");

  const failures = collectFailures({
    registeredPaths: readRegisteredPaths(),
    pathExists: (relativePath) => fs.existsSync(path.join(ROOT_DIR, relativePath)),
    gitStatusPorcelain: readGitStatus(),
    makeHelpOutput: readMakeHelp(),
    danglingHits: readDanglingHits(),
  });

  if (failures.length > 0) {
    console.error("❌ サンプル削除の検証に失敗しました:");
    for (const failure of failures) {
      console.error(`  - ${failure}`);
    }
    process.exit(1);
  }

  selfDestruct();
  console.log("✅ 過不足なく削除・残留なしを確認し、削除確認スクリプトも自消滅しました。");
}

try {
  main();
} catch (error) {
  console.error(`❌ 検証エラー: ${(error as Error).message}`);
  process.exit(1);
}
