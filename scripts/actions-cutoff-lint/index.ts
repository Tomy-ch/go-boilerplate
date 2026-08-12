#!/usr/bin/env -S tsx
// ジョブが打ち切られたときの振る舞いが定義されているかを検査する lint スクリプト。
// `timeout-minutes` / `if:` / `title:` の 3 点を 1 本の検査にまとめる理由と、それぞれが
// 打ち切りを不可視にする経路は .github/workflows/README.md の Job Cut-off が持つ。
//
// 静的に読めるものだけを見る。`!always()` のように到達性を打ち消す式は書けてしまうが、規約が正で
// この検査はその近似にあたる。1 件でも違反があれば非 0 で終了する。
//
// この入口はファイル入出力と終了コードだけを担う。走査と判定は ./cutoff にある。

import fs from "node:fs";
import path from "node:path";

import { COMMENT_ACTION, scanWorkflow } from "./cutoff";
import { type Finding, formatFindings } from "../lib/lint-report";
import { selectWorkflowFiles } from "../lib/workflow";

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
