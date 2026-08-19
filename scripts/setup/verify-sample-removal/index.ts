#!/usr/bin/env -S tsx
// サンプル削除が「過不足なく」完了したことを検証し、最後に自身と付随ファイルを自爆削除する。
// remove-sample-api は manifest ごと自消滅するため、削除確認は manifest ではなく
// remove-sample-api が書き出した .sample-removal-snapshot.json を唯一の入力として行う（bootstrap 対策）。
//
// 判定は ./verify.ts が持ち、ここは git / make / grep の起動と終了コードだけを担う。

import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import {
  buildDanglingCommand,
  collectFailures,
  extractFileRefs,
  parseSnapshot,
  reachableFiles,
  selfDestructTargets,
} from "./verify";

const SELF_DIR = path.dirname(fileURLToPath(import.meta.url));
const SETUP_DIR = path.resolve(SELF_DIR, "..");
const ROOT_DIR = path.resolve(SETUP_DIR, "../..");
const SNAPSHOT_PATH = path.join(SETUP_DIR, ".sample-removal-snapshot.json");

const SELF_DESTRUCT_PATHS = selfDestructTargets(SELF_DIR, SNAPSHOT_PATH);

/** OpenAPI 正本の入口。ここから $ref を辿れないファイルが孤立候補になる。 */
const OPENAPI_ENTRYPOINT = "openapi/openapi.yaml";
/** 孤立を見る範囲。paths 配下は撤去でファイルごと消えるため、残り得るのは components だけ。 */
const OPENAPI_COMPONENTS_DIR = "openapi/components";

function readRegisteredPaths(): string[] {
  if (!fs.existsSync(SNAPSHOT_PATH)) {
    throw new Error(
      `スナップショットが見つかりません: ${SNAPSHOT_PATH}（remove-sample-api 未実行の可能性）`,
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

/** 撤去後も残っている components 配下の YAML をリポジトリ相対パスで列挙する。 */
function listSurvivingComponents(): string[] {
  const rootDir = path.join(ROOT_DIR, OPENAPI_COMPONENTS_DIR);
  if (!fs.existsSync(rootDir)) {
    return [];
  }

  return fs
    .readdirSync(rootDir, { recursive: true, withFileTypes: true })
    .filter((entry) => entry.isFile() && /\.ya?ml$/.test(entry.name))
    .map((entry) => path.posix.join(path.relative(ROOT_DIR, entry.parentPath), entry.name));
}

/** 作業ツリーの内容から $ref を読む。撤去後の状態。 */
function refsFromWorktree(relativePath: string): string[] {
  const absolutePath = path.join(ROOT_DIR, relativePath);

  return fs.existsSync(absolutePath) ? extractFileRefs(fs.readFileSync(absolutePath, "utf8")) : [];
}

// HEAD の内容から $ref を読む。撤去は未コミットなので HEAD が撤去前の状態にあたる。
// 存在しないパスでは git が非 0 で終わるため、参照なしとして扱う。
function refsFromHead(relativePath: string): string[] {
  try {
    return extractFileRefs(
      execFileSync("git", ["show", `HEAD:${relativePath}`], {
        cwd: ROOT_DIR,
        encoding: "utf8",
        stdio: ["ignore", "pipe", "ignore"],
        maxBuffer: 64 * 1024 * 1024,
      }),
    );
  } catch {
    return [];
  }
}

function selfDestruct(): void {
  for (const target of SELF_DESTRUCT_PATHS) {
    fs.rmSync(target, { force: true, recursive: true });
  }
}

function main(): void {
  console.log("🔍 サンプル削除の検証を開始します（過不足・残留参照・定義の孤立・ツール自消滅）。");

  const failures = collectFailures({
    registeredPaths: readRegisteredPaths(),
    pathExists: (relativePath) => fs.existsSync(path.join(ROOT_DIR, relativePath)),
    gitStatusPorcelain: readGitStatus(),
    makeHelpOutput: readMakeHelp(),
    danglingHits: readDanglingHits(),
    survivingComponents: listSurvivingComponents(),
    reachableBefore: reachableFiles(OPENAPI_ENTRYPOINT, refsFromHead),
    reachableAfter: reachableFiles(OPENAPI_ENTRYPOINT, refsFromWorktree),
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
