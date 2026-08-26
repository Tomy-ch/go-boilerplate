#!/usr/bin/env -S tsx
// ワークフローが自前で組む Markdown フェンス / inline code span の退行を検査する lint スクリプト。
// フェンス長を本文から取る根拠と、span へ値を補間する危険は .github/workflows/README.md の
// PR comment fencing が持つ。
//
// 機械的に判定できる 3 点のみを見る（「その本文が攻撃者制御か」は判定しない）。
//
//   1. `run:` ブロックが固定長のフェンスを出力していないこと
//   2. 複数のワークフローが持つ `fence_for` の実装が互いに完全一致していること
//   3. 素通し呼び出しを持つワークフローが、inline code span の内側へ値を補間していないこと
//
// 2 の複製をシェルファイルへ切り出して `source` する経路は採らない。`actionlint` が読むのは
// `.github/workflows/**`、`actions-shellcheck` が読むのは `.github/actions/**` だけなので、集約先は
// どちらの検査対象でもなくなる。複製が持ち込む唯一のリスク（片方だけが直る drift）は 2 が止める。
//
// 3 の粒度はワークフローファイル単位で、ステップから本文ファイルへのデータフローは追わない。
// 素通し呼び出しと `details-summary` 付き呼び出しが同居するファイルでは過剰検出になり得るが、
// 意図した単純化として除外リストで受ける。span を変数経由や jq の連結で組む形は検出できない。
//
// この入口はファイル入出力と終了コードだけを担う。走査と判定は ./fence にある。

import fs from "node:fs";
import path from "node:path";

import {
  PASS_THROUGH_EXCLUSIONS,
  compareImplementations,
  scanWorkflow,
} from "./fence";
import { selectWorkflowFiles } from "../lib/workflow";

const REPO_ROOT = process.cwd();
const WORKFLOWS_DIR = ".github/workflows";

function listWorkflows(): string[] {
  return selectWorkflowFiles(fs.readdirSync(path.join(REPO_ROOT, WORKFLOWS_DIR)), WORKFLOWS_DIR);
}

const violations: string[] = [];
const implementations: Array<[string, string]> = [];
const workflowFiles = listWorkflows();

for (const file of workflowFiles) {
  const lines = fs.readFileSync(path.join(REPO_ROOT, file), "utf8").split("\n");
  const scan = scanWorkflow(file, lines);

  violations.push(...scan.violations);
  if (scan.implementation !== null) implementations.push([file, scan.implementation]);
}

violations.push(...compareImplementations(implementations));

if (violations.length > 0) {
  for (const violation of violations) console.error(`✗ ${violation}`);
  console.error(`\n✗ pr-comment-fence-lint: ${violations.length} 件の違反があります`);
  process.exit(1);
}

// 除外を黙ったまま緑にすると「検査が通った」と「見ていない」の区別が付かなくなる。
for (const [name, reason] of PASS_THROUGH_EXCLUSIONS) {
  console.log(`- span 検査を除外: ${name} — ${reason}`);
}

console.log(
  `✓ pr-comment-fence-lint: ${workflowFiles.length} ワークフロー / fence_for 実装 ${implementations.length} 件すべて OK`,
);
