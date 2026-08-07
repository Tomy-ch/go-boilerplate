#!/usr/bin/env -S tsx
// ジョブが打ち切られたときの振る舞いが定義されているかを検査する lint スクリプト。
//
// 打ち切り（タイムアウト / キャンセル / ランナー障害）は複数の欠落が組み合わさって不可視になる。
// 一つが状況を作り、残りが状況を見えなくするので、まとめて 1 つの検査として見る。
//
//   A. `upsert-pr-comment` を呼ぶステップの `if:` が cancelled に到達しない
//      Actions はステータスチェック関数を含まないカスタム `if:` に暗黙で `success() &&` を前置する。
//      打ち切られたジョブではこのステップがスキップされ、PR には何のコメントも残らない。
//      失敗ステップ側は `always()` を持つことが多いので、赤くはなるが理由が読めない状態になる。
//
//   B. `upsert-pr-comment` を呼ぶステップの `title:` に打ち切り時の見出しが無い
//      本文を `tee` で書くジョブは、打ち切られても書きかけのファイルを残す。action 側はファイルの
//      有無しか見られないので、この経路は「完了」と区別がつかない。一方 `title` を出力する行までは
//      到達していないので、呼び出し側が `|| '…CUT OFF…'` を持つかどうかが唯一の判定材料になる。
//
//   C. ジョブに `timeout-minutes:` が無い
//      GitHub 既定の 360 分まで走り続け、ハングした 1 ジョブがランナーを 6 時間占有しうる。
//
// 静的に読めるものだけを見る。`!always()` のように到達性を打ち消す式は書けてしまうが、規約が正で
// この検査はその近似にあたる。1 件でも違反があれば非 0 で終了する。
//
// この入口はファイル入出力と終了コードだけを担う。走査と判定は lib/actions-cutoff.ts にある。

import fs from "node:fs";
import path from "node:path";

import { COMMENT_ACTION, scanWorkflow } from "./lib/actions-cutoff";
import { type Finding, formatFindings } from "./lib/lint-report";
import { selectWorkflowFiles } from "./lib/workflow";

const REPO_ROOT = process.cwd();
const WORKFLOWS_DIR = ".github/workflows";

function listWorkflowFiles(): string[] {
  const dir = path.join(REPO_ROOT, WORKFLOWS_DIR);
  if (!fs.existsSync(dir)) return [];

  return selectWorkflowFiles(fs.readdirSync(dir), WORKFLOWS_DIR);
}

const workflowFiles = listWorkflowFiles();

// 検査対象 0 件は「問題なし」ではなく「検査が働いていない」。実行位置の誤りを成功と report しない。
if (workflowFiles.length === 0) {
  console.error(
    `✘ actions-cutoff-lint: ${WORKFLOWS_DIR}/ にワークフローが見つかりません（リポジトリルートで実行してください）`,
  );
  process.exit(2);
}

const findings: Finding[] = [];
let checkedJobs = 0;
let checkedSteps = 0;

for (const rel of workflowFiles) {
  const source = fs.readFileSync(path.join(REPO_ROOT, rel), "utf8");
  const scan = scanWorkflow(rel, source);

  if (!scan.found) {
    console.error(`✘ actions-cutoff-lint: ${rel} に jobs: が見つかりません`);
    process.exit(2);
  }

  findings.push(...scan.findings);
  checkedJobs += scan.checkedJobs;
  checkedSteps += scan.checkedSteps;
}

// 1 件も拾えないのは、規約が守られているのではなく検査が的を外している状態。パスの変更や
// 書式の揺れで正規表現が一致しなくなっても、緑で通してしまわないようにする。
if (checkedSteps === 0) {
  console.error(
    `✘ actions-cutoff-lint: ${COMMENT_ACTION} を呼ぶステップが 1 件も見つかりません（検査が的を外しています）`,
  );
  process.exit(2);
}

if (findings.length > 0) {
  console.error(`✘ actions-cutoff-lint: ${findings.length} 件の違反\n`);
  console.error(formatFindings(findings));
  console.error(
    `\n検査 ${workflowFiles.length} ワークフロー / ${checkedJobs} ジョブ / ${checkedSteps} コメントステップ中 ${findings.length} 件 NG`,
  );
  process.exit(1);
}

console.log(
  `✓ actions-cutoff-lint: ${workflowFiles.length} ワークフロー / ${checkedJobs} ジョブ / ${checkedSteps} コメントステップすべて OK`,
);
