#!/usr/bin/env -S tsx
// `upsert-pr-comment` を使うジョブに secret が渡っていないかを検査する lint スクリプト。
// マスキングを当てにできない理由と GITHUB_TOKEN だけを許可する根拠は
// .github/workflows/README.md の「A job that comments on a PR is passed no secret」が持つ。
//
// 判断は含めず、ワークフロー定義から機械的に導けるものだけを見る。1 件でも違反があれば非 0 で終了する。
//
// この入口はファイル入出力と終了コードだけを担う。走査と判定は ./secret にある。

import fs from "node:fs";
import path from "node:path";

import { type Finding, formatFindings } from "../lib/lint-report";
import { scanWorkflow } from "./secret";
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
    `✘ pr-comment-secret-lint: ${WORKFLOWS_DIR}/ にワークフローが見つかりません（リポジトリルートで実行してください）`,
  );
  process.exit(2);
}

const findings: Finding[] = [];
let checkedJobs = 0;

for (const rel of workflowFiles) {
  const source = fs.readFileSync(path.join(REPO_ROOT, rel), "utf8");
  const scan = scanWorkflow(rel, source);

  if (!scan.found) {
    console.error(`✘ pr-comment-secret-lint: ${rel} に jobs: が見つかりません`);
    process.exit(2);
  }

  findings.push(...scan.findings);
  checkedJobs += scan.commentingJobs;
}

if (findings.length > 0) {
  console.error(`✘ pr-comment-secret-lint: ${findings.length} 件の違反\n`);
  console.error(formatFindings(findings));
  console.error(
    `\n検査 ${workflowFiles.length} ワークフロー / ${checkedJobs} ジョブ中 ${findings.length} 件 NG`,
  );
  process.exit(1);
}

console.log(
  `✓ pr-comment-secret-lint: ${workflowFiles.length} ワークフロー / ${checkedJobs} ジョブすべて OK`,
);
